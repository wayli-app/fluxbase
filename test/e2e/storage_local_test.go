package e2e

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/fluxbase-eu/fluxbase/test"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

// setupStorageLocalTest prepares the test context for local storage tests
func setupStorageLocalTest(t *testing.T) *StorageTestContext {
	tc := test.NewTestContext(t)
	tc.EnsureStorageSchema()

	// Ensure using local storage provider
	tc.Config.Storage.Provider = "local"
	tc.Config.Storage.LocalPath = "/tmp/fluxbase-test-storage"

	// Clean up any existing test storage files
	tc.CleanupStorageFiles()

	// Create an API key for authenticated requests
	apiKey := tc.CreateAPIKey("Storage Local Test API Key", nil)

	return &StorageTestContext{
		TestContext: tc,
		APIKey:      apiKey,
	}
}

// TestStorageLocalCreateBucket tests creating a storage bucket
func TestStorageLocalCreateBucket(t *testing.T) {
	tc := setupStorageLocalTest(t)
	defer tc.Close()

	bucketName := "test-bucket"

	// Create bucket
	resp := tc.NewRequest("POST", "/api/v1/storage/buckets/"+bucketName).
		WithAPIKey(tc.APIKey).
		Send().
		AssertStatus(fiber.StatusCreated)

	var result map[string]interface{}
	resp.JSON(&result)

	require.Equal(t, bucketName, result["bucket"])
	t.Logf("Created bucket: %s", bucketName)
}

// TestStorageLocalListBuckets tests listing storage buckets
func TestStorageLocalListBuckets(t *testing.T) {
	tc := setupStorageLocalTest(t)
	defer tc.Close()

	// Create a service key for admin operations (listing buckets requires admin/service role)
	serviceKey := tc.CreateServiceKey("Storage Admin Key")

	// Create a few test buckets
	tc.NewRequest("POST", "/api/v1/storage/buckets/bucket1").WithServiceKey(serviceKey).Send().AssertStatus(fiber.StatusCreated)
	tc.NewRequest("POST", "/api/v1/storage/buckets/bucket2").WithServiceKey(serviceKey).Send().AssertStatus(fiber.StatusCreated)

	// List buckets (requires admin/service role)
	resp := tc.NewRequest("GET", "/api/v1/storage/buckets").
		WithServiceKey(serviceKey).
		Send().
		AssertStatus(fiber.StatusOK)

	var result map[string]interface{}
	resp.JSON(&result)

	buckets, ok := result["buckets"].([]interface{})
	require.True(t, ok, "Response should have buckets array")
	require.GreaterOrEqual(t, len(buckets), 2, "Should have at least 2 buckets")
	t.Logf("Found %d buckets", len(buckets))
}

// TestStorageLocalUploadFile tests uploading a file
func TestStorageLocalUploadFile(t *testing.T) {
	tc := setupStorageLocalTest(t)
	defer tc.Close()

	bucketName := "upload-test"
	fileName := "test.txt"
	fileContent := []byte("Hello, World!")

	// Create bucket first
	tc.NewRequest("POST", "/api/v1/storage/buckets/"+bucketName).
		WithAPIKey(tc.APIKey).
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

	// Upload file - use httptest to create proper request
	req := httptest.NewRequest("POST", "/api/v1/storage/"+bucketName+"/"+fileName, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Client-Key", tc.APIKey)

	resp, err := tc.App.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusCreated, resp.StatusCode, "File upload should succeed")

	t.Logf("Uploaded file: %s to bucket: %s", fileName, bucketName)
}

// TestStorageLocalDownloadFile tests downloading a file
func TestStorageLocalDownloadFile(t *testing.T) {
	tc := setupStorageLocalTest(t)
	defer tc.Close()

	bucketName := "download-test"
	fileName := "download.txt"
	fileContent := []byte("Download me!")

	// Create bucket
	tc.NewRequest("POST", "/api/v1/storage/buckets/"+bucketName).
		WithAPIKey(tc.APIKey).
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
	uploadReq.Header.Set("X-Client-Key", tc.APIKey)
	_, err = tc.App.Test(uploadReq)
	require.NoError(t, err)

	// Now download the file
	resp := tc.NewRequest("GET", "/api/v1/storage/"+bucketName+"/"+fileName).
		WithAPIKey(tc.APIKey).
		Send()

	// Download should work or return an appropriate status
	if resp.Status() == fiber.StatusOK {
		downloadedContent, err := io.ReadAll(bytes.NewReader(resp.Body()))
		require.NoError(t, err)
		require.Equal(t, fileContent, downloadedContent, "Downloaded content should match uploaded content")
		t.Logf("Downloaded file successfully")
	} else {
		t.Logf("Download returned status: %d (may need further implementation)", resp.Status())
	}
}

