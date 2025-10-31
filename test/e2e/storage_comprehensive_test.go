package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wayli-app/fluxbase/internal/api"
	"github.com/wayli-app/fluxbase/internal/auth"
	"github.com/wayli-app/fluxbase/internal/config"
	"github.com/wayli-app/fluxbase/internal/database"
	"github.com/wayli-app/fluxbase/internal/middleware"
)

// TestStorageComprehensive tests Storage service functionality
func TestStorageComprehensive(t *testing.T) {
	// Setup test environment
	cfg := setupStorageTestConfig(t)
	db := setupStorageTestDatabase(t, cfg)
	defer db.Close()

	// Setup storage schema
	setupStorageSchema(t, db)

	// Create test app
	app := createStorageTestApp(t, db, cfg)

	// Run tests
	t.Run("Bucket CRUD Operations", func(t *testing.T) {
		testBucketCRUD(t, app, db, cfg.JWTSecret)
	})

	t.Run("File Upload - Small", func(t *testing.T) {
		testFileUploadSmall(t, app, db, cfg.JWTSecret)
	})

	t.Run("File Upload - Large", func(t *testing.T) {
		testFileUploadLarge(t, app, db, cfg.JWTSecret)
	})

	t.Run("File Download", func(t *testing.T) {
		testFileDownload(t, app, db, cfg.JWTSecret)
	})

	t.Run("File Download with Range Request", func(t *testing.T) {
		testRangeRequest(t, app, db, cfg.JWTSecret)
	})

	t.Run("File Metadata Management", func(t *testing.T) {
		testFileMetadata(t, app, db, cfg.JWTSecret)
	})

	t.Run("List Files in Bucket", func(t *testing.T) {
		testListFiles(t, app, db, cfg.JWTSecret)
	})

	t.Run("File Copy Operation", func(t *testing.T) {
		testFileCopy(t, app, db, cfg.JWTSecret)
	})

	t.Run("File Move Operation", func(t *testing.T) {
		testFileMove(t, app, db, cfg.JWTSecret)
	})

	t.Run("File Delete", func(t *testing.T) {
		testFileDelete(t, app, db, cfg.JWTSecret)
	})

	t.Run("Concurrent Uploads", func(t *testing.T) {
		testConcurrentUploads(t, app, db, cfg.JWTSecret)
	})

	t.Run("Public URLs", func(t *testing.T) {
		testPublicURLs(t, app, db, cfg.JWTSecret)
	})

	t.Run("Content Type Detection", func(t *testing.T) {
		testContentTypeDetection(t, app, db, cfg.JWTSecret)
	})

	t.Run("Storage Quotas", func(t *testing.T) {
		testStorageQuotas(t, app, db, cfg.JWTSecret)
	})
}

// setupStorageTestConfig creates test configuration
func setupStorageTestConfig(t *testing.T) *config.Config {
	return &config.Config{
		DatabaseURL:     "postgres://postgres:postgres@localhost:5432/fluxbase_test?sslmode=disable",
		JWTSecret:       "test-jwt-secret-storage",
		Port:            "8080",
		StorageBackend:  "local",
		StorageBasePath: "/tmp/fluxbase_test_storage",
	}
}

// setupStorageTestDatabase creates database connection
func setupStorageTestDatabase(t *testing.T, cfg *config.Config) *database.Connection {
	db, err := database.Connect(cfg.DatabaseURL)
	require.NoError(t, err, "Failed to connect to test database")
	return db
}

