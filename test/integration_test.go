package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wayli-app/fluxbase/internal/api"
	"github.com/wayli-app/fluxbase/internal/config"
	"github.com/wayli-app/fluxbase/internal/database"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// IntegrationTestSuite defines the integration test suite
type IntegrationTestSuite struct {
	suite.Suite
	db     *database.Connection
	server *api.Server
	app    *fiber.App
	config *config.Config
}

// SetupSuite runs once before all tests
func (suite *IntegrationTestSuite) SetupSuite() {
	// Load test configuration
	cfg := &config.Config{
		Server: config.ServerConfig{
			Address:      ":8081",
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
			BodyLimit:    10 * 1024 * 1024,
		},
		Database: config.DatabaseConfig{
			Host:            "localhost",
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
			JWTSecret:       "test-secret-key",
			JWTExpiry:       15 * time.Minute,
			RefreshExpiry:   7 * 24 * time.Hour,
			PasswordMinLen:  8,
			BcryptCost:      10,
			EnableSignup:    true,
			EnableMagicLink: true,
		},
		Debug: true,
	}

	suite.config = cfg

	// Connect to test database
	db, err := database.NewConnection(cfg.Database)
	require.NoError(suite.T(), err, "Failed to connect to test database")
	suite.db = db

	// Run migrations
	err = db.Migrate()
	require.NoError(suite.T(), err, "Failed to run migrations")

	// Create test server
	server := api.NewServer(cfg, db)
	suite.server = server
	suite.app = server.App() // Assuming we add a method to expose the Fiber app
}

// TearDownSuite runs once after all tests
func (suite *IntegrationTestSuite) TearDownSuite() {
	// Clean up database
	ctx := context.Background()
	_, err := suite.db.Exec(ctx, `
		DROP SCHEMA IF EXISTS public CASCADE;
		DROP SCHEMA IF EXISTS auth CASCADE;
		DROP SCHEMA IF EXISTS storage CASCADE;
		DROP SCHEMA IF EXISTS realtime CASCADE;
		DROP SCHEMA IF EXISTS functions CASCADE;
		CREATE SCHEMA public;
	`)
	require.NoError(suite.T(), err)

	// Close database connection
	suite.db.Close()
}

// SetupTest runs before each test
func (suite *IntegrationTestSuite) SetupTest() {
	// Clean test data before each test
	ctx := context.Background()
	_, err := suite.db.Exec(ctx, `
		TRUNCATE TABLE auth.users CASCADE;
		TRUNCATE TABLE auth.sessions CASCADE;
		TRUNCATE TABLE storage.buckets CASCADE;
		TRUNCATE TABLE storage.objects CASCADE;
	`)
	require.NoError(suite.T(), err)
}

// Test REST API

func (suite *IntegrationTestSuite) TestHealthEndpoint() {
	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := suite.app.Test(req)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "ok", result["status"])
}

func (suite *IntegrationTestSuite) TestGetTables() {
	req := httptest.NewRequest("GET", "/api/rest/", nil)
	resp, err := suite.app.Test(req)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var tables []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&tables)
	require.NoError(suite.T(), err)

	assert.NotEmpty(suite.T(), tables)
}

