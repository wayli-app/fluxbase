package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog/log"
)

// S3Storage implements the Storage interface using S3-compatible storage (AWS S3, MinIO, etc.)
type S3Storage struct {
	client *minio.Client
	core   *minio.Core // Core client for low-level operations like multipart upload
	region string
}

// NewS3Storage creates a new S3-compatible storage provider
// Works with AWS S3, MinIO, Wasabi, DigitalOcean Spaces, and other S3-compatible services
func NewS3Storage(endpoint, accessKey, secretKey, region string, useSSL bool, forcePathStyle bool) (*S3Storage, error) {
	// Determine bucket lookup style
	// Path-style: endpoint/bucket (required for MinIO, R2, Spaces, etc.)
	// DNS-style: bucket.endpoint (AWS S3 default)
	bucketLookup := minio.BucketLookupAuto
	if forcePathStyle {
		bucketLookup = minio.BucketLookupPath
	}

	// Create MinIO client (works with S3-compatible services)
	client, err := minio.New(endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure:       useSSL,
		Region:       region,
		BucketLookup: bucketLookup,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	// Create Core client for low-level operations (multipart upload)
	core := &minio.Core{Client: client}

	log.Info().
		Str("endpoint", endpoint).
		Str("region", region).
		Bool("ssl", useSSL).
		Bool("force_path_style", forcePathStyle).
		Msg("S3-compatible storage initialized")

	return &S3Storage{
		client: client,
		core:   core,
		region: region,
	}, nil
}

// Name returns the provider name
func (s3 *S3Storage) Name() string {
	return "s3"
}

// Health checks if the storage is healthy
func (s3 *S3Storage) Health(ctx context.Context) error {
	// Try to list buckets as health check
	_, err := s3.client.ListBuckets(ctx)
	if err != nil {
		return fmt.Errorf("S3 health check failed: %w", err)
	}
	return nil
}

// Upload uploads a file to S3
func (s3 *S3Storage) Upload(ctx context.Context, bucket, key string, data io.Reader, size int64, opts *UploadOptions) (*Object, error) {
	if opts == nil {
		opts = &UploadOptions{}
	}

	// Prepare upload options
	putOpts := minio.PutObjectOptions{
		ContentType:     opts.ContentType,
		UserMetadata:    opts.Metadata,
		CacheControl:    opts.CacheControl,
		ContentEncoding: opts.ContentEncoding,
	}

	// Upload the object
	info, err := s3.client.PutObject(ctx, bucket, key, data, size, putOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to S3: %w", err)
	}

	log.Debug().
		Str("bucket", bucket).
		Str("key", key).
		Int64("size", info.Size).
		Msg("File uploaded to S3")

	return &Object{
		Key:          key,
		Bucket:       bucket,
		Size:         info.Size,
		ContentType:  opts.ContentType,
		LastModified: time.Now(),
		ETag:         info.ETag,
		Metadata:     opts.Metadata,
	}, nil
}

// Download downloads a file from S3
func (s3 *S3Storage) Download(ctx context.Context, bucket, key string, opts *DownloadOptions) (io.ReadCloser, *Object, error) {
	// Get object metadata first
	stat, err := s3.client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get object info: %w", err)
	}

	// Get the object
	getOpts := minio.GetObjectOptions{}
	if opts != nil {
		if opts.IfModifiedSince != nil {
			_ = getOpts.SetModified(*opts.IfModifiedSince)
		}
		if opts.IfUnmodifiedSince != nil {
			_ = getOpts.SetUnmodified(*opts.IfUnmodifiedSince)
		}
		if opts.IfMatch != "" {
			_ = getOpts.SetMatchETag(opts.IfMatch)
		}
		if opts.IfNoneMatch != "" {
			_ = getOpts.SetMatchETagExcept(opts.IfNoneMatch)
		}
		// Set range header if specified (e.g., "bytes=0-1023")
		if opts.Range != "" {
			// Parse the range string (e.g., "bytes=2-5")
			// SetRange expects offset and length
			// For "bytes=2-5", offset=2, and we want bytes 2,3,4,5 (4 bytes)
			// But MinIO's SetRange takes (offset, length-1) for the end byte
			var start, end int64
			if _, err := fmt.Sscanf(opts.Range, "bytes=%d-%d", &start, &end); err == nil {
				_ = getOpts.SetRange(start, end)
			}
		}
	}

	reader, err := s3.client.GetObject(ctx, bucket, key, getOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to download from S3: %w", err)
	}

	// Calculate actual size (considering Range request)
	actualSize := stat.Size
	if opts != nil && opts.Range != "" {
		var start, end int64
		if _, err := fmt.Sscanf(opts.Range, "bytes=%d-%d", &start, &end); err == nil {
			// Clamp to actual file size
			if end >= stat.Size {
				end = stat.Size - 1
			}
			if start <= end && start < stat.Size {
				actualSize = end - start + 1
			}
		}
	}

	object := &Object{
		Key:          key,
		Bucket:       bucket,
		Size:         actualSize,
		ContentType:  stat.ContentType,
		LastModified: stat.LastModified,
		ETag:         stat.ETag,
		Metadata:     stat.UserMetadata,
	}

	return reader, object, nil
}

