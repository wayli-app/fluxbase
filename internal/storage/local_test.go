//nolint:errcheck // Test code - error handling not critical
package storage

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupLocalStorage(t *testing.T) (*LocalStorage, string) {
	// Create temporary directory
	tmpDir := t.TempDir()

	storage, err := NewLocalStorage(tmpDir, "http://localhost:8080", "test-signing-secret")
	require.NoError(t, err)

	return storage, tmpDir
}

func TestNewLocalStorage(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := NewLocalStorage(tmpDir, "http://localhost:8080", "test-signing-secret")

	assert.NoError(t, err)
	assert.NotNil(t, storage)
	assert.Equal(t, tmpDir, storage.basePath)
	assert.Equal(t, "http://localhost:8080", storage.baseURL)
	assert.Equal(t, "test-signing-secret", storage.signingSecret)

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

	// Generate a signed URL
	url, err := storage.GenerateSignedURL(ctx, "test-bucket", "test-file.txt", nil)

	assert.NoError(t, err)
	assert.Contains(t, url, "http://localhost:8080/api/v1/storage/object?token=")

	// Test with custom options
	opts := &SignedURLOptions{
		ExpiresIn: 3600 * 1000000000, // 1 hour in nanoseconds
		Method:    "GET",
	}
	url2, err := storage.GenerateSignedURL(ctx, "test-bucket", "test-file.txt", opts)

	assert.NoError(t, err)
	assert.Contains(t, url2, "http://localhost:8080/api/v1/storage/object?token=")
}

func TestLocalStorage_ValidateSignedToken(t *testing.T) {
	storage, _ := setupLocalStorage(t)
	ctx := context.Background()

	// Generate a signed URL
	url, err := storage.GenerateSignedURL(ctx, "test-bucket", "path/to/file.txt", nil)
	require.NoError(t, err)

	// Extract token from URL
	parts := strings.Split(url, "token=")
	require.Len(t, parts, 2)
	token := parts[1]

	// Validate the token
	bucket, key, method, err := storage.ValidateSignedToken(token)
	assert.NoError(t, err)
	assert.Equal(t, "test-bucket", bucket)
	assert.Equal(t, "path/to/file.txt", key)
	assert.Equal(t, "GET", method)
}

func TestLocalStorage_ValidateSignedToken_Invalid(t *testing.T) {
	storage, _ := setupLocalStorage(t)

	// Test invalid token
	_, _, _, err := storage.ValidateSignedToken("invalid-token")
	assert.Error(t, err)

	// Test tampered token
	_, _, _, err = storage.ValidateSignedToken("dGFtcGVyZWQ=")
	assert.Error(t, err)
}

