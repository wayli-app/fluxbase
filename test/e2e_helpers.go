// Package test provides testing utilities and helpers for Fluxbase e2e tests.
//
// # Test Contexts
//
// Two test contexts are available, each using a different database user:
//
//   - NewTestContext: Uses fluxbase_app user WITH BYPASSRLS privilege (RLS policies are NOT enforced)
//   - NewRLSTestContext: Uses fluxbase_rls_test user WITHOUT BYPASSRLS privilege (RLS policies ARE enforced)
//
// Using the correct context is critical for test correctness. Use NewRLSTestContext ONLY when explicitly
// testing Row-Level Security policies. For all other tests, use NewTestContext.
//
// # Helper Method Categories
//
// Context Creation:
//   - NewTestContext() - Standard context (with BYPASSRLS)
//   - NewRLSTestContext() - RLS testing context (no BYPASSRLS)
//
// User Management:
//   - CreateTestUser() - Create regular app user
//   - CreateDashboardAdminUser() - Create platform admin user
//
// Authentication:
//   - CreateAPIKey() - Create API key (respects RLS)
//   - CreateServiceKey() - Create service key (bypasses RLS!)
//   - GenerateAnonKey() - Generate anonymous JWT token
//   - GetAuthToken() - Sign in and get JWT
//   - GetDashboardAuthToken() - Admin sign in
//
// Database Operations:
//   - ExecuteSQL() - Execute as fluxbase_app
//   - ExecuteSQLAsSuperuser() - Execute as postgres superuser
//   - QuerySQL() - Query as fluxbase_app
//   - QuerySQLAsSuperuser() - Query as postgres superuser
//   - QuerySQLAsRLSUser() - Query as fluxbase_rls_test with RLS enforced
//
// Email Testing:
//   - GetMailHogEmails() - Get all emails
//   - ClearMailHogEmails() - Clear all emails
//   - WaitForEmail() - Wait for specific email with timeout
//
// Schema Management:
//   - EnsureAuthSchema() - Ensure auth tables exist
//   - EnsureStorageSchema() - Ensure storage tables exist
//   - EnsureFunctionsSchema() - Ensure functions tables exist
//
// Utilities:
//   - WaitForCondition() - Poll until condition met or timeout
//   - CleanupStorageFiles() - Clean storage bucket
//   - CleanDatabase() - Truncate all tables
package test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/api"
	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// TestContext holds all testing dependencies including database connection,
// API server, Fiber app, and configuration.
//
// Always close the context with defer tc.Close() to ensure proper cleanup.
type TestContext struct {
	DB     *database.Connection
	Server *api.Server
	App    *fiber.App
	Config *config.Config
	T      *testing.T
}

