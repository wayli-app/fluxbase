//nolint:errcheck // Test code - error handling not critical
package storage

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupLocalStorage(t *testing.T) (*LocalStorage, string) {
	// Create temporary directory
	tmpDir := t.TempDir()

	storage, err := NewLocalStorage(tmpDir)
	require.NoError(t, err)

	return storage, tmpDir
}

func TestNewLocalStorage(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := NewLocalStorage(tmpDir)

	assert.NoError(t, err)
	assert.NotNil(t, storage)
	assert.Equal(t, tmpDir, storage.basePath)

	// Verify directory was created
	_, err = os.Stat(tmpDir)
	assert.NoError(t, err)
}

func TestLocalStorage_Name(t *testing.T) {
	storage, _ := setupLocalStorage(t)

	assert.Equal(t, "local", storage.Name())
}

func TestLocalStorage_Health(t *testing.T) {
	storage, _ := setupLocalStorage(t)

	err := storage.Health(context.Background())

	assert.NoError(t, err)
}

func TestLocalStorage_UploadAndDownload(t *testing.T) {
	storage, _ := setupLocalStorage(t)
	ctx := context.Background()

	// Test data
	bucket := "test-bucket"
	key := "test-file.txt"
	content := "Hello, World!"

	// Upload
	opts := &UploadOptions{
		ContentType: "text/plain",
		Metadata: map[string]string{
			"author": "test-user",
		},
	}

	obj, err := storage.Upload(ctx, bucket, key, strings.NewReader(content), int64(len(content)), opts)

	require.NoError(t, err)
	assert.Equal(t, key, obj.Key)
	assert.Equal(t, bucket, obj.Bucket)
	assert.Equal(t, int64(len(content)), obj.Size)
	assert.Equal(t, "text/plain", obj.ContentType)
	assert.NotEmpty(t, obj.ETag)

	// Download
	reader, downloadedObj, err := storage.Download(ctx, bucket, key, nil)
	require.NoError(t, err)
	defer reader.Close()

	assert.Equal(t, key, downloadedObj.Key)
	assert.Equal(t, bucket, downloadedObj.Bucket)
	assert.Equal(t, int64(len(content)), downloadedObj.Size)
	assert.Equal(t, "text/plain", downloadedObj.ContentType)

	// Read content
	buf := make([]byte, len(content))
	n, err := reader.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, len(content), n)
	assert.Equal(t, content, string(buf))
}

func TestLocalStorage_UploadWithPath(t *testing.T) {
	storage, _ := setupLocalStorage(t)
	ctx := context.Background()

	bucket := "test-bucket"
	key := "path/to/nested/file.txt"
	content := "nested file"

	obj, err := storage.Upload(ctx, bucket, key, strings.NewReader(content), int64(len(content)), nil)

	require.NoError(t, err)
	assert.Equal(t, key, obj.Key)

	// Verify file exists at correct path
	reader, _, err := storage.Download(ctx, bucket, key, nil)
	require.NoError(t, err)
	reader.Close()
}