// Delete deletes a file from S3
func (s3 *S3Storage) Delete(ctx context.Context, bucket, key string) error {
	err := s3.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete from S3: %w", err)
	}

	log.Debug().
		Str("bucket", bucket).
		Str("key", key).
		Msg("File deleted from S3")

	return nil
}

// Exists checks if a file exists
func (s3 *S3Storage) Exists(ctx context.Context, bucket, key string) (bool, error) {
	_, err := s3.client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetObject gets object metadata without downloading the file
func (s3 *S3Storage) GetObject(ctx context.Context, bucket, key string) (*Object, error) {
	stat, err := s3.client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object info: %w", err)
	}

	return &Object{
		Key:          key,
		Bucket:       bucket,
		Size:         stat.Size,
		ContentType:  stat.ContentType,
		LastModified: stat.LastModified,
		ETag:         stat.ETag,
		Metadata:     stat.UserMetadata,
	}, nil
}

// List lists objects in a bucket
func (s3 *S3Storage) List(ctx context.Context, bucket string, opts *ListOptions) (*ListResult, error) {
	if opts == nil {
		opts = &ListOptions{MaxKeys: 1000}
	}
	if opts.MaxKeys == 0 {
		opts.MaxKeys = 1000
	}

	listOpts := minio.ListObjectsOptions{
		Prefix:    opts.Prefix,
		Recursive: opts.Delimiter == "", // If no delimiter, list recursively
		MaxKeys:   opts.MaxKeys,
	}

	var objects []Object

	objectCh := s3.client.ListObjects(ctx, bucket, listOpts)
	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", object.Err)
		}

		objects = append(objects, Object{
			Key:          object.Key,
			Bucket:       bucket,
			Size:         object.Size,
			ContentType:  object.ContentType,
			LastModified: object.LastModified,
			ETag:         object.ETag,
		})

		// Stop if we reached max keys
		if len(objects) >= opts.MaxKeys {
			break
		}
	}

	return &ListResult{
		Objects:        objects,
		CommonPrefixes: []string{}, // MinIO SDK doesn't expose prefixes in the same way
		IsTruncated:    len(objects) == opts.MaxKeys,
	}, nil
}

// CreateBucket creates a new bucket
func (s3 *S3Storage) CreateBucket(ctx context.Context, bucket string) error {
	// Check if bucket already exists
	exists, err := s3.client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if exists {
		return fmt.Errorf("bucket already exists")
	}

	// Create the bucket
	err = s3.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{
		Region: s3.region,
	})
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	log.Info().Str("bucket", bucket).Msg("Bucket created")
	return nil
}

// DeleteBucket deletes a bucket (must be empty)
func (s3 *S3Storage) DeleteBucket(ctx context.Context, bucket string) error {
	// Check if bucket is empty
	objects := s3.client.ListObjects(ctx, bucket, minio.ListObjectsOptions{
		MaxKeys: 1,
	})
	for range objects {
		return fmt.Errorf("bucket is not empty")
	}

	// Delete the bucket
	err := s3.client.RemoveBucket(ctx, bucket)
	if err != nil {
		return fmt.Errorf("failed to delete bucket: %w", err)
	}

	log.Info().Str("bucket", bucket).Msg("Bucket deleted")
	return nil
}

