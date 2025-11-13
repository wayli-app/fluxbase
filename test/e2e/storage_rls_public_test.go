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

// TestStorageRLS_PublicBucketAccess verifies unauthenticated access to public buckets
func TestStorageRLS_PublicBucketAccess(t *testing.T) {
	tc := test.NewRLSTestContext(t)
	defer tc.Close()

	// Clean up storage for this test
	tc.CleanupStorageFiles()

	// Create user and service key (for bucket creation)
	userEmail := "user-" + test.RandomEmail()
	_, userToken := tc.CreateTestUser(userEmail, "password123")

	serviceKey := tc.CreateServiceKey("test-bucket-creation")

	// Create public bucket using service key
	bucketName := "public-assets"
	tc.NewRequest("POST", "/api/v1/storage/buckets/"+bucketName).
		WithServiceKey(serviceKey).
		WithBody(map[string]interface{}{
			"public": true,
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	// Upload file as authenticated user
	fileName := "logo.png"
	fileContent := []byte("PNG data")

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
	t.Logf("✅ User uploaded file to public bucket")

	// Unauthenticated user can READ from public bucket
	downloadResp := tc.NewRequest("GET", "/api/v1/storage/"+bucketName+"/"+fileName).
		Unauthenticated().
		Send()

	require.Equal(t, fiber.StatusOK, downloadResp.Status(),
		"Unauthenticated user should be able to read from public bucket")
	require.Equal(t, fileContent, downloadResp.Body(), "Content should match")
	t.Logf("✅ Unauthenticated user can download from public bucket")

	// Unauthenticated user CANNOT write to public bucket
	hackerFileName := "hacker.txt"
	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)
	part, err = writer.CreateFormFile("file", hackerFileName)
	require.NoError(t, err)
	_, err = part.Write([]byte("hack"))
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)

	uploadReq = httptest.NewRequest("POST", "/api/v1/storage/"+bucketName+"/"+hackerFileName, body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	// No authorization header

	uploadResp, err = tc.App.Test(uploadReq)
	require.NoError(t, err)
	require.Contains(t, []int{fiber.StatusUnauthorized, fiber.StatusForbidden},
		uploadResp.StatusCode, "Unauthenticated user should not be able to upload")
	t.Logf("✅ Unauthenticated user cannot upload to public bucket (status: %d)", uploadResp.StatusCode)

	// Unauthenticated user CANNOT delete from public bucket
	deleteResp := tc.NewRequest("DELETE", "/api/v1/storage/"+bucketName+"/"+fileName).
		Unauthenticated().
		Send()

	require.Contains(t, []int{fiber.StatusUnauthorized, fiber.StatusForbidden},
		deleteResp.Status(), "Unauthenticated user should not be able to delete")
	t.Logf("✅ Unauthenticated user cannot delete from public bucket (status: %d)", deleteResp.Status())

	// Toggle bucket to private using service key
	updateResp := tc.NewRequest("PUT", "/api/v1/storage/buckets/"+bucketName).
		WithServiceKey(serviceKey).
		WithBody(map[string]interface{}{
			"public": false,
		}).
		Send()

	require.Contains(t, []int{fiber.StatusOK, fiber.StatusNoContent},
		updateResp.Status(), "Update bucket to private should succeed")
	t.Logf("✅ Service key toggled bucket to private")

	// Unauthenticated user can no longer access
	download2Resp := tc.NewRequest("GET", "/api/v1/storage/"+bucketName+"/"+fileName).
		Unauthenticated().
		Send()

	require.Contains(t, []int{fiber.StatusNotFound, fiber.StatusForbidden, fiber.StatusUnauthorized},
		download2Resp.Status(), "Unauthenticated user should not access private bucket")
	t.Logf("✅ Unauthenticated user cannot access after bucket made private (status: %d)", download2Resp.Status())

	// Authenticated user can still access
	download3Resp := tc.NewRequest("GET", "/api/v1/storage/"+bucketName+"/"+fileName).
		WithAuth(userToken).
		Send()

	require.Equal(t, fiber.StatusOK, download3Resp.Status(),
		"File owner should still be able to access after bucket made private")
	require.Equal(t, fileContent, download3Resp.Body(), "Content should match")
	t.Logf("✅ Authenticated owner can still access after bucket made private")

	t.Logf("✅ Public bucket access control works correctly")
}