func TestLocalStorage_Delete(t *testing.T) {
	storage, _ := setupLocalStorage(t)
	ctx := context.Background()

	bucket := "test-bucket"
	key := "file-to-delete.txt"

	// Upload first
	_, err := storage.Upload(ctx, bucket, key, strings.NewReader("data"), 4, nil)
	require.NoError(t, err)

	// Verify exists
	exists, err := storage.Exists(ctx, bucket, key)
	require.NoError(t, err)
	assert.True(t, exists)

	// Delete
	err = storage.Delete(ctx, bucket, key)
	assert.NoError(t, err)

	// Verify deleted
	exists, err = storage.Exists(ctx, bucket, key)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestLocalStorage_DeleteNonExistent(t *testing.T) {
	storage, _ := setupLocalStorage(t)
	ctx := context.Background()

	err := storage.Delete(ctx, "bucket", "nonexistent.txt")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestLocalStorage_Exists(t *testing.T) {
	storage, _ := setupLocalStorage(t)
	ctx := context.Background()

	bucket := "test-bucket"
	key := "existing-file.txt"

	// Non-existent file
	exists, err := storage.Exists(ctx, bucket, key)
	require.NoError(t, err)
	assert.False(t, exists)

	// Upload file
	_, err = storage.Upload(ctx, bucket, key, strings.NewReader("data"), 4, nil)
	require.NoError(t, err)

	// Should exist now
	exists, err = storage.Exists(ctx, bucket, key)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestLocalStorage_GetObject(t *testing.T) {
	storage, _ := setupLocalStorage(t)
	ctx := context.Background()

	bucket := "test-bucket"
	key := "metadata-file.txt"
	content := "test content"

	// Upload with metadata
	opts := &UploadOptions{
		ContentType: "text/plain",
		Metadata: map[string]string{
			"version": "1.0",
			"author":  "test",
		},
	}

	_, err := storage.Upload(ctx, bucket, key, strings.NewReader(content), int64(len(content)), opts)
	require.NoError(t, err)

	// Get object metadata
	obj, err := storage.GetObject(ctx, bucket, key)

	require.NoError(t, err)
	assert.Equal(t, key, obj.Key)
	assert.Equal(t, bucket, obj.Bucket)
	assert.Equal(t, int64(len(content)), obj.Size)
	assert.Equal(t, "text/plain", obj.ContentType)
	assert.Equal(t, "1.0", obj.Metadata["version"])
	assert.Equal(t, "test", obj.Metadata["author"])
}

func TestLocalStorage_List(t *testing.T) {
	storage, _ := setupLocalStorage(t)
	ctx := context.Background()

	bucket := "test-bucket"

	// Upload multiple files
	files := []string{
		"file1.txt",
		"file2.txt",
		"dir1/file3.txt",
		"dir1/file4.txt",
		"dir2/file5.txt",
	}

	for _, file := range files {
		_, err := storage.Upload(ctx, bucket, file, strings.NewReader("data"), 4, nil)
		require.NoError(t, err)
	}

	// List all files
	result, err := storage.List(ctx, bucket, &ListOptions{})

	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(result.Objects), len(files))

	// Verify keys
	keys := make([]string, len(result.Objects))
	for i, obj := range result.Objects {
		keys[i] = obj.Key
	}

	for _, file := range files {
		assert.Contains(t, keys, filepath.ToSlash(file))
	}
}

func TestLocalStorage_ListWithPrefix(t *testing.T) {
	storage, _ := setupLocalStorage(t)
	ctx := context.Background()

	bucket := "test-bucket"

	// Upload files
	files := []string{
		"images/photo1.jpg",
		"images/photo2.jpg",
		"documents/doc1.pdf",
	}

	for _, file := range files {
		_, err := storage.Upload(ctx, bucket, file, strings.NewReader("data"), 4, nil)
		require.NoError(t, err)
	}

	// List with prefix
	result, err := storage.List(ctx, bucket, &ListOptions{
		Prefix: "images/",
	})

	require.NoError(t, err)
	assert.Equal(t, 2, len(result.Objects))

	for _, obj := range result.Objects {
		assert.True(t, strings.HasPrefix(obj.Key, "images/"))
	}
}

func TestLocalStorage_ListWithLimit(t *testing.T) {
	storage, _ := setupLocalStorage(t)
	ctx := context.Background()

	bucket := "test-bucket"

	// Upload multiple files
	for i := 0; i < 10; i++ {
		key := filepath.Join("file", string(rune('0'+i))+".txt")
		_, err := storage.Upload(ctx, bucket, key, strings.NewReader("data"), 4, nil)
		require.NoError(t, err)
	}

	// List with limit
	result, err := storage.List(ctx, bucket, &ListOptions{
		MaxKeys: 5,
	})

	require.NoError(t, err)
	assert.LessOrEqual(t, len(result.Objects), 5)
}

func TestLocalStorage_CreateBucket(t *testing.T) {
	storage, basePath := setupLocalStorage(t)
	ctx := context.Background()

	bucket := "new-bucket"

	err := storage.CreateBucket(ctx, bucket)

	require.NoError(t, err)

	// Verify bucket directory exists
	bucketPath := filepath.Join(basePath, bucket)
	info, err := os.Stat(bucketPath)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestLocalStorage_CreateBucketAlreadyExists(t *testing.T) {
	storage, _ := setupLocalStorage(t)
	ctx := context.Background()

	bucket := "existing-bucket"

	// Create once
	err := storage.CreateBucket(ctx, bucket)
	require.NoError(t, err)

	// Try to create again
	err = storage.CreateBucket(ctx, bucket)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestLocalStorage_DeleteBucket(t *testing.T) {
	storage, basePath := setupLocalStorage(t)
	ctx := context.Background()

	bucket := "bucket-to-delete"

	// Create bucket
	err := storage.CreateBucket(ctx, bucket)
	require.NoError(t, err)

	// Delete bucket
	err = storage.DeleteBucket(ctx, bucket)

	require.NoError(t, err)

	// Verify bucket is gone
	bucketPath := filepath.Join(basePath, bucket)
	_, err = os.Stat(bucketPath)
	assert.True(t, os.IsNotExist(err))
}

func TestLocalStorage_DeleteBucketNotEmpty(t *testing.T) {
	storage, _ := setupLocalStorage(t)
	ctx := context.Background()

	bucket := "non-empty-bucket"

	// Create bucket and add file
	err := storage.CreateBucket(ctx, bucket)
	require.NoError(t, err)

	_, err = storage.Upload(ctx, bucket, "file.txt", strings.NewReader("data"), 4, nil)
	require.NoError(t, err)

	// Try to delete non-empty bucket
	err = storage.DeleteBucket(ctx, bucket)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not empty")
}

func TestLocalStorage_BucketExists(t *testing.T) {
	storage, _ := setupLocalStorage(t)
	ctx := context.Background()

	bucket := "test-bucket"

	// Should not exist initially
	exists, err := storage.BucketExists(ctx, bucket)
	require.NoError(t, err)
	assert.False(t, exists)

	// Create bucket
	err = storage.CreateBucket(ctx, bucket)
	require.NoError(t, err)

	// Should exist now
	exists, err = storage.BucketExists(ctx, bucket)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestLocalStorage_ListBuckets(t *testing.T) {
	storage, _ := setupLocalStorage(t)
	ctx := context.Background()

	buckets := []string{"bucket1", "bucket2", "bucket3"}

	// Create buckets
	for _, bucket := range buckets {
		err := storage.CreateBucket(ctx, bucket)
		require.NoError(t, err)
	}

	// List buckets
	result, err := storage.ListBuckets(ctx)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(result), len(buckets))

	for _, bucket := range buckets {
		assert.Contains(t, result, bucket)
	}
}

func TestLocalStorage_CopyObject(t *testing.T) {
	storage, _ := setupLocalStorage(t)
	ctx := context.Background()

	srcBucket := "src-bucket"
	srcKey := "source.txt"
	destBucket := "dest-bucket"
	destKey := "destination.txt"
	content := "copy me"

	// Create buckets
	_ = storage.CreateBucket(ctx, srcBucket)
	_ = storage.CreateBucket(ctx, destBucket)

	// Upload source file
	_, err := storage.Upload(ctx, srcBucket, srcKey, strings.NewReader(content), int64(len(content)), nil)
	require.NoError(t, err)

	// Copy
	err = storage.CopyObject(ctx, srcBucket, srcKey, destBucket, destKey)

	require.NoError(t, err)

	// Verify both exist
	srcExists, _ := storage.Exists(ctx, srcBucket, srcKey)
	destExists, _ := storage.Exists(ctx, destBucket, destKey)

	assert.True(t, srcExists)
	assert.True(t, destExists)

	// Verify content is same
	reader, _, err := storage.Download(ctx, destBucket, destKey, nil)
	require.NoError(t, err)
	defer reader.Close()

	buf := make([]byte, len(content))
	_, _ = reader.Read(buf)
	assert.Equal(t, content, string(buf))
}

func TestLocalStorage_MoveObject(t *testing.T) {
	storage, _ := setupLocalStorage(t)
	ctx := context.Background()

	srcBucket := "src-bucket"
	srcKey := "source.txt"
	destBucket := "dest-bucket"
	destKey := "destination.txt"
	content := "move me"

	// Create buckets
	_ = storage.CreateBucket(ctx, srcBucket)
	_ = storage.CreateBucket(ctx, destBucket)

	// Upload source file
	_, err := storage.Upload(ctx, srcBucket, srcKey, strings.NewReader(content), int64(len(content)), nil)
	require.NoError(t, err)

	// Move
	err = storage.MoveObject(ctx, srcBucket, srcKey, destBucket, destKey)

	require.NoError(t, err)

	// Verify source is gone and dest exists
	srcExists, _ := storage.Exists(ctx, srcBucket, srcKey)
	destExists, _ := storage.Exists(ctx, destBucket, destKey)

	assert.False(t, srcExists)
	assert.True(t, destExists)
}

func TestLocalStorage_GenerateSignedURL(t *testing.T) {
	storage, _ := setupLocalStorage(t)
	ctx := context.Background()

	// Signed URLs not supported for local storage
	_, err := storage.GenerateSignedURL(ctx, "bucket", "key", nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}