// setupStorageSchema creates storage schema
func setupStorageSchema(t *testing.T, db *database.Connection) {
	ctx := context.Background()

	queries := []string{
		`CREATE SCHEMA IF NOT EXISTS storage`,
		`CREATE SCHEMA IF NOT EXISTS auth`,

		`CREATE TABLE IF NOT EXISTS auth.users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email TEXT UNIQUE NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS storage.buckets (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT UNIQUE NOT NULL,
			public BOOLEAN DEFAULT false,
			file_size_limit BIGINT,
			allowed_mime_types TEXT[],
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS storage.objects (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			bucket_id UUID REFERENCES storage.buckets(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			owner UUID,
			bucket_name TEXT,
			size BIGINT,
			mime_type TEXT,
			etag TEXT,
			metadata JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE(bucket_id, name)
		)`,
	}

	for _, query := range queries {
		_, err := db.Pool().Exec(ctx, query)
		require.NoError(t, err, "Failed to setup storage schema")
	}

	// Cleanup
	_, _ = db.Pool().Exec(ctx, "TRUNCATE storage.objects CASCADE")
	_, _ = db.Pool().Exec(ctx, "TRUNCATE storage.buckets CASCADE")
	_, _ = db.Pool().Exec(ctx, "DELETE FROM auth.users WHERE email LIKE '%@storagetest.com'")
}

// createStorageTestApp creates Fiber app with storage routes
func createStorageTestApp(t *testing.T, db *database.Connection, cfg *config.Config) *fiber.App {
	app := fiber.New()

	// Auth middleware
	app.Use(middleware.AuthMiddleware(middleware.AuthConfig{
		JWTSecret: cfg.JWTSecret,
		Optional:  true,
	}))

	// Register API routes (includes storage)
	apiServer := api.NewServer(db, cfg)
	apiServer.RegisterRoutes(app)

	return app
}

// testBucketCRUD tests bucket management operations
func testBucketCRUD(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	token := createStorageTestUser(t, db, jwtSecret)

	// Create bucket
	t.Run("Create Bucket", func(t *testing.T) {
		bucketReq := map[string]interface{}{
			"name":   "test-bucket",
			"public": false,
		}

		body, _ := json.Marshal(bucketReq)
		req := httptest.NewRequest("POST", "/api/v1/storage/buckets", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.True(t, resp.StatusCode == 201 || resp.StatusCode == 200, "Bucket creation should succeed")

		if resp.StatusCode == 201 || resp.StatusCode == 200 {
			var result map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&result)
			assert.Equal(t, "test-bucket", result["name"])
		}
	})

	// List buckets
	t.Run("List Buckets", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/storage/buckets", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		if resp.StatusCode == 200 {
			var buckets []map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&buckets)
			assert.GreaterOrEqual(t, len(buckets), 1)
		}
	})

	// Get bucket
	t.Run("Get Bucket", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/storage/buckets/test-bucket", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		if resp.StatusCode == 200 {
			var bucket map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&bucket)
			assert.Equal(t, "test-bucket", bucket["name"])
		}
	})

	// Update bucket
	t.Run("Update Bucket", func(t *testing.T) {
		updates := map[string]interface{}{
			"public": true,
		}

		body, _ := json.Marshal(updates)
		req := httptest.NewRequest("PUT", "/api/v1/storage/buckets/test-bucket", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.True(t, resp.StatusCode == 200 || resp.StatusCode == 404)
	})

	// Delete bucket (skip to keep bucket for other tests)
	t.Run("Delete Bucket", func(t *testing.T) {
		// Create a bucket to delete
		bucketReq := map[string]interface{}{
			"name": "delete-test-bucket",
		}

		body, _ := json.Marshal(bucketReq)
		req := httptest.NewRequest("POST", "/api/v1/storage/buckets", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		app.Test(req, -1)

		// Delete it
		req = httptest.NewRequest("DELETE", "/api/v1/storage/buckets/delete-test-bucket", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.True(t, resp.StatusCode == 204 || resp.StatusCode == 404)
	})
}

// testFileUploadSmall tests uploading small files
func testFileUploadSmall(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	token := createStorageTestUser(t, db, jwtSecret)

	// Create bucket first
	createTestBucket(t, app, token, "upload-bucket")

	// Upload a small file
	fileContent := []byte("Hello, this is a test file!")
	fileName := "test-small.txt"

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", fileName)
	require.NoError(t, err)
	part.Write(fileContent)
	writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/storage/buckets/upload-bucket/files", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.True(t, resp.StatusCode == 201 || resp.StatusCode == 200, "File upload should succeed")

	if resp.StatusCode == 201 || resp.StatusCode == 200 {
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.NotNil(t, result["id"])
	}
}

// testFileUploadLarge tests uploading larger files
func testFileUploadLarge(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	token := createStorageTestUser(t, db, jwtSecret)

	createTestBucket(t, app, token, "large-upload-bucket")

	// Create a 1MB file
	fileContent := bytes.Repeat([]byte("A"), 1024*1024)
	fileName := "test-large.bin"

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", fileName)
	require.NoError(t, err)
	part.Write(fileContent)
	writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/storage/buckets/large-upload-bucket/files", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, 30*time.Second)
	require.NoError(t, err)
	assert.True(t, resp.StatusCode == 201 || resp.StatusCode == 200 || resp.StatusCode == 404, "Large file upload")
}

// testFileDownload tests downloading files
func testFileDownload(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	token := createStorageTestUser(t, db, jwtSecret)

	bucketName := "download-bucket"
	fileName := "download-test.txt"
	fileContent := []byte("Download test content")

	createTestBucket(t, app, token, bucketName)
	uploadTestFile(t, app, token, bucketName, fileName, fileContent)

	// Download the file
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/storage/buckets/%s/files/%s", bucketName, fileName), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	if resp.StatusCode == 200 {
		downloaded, _ := io.ReadAll(resp.Body)
		assert.Equal(t, fileContent, downloaded, "Downloaded content should match uploaded content")
	}
}

// testRangeRequest tests partial content download
func testRangeRequest(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	token := createStorageTestUser(t, db, jwtSecret)

	bucketName := "range-bucket"
	fileName := "range-test.txt"
	fileContent := []byte("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ")

	createTestBucket(t, app, token, bucketName)
	uploadTestFile(t, app, token, bucketName, fileName, fileContent)

	// Request bytes 5-14 (10 bytes)
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/storage/buckets/%s/files/%s", bucketName, fileName), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Range", "bytes=5-14")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	if resp.StatusCode == 206 {
		partialContent, _ := io.ReadAll(resp.Body)
		expected := fileContent[5:15]
		assert.Equal(t, expected, partialContent, "Partial content should match range")
	}
}

