// Package testutil provides shared test helper functions for unit testing.
//
// For integration tests requiring database access, use test/dbhelpers instead.
// This package provides lightweight helpers for unit tests without external dependencies.
package testutil

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/config"
)

// =============================================================================
// Database Helpers
// =============================================================================

// SetupTestDB creates a test database connection.
//
// This function connects to the test database using environment variables or default config.
// For integration tests, consider using test/dbhelpers.NewDBTestContext() instead, which provides
// additional features like connection pooling and test table setup.
//
// Example:
//
//	db := testutil.SetupTestDB(t)
//	defer testutil.CleanupTestDB(t, db)
func SetupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	// Load configuration from environment variables
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Build connection URL
	connURL := buildConnectionURL(cfg.Database)

	// Create connection pool
	pool, err := pgxpool.New(context.Background(), connURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("Failed to ping test database: %v", err)
	}

	log.Info().Str("database", cfg.Database.Database).Msg("Test database connection established")
	return pool
}

// buildConnectionURL constructs a PostgreSQL connection URL from config.
func buildConnectionURL(cfg config.DatabaseConfig) string {
	return fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
	)
}

// CleanupTestDB cleans up a test database connection.
func CleanupTestDB(t *testing.T, db *pgxpool.Pool) {
	t.Helper()
	if db != nil {
		db.Close()
	}
}

// CreateTestUser creates a test user in the database.
// Returns the user ID.
//
// Uses the auth.users table schema with password_hash column.
func CreateTestUser(t *testing.T, db *pgxpool.Pool, email string) string {
	t.Helper()
	var userID string
	err := db.QueryRow(context.Background(),
		"INSERT INTO auth.users (email, password_hash, email_verified) VALUES ($1, 'hash', true) RETURNING id",
		email).Scan(&userID)
	require.NoError(t, err)
	return userID
}

// CreateTestTable creates a test table with the specified columns.
//
// Example:
//
//	testutil.CreateTestTable(t, db, "public", "test_products",
//		[]testutil.Column{{Name: "id", Type: "serial PRIMARY KEY"}, {Name: "name", Type: "text"}})
func CreateTestTable(t *testing.T, db *pgxpool.Pool, schema, table string, columns []Column) {
	t.Helper()

	var columnDefs []string
	for _, col := range columns {
		columnDefs = append(columnDefs, fmt.Sprintf("%s %s", col.Name, col.Type))
	}

	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (%s)", schema, table, commaSeparate(columnDefs))
	_, err := db.Exec(context.Background(), query)
	require.NoError(t, err, "Failed to create test table")
}

// Column represents a table column definition.
type Column struct {
	Name string
	Type string
}

// DropTestTable drops a test table.
func DropTestTable(t *testing.T, db *pgxpool.Pool, schema, table string) {
	t.Helper()
	query := fmt.Sprintf("DROP TABLE IF EXISTS %s.%s CASCADE", schema, table)
	_, err := db.Exec(context.Background(), query)
	require.NoError(t, err, "Failed to drop test table")
}

// TruncateTable truncates a table (removes all data but keeps structure).
func TruncateTable(t *testing.T, db *pgxpool.Pool, schema, table string) {
	t.Helper()
	query := fmt.Sprintf("TRUNCATE TABLE %s.%s CASCADE", schema, table)
	_, err := db.Exec(context.Background(), query)
	require.NoError(t, err, "Failed to truncate table")
}

// =============================================================================
// Async/Condition Helpers
// =============================================================================

// WaitForCondition polls until a condition is met or timeout expires.
//
// Example:
//
//	testutil.WaitForCondition(t, func() bool {
//	    return db.Ping(context.Background()) == nil
//	}, 5*time.Second, "database to become available")
func WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration, msg string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		<-ticker.C
	}

	require.Failf(t, "condition not met", "timed out waiting for %s (timeout: %v)", msg, timeout)
}

// =============================================================================
// Fiber HTTP Helpers
// =============================================================================

