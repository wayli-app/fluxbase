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

// TestStorageRLS_UserIsolation verifies strict RLS enforcement at database level
func TestStorageRLS_UserIsolation(t *testing.T) {
	tc := test.NewRLSTestContext(t)
	defer tc.Close()

	// Clean up storage for this test
	tc.CleanupStorageFiles()

	// Create two users
	user1Email := "user1-" + test.RandomEmail()
	user1ID, user1Token := tc.CreateTestUser(user1Email, "password123")

	user2Email := "user2-" + test.RandomEmail()
	user2ID, user2Token := tc.CreateTestUser(user2Email, "password123")

	// Create service key and bucket
	serviceKey := tc.CreateServiceKey("test-bucket-creation")

	bucketName := "private-bucket"
	tc.NewRequest("POST", "/api/v1/storage/buckets/"+bucketName).
		WithServiceKey(serviceKey).
		Send().
		AssertStatus(fiber.StatusCreated)

	// User1 uploads file
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
	uploadReq.Header.Set("Authorization", "Bearer "+user1Token)

	uploadResp, err := tc.App.Test(uploadReq)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusCreated, uploadResp.StatusCode)
	t.Logf("✅ User1 uploaded file")

	// User2 lists files - should not see user1's file
	listResp := tc.NewRequest("GET", "/api/v1/storage/"+bucketName).
		WithAuth(user2Token).
		Send()

	if listResp.Status() == fiber.StatusOK {
		var listResult map[string]interface{}
		listResp.JSON(&listResult)

		files, ok := listResult["objects"].([]interface{})
		if ok {
			require.Len(t, files, 0, "User2 should not see any files (user1's file should be hidden by RLS)")
			t.Logf("✅ User2 cannot see user1's file in listing (RLS filtered)")
		}
	}

	// User2 tries direct download - should fail
	downloadResp := tc.NewRequest("GET", "/api/v1/storage/"+bucketName+"/"+fileName).
		WithAuth(user2Token).
		Send()

	require.Contains(t, []int{fiber.StatusNotFound, fiber.StatusForbidden},
		downloadResp.Status(), "User2 should not be able to download user1's file")
	t.Logf("✅ User2 cannot download user1's file (status: %d)", downloadResp.Status())

	// User2 tries to delete - should fail
	deleteResp := tc.NewRequest("DELETE", "/api/v1/storage/"+bucketName+"/"+fileName).
		WithAuth(user2Token).
		Send()

	require.Contains(t, []int{fiber.StatusNotFound, fiber.StatusForbidden},
		deleteResp.Status(), "User2 should not be able to delete user1's file")
	t.Logf("✅ User2 cannot delete user1's file (status: %d)", deleteResp.Status())

	// CRITICAL TEST: Verify RLS is actually filtering at database level, not just API
	// Query database directly as user2's role to ensure RLS policies are enforced
	results := tc.QuerySQLAsRLSUser(
		"SELECT COUNT(*) as count FROM storage.objects WHERE bucket_id = $1",
		user2ID, // Set RLS context as user2
		bucketName)

	require.Len(t, results, 1, "Query should return result")
	count, ok := results[0]["count"].(int64)
	require.True(t, ok, "Count should be int64")
	require.Equal(t, int64(0), count, "RLS should hide user1's file at database level when queried as user2")
	t.Logf("✅ RLS filters at database level (user2 sees 0 rows)")

	// Query as user1 should see the file
	results = tc.QuerySQLAsRLSUser(
		"SELECT COUNT(*) as count FROM storage.objects WHERE bucket_id = $1",
		user1ID, // Set RLS context as user1
		bucketName)

	require.Len(t, results, 1, "Query should return result")
	count, ok = results[0]["count"].(int64)
	require.True(t, ok, "Count should be int64")
	require.Equal(t, int64(1), count, "User1 should see their own file at database level")
	t.Logf("✅ RLS allows user1 to see their own file at database level")

	// Verify superuser can see all files (bypasses RLS)
	results = tc.QuerySQLAsSuperuser(
		"SELECT COUNT(*) as count FROM storage.objects WHERE bucket_id = $1",
		bucketName)

	require.Len(t, results, 1, "Query should return result")
	count, ok = results[0]["count"].(int64)
	require.True(t, ok, "Count should be int64")
	require.Equal(t, int64(1), count, "Superuser should see all files (bypasses RLS)")
	t.Logf("✅ Superuser can see all files (RLS bypass confirmed)")

	t.Logf("✅ User isolation via RLS works correctly at both API and database level")
}
