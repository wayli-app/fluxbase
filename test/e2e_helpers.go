package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
	"github.com/wayli-app/fluxbase/internal/api"
	"github.com/wayli-app/fluxbase/internal/config"
	"github.com/wayli-app/fluxbase/internal/database"
)

// TestContext holds all testing dependencies
type TestContext struct {
	DB     *database.Connection
	Server *api.Server
	App    *fiber.App
	Config *config.Config
	T      *testing.T
}

// NewTestContext creates a new test context with database and server
func NewTestContext(t *testing.T) *TestContext {
	cfg := GetTestConfig()

	// Connect to test database
	db, err := database.NewConnection(cfg.Database)
	require.NoError(t, err, "Failed to connect to test database")

	// Ensure database is healthy
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.Health(ctx)
	require.NoError(t, err, "Database is not healthy")

	// Run migrations BEFORE creating server so REST API can discover tables
	err = db.Migrate()
	require.NoError(t, err, "Failed to run migrations")

	// Create server (REST API will now see all migrated tables)
	server := api.NewServer(cfg, db)

	return &TestContext{
		DB:     db,
		Server: server,
		App:    server.App(),
		Config: cfg,
		T:      t,
	}
}

// Close cleans up test context resources
func (tc *TestContext) Close() {
	// Shutdown the server first to stop all background goroutines
	if tc.Server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tc.Server.Shutdown(ctx)
	}

	// Then close the database connection
	if tc.DB != nil {
		tc.DB.Close()
	}
}

// CleanDatabase truncates all tables in the test database
func (tc *TestContext) CleanDatabase() {
	ctx := context.Background()

	// Get all tables
	tables, err := tc.DB.Inspector().GetAllTables(ctx, "public")
	require.NoError(tc.T, err)

	// Truncate each table
	for _, table := range tables {
		_, err := tc.DB.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table.Name))
		require.NoError(tc.T, err)
	}
}

// CreateTestTable creates a table for testing
func (tc *TestContext) CreateTestTable(tableName, schema string) {
	ctx := context.Background()
	_, err := tc.DB.Exec(ctx, schema)
	require.NoError(tc.T, err)
}

// DropTestTable drops a test table
func (tc *TestContext) DropTestTable(tableName string) {
	ctx := context.Background()
	_, err := tc.DB.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", tableName))
	require.NoError(tc.T, err)
}

// getEnvOrDefault gets an environment variable or returns a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetTestConfig returns a test configuration
func GetTestConfig() *config.Config {
	// Allow environment variables to override defaults for CI
	dbHost := getEnvOrDefault("FLUXBASE_DATABASE_HOST", "postgres")
	smtpHost := getEnvOrDefault("FLUXBASE_EMAIL_SMTP_HOST", "mailhog")
	s3Endpoint := getEnvOrDefault("FLUXBASE_STORAGE_S3_ENDPOINT", "minio:9000")

	return &config.Config{
		Server: config.ServerConfig{
			Address:      ":8081",
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
			BodyLimit:    10 * 1024 * 1024,
		},
		Database: config.DatabaseConfig{
			Host:            dbHost,
			Port:            5432,
			User:            "postgres",
			Password:        "postgres",
			Database:        "fluxbase_test",
			SSLMode:         "disable",
			MaxConnections:  10,
			MinConnections:  2,
			MaxConnLifetime: 1 * time.Hour,
			MaxConnIdleTime: 30 * time.Minute,
			HealthCheck:     1 * time.Minute,
		},
		Auth: config.AuthConfig{
			JWTSecret:       "test-secret-key-for-testing-only",
			JWTExpiry:       15 * time.Minute,
			RefreshExpiry:   7 * 24 * time.Hour,
			PasswordMinLen:  8,
			BcryptCost:      10,
			EnableSignup:    true,
			EnableMagicLink: true,
		},
		Realtime: config.RealtimeConfig{
			Enabled:        false, // Disabled for most tests
			MaxConnections: 1000,
			PingInterval:   30 * time.Second,
			PongTimeout:    10 * time.Second,
		},
		Storage: config.StorageConfig{
			Provider:      "local",
			LocalPath:     "/tmp/fluxbase-test-storage",
			MaxUploadSize: 100 * 1024 * 1024, // 100MB
			S3Endpoint:    s3Endpoint,
			S3AccessKey:   "minioadmin",
			S3SecretKey:   "minioadmin",
			S3Bucket:      "fluxbase-test",
			S3Region:      "us-east-1",
		},
		Email: config.EmailConfig{
			Enabled:     true,
			SMTPHost:    smtpHost,
			SMTPPort:    1025,
			FromAddress: "test@fluxbase.test",
			FromName:    "Fluxbase Test",
		},
		Debug: true,
	}
}

