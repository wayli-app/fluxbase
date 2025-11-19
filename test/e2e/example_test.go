// Package e2e_test provides an example test demonstrating best practices for Fluxbase testing.
//
// This file serves as a reference for new developers learning how to write tests.
// It demonstrates:
//   - Proper test setup and teardown
//   - Given-When-Then structure
//   - Database state verification
//   - Fluent API usage
//   - Authentication patterns
package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
	test "github.com/wayli-app/fluxbase/test"
)

// TestExampleRESTCreateAndVerify demonstrates the standard pattern for testing
// a CREATE operation with proper database verification.
//
// This test shows:
//   - Setup function for test isolation
//   - Given-When-Then structure with clear comments
//   - Creating test data via API
//   - Verifying response structure
//   - Verifying database state after mutation
func TestExampleRESTCreateAndVerify(t *testing.T) {
	// GIVEN: A clean test environment with API authentication
	tc := setupExampleTest(t)
	defer tc.Close()

	// Create an API key for authentication
	apiKey := tc.CreateAPIKey("Example API Key", nil)

	// WHEN: Creating a new product via POST request
	resp := tc.NewRequest("POST", "/api/v1/tables/products").
		WithAPIKey(apiKey). // Authenticate with API key
		WithBody(map[string]interface{}{
			"name":  "Example Product",
			"price": 99.99,
		}).
		Send()

	// THEN: Product is created successfully with HTTP 201 status
	resp.AssertStatus(fiber.StatusCreated)

	// AND: Response contains the created product with ID
	var created map[string]interface{}
	resp.JSON(&created)

	require.NotNil(t, created["id"], "Response should contain product ID")
	require.Equal(t, "Example Product", created["name"], "Response should contain product name")
	require.Equal(t, 99.99, created["price"], "Response should contain product price")

	productID := created["id"]

	// AND: Product exists in database with correct values
	// This is CRITICAL - always verify database state after mutations!
	rows := tc.QuerySQL("SELECT * FROM products WHERE id = $1", productID)
	require.Len(t, rows, 1, "Product should exist in database")
	require.Equal(t, "Example Product", rows[0]["name"], "Database should have correct name")
	require.Equal(t, 99.99, rows[0]["price"], "Database should have correct price")
	require.NotNil(t, rows[0]["created_at"], "Database should have created_at timestamp")
	require.NotNil(t, rows[0]["updated_at"], "Database should have updated_at timestamp")
}

