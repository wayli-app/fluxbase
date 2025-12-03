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

// TestStorageRLS_BucketVisibilityToggle verifies public/private toggle
func TestStorageRLS_BucketVisibilityToggle(t *testing.T) {
	tc := test.NewRLSTestContext(t)
	defer tc.Close()

	// Clean up storage for this test
	tc.CleanupStorageFiles()

	// Create user
	userEmail := "user-" + test.RandomEmail()
	_, userToken := tc.CreateTestUser(userEmail, "password123")

	// Create service key for bucket management
	serviceKey := tc.CreateServiceKey("test-bucket-management")

	bucketName := "test-bucket"
	tc.NewRequest("POST", "/api/v1/storage/buckets/"+bucketName).
		WithServiceKey(serviceKey).
		Send().
		AssertStatus(fiber.StatusCreated)

	// Verify bucket is private by default
	results := tc.QuerySQLAsSuperuser(
		"SELECT public FROM storage.buckets WHERE id = $1", bucketName)
	require.Len(t, results, 1, "Bucket should exist")
	require.False(t, results[0]["public"].(bool), "Bucket should be private by default")
	t.Logf("✅ Bucket created as private by default")

	// Upload file as user
	fileName := "file.txt"
	fileContent := []byte("test data")

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
	t.Logf("✅ User uploaded file to private bucket")

	// Unauthenticated access fails
	downloadResp := tc.NewRequest("GET", "/api/v1/storage/"+bucketName+"/"+fileName).
		Unauthenticated().
		Send()

	require.Contains(t, []int{fiber.StatusNotFound, fiber.StatusForbidden, fiber.StatusUnauthorized},
		downloadResp.Status(), "Unauthenticated access should fail for private bucket")
	t.Logf("✅ Unauthenticated user cannot access private bucket (status: %d)", downloadResp.Status())

	// Service key toggles bucket to public
	updateResp := tc.NewRequest("PUT", "/api/v1/storage/buckets/"+bucketName).
		WithServiceKey(serviceKey).
		WithBody(map[string]interface{}{
			"public": true,
		}).
		Send()

	require.Contains(t, []int{fiber.StatusOK, fiber.StatusNoContent},
		updateResp.Status(), "Service key should be able to toggle bucket to public")
	t.Logf("✅ Service key toggled bucket to public")

	// Verify bucket is now public
	results = tc.QuerySQLAsSuperuser(
		"SELECT public FROM storage.buckets WHERE id = $1", bucketName)
	require.Len(t, results, 1, "Bucket should exist")
	require.True(t, results[0]["public"].(bool), "Bucket should now be public")

	// Unauthenticated access now works
	download2Resp := tc.NewRequest("GET", "/api/v1/storage/"+bucketName+"/"+fileName).
		Unauthenticated().
		Send()

	require.Equal(t, fiber.StatusOK, download2Resp.Status(),
		"Unauthenticated access should work for public bucket")
	require.Equal(t, fileContent, download2Resp.Body(), "Content should match")
	t.Logf("✅ Unauthenticated user can now access public bucket")

	// Toggle back to private
	update2Resp := tc.NewRequest("PUT", "/api/v1/storage/buckets/"+bucketName).
		WithServiceKey(serviceKey).
		WithBody(map[string]interface{}{
			"public": false,
		}).
		Send()

	require.Contains(t, []int{fiber.StatusOK, fiber.StatusNoContent},
		update2Resp.Status(), "Service key should be able to toggle bucket back to private")
	t.Logf("✅ Service key toggled bucket back to private")

	// Verify bucket is private again
	results = tc.QuerySQLAsSuperuser(
		"SELECT public FROM storage.buckets WHERE id = $1", bucketName)
	require.Len(t, results, 1, "Bucket should exist")
	require.False(t, results[0]["public"].(bool), "Bucket should be private again")

	// Access revoked again
	download3Resp := tc.NewRequest("GET", "/api/v1/storage/"+bucketName+"/"+fileName).
		Unauthenticated().
		Send()

	require.Contains(t, []int{fiber.StatusNotFound, fiber.StatusForbidden, fiber.StatusUnauthorized},
		download3Resp.Status(), "Unauthenticated access should fail again for private bucket")
	t.Logf("✅ Unauthenticated user cannot access after bucket made private again (status: %d)", download3Resp.Status())

	// Regular user cannot toggle bucket settings
	update3Resp := tc.NewRequest("PUT", "/api/v1/storage/buckets/"+bucketName).
		WithAuth(userToken).
		WithBody(map[string]interface{}{
			"public": true,
		}).
		Send()

	require.Contains(t, []int{fiber.StatusForbidden, fiber.StatusUnauthorized, fiber.StatusNotFound},
		update3Resp.Status(), "Regular user should not be able to update bucket settings")
	t.Logf("✅ Regular user cannot update bucket settings (status: %d)", update3Resp.Status())

	// Verify bucket is still private (not changed by non-admin attempt)
	results = tc.QuerySQLAsSuperuser(
		"SELECT public FROM storage.buckets WHERE id = $1", bucketName)
	require.Len(t, results, 1, "Bucket should exist")
	require.False(t, results[0]["public"].(bool), "Bucket should still be private")

	t.Logf("✅ Bucket visibility toggle works correctly")
}
