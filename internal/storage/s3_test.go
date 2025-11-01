package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: These tests require a running MinIO instance or mock S3 service
// For true unit tests, we would need to implement a mock MinIO client
// These are integration-style tests that test against a real MinIO instance

// setupS3Storage creates an S3Storage instance for testing
// This requires a running MinIO instance (e.g., via Docker)
func setupS3Storage(t *testing.T) *S3Storage {
	t.Helper()

	// Check if we should skip S3 tests
	// Set SKIP_S3_TESTS=1 to skip these tests
	// These tests require a running MinIO instance at localhost:9000
	if testing.Short() {
		t.Skip("Skipping S3 tests in short mode")
	}

	// Try to connect to local MinIO instance
	// docker run -p 9000:9000 -p 9001:9001 -e "MINIO_ROOT_USER=minioadmin" -e "MINIO_ROOT_PASSWORD=minioadmin" minio/minio server /data --console-address ":9001"
	s3, err := NewS3Storage("minio:9000", "minioadmin", "minioadmin", "us-east-1", false)
	if err != nil {
		t.Skipf("Skipping S3 tests: cannot connect to MinIO at minio:9000: %v", err)
	}

	// Test connection by listing buckets
	ctx := context.Background()
	_, err = s3.ListBuckets(ctx)
	if err != nil {
		t.Skipf("Skipping S3 tests: MinIO not available: %v", err)
	}

	return s3
}

// generateUniqueBucketName creates a unique bucket name for test isolation
func generateUniqueBucketName(prefix string) string {
	// Use timestamp + random number to ensure uniqueness across parallel tests
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), rand.Int63n(1000000))
}

// cleanupS3Bucket removes all objects from a bucket for cleanup
func cleanupS3Bucket(t *testing.T, s3 *S3Storage, bucket string) {
	t.Helper()
	ctx := context.Background()

	// List all objects
	result, _ := s3.List(ctx, bucket, &ListOptions{})
	if result != nil {
		for _, obj := range result.Objects {
			_ = s3.Delete(ctx, bucket, obj.Key)
		}
	}

	// Delete the bucket - ignore errors as bucket might not exist
	_ = s3.DeleteBucket(ctx, bucket)
}

func TestS3Storage_Name(t *testing.T) {
	s3 := setupS3Storage(t)
	assert.Equal(t, "s3", s3.Name())
}

func TestS3Storage_Health(t *testing.T) {
	s3 := setupS3Storage(t)
	ctx := context.Background()

	err := s3.Health(ctx)
	assert.NoError(t, err)
}

func TestS3Storage_UploadAndDownload(t *testing.T) {
	s3 := setupS3Storage(t)
	ctx := context.Background()
	bucket := generateUniqueBucketName("test-upload")
	key := "test-file.txt"

	// Create bucket
	err := s3.CreateBucket(ctx, bucket)
	require.NoError(t, err)
	defer cleanupS3Bucket(t, s3, bucket)

	// Upload file
	content := []byte("Hello, World!")
	opts := &UploadOptions{
		ContentType: "text/plain",
		Metadata:    map[string]string{"custom-key": "custom-value"},
	}

	obj, err := s3.Upload(ctx, bucket, key, bytes.NewReader(content), int64(len(content)), opts)
	require.NoError(t, err)
	assert.Equal(t, key, obj.Key)
	assert.Equal(t, bucket, obj.Bucket)
	assert.Equal(t, int64(len(content)), obj.Size)
	assert.Equal(t, "text/plain", obj.ContentType)
	assert.NotEmpty(t, obj.ETag)

	// Download file
	reader, downloadObj, err := s3.Download(ctx, bucket, key, &DownloadOptions{})
	require.NoError(t, err)
	defer reader.Close()

	assert.Equal(t, key, downloadObj.Key)
	assert.Equal(t, bucket, downloadObj.Bucket)
	assert.Equal(t, int64(len(content)), downloadObj.Size)

	// Read content
	downloadedContent, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, content, downloadedContent)
}

func TestS3Storage_UploadWithPath(t *testing.T) {
	s3 := setupS3Storage(t)
	ctx := context.Background()
	bucket := generateUniqueBucketName("test-path")
	key := "path/to/nested/file.txt"

	err := s3.CreateBucket(ctx, bucket)
	require.NoError(t, err)
	defer cleanupS3Bucket(t, s3, bucket)

	content := []byte("nested file")
	obj, err := s3.Upload(ctx, bucket, key, bytes.NewReader(content), int64(len(content)), &UploadOptions{})
	require.NoError(t, err)
	assert.Equal(t, key, obj.Key)
}