// BucketExists checks if a bucket exists
func (s3 *S3Storage) BucketExists(ctx context.Context, bucket string) (bool, error) {
	exists, err := s3.client.BucketExists(ctx, bucket)
	if err != nil {
		return false, fmt.Errorf("failed to check bucket existence: %w", err)
	}
	return exists, nil
}

// ListBuckets lists all buckets
func (s3 *S3Storage) ListBuckets(ctx context.Context) ([]string, error) {
	buckets, err := s3.client.ListBuckets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}

	var names []string
	for _, bucket := range buckets {
		names = append(names, bucket.Name)
	}

	return names, nil
}

// GenerateSignedURL generates a presigned URL for temporary access
func (s3 *S3Storage) GenerateSignedURL(ctx context.Context, bucket, key string, opts *SignedURLOptions) (string, error) {
	if opts == nil {
		opts = &SignedURLOptions{
			ExpiresIn: 15 * time.Minute,
			Method:    "GET",
		}
	}

	var presignedURL *url.URL
	var err error

	switch strings.ToUpper(opts.Method) {
	case "GET":
		presignedURL, err = s3.client.PresignedGetObject(ctx, bucket, key, opts.ExpiresIn, nil)
	case "PUT":
		presignedURL, err = s3.client.PresignedPutObject(ctx, bucket, key, opts.ExpiresIn)
	case "DELETE":
		// MinIO SDK doesn't have PresignedDeleteObject, use custom HTTP method
		reqParams := make(url.Values)
		reqParams.Set("X-Amz-Expires", fmt.Sprintf("%d", int(opts.ExpiresIn.Seconds())))
		presignedURL, err = s3.client.PresignedGetObject(ctx, bucket, key, opts.ExpiresIn, reqParams)
		// Note: DELETE method would need to be set by the client making the request
	default:
		return "", fmt.Errorf("unsupported method: %s", opts.Method)
	}

	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %w", err)
	}

	return presignedURL.String(), nil
}

// CopyObject copies an object within S3
func (s3 *S3Storage) CopyObject(ctx context.Context, srcBucket, srcKey, destBucket, destKey string) error {
	srcOpts := minio.CopySrcOptions{
		Bucket: srcBucket,
		Object: srcKey,
	}

	destOpts := minio.CopyDestOptions{
		Bucket: destBucket,
		Object: destKey,
	}

	_, err := s3.client.CopyObject(ctx, destOpts, srcOpts)
	if err != nil {
		return fmt.Errorf("failed to copy object: %w", err)
	}

	log.Debug().
		Str("src_bucket", srcBucket).
		Str("src_key", srcKey).
		Str("dest_bucket", destBucket).
		Str("dest_key", destKey).
		Msg("Object copied in S3")

	return nil
}

// MoveObject moves an object (copy + delete)
func (s3 *S3Storage) MoveObject(ctx context.Context, srcBucket, srcKey, destBucket, destKey string) error {
	// Copy the object
	if err := s3.CopyObject(ctx, srcBucket, srcKey, destBucket, destKey); err != nil {
		return err
	}

	// Delete the source
	if err := s3.Delete(ctx, srcBucket, srcKey); err != nil {
		// Try to clean up the destination
		_ = s3.Delete(ctx, destBucket, destKey)
		return fmt.Errorf("failed to delete source after copy: %w", err)
	}

	log.Debug().
		Str("src_bucket", srcBucket).
		Str("src_key", srcKey).
		Str("dest_bucket", destBucket).
		Str("dest_key", destKey).
		Msg("Object moved in S3")

	return nil
}

