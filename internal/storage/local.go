package storage

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// LocalStorage implements the Storage interface using local filesystem
type LocalStorage struct {
	basePath      string
	baseURL       string // Base URL for generating signed URLs (e.g., "http://localhost:8080")
	signingSecret string // Secret for signing URLs
}

// signedURLToken represents the data encoded in a signed URL token
type signedURLToken struct {
	Bucket    string `json:"b"`
	Key       string `json:"k"`
	ExpiresAt int64  `json:"e"`
	Method    string `json:"m"`
	// Transform options (optional, for image downloads)
	TrWidth   int    `json:"tw,omitempty"` // Transform width
	TrHeight  int    `json:"th,omitempty"` // Transform height
	TrFormat  string `json:"tf,omitempty"` // Transform format
	TrQuality int    `json:"tq,omitempty"` // Transform quality
	TrFit     string `json:"ti,omitempty"` // Transform fit mode
}

// SignedTokenResult contains the result of validating a signed URL token
type SignedTokenResult struct {
	Bucket           string
	Key              string
	Method           string
	TransformWidth   int
	TransformHeight  int
	TransformFormat  string
	TransformQuality int
	TransformFit     string
}

// NewLocalStorage creates a new local filesystem storage provider
func NewLocalStorage(basePath, baseURL, signingSecret string) (*LocalStorage, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &LocalStorage{
		basePath:      basePath,
		baseURL:       strings.TrimSuffix(baseURL, "/"),
		signingSecret: signingSecret,
	}, nil
}

// Name returns the provider name
func (ls *LocalStorage) Name() string {
	return "local"
}

// Health checks if the storage is healthy
func (ls *LocalStorage) Health(ctx context.Context) error {
	// Check if base path is accessible
	if _, err := os.Stat(ls.basePath); err != nil {
		return fmt.Errorf("storage directory not accessible: %w", err)
	}

	// Try to create a test file
	testFile := filepath.Join(ls.basePath, ".health_check")
	if err := os.WriteFile(testFile, []byte("ok"), 0644); err != nil {
		return fmt.Errorf("storage directory not writable: %w", err)
	}

	// Clean up test file
	_ = os.Remove(testFile)

	return nil
}

// validatePath validates that a path component doesn't contain path traversal sequences
func validatePathComponent(component string) error {
	if component == "" {
		return fmt.Errorf("empty path component")
	}
	// Check for path traversal patterns
	if strings.Contains(component, "..") {
		return fmt.Errorf("path traversal detected: '..' not allowed")
	}
	// Check for null bytes (can be used to bypass validation)
	if strings.Contains(component, "\x00") {
		return fmt.Errorf("null bytes not allowed in path")
	}
	// Check for absolute paths
	if filepath.IsAbs(component) || strings.HasPrefix(component, "/") || strings.HasPrefix(component, "\\") {
		return fmt.Errorf("absolute paths not allowed")
	}
	return nil
}

// getPath returns the full filesystem path for a bucket/key
// Returns an error if path traversal is detected
func (ls *LocalStorage) getPath(bucket, key string) (string, error) {
	// Validate bucket name
	if err := validatePathComponent(bucket); err != nil {
		return "", fmt.Errorf("invalid bucket: %w", err)
	}

	// Validate each component of the key path
	keyParts := strings.Split(filepath.ToSlash(key), "/")
	for _, part := range keyParts {
		if part == "" {
			continue // Skip empty parts from leading/trailing slashes
		}
		if err := validatePathComponent(part); err != nil {
			return "", fmt.Errorf("invalid key path: %w", err)
		}
	}

	// Build the full path
	fullPath := filepath.Join(ls.basePath, bucket, key)

	// Clean the path and verify it's still within the base path
	fullPath = filepath.Clean(fullPath)
	bucketPath := filepath.Clean(filepath.Join(ls.basePath, bucket))

	// Double-check: the full path must start with the bucket path
	if !strings.HasPrefix(fullPath, bucketPath) {
		return "", fmt.Errorf("path escapes bucket directory")
	}

	return fullPath, nil
}