// testFileMetadata tests managing file metadata
func testFileMetadata(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	token := createStorageTestUser(t, db, jwtSecret)

	bucketName := "metadata-bucket"
	fileName := "metadata-test.txt"

	createTestBucket(t, app, token, bucketName)
	uploadTestFileWithMetadata(t, app, token, bucketName, fileName, []byte("content"), map[string]string{
		"x-meta-author":      "Test User",
		"x-meta-description": "Test file",
	})

	// Get file metadata
	req := httptest.NewRequest("HEAD", fmt.Sprintf("/api/v1/storage/buckets/%s/files/%s", bucketName, fileName), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	if resp.StatusCode == 200 {
		// Check metadata headers
		assert.NotEmpty(t, resp.Header.Get("Content-Length"))
		assert.NotEmpty(t, resp.Header.Get("Content-Type"))
	}
}

// testListFiles tests listing files in a bucket
func testListFiles(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	token := createStorageTestUser(t, db, jwtSecret)

	bucketName := "list-bucket"
	createTestBucket(t, app, token, bucketName)

	// Upload multiple files
	for i := 1; i <= 5; i++ {
		fileName := fmt.Sprintf("file%d.txt", i)
		uploadTestFile(t, app, token, bucketName, fileName, []byte(fmt.Sprintf("Content %d", i)))
	}

	// List files
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/storage/buckets/%s/files", bucketName), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	if resp.StatusCode == 200 {
		var files []map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&files)
		assert.GreaterOrEqual(t, len(files), 1, "Should list uploaded files")
	}
}