// APIRequest is a helper for making HTTP requests to the test server
type APIRequest struct {
	tc      *TestContext
	method  string
	path    string
	body    interface{}
	headers map[string]string
}

// NewAPIRequest creates a new API request builder
func (tc *TestContext) NewRequest(method, path string) *APIRequest {
	return &APIRequest{
		tc:      tc,
		method:  method,
		path:    path,
		headers: make(map[string]string),
	}
}

// WithBody sets the request body
func (r *APIRequest) WithBody(body interface{}) *APIRequest {
	r.body = body
	return r
}

// WithHeader sets a request header
func (r *APIRequest) WithHeader(key, value string) *APIRequest {
	r.headers[key] = value
	return r
}

// WithAuth sets the Authorization header
func (r *APIRequest) WithAuth(token string) *APIRequest {
	r.headers["Authorization"] = "Bearer " + token
	return r
}

// Send executes the request and returns the response
func (r *APIRequest) Send() *APIResponse {
	var bodyReader io.Reader

	if r.body != nil {
		bodyBytes, err := json.Marshal(r.body)
		require.NoError(r.tc.T, err)
		bodyReader = bytes.NewReader(bodyBytes)

		// Auto-set content type if not already set
		if _, ok := r.headers["Content-Type"]; !ok {
			r.headers["Content-Type"] = "application/json"
		}
	}

	req := httptest.NewRequest(r.method, r.path, bodyReader)

	// Set headers
	for key, value := range r.headers {
		req.Header.Set(key, value)
	}

	// Execute request with 5 second timeout
	resp, err := r.tc.App.Test(req, 5000) // 5 second timeout
	require.NoError(r.tc.T, err)

	return &APIResponse{
		tc:       r.tc,
		Response: resp,
	}
}

// APIResponse wraps an HTTP response with helper methods
type APIResponse struct {
	tc       *TestContext
	Response *http.Response
	body     []byte // cached body
}

// Status returns the HTTP status code
func (r *APIResponse) Status() int {
	return r.Response.StatusCode
}

// Body returns the response body as bytes
func (r *APIResponse) Body() []byte {
	if r.body == nil {
		body, err := io.ReadAll(r.Response.Body)
		require.NoError(r.tc.T, err)
		r.body = body
	}
	return r.body
}

// JSON unmarshals the response body into the provided interface
func (r *APIResponse) JSON(v interface{}) {
	err := json.Unmarshal(r.Body(), v)
	require.NoError(r.tc.T, err)
}

// AssertStatus asserts the HTTP status code
func (r *APIResponse) AssertStatus(expectedStatus int) *APIResponse {
	require.Equal(r.tc.T, expectedStatus, r.Status(),
		"Expected status %d but got %d. Body: %s", expectedStatus, r.Status(), string(r.Body()))
	return r
}

// AssertJSON asserts that the response contains JSON matching the expected value
func (r *APIResponse) AssertJSON(expected interface{}) *APIResponse {
	var actual interface{}
	r.JSON(&actual)
	require.Equal(r.tc.T, expected, actual)
	return r
}

// AssertContains asserts that the response body contains the given substring
func (r *APIResponse) AssertContains(substring string) *APIResponse {
	require.Contains(r.tc.T, string(r.Body()), substring)
	return r
}

// Header returns a response header value
func (r *APIResponse) Header(key string) string {
	return r.Response.Header.Get(key)
}

// AssertHeader asserts that a response header has the expected value
func (r *APIResponse) AssertHeader(key, expectedValue string) *APIResponse {
	require.Equal(r.tc.T, expectedValue, r.Header(key))
	return r
}

// TestDataBuilder helps create test data easily
type TestDataBuilder struct {
	tc        *TestContext
	tableName string
	rows      []map[string]interface{}
}

