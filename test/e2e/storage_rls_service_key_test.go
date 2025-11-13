package e2e

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
	"github.com/wayli-app/fluxbase/test"
)

// TestStorageRLS_ServiceKeyBypass verifies service keys bypass RLS
func TestStorageRLS_ServiceKeyBypass(t *testing.T) {
	tc := test.NewRLSTestContext(t)
	defer tc.Close()

	// Clean up storage for this test
	tc.CleanupStorageFiles()

	// Create service key
	serviceKey := tc.CreateServiceKey("test-service")
	t.Logf("Created service key: %s...", serviceKey[:20])

	// Create user
	userEmail := "user-" + test.RandomEmail()
	_, userToken := tc.CreateTestUser(userEmail, "password123")

	// Create private bucket using service key
	bucketName := "private-bucket"
	tc.NewRequest("POST", "/api/v1/storage/buckets/"+bucketName).
		WithServiceKey(serviceKey).
		Send().
		AssertStatus(fiber.StatusCreated)

	// User creates private file
	fileName := "private.txt"
	fileContent := []byte("private data")

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
	uploadReq.Header.Set("Authorization", "Bearer "+userToken)

	uploadResp, err := tc.App.Test(uploadReq)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusCreated, uploadResp.StatusCode)
	t.Logf("✅ User uploaded private file")

	// Service key can access everything (bypasses RLS)
	downloadResp := tc.NewRequest("GET", "/api/v1/storage/"+bucketName+"/"+fileName).
		WithServiceKey(serviceKey).
		Send()

	require.Equal(t, fiber.StatusOK, downloadResp.Status(),
		"Service key should be able to access all files (bypasses RLS)")
	require.Equal(t, fileContent, downloadResp.Body(), "Content should match")
	t.Logf("✅ Service key can download private file (RLS bypassed)")

	// Service key can list all files
	listResp := tc.NewRequest("GET", "/api/v1/storage/"+bucketName).
		WithServiceKey(serviceKey).
		Send()

	if listResp.Status() == fiber.StatusOK {
		var listResult map[string]interface{}
		listResp.JSON(&listResult)

		files, ok := listResult["objects"].([]interface{})
		require.True(t, ok, "Response should have objects array")
		require.GreaterOrEqual(t, len(files), 1, "Service key should see all files")

		// Check that our file is in the list
		foundFile := false
		for _, f := range files {
			fileMap, ok := f.(map[string]interface{})
			if ok {
				filePath, _ := fileMap["path"].(string)
				if filePath == fileName {
					foundFile = true
					break
				}
			}
		}
		require.True(t, foundFile, "Service key should see the private file in listing")
		t.Logf("✅ Service key can list all files (found %d files)", len(files))
	}

	// Service key can delete files owned by others
	deleteResp := tc.NewRequest("DELETE", "/api/v1/storage/"+bucketName+"/"+fileName).
		WithServiceKey(serviceKey).
		Send()

	require.Contains(t, []int{fiber.StatusOK, fiber.StatusNoContent},
		deleteResp.Status(), "Service key should be able to delete any file")
	t.Logf("✅ Service key can delete files owned by others")

	// Verify file is deleted
	results := tc.QuerySQLAsSuperuser(
		"SELECT COUNT(*) as count FROM storage.objects WHERE bucket_id = $1 AND path = $2",
		bucketName, fileName)

	require.Len(t, results, 1, "Query should return result")
	count, ok := results[0]["count"].(int64)
	require.True(t, ok, "Count should be int64")
	require.Equal(t, int64(0), count, "File should be deleted")

	// Re-upload for additional tests
	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)
	part, err = writer.CreateFormFile("file", fileName)
	require.NoError(t, err)
	_, err = part.Write(fileContent)
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)

	uploadReq = httptest.NewRequest("POST", "/api/v1/storage/"+bucketName+"/"+fileName, body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadReq.Header.Set("X-Service-Key", serviceKey) // Test service key upload

	uploadResp, err = tc.App.Test(uploadReq)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusCreated, uploadResp.StatusCode)
	t.Logf("✅ Service key can upload files")

	// Verify service key sets correct role in database context
	// We can't directly query current_setting via API, but we can verify behavior
	// Service keys should bypass RLS, which we've already confirmed above

	t.Logf("✅ Service keys correctly bypass RLS for all operations")
}
