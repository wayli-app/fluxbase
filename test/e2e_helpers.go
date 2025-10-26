package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wayli-app/fluxbase/internal/api"
	"github.com/wayli-app/fluxbase/internal/config"
	"github.com/wayli-app/fluxbase/internal/database"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
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

	// Create server
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

// GetTestConfig returns a test configuration
func GetTestConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Address:      ":8081",
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
			BodyLimit:    10 * 1024 * 1024,
		},
		Database: config.DatabaseConfig{
			Host:            "postgres",
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
		},
		Debug: true,
	}
}

// APIRequest is a helper for making HTTP requests to the test server
type APIRequest struct {
	tc     *TestContext
	method string
	path   string
	body   interface{}
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

	// Execute request
	resp, err := r.tc.App.Test(req, -1) // -1 means no timeout
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
