package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

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

// TestComprehensiveRESTAPI tests all REST API operations end-to-end
func TestComprehensiveRESTAPI(t *testing.T) {
	// Setup test database and server
	cfg := setupTestConfig(t)
	db := setupTestDatabase(t, cfg)
	defer db.Close()

	// Create test schema and tables
	setupTestSchema(t, db)

	// Create Fiber app with all middleware
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Setup middleware
	jwtSecret := cfg.JWTSecret
	app.Use(middleware.AuthMiddleware(middleware.AuthConfig{
		JWTSecret: jwtSecret,
		Optional:  true,
	}))
	app.Use(middleware.RLSMiddleware(middleware.RLSConfig{
		DB:      db,
		Enabled: true,
	}))

	// Setup API routes
	apiServer := api.NewServer(db, cfg)
	apiServer.RegisterRoutes(app)

	// Run test suites
	t.Run("CRUD Operations", func(t *testing.T) {
		testCRUDOperations(t, app, db, jwtSecret)
	})

	t.Run("Query Operators", func(t *testing.T) {
		testQueryOperators(t, app, db, jwtSecret)
	})

	t.Run("Full-Text Search", func(t *testing.T) {
		testFullTextSearch(t, app, db, jwtSecret)
	})

	t.Run("JSONB Operators", func(t *testing.T) {
		testJSONBOperators(t, app, db, jwtSecret)
	})

	t.Run("Array Operators", func(t *testing.T) {
		testArrayOperators(t, app, db, jwtSecret)
	})

	t.Run("Aggregations", func(t *testing.T) {
		testAggregations(t, app, db, jwtSecret)
	})

	t.Run("Upsert Operations", func(t *testing.T) {
		testUpsertOperations(t, app, db, jwtSecret)
	})

	t.Run("Batch Operations", func(t *testing.T) {
		testBatchOperations(t, app, db, jwtSecret)
	})

	t.Run("Views and RPC", func(t *testing.T) {
		testViewsAndRPC(t, app, db, jwtSecret)
	})

	t.Run("Row-Level Security", func(t *testing.T) {
		testRowLevelSecurity(t, app, db, jwtSecret)
	})

	t.Run("Authentication Context", func(t *testing.T) {
		testAuthenticationContext(t, app, db, jwtSecret)
	})
}

// setupTestConfig creates a test configuration
func setupTestConfig(t *testing.T) *config.Config {
	return &config.Config{
		DatabaseURL: "postgres://postgres:postgres@localhost:5432/fluxbase_test?sslmode=disable",
		JWTSecret:   "test-secret-key-for-testing-only",
		Port:        "8080",
	}
}

// setupTestDatabase creates a test database connection
func setupTestDatabase(t *testing.T, cfg *config.Config) *database.Connection {
	db, err := database.Connect(cfg.DatabaseURL)
	require.NoError(t, err, "Failed to connect to test database")
	return db
}