// NewTestData creates a new test data builder
func (tc *TestContext) NewTestData(tableName string) *TestDataBuilder {
	return &TestDataBuilder{
		tc:        tc,
		tableName: tableName,
		rows:      make([]map[string]interface{}, 0),
	}
}

// Row adds a row to be inserted
func (b *TestDataBuilder) Row(data map[string]interface{}) *TestDataBuilder {
	b.rows = append(b.rows, data)
	return b
}

// Insert inserts all rows into the database
func (b *TestDataBuilder) Insert() {
	for _, row := range b.rows {
		body, _ := json.Marshal(row)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/rest/%s", b.tableName), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := b.tc.App.Test(req, -1)
		require.NoError(b.tc.T, err)
		require.Equal(b.tc.T, fiber.StatusCreated, resp.StatusCode,
			"Failed to insert test data into %s", b.tableName)
	}
}

// WaitForCondition waits for a condition to be true with timeout
func (tc *TestContext) WaitForCondition(timeout time.Duration, checkInterval time.Duration, condition func() bool) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(checkInterval)
	}

	return false
}

// ExecuteSQL executes raw SQL for test setup
func (tc *TestContext) ExecuteSQL(sql string, args ...interface{}) {
	ctx := context.Background()
	_, err := tc.DB.Exec(ctx, sql, args...)
	require.NoError(tc.T, err)
}

// QuerySQL executes a SQL query and returns results
func (tc *TestContext) QuerySQL(sql string, args ...interface{}) []map[string]interface{} {
	ctx := context.Background()
	rows, err := tc.DB.Query(ctx, sql, args...)
	require.NoError(tc.T, err)
	defer rows.Close()

	results := make([]map[string]interface{}, 0)

	for rows.Next() {
		values, err := rows.Values()
		require.NoError(tc.T, err)

		row := make(map[string]interface{})
		for i, col := range rows.FieldDescriptions() {
			row[string(col.Name)] = values[i]
		}

		results = append(results, row)
	}

	return results
}

// RunMigrations runs database migrations
func (tc *TestContext) RunMigrations() {
	err := tc.DB.Migrate()
	require.NoError(tc.T, err, "Failed to run migrations")
}