// Upload uploads a file to local storage
func (ls *LocalStorage) Upload(ctx context.Context, bucket, key string, data io.Reader, size int64, opts *UploadOptions) (*Object, error) {
	if opts == nil {
		opts = &UploadOptions{}
	}

	// Validate and get file path
	filePath, err := ls.getPath(bucket, key)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Create bucket directory if it doesn't exist
	bucketPath := filepath.Join(ls.basePath, bucket)
	if err := os.MkdirAll(bucketPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create bucket directory: %w", err)
	}

	// Create parent directories for the key
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Calculate MD5 hash while writing
	hash := md5.New()
	writer := io.MultiWriter(file, hash)

	// Copy data to file
	written, err := io.Copy(writer, data)
	if err != nil {
		_ = os.Remove(filePath)
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	// Get file info
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Calculate ETag (MD5 hash)
	etag := hex.EncodeToString(hash.Sum(nil))

	// Save metadata as extended attributes or separate file
	if len(opts.Metadata) > 0 {
		metaPath := filePath + ".meta"
		metaData := ""
		for k, v := range opts.Metadata {
			metaData += fmt.Sprintf("%s=%s\n", k, v)
		}
		if opts.ContentType != "" {
			metaData += fmt.Sprintf("content-type=%s\n", opts.ContentType)
		}
		_ = os.WriteFile(metaPath, []byte(metaData), 0644)
	}

	log.Debug().
		Str("bucket", bucket).
		Str("key", key).
		Int64("size", written).
		Msg("File uploaded to local storage")

	return &Object{
		Key:          key,
		Bucket:       bucket,
		Size:         info.Size(),
		ContentType:  opts.ContentType,
		LastModified: info.ModTime(),
		ETag:         etag,
		Metadata:     opts.Metadata,
	}, nil
}

// limitedReadCloser wraps a Reader with a Closer
type limitedReadCloser struct {
	reader io.Reader
	closer io.Closer
}

func (l *limitedReadCloser) Read(p []byte) (n int, err error) {
	return l.reader.Read(p)
}

func (l *limitedReadCloser) Close() error {
	return l.closer.Close()
}

// Download downloads a file from local storage
func (ls *LocalStorage) Download(ctx context.Context, bucket, key string, opts *DownloadOptions) (io.ReadCloser, *Object, error) {
	filePath, err := ls.getPath(bucket, key)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid path: %w", err)
	}

	// Check if file exists
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("object not found")
		}
		return nil, nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}

	// Load metadata
	metadata := make(map[string]string)
	contentType := "application/octet-stream"
	metaPath := filePath + ".meta"
	if metaData, err := os.ReadFile(metaPath); err == nil {
		lines := strings.Split(string(metaData), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				if parts[0] == "content-type" {
					contentType = parts[1]
				} else {
					metadata[parts[0]] = parts[1]
				}
			}
		}
	}

	totalSize := info.Size()
	var reader io.ReadCloser = file

	// Handle Range header for partial content requests
	if opts != nil && opts.Range != "" {
		var start, end int64
		if _, err := fmt.Sscanf(opts.Range, "bytes=%d-%d", &start, &end); err == nil {
			// Validate range
			if start < 0 {
				start = 0
			}
			if end >= totalSize {
				end = totalSize - 1
			}
			if start > end || start >= totalSize {
				_ = file.Close()
				return nil, nil, fmt.Errorf("invalid range: requested range not satisfiable")
			}

			// Seek to start position
			if _, err := file.Seek(start, io.SeekStart); err != nil {
				_ = file.Close()
				return nil, nil, fmt.Errorf("failed to seek: %w", err)
			}

			// Create limited reader for the range
			length := end - start + 1
			reader = &limitedReadCloser{
				reader: io.LimitReader(file, length),
				closer: file,
			}
			totalSize = length
		}
	}

	object := &Object{
		Key:          key,
		Bucket:       bucket,
		Size:         totalSize,
		ContentType:  contentType,
		LastModified: info.ModTime(),
		Metadata:     metadata,
	}

	return reader, object, nil
}

// Delete deletes a file from local storage
func (ls *LocalStorage) Delete(ctx context.Context, bucket, key string) error {
	filePath, err := ls.getPath(bucket, key)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Delete the file
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("object not found")
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// Delete metadata file if it exists
	metaPath := filePath + ".meta"
	_ = os.Remove(metaPath)

	log.Debug().
		Str("bucket", bucket).
		Str("key", key).
		Msg("File deleted from local storage")

	return nil
}