// InitChunkedUpload starts a new chunked upload session using S3 multipart upload
func (s3s *S3Storage) InitChunkedUpload(ctx context.Context, bucket, key string, totalSize int64, chunkSize int64, opts *UploadOptions) (*ChunkedUploadSession, error) {
	// Prepare multipart upload options
	putOpts := minio.PutObjectOptions{}
	if opts != nil {
		putOpts.ContentType = opts.ContentType
		putOpts.UserMetadata = opts.Metadata
		putOpts.CacheControl = opts.CacheControl
		putOpts.ContentEncoding = opts.ContentEncoding
	}

	// Start multipart upload using Core client
	uploadID, err := s3s.core.NewMultipartUpload(ctx, bucket, key, putOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate multipart upload: %w", err)
	}

	totalChunks := int((totalSize + chunkSize - 1) / chunkSize)

	session := &ChunkedUploadSession{
		UploadID:        uploadID, // Use S3's uploadID directly as our session ID
		Bucket:          bucket,
		Key:             key,
		TotalSize:       totalSize,
		ChunkSize:       chunkSize,
		TotalChunks:     totalChunks,
		CompletedChunks: []int{},
		S3UploadID:      uploadID,
		S3PartETags:     make(map[int]string),
		Status:          "active",
		CreatedAt:       time.Now(),
		ExpiresAt:       time.Now().Add(24 * time.Hour),
	}

	if opts != nil {
		session.ContentType = opts.ContentType
		session.Metadata = opts.Metadata
		session.CacheControl = opts.CacheControl
	}

	log.Debug().
		Str("uploadID", uploadID).
		Str("bucket", bucket).
		Str("key", key).
		Int64("totalSize", totalSize).
		Int("totalChunks", totalChunks).
		Msg("S3 multipart upload session initialized")

	return session, nil
}

// UploadChunk uploads a single chunk using S3 multipart upload
func (s3s *S3Storage) UploadChunk(ctx context.Context, session *ChunkedUploadSession, chunkIndex int, data io.Reader, size int64) (*ChunkResult, error) {
	if session == nil {
		return nil, fmt.Errorf("session is nil")
	}

	if chunkIndex < 0 || chunkIndex >= session.TotalChunks {
		return nil, fmt.Errorf("invalid chunk index: %d (total chunks: %d)", chunkIndex, session.TotalChunks)
	}

	// S3 part numbers are 1-indexed
	partNumber := chunkIndex + 1

	// Upload the part using Core client
	objectPart, err := s3s.core.PutObjectPart(
		ctx,
		session.Bucket,
		session.Key,
		session.S3UploadID,
		partNumber,
		data,
		size,
		minio.PutObjectPartOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upload part %d: %w", partNumber, err)
	}

	log.Debug().
		Str("uploadID", session.S3UploadID).
		Int("partNumber", partNumber).
		Int64("size", objectPart.Size).
		Str("etag", objectPart.ETag).
		Msg("S3 chunk uploaded")

	return &ChunkResult{
		ChunkIndex: chunkIndex,
		ETag:       objectPart.ETag,
		Size:       objectPart.Size,
	}, nil
}

// CompleteChunkedUpload finalizes the upload by completing the S3 multipart upload
func (s3s *S3Storage) CompleteChunkedUpload(ctx context.Context, session *ChunkedUploadSession) (*Object, error) {
	if session == nil {
		return nil, fmt.Errorf("session is nil")
	}

	// Build list of completed parts
	completeParts := make([]minio.CompletePart, 0, session.TotalChunks)
	for i := 0; i < session.TotalChunks; i++ {
		etag, ok := session.S3PartETags[i]
		if !ok {
			return nil, fmt.Errorf("missing ETag for chunk %d", i)
		}
		completeParts = append(completeParts, minio.CompletePart{
			PartNumber: i + 1, // 1-indexed
			ETag:       etag,
		})
	}

	// Complete the multipart upload using Core client
	uploadInfo, err := s3s.core.CompleteMultipartUpload(
		ctx,
		session.Bucket,
		session.Key,
		session.S3UploadID,
		completeParts,
		minio.PutObjectOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	log.Info().
		Str("uploadID", session.S3UploadID).
		Str("bucket", session.Bucket).
		Str("key", session.Key).
		Int64("size", session.TotalSize).
		Msg("S3 multipart upload completed")

	return &Object{
		Key:          session.Key,
		Bucket:       session.Bucket,
		Size:         session.TotalSize,
		ContentType:  session.ContentType,
		LastModified: time.Now(),
		ETag:         uploadInfo.ETag,
		Metadata:     session.Metadata,
	}, nil
}

// AbortChunkedUpload cancels the S3 multipart upload and cleans up
func (s3s *S3Storage) AbortChunkedUpload(ctx context.Context, session *ChunkedUploadSession) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}

	err := s3s.core.AbortMultipartUpload(ctx, session.Bucket, session.Key, session.S3UploadID)
	if err != nil {
		return fmt.Errorf("failed to abort multipart upload: %w", err)
	}

	log.Info().
		Str("uploadID", session.S3UploadID).
		Msg("S3 multipart upload aborted")

	return nil
}