func TestValidatePathComponent(t *testing.T) {
	tests := []struct {
		name      string
		component string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid simple name",
			component: "myfile.txt",
			wantErr:   false,
		},
		{
			name:      "valid name with dashes and underscores",
			component: "my-file_name.txt",
			wantErr:   false,
		},
		{
			name:      "valid name with numbers",
			component: "file123",
			wantErr:   false,
		},
		{
			name:      "valid my-category",
			component: "my-category",
			wantErr:   false,
		},
		{
			name:      "valid category_123",
			component: "category_123",
			wantErr:   false,
		},
		{
			name:      "valid logs",
			component: "logs",
			wantErr:   false,
		},
		{
			name:      "valid single char",
			component: "a",
			wantErr:   false,
		},
		{
			name:      "valid test-category-name",
			component: "test-category-name",
			wantErr:   false,
		},
		{
			name:      "empty component",
			component: "",
			wantErr:   true,
			errMsg:    "empty path component",
		},
		{
			name:      "path traversal with double dots",
			component: "..",
			wantErr:   true,
			errMsg:    "path traversal detected",
		},
		{
			name:      "path traversal ../etc",
			component: "../etc",
			wantErr:   true,
			errMsg:    "path traversal detected",
		},
		{
			name:      "path traversal sub/../../etc",
			component: "sub/../../etc",
			wantErr:   true,
			errMsg:    "path traversal detected",
		},
		{
			name:      "path traversal embedded",
			component: "foo/../bar",
			wantErr:   true,
			errMsg:    "path traversal detected",
		},
		{
			name:      "null byte injection in middle",
			component: "cat\x00egory",
			wantErr:   true,
			errMsg:    "null bytes not allowed",
		},
		{
			name:      "null byte injection",
			component: "file\x00.txt",
			wantErr:   true,
			errMsg:    "null bytes not allowed",
		},
		{
			name:      "absolute path /etc/passwd",
			component: "/etc/passwd",
			wantErr:   true,
			errMsg:    "absolute paths not allowed",
		},
		{
			name:      "absolute path /var/logs",
			component: "/var/logs",
			wantErr:   true,
			errMsg:    "absolute paths not allowed",
		},
		{
			name:      "absolute path with backslash",
			component: "\\etc\\passwd",
			wantErr:   true,
			errMsg:    "absolute paths not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePathComponent(tt.component)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLocalStorage_GetPath_PathTraversal(t *testing.T) {
	storage, _ := setupLocalStorage(t)

	tests := []struct {
		name    string
		bucket  string
		key     string
		wantErr bool
	}{
		{
			name:    "valid path",
			bucket:  "mybucket",
			key:     "path/to/file.txt",
			wantErr: false,
		},
		{
			name:    "bucket with path traversal",
			bucket:  "../etc",
			key:     "passwd",
			wantErr: true,
		},
		{
			name:    "key with path traversal",
			bucket:  "mybucket",
			key:     "../../../etc/passwd",
			wantErr: true,
		},
		{
			name:    "key with embedded traversal",
			bucket:  "mybucket",
			key:     "foo/../bar",
			wantErr: true,
		},
		{
			name:    "empty bucket",
			bucket:  "",
			key:     "file.txt",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := storage.getPath(tt.bucket, tt.key)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLocalStorage_ChunkedUpload(t *testing.T) {
	storage, _ := setupLocalStorage(t)
	ctx := context.Background()

	bucket := "test-bucket"
	key := "large-file.dat"
	chunkSize := int64(1024) // 1KB chunks
	totalSize := int64(3000) // 3KB total (3 chunks, last one partial)

	// Create bucket
	err := storage.CreateBucket(ctx, bucket)
	require.NoError(t, err)

	// Initialize chunked upload
	opts := &UploadOptions{
		ContentType: "application/octet-stream",
		Metadata: map[string]string{
			"source": "test",
		},
	}

	session, err := storage.InitChunkedUpload(ctx, bucket, key, totalSize, chunkSize, opts)

	require.NoError(t, err)
	assert.NotEmpty(t, session.UploadID)
	assert.Equal(t, bucket, session.Bucket)
	assert.Equal(t, key, session.Key)
	assert.Equal(t, totalSize, session.TotalSize)
	assert.Equal(t, chunkSize, session.ChunkSize)
	assert.Equal(t, 3, session.TotalChunks) // ceil(3000/1024) = 3
	assert.Equal(t, "active", session.Status)
	assert.Equal(t, "application/octet-stream", session.ContentType)
	assert.Empty(t, session.CompletedChunks)

	// Upload chunks
	for i := 0; i < session.TotalChunks; i++ {
		var chunkData []byte
		if i == session.TotalChunks-1 {
			// Last chunk is partial
			chunkData = make([]byte, totalSize-int64(i)*chunkSize)
		} else {
			chunkData = make([]byte, chunkSize)
		}
		// Fill with pattern
		for j := range chunkData {
			chunkData[j] = byte(i + 'A')
		}

		result, err := storage.UploadChunk(ctx, session, i, strings.NewReader(string(chunkData)), int64(len(chunkData)))
		require.NoError(t, err)
		assert.Equal(t, i, result.ChunkIndex)
		assert.NotEmpty(t, result.ETag)
		assert.Equal(t, int64(len(chunkData)), result.Size)
	}

	// Complete the upload
	obj, err := storage.CompleteChunkedUpload(ctx, session)

	require.NoError(t, err)
	assert.Equal(t, key, obj.Key)
	assert.Equal(t, bucket, obj.Bucket)
	assert.Equal(t, totalSize, obj.Size)
	assert.NotEmpty(t, obj.ETag)

	// Verify file exists
	exists, _ := storage.Exists(ctx, bucket, key)
	assert.True(t, exists)

	// Verify file size
	fileObj, err := storage.GetObject(ctx, bucket, key)
	require.NoError(t, err)
	assert.Equal(t, totalSize, fileObj.Size)
}

func TestLocalStorage_ChunkedUpload_InvalidChunkIndex(t *testing.T) {
	storage, _ := setupLocalStorage(t)
	ctx := context.Background()

	bucket := "test-bucket"
	key := "file.dat"

	err := storage.CreateBucket(ctx, bucket)
	require.NoError(t, err)

	session, err := storage.InitChunkedUpload(ctx, bucket, key, 1024, 512, nil)
	require.NoError(t, err)

	// Try invalid chunk index (negative)
	_, err = storage.UploadChunk(ctx, session, -1, strings.NewReader("data"), 4)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid chunk index")

	// Try invalid chunk index (too high)
	_, err = storage.UploadChunk(ctx, session, 10, strings.NewReader("data"), 4)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid chunk index")
}

func TestLocalStorage_ChunkedUpload_NilSession(t *testing.T) {
	storage, _ := setupLocalStorage(t)
	ctx := context.Background()

	// UploadChunk with nil session
	_, err := storage.UploadChunk(ctx, nil, 0, strings.NewReader("data"), 4)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session is nil")

	// CompleteChunkedUpload with nil session
	_, err = storage.CompleteChunkedUpload(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session is nil")

	// AbortChunkedUpload with nil session
	err = storage.AbortChunkedUpload(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session is nil")
}

func TestLocalStorage_AbortChunkedUpload(t *testing.T) {
	storage, basePath := setupLocalStorage(t)
	ctx := context.Background()

	bucket := "test-bucket"
	key := "file.dat"

	err := storage.CreateBucket(ctx, bucket)
	require.NoError(t, err)

	session, err := storage.InitChunkedUpload(ctx, bucket, key, 1024, 512, nil)
	require.NoError(t, err)

	// Verify chunk directory exists
	chunkDir := filepath.Join(basePath, ".chunked", session.UploadID)
	_, err = os.Stat(chunkDir)
	assert.NoError(t, err)

	// Upload a chunk
	_, err = storage.UploadChunk(ctx, session, 0, strings.NewReader(strings.Repeat("a", 512)), 512)
	require.NoError(t, err)

	// Abort the upload
	err = storage.AbortChunkedUpload(ctx, session)
	require.NoError(t, err)

	// Verify chunk directory is removed
	_, err = os.Stat(chunkDir)
	assert.True(t, os.IsNotExist(err))
}

func TestLocalStorage_GetChunkedUploadSession(t *testing.T) {
	storage, _ := setupLocalStorage(t)
	ctx := context.Background()

	bucket := "test-bucket"
	key := "file.dat"

	err := storage.CreateBucket(ctx, bucket)
	require.NoError(t, err)

	// Initialize session
	opts := &UploadOptions{
		ContentType: "text/plain",
	}
	session, err := storage.InitChunkedUpload(ctx, bucket, key, 2048, 1024, opts)
	require.NoError(t, err)

	// Upload one chunk
	_, err = storage.UploadChunk(ctx, session, 0, strings.NewReader(strings.Repeat("a", 1024)), 1024)
	require.NoError(t, err)

	// Get session
	retrieved, err := storage.GetChunkedUploadSession(session.UploadID)

	require.NoError(t, err)
	assert.Equal(t, session.UploadID, retrieved.UploadID)
	assert.Equal(t, bucket, retrieved.Bucket)
	assert.Equal(t, key, retrieved.Key)
	assert.Equal(t, int64(2048), retrieved.TotalSize)
	assert.Equal(t, int64(1024), retrieved.ChunkSize)
	assert.Equal(t, 2, retrieved.TotalChunks)
	// CompletedChunks is updated based on actual chunk files
	assert.Contains(t, retrieved.CompletedChunks, 0)
}

func TestLocalStorage_GetChunkedUploadSession_NotFound(t *testing.T) {
	storage, _ := setupLocalStorage(t)

	_, err := storage.GetChunkedUploadSession("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestLocalStorage_CompleteChunkedUpload_MissingChunk(t *testing.T) {
	storage, _ := setupLocalStorage(t)
	ctx := context.Background()

	bucket := "test-bucket"
	key := "file.dat"

	err := storage.CreateBucket(ctx, bucket)
	require.NoError(t, err)

	// Initialize 3-chunk upload
	session, err := storage.InitChunkedUpload(ctx, bucket, key, 3072, 1024, nil)
	require.NoError(t, err)

	// Only upload chunk 0 and chunk 2 (missing chunk 1)
	_, err = storage.UploadChunk(ctx, session, 0, strings.NewReader(strings.Repeat("a", 1024)), 1024)
	require.NoError(t, err)
	_, err = storage.UploadChunk(ctx, session, 2, strings.NewReader(strings.Repeat("c", 1024)), 1024)
	require.NoError(t, err)

	// Try to complete - should fail
	_, err = storage.CompleteChunkedUpload(ctx, session)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing chunk 1")
}

func TestLocalStorage_ValidateSignedTokenFull(t *testing.T) {
	storage, _ := setupLocalStorage(t)
	ctx := context.Background()

	// Generate signed URL with transform options
	opts := &SignedURLOptions{
		Method:           "GET",
		TransformWidth:   800,
		TransformHeight:  600,
		TransformFormat:  "webp",
		TransformQuality: 85,
		TransformFit:     "cover",
	}

	url, err := storage.GenerateSignedURL(ctx, "mybucket", "images/photo.jpg", opts)
	require.NoError(t, err)

	// Extract token
	parts := strings.Split(url, "token=")
	require.Len(t, parts, 2)
	token := parts[1]

	// Validate with full result
	result, err := storage.ValidateSignedTokenFull(token)

	require.NoError(t, err)
	assert.Equal(t, "mybucket", result.Bucket)
	assert.Equal(t, "images/photo.jpg", result.Key)
	assert.Equal(t, "GET", result.Method)
	assert.Equal(t, 800, result.TransformWidth)
	assert.Equal(t, 600, result.TransformHeight)
	assert.Equal(t, "webp", result.TransformFormat)
	assert.Equal(t, 85, result.TransformQuality)
	assert.Equal(t, "cover", result.TransformFit)
}

func TestLocalStorage_ValidateSignedTokenFull_Invalid(t *testing.T) {
	storage, _ := setupLocalStorage(t)

	// Invalid token encoding
	_, err := storage.ValidateSignedTokenFull("not-valid-base64!")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token encoding")

	// Token too short
	_, err = storage.ValidateSignedTokenFull("YWJj") // "abc" in base64
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token length")
}

func TestLocalStorage_Download_RangeRequest(t *testing.T) {
	storage, _ := setupLocalStorage(t)
	ctx := context.Background()

	bucket := "test-bucket"
	key := "rangetest.txt"
	content := "0123456789ABCDEFGHIJ" // 20 bytes

	// Upload file
	_, err := storage.Upload(ctx, bucket, key, strings.NewReader(content), int64(len(content)), nil)
	require.NoError(t, err)

	// Download with range
	downloadOpts := &DownloadOptions{
		Range: "bytes=5-14",
	}

	reader, obj, err := storage.Download(ctx, bucket, key, downloadOpts)
	require.NoError(t, err)
	defer reader.Close()

	assert.Equal(t, int64(10), obj.Size) // 14-5+1 = 10 bytes

	buf := make([]byte, 10)
	n, _ := reader.Read(buf)
	assert.Equal(t, 10, n)
	assert.Equal(t, "56789ABCDE", string(buf))
}

func TestLocalStorage_Download_InvalidRange(t *testing.T) {
	storage, _ := setupLocalStorage(t)
	ctx := context.Background()

	bucket := "test-bucket"
	key := "file.txt"
	content := "short"

	// Upload file
	_, err := storage.Upload(ctx, bucket, key, strings.NewReader(content), int64(len(content)), nil)
	require.NoError(t, err)

	// Download with invalid range (start > size)
	downloadOpts := &DownloadOptions{
		Range: "bytes=100-200",
	}

	_, _, err = storage.Download(ctx, bucket, key, downloadOpts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not satisfiable")
}

func TestLocalStorage_UpdateChunkedUploadSession(t *testing.T) {
	storage, _ := setupLocalStorage(t)
	ctx := context.Background()

	bucket := "test-bucket"
	key := "file.dat"

	err := storage.CreateBucket(ctx, bucket)
	require.NoError(t, err)

	session, err := storage.InitChunkedUpload(ctx, bucket, key, 2048, 1024, nil)
	require.NoError(t, err)

	// Modify session
	session.Status = "paused"
	session.CompletedChunks = []int{0}

	// Update session
	err = storage.UpdateChunkedUploadSession(session)
	require.NoError(t, err)

	// Retrieve and verify
	retrieved, err := storage.GetChunkedUploadSession(session.UploadID)
	require.NoError(t, err)
	assert.Equal(t, "paused", retrieved.Status)
}

// TestLocalStorage_ChunkedUpload_OwnerIDDurability covers the root cause of the
// chunked-upload RLS bug: InitChunkedUpload writes session.json before the
// caller assigns OwnerID on the returned struct. A re-persist via
// UpdateChunkedUploadSession must make the owner survive a reload, otherwise
// storeUploadedObject inserts storage.objects with owner_id = NULL and fails
// the storage_objects_insert RLS policy for authenticated users.
func TestLocalStorage_ChunkedUpload_OwnerIDDurability(t *testing.T) {
	storage, _ := setupLocalStorage(t)
	ctx := context.Background()

	err := storage.CreateBucket(ctx, "temp-files")
	require.NoError(t, err)

	// Simulate the init handler: InitChunkedUpload returns a session whose
	// OwnerID is empty (it's not a parameter and never set inside the provider).
	session, err := storage.InitChunkedUpload(ctx, "temp-files", "user-123/timestamp-file.geojson", 2048, 1024, nil)
	require.NoError(t, err)
	require.Empty(t, session.OwnerID, "InitChunkedUpload must not set OwnerID")

	// Reloading right after init reproduces the bug: owner is lost.
	reloadedBeforeUpdate, err := storage.GetChunkedUploadSession(session.UploadID)
	require.NoError(t, err)
	assert.Empty(t, reloadedBeforeUpdate.OwnerID, "owner is empty until re-persisted")

	// The init handler assigns the owner after init and must re-persist it.
	session.OwnerID = "user-123"
	err = storage.UpdateChunkedUploadSession(session)
	require.NoError(t, err)

	// After re-persist, the owner survives a reload — fixing the RLS failure.
	reloadedAfterUpdate, err := storage.GetChunkedUploadSession(session.UploadID)
	require.NoError(t, err)
	assert.Equal(t, "user-123", reloadedAfterUpdate.OwnerID,
		"OwnerID must survive a session reload after UpdateChunkedUploadSession")
}

func TestLocalStorage_GenerateSignedURL_NoSecret(t *testing.T) {
	tmpDir := t.TempDir()

	// Create storage without signing secret
	storage, err := NewLocalStorage(tmpDir, "http://localhost:8080", "")
	require.NoError(t, err)

	ctx := context.Background()
	_, err = storage.GenerateSignedURL(ctx, "bucket", "key", nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "signing secret not configured")
}

func TestLocalStorage_GenerateSignedURL_NoBaseURL(t *testing.T) {
	tmpDir := t.TempDir()

	// Create storage without base URL
	storage, err := NewLocalStorage(tmpDir, "", "secret")
	require.NoError(t, err)

	ctx := context.Background()
	_, err = storage.GenerateSignedURL(ctx, "bucket", "key", nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "base URL not configured")
}

func TestLocalStorage_ValidateSignedToken_NoSecret(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := NewLocalStorage(tmpDir, "http://localhost", "")
	require.NoError(t, err)

	_, _, _, err = storage.ValidateSignedToken("sometoken")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "signing secret not configured")
}

func TestLocalStorage_CleanupExpiredChunkedUploads(t *testing.T) {
	storage, basePath := setupLocalStorage(t)
	ctx := context.Background()

	// Create a bucket
	err := storage.CreateBucket(ctx, "test-bucket")
	require.NoError(t, err)

	// Create an expired upload session manually
	expiredID := "expired-upload-id"
	chunkDir := filepath.Join(basePath, ".chunked", expiredID)
	err = os.MkdirAll(chunkDir, 0o755)
	require.NoError(t, err)

	// Create session file with past expiry
	session := ChunkedUploadSession{
		UploadID:    expiredID,
		Bucket:      "test-bucket",
		Key:         "expired-file.dat",
		TotalSize:   1024,
		ChunkSize:   512,
		TotalChunks: 2,
		Status:      "active",
		ExpiresAt:   time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
	}

	sessionData, _ := json.Marshal(session)
	sessionPath := filepath.Join(chunkDir, "session.json")
	err = os.WriteFile(sessionPath, sessionData, 0o644)
	require.NoError(t, err)

	// Run cleanup
	cleaned, err := storage.CleanupExpiredChunkedUploads(ctx)

	require.NoError(t, err)
	assert.Equal(t, 1, cleaned)

	// Verify directory was removed
	_, err = os.Stat(chunkDir)
	assert.True(t, os.IsNotExist(err))
}

func TestLocalStorage_CleanupExpiredChunkedUploads_ActiveSession(t *testing.T) {
	storage, basePath := setupLocalStorage(t)
	ctx := context.Background()

	err := storage.CreateBucket(ctx, "test-bucket")
	require.NoError(t, err)

	// Create a non-expired session
	activeID := "active-upload-id"
	chunkDir := filepath.Join(basePath, ".chunked", activeID)
	err = os.MkdirAll(chunkDir, 0o755)
	require.NoError(t, err)

	session := ChunkedUploadSession{
		UploadID:    activeID,
		Bucket:      "test-bucket",
		Key:         "active-file.dat",
		TotalSize:   1024,
		ChunkSize:   512,
		TotalChunks: 2,
		Status:      "active",
		ExpiresAt:   time.Now().Add(24 * time.Hour), // Expires in 24 hours
	}

	sessionData, _ := json.Marshal(session)
	sessionPath := filepath.Join(chunkDir, "session.json")
	err = os.WriteFile(sessionPath, sessionData, 0o644)
	require.NoError(t, err)

	// Run cleanup
	cleaned, err := storage.CleanupExpiredChunkedUploads(ctx)

	require.NoError(t, err)
	assert.Equal(t, 0, cleaned)

	// Verify directory still exists
	_, err = os.Stat(chunkDir)
	assert.NoError(t, err)
}

func TestLocalStorage_CleanupExpiredChunkedUploads_NoChunkedDir(t *testing.T) {
	storage, _ := setupLocalStorage(t)
	ctx := context.Background()

	// No .chunked directory exists, should return 0 without error
	cleaned, err := storage.CleanupExpiredChunkedUploads(ctx)

	assert.NoError(t, err)
	assert.Equal(t, 0, cleaned)
}