// SetupTestServer creates a test Fiber server with common middleware.
//
// Example:
//
//	app := testutil.SetupTestServer(t)
//	// Register routes and test
func SetupTestServer(t *testing.T) *fiber.App {
	t.Helper()

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			var e *fiber.Error
			if errors.As(err, &e) {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	return app
}

// =============================================================================
// Assertion Helpers
// =============================================================================

// AssertJSONEqual asserts that two JSON strings represent equal objects.
func AssertJSONEqual(t *testing.T, expected, actual string) {
	t.Helper()

	var expectedJSON, actualJSON interface{}
	err := json.Unmarshal([]byte(expected), &expectedJSON)
	require.NoError(t, err, "Failed to parse expected JSON")
	err = json.Unmarshal([]byte(actual), &actualJSON)
	require.NoError(t, err, "Failed to parse actual JSON")

	assert.Equal(t, expectedJSON, actualJSON, "JSON objects are not equal")
}

// AssertJSONContains asserts that the actual JSON contains all fields from expected JSON.
func AssertJSONContains(t *testing.T, expected, actual string) {
	t.Helper()

	var expectedJSON, actualJSON map[string]interface{}
	err := json.Unmarshal([]byte(expected), &expectedJSON)
	require.NoError(t, err, "Failed to parse expected JSON")
	err = json.Unmarshal([]byte(actual), &actualJSON)
	require.NoError(t, err, "Failed to parse actual JSON")

	for key, expectedValue := range expectedJSON {
		actualValue, exists := actualJSON[key]
		assert.True(t, exists, "Key %s not found in actual JSON", key)
		if exists {
			assert.Equal(t, expectedValue, actualValue, "Value for key %s does not match", key)
		}
	}
}

// =============================================================================
// Error Helpers
// =============================================================================

// AssertErrorIs asserts that an error is of a specific type.
func AssertErrorIs(t *testing.T, err error, target error) {
	t.Helper()
	if err == nil {
		t.Fatalf("Expected error %v, got nil", target)
	}
	assert.ErrorIs(t, err, target, "Error is not of expected type")
}

// AssertErrorContains asserts that an error's message contains a substring.
func AssertErrorContains(t *testing.T, err error, substring string) {
	t.Helper()
	if err == nil {
		t.Fatalf("Expected error containing %q, got nil", substring)
	}
	assert.Contains(t, err.Error(), substring, "Error message does not contain expected substring")
}

// =============================================================================
// Time Helpers
// =============================================================================

// MockTime is a helper for time-sensitive tests.
// It allows you to control the current time in tests.
type MockTime struct {
	mu     sync.Mutex
	now    time.Time
	frozen bool
	offset time.Duration
}

// NewMockTime creates a new mock time helper.
func NewMockTime() *MockTime {
	return &MockTime{
		now:    time.Now(),
		frozen: false,
	}
}

// Freeze freezes time at the current moment.
func (m *MockTime) Freeze() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.now = time.Now()
	m.frozen = true
}

// Unfreeze unfreezes time.
func (m *MockTime) Unfreeze() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.frozen = false
}

// Set sets the current time to a specific value.
func (m *MockTime) Set(t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.now = t
	m.frozen = true
}

// Add adds a duration to the current time.
func (m *MockTime) Add(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.now = m.now.Add(d)
}

// Now returns the current (mock) time.
func (m *MockTime) Now() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.frozen {
		return m.now
	}
	return time.Now().Add(m.offset)
}

// =============================================================================
// Buffer Helpers
// =============================================================================

// ReadBody reads the entire body into a string.
func ReadBody(t *testing.T, r io.ReadCloser) string {
	t.Helper()

	if r == nil {
		return ""
	}

	data, err := io.ReadAll(r)
	require.NoError(t, err, "Failed to read body")
	_ = r.Close() // Best effort close, body already read

	return string(data)
}

// =============================================================================
// SQL Helpers
// =============================================================================

// NullString creates a sql.NullString from a string pointer.
func NullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *s, Valid: true}
}

// NullInt creates a sql.NullInt64 from an int pointer.
func NullInt(i *int) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: int64(*i), Valid: true}
}

// StringPtr creates a string pointer.
func StringPtr(s string) *string {
	return &s
}

// IntPtr creates an int pointer.
func IntPtr(i int) *int {
	return &i
}

// =============================================================================
// Internal Helpers
// =============================================================================

func commaSeparate(items []string) string {
	if len(items) == 0 {
		return ""
	}
	result := items[0]
	for _, item := range items[1:] {
		result += ", " + item
	}
	return result
}
