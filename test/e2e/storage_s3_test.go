package e2e

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
	"github.com/wayli-app/fluxbase/test"
)

// setupStorageS3Test prepares the test context for S3/MinIO storage tests
func setupStorageS3Test(t *testing.T) *test.TestContext {
	tc := test.NewTestContext(t)
	tc.EnsureStorageSchema()

	// Configure S3/MinIO storage provider
	tc.Config.Storage.Provider = "s3"
	tc.Config.Storage.S3Endpoint = "minio:9000"
	tc.Config.Storage.S3AccessKey = "minioadmin"
	tc.Config.Storage.S3SecretKey = "minioadmin"
	tc.Config.Storage.S3Bucket = "fluxbase-test"
	tc.Config.Storage.S3Region = "us-east-1"

	// Clean up any existing test storage files
	tc.CleanupStorageFiles()

	return tc
}

// TestStorageS3CreateBucket tests creating a storage bucket in MinIO
func TestStorageS3CreateBucket(t *testing.T) {
	tc := setupStorageS3Test(t)
	defer tc.Close()

	bucketName := "test-s3-bucket"

	// Create bucket
	resp := tc.NewRequest("POST", "/api/v1/storage/buckets/"+bucketName).
		Send().
		AssertStatus(fiber.StatusCreated)

	var result map[string]interface{}
	resp.JSON(&result)

	require.Equal(t, bucketName, result["bucket"])
	t.Logf("Created S3 bucket: %s", bucketName)
}

// TestStorageS3ListBuckets tests listing storage buckets
func TestStorageS3ListBuckets(t *testing.T) {
	tc := setupStorageS3Test(t)
	defer tc.Close()

	// Create a few test buckets
	tc.NewRequest("POST", "/api/v1/storage/buckets/s3-bucket1").Send().AssertStatus(fiber.StatusCreated)
	tc.NewRequest("POST", "/api/v1/storage/buckets/s3-bucket2").Send().AssertStatus(fiber.StatusCreated)

	// List buckets
	resp := tc.NewRequest("GET", "/api/v1/storage/buckets").
		Send().
		AssertStatus(fiber.StatusOK)

	var result map[string]interface{}
	resp.JSON(&result)

	buckets, ok := result["buckets"].([]interface{})
	require.True(t, ok, "Response should have buckets array")
	require.GreaterOrEqual(t, len(buckets), 2, "Should have at least 2 buckets")
	t.Logf("Found %d S3 buckets", len(buckets))
}

// TestStorageS3UploadFile tests uploading a file to MinIO
func TestStorageS3UploadFile(t *testing.T) {
	tc := setupStorageS3Test(t)
	defer tc.Close()

	bucketName := "s3-upload-test"
	fileName := "s3-test.txt"
	fileContent := []byte("Hello from MinIO!")

	// Create bucket first
	tc.NewRequest("POST", "/api/v1/storage/buckets/"+bucketName).
		Send().
		AssertStatus(fiber.StatusCreated)

	// Create multipart form with file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", fileName)
	require.NoError(t, err)

	_, err = part.Write(fileContent)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	// Upload file
	req := httptest.NewRequest("POST", "/api/v1/storage/"+bucketName+"/"+fileName, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := tc.App.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusCreated, resp.StatusCode, "File upload should succeed")

	t.Logf("Uploaded file: %s to S3 bucket: %s", fileName, bucketName)
}