// NewTestContext creates a test context using the fluxbase_app database user.
//
// Database User: fluxbase_app
// Privilege: Has BYPASSRLS (Row-Level Security policies are NOT enforced)
//
// Use this context for:
//   - General REST API testing
//   - Authentication flows
//   - Storage operations
//   - Any test where RLS should be bypassed
//
// Do NOT use for:
//   - Testing RLS policies (use NewRLSTestContext instead)
//
// Example:
//
//	func TestRESTAPI(t *testing.T) {
//	    tc := test.NewTestContext(t)
//	    defer tc.Close()
//
//	    resp := tc.NewRequest("GET", "/api/v1/tables/products").
//	        WithAPIKey(tc.CreateAPIKey("test", nil)).
//	        Send().
//	        AssertStatus(fiber.StatusOK)
//	}
func NewTestContext(t *testing.T) *TestContext {
	cfg := GetTestConfig()

	// Log the database user being used for debugging
	log.Info().
		Str("db_user", cfg.Database.User).
		Str("db_admin_user", cfg.Database.AdminUser).
		Str("db_host", cfg.Database.Host).
		Str("db_database", cfg.Database.Database).
		Msg("Test database configuration")

	// Connect to test database
	db, err := database.NewConnection(cfg.Database)
	require.NoError(t, err, "Failed to connect to test database")

	// Ensure database is healthy
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.Health(ctx)
	require.NoError(t, err, "Database is not healthy")

	// Run migrations BEFORE creating server so REST API can discover tables
	// Note: In CI, migrations are already applied by postgres user during database setup
	// The Migrate() function handles this case gracefully (returns ErrNoChange)
	// Skip migrations entirely to avoid permission issues when CI already ran them
	if os.Getenv("CI") == "" {
		// Only run migrations locally (not in CI)
		err = db.Migrate()
		require.NoError(t, err, "Failed to run migrations")
	}

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

// NewRLSTestContext creates a test context using the fluxbase_rls_test database user.
//
// Database User: fluxbase_rls_test
// Privilege: Does NOT have BYPASSRLS (Row-Level Security policies ARE enforced)
//
// Use this context ONLY for:
//   - Testing RLS policies
//   - Verifying data isolation between users
//   - Testing security boundaries
//
// Do NOT use for:
//   - General API testing (use NewTestContext instead)
//   - Any test where RLS should be bypassed
//
// Example:
//
//	func TestRLSUserIsolation(t *testing.T) {
//	    tc := test.NewRLSTestContext(t)  // RLS policies will be enforced
//	    defer tc.Close()
//
//	    user1ID, token1 := tc.CreateTestUser("user1@example.com", "password123")
//	    user2ID, token2 := tc.CreateTestUser("user2@example.com", "password123")
//
//	    // Create task as user1
//	    tc.NewRequest("POST", "/api/v1/tables/tasks").
//	        WithAuth(token1).
//	        WithBody(map[string]interface{}{"title": "User 1 Task", "user_id": user1ID}).
//	        Send().
//	        AssertStatus(fiber.StatusCreated)
//
//	    // User2 should NOT see user1's task
//	    resp := tc.NewRequest("GET", "/api/v1/tables/tasks").
//	        WithAuth(token2).
//	        Send().
//	        AssertStatus(fiber.StatusOK)
//
//	    var tasks []map[string]interface{}
//	    resp.JSON(&tasks)
//	    require.Len(t, tasks, 0, "User 2 should not see User 1's private tasks")
//	}
func NewRLSTestContext(t *testing.T) *TestContext {
	cfg := GetTestConfig()

	// Override database user to use RLS test user (without BYPASSRLS privilege)
	cfg.Database.User = "fluxbase_rls_test"
	cfg.Database.AdminUser = "fluxbase_rls_test"
	cfg.Database.Password = "fluxbase_rls_test_password"

	// Log the database user being used for debugging
	log.Info().
		Str("db_user", cfg.Database.User).
		Str("db_admin_user", cfg.Database.AdminUser).
		Str("db_host", cfg.Database.Host).
		Str("db_database", cfg.Database.Database).
		Msg("RLS test database configuration (user without BYPASSRLS)")

	// Connect to test database
	db, err := database.NewConnection(cfg.Database)
	require.NoError(t, err, "Failed to connect to test database with RLS user")

	// Ensure database is healthy
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.Health(ctx)
	require.NoError(t, err, "Database is not healthy")

	// Don't run migrations as RLS user - migrations should already be applied
	// by setup using fluxbase_app user

	// Create server (REST API will see all migrated tables)
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
	// Load .env file if it exists (for local development)
	// Use Overload() to override existing environment variables (e.g., from devcontainer)
	// Ignore errors since .env may not exist in CI
	_ = godotenv.Overload()
	_ = godotenv.Overload("../.env") // Try parent directory for test subdirectories

	// Allow environment variables to override defaults for CI
	dbHost := getEnvOrDefault("FLUXBASE_DATABASE_HOST", "postgres")
	dbUser := getEnvOrDefault("FLUXBASE_DATABASE_USER", "fluxbase_app")
	dbAdminUser := getEnvOrDefault("FLUXBASE_DATABASE_ADMIN_USER", "postgres") // Default to postgres for migrations
	dbPassword := getEnvOrDefault("FLUXBASE_DATABASE_PASSWORD", "fluxbase_app_password")
	dbAdminPassword := getEnvOrDefault("FLUXBASE_DATABASE_ADMIN_PASSWORD", "postgres") // Default to postgres password
	dbDatabase := getEnvOrDefault("FLUXBASE_DATABASE_DATABASE", "fluxbase_test")
	smtpHost := getEnvOrDefault("FLUXBASE_EMAIL_SMTP_HOST", "mailhog")
	s3Endpoint := getEnvOrDefault("FLUXBASE_STORAGE_S3_ENDPOINT", "minio:9000")
	functionsDir := getEnvOrDefault("FLUXBASE_FUNCTIONS_FUNCTIONS_DIR", "")

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
			User:            dbUser,          // Runtime user (configurable via env)
			AdminUser:       dbAdminUser,     // Admin user for migrations (configurable via env)
			Password:        dbPassword,      // Runtime password (configurable via env)
			AdminPassword:   dbAdminPassword, // Admin password (configurable via env)
			Database:        dbDatabase,
			SSLMode:         "disable",
			MaxConnections:  20,               // Support parallel test execution (4-8 tests concurrent)
			MinConnections:  4,                // Keep warm connections for parallel tests
			MaxConnLifetime: 5 * time.Minute,  // Shorter lifetime to recycle connections
			MaxConnIdleTime: 30 * time.Second, // Close idle connections faster
			HealthCheck:     1 * time.Minute,
		},
		Auth: config.AuthConfig{
			JWTSecret:       "test-secret-key-for-testing-only",
			JWTExpiry:       15 * time.Minute,
			RefreshExpiry:   7 * 24 * time.Hour,
			PasswordMinLen:  8,
			BcryptCost:      4, // Reduced for test speed (4=~5ms vs 10=~100ms per hash, 20x faster)
			EnableSignup:    true,
			EnableMagicLink: true,
			TOTPIssuer:      "Fluxbase", // Default TOTP issuer for 2FA
		},
		Security: config.SecurityConfig{
			SetupToken:            "test-setup-token-for-e2e-testing",
			EnableGlobalRateLimit: false,
			AdminSetupRateLimit:   10,
			AdminSetupRateWindow:  15 * time.Minute,
			AdminLoginRateLimit:   10,
			AdminLoginRateWindow:  15 * time.Minute,
			AuthLoginRateLimit:    10,
			AuthLoginRateWindow:   15 * time.Minute,
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
		Functions: config.FunctionsConfig{
			Enabled:            true,
			FunctionsDir:       functionsDir,
			AutoLoadOnBoot:     false,
			DefaultTimeout:     30,
			MaxTimeout:         300,
			DefaultMemoryLimit: 128,
			MaxMemoryLimit:     1024,
		},
		Jobs: config.JobsConfig{
			Enabled:                 false, // Disabled for tests by default
			JobsDir:                 "",
			AutoLoadOnBoot:          false,
			EmbeddedWorkerCount:     0, // No workers in tests
			DefaultMaxDuration:      5 * time.Minute,
			MaxMaxDuration:          1 * time.Hour,
			DefaultProgressTimeout:  60 * time.Second,
			PollInterval:            1 * time.Second,
			WorkerHeartbeatInterval: 10 * time.Second,
		},
		API: config.APIConfig{
			MaxPageSize:     1000,
			MaxTotalResults: 10000,
			DefaultPageSize: 1000,
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

// WithAuth sets the Authorization header with Bearer token (JWT).
//
// This is an alias for WithBearerToken for backward compatibility.
//
// Use for: User authentication with JWT tokens
// RLS: Respects RLS policies for the authenticated user
//
// Example:
//
//	userID, token := tc.CreateTestUser("user@example.com", "password123")
//	resp := tc.NewRequest("GET", "/api/v1/auth/user").
//	    WithAuth(token).
//	    Send()
func (r *APIRequest) WithAuth(token string) *APIRequest {
	return r.WithBearerToken(token)
}

// WithBearerToken sets the Authorization header with a Bearer token (JWT).
//
// Header Set: Authorization: Bearer {token}
// Use for: User authentication with JWT tokens
// RLS: Respects RLS policies for the authenticated user
//
// Example:
//
//	userID, token := tc.CreateTestUser("user@example.com", "password123")
//	resp := tc.NewRequest("GET", "/api/v1/tables/tasks").
//	    WithBearerToken(token).
//	    Send()
func (r *APIRequest) WithBearerToken(token string) *APIRequest {
	r.headers["Authorization"] = "Bearer " + token
	return r
}

// WithAPIKey sets the X-API-Key header for API key authentication.
//
// Header Set: X-API-Key: {apiKey}
// Use for: Project-level API key authentication
// RLS: Respects RLS policies
//
// Example:
//
//	apiKey := tc.CreateAPIKey("My API Key", []string{"read", "write"})
//	resp := tc.NewRequest("GET", "/api/v1/tables/products").
//	    WithAPIKey(apiKey).
//	    Send()
func (r *APIRequest) WithAPIKey(apiKey string) *APIRequest {
	r.headers["X-API-Key"] = apiKey
	return r
}

// WithServiceKey sets the X-Service-Key header for service role authentication.
//
// ⚠️ WARNING: Service keys BYPASS RLS POLICIES!
//
// Header Set: X-Service-Key: {serviceKey}
// Use for: Admin operations, bypassing RLS
// RLS: BYPASSES all RLS policies (full database access)
//
// Only use service keys for:
//   - Administrative operations
//   - System-level tasks
//   - Operations that need to bypass RLS
//
// Example:
//
//	serviceKey := tc.CreateServiceKey("Admin Service Key")
//	resp := tc.NewRequest("DELETE", "/api/v1/admin/users/"+userID).
//	    WithServiceKey(serviceKey).
//	    Send()
func (r *APIRequest) WithServiceKey(serviceKey string) *APIRequest {
	r.headers["X-Service-Key"] = serviceKey
	return r
}

// Unauthenticated explicitly marks this request as unauthenticated by removing all auth headers.
//
// Use for:
//   - Testing authentication failures
//   - Testing public endpoints
//   - Making intent explicit in tests
//
// Note: This is the default behavior, but using this method makes test intent clearer.
//
// Example:
//
//	resp := tc.NewRequest("GET", "/api/v1/auth/user").
//	    Unauthenticated().
//	    Send().
//	    AssertStatus(fiber.StatusUnauthorized)
func (r *APIRequest) Unauthenticated() *APIRequest {
	// Remove any authentication headers that may have been set
	delete(r.headers, "Authorization")
	delete(r.headers, "X-API-Key")
	delete(r.headers, "X-Service-Key")
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

	// Execute request with 15 second timeout (to accommodate race detector slowness)
	resp, err := r.tc.App.Test(req, 15000) // 15 second timeout
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

// WaitForCondition polls a condition function until it returns true or timeout occurs.
//
// Parameters:
//   - timeout: Maximum time to wait
//   - checkInterval: Time between condition checks
//   - condition: Function that returns true when condition is met
//
// Returns: true if condition met within timeout, false if timeout occurred
//
// Use this instead of time.Sleep() to make tests more reliable and faster.
//
// Example:
//
//	// Wait for webhook to be delivered (max 5 seconds, check every 100ms)
//	success := tc.WaitForCondition(5*time.Second, 100*time.Millisecond, func() bool {
//	    results := tc.QuerySQL("SELECT COUNT(*) FROM webhook_events WHERE webhook_id = $1", webhookID)
//	    return results[0]["count"].(int64) > 0
//	})
//	require.True(t, success, "Webhook should be delivered within 5 seconds")
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

// ExecuteSQL executes raw SQL as the fluxbase_app database user.
//
// Use for:
//   - Test data setup
//   - Table truncation
//   - Schema modifications
//
// RLS: If using NewTestContext, RLS is bypassed (BYPASSRLS privilege).
// If using NewRLSTestContext, RLS is enforced.
//
// Example:
//
//	tc.ExecuteSQL("TRUNCATE TABLE products CASCADE")
//	tc.ExecuteSQL("INSERT INTO products (name, price) VALUES ($1, $2)", "Test Product", 29.99)
func (tc *TestContext) ExecuteSQL(sql string, args ...interface{}) {
	ctx := context.Background()
	_, err := tc.DB.Exec(ctx, sql, args...)
	require.NoError(tc.T, err)
}

// ExecuteSQLAsSuperuser executes raw SQL as the postgres superuser.
//
// Database User: postgres (superuser)
// Privilege: Full database access, always bypasses RLS
//
// Use for:
//   - Setting up test data that violates RLS policies
//   - Granting permissions
//   - Creating schemas
//   - Administrative operations
//
// Example:
//
//	// Insert task for user without going through RLS
//	tc.ExecuteSQLAsSuperuser(`
//	    INSERT INTO tasks (user_id, title, description)
//	    VALUES ($1, 'Admin Created Task', 'Created by superuser')
//	`, userID)
func (tc *TestContext) ExecuteSQLAsSuperuser(sql string, args ...interface{}) {
	ctx := context.Background()

	// Create a temporary connection as postgres superuser
	connStr := fmt.Sprintf("host=%s port=%d user=postgres password=postgres dbname=%s sslmode=disable",
		tc.Config.Database.Host, tc.Config.Database.Port, tc.Config.Database.Database)

	conn, err := pgx.Connect(ctx, connStr)
	require.NoError(tc.T, err, "Failed to connect as superuser")
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, sql, args...)
	require.NoError(tc.T, err, "Failed to execute SQL as superuser")
}

// QuerySQL executes a SQL query and returns results with PostgreSQL types converted to Go types.
//
// PostgreSQL types are automatically converted:
//   - NUMERIC → float64
//   - TIMESTAMP → time.Time
//   - UUID → string
//   - TEXT → string
//   - BOOLEAN → bool
//
// This makes test assertions straightforward without dealing with pgtype.* types.
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
			// Convert pgtype values to standard Go types for easier testing
			row[string(col.Name)] = convertPgTypeToGoType(values[i])
		}

		results = append(results, row)
	}

	return results
}