// Exists checks if a file exists
func (ls *LocalStorage) Exists(ctx context.Context, bucket, key string) (bool, error) {
	filePath, err := ls.getPath(bucket, key)
	if err != nil {
		return false, fmt.Errorf("invalid path: %w", err)
	}
	_, err = os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetObject gets object metadata without downloading the file
func (ls *LocalStorage) GetObject(ctx context.Context, bucket, key string) (*Object, error) {
	filePath, err := ls.getPath(bucket, key)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("object not found")
		}
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Load metadata
	metadata := make(map[string]string)
	contentType := "application/octet-stream"
	metaPath := filePath + ".meta"
	if metaData, err := os.ReadFile(metaPath); err == nil {
		lines := strings.Split(string(metaData), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				if parts[0] == "content-type" {
					contentType = parts[1]
				} else {
					metadata[parts[0]] = parts[1]
				}
			}
		}
	}

	return &Object{
		Key:          key,
		Bucket:       bucket,
		Size:         info.Size(),
		ContentType:  contentType,
		LastModified: info.ModTime(),
		Metadata:     metadata,
	}, nil
}

// List lists objects in a bucket
func (ls *LocalStorage) List(ctx context.Context, bucket string, opts *ListOptions) (*ListResult, error) {
	if opts == nil {
		opts = &ListOptions{MaxKeys: 1000}
	}
	if opts.MaxKeys == 0 {
		opts.MaxKeys = 1000
	}

	// Validate bucket name
	if err := validatePathComponent(bucket); err != nil {
		return nil, fmt.Errorf("invalid bucket: %w", err)
	}

	// Validate prefix if provided
	if opts.Prefix != "" {
		prefixParts := strings.Split(filepath.ToSlash(opts.Prefix), "/")
		for _, part := range prefixParts {
			if part == "" {
				continue
			}
			if err := validatePathComponent(part); err != nil {
				return nil, fmt.Errorf("invalid prefix: %w", err)
			}
		}
	}

	bucketPath := filepath.Join(ls.basePath, bucket)

	// Check if bucket exists
	if _, err := os.Stat(bucketPath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("bucket not found")
		}
		return nil, err
	}

	var objects []Object
	prefixes := make(map[string]bool)

	searchPath := bucketPath
	if opts.Prefix != "" {
		searchPath = filepath.Join(bucketPath, opts.Prefix)
		// Double-check the searchPath is still within bucket
		cleanSearch := filepath.Clean(searchPath)
		cleanBucket := filepath.Clean(bucketPath)
		if !strings.HasPrefix(cleanSearch, cleanBucket) {
			return nil, fmt.Errorf("prefix escapes bucket directory")
		}
	}

	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Skip metadata files
		if strings.HasSuffix(path, ".meta") {
			return nil
		}

		// Get relative path from bucket
		relPath, err := filepath.Rel(bucketPath, path)
		if err != nil {
			return err
		}

		// Convert to forward slashes for consistency
		key := filepath.ToSlash(relPath)

		// Apply prefix filter
		if opts.Prefix != "" && !strings.HasPrefix(key, opts.Prefix) {
			return nil
		}

		// Apply delimiter (for directory-like listing)
		if opts.Delimiter != "" {
			afterPrefix := strings.TrimPrefix(key, opts.Prefix)
			if idx := strings.Index(afterPrefix, opts.Delimiter); idx != -1 {
				// This is a "directory"
				prefix := opts.Prefix + afterPrefix[:idx+1]
				prefixes[prefix] = true
				return nil
			}
		}

		// Apply max keys limit
		if len(objects) >= opts.MaxKeys {
			return filepath.SkipDir
		}

		objects = append(objects, Object{
			Key:          key,
			Bucket:       bucket,
			Size:         info.Size(),
			LastModified: info.ModTime(),
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	// Convert prefixes map to slice
	commonPrefixes := make([]string, 0, len(prefixes))
	for prefix := range prefixes {
		commonPrefixes = append(commonPrefixes, prefix)
	}

	return &ListResult{
		Objects:        objects,
		CommonPrefixes: commonPrefixes,
		IsTruncated:    len(objects) == opts.MaxKeys,
	}, nil
}

// CreateBucket creates a new bucket
func (ls *LocalStorage) CreateBucket(ctx context.Context, bucket string) error {
	bucketPath := filepath.Join(ls.basePath, bucket)

	// Check if bucket already exists
	if _, err := os.Stat(bucketPath); err == nil {
		return fmt.Errorf("bucket already exists")
	}

	// Create bucket directory
	if err := os.MkdirAll(bucketPath, 0755); err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	log.Info().Str("bucket", bucket).Msg("Bucket created")
	return nil
}

// DeleteBucket deletes a bucket (must be empty)
func (ls *LocalStorage) DeleteBucket(ctx context.Context, bucket string) error {
	bucketPath := filepath.Join(ls.basePath, bucket)

	// Check if bucket exists
	if _, err := os.Stat(bucketPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("bucket not found")
		}
		return err
	}

	// Check if bucket contains any files (not just directories)
	hasFiles := false
	err := filepath.Walk(bucketPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip the bucket directory itself and any metadata files
		if path != bucketPath && !info.IsDir() && !strings.HasSuffix(path, ".meta") {
			hasFiles = true
			return filepath.SkipDir // Stop walking once we find a file
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to check bucket contents: %w", err)
	}

	if hasFiles {
		return fmt.Errorf("bucket is not empty")
	}

	// Delete bucket directory and all empty subdirectories
	if err := os.RemoveAll(bucketPath); err != nil {
		return fmt.Errorf("failed to delete bucket: %w", err)
	}

	log.Info().Str("bucket", bucket).Msg("Bucket deleted")
	return nil
}

// BucketExists checks if a bucket exists
func (ls *LocalStorage) BucketExists(ctx context.Context, bucket string) (bool, error) {
	bucketPath := filepath.Join(ls.basePath, bucket)
	info, err := os.Stat(bucketPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.IsDir(), nil
}

// ListBuckets lists all buckets
func (ls *LocalStorage) ListBuckets(ctx context.Context) ([]string, error) {
	entries, err := os.ReadDir(ls.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}

	var buckets []string
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			buckets = append(buckets, entry.Name())
		}
	}

	return buckets, nil
}