// TestExampleRESTUpdateAndVerify demonstrates testing an UPDATE operation
// with database state verification.
func TestExampleRESTUpdateAndVerify(t *testing.T) {
	// GIVEN: An existing product in the database
	tc := setupExampleTest(t)
	defer tc.Close()

	apiKey := tc.CreateAPIKey("Example API Key", nil)

	// Create initial product
	createResp := tc.NewRequest("POST", "/api/v1/tables/products").
		WithAPIKey(apiKey).
		WithBody(map[string]interface{}{
			"name":  "Original Name",
			"price": 50.00,
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	var product map[string]interface{}
	createResp.JSON(&product)
	productID := product["id"]

	// WHEN: Updating the product name and price
	updateResp := tc.NewRequest("PATCH", "/api/v1/tables/products?id=eq."+fmt.Sprintf("%v", productID)).
		WithAPIKey(apiKey).
		WithBody(map[string]interface{}{
			"name":  "Updated Name",
			"price": 75.00,
		}).
		Send()

	// THEN: Update succeeds with HTTP 200 status
	updateResp.AssertStatus(fiber.StatusOK)

	// AND: Changes are persisted in database
	rows := tc.QuerySQL("SELECT * FROM products WHERE id = $1", productID)
	require.Len(t, rows, 1, "Product should still exist")
	require.Equal(t, "Updated Name", rows[0]["name"], "Name should be updated in database")
	require.Equal(t, 75.00, rows[0]["price"], "Price should be updated in database")
}

// TestExampleRESTDeleteAndVerify demonstrates testing a DELETE operation
// with database state verification.
func TestExampleRESTDeleteAndVerify(t *testing.T) {
	// GIVEN: An existing product in the database
	tc := setupExampleTest(t)
	defer tc.Close()

	apiKey := tc.CreateAPIKey("Example API Key", nil)

	// Create product to delete
	createResp := tc.NewRequest("POST", "/api/v1/tables/products").
		WithAPIKey(apiKey).
		WithBody(map[string]interface{}{
			"name":  "Product To Delete",
			"price": 25.00,
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	var product map[string]interface{}
	createResp.JSON(&product)
	productID := product["id"]

	// Verify product exists before deletion
	rows := tc.QuerySQL("SELECT * FROM products WHERE id = $1", productID)
	require.Len(t, rows, 1, "Product should exist before deletion")

	// WHEN: Deleting the product (using batch delete with filter)
	deleteResp := tc.NewRequest("DELETE", "/api/v1/tables/products?id=eq."+fmt.Sprintf("%v", productID)).
		WithAPIKey(apiKey).
		Send()

	// THEN: Delete succeeds with HTTP 200 (batch delete returns deleted records)
	deleteResp.AssertStatus(fiber.StatusOK)

	// AND: Product is removed from database
	rows = tc.QuerySQL("SELECT * FROM products WHERE id = $1", productID)
	require.Len(t, rows, 0, "Product should be deleted from database")
}

// TestExampleAuthenticationFlow demonstrates testing a complete authentication flow
// from signup to accessing protected resources.
func TestExampleAuthenticationFlow(t *testing.T) {
	// GIVEN: A clean auth environment
	tc := setupExampleTest(t)
	defer tc.Close()

	tc.EnsureAuthSchema()
	tc.ExecuteSQL("TRUNCATE TABLE auth.users CASCADE")

	email := "example@test.com"
	password := "SecurePassword123!"

	// WHEN: User signs up with email and password
	signupResp := tc.NewRequest("POST", "/api/v1/auth/signup").
		WithBody(map[string]interface{}{
			"email":    email,
			"password": password,
		}).
		Send()

	// THEN: Signup succeeds and returns access token
	signupResp.AssertStatus(fiber.StatusCreated)

	var signupResult map[string]interface{}
	signupResp.JSON(&signupResult)

	require.NotNil(t, signupResult["access_token"], "Should return access token")
	require.NotNil(t, signupResult["refresh_token"], "Should return refresh token")
	require.NotNil(t, signupResult["user"], "Should return user object")

	accessToken := signupResult["access_token"].(string)

	// AND: User can access protected resources with token
	profileResp := tc.NewRequest("GET", "/api/v1/auth/user").
		WithAuth(accessToken). // Use JWT token for authentication
		Send()

	// THEN: Protected resource is accessible
	profileResp.AssertStatus(fiber.StatusOK)

	var user map[string]interface{}
	profileResp.JSON(&user)
	require.Equal(t, email, user["email"], "Should return correct user email")
}

// TestExampleNegativeCase demonstrates testing error handling and negative cases.
//
// Always test both success AND failure paths!
func TestExampleNegativeCase(t *testing.T) {
	// GIVEN: A test environment
	tc := setupExampleTest(t)
	defer tc.Close()

	apiKey := tc.CreateAPIKey("Example API Key", nil)

	// WHEN: Attempting to create product with invalid data (missing required field)
	resp := tc.NewRequest("POST", "/api/v1/tables/products").
		WithAPIKey(apiKey).
		WithBody(map[string]interface{}{
			// Missing "name" field (required)
			"price": 29.99,
		}).
		Send()

	// THEN: Request fails with specific error status
	// ✅ GOOD: Use specific status code
	resp.AssertStatus(fiber.StatusBadRequest)

	// ❌ BAD: Don't use permissive checks like:
	// require.True(t, resp.Status() >= 400, "Should return error")

	// AND: Error response contains meaningful error message
	var errResp map[string]interface{}
	resp.JSON(&errResp)
	require.NotNil(t, errResp["error"], "Should return error message")
}

// TestExampleDuplicateKeyError demonstrates testing duplicate key violations.
//
// The API returns 409 Conflict for unique constraint violations.
func TestExampleDuplicateKeyError(t *testing.T) {
	// GIVEN: A table with a unique constraint on name
	tc := setupExampleTest(t)
	defer tc.Close()

	apiKey := tc.CreateAPIKey("Example API Key", nil)

	// Add unique constraint for this test (drop first in case it exists from previous failed run)
	// Drop both constraint and index since UNIQUE constraints create an underlying index
	tc.ExecuteSQL("ALTER TABLE products DROP CONSTRAINT IF EXISTS products_name_key")
	tc.ExecuteSQL("DROP INDEX IF EXISTS products_name_key")
	tc.ExecuteSQL("ALTER TABLE products ADD CONSTRAINT products_name_key UNIQUE (name)")
	defer func() {
		tc.ExecuteSQL("ALTER TABLE products DROP CONSTRAINT IF EXISTS products_name_key")
		tc.ExecuteSQL("DROP INDEX IF EXISTS products_name_key")
	}()

	// Create initial product
	tc.NewRequest("POST", "/api/v1/tables/products").
		WithAPIKey(apiKey).
		WithBody(map[string]interface{}{
			"name":  "Unique Product",
			"price": 10.00,
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	// WHEN: Attempting to insert duplicate name
	resp := tc.NewRequest("POST", "/api/v1/tables/products").
		WithAPIKey(apiKey).
		WithBody(map[string]interface{}{
			"name":  "Unique Product", // Duplicate!
			"price": 15.00,
		}).
		Send()

	// THEN: Request fails with 409 Conflict (NOT 500!)
	resp.AssertStatus(fiber.StatusConflict)

	// AND: Error message explains the conflict
	var errResp map[string]interface{}
	resp.JSON(&errResp)
	require.NotNil(t, errResp["error"], "Should return error message")
}

// TestExampleInvalidDataType demonstrates testing invalid data type errors.
//
// The API returns 400 Bad Request for type mismatches.
func TestExampleInvalidDataType(t *testing.T) {
	// GIVEN: A test environment
	tc := setupExampleTest(t)
	defer tc.Close()

	apiKey := tc.CreateAPIKey("Example API Key", nil)

	// WHEN: Sending string value for numeric field
	resp := tc.NewRequest("POST", "/api/v1/tables/products").
		WithAPIKey(apiKey).
		WithBody(map[string]interface{}{
			"name":  "Test Product",
			"price": "not-a-number", // Invalid: string instead of number
		}).
		Send()

	// THEN: Request fails with 400 Bad Request (NOT 500!)
	resp.AssertStatus(fiber.StatusBadRequest)

	// AND: Error message explains the type error
	var errResp map[string]interface{}
	resp.JSON(&errResp)
	require.NotNil(t, errResp["error"], "Should return error message")
}

// TestExampleWithoutAuthentication demonstrates testing unauthenticated requests.
func TestExampleWithoutAuthentication(t *testing.T) {
	// GIVEN: A test environment
	tc := setupExampleTest(t)
	defer tc.Close()

	// WHEN: Attempting to access protected resource without authentication
	resp := tc.NewRequest("GET", "/api/v1/auth/user").
		Unauthenticated(). // Explicitly mark as unauthenticated (makes intent clear)
		Send()

	// THEN: Request is rejected with 401 Unauthorized
	resp.AssertStatus(fiber.StatusUnauthorized)

	var errResp map[string]interface{}
	resp.JSON(&errResp)
	require.NotNil(t, errResp["error"], "Should return error message")
}

// TestExampleWaitForCondition demonstrates using WaitForCondition instead of sleep.
//
// ✅ GOOD: Use polling with timeout
// ❌ BAD: Use time.Sleep()
func TestExampleWaitForCondition(t *testing.T) {
	// GIVEN: A test environment
	tc := setupExampleTest(t)
	defer tc.Close()

	apiKey := tc.CreateAPIKey("Example API Key", nil)

	// WHEN: Creating a product (simulating async operation)
	resp := tc.NewRequest("POST", "/api/v1/tables/products").
		WithAPIKey(apiKey).
		WithBody(map[string]interface{}{
			"name":  "Async Product",
			"price": 42.00,
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	var product map[string]interface{}
	resp.JSON(&product)
	productID := product["id"]

	// THEN: Wait for product to appear in database (with timeout)
	// ✅ GOOD: Poll until condition is met
	success := tc.WaitForCondition(5*time.Second, 100*time.Millisecond, func() bool {
		rows := tc.QuerySQL("SELECT * FROM products WHERE id = $1", productID)
		return len(rows) > 0
	})

	require.True(t, success, "Product should appear in database within 5 seconds")

	// ❌ BAD: Don't use hard-coded sleep:
	// time.Sleep(2 * time.Second)
}

// setupExampleTest creates a clean test context for example tests.
//
// This pattern should be used in all test files:
//  1. Create a setup function per test file
//  2. Clean relevant tables for test isolation
//  3. Return the test context
func setupExampleTest(t *testing.T) *test.TestContext {
	tc := test.NewTestContext(t)

	// Truncate products table to ensure test isolation
	// Each test starts with a clean slate
	tc.ExecuteSQL("TRUNCATE TABLE products CASCADE")

	return tc
}

// Note: Always call defer tc.Close() after creating a test context!
// This ensures proper cleanup even if the test fails.
//
// Example:
//   tc := setupExampleTest(t)
//   defer tc.Close()  // ✅ CRITICAL: Always include this