// convertPgTypeToGoType converts PostgreSQL driver types to standard Go types for testing.
// This makes test assertions much easier by avoiding pgtype.* types.
func convertPgTypeToGoType(v interface{}) interface{} {
	if v == nil {
		return nil
	}

	// Handle UUID byte array (from pgx v5 rows.Values())
	if bytes, ok := v.([16]uint8); ok {
		// Convert UUID bytes to standard string format (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)
		return fmt.Sprintf("%x-%x-%x-%x-%x",
			bytes[0:4],
			bytes[4:6],
			bytes[6:8],
			bytes[8:10],
			bytes[10:16])
	}

	// Handle common pgtype conversions
	switch val := v.(type) {
	case pgtype.Numeric:
		// Convert NUMERIC to float64
		if !val.Valid {
			return nil
		}
		f, err := val.Float64Value()
		if err == nil {
			return f.Float64
		}
		// Fallback to string representation if float conversion fails
		return val.Int.String()
	case pgtype.Text:
		if !val.Valid {
			return nil
		}
		return val.String
	case pgtype.Bool:
		if !val.Valid {
			return nil
		}
		return val.Bool
	case pgtype.UUID:
		if !val.Valid {
			return nil
		}
		// Convert UUID bytes to standard string format (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)
		return fmt.Sprintf("%x-%x-%x-%x-%x",
			val.Bytes[0:4],
			val.Bytes[4:6],
			val.Bytes[6:8],
			val.Bytes[8:10],
			val.Bytes[10:16])
	case pgtype.Timestamp:
		if !val.Valid {
			return nil
		}
		return val.Time
	case pgtype.Timestamptz:
		if !val.Valid {
			return nil
		}
		return val.Time
	default:
		// Return as-is for types that don't need conversion
		return v
	}
}