// CleanupExpiredMultipartUploads lists and aborts incomplete multipart uploads
// that are older than the specified max age. This prevents storage costs from
// orphaned multipart uploads that were never completed or aborted.
func (s3s *S3Storage) CleanupExpiredMultipartUploads(ctx context.Context, bucket string, maxAge time.Duration) (int, error) {
	cleaned := 0
	cutoff := time.Now().Add(-maxAge)

	// List incomplete multipart uploads
	// Note: This lists all incomplete uploads, not just those initiated by us
	for upload := range s3s.client.ListIncompleteUploads(ctx, bucket, "", true) {
		if upload.Err != nil {
			log.Warn().Err(upload.Err).Str("bucket", bucket).Msg("Error listing incomplete uploads")
			continue
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return cleaned, ctx.Err()
		default:
		}

		// Only abort uploads older than the cutoff
		if upload.Initiated.Before(cutoff) {
			err := s3s.core.AbortMultipartUpload(ctx, bucket, upload.Key, upload.UploadID)
			if err != nil {
				log.Warn().
					Err(err).
					Str("bucket", bucket).
					Str("key", upload.Key).
					Str("upload_id", upload.UploadID).
					Msg("Failed to abort expired multipart upload")
				continue
			}

			cleaned++
			log.Debug().
				Str("bucket", bucket).
				Str("key", upload.Key).
				Str("upload_id", upload.UploadID).
				Time("initiated", upload.Initiated).
				Msg("Aborted expired multipart upload")
		}
	}

	if cleaned > 0 {
		log.Info().
			Int("cleaned", cleaned).
			Str("bucket", bucket).
			Dur("max_age", maxAge).
			Msg("Cleaned up expired S3 multipart uploads")
	}

	return cleaned, nil
}

// StartMultipartUploadCleanup starts a background goroutine to periodically clean up
// expired multipart uploads across all buckets. Call this once when initializing the storage.
func (s3s *S3Storage) StartMultipartUploadCleanup(ctx context.Context, maxAge time.Duration) {
	go func() {
		// Run cleanup every hour
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		// Also run once on startup after a short delay
		time.Sleep(30 * time.Second)
		s3s.cleanupAllBuckets(ctx, maxAge)

		for {
			select {
			case <-ticker.C:
				s3s.cleanupAllBuckets(ctx, maxAge)
			case <-ctx.Done():
				return
			}
		}
	}()
}

// cleanupAllBuckets runs cleanup across all buckets
func (s3s *S3Storage) cleanupAllBuckets(ctx context.Context, maxAge time.Duration) {
	buckets, err := s3s.client.ListBuckets(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list buckets for multipart upload cleanup")
		return
	}

	totalCleaned := 0
	for _, bucket := range buckets {
		cleaned, err := s3s.CleanupExpiredMultipartUploads(ctx, bucket.Name, maxAge)
		if err != nil {
			log.Error().Err(err).Str("bucket", bucket.Name).Msg("Failed to cleanup multipart uploads")
			continue
		}
		totalCleaned += cleaned
	}

	if totalCleaned > 0 {
		log.Info().Int("total_cleaned", totalCleaned).Msg("Completed S3 multipart upload cleanup across all buckets")
	}
}
