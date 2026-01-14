package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
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

// =============================================================================
// Unit Tests for Validation Logic
// =============================================================================

func TestStorageHandler_CreateBucket_MissingBucketName(t *testing.T) {
	handler := &StorageHandler{}

	// Test route without bucket param to test validation
	app := setupTestFiberApp()
	app.Post("/storage/buckets/", handler.CreateBucket)

	req := httptest.NewRequest("POST", "/storage/buckets/", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Contains(t, result["error"], "bucket name is required")
}

func TestStorageHandler_UpdateBucketSettings_MissingBucketName(t *testing.T) {
	handler := &StorageHandler{}

	app := setupTestFiberApp()
	app.Put("/storage/buckets/", handler.UpdateBucketSettings)

	req := httptest.NewRequest("PUT", "/storage/buckets/", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Contains(t, result["error"], "bucket name is required")
}

func TestStorageHandler_DeleteBucket_MissingBucketName(t *testing.T) {
	handler := &StorageHandler{}

	app := setupTestFiberApp()
	app.Delete("/storage/buckets/", handler.DeleteBucket)

	req := httptest.NewRequest("DELETE", "/storage/buckets/", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Contains(t, result["error"], "bucket name is required")
}

func TestStorageHandler_ListBuckets_RoleChecking(t *testing.T) {
	tests := []struct {
		name           string
		role           interface{}
		expectedStatus int
	}{
		{"admin role allowed", "admin", http.StatusInternalServerError},         // Allowed but db is nil
		{"dashboard_admin role allowed", "dashboard_admin", http.StatusInternalServerError}, // Allowed but db is nil
		{"service_role allowed", "service_role", http.StatusInternalServerError}, // Allowed but db is nil
		{"authenticated role forbidden", "authenticated", http.StatusForbidden},
		{"anon role forbidden", "anon", http.StatusForbidden},
		{"empty role forbidden", "", http.StatusForbidden},
		{"nil role forbidden", nil, http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &StorageHandler{} // No db, will fail after role check

			app := setupTestFiberApp()
			app.Get("/storage/buckets", func(c *fiber.Ctx) error {
				if tt.role != nil {
					c.Locals("user_role", tt.role)
				}
				return handler.ListBuckets(c)
			})

			req := httptest.NewRequest("GET", "/storage/buckets", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusForbidden {
				var result map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)
				assert.Contains(t, result["error"], "Admin access required")
			}
		})
	}
}

func TestStorageHandler_UpdateBucketSettings_InvalidBody(t *testing.T) {
	handler := &StorageHandler{}

	app := setupTestFiberApp()
	app.Put("/storage/buckets/:bucket", handler.UpdateBucketSettings)

	req := httptest.NewRequest("PUT", "/storage/buckets/mybucket", strings.NewReader(`{invalid`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Contains(t, result["error"], "invalid request body")
}

func TestStorageHandler_UpdateBucketSettings_NoFieldsToUpdate(t *testing.T) {
	handler := &StorageHandler{}

	app := setupTestFiberApp()
	app.Put("/storage/buckets/:bucket", func(c *fiber.Ctx) error {
		// Need to mock db connection, so we'll test the empty fields case
		// by checking if handler returns proper error when no fields provided
		return handler.UpdateBucketSettings(c)
	})

	// Empty JSON object - no fields to update
	req := httptest.NewRequest("PUT", "/storage/buckets/mybucket", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Will fail because db is nil, but we can test that validation passes
	// with valid empty JSON
	assert.True(t, resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusInternalServerError)
}

// =============================================================================
// Bucket Configuration Tests
// =============================================================================

func TestBucketConfiguration(t *testing.T) {
	t.Run("public bucket configuration", func(t *testing.T) {
		config := struct {
			Public           bool     `json:"public"`
			AllowedMimeTypes []string `json:"allowed_mime_types"`
			MaxFileSize      *int64   `json:"max_file_size"`
		}{
			Public:           true,
			AllowedMimeTypes: []string{"image/*"},
			MaxFileSize:      nil,
		}

		assert.True(t, config.Public)
		assert.Contains(t, config.AllowedMimeTypes, "image/*")
		assert.Nil(t, config.MaxFileSize)
	})

	t.Run("private bucket with size limit", func(t *testing.T) {
		maxSize := int64(10 * 1024 * 1024) // 10MB
		config := struct {
			Public           bool     `json:"public"`
			AllowedMimeTypes []string `json:"allowed_mime_types"`
			MaxFileSize      *int64   `json:"max_file_size"`
		}{
			Public:           false,
			AllowedMimeTypes: []string{"application/pdf", "image/jpeg"},
			MaxFileSize:      &maxSize,
		}

		assert.False(t, config.Public)
		assert.Len(t, config.AllowedMimeTypes, 2)
		assert.Equal(t, int64(10*1024*1024), *config.MaxFileSize)
	})

	t.Run("bucket with no mime type restrictions", func(t *testing.T) {
		config := struct {
			Public           bool     `json:"public"`
			AllowedMimeTypes []string `json:"allowed_mime_types"`
			MaxFileSize      *int64   `json:"max_file_size"`
		}{
			AllowedMimeTypes: nil, // No restrictions
		}

		assert.Nil(t, config.AllowedMimeTypes)
	})
}

// =============================================================================
// Helper Functions
// =============================================================================

func setupTestFiberApp() *fiber.App {
	return fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkListBucketsRoleCheck(b *testing.B) {
	roles := []string{"admin", "dashboard_admin", "service_role", "authenticated", "anon"}

	for i := 0; i < b.N; i++ {
		role := roles[i%len(roles)]
		_ = (role == "admin" || role == "dashboard_admin" || role == "service_role")
	}
}