// GenerateSignedURL generates a signed URL for temporary access to local storage
func (ls *LocalStorage) GenerateSignedURL(ctx context.Context, bucket, key string, opts *SignedURLOptions) (string, error) {
	if ls.signingSecret == "" {
		return "", fmt.Errorf("signing secret not configured for local storage")
	}
	if ls.baseURL == "" {
		return "", fmt.Errorf("base URL not configured for local storage")
	}

	if opts == nil {
		opts = &SignedURLOptions{
			ExpiresIn: 15 * time.Minute,
			Method:    "GET",
		}
	}
	if opts.ExpiresIn == 0 {
		opts.ExpiresIn = 15 * time.Minute
	}
	if opts.Method == "" {
		opts.Method = "GET"
	}

	// Create token data
	token := signedURLToken{
		Bucket:    bucket,
		Key:       key,
		ExpiresAt: time.Now().Add(opts.ExpiresIn).Unix(),
		Method:    opts.Method,
		// Include transform options if specified
		TrWidth:   opts.TransformWidth,
		TrHeight:  opts.TransformHeight,
		TrFormat:  opts.TransformFormat,
		TrQuality: opts.TransformQuality,
		TrFit:     opts.TransformFit,
	}

	// Encode token to JSON
	tokenJSON, err := json.Marshal(token)
	if err != nil {
		return "", fmt.Errorf("failed to encode token: %w", err)
	}

	// Sign the token with HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(ls.signingSecret))
	mac.Write(tokenJSON)
	signature := mac.Sum(nil)

	// Combine token and signature, then base64 encode
	combined := append(tokenJSON, signature...)
	encodedToken := base64.URLEncoding.EncodeToString(combined)

	// Build the signed URL
	signedURL := fmt.Sprintf("%s/api/v1/storage/object?token=%s", ls.baseURL, url.QueryEscape(encodedToken))

	return signedURL, nil
}

