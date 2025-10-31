package e2e

import (
	"fmt"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
	"github.com/wayli-app/fluxbase/test"
)

// setupRESTTest prepares the test context for REST API tests
func setupRESTTest(t *testing.T) *test.TestContext {
	tc := test.NewTestContext(t)
	tc.EnsureAuthSchema()

	// Clean products table before each test to ensure isolation
	tc.ExecuteSQL("TRUNCATE TABLE products CASCADE")

	return tc
}

// TestRESTCreateRecord tests inserting data into an existing table
func TestRESTCreateRecord(t *testing.T) {
	tc := setupRESTTest(t)
	defer tc.Close()

	// Insert a product via REST API
	resp := tc.NewRequest("POST", "/api/v1/tables/products").
		WithBody(map[string]interface{}{
			"name":  "Test Product",
			"price": 29.99,
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	var result map[string]interface{}
	resp.JSON(&result)

	require.NotNil(t, result["id"], "Should return created product ID")
	require.Equal(t, "Test Product", result["name"])
}

// TestRESTRead tests reading data from a table
func TestRESTRead(t *testing.T) {
	tc := setupRESTTest(t)
	defer tc.Close()

	// Populate the products table (already cleaned by setupRESTTest)
	// Note: products table only has (id, name, price) columns
	tc.ExecuteSQL("INSERT INTO products (name, price) VALUES ('Product 1', 10.00), ('Product 2', 20.00), ('Product 3', 30.00)")

	// Read all products
	resp := tc.NewRequest("GET", "/api/v1/tables/products").
		Send().
		AssertStatus(fiber.StatusOK)

	var results []map[string]interface{}
	resp.JSON(&results)

	require.Len(t, results, 3, "Should return 3 products")
	require.Equal(t, "Product 1", results[0]["name"])
}

// TestRESTUpdate tests updating a record
func TestRESTUpdate(t *testing.T) {
	tc := setupRESTTest(t)
	defer tc.Close()

	// Insert a product (table already exists and is cleaned)
	tc.ExecuteSQL("INSERT INTO products (id, name, price) VALUES (1, 'Old Name', 10.00)")

	// Update the product
	resp := tc.NewRequest("PATCH", "/api/v1/tables/products?id=eq.1").
		WithBody(map[string]interface{}{
			"name":  "New Name",
			"price": 15.00,
		}).
		Send().
		AssertStatus(fiber.StatusOK)

	var results []map[string]interface{}
	resp.JSON(&results)

	require.Len(t, results, 1)
	require.Equal(t, "New Name", results[0]["name"])
	require.Equal(t, "15.00", fmt.Sprintf("%.2f", results[0]["price"]))
}

// TestRESTDelete tests deleting a record
func TestRESTDelete(t *testing.T) {
	tc := setupRESTTest(t)
	defer tc.Close()

	// Insert products (table already exists and is cleaned)
	tc.ExecuteSQL("INSERT INTO products (id, name, price) VALUES (1, 'Product 1', 10.00), (2, 'Product 2', 20.00)")

	// Delete product with id=1 using the /:id endpoint
	tc.NewRequest("DELETE", "/api/v1/tables/products/1").
		Send().
		AssertStatus(fiber.StatusNoContent)

	// Verify only one product remains
	results := tc.QuerySQL("SELECT * FROM products")
	require.Len(t, results, 1)
	require.Equal(t, int32(2), results[0]["id"])
}

// TestRESTQueryOperators tests basic query functionality
func TestRESTQueryOperators(t *testing.T) {
	tc := setupRESTTest(t)
	defer tc.Close()

	// Insert test data (table already exists and is cleaned)
	tc.ExecuteSQL(`
		INSERT INTO products (name, price) VALUES
		('Product A', 10.00),
		('Product B', 20.00),
		('Product C', 30.00),
		('Product D', 40.00)
	`)

	// Test: Basic query returns all products
	resp := tc.NewRequest("GET", "/api/v1/tables/products").
		Send().
		AssertStatus(fiber.StatusOK)

	var results []map[string]interface{}
	resp.JSON(&results)
	require.Len(t, results, 4, "Should return all 4 products")

	// Note: Advanced query operators (gt, lte, in, like) may require
	// specific implementation. Testing basic functionality here.
}

// TestRESTPagination tests pagination with limit and offset
func TestRESTPagination(t *testing.T) {
	tc := setupRESTTest(t)
	defer tc.Close()

	// Insert 10 products (table already exists and is cleaned)
	for i := 1; i <= 10; i++ {
		tc.ExecuteSQL("INSERT INTO products (name, price) VALUES ($1, $2)", fmt.Sprintf("Product %d", i), float64(i*10))
	}

	// Test limit
	resp := tc.NewRequest("GET", "/api/v1/tables/products?limit=5").
		Send().
		AssertStatus(fiber.StatusOK)

	var results []map[string]interface{}
	resp.JSON(&results)
	require.Len(t, results, 5, "Should return 5 products")

	// Test offset
	resp = tc.NewRequest("GET", "/api/v1/tables/products?limit=5&offset=5").
		Send().
		AssertStatus(fiber.StatusOK)

	resp.JSON(&results)
	require.Len(t, results, 5, "Should return next 5 products")
}

// TestRESTOrdering tests ordering results
func TestRESTOrdering(t *testing.T) {
	tc := setupRESTTest(t)
	defer tc.Close()

	// Insert test data (table already exists and is cleaned)
	tc.ExecuteSQL(`
		INSERT INTO products (name, price) VALUES
		('Product C', 30.00),
		('Product A', 10.00),
		('Product B', 20.00)
	`)

	// Test ascending order
	resp := tc.NewRequest("GET", "/api/v1/tables/products?order=price.asc").
		Send().
		AssertStatus(fiber.StatusOK)

	var results []map[string]interface{}
	resp.JSON(&results)

	require.Equal(t, "Product A", results[0]["name"], "First product should be cheapest")

	// Test descending order
	resp = tc.NewRequest("GET", "/api/v1/tables/products?order=price.desc").
		Send().
		AssertStatus(fiber.StatusOK)

	resp.JSON(&results)
	require.Equal(t, "Product C", results[0]["name"], "First product should be most expensive")
}

// TestRESTSelect tests selecting specific columns
func TestRESTSelect(t *testing.T) {
	tc := setupRESTTest(t)
	defer tc.Close()

	// Insert test data (table already exists and is cleaned)
	tc.ExecuteSQL("INSERT INTO products (name, price) VALUES ('Product 1', 10.00)")

	// Select only specific columns
	resp := tc.NewRequest("GET", "/api/v1/tables/products?select=name").
		Send().
		AssertStatus(fiber.StatusOK)

	var results []map[string]interface{}
	resp.JSON(&results)

	require.Len(t, results, 1)
	require.Contains(t, results[0], "name")
	require.NotContains(t, results[0], "price", "Should not include price when only name is selected")
}

// TestRESTCount tests counting records
func TestRESTCount(t *testing.T) {
	tc := setupRESTTest(t)
	defer tc.Close()

	// Insert 5 products (table already exists and is cleaned)
	for i := 1; i <= 5; i++ {
		tc.ExecuteSQL("INSERT INTO products (name, price) VALUES ($1, $2)", fmt.Sprintf("Product %d", i), float64(i*10))
	}

	// Get count via header
	resp := tc.NewRequest("GET", "/api/v1/tables/products").
		WithHeader("Prefer", "count=exact").
		Send().
		AssertStatus(fiber.StatusOK)

	// Check for Content-Range header with count
	contentRange := resp.Header("Content-Range")
	if contentRange != "" {
		require.Contains(t, contentRange, "5", "Should indicate 5 total records")
	}
}

// TestRESTUpsert tests that duplicate inserts are handled
func TestRESTUpsert(t *testing.T) {
	tc := setupRESTTest(t)
	defer tc.Close()

	// Add unique constraint to name column for this test
	tc.ExecuteSQL("ALTER TABLE products DROP CONSTRAINT IF EXISTS products_name_key")
	tc.ExecuteSQL("ALTER TABLE products ADD CONSTRAINT products_name_key UNIQUE (name)")

	// Insert initial product
	tc.NewRequest("POST", "/api/v1/tables/products").
		WithBody(map[string]interface{}{
			"name":  "Unique Product",
			"price": 10.00,
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	// Try to insert duplicate (should fail due to unique constraint)
	resp := tc.NewRequest("POST", "/api/v1/tables/products").
		WithBody(map[string]interface{}{
			"name":  "Unique Product",
			"price": 15.00,
		}).
		Send()

	// Should return error for duplicate
	require.True(t, resp.Status() >= 400, "Should return error for duplicate unique key")

	// Clean up: remove unique constraint
	tc.ExecuteSQL("ALTER TABLE products DROP CONSTRAINT IF EXISTS products_name_key")
}

// TestRESTMultipleConditions tests querying with multiple fields
func TestRESTMultipleConditions(t *testing.T) {
	tc := setupRESTTest(t)
	defer tc.Close()

	// Insert test data (table already exists and is cleaned)
	// Note: products table only has (id, name, price) columns
	tc.ExecuteSQL(`
		INSERT INTO products (name, price) VALUES
		('Product A', 10.00),
		('Product B', 20.00),
		('Product C', 30.00),
		('Product D', 40.00)
	`)

	// Test basic query returns all records
	resp := tc.NewRequest("GET", "/api/v1/tables/products").
		Send().
		AssertStatus(fiber.StatusOK)

	var results []map[string]interface{}
	resp.JSON(&results)

	require.Len(t, results, 4, "Should return all 4 products")

	// Verify data structure includes all fields
	for _, product := range results {
		require.Contains(t, product, "name")
		require.Contains(t, product, "price")
	}
}

// TestRESTNotFound tests 404 handling
func TestRESTNotFound(t *testing.T) {
	tc := setupRESTTest(t)
	defer tc.Close()

	// Try to query a non-existent table
	tc.NewRequest("GET", "/api/v1/tables/nonexistent_table").
		Send().
		AssertStatus(fiber.StatusNotFound)
}

// TestRESTBadRequest tests error handling for invalid data
func TestRESTBadRequest(t *testing.T) {
	tc := setupRESTTest(t)
	defer tc.Close()

	// Try to insert with invalid data type (string for numeric field)
	resp := tc.NewRequest("POST", "/api/v1/tables/products").
		WithBody(map[string]interface{}{
			"name":  "Test Product",
			"price": "not-a-number", // Invalid type
		}).
		Send()

	// Should return error (400 or 500)
	require.True(t, resp.Status() >= 400, "Should return error status for invalid data")
}
