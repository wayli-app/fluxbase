package api

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/fluxbase-eu/fluxbase/internal/storage"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

// setupStorageTestServer creates a test server with storage routes
func setupStorageTestServer(t *testing.T) (*fiber.App, string, *database.Connection) {
	t.Helper()

	// Skip integration tests when running with -short flag
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary directory for storage
	tempDir := t.TempDir()

	// Create storage configuration
	cfg := &config.StorageConfig{
		Provider:      "local",
		LocalPath:     tempDir,
		MaxUploadSize: 10 * 1024 * 1024, // 10MB
	}

	// Initialize storage service
	storageService, err := storage.NewService(cfg, "http://localhost:8080", "test-signing-secret")
	require.NoError(t, err)

	// Get database configuration from environment variables
	// Supports both FLUXBASE_DATABASE_* (used in CI) and DB_* (used locally)
	dbHost := os.Getenv("FLUXBASE_DATABASE_HOST")
	if dbHost == "" {
		dbHost = os.Getenv("DB_HOST")
	}
	if dbHost == "" {
		dbHost = "localhost" // Default for local development
	}

	dbUser := os.Getenv("FLUXBASE_DATABASE_USER")
	if dbUser == "" {
		dbUser = "fluxbase_app"
	}

	dbPassword := os.Getenv("FLUXBASE_DATABASE_PASSWORD")
	if dbPassword == "" {
		dbPassword = "fluxbase_app_password"
	}

	dbDatabase := os.Getenv("FLUXBASE_DATABASE_DATABASE")
	if dbDatabase == "" {
		dbDatabase = "fluxbase_test"
	}

	// Create minimal database configuration for testing
	dbConfig := config.DatabaseConfig{
		Host:            dbHost,
		Port:            5432,
		User:            dbUser,
		Password:        dbPassword,
		Database:        dbDatabase,
		SSLMode:         "disable",
		MaxConnections:  5,
		MinConnections:  1,
		MaxConnLifetime: 5 * time.Minute,
		MaxConnIdleTime: 5 * time.Minute,
		HealthCheck:     30 * time.Second,
	}

	// Connect to database
	db, err := database.NewConnection(dbConfig)
	require.NoError(t, err)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = db.Health(ctx)
	require.NoError(t, err)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Setup storage routes
	storageHandler := NewStorageHandler(storageService, db, nil)
	api := app.Group("/api/v1")
	storageRoutes := api.Group("/storage")

	// Bucket management
	storageRoutes.Get("/buckets", storageHandler.ListBuckets)
	storageRoutes.Post("/buckets/:bucket", storageHandler.CreateBucket)
	storageRoutes.Delete("/buckets/:bucket", storageHandler.DeleteBucket)

	// File operations
	storageRoutes.Post("/:bucket/*", storageHandler.UploadFile)
	storageRoutes.Get("/:bucket/*", storageHandler.DownloadFile)
	storageRoutes.Delete("/:bucket/*", storageHandler.DeleteFile)
	storageRoutes.Head("/:bucket/*", storageHandler.GetFileInfo)
	storageRoutes.Get("/:bucket", storageHandler.ListFiles)

	// Advanced features
	storageRoutes.Post("/:bucket/multipart", storageHandler.MultipartUpload)
	storageRoutes.Post("/:bucket/*/signed-url", storageHandler.GenerateSignedURL)

	return app, tempDir, db
}

// createTestBucket is a helper to create a bucket for tests
func createTestBucket(t *testing.T, app *fiber.App, bucketName string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/buckets/"+bucketName, nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
}

// uploadTestFile is a helper to upload a file for tests
func uploadTestFile(t *testing.T, app *fiber.App, bucket, path, content string) {
	t.Helper()
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", path)
	require.NoError(t, err)
	_, err = part.Write([]byte(content))
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/"+bucket+"/"+path, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := app.Test(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
}