// ValidateSignedToken validates a signed URL token and returns the bucket and key
func (ls *LocalStorage) ValidateSignedToken(token string) (bucket, key, method string, err error) {
	if ls.signingSecret == "" {
		return "", "", "", fmt.Errorf("signing secret not configured")
	}

	// Decode the base64 token
	decoded, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid token encoding")
	}

	// Token must be at least 32 bytes (signature length) + some JSON
	if len(decoded) < 33 {
		return "", "", "", fmt.Errorf("invalid token length")
	}

	// Split token and signature (last 32 bytes are the HMAC-SHA256 signature)
	tokenJSON := decoded[:len(decoded)-32]
	providedSig := decoded[len(decoded)-32:]

	// Verify signature
	mac := hmac.New(sha256.New, []byte(ls.signingSecret))
	mac.Write(tokenJSON)
	expectedSig := mac.Sum(nil)

	if !hmac.Equal(providedSig, expectedSig) {
		return "", "", "", fmt.Errorf("invalid token signature")
	}

	// Parse token data
	var tokenData signedURLToken
	if err := json.Unmarshal(tokenJSON, &tokenData); err != nil {
		return "", "", "", fmt.Errorf("invalid token data")
	}

	// Check expiration
	if time.Now().Unix() > tokenData.ExpiresAt {
		return "", "", "", fmt.Errorf("token expired")
	}

	return tokenData.Bucket, tokenData.Key, tokenData.Method, nil
}

// ValidateSignedTokenFull validates a signed URL token and returns the full result including transforms
func (ls *LocalStorage) ValidateSignedTokenFull(token string) (*SignedTokenResult, error) {
	if ls.signingSecret == "" {
		return nil, fmt.Errorf("signing secret not configured")
	}

	// Decode the base64 token
	decoded, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token encoding")
	}

	// Token must be at least 32 bytes (signature length) + some JSON
	if len(decoded) < 33 {
		return nil, fmt.Errorf("invalid token length")
	}

	// Split token and signature (last 32 bytes are the HMAC-SHA256 signature)
	tokenJSON := decoded[:len(decoded)-32]
	providedSig := decoded[len(decoded)-32:]

	// Verify signature
	mac := hmac.New(sha256.New, []byte(ls.signingSecret))
	mac.Write(tokenJSON)
	expectedSig := mac.Sum(nil)

	if !hmac.Equal(providedSig, expectedSig) {
		return nil, fmt.Errorf("invalid token signature")
	}

	// Parse token data
	var tokenData signedURLToken
	if err := json.Unmarshal(tokenJSON, &tokenData); err != nil {
		return nil, fmt.Errorf("invalid token data")
	}

	// Check expiration
	if time.Now().Unix() > tokenData.ExpiresAt {
		return nil, fmt.Errorf("token expired")
	}

	return &SignedTokenResult{
		Bucket:           tokenData.Bucket,
		Key:              tokenData.Key,
		Method:           tokenData.Method,
		TransformWidth:   tokenData.TrWidth,
		TransformHeight:  tokenData.TrHeight,
		TransformFormat:  tokenData.TrFormat,
		TransformQuality: tokenData.TrQuality,
		TransformFit:     tokenData.TrFit,
	}, nil
}