func (suite *IntegrationTestSuite) TestCRUDOperations() {
	// First, create a test table
	ctx := context.Background()
	_, err := suite.db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_items (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL,
			description TEXT,
			quantity INTEGER DEFAULT 0,
			active BOOLEAN DEFAULT true,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)
	`)
	require.NoError(suite.T(), err)

	// Test CREATE
	createPayload := map[string]interface{}{
		"name":        "Test Item",
		"description": "This is a test item",
		"quantity":    10,
	}

	body, _ := json.Marshal(createPayload)
	req := httptest.NewRequest("POST", "/api/rest/test_items", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := suite.app.Test(req)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

	var createdItem map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&createdItem)
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), createdItem["id"])
	assert.Equal(suite.T(), "Test Item", createdItem["name"])

	itemID := createdItem["id"].(string)

	// Test READ (single item)
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/rest/test_items/%s", itemID), nil)
	resp, err = suite.app.Test(req)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var fetchedItem map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&fetchedItem)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), itemID, fetchedItem["id"])

	// Test UPDATE
	updatePayload := map[string]interface{}{
		"name":     "Updated Item",
		"quantity": 20,
	}

	body, _ = json.Marshal(updatePayload)
	req = httptest.NewRequest("PATCH", fmt.Sprintf("/api/rest/test_items/%s", itemID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err = suite.app.Test(req)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var updatedItem map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&updatedItem)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Updated Item", updatedItem["name"])
	assert.Equal(suite.T(), float64(20), updatedItem["quantity"])

	// Test LIST with filters
	req = httptest.NewRequest("GET", "/api/rest/test_items?name=eq.Updated Item", nil)
	resp, err = suite.app.Test(req)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var items []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&items)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), items, 1)

	// Test DELETE
	req = httptest.NewRequest("DELETE", fmt.Sprintf("/api/rest/test_items/%s", itemID), nil)
	resp, err = suite.app.Test(req)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusNoContent, resp.StatusCode)

	// Verify deletion
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/rest/test_items/%s", itemID), nil)
	resp, err = suite.app.Test(req)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusNotFound, resp.StatusCode)
}

func (suite *IntegrationTestSuite) TestQueryParameters() {
	// Create test data
	ctx := context.Background()
	_, err := suite.db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS products (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL,
			price DECIMAL(10,2),
			category TEXT,
			in_stock BOOLEAN DEFAULT true,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);

		INSERT INTO products (name, price, category, in_stock) VALUES
			('Product A', 10.99, 'electronics', true),
			('Product B', 25.50, 'electronics', false),
			('Product C', 5.99, 'books', true),
			('Product D', 15.00, 'books', true),
			('Product E', 99.99, 'electronics', true);
	`)
	require.NoError(suite.T(), err)

	// Test filtering
	req := httptest.NewRequest("GET", "/api/rest/products?category=eq.electronics", nil)
	resp, err := suite.app.Test(req)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var products []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&products)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), products, 3)

	// Test ordering
	req = httptest.NewRequest("GET", "/api/rest/products?order=price.desc", nil)
	resp, err = suite.app.Test(req)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	err = json.NewDecoder(resp.Body).Decode(&products)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), float64(99.99), products[0]["price"])

	// Test pagination
	req = httptest.NewRequest("GET", "/api/rest/products?limit=2&offset=1", nil)
	resp, err = suite.app.Test(req)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	err = json.NewDecoder(resp.Body).Decode(&products)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), products, 2)

	// Test complex filters
	req = httptest.NewRequest("GET", "/api/rest/products?price=gt.10&in_stock=eq.true", nil)
	resp, err = suite.app.Test(req)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	err = json.NewDecoder(resp.Body).Decode(&products)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), products, 2)
}

// Test Authentication (placeholder for now)

func (suite *IntegrationTestSuite) TestAuthSignup() {
	payload := map[string]interface{}{
		"email":    "test@example.com",
		"password": "SecurePassword123!",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := suite.app.Test(req)
	require.NoError(suite.T(), err)

	// For now, we expect this to return a placeholder response
	// When auth is implemented, update this test
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
}

// Benchmark Tests

func BenchmarkRESTGetList(b *testing.B) {
	// Setup
	cfg := getTestConfig()
	db, _ := database.NewConnection(cfg.Database)
	defer db.Close()

	server := api.NewServer(cfg, db)
	app := server.App()

	// Create test data
	ctx := context.Background()
	db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS benchmark_items (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT,
			value INTEGER
		);

		INSERT INTO benchmark_items (name, value)
		SELECT 'Item ' || i, i
		FROM generate_series(1, 1000) i;
	`)

	b.ResetTimer()

	// Run benchmark
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/rest/benchmark_items?limit=100", nil)
		app.Test(req)
	}
}

func BenchmarkRESTCreate(b *testing.B) {
	// Setup
	cfg := getTestConfig()
	db, _ := database.NewConnection(cfg.Database)
	defer db.Close()

	server := api.NewServer(cfg, db)
	app := server.App()

	// Create table
	ctx := context.Background()
	db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS benchmark_creates (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT,
			value INTEGER
		);
	`)

	b.ResetTimer()

	// Run benchmark
	for i := 0; i < b.N; i++ {
		payload := map[string]interface{}{
			"name":  fmt.Sprintf("Item %d", i),
			"value": i,
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/rest/benchmark_creates", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		app.Test(req)
	}
}

// Helper functions

func getTestConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Address:      ":8081",
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
			BodyLimit:    10 * 1024 * 1024,
		},
		Database: config.DatabaseConfig{
			Host:            "localhost",
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
			JWTSecret:      "test-secret-key",
			JWTExpiry:      15 * time.Minute,
			RefreshExpiry:  7 * 24 * time.Hour,
			PasswordMinLen: 8,
			BcryptCost:     10,
		},
		Debug: true,
	}
}

// Run the test suite
func TestIntegrationSuite(t *testing.T) {
	// Skip if not in integration test mode
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	suite.Run(t, new(IntegrationTestSuite))
}