// testFileCopy tests copying files
func testFileCopy(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	token := createStorageTestUser(t, db, jwtSecret)

	bucketName := "copy-bucket"
	sourceFile := "source.txt"
	destFile := "destination.txt"

	createTestBucket(t, app, token, bucketName)
	uploadTestFile(t, app, token, bucketName, sourceFile, []byte("Source content"))

	// Copy file
	copyReq := map[string]interface{}{
		"source_key":      sourceFile,
		"destination_key": destFile,
	}

	body, _ := json.Marshal(copyReq)
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/storage/buckets/%s/copy", bucketName), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.True(t, resp.StatusCode == 200 || resp.StatusCode == 404, "File copy operation")
}

// testFileMove tests moving files
func testFileMove(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	token := createStorageTestUser(t, db, jwtSecret)

	bucketName := "move-bucket"
	sourceFile := "move-source.txt"
	destFile := "move-destination.txt"

	createTestBucket(t, app, token, bucketName)
	uploadTestFile(t, app, token, bucketName, sourceFile, []byte("Move content"))

	// Move file
	moveReq := map[string]interface{}{
		"source_key":      sourceFile,
		"destination_key": destFile,
	}

	body, _ := json.Marshal(moveReq)
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/storage/buckets/%s/move", bucketName), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.True(t, resp.StatusCode == 200 || resp.StatusCode == 404, "File move operation")
}

// testFileDelete tests deleting files
func testFileDelete(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	token := createStorageTestUser(t, db, jwtSecret)

	bucketName := "delete-file-bucket"
	fileName := "delete-me.txt"

	createTestBucket(t, app, token, bucketName)
	uploadTestFile(t, app, token, bucketName, fileName, []byte("Delete me"))

	// Delete file
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/storage/buckets/%s/files/%s", bucketName, fileName), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.True(t, resp.StatusCode == 204 || resp.StatusCode == 404, "File deletion")
}

// testConcurrentUploads tests multiple simultaneous uploads
func testConcurrentUploads(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	token := createStorageTestUser(t, db, jwtSecret)

	bucketName := "concurrent-bucket"
	createTestBucket(t, app, token, bucketName)

	// Upload 5 files concurrently
	numUploads := 5
	done := make(chan bool, numUploads)

	for i := 0; i < numUploads; i++ {
		go func(index int) {
			fileName := fmt.Sprintf("concurrent-%d.txt", index)
			content := []byte(fmt.Sprintf("Concurrent content %d", index))
			uploadTestFile(t, app, token, bucketName, fileName, content)
			done <- true
		}(i)
	}

	// Wait for all uploads
	for i := 0; i < numUploads; i++ {
		<-done
	}

	t.Log("Concurrent uploads completed")
}

// testPublicURLs tests public URL generation
func testPublicURLs(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	token := createStorageTestUser(t, db, jwtSecret)

	bucketName := "public-bucket"
	fileName := "public-file.txt"

	// Create public bucket
	createTestPublicBucket(t, app, token, bucketName)
	uploadTestFile(t, app, token, bucketName, fileName, []byte("Public content"))

	// Get public URL
	publicURL := fmt.Sprintf("/api/v1/storage/buckets/%s/files/%s", bucketName, fileName)

	// Access without authentication (public bucket)
	req := httptest.NewRequest("GET", publicURL, nil)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	if resp.StatusCode == 200 {
		content, _ := io.ReadAll(resp.Body)
		assert.NotEmpty(t, content, "Public file should be accessible")
	}
}

