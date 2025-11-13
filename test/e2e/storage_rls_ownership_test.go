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

// TestStorageRLS_FileOwnership verifies owner-based access control
func TestStorageRLS_FileOwnership(t *testing.T) {
	tc := test.NewRLSTestContext(t) // RLS policies will be enforced
	defer tc.Close()

	// Clean up storage for this test
	tc.CleanupStorageFiles()

	// Create two users
	user1Email := "user1-" + test.RandomEmail()
	user1ID, user1Token := tc.CreateTestUser(user1Email, "password123")

	user2Email := "user2-" + test.RandomEmail()
	user2ID, user2Token := tc.CreateTestUser(user2Email, "password123")

	t.Logf("Created user1: %s (%s)", user1Email, user1ID)
	t.Logf("Created user2: %s (%s)", user2Email, user2ID)

	// Create private bucket using service key
	serviceKey := tc.CreateServiceKey("test-bucket-creation")

	bucketName := "private-bucket"
	tc.NewRequest("POST", "/api/v1/storage/buckets/"+bucketName).
		WithServiceKey(serviceKey).
		Send().
		AssertStatus(fiber.StatusCreated)

	// User1 uploads file
	fileName := "user1-file.txt"
	fileContent := []byte("user1 data")

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
	require.Equal(t, fiber.StatusCreated, uploadResp.StatusCode, "User1 should upload successfully")

	// User1 can access their own file
	downloadResp := tc.NewRequest("GET", "/api/v1/storage/"+bucketName+"/"+fileName).
		WithAuth(user1Token).
		Send()

	require.Equal(t, fiber.StatusOK, downloadResp.Status(), "User1 should access their own file")
	require.Equal(t, fileContent, downloadResp.Body(), "Content should match")

	// User2 should NOT see user1's file in list
	listResp := tc.NewRequest("GET", "/api/v1/storage/"+bucketName).
		WithAuth(user2Token).
		Send()

	// List should succeed but return empty or exclude user1's file
	if listResp.Status() == fiber.StatusOK {
		var listResult map[string]interface{}
		listResp.JSON(&listResult)

		files, ok := listResult["objects"].([]interface{})
		if ok {
			// Check that user1's file is NOT in the list
			for _, f := range files {
				fileMap, ok := f.(map[string]interface{})
				if ok {
					filePath, _ := fileMap["path"].(string)
					require.NotEqual(t, fileName, filePath,
						"User2 should not see user1's file in listing")
				}
			}
			t.Logf("✅ User2 cannot see user1's file in list (saw %d files)", len(files))
		}
	}

	// User2 CANNOT download user1's file
	download2Resp := tc.NewRequest("GET", "/api/v1/storage/"+bucketName+"/"+fileName).
		WithAuth(user2Token).
		Send()

	require.Contains(t, []int{fiber.StatusNotFound, fiber.StatusForbidden},
		download2Resp.Status(),
		"User2 should not be able to download user1's file (expected 404 or 403, got %d)",
		download2Resp.Status())
	t.Logf("✅ User2 cannot download user1's file (got status %d)", download2Resp.Status())

	// User1 can delete their own file
	deleteResp := tc.NewRequest("DELETE", "/api/v1/storage/"+bucketName+"/"+fileName).
		WithAuth(user1Token).
		Send()

	require.Contains(t, []int{fiber.StatusOK, fiber.StatusNoContent},
		deleteResp.Status(), "User1 should be able to delete their own file")
	t.Logf("✅ User1 can delete their own file")

	// Re-upload for next test
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
	uploadReq.Header.Set("Authorization", "Bearer "+user1Token)

	uploadResp, err = tc.App.Test(uploadReq)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusCreated, uploadResp.StatusCode)

	// User2 cannot delete user1's file
	delete2Resp := tc.NewRequest("DELETE", "/api/v1/storage/"+bucketName+"/"+fileName).
		WithAuth(user2Token).
		Send()

	require.Contains(t, []int{fiber.StatusNotFound, fiber.StatusForbidden},
		delete2Resp.Status(),
		"User2 should not be able to delete user1's file")
	t.Logf("✅ User2 cannot delete user1's file (got status %d)", delete2Resp.Status())

	// Verify file still exists (using superuser query to bypass RLS)
	results := tc.QuerySQLAsSuperuser(
		"SELECT owner_id FROM storage.objects WHERE bucket_id = $1 AND path = $2",
		bucketName, fileName)
	require.Len(t, results, 1, "File should still exist after user2's failed delete")
	require.Equal(t, user1ID, results[0]["owner_id"], "File should still be owned by user1")

	t.Logf("✅ File ownership RLS policies working correctly")
}