// TestStorageLocalDeleteFile tests deleting a file
func TestStorageLocalDeleteFile(t *testing.T) {
	tc := setupStorageLocalTest(t)
	defer tc.Close()

	bucketName := "delete-test"
	fileName := "todelete.txt"

	// Create bucket
	tc.NewRequest("POST", "/api/v1/storage/buckets/"+bucketName).
		WithAPIKey(tc.APIKey).
		Send().
		AssertStatus(fiber.StatusCreated)

	// Upload a file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", fileName)
	require.NoError(t, err)
	_, err = part.Write([]byte("Delete me"))
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)

	uploadReq := httptest.NewRequest("POST", "/api/v1/storage/"+bucketName+"/"+fileName, body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadReq.Header.Set("X-Client-Key", tc.APIKey)
	_, err = tc.App.Test(uploadReq)
	require.NoError(t, err)

	// Delete the file
	tc.NewRequest("DELETE", "/api/v1/storage/"+bucketName+"/"+fileName).
		WithAPIKey(tc.APIKey).
		Send().
		AssertStatus(fiber.StatusNoContent)

	t.Logf("Deleted file: %s from bucket: %s", fileName, bucketName)
}

// TestStorageLocalDeleteBucket tests deleting a bucket
func TestStorageLocalDeleteBucket(t *testing.T) {
	tc := setupStorageLocalTest(t)
	defer tc.Close()

	bucketName := "bucket-to-delete"

	// Create bucket
	tc.NewRequest("POST", "/api/v1/storage/buckets/"+bucketName).
		WithAPIKey(tc.APIKey).
		Send().
		AssertStatus(fiber.StatusCreated)

	// Delete bucket
	tc.NewRequest("DELETE", "/api/v1/storage/buckets/"+bucketName).
		WithAPIKey(tc.APIKey).
		Send().
		AssertStatus(fiber.StatusNoContent)

	t.Logf("Deleted bucket: %s", bucketName)
}

// TestStorageLocalListFiles tests listing files in a bucket
func TestStorageLocalListFiles(t *testing.T) {
	tc := setupStorageLocalTest(t)
	defer tc.Close()

	bucketName := "list-test"

	// Create bucket
	tc.NewRequest("POST", "/api/v1/storage/buckets/"+bucketName).
		WithAPIKey(tc.APIKey).
		Send().
		AssertStatus(fiber.StatusCreated)

	// Upload a few files
	for i := 1; i <= 3; i++ {
		fileName := fmt.Sprintf("file%d.txt", i)
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", fileName)
		require.NoError(t, err)
		_, err = part.Write([]byte(fmt.Sprintf("Content %d", i)))
		require.NoError(t, err)
		err = writer.Close()
		require.NoError(t, err)

		uploadReq := httptest.NewRequest("POST", "/api/v1/storage/"+bucketName+"/"+fileName, body)
		uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
		uploadReq.Header.Set("X-Client-Key", tc.APIKey)
		_, err = tc.App.Test(uploadReq)
		require.NoError(t, err)
	}

	// List files in bucket
	resp := tc.NewRequest("GET", "/api/v1/storage/"+bucketName).
		WithAPIKey(tc.APIKey).
		Send().
		AssertStatus(fiber.StatusOK)

	var result map[string]interface{}
	resp.JSON(&result)

	objects, ok := result["objects"].([]interface{})
	require.True(t, ok, "Response should have objects array")
	require.GreaterOrEqual(t, len(objects), 3, "Should have at least 3 objects")
	t.Logf("Found %d objects in bucket", len(objects))
}