// setupTestSchema creates test tables and data
func setupTestSchema(t *testing.T, db *database.Connection) {
	ctx := context.Background()

	// Create test tables
	queries := []string{
		// Products table for testing CRUD
		`CREATE TABLE IF NOT EXISTS products (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			price NUMERIC(10, 2) NOT NULL,
			stock INT NOT NULL DEFAULT 0,
			tags TEXT[],
			metadata JSONB,
			search_vector tsvector,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,

		// Create index for full-text search
		`CREATE INDEX IF NOT EXISTS products_search_idx ON products USING GIN(search_vector)`,

		// Orders table for testing relationships
		`CREATE TABLE IF NOT EXISTS orders (
			id SERIAL PRIMARY KEY,
			user_id UUID NOT NULL,
			product_id INT REFERENCES products(id),
			quantity INT NOT NULL,
			total NUMERIC(10, 2) NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,

		// Tasks table for testing RLS
		`CREATE TABLE IF NOT EXISTS tasks (
			id SERIAL PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			completed BOOLEAN DEFAULT FALSE,
			user_id UUID NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,

		// Enable RLS on tasks
		`ALTER TABLE tasks ENABLE ROW LEVEL SECURITY`,

		// RLS policy: users can only see their own tasks
		`DROP POLICY IF EXISTS tasks_user_policy ON tasks`,
		`CREATE POLICY tasks_user_policy ON tasks
			FOR ALL
			USING (user_id::text = current_setting('app.user_id', true))
			WITH CHECK (user_id::text = current_setting('app.user_id', true))`,

		// Create a view for testing
		`CREATE OR REPLACE VIEW product_summary AS
			SELECT
				p.id,
				p.name,
				p.price,
				COUNT(o.id) as order_count,
				SUM(o.quantity) as total_sold
			FROM products p
			LEFT JOIN orders o ON p.id = o.product_id
			GROUP BY p.id, p.name, p.price`,

		// Create an RPC function for testing
		`CREATE OR REPLACE FUNCTION get_product_stats(product_id_param INT)
		RETURNS TABLE(product_id INT, name TEXT, total_orders BIGINT, total_revenue NUMERIC) AS $$
		BEGIN
			RETURN QUERY
			SELECT
				p.id as product_id,
				p.name,
				COUNT(o.id) as total_orders,
				SUM(o.total) as total_revenue
			FROM products p
			LEFT JOIN orders o ON p.id = o.product_id
			WHERE p.id = product_id_param
			GROUP BY p.id, p.name;
		END;
		$$ LANGUAGE plpgsql SECURITY DEFINER`,
	}

	for _, query := range queries {
		_, err := db.Pool().Exec(ctx, query)
		require.NoError(t, err, "Failed to execute schema query: %s", query)
	}

	// Clean up any existing test data
	cleanupQueries := []string{
		"TRUNCATE orders CASCADE",
		"TRUNCATE products CASCADE",
		"TRUNCATE tasks CASCADE",
	}

	for _, query := range cleanupQueries {
		_, err := db.Pool().Exec(ctx, query)
		require.NoError(t, err, "Failed to cleanup test data")
	}
}

// testCRUDOperations tests Create, Read, Update, Delete operations
func testCRUDOperations(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	// Create a test user and get JWT token
	token := createTestUserAndToken(t, db, jwtSecret)

	// Test CREATE (POST)
	t.Run("Create Product", func(t *testing.T) {
		product := map[string]interface{}{
			"name":        "Test Product",
			"description": "A test product",
			"price":       29.99,
			"stock":       100,
			"tags":        []string{"test", "demo"},
			"metadata":    map[string]interface{}{"color": "blue", "size": "large"},
		}

		body, _ := json.Marshal(product)
		req := httptest.NewRequest("POST", "/api/v1/rest/products", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "Test Product", result["name"])
		assert.Equal(t, 29.99, result["price"])
	})

	// Test READ (GET)
	t.Run("List Products", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/rest/products", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var results []map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&results)
		assert.GreaterOrEqual(t, len(results), 1)
	})

	// Test READ by ID (GET)
	t.Run("Get Product by ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/rest/products/1", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.NotNil(t, result["id"])
	})

	// Test UPDATE (PUT)
	t.Run("Update Product", func(t *testing.T) {
		updates := map[string]interface{}{
			"name":  "Updated Product",
			"price": 39.99,
			"stock": 150,
		}

		body, _ := json.Marshal(updates)
		req := httptest.NewRequest("PUT", "/api/v1/rest/products/1", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "Updated Product", result["name"])
		assert.Equal(t, 39.99, result["price"])
	})

	// Test PATCH (partial update)
	t.Run("Patch Product", func(t *testing.T) {
		updates := map[string]interface{}{
			"stock": 200,
		}

		body, _ := json.Marshal(updates)
		req := httptest.NewRequest("PATCH", "/api/v1/rest/products/1", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, float64(200), result["stock"])
	})

	// Test DELETE
	t.Run("Delete Product", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/rest/products/1", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		// Verify deletion
		req = httptest.NewRequest("GET", "/api/v1/rest/products/1", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err = app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

// testQueryOperators tests all query filter operators
func testQueryOperators(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	token := createTestUserAndToken(t, db, jwtSecret)

	// Insert test data
	setupProductTestData(t, app, token)

	tests := []struct {
		name          string
		query         string
		expectedCount int
	}{
		{"Equal (eq)", "?price=eq.29.99", 1},
		{"Not Equal (neq)", "?price=neq.29.99", 2},
		{"Greater Than (gt)", "?price=gt.30", 1},
		{"Greater or Equal (gte)", "?price=gte.29.99", 3},
		{"Less Than (lt)", "?price=lt.30", 2},
		{"Less or Equal (lte)", "?price=lte.29.99", 2},
		{"Like", "?name=like.*Product*", 3},
		{"ILike (case-insensitive)", "?name=ilike.*product*", 3},
		{"In", "?price=in.(29.99,39.99)", 2},
		{"Is Null", "?description=is.null", 0},
		{"Is Not Null", "?description=not.is.null", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/rest/products"+tt.query, nil)
			req.Header.Set("Authorization", "Bearer "+token)

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var results []map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&results)
			assert.Equal(t, tt.expectedCount, len(results), "Query: %s", tt.query)
		})
	}
}

// testFullTextSearch tests full-text search capabilities
func testFullTextSearch(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	token := createTestUserAndToken(t, db, jwtSecret)

	// Update search vectors
	ctx := context.Background()
	_, err := db.Pool().Exec(ctx, `
		UPDATE products
		SET search_vector = to_tsvector('english', coalesce(name, '') || ' ' || coalesce(description, ''))
	`)
	require.NoError(t, err)

	t.Run("FTS - Full Text Search", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/rest/products?search_vector=fts.product", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var results []map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&results)
		assert.GreaterOrEqual(t, len(results), 1)
	})
}

// testJSONBOperators tests JSONB query operators
func testJSONBOperators(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	token := createTestUserAndToken(t, db, jwtSecret)

	t.Run("JSONB Contains (cs)", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/rest/products?metadata=cs.{\"color\":\"blue\"}", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var results []map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&results)
		assert.GreaterOrEqual(t, len(results), 0)
	})

	t.Run("JSONB Contained By (cd)", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/rest/products?metadata=cd.{\"color\":\"blue\",\"size\":\"large\",\"extra\":\"data\"}", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// testArrayOperators tests array query operators
func testArrayOperators(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	token := createTestUserAndToken(t, db, jwtSecret)

	t.Run("Array Overlap (ov)", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/rest/products?tags=ov.{test,demo}", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var results []map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&results)
		assert.GreaterOrEqual(t, len(results), 1)
	})

	t.Run("Array Contains (cs)", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/rest/products?tags=cs.{test}", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// testAggregations tests GROUP BY and aggregation functions
func testAggregations(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	token := createTestUserAndToken(t, db, jwtSecret)

	// Create orders for aggregation
	setupOrderTestData(t, app, token, db)

	t.Run("Count Aggregation", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/rest/orders?select=status,count&group_by=status", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var results []map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&results)
		assert.GreaterOrEqual(t, len(results), 1)
	})

	t.Run("Sum Aggregation", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/rest/orders?select=status,sum(total)&group_by=status", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Avg, Min, Max Aggregations", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/rest/orders?select=status,avg(total),min(total),max(total)&group_by=status", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// testUpsertOperations tests upsert (INSERT ... ON CONFLICT DO UPDATE)
func testUpsertOperations(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	token := createTestUserAndToken(t, db, jwtSecret)

	t.Run("Upsert - Insert New", func(t *testing.T) {
		product := map[string]interface{}{
			"id":    999,
			"name":  "Upsert Product",
			"price": 49.99,
			"stock": 50,
		}

		body, _ := json.Marshal(product)
		req := httptest.NewRequest("POST", "/api/v1/rest/products?on_conflict=id", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Prefer", "resolution=merge-duplicates")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("Upsert - Update Existing", func(t *testing.T) {
		product := map[string]interface{}{
			"id":    999,
			"name":  "Updated Upsert Product",
			"price": 59.99,
			"stock": 75,
		}

		body, _ := json.Marshal(product)
		req := httptest.NewRequest("POST", "/api/v1/rest/products?on_conflict=id", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Prefer", "resolution=merge-duplicates")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated)

		// Verify update
		req = httptest.NewRequest("GET", "/api/v1/rest/products/999", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err = app.Test(req)
		require.NoError(t, err)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "Updated Upsert Product", result["name"])
	})
}

// testBatchOperations tests batch insert, update, delete
func testBatchOperations(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	token := createTestUserAndToken(t, db, jwtSecret)

	t.Run("Batch Insert", func(t *testing.T) {
		products := []map[string]interface{}{
			{"name": "Batch Product 1", "price": 10.00, "stock": 10},
			{"name": "Batch Product 2", "price": 20.00, "stock": 20},
			{"name": "Batch Product 3", "price": 30.00, "stock": 30},
		}

		body, _ := json.Marshal(products)
		req := httptest.NewRequest("POST", "/api/v1/rest/products", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var results []map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&results)
		assert.Equal(t, 3, len(results))
	})

	t.Run("Batch Update", func(t *testing.T) {
		updates := map[string]interface{}{
			"stock": 100,
		}

		body, _ := json.Marshal(updates)
		req := httptest.NewRequest("PATCH", "/api/v1/rest/products?name=like.Batch*", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Batch Delete", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/rest/products?name=like.Batch*", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})
}

// testViewsAndRPC tests database views and RPC functions
func testViewsAndRPC(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	token := createTestUserAndToken(t, db, jwtSecret)

	t.Run("Query View", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/rest/product_summary", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var results []map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&results)
		assert.GreaterOrEqual(t, len(results), 0)
	})

	t.Run("Call RPC Function", func(t *testing.T) {
		params := map[string]interface{}{
			"product_id_param": 1,
		}

		body, _ := json.Marshal(params)
		req := httptest.NewRequest("POST", "/api/v1/rpc/get_product_stats", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound)
	})
}

// testRowLevelSecurity tests RLS policy enforcement
func testRowLevelSecurity(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	// Create two users with different IDs
	user1ID, token1 := createTestUserWithIDAndToken(t, db, jwtSecret, "user1@test.com")
	user2ID, token2 := createTestUserWithIDAndToken(t, db, jwtSecret, "user2@test.com")

	// User 1 creates tasks
	t.Run("User 1 Creates Tasks", func(t *testing.T) {
		tasks := []map[string]interface{}{
			{"title": "User 1 Task 1", "user_id": user1ID},
			{"title": "User 1 Task 2", "user_id": user1ID},
		}

		for _, task := range tasks {
			body, _ := json.Marshal(task)
			req := httptest.NewRequest("POST", "/api/v1/rest/tasks", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token1)

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusCreated, resp.StatusCode)
		}
	})

	// User 2 creates tasks
	t.Run("User 2 Creates Tasks", func(t *testing.T) {
		task := map[string]interface{}{
			"title":   "User 2 Task 1",
			"user_id": user2ID,
		}

		body, _ := json.Marshal(task)
		req := httptest.NewRequest("POST", "/api/v1/rest/tasks", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token2)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	// User 1 can only see their own tasks (RLS enforcement)
	t.Run("User 1 Sees Only Their Tasks", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/rest/tasks", nil)
		req.Header.Set("Authorization", "Bearer "+token1)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var results []map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&results)

		// Should only see 2 tasks (their own)
		assert.Equal(t, 2, len(results))
		for _, task := range results {
			assert.Equal(t, user1ID, task["user_id"])
		}
	})

	// User 2 can only see their own tasks
	t.Run("User 2 Sees Only Their Tasks", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/rest/tasks", nil)
		req.Header.Set("Authorization", "Bearer "+token2)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var results []map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&results)

		// Should only see 1 task (their own)
		assert.Equal(t, 1, len(results))
		assert.Equal(t, user2ID, results[0]["user_id"])
	})

	// Anonymous user sees no tasks
	t.Run("Anonymous User Sees No Tasks", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/rest/tasks", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var results []map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&results)
		assert.Equal(t, 0, len(results))
	})
}

// testAuthenticationContext tests that authentication context is properly passed
func testAuthenticationContext(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	userID, token := createTestUserWithIDAndToken(t, db, jwtSecret, "authtest@test.com")

	t.Run("Authenticated Request Has User Context", func(t *testing.T) {
		task := map[string]interface{}{
			"title":   "Auth Context Test",
			"user_id": userID,
		}

		body, _ := json.Marshal(task)
		req := httptest.NewRequest("POST", "/api/v1/rest/tasks", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, userID, result["user_id"])
	})

	t.Run("Unauthenticated Request Has No User Context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/rest/products", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		// Should still return OK (anonymous access allowed for products)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// Helper functions

func createTestUserAndToken(t *testing.T, db *database.Connection, jwtSecret string) string {
	userID, token := createTestUserWithIDAndToken(t, db, jwtSecret, "test@example.com")
	_ = userID
	return token
}

func createTestUserWithIDAndToken(t *testing.T, db *database.Connection, jwtSecret string, email string) (string, string) {
	ctx := context.Background()

	// Create user in database
	userID := uuid.New().String()
	_, err := db.Pool().Exec(ctx, `
		INSERT INTO auth.users (id, email, encrypted_password, email_confirmed_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (email) DO NOTHING
	`, userID, email, "$2a$10$FAKEHASH")
	require.NoError(t, err)

	// Generate JWT token
	authService := auth.NewService(db, jwtSecret, "smtp://fake", "noreply@test.com")
	token, err := authService.GenerateJWT(userID, email)
	require.NoError(t, err)

	return userID, token
}

func setupProductTestData(t *testing.T, app *fiber.App, token string) {
	products := []map[string]interface{}{
		{"name": "Product A", "description": "First product", "price": 29.99, "stock": 100},
		{"name": "Product B", "description": "Second product", "price": 19.99, "stock": 50},
		{"name": "Product C", "description": "Third product", "price": 39.99, "stock": 75},
	}

	for _, product := range products {
		body, _ := json.Marshal(product)
		req := httptest.NewRequest("POST", "/api/v1/rest/products", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		_, _ = app.Test(req)
	}
}

func setupOrderTestData(t *testing.T, app *fiber.App, token string, db *database.Connection) {
	userID := uuid.New()

	orders := []map[string]interface{}{
		{"user_id": userID, "product_id": 1, "quantity": 2, "total": 59.98, "status": "pending"},
		{"user_id": userID, "product_id": 2, "quantity": 1, "total": 19.99, "status": "completed"},
		{"user_id": userID, "product_id": 3, "quantity": 3, "total": 119.97, "status": "pending"},
	}

	for _, order := range orders {
		body, _ := json.Marshal(order)
		req := httptest.NewRequest("POST", "/api/v1/rest/orders", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		_, _ = app.Test(req)
	}
}