// CopyObject copies an object within storage
func (ls *LocalStorage) CopyObject(ctx context.Context, srcBucket, srcKey, destBucket, destKey string) error {
	srcPath, err := ls.getPath(srcBucket, srcKey)
	if err != nil {
		return fmt.Errorf("invalid source path: %w", err)
	}
	destPath, err := ls.getPath(destBucket, destKey)
	if err != nil {
		return fmt.Errorf("invalid destination path: %w", err)
	}

	// Create destination directory
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Open source file
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer func() { _ = src.Close() }()

	// Create destination file
	dest, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() { _ = dest.Close() }()

	// Copy data
	if _, err := io.Copy(dest, src); err != nil {
		_ = os.Remove(destPath)
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Copy metadata if it exists
	srcMeta := srcPath + ".meta"
	if _, err := os.Stat(srcMeta); err == nil {
		destMeta := destPath + ".meta"
		srcMetaData, _ := os.ReadFile(srcMeta)
		_ = os.WriteFile(destMeta, srcMetaData, 0644)
	}

	return nil
}

// MoveObject moves an object (copy + delete)
func (ls *LocalStorage) MoveObject(ctx context.Context, srcBucket, srcKey, destBucket, destKey string) error {
	// Copy the object
	if err := ls.CopyObject(ctx, srcBucket, srcKey, destBucket, destKey); err != nil {
		return err
	}

	// Delete the source
	if err := ls.Delete(ctx, srcBucket, srcKey); err != nil {
		// Try to clean up the destination
		_ = ls.Delete(ctx, destBucket, destKey)
		return fmt.Errorf("failed to delete source after copy: %w", err)
	}

	return nil
}

// getChunkedUploadDir returns the path to the chunked upload directory for a session
func (ls *LocalStorage) getChunkedUploadDir(uploadID string) string {
	return filepath.Join(ls.basePath, ".chunked", uploadID)
}

// getChunkPath returns the path to a specific chunk file
func (ls *LocalStorage) getChunkPath(uploadID string, chunkIndex int) string {
	return filepath.Join(ls.getChunkedUploadDir(uploadID), fmt.Sprintf("chunk_%06d", chunkIndex))
}

// InitChunkedUpload starts a new chunked upload session for local storage
func (ls *LocalStorage) InitChunkedUpload(ctx context.Context, bucket, key string, totalSize int64, chunkSize int64, opts *UploadOptions) (*ChunkedUploadSession, error) {
	// Validate bucket and key
	if _, err := ls.getPath(bucket, key); err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Generate cryptographically secure upload ID to prevent session hijacking
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, fmt.Errorf("failed to generate secure upload ID: %w", err)
	}
	uploadID := hex.EncodeToString(randomBytes)

	// Create chunked upload directory
	chunkDir := ls.getChunkedUploadDir(uploadID)
	if err := os.MkdirAll(chunkDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create chunk directory: %w", err)
	}

	totalChunks := int((totalSize + chunkSize - 1) / chunkSize)

	session := &ChunkedUploadSession{
		UploadID:        uploadID,
		Bucket:          bucket,
		Key:             key,
		TotalSize:       totalSize,
		ChunkSize:       chunkSize,
		TotalChunks:     totalChunks,
		CompletedChunks: []int{},
		Status:          "active",
		CreatedAt:       time.Now(),
		ExpiresAt:       time.Now().Add(24 * time.Hour),
	}

	if opts != nil {
		session.ContentType = opts.ContentType
		session.Metadata = opts.Metadata
		session.CacheControl = opts.CacheControl
	}

	// Save session metadata to a file
	sessionPath := filepath.Join(chunkDir, "session.json")
	sessionData, err := json.Marshal(session)
	if err != nil {
		_ = os.RemoveAll(chunkDir)
		return nil, fmt.Errorf("failed to marshal session: %w", err)
	}
	if err := os.WriteFile(sessionPath, sessionData, 0644); err != nil {
		_ = os.RemoveAll(chunkDir)
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	log.Debug().
		Str("uploadID", uploadID).
		Str("bucket", bucket).
		Str("key", key).
		Int64("totalSize", totalSize).
		Int("totalChunks", totalChunks).
		Msg("Chunked upload session initialized")

	return session, nil
}

// UploadChunk uploads a single chunk of data for local storage
func (ls *LocalStorage) UploadChunk(ctx context.Context, session *ChunkedUploadSession, chunkIndex int, data io.Reader, size int64) (*ChunkResult, error) {
	if session == nil {
		return nil, fmt.Errorf("session is nil")
	}

	if chunkIndex < 0 || chunkIndex >= session.TotalChunks {
		return nil, fmt.Errorf("invalid chunk index: %d (total chunks: %d)", chunkIndex, session.TotalChunks)
	}

	// Verify session directory exists
	chunkDir := ls.getChunkedUploadDir(session.UploadID)
	if _, err := os.Stat(chunkDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("upload session not found")
	}

	// Create chunk file
	chunkPath := ls.getChunkPath(session.UploadID, chunkIndex)
	file, err := os.Create(chunkPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create chunk file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Calculate MD5 hash while writing
	hash := md5.New()
	writer := io.MultiWriter(file, hash)

	// Copy data to chunk file
	written, err := io.Copy(writer, data)
	if err != nil {
		_ = os.Remove(chunkPath)
		return nil, fmt.Errorf("failed to write chunk: %w", err)
	}

	etag := hex.EncodeToString(hash.Sum(nil))

	log.Debug().
		Str("uploadID", session.UploadID).
		Int("chunkIndex", chunkIndex).
		Int64("size", written).
		Msg("Chunk uploaded")

	return &ChunkResult{
		ChunkIndex: chunkIndex,
		ETag:       etag,
		Size:       written,
	}, nil
}

// CompleteChunkedUpload finalizes the upload and assembles the file for local storage
func (ls *LocalStorage) CompleteChunkedUpload(ctx context.Context, session *ChunkedUploadSession) (*Object, error) {
	if session == nil {
		return nil, fmt.Errorf("session is nil")
	}

	chunkDir := ls.getChunkedUploadDir(session.UploadID)

	// Verify all chunks exist
	for i := 0; i < session.TotalChunks; i++ {
		chunkPath := ls.getChunkPath(session.UploadID, i)
		if _, err := os.Stat(chunkPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("missing chunk %d", i)
		}
	}

	// Get destination path
	destPath, err := ls.getPath(session.Bucket, session.Key)
	if err != nil {
		return nil, fmt.Errorf("invalid destination path: %w", err)
	}

	// Create parent directories
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Create destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() { _ = destFile.Close() }()

	// Calculate MD5 hash while assembling
	hash := md5.New()
	writer := io.MultiWriter(destFile, hash)

	// Concatenate all chunks
	var totalWritten int64
	for i := 0; i < session.TotalChunks; i++ {
		chunkPath := ls.getChunkPath(session.UploadID, i)
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			_ = destFile.Close()
			_ = os.Remove(destPath)
			return nil, fmt.Errorf("failed to open chunk %d: %w", i, err)
		}

		written, err := io.Copy(writer, chunkFile)
		_ = chunkFile.Close()
		if err != nil {
			_ = destFile.Close()
			_ = os.Remove(destPath)
			return nil, fmt.Errorf("failed to copy chunk %d: %w", i, err)
		}
		totalWritten += written
	}

	etag := hex.EncodeToString(hash.Sum(nil))

	// Save metadata if present
	if len(session.Metadata) > 0 || session.ContentType != "" {
		metaPath := destPath + ".meta"
		metaData := ""
		for k, v := range session.Metadata {
			metaData += fmt.Sprintf("%s=%s\n", k, v)
		}
		if session.ContentType != "" {
			metaData += fmt.Sprintf("content-type=%s\n", session.ContentType)
		}
		_ = os.WriteFile(metaPath, []byte(metaData), 0644)
	}

	// Clean up chunk directory
	if err := os.RemoveAll(chunkDir); err != nil {
		log.Warn().Err(err).Str("uploadID", session.UploadID).Msg("Failed to clean up chunk directory")
	}

	// Get final file info
	info, err := os.Stat(destPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat final file: %w", err)
	}

	log.Info().
		Str("uploadID", session.UploadID).
		Str("bucket", session.Bucket).
		Str("key", session.Key).
		Int64("size", totalWritten).
		Msg("Chunked upload completed")

	return &Object{
		Key:          session.Key,
		Bucket:       session.Bucket,
		Size:         info.Size(),
		ContentType:  session.ContentType,
		LastModified: info.ModTime(),
		ETag:         etag,
		Metadata:     session.Metadata,
	}, nil
}

// AbortChunkedUpload cancels the upload and cleans up chunks for local storage
func (ls *LocalStorage) AbortChunkedUpload(ctx context.Context, session *ChunkedUploadSession) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}

	chunkDir := ls.getChunkedUploadDir(session.UploadID)

	// Remove the entire chunk directory
	if err := os.RemoveAll(chunkDir); err != nil {
		return fmt.Errorf("failed to remove chunk directory: %w", err)
	}

	log.Info().
		Str("uploadID", session.UploadID).
		Msg("Chunked upload aborted")

	return nil
}