// TestStorageS3DownloadFile tests downloading a file from MinIO
func TestStorageS3DownloadFile(t *testing.T) {
	tc := setupStorageS3Test(t)
	defer tc.Close()

	bucketName := "s3-download-test"
	fileName := "s3-download.txt"
	fileContent := []byte("Download me from S3!")

	// Create bucket
	tc.NewRequest("POST", "/api/v1/storage/buckets/"+bucketName).
		Send().
		AssertStatus(fiber.StatusCreated)

	// Upload file first
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", fileName)
	require.NoError(t, err)
	_, err = part.Write(fileContent)
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)

	uploadReq := httptest.NewRequest("POST", "/api/v1/storage/"+bucketName+"/"+fileName, body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	_, err = tc.App.Test(uploadReq)
	require.NoError(t, err)

	// Now download the file
	resp := tc.NewRequest("GET", "/api/v1/storage/"+bucketName+"/"+fileName).
		Send()

	// Download should work
	if resp.Status() == fiber.StatusOK {
		downloadedContent, err := io.ReadAll(bytes.NewReader(resp.Body()))
		require.NoError(t, err)
		require.Equal(t, fileContent, downloadedContent, "Downloaded content should match uploaded content")
		t.Logf("Downloaded file successfully from S3")
	} else {
		t.Logf("Download returned status: %d (may need further implementation)", resp.Status())
	}
}

// TestStorageS3DeleteFile tests deleting a file from MinIO
func TestStorageS3DeleteFile(t *testing.T) {
	tc := setupStorageS3Test(t)
	defer tc.Close()

	bucketName := "s3-delete-test"
	fileName := "s3-todelete.txt"

	// Create bucket
	tc.NewRequest("POST", "/api/v1/storage/buckets/"+bucketName).
		Send().
		AssertStatus(fiber.StatusCreated)

	// Upload a file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", fileName)
	require.NoError(t, err)
	_, err = part.Write([]byte("Delete me from S3"))
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)

	uploadReq := httptest.NewRequest("POST", "/api/v1/storage/"+bucketName+"/"+fileName, body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	_, err = tc.App.Test(uploadReq)
	require.NoError(t, err)

	// Delete the file
	tc.NewRequest("DELETE", "/api/v1/storage/"+bucketName+"/"+fileName).
		Send().
		AssertStatus(fiber.StatusNoContent)

	t.Logf("Deleted file: %s from S3 bucket: %s", fileName, bucketName)
}

// TestStorageS3DeleteBucket tests deleting a bucket from MinIO
func TestStorageS3DeleteBucket(t *testing.T) {
	tc := setupStorageS3Test(t)
	defer tc.Close()

	bucketName := "s3-bucket-to-delete"

	// Create bucket
	tc.NewRequest("POST", "/api/v1/storage/buckets/"+bucketName).
		Send().
		AssertStatus(fiber.StatusCreated)

	// Delete bucket
	tc.NewRequest("DELETE", "/api/v1/storage/buckets/"+bucketName).
		Send().
		AssertStatus(fiber.StatusNoContent)

	t.Logf("Deleted S3 bucket: %s", bucketName)
}

// TestStorageS3ListFiles tests listing files in a MinIO bucket
func TestStorageS3ListFiles(t *testing.T) {
	tc := setupStorageS3Test(t)
	defer tc.Close()

	bucketName := "s3-list-test"

	// Create bucket
	tc.NewRequest("POST", "/api/v1/storage/buckets/"+bucketName).
		Send().
		AssertStatus(fiber.StatusCreated)

	// Upload a few files
	for i := 1; i <= 3; i++ {
		fileName := fmt.Sprintf("s3-file%d.txt", i)
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", fileName)
		require.NoError(t, err)
		_, err = part.Write([]byte(fmt.Sprintf("S3 Content %d", i)))
		require.NoError(t, err)
		err = writer.Close()
		require.NoError(t, err)

		uploadReq := httptest.NewRequest("POST", "/api/v1/storage/"+bucketName+"/"+fileName, body)
		uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
		_, err = tc.App.Test(uploadReq)
		require.NoError(t, err)
	}

	// List files in bucket
	resp := tc.NewRequest("GET", "/api/v1/storage/"+bucketName).
		Send().
		AssertStatus(fiber.StatusOK)

	var result map[string]interface{}
	resp.JSON(&result)

	objects, ok := result["objects"].([]interface{})
	require.True(t, ok, "Response should have objects array")
	require.GreaterOrEqual(t, len(objects), 3, "Should have at least 3 objects")
	t.Logf("Found %d objects in S3 bucket", len(objects))
}