// CreateTestUser creates a test user with email and password, returns userID and JWT token
func (tc *TestContext) CreateTestUser(email, password string) (userID, token string) {
	// First, signup the user
	signupResp := tc.NewRequest("POST", "/api/v1/auth/signup").
		WithBody(map[string]interface{}{
			"email":    email,
			"password": password,
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	var signupResult map[string]interface{}
	signupResp.JSON(&signupResult)

	// Extract user ID and token
	if user, ok := signupResult["user"].(map[string]interface{}); ok {
		if id, ok := user["id"].(string); ok {
			userID = id
		}
	}

	if accessToken, ok := signupResult["access_token"].(string); ok {
		token = accessToken
	}

	require.NotEmpty(tc.T, userID, "User ID not returned from signup")
	require.NotEmpty(tc.T, token, "Access token not returned from signup")

	return userID, token
}

// GetAuthToken signs in with email/password and returns JWT token
func (tc *TestContext) GetAuthToken(email, password string) string {
	resp := tc.NewRequest("POST", "/api/v1/auth/signin").
		WithBody(map[string]interface{}{
			"email":    email,
			"password": password,
		}).
		Send().
		AssertStatus(fiber.StatusOK)

	var result map[string]interface{}
	resp.JSON(&result)

	token, ok := result["access_token"].(string)
	require.True(tc.T, ok, "access_token not found in signin response")
	require.NotEmpty(tc.T, token, "access_token is empty")

	return token
}

// EnsureAuthSchema ensures auth schema and tables exist
func (tc *TestContext) EnsureAuthSchema() {
	ctx := context.Background()

	queries := []string{
		`CREATE SCHEMA IF NOT EXISTS auth`,
		`CREATE TABLE IF NOT EXISTS auth.users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email TEXT UNIQUE NOT NULL,
			encrypted_password TEXT,
			email_confirmed_at TIMESTAMPTZ,
			confirmation_token TEXT,
			confirmation_sent_at TIMESTAMPTZ,
			recovery_token TEXT,
			recovery_sent_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS auth.sessions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
			refresh_token TEXT NOT NULL,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
	}

	for _, query := range queries {
		_, err := tc.DB.Exec(ctx, query)
		require.NoError(tc.T, err, "Failed to create auth schema")
	}
}

// EnsureStorageSchema ensures storage schema and tables exist
func (tc *TestContext) EnsureStorageSchema() {
	ctx := context.Background()

	queries := []string{
		`CREATE SCHEMA IF NOT EXISTS storage`,
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
		_, err := tc.DB.Exec(ctx, query)
		require.NoError(tc.T, err, "Failed to create storage schema")
	}
}

// EnsureFunctionsSchema ensures functions schema and tables exist
func (tc *TestContext) EnsureFunctionsSchema() {
	ctx := context.Background()

	queries := []string{
		`CREATE SCHEMA IF NOT EXISTS functions`,
		`CREATE TABLE IF NOT EXISTS functions.functions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT UNIQUE NOT NULL,
			body TEXT NOT NULL,
			enabled BOOLEAN DEFAULT true,
			timeout_ms INTEGER DEFAULT 5000,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,
	}

	for _, query := range queries {
		_, err := tc.DB.Exec(ctx, query)
		require.NoError(tc.T, err, "Failed to create functions schema")
	}
}

// MailHogMessage represents an email message from MailHog
type MailHogMessage struct {
	ID   string `json:"ID"`
	From struct {
		Mailbox string `json:"Mailbox"`
		Domain  string `json:"Domain"`
	} `json:"From"`
	To []struct {
		Mailbox string `json:"Mailbox"`
		Domain  string `json:"Domain"`
	} `json:"To"`
	Content struct {
		Headers map[string][]string `json:"Headers"`
		Body    string              `json:"Body"`
	} `json:"Content"`
	Created time.Time `json:"Created"`
}

// MailHogResponse represents the MailHog API response
type MailHogResponse struct {
	Total    int              `json:"total"`
	Count    int              `json:"count"`
	Start    int              `json:"start"`
	Messages []MailHogMessage `json:"items"`
}

// GetMailHogEmails queries the MailHog API for sent emails
func (tc *TestContext) GetMailHogEmails() ([]MailHogMessage, error) {
	// Query MailHog API
	mailhogURL := "http://mailhog:8025/api/v2/messages"

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(mailhogURL)
	if err != nil {
		return nil, fmt.Errorf("failed to query MailHog: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MailHog returned status %d", resp.StatusCode)
	}

	var mailhogResp MailHogResponse
	if err := json.NewDecoder(resp.Body).Decode(&mailhogResp); err != nil {
		return nil, fmt.Errorf("failed to decode MailHog response: %w", err)
	}

	return mailhogResp.Messages, nil
}

// ClearMailHogEmails deletes all emails from MailHog
func (tc *TestContext) ClearMailHogEmails() error {
	mailhogURL := "http://mailhog:8025/api/v1/messages"

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodDelete, mailhogURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete MailHog messages: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("MailHog delete returned status %d", resp.StatusCode)
	}

	return nil
}

// CleanupStorageFiles deletes test files from local and MinIO storage
func (tc *TestContext) CleanupStorageFiles() {
	// Clean up local storage
	if tc.Config.Storage.Provider == "local" || tc.Config.Storage.LocalPath != "" {
		localPath := tc.Config.Storage.LocalPath
		if localPath != "" && localPath != "/" {
			os.RemoveAll(localPath)
		}
	}

	// Clean up MinIO storage - no cleanup needed for now
	// S3 buckets are ephemeral in test environment

	// Also clean up storage metadata from database
	ctx := context.Background()
	_, _ = tc.DB.Exec(ctx, "TRUNCATE TABLE storage.objects CASCADE")
	_, _ = tc.DB.Exec(ctx, "TRUNCATE TABLE storage.buckets CASCADE")
}

// WaitForEmail waits for an email to arrive in MailHog matching a filter
func (tc *TestContext) WaitForEmail(timeout time.Duration, filter func(MailHogMessage) bool) *MailHogMessage {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		emails, err := tc.GetMailHogEmails()
		if err == nil {
			for _, email := range emails {
				if filter(email) {
					return &email
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}