func TestS3Storage_Delete(t *testing.T) {
	s3 := setupS3Storage(t)
	ctx := context.Background()
	bucket := generateUniqueBucketName("test-delete")
	key := "file-to-delete.txt"

	err := s3.CreateBucket(ctx, bucket)
	require.NoError(t, err)
	defer cleanupS3Bucket(t, s3, bucket)

	// Upload file
	content := []byte("test")
	_, err = s3.Upload(ctx, bucket, key, bytes.NewReader(content), int64(len(content)), &UploadOptions{})
	require.NoError(t, err)

	// Verify exists
	exists, err := s3.Exists(ctx, bucket, key)
	require.NoError(t, err)
	assert.True(t, exists)

	// Delete file
	err = s3.Delete(ctx, bucket, key)
	require.NoError(t, err)

	// Verify deleted
	exists, err = s3.Exists(ctx, bucket, key)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestS3Storage_DeleteNonExistent(t *testing.T) {
	s3 := setupS3Storage(t)
	ctx := context.Background()
	bucket := generateUniqueBucketName("test-delete-nonexist")

	err := s3.CreateBucket(ctx, bucket)
	require.NoError(t, err)
	defer cleanupS3Bucket(t, s3, bucket)

	// Deleting non-existent file should not error (S3 behavior)
	err = s3.Delete(ctx, bucket, "non-existent.txt")
	assert.NoError(t, err)
}

func TestS3Storage_Exists(t *testing.T) {
	s3 := setupS3Storage(t)
	ctx := context.Background()
	bucket := generateUniqueBucketName("test-exists")
	key := "existing-file.txt"

	err := s3.CreateBucket(ctx, bucket)
	require.NoError(t, err)
	defer cleanupS3Bucket(t, s3, bucket)

	// File doesn't exist yet
	exists, err := s3.Exists(ctx, bucket, key)
	require.NoError(t, err)
	assert.False(t, exists)

	// Upload file
	content := []byte("test")
	_, err = s3.Upload(ctx, bucket, key, bytes.NewReader(content), int64(len(content)), &UploadOptions{})
	require.NoError(t, err)

	// File now exists
	exists, err = s3.Exists(ctx, bucket, key)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestS3Storage_GetObject(t *testing.T) {
	s3 := setupS3Storage(t)
	ctx := context.Background()
	bucket := generateUniqueBucketName("test-getobj")
	key := "metadata-file.txt"

	err := s3.CreateBucket(ctx, bucket)
	require.NoError(t, err)
	defer cleanupS3Bucket(t, s3, bucket)

	// Upload with metadata
	content := []byte("test content")
	uploadOpts := &UploadOptions{
		ContentType: "text/plain",
		Metadata:    map[string]string{"author": "test-user", "version": "1"},
	}
	_, err = s3.Upload(ctx, bucket, key, bytes.NewReader(content), int64(len(content)), uploadOpts)
	require.NoError(t, err)

	// Get object metadata
	obj, err := s3.GetObject(ctx, bucket, key)
	require.NoError(t, err)
	assert.Equal(t, key, obj.Key)
	assert.Equal(t, bucket, obj.Bucket)
	assert.Equal(t, int64(len(content)), obj.Size)
	assert.Equal(t, "text/plain", obj.ContentType)
	assert.NotEmpty(t, obj.ETag)
	assert.NotNil(t, obj.Metadata)
}

func TestS3Storage_List(t *testing.T) {
	s3 := setupS3Storage(t)
	ctx := context.Background()
	bucket := generateUniqueBucketName("test-list")

	err := s3.CreateBucket(ctx, bucket)
	require.NoError(t, err)
	defer cleanupS3Bucket(t, s3, bucket)

	// Upload multiple files
	files := []string{"file1.txt", "file2.txt", "dir1/file3.txt", "dir1/file4.txt", "dir2/file5.txt"}
	for _, key := range files {
		content := []byte("test")
		_, err := s3.Upload(ctx, bucket, key, bytes.NewReader(content), int64(len(content)), &UploadOptions{})
		require.NoError(t, err)
	}

	// List all files
	result, err := s3.List(ctx, bucket, &ListOptions{})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(result.Objects), 5)
	assert.False(t, result.IsTruncated)
}

func TestS3Storage_ListWithPrefix(t *testing.T) {
	s3 := setupS3Storage(t)
	ctx := context.Background()
	bucket := generateUniqueBucketName("test-prefix")

	err := s3.CreateBucket(ctx, bucket)
	require.NoError(t, err)
	defer cleanupS3Bucket(t, s3, bucket)

	// Upload files with different prefixes
	files := []string{"images/photo1.jpg", "images/photo2.jpg", "documents/doc1.pdf"}
	for _, key := range files {
		content := []byte("test")
		_, err := s3.Upload(ctx, bucket, key, bytes.NewReader(content), int64(len(content)), &UploadOptions{})
		require.NoError(t, err)
	}

	// List with prefix
	result, err := s3.List(ctx, bucket, &ListOptions{Prefix: "images/"})
	require.NoError(t, err)
	assert.Len(t, result.Objects, 2)
	for _, obj := range result.Objects {
		assert.Contains(t, obj.Key, "images/")
	}
}

func TestS3Storage_ListWithLimit(t *testing.T) {
	s3 := setupS3Storage(t)
	ctx := context.Background()
	bucket := generateUniqueBucketName("test-limit")

	err := s3.CreateBucket(ctx, bucket)
	require.NoError(t, err)
	defer cleanupS3Bucket(t, s3, bucket)

	// Upload 10 files
	for i := 0; i < 10; i++ {
		content := []byte("test")
		key := "file/" + string(rune('0'+i)) + ".txt"
		_, err := s3.Upload(ctx, bucket, key, bytes.NewReader(content), int64(len(content)), &UploadOptions{})
		require.NoError(t, err)
	}

	// List with limit
	limit := 3
	result, err := s3.List(ctx, bucket, &ListOptions{MaxKeys: limit})
	require.NoError(t, err)
	assert.LessOrEqual(t, len(result.Objects), limit)
}

func TestS3Storage_CreateBucket(t *testing.T) {
	s3 := setupS3Storage(t)
	ctx := context.Background()
	bucket := generateUniqueBucketName("test-new")

	defer cleanupS3Bucket(t, s3, bucket)

	err := s3.CreateBucket(ctx, bucket)
	require.NoError(t, err)

	// Verify bucket exists
	exists, err := s3.BucketExists(ctx, bucket)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestS3Storage_CreateBucketAlreadyExists(t *testing.T) {
	s3 := setupS3Storage(t)
	ctx := context.Background()
	bucket := generateUniqueBucketName("test-existing")

	defer cleanupS3Bucket(t, s3, bucket)

	// Create bucket first time
	err := s3.CreateBucket(ctx, bucket)
	require.NoError(t, err)

	// Create bucket second time - should error (S3 behavior)
	err = s3.CreateBucket(ctx, bucket)
	assert.Error(t, err, "Creating duplicate bucket should return error")
	assert.Contains(t, err.Error(), "already exists", "Error should indicate bucket exists")
}

func TestS3Storage_DeleteBucket(t *testing.T) {
	s3 := setupS3Storage(t)
	ctx := context.Background()
	bucket := generateUniqueBucketName("test-delete")

	err := s3.CreateBucket(ctx, bucket)
	require.NoError(t, err)

	err = s3.DeleteBucket(ctx, bucket)
	require.NoError(t, err)

	// Verify deleted
	exists, err := s3.BucketExists(ctx, bucket)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestS3Storage_DeleteBucketNotEmpty(t *testing.T) {
	s3 := setupS3Storage(t)
	ctx := context.Background()
	bucket := generateUniqueBucketName("test-nonempty")

	err := s3.CreateBucket(ctx, bucket)
	require.NoError(t, err)
	defer cleanupS3Bucket(t, s3, bucket)

	// Upload a file
	content := []byte("test")
	_, err = s3.Upload(ctx, bucket, "file.txt", bytes.NewReader(content), int64(len(content)), &UploadOptions{})
	require.NoError(t, err)

	// Try to delete non-empty bucket
	err = s3.DeleteBucket(ctx, bucket)
	assert.Error(t, err) // S3 doesn't allow deleting non-empty buckets
}

func TestS3Storage_BucketExists(t *testing.T) {
	s3 := setupS3Storage(t)
	ctx := context.Background()
	bucket := generateUniqueBucketName("test-exists-check")

	// Bucket doesn't exist yet
	exists, err := s3.BucketExists(ctx, bucket)
	require.NoError(t, err)
	assert.False(t, exists)

	// Create bucket
	err = s3.CreateBucket(ctx, bucket)
	require.NoError(t, err)
	defer cleanupS3Bucket(t, s3, bucket)

	// Bucket now exists
	exists, err = s3.BucketExists(ctx, bucket)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestS3Storage_ListBuckets(t *testing.T) {
	s3 := setupS3Storage(t)
	ctx := context.Background()

	// Create test buckets
	testBuckets := []string{"test-list-bucket-1", "test-list-bucket-2", "test-list-bucket-3"}
	for _, bucket := range testBuckets {
		err := s3.CreateBucket(ctx, bucket)
		require.NoError(t, err)
		defer cleanupS3Bucket(t, s3, bucket)
	}

	// List buckets
	buckets, err := s3.ListBuckets(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(buckets), 3)

	// Verify our test buckets are in the list
	bucketSet := make(map[string]bool)
	for _, b := range buckets {
		bucketSet[b] = true
	}
	for _, tb := range testBuckets {
		assert.True(t, bucketSet[tb], "Bucket %s should be in list", tb)
	}
}

func TestS3Storage_CopyObject(t *testing.T) {
	s3 := setupS3Storage(t)
	ctx := context.Background()
	srcBucket := "test-src-bucket"
	destBucket := "test-dest-bucket"

	err := s3.CreateBucket(ctx, srcBucket)
	require.NoError(t, err)
	defer cleanupS3Bucket(t, s3, srcBucket)

	err = s3.CreateBucket(ctx, destBucket)
	require.NoError(t, err)
	defer cleanupS3Bucket(t, s3, destBucket)

	// Upload source file
	content := []byte("source content")
	srcKey := "source.txt"
	_, err = s3.Upload(ctx, srcBucket, srcKey, bytes.NewReader(content), int64(len(content)), &UploadOptions{})
	require.NoError(t, err)

	// Copy object
	destKey := "destination.txt"
	err = s3.CopyObject(ctx, srcBucket, srcKey, destBucket, destKey)
	require.NoError(t, err)

	// Verify destination exists
	exists, err := s3.Exists(ctx, destBucket, destKey)
	require.NoError(t, err)
	assert.True(t, exists)

	// Verify source still exists
	exists, err = s3.Exists(ctx, srcBucket, srcKey)
	require.NoError(t, err)
	assert.True(t, exists)

	// Verify content
	reader, _, err := s3.Download(ctx, destBucket, destKey, &DownloadOptions{})
	require.NoError(t, err)
	defer reader.Close()

	downloadedContent, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, content, downloadedContent)
}

func TestS3Storage_MoveObject(t *testing.T) {
	s3 := setupS3Storage(t)
	ctx := context.Background()
	srcBucket := "test-move-src-bucket"
	destBucket := "test-move-dest-bucket"

	err := s3.CreateBucket(ctx, srcBucket)
	require.NoError(t, err)
	defer cleanupS3Bucket(t, s3, srcBucket)

	err = s3.CreateBucket(ctx, destBucket)
	require.NoError(t, err)
	defer cleanupS3Bucket(t, s3, destBucket)

	// Upload source file
	content := []byte("move content")
	srcKey := "source.txt"
	_, err = s3.Upload(ctx, srcBucket, srcKey, bytes.NewReader(content), int64(len(content)), &UploadOptions{})
	require.NoError(t, err)

	// Move object
	destKey := "destination.txt"
	err = s3.MoveObject(ctx, srcBucket, srcKey, destBucket, destKey)
	require.NoError(t, err)

	// Verify destination exists
	exists, err := s3.Exists(ctx, destBucket, destKey)
	require.NoError(t, err)
	assert.True(t, exists)

	// Verify source no longer exists
	exists, err = s3.Exists(ctx, srcBucket, srcKey)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestS3Storage_GenerateSignedURL(t *testing.T) {
	s3 := setupS3Storage(t)
	ctx := context.Background()
	bucket := generateUniqueBucketName("test-signed-url")
	key := "test-file.txt"

	err := s3.CreateBucket(ctx, bucket)
	require.NoError(t, err)
	defer cleanupS3Bucket(t, s3, bucket)

	// Upload file
	content := []byte("test content")
	_, err = s3.Upload(ctx, bucket, key, bytes.NewReader(content), int64(len(content)), &UploadOptions{})
	require.NoError(t, err)

	// Generate signed URL for GET
	opts := &SignedURLOptions{
		Method:    "GET",
		ExpiresIn: 1 * time.Hour,
	}
	url, err := s3.GenerateSignedURL(ctx, bucket, key, opts)
	require.NoError(t, err)
	assert.NotEmpty(t, url)
	assert.Contains(t, url, bucket)
	assert.Contains(t, url, key)
}

func TestS3Storage_DownloadWithRange(t *testing.T) {
	s3 := setupS3Storage(t)
	ctx := context.Background()
	bucket := generateUniqueBucketName("test-range")
	key := "range-file.txt"

	err := s3.CreateBucket(ctx, bucket)
	require.NoError(t, err)
	defer cleanupS3Bucket(t, s3, bucket)

	// Upload file with known content
	content := []byte("0123456789")
	_, err = s3.Upload(ctx, bucket, key, bytes.NewReader(content), int64(len(content)), &UploadOptions{})
	require.NoError(t, err)

	// Download with range (bytes 2-5)
	opts := &DownloadOptions{
		Range: "bytes=2-5",
	}
	reader, _, err := s3.Download(ctx, bucket, key, opts)
	require.NoError(t, err)
	defer reader.Close()

	// Note: The size in obj might be the full size, not the range size
	// This depends on S3 implementation
	downloadedContent, err := io.ReadAll(reader)
	require.NoError(t, err)
	// Range "bytes=2-5" should return bytes at index 2, 3, 4, 5 (inclusive)
	assert.Equal(t, []byte("2345"), downloadedContent)
}
