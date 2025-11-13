package e2e

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
	"github.com/wayli-app/fluxbase/test"
)

// TestStorageRLS_StorageProviders verifies RLS works with both local and S3
func TestStorageRLS_StorageProviders(t *testing.T) {
	// Determine which providers to test
	providers := []string{"local"}

	// Only test S3 if environment is configured
	if os.Getenv("FLUXBASE_STORAGE_S3_BUCKET") != "" || os.Getenv("S3_BUCKET") != "" {
		providers = append(providers, "s3")
	}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			tc := test.NewRLSTestContext(t)
			defer tc.Close()

			// Configure storage provider
			tc.Config.Storage.Provider = provider
			if provider == "local" {
				tc.Config.Storage.LocalPath = "/tmp/fluxbase-test-storage-rls"
			} else {
				// S3 config already set in test config
				tc.Config.Storage.S3Bucket = "fluxbase-test"
			}

			// Clean up storage for this test
			tc.CleanupStorageFiles()

			// Create two users
			user1Email := "user1-" + test.RandomEmail()
			user1ID, user1Token := tc.CreateTestUser(user1Email, "password123")

			user2Email := "user2-" + test.RandomEmail()
			user2ID, user2Token := tc.CreateTestUser(user2Email, "password123")

			// Create service key and bucket
			serviceKey := tc.CreateServiceKey("test-bucket-creation")

			bucketName := "test-bucket"
			tc.NewRequest("POST", "/api/v1/storage/buckets/"+bucketName).
				WithServiceKey(serviceKey).
				Send().
				AssertStatus(fiber.StatusCreated)

			t.Logf("✅ [%s] Created bucket", provider)

			// User1 uploads file
			fileName := "file.txt"
			fileContent := []byte("test data for " + provider)

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

			uploadResp, err := tc.App.Test(uploadReq, 15000) // 15 second timeout for S3
			require.NoError(t, err)
			require.Equal(t, fiber.StatusCreated, uploadResp.StatusCode,
				"Upload should succeed with %s provider", provider)

			t.Logf("✅ [%s] User1 uploaded file", provider)

			// User1 can access
			downloadResp := tc.NewRequest("GET", "/api/v1/storage/"+bucketName+"/"+fileName).
				WithAuth(user1Token).
				Send()

			require.Equal(t, fiber.StatusOK, downloadResp.Status(),
				"User1 should be able to download with %s provider", provider)
			require.Equal(t, fileContent, downloadResp.Body(), "Content should match")

			t.Logf("✅ [%s] User1 can download their file", provider)

			// User2 cannot access
			download2Resp := tc.NewRequest("GET", "/api/v1/storage/"+bucketName+"/"+fileName).
				WithAuth(user2Token).
				Send()

			require.Contains(t, []int{fiber.StatusNotFound, fiber.StatusForbidden},
				download2Resp.Status(),
				"User2 should not be able to access user1's file with %s provider", provider)

			t.Logf("✅ [%s] User2 cannot access user1's file (status: %d)", provider, download2Resp.Status())

			// Verify metadata is in database
			results := tc.QuerySQLAsSuperuser(
				"SELECT owner_id, bucket_id, path FROM storage.objects WHERE path = $1 AND bucket_id = $2",
				fileName, bucketName)

			require.Len(t, results, 1, "File metadata should exist in database")
			require.Equal(t, user1ID, results[0]["owner_id"], "owner_id should be set to user1")
			require.Equal(t, bucketName, results[0]["bucket_id"], "bucket_id should match")
			require.Equal(t, fileName, results[0]["path"], "path should match")

			t.Logf("✅ [%s] File metadata correctly stored in database with owner_id", provider)

			// Test RLS at database level
			user2Results := tc.QuerySQLAsRLSUser(
				"SELECT COUNT(*) as count FROM storage.objects WHERE bucket_id = $1",
				user2ID, // Query as user2
				bucketName)

			require.Len(t, user2Results, 1, "Query should return result")
			count, ok := user2Results[0]["count"].(int64)
			require.True(t, ok, "Count should be int64")
			require.Equal(t, int64(0), count,
				"RLS should hide user1's file from user2 at database level with %s provider", provider)

			t.Logf("✅ [%s] RLS correctly filters at database level", provider)

			// User1 can delete their file
			deleteResp := tc.NewRequest("DELETE", "/api/v1/storage/"+bucketName+"/"+fileName).
				WithAuth(user1Token).
				Send()

			require.Contains(t, []int{fiber.StatusOK, fiber.StatusNoContent},
				deleteResp.Status(), "User1 should be able to delete their file with %s provider", provider)

			t.Logf("✅ [%s] User1 can delete their file", provider)

			// Verify file is deleted from database
			results = tc.QuerySQLAsSuperuser(
				"SELECT COUNT(*) as count FROM storage.objects WHERE path = $1 AND bucket_id = $2",
				fileName, bucketName)

			require.Len(t, results, 1, "Query should return result")
			count, ok = results[0]["count"].(int64)
			require.True(t, ok, "Count should be int64")
			require.Equal(t, int64(0), count, "File should be deleted from database")

			t.Logf("✅ [%s] All RLS tests passed with this provider", provider)
		})
	}

	t.Logf("✅ RLS works correctly across all tested storage providers")
}