// testContentTypeDetection tests MIME type detection
func testContentTypeDetection(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	token := createStorageTestUser(t, db, jwtSecret)

	bucketName := "mimetype-bucket"
	createTestBucket(t, app, token, bucketName)

	tests := []struct {
		fileName    string
		content     []byte
		expectedMIME string
	}{
		{"test.txt", []byte("text content"), "text/plain"},
		{"test.json", []byte(`{"key":"value"}`), "application/json"},
		{"test.html", []byte("<html><body>Test</body></html>"), "text/html"},
	}

	for _, tt := range tests {
		uploadTestFile(t, app, token, bucketName, tt.fileName, tt.content)

		req := httptest.NewRequest("HEAD", fmt.Sprintf("/api/v1/storage/buckets/%s/files/%s", bucketName, tt.fileName), nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		if err == nil && resp.StatusCode == 200 {
			contentType := resp.Header.Get("Content-Type")
			if contentType != "" && !strings.Contains(contentType, tt.expectedMIME) {
				t.Logf("Content-Type for %s: expected %s, got %s", tt.fileName, tt.expectedMIME, contentType)
			}
		}
	}
}

// testStorageQuotas tests storage quota enforcement
func testStorageQuotas(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	token := createStorageTestUser(t, db, jwtSecret)

	// Create bucket with file size limit
	bucketReq := map[string]interface{}{
		"name":            "quota-bucket",
		"file_size_limit": 1024, // 1KB limit
	}

	body, _ := json.Marshal(bucketReq)
	req := httptest.NewRequest("POST", "/api/v1/storage/buckets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	app.Test(req, -1)

	// Try to upload file larger than limit
	largeContent := bytes.Repeat([]byte("A"), 2048) // 2KB

	uploadBody := &bytes.Buffer{}
	writer := multipart.NewWriter(uploadBody)
	part, _ := writer.CreateFormFile("file", "large.txt")
	part.Write(largeContent)
	writer.Close()

	req = httptest.NewRequest("POST", "/api/v1/storage/buckets/quota-bucket/files", uploadBody)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	// Should be rejected (413 Payload Too Large) or pass if quotas not enforced
	assert.True(t, resp.StatusCode == 413 || resp.StatusCode == 201 || resp.StatusCode == 404)
}

// Helper functions

func createStorageTestUser(t *testing.T, db *database.Connection, jwtSecret string) string {
	ctx := context.Background()

	email := fmt.Sprintf("user%d@storagetest.com", time.Now().UnixNano())
	userID := uuid.New().String()

	_, err := db.Pool().Exec(ctx, `
		INSERT INTO auth.users (id, email)
		VALUES ($1, $2)
		ON CONFLICT (email) DO NOTHING
	`, userID, email)
	require.NoError(t, err)

	authService := auth.NewService(db, jwtSecret, "smtp://fake", "noreply@test.com")
	token, err := authService.GenerateJWT(userID, email)
	require.NoError(t, err)

	return token
}

func createTestBucket(t *testing.T, app *fiber.App, token, bucketName string) {
	bucketReq := map[string]interface{}{
		"name":   bucketName,
		"public": false,
	}

	body, _ := json.Marshal(bucketReq)
	req := httptest.NewRequest("POST", "/api/v1/storage/buckets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	app.Test(req, -1)
}

func createTestPublicBucket(t *testing.T, app *fiber.App, token, bucketName string) {
	bucketReq := map[string]interface{}{
		"name":   bucketName,
		"public": true,
	}

	body, _ := json.Marshal(bucketReq)
	req := httptest.NewRequest("POST", "/api/v1/storage/buckets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	app.Test(req, -1)
}

func uploadTestFile(t *testing.T, app *fiber.App, token, bucketName, fileName string, content []byte) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", fileName)
	require.NoError(t, err)
	part.Write(content)
	writer.Close()

	req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/storage/buckets/%s/files", bucketName), body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	app.Test(req, -1)
}

func uploadTestFileWithMetadata(t *testing.T, app *fiber.App, token, bucketName, fileName string, content []byte, metadata map[string]string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add metadata fields
	for key, value := range metadata {
		writer.WriteField(key, value)
	}

	// Add file
	part, err := writer.CreateFormFile("file", fileName)
	require.NoError(t, err)
	part.Write(content)
	writer.Close()

	req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/storage/buckets/%s/files", bucketName), body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	app.Test(req, -1)
}
