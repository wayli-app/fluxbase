package storage

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

// LocalStorage implements the Storage interface using local filesystem
type LocalStorage struct {
	basePath string
}

// NewLocalStorage creates a new local filesystem storage provider
func NewLocalStorage(basePath string) (*LocalStorage, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &LocalStorage{
		basePath: basePath,
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
	os.Remove(testFile)

	return nil
}

// getPath returns the full filesystem path for a bucket/key
func (ls *LocalStorage) getPath(bucket, key string) string {
	return filepath.Join(ls.basePath, bucket, key)
}

// Upload uploads a file to local storage
func (ls *LocalStorage) Upload(ctx context.Context, bucket, key string, data io.Reader, size int64, opts *UploadOptions) (*Object, error) {
	if opts == nil {
		opts = &UploadOptions{}
	}

	// Create bucket directory if it doesn't exist
	bucketPath := filepath.Join(ls.basePath, bucket)
	if err := os.MkdirAll(bucketPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create bucket directory: %w", err)
	}

	filePath := ls.getPath(bucket, key)

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
	defer file.Close()

	// Calculate MD5 hash while writing
	hash := md5.New()
	writer := io.MultiWriter(file, hash)

	// Copy data to file
	written, err := io.Copy(writer, data)
	if err != nil {
		os.Remove(filePath)
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
	filePath := ls.getPath(bucket, key)

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
				file.Close()
				return nil, nil, fmt.Errorf("invalid range: requested range not satisfiable")
			}

			// Seek to start position
			if _, err := file.Seek(start, io.SeekStart); err != nil {
				file.Close()
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
	filePath := ls.getPath(bucket, key)

	// Delete the file
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("object not found")
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// Delete metadata file if it exists
	metaPath := filePath + ".meta"
	os.Remove(metaPath)

	log.Debug().
		Str("bucket", bucket).
		Str("key", key).
		Msg("File deleted from local storage")

	return nil
}

// Exists checks if a file exists
func (ls *LocalStorage) Exists(ctx context.Context, bucket, key string) (bool, error) {
	filePath := ls.getPath(bucket, key)
	_, err := os.Stat(filePath)
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
	filePath := ls.getPath(bucket, key)

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

// GenerateSignedURL generates a signed URL (not supported for local storage)
func (ls *LocalStorage) GenerateSignedURL(ctx context.Context, bucket, key string, opts *SignedURLOptions) (string, error) {
	return "", fmt.Errorf("signed URLs not supported for local storage")
}

// CopyObject copies an object within storage
func (ls *LocalStorage) CopyObject(ctx context.Context, srcBucket, srcKey, destBucket, destKey string) error {
	srcPath := ls.getPath(srcBucket, srcKey)
	destPath := ls.getPath(destBucket, destKey)

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
	defer src.Close()

	// Create destination file
	dest, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dest.Close()

	// Copy data
	if _, err := io.Copy(dest, src); err != nil {
		os.Remove(destPath)
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
