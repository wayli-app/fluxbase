package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorageAPI_CreateBucket(t *testing.T) {
	app, _, db := setupStorageTestServer(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/buckets/test-bucket", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, "test-bucket", result["bucket"])
}

func TestStorageAPI_CreateBucketAlreadyExists(t *testing.T) {
	app, _, db := setupStorageTestServer(t)
	defer db.Close()

	// Create bucket first time
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/buckets/existing-bucket", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Create bucket second time
	req = httptest.NewRequest(http.MethodPost, "/api/v1/storage/buckets/existing-bucket", nil)
	resp, err = app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should return 409 Conflict
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestStorageAPI_ListBuckets(t *testing.T) {
	app, _, db := setupStorageTestServer(t)
	defer db.Close()

	// Create some buckets
	buckets := []string{"bucket1", "bucket2", "bucket3"}
	for _, bucket := range buckets {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/buckets/"+bucket, nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		resp.Body.Close()
	}

	// List buckets
	req := httptest.NewRequest(http.MethodGet, "/api/v1/storage/buckets", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	bucketsResult := result["buckets"].([]interface{})
	assert.GreaterOrEqual(t, len(bucketsResult), 3)
}

func TestStorageAPI_DeleteBucket(t *testing.T) {
	app, _, db := setupStorageTestServer(t)
	defer db.Close()

	// Create bucket
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/buckets/to-delete", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	resp.Body.Close()

	// Delete bucket
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/storage/buckets/to-delete", nil)
	resp, err = app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestStorageAPI_DeleteBucketNotEmpty(t *testing.T) {
	app, _, db := setupStorageTestServer(t)
	defer db.Close()

	// Create bucket
	createTestBucket(t, app, "nonempty-bucket")

	// Upload a file
	uploadTestFile(t, app, "nonempty-bucket", "file.txt", "content")

	// Try to delete non-empty bucket
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/storage/buckets/nonempty-bucket", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusConflict, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Contains(t, result["error"], "not empty")
}

func TestStorageAPI_InvalidBucketName(t *testing.T) {
	app, _, db := setupStorageTestServer(t)
	defer db.Close()

	// Try to create bucket with invalid name
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/buckets/Invalid_Bucket!", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should succeed (validation is provider-specific)
	// Local storage is more lenient than S3
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}
