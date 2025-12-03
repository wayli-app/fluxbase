package e2e

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/fluxbase-eu/fluxbase/test"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

// TestStorageRLS_AdminAccess verifies dashboard admins can access everything
func TestStorageRLS_AdminAccess(t *testing.T) {
	tc := test.NewTestContext(t) // Use regular context for dashboard admin testing
	defer tc.Close()

	// Clean up storage for this test
	tc.CleanupStorageFiles()

	// Create dashboard admin
	adminEmail := "admin-" + test.RandomEmail()
	_, adminToken := tc.CreateDashboardAdminUser(adminEmail, "password123")

	// Create regular user
	userEmail := "user-" + test.RandomEmail()
	userID, userToken := tc.CreateTestUser(userEmail, "password123")

	// Admin creates a private bucket
	bucketName := "user-private"
	tc.NewRequest("POST", "/api/v1/storage/buckets/"+bucketName).
		WithAuth(adminToken).
		Send().
		AssertStatus(fiber.StatusCreated)

	// Regular user uploads file to private bucket
	fileName := "secret.txt"
	fileContent := []byte("secret data")

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
	require.Equal(t, fiber.StatusCreated, uploadResp.StatusCode, "User should be able to upload file")

	// Verify the file has owner_id set to regular user
	results := tc.QuerySQLAsSuperuser(
		"SELECT owner_id FROM storage.objects WHERE bucket_id = $1 AND path = $2",
		bucketName, fileName)
	require.Len(t, results, 1, "File should exist in database")
	require.Equal(t, userID, results[0]["owner_id"], "File should be owned by regular user")

	// Admin should be able to see all buckets
	listResp := tc.NewRequest("GET", "/api/v1/storage/buckets").
		WithAuth(adminToken).
		Send().
		AssertStatus(fiber.StatusOK)

	var listResult map[string]interface{}
	listResp.JSON(&listResult)

	buckets, ok := listResult["buckets"].([]interface{})
	require.True(t, ok, "Response should have buckets array")

	// Check if user-private bucket is in the list
	foundBucket := false
	for _, b := range buckets {
		bucketMap, ok := b.(map[string]interface{})
		if ok && bucketMap["id"] == bucketName {
			foundBucket = true
			break
		}
	}
	require.True(t, foundBucket, "Admin should see the user-private bucket")

	// Admin should be able to download the file (even though they don't own it)
	downloadResp := tc.NewRequest("GET", "/api/v1/storage/"+bucketName+"/"+fileName).
		WithAuth(adminToken).
		Send()

	// Admin with dashboard_admin role should bypass RLS and access the file
	require.Equal(t, fiber.StatusOK, downloadResp.Status(),
		"Admin should be able to download user's private file (bypasses RLS)")

	downloadedContent := downloadResp.Body()
	require.Equal(t, fileContent, downloadedContent, "Downloaded content should match uploaded content")

	// Admin can update bucket settings
	updateResp := tc.NewRequest("PUT", "/api/v1/storage/buckets/"+bucketName).
		WithAuth(adminToken).
		WithBody(map[string]interface{}{
			"public": true,
		}).
		Send()

	require.Contains(t, []int{fiber.StatusOK, fiber.StatusNoContent},
		updateResp.Status(), "Admin should be able to update bucket settings")

	// Verify bucket is now public
	results = tc.QuerySQLAsSuperuser(
		"SELECT public FROM storage.buckets WHERE id = $1", bucketName)
	require.Len(t, results, 1, "Bucket should exist")
	require.True(t, results[0]["public"].(bool), "Bucket should now be public")

	t.Logf("✅ Admin can access all buckets and files, and update bucket settings")
}