// GetChunkedUploadSession retrieves a chunked upload session from local storage
func (ls *LocalStorage) GetChunkedUploadSession(uploadID string) (*ChunkedUploadSession, error) {
	chunkDir := ls.getChunkedUploadDir(uploadID)
	sessionPath := filepath.Join(chunkDir, "session.json")

	sessionData, err := os.ReadFile(sessionPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("upload session not found")
		}
		return nil, fmt.Errorf("failed to read session: %w", err)
	}

	var session ChunkedUploadSession
	if err := json.Unmarshal(sessionData, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	// Update completed chunks by checking which chunk files exist
	session.CompletedChunks = []int{}
	for i := 0; i < session.TotalChunks; i++ {
		chunkPath := ls.getChunkPath(uploadID, i)
		if _, err := os.Stat(chunkPath); err == nil {
			session.CompletedChunks = append(session.CompletedChunks, i)
		}
	}

	return &session, nil
}

// UpdateChunkedUploadSession updates a session file after chunk upload
func (ls *LocalStorage) UpdateChunkedUploadSession(session *ChunkedUploadSession) error {
	chunkDir := ls.getChunkedUploadDir(session.UploadID)
	sessionPath := filepath.Join(chunkDir, "session.json")

	sessionData, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := os.WriteFile(sessionPath, sessionData, 0644); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

// CleanupExpiredChunkedUploads removes expired chunked upload sessions and their files
// This should be called periodically to prevent storage leaks
func (ls *LocalStorage) CleanupExpiredChunkedUploads(ctx context.Context) (int, error) {
	chunkedDir := filepath.Join(ls.basePath, ".chunked")

	// Check if chunked directory exists
	if _, err := os.Stat(chunkedDir); os.IsNotExist(err) {
		return 0, nil // No chunked uploads to clean
	}

	entries, err := os.ReadDir(chunkedDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read chunked upload directory: %w", err)
	}

	cleaned := 0
	now := time.Now()

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return cleaned, ctx.Err()
		default:
		}

		uploadID := entry.Name()
		sessionPath := filepath.Join(chunkedDir, uploadID, "session.json")

		sessionData, err := os.ReadFile(sessionPath)
		if err != nil {
			// If we can't read the session, check directory age
			// Remove directories older than 48 hours with no valid session
			info, statErr := entry.Info()
			if statErr == nil && now.Sub(info.ModTime()) > 48*time.Hour {
				if rmErr := os.RemoveAll(filepath.Join(chunkedDir, uploadID)); rmErr == nil {
					cleaned++
					log.Debug().Str("upload_id", uploadID).Msg("Removed orphaned chunked upload directory")
				}
			}
			continue
		}

		var session ChunkedUploadSession
		if err := json.Unmarshal(sessionData, &session); err != nil {
			// Invalid session, remove if old
			info, statErr := entry.Info()
			if statErr == nil && now.Sub(info.ModTime()) > 48*time.Hour {
				if rmErr := os.RemoveAll(filepath.Join(chunkedDir, uploadID)); rmErr == nil {
					cleaned++
					log.Debug().Str("upload_id", uploadID).Msg("Removed chunked upload with invalid session")
				}
			}
			continue
		}

		// Check if session is expired
		if now.After(session.ExpiresAt) {
			if err := os.RemoveAll(filepath.Join(chunkedDir, uploadID)); err == nil {
				cleaned++
				log.Debug().
					Str("upload_id", uploadID).
					Str("bucket", session.Bucket).
					Str("key", session.Key).
					Time("expired_at", session.ExpiresAt).
					Msg("Removed expired chunked upload session")
			} else {
				log.Warn().Err(err).Str("upload_id", uploadID).Msg("Failed to remove expired chunked upload")
			}
		}
	}

	if cleaned > 0 {
		log.Info().Int("cleaned", cleaned).Msg("Cleaned up expired chunked upload sessions")
	}

	return cleaned, nil
}

// StartChunkedUploadCleanup starts a background goroutine to periodically clean up
// expired chunked upload sessions. Call this once when initializing the storage.
func (ls *LocalStorage) StartChunkedUploadCleanup(ctx context.Context) {
	go func() {
		// Run cleanup every hour
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		// Also run once on startup after a short delay
		time.Sleep(30 * time.Second)
		if _, err := ls.CleanupExpiredChunkedUploads(ctx); err != nil {
			log.Error().Err(err).Msg("Failed to cleanup expired chunked uploads on startup")
		}

		for {
			select {
			case <-ticker.C:
				if _, err := ls.CleanupExpiredChunkedUploads(ctx); err != nil {
					log.Error().Err(err).Msg("Failed to cleanup expired chunked uploads")
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