// QuerySQLAsSuperuser executes a SQL query as postgres superuser (bypasses RLS)
// This is useful for verifying test data that should not be visible through RLS
func (tc *TestContext) QuerySQLAsSuperuser(sql string, args ...interface{}) []map[string]interface{} {
	ctx := context.Background()

	// Create a temporary connection as postgres superuser
	connStr := fmt.Sprintf("host=%s port=%d user=postgres password=postgres dbname=%s sslmode=disable",
		tc.Config.Database.Host, tc.Config.Database.Port, tc.Config.Database.Database)

	conn, err := pgx.Connect(ctx, connStr)
	require.NoError(tc.T, err, "Failed to connect as superuser")
	defer conn.Close(ctx)

	rows, err := conn.Query(ctx, sql, args...)
	require.NoError(tc.T, err, "Failed to query as superuser")
	defer rows.Close()

	results := make([]map[string]interface{}, 0)

	for rows.Next() {
		values, err := rows.Values()
		require.NoError(tc.T, err)

		row := make(map[string]interface{})
		for i, col := range rows.FieldDescriptions() {
			// Convert pgtype values to standard Go types for easier testing
			row[string(col.Name)] = convertPgTypeToGoType(values[i])
		}

		results = append(results, row)
	}

	return results
}

// QuerySQLAsRLSUser executes a SQL query as the fluxbase_rls_test user with RLS enforced
// This simulates a regular authenticated user with a specific user_id and role
//
// Parameters:
//   - sql: The SQL query to execute
//   - userID: The user ID to set in RLS context (will be set as app.user_id)
//   - args: Optional query arguments
//
// The RLS context is set with:
//   - app.user_id = userID parameter
//   - app.role = 'authenticated'
//
// Use this to test that RLS policies correctly restrict data access.
//
// Example:
//
//	// Test that user1 cannot see user2's records
//	records := tc.QuerySQLAsRLSUser(`SELECT * FROM tasks WHERE user_id = $1`, user1ID, user2ID)
//	require.Len(t, records, 0, "User1 should not see User2's tasks")
func (tc *TestContext) QuerySQLAsRLSUser(sql string, userID string, args ...interface{}) []map[string]interface{} {
	ctx := context.Background()

	// Create a temporary connection as fluxbase_rls_test user (no BYPASSRLS)
	connStr := fmt.Sprintf("host=%s port=%d user=fluxbase_rls_test password=fluxbase_rls_test_password dbname=%s sslmode=disable",
		tc.Config.Database.Host, tc.Config.Database.Port, tc.Config.Database.Database)

	conn, err := pgx.Connect(ctx, connStr)
	require.NoError(tc.T, err, "Failed to connect as RLS test user")
	defer conn.Close(ctx)

	// Begin a transaction to set RLS context
	tx, err := conn.Begin(ctx)
	require.NoError(tc.T, err, "Failed to begin transaction")
	defer func() { _ = tx.Rollback(ctx) }()

	// Set RLS context variables (these affect RLS policy checks)
	// Set request.jwt.claims with user ID and role (Supabase/Fluxbase format)
	jwtClaims := fmt.Sprintf(`{"sub":"%s","role":"authenticated"}`, userID)
	_, err = tx.Exec(ctx, "SELECT set_config('request.jwt.claims', $1, true)", jwtClaims)
	require.NoError(tc.T, err, "Failed to set request.jwt.claims")

	// Execute the query with RLS context applied
	rows, err := tx.Query(ctx, sql, args...)
	require.NoError(tc.T, err, "Failed to query as RLS user")
	defer rows.Close()

	results := make([]map[string]interface{}, 0)

	for rows.Next() {
		values, err := rows.Values()
		require.NoError(tc.T, err)

		row := make(map[string]interface{})
		for i, col := range rows.FieldDescriptions() {
			// Convert pgtype values to standard Go types for easier testing
			row[string(col.Name)] = convertPgTypeToGoType(values[i])
		}

		results = append(results, row)
	}

	// Commit transaction (though we're only reading)
	err = tx.Commit(ctx)
	require.NoError(tc.T, err, "Failed to commit transaction")

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

// CreateTestUserWithRole creates a test user with a specific role, returns userID and JWT token
func (tc *TestContext) CreateTestUserWithRole(email, password, role string) (userID, token string) {
	// Create the user normally
	userID, token = tc.CreateTestUser(email, password)

	// Update the user's role in the database as superuser
	tc.ExecuteSQLAsSuperuser(`
		UPDATE auth.users
		SET role = $1
		WHERE id = $2
	`, role, userID)

	// Invalidate the old token and create a new session with the updated role
	// This ensures the JWT contains the correct role
	tc.ExecuteSQLAsSuperuser(`
		DELETE FROM auth.sessions WHERE user_id = $1
	`, userID)

	// Login again to get a token with the updated role
	loginResp := tc.NewRequest("POST", "/api/v1/auth/signin").
		WithBody(map[string]interface{}{
			"email":    email,
			"password": password,
		}).
		Send().
		AssertStatus(fiber.StatusOK)

	var loginResult map[string]interface{}
	loginResp.JSON(&loginResult)

	if accessToken, ok := loginResult["access_token"].(string); ok {
		token = accessToken
	}

	require.NotEmpty(tc.T, token, "Access token not returned from signin")

	return userID, token
}

// CreateDashboardAdminUser creates a Fluxbase dashboard admin user (platform administrator)
// and returns userID and token. This is different from app users in auth.users.
func (tc *TestContext) CreateDashboardAdminUser(email, password string) (userID, token string) {
	ctx := context.Background()

	// Try the initial setup endpoint (creates first admin user)
	setupResp := tc.NewRequest("POST", "/api/v1/admin/setup").
		WithBody(map[string]interface{}{
			"email":       email,
			"password":    password,
			"name":        "Admin User",
			"setup_token": tc.Config.Security.SetupToken,
		}).
		Send()

	// If setup already done or successful, extract token
	if setupResp.Status() == fiber.StatusCreated || setupResp.Status() == fiber.StatusOK {
		var result map[string]interface{}
		setupResp.JSON(&result)

		if user, ok := result["user"].(map[string]interface{}); ok {
			if id, ok := user["id"].(string); ok {
				userID = id
			}
		}

		if accessToken, ok := result["access_token"].(string); ok {
			token = accessToken
			return userID, token
		}
	}

	// If setup was already done, create user directly in database using bcrypt
	// Use Go's bcrypt package to match what the Login function expects
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		require.NoError(tc.T, err, "Failed to hash password")
	}

	// Insert dashboard user with bcrypt hash using superuser connection
	// This is necessary for RLS test contexts where the test user doesn't have BYPASSRLS
	// Create a temporary connection as postgres superuser
	connStr := fmt.Sprintf("host=%s port=%d user=postgres password=postgres dbname=%s sslmode=disable",
		tc.Config.Database.Host, tc.Config.Database.Port, tc.Config.Database.Database)

	conn, err := pgx.Connect(ctx, connStr)
	require.NoError(tc.T, err, "Failed to connect as superuser for dashboard admin creation")
	defer conn.Close(ctx)

	err = conn.QueryRow(ctx,
		`INSERT INTO dashboard.users (email, password_hash, full_name, role, email_verified)
		 VALUES ($1, $2, $3, 'dashboard_admin', true)
		 ON CONFLICT (email) DO UPDATE
		 SET password_hash = EXCLUDED.password_hash
		 RETURNING id`,
		email, string(passwordHash), "Admin User").Scan(&userID)

	if err != nil {
		require.NoError(tc.T, err, "Failed to create dashboard admin user")
	}

	// Get token by signing in
	token = tc.GetDashboardAuthToken(email, password)

	require.NotEmpty(tc.T, userID, "Dashboard admin user ID not created")
	require.NotEmpty(tc.T, token, "Dashboard admin token not created")

	return userID, token
}

