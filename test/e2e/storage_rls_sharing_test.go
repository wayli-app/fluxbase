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

// TestStorageRLS_FileSharing verifies share/revoke functionality
func TestStorageRLS_FileSharing(t *testing.T) {
	tc := test.NewRLSTestContext(t)
	defer tc.Close()

	// Clean up storage for this test
	tc.CleanupStorageFiles()

	// Create two users
	user1Email := "user1-" + test.RandomEmail()
	_, user1Token := tc.CreateTestUser(user1Email, "password123")

	user2Email := "user2-" + test.RandomEmail()
	user2ID, user2Token := tc.CreateTestUser(user2Email, "password123")

	// Create service key and bucket
	serviceKey := tc.CreateServiceKey("test-bucket-creation")

	bucketName := "shared-bucket"
	tc.NewRequest("POST", "/api/v1/storage/buckets/"+bucketName).
		WithServiceKey(serviceKey).
		Send().
		AssertStatus(fiber.StatusCreated)

	// User1 uploads file
	fileName := "document.pdf"
	fileContent := []byte("confidential content")

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

	// User2 cannot access initially
	downloadResp := tc.NewRequest("GET", "/api/v1/storage/"+bucketName+"/"+fileName).
		WithAuth(user2Token).
		Send()

	require.Contains(t, []int{fiber.StatusNotFound, fiber.StatusForbidden},
		downloadResp.Status(), "User2 should not have access before sharing")
	t.Logf("✅ User2 cannot access before sharing (status: %d)", downloadResp.Status())

	// User1 shares with read permission
	shareResp := tc.NewRequest("POST", "/api/v1/storage/"+bucketName+"/"+fileName+"/share").
		WithAuth(user1Token).
		WithBody(map[string]interface{}{
			"user_id":    user2ID,
			"permission": "read",
		}).
		Send()

	require.Contains(t, []int{fiber.StatusOK, fiber.StatusCreated},
		shareResp.Status(), "Share request should succeed")
	t.Logf("✅ User1 shared file with user2 (read permission)")

	// User2 can now read
	download2Resp := tc.NewRequest("GET", "/api/v1/storage/"+bucketName+"/"+fileName).
		WithAuth(user2Token).
		Send()

	require.Equal(t, fiber.StatusOK, download2Resp.Status(), "User2 should be able to read shared file")
	require.Equal(t, fileContent, download2Resp.Body(), "Downloaded content should match")
	t.Logf("✅ User2 can now download shared file")

	// User2 tries to delete (should fail - only has read permission)
	deleteResp := tc.NewRequest("DELETE", "/api/v1/storage/"+bucketName+"/"+fileName).
		WithAuth(user2Token).
		Send()

	require.Contains(t, []int{fiber.StatusForbidden, fiber.StatusNotFound},
		deleteResp.Status(), "User2 should not be able to delete with only read permission")
	t.Logf("✅ User2 cannot delete with read permission (status: %d)", deleteResp.Status())

	// User1 upgrades to write permission
	upgradeResp := tc.NewRequest("POST", "/api/v1/storage/"+bucketName+"/"+fileName+"/share").
		WithAuth(user1Token).
		WithBody(map[string]interface{}{
			"user_id":    user2ID,
			"permission": "write",
		}).
		Send()

	require.Contains(t, []int{fiber.StatusOK, fiber.StatusCreated},
		upgradeResp.Status(), "Upgrade permission should succeed")
	t.Logf("✅ User1 upgraded user2's permission to write")

	// User2 can now delete
	delete2Resp := tc.NewRequest("DELETE", "/api/v1/storage/"+bucketName+"/"+fileName).
		WithAuth(user2Token).
		Send()

	require.Contains(t, []int{fiber.StatusOK, fiber.StatusNoContent},
		delete2Resp.Status(), "User2 should be able to delete with write permission")
	t.Logf("✅ User2 can delete with write permission")

	// Re-upload and test revoke
	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)
	part, err = writer.CreateFormFile("file", fileName)
	require.NoError(t, err)
	_, err = part.Write([]byte("content v2"))
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)

	uploadReq = httptest.NewRequest("POST", "/api/v1/storage/"+bucketName+"/"+fileName, body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadReq.Header.Set("Authorization", "Bearer "+user1Token)

	uploadResp, err = tc.App.Test(uploadReq)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusCreated, uploadResp.StatusCode)

	// Share again
	tc.NewRequest("POST", "/api/v1/storage/"+bucketName+"/"+fileName+"/share").
		WithAuth(user1Token).
		WithBody(map[string]interface{}{
			"user_id":    user2ID,
			"permission": "read",
		}).
		Send()

	// User1 revokes access
	revokeResp := tc.NewRequest("DELETE", "/api/v1/storage/"+bucketName+"/"+fileName+"/share/"+user2ID).
		WithAuth(user1Token).
		Send()

	require.Contains(t, []int{fiber.StatusOK, fiber.StatusNoContent},
		revokeResp.Status(), "Revoke should succeed")
	t.Logf("✅ User1 revoked user2's access")

	// User2 loses access
	download3Resp := tc.NewRequest("GET", "/api/v1/storage/"+bucketName+"/"+fileName).
		WithAuth(user2Token).
		Send()

	require.Contains(t, []int{fiber.StatusNotFound, fiber.StatusForbidden},
		download3Resp.Status(), "User2 should lose access after revoke")
	t.Logf("✅ User2 cannot access after revoke (status: %d)", download3Resp.Status())

	// Test listShares
	// Re-share to test listing
	tc.NewRequest("POST", "/api/v1/storage/"+bucketName+"/"+fileName+"/share").
		WithAuth(user1Token).
		WithBody(map[string]interface{}{
			"user_id":    user2ID,
			"permission": "read",
		}).
		Send()

	listSharesResp := tc.NewRequest("GET", "/api/v1/storage/"+bucketName+"/"+fileName+"/shares").
		WithAuth(user1Token).
		Send()

	if listSharesResp.Status() == fiber.StatusOK {
		var listResult map[string]interface{}
		listSharesResp.JSON(&listResult)

		shares, ok := listResult["shares"].([]interface{})
		require.True(t, ok, "Response should have shares array")
		require.Len(t, shares, 1, "Should have exactly one share")

		share := shares[0].(map[string]interface{})
		require.Equal(t, user2ID, share["user_id"], "Share should be for user2")
		require.Equal(t, "read", share["permission"], "Permission should be read")
		t.Logf("✅ List shares returns correct data")
	}

	t.Logf("✅ File sharing functionality works correctly")
}