// GetDashboardAuthToken signs in with email/password for dashboard users and returns JWT token
func (tc *TestContext) GetDashboardAuthToken(email, password string) string {
	resp := tc.NewRequest("POST", "/api/v1/admin/login").
		WithBody(map[string]interface{}{
			"email":    email,
			"password": password,
		}).
		Send()

	// Check if login succeeded
	if resp.Status() != fiber.StatusOK {
		tc.T.Logf("Dashboard login failed with status %d, body: %s", resp.Status(), string(resp.Body()))
		require.Equal(tc.T, fiber.StatusOK, resp.Status(), "Dashboard login should succeed")
	}

	var result map[string]interface{}
	resp.JSON(&result)

	token, ok := result["access_token"].(string)
	require.True(tc.T, ok, "access_token not found in dashboard login response")
	require.NotEmpty(tc.T, token, "access_token is empty")

	return token
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
// Note: auth schema is already created by migrations, so we only ensure tables exist
func (tc *TestContext) EnsureAuthSchema() {
	ctx := context.Background()

	// Note: Don't create auth schema here - it's already created by migrations
	// The RLS test user only has USAGE and CREATE on auth schema (for tables),
	// not permission to create the schema itself
	queries := []string{
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
		require.NoError(tc.T, err, "Failed to ensure auth tables exist")
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

// EnsureRLSTestTables ensures test tables for RLS testing exist with proper policies
func (tc *TestContext) EnsureRLSTestTables() {
	ctx := context.Background()

	queries := []string{
		// Ensure uuid-ossp extension is available for uuid_generate_v4()
		`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`,

		// Create tasks table for RLS testing
		`CREATE TABLE IF NOT EXISTS public.tasks (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			description TEXT,
			completed BOOLEAN DEFAULT FALSE,
			is_public BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,

		// Enable RLS on tasks table
		`ALTER TABLE public.tasks ENABLE ROW LEVEL SECURITY`,
		`ALTER TABLE public.tasks FORCE ROW LEVEL SECURITY`,

		// Drop existing policies if they exist (for idempotency)
		`DROP POLICY IF EXISTS tasks_select_own ON public.tasks`,
		`DROP POLICY IF EXISTS tasks_insert_own ON public.tasks`,
		`DROP POLICY IF EXISTS tasks_update_own ON public.tasks`,
		`DROP POLICY IF EXISTS tasks_delete_own ON public.tasks`,

		// CREATE policies for tasks table
		// SELECT: Users can see their own tasks or public tasks
		`CREATE POLICY tasks_select_own ON public.tasks
			FOR SELECT
			USING (user_id = auth.current_user_id() OR is_public = true)`,

		// INSERT: Authenticated users can insert tasks
		`CREATE POLICY tasks_insert_own ON public.tasks
			FOR INSERT
			WITH CHECK (user_id = auth.current_user_id())`,

		// UPDATE: Users can update their own tasks
		`CREATE POLICY tasks_update_own ON public.tasks
			FOR UPDATE
			USING (user_id = auth.current_user_id())`,

		// DELETE: Users can delete their own tasks
		`CREATE POLICY tasks_delete_own ON public.tasks
			FOR DELETE
			USING (user_id = auth.current_user_id())`,
	}

	// Create a temporary connection as postgres superuser to create tables and policies
	// This is necessary because the RLS test user doesn't have table ownership permissions
	connStr := fmt.Sprintf("host=%s port=%d user=postgres password=postgres dbname=%s sslmode=disable",
		tc.Config.Database.Host, tc.Config.Database.Port, tc.Config.Database.Database)

	conn, err := pgx.Connect(ctx, connStr)
	require.NoError(tc.T, err, "Failed to connect as superuser for table creation")
	defer conn.Close(ctx)

	for _, query := range queries {
		_, err := conn.Exec(ctx, query)
		require.NoError(tc.T, err, "Failed to create RLS test tables: %v", err)
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

	// Restore default buckets after cleanup
	_, _ = tc.DB.Exec(ctx, `
		INSERT INTO storage.buckets (id, name, public) VALUES
			('public', 'public', true),
			('temp-files', 'temp-files', false),
			('user-uploads', 'user-uploads', false)
		ON CONFLICT (id) DO NOTHING
	`)
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

// CreateAPIKey creates a test API key and returns the plaintext key for use in tests
func (tc *TestContext) CreateAPIKey(name string, scopes []string) string {
	ctx := context.Background()

	// Create API key service
	apiKeyService := auth.NewAPIKeyService(tc.DB.Pool())

	// Generate API key with provided scopes
	// Use default scopes if none provided (read/write for tables, storage, functions)
	if len(scopes) == 0 {
		scopes = []string{"*"} // All scopes for testing
	}

	keyWithPlaintext, err := apiKeyService.GenerateAPIKey(
		ctx,
		name,
		nil, // no description
		nil, // no user association
		scopes,
		1000, // high rate limit for tests
		nil,  // no expiration
	)
	require.NoError(tc.T, err, "Failed to create API key")
	require.NotEmpty(tc.T, keyWithPlaintext.PlaintextKey, "API key plaintext is empty")

	return keyWithPlaintext.PlaintextKey
}

// CreateServiceKey creates a test service key and returns the plaintext key for use in tests
// Service keys have elevated privileges and bypass RLS
func (tc *TestContext) CreateServiceKey(name string) string {
	ctx := context.Background()

	// Generate random key bytes
	keyBytes := make([]byte, 32)
	_, err := rand.Read(keyBytes)
	require.NoError(tc.T, err, "Failed to generate random bytes")

	// Format as service key with sk_test_ prefix
	plaintextKey := "sk_test_" + base64.URLEncoding.EncodeToString(keyBytes)

	// Extract prefix (first 16 chars for identification to ensure uniqueness)
	// This includes "sk_test_" plus some random chars to avoid collisions
	keyPrefix := plaintextKey[:16]

	// Hash with bcrypt (cost 10 for tests, faster than production cost 12)
	keyHash, err := bcrypt.GenerateFromPassword([]byte(plaintextKey), 10)
	require.NoError(tc.T, err, "Failed to hash service key")

	// Insert into auth.service_keys table
	query := `
		INSERT INTO auth.service_keys (name, description, key_hash, key_prefix, enabled)
		VALUES ($1, $2, $3, $4, true)
		RETURNING id
	`

	var keyID uuid.UUID
	err = tc.DB.QueryRow(ctx, query,
		name,
		"Test service key for "+name,
		string(keyHash),
		keyPrefix,
	).Scan(&keyID)
	require.NoError(tc.T, err, "Failed to insert service key")
	require.NotEmpty(tc.T, keyID, "Service key ID is empty")

	return plaintextKey
}

// GenerateAnonKey generates an anonymous JWT token (anon key) for testing
// Anon keys are JWT tokens with role="anon" that allow anonymous access
func (tc *TestContext) GenerateAnonKey() string {
	// Create JWT manager using test config
	jwtManager := auth.NewJWTManager(
		tc.Config.Auth.JWTSecret,
		tc.Config.Auth.JWTExpiry,
		tc.Config.Auth.RefreshExpiry,
	)

	// Generate anonymous access token with random user ID
	userID := uuid.New().String()
	token, err := jwtManager.GenerateAnonymousAccessToken(userID)
	require.NoError(tc.T, err, "Failed to generate anon key")
	require.NotEmpty(tc.T, token, "Anon key is empty")

	return token
}

// CreateUser creates a regular app user (auth.users) and returns userID and JWT token
// This is an alias for CreateTestUser for clarity
func (tc *TestContext) CreateUser(email, password string) (userID, token string) {
	return tc.CreateTestUser(email, password)
}

// WithJSON is a convenience method to set the request body as JSON
func (r *APIRequest) WithJSON(data interface{}) *APIRequest {
	return r.WithBody(data)
}

// RandomEmail generates a random email address for testing to avoid conflicts
func RandomEmail() string {
	return fmt.Sprintf("test-%s@example.com", uuid.New().String()[:8])
}
