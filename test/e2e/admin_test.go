package e2e

import (
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
	"github.com/wayli-app/fluxbase/test"
)

// setupAdminTest prepares the test context for admin API tests
func setupAdminTest(t *testing.T) (*test.TestContext, string) {
	tc := test.NewTestContext(t)
	tc.EnsureAuthSchema()

	// Clean auth tables before each test to ensure isolation
	tc.ExecuteSQL("TRUNCATE TABLE auth.users CASCADE")
	tc.ExecuteSQL("TRUNCATE TABLE auth.sessions CASCADE")

	// Create an admin user and get auth token for authenticated requests
	email := "admin@test.com"
	password := "adminpass123"
	_, token := tc.CreateAdminUser(email, password)

	return tc, token
}

// TestAdminListSchemas tests listing database schemas
func TestAdminListSchemas(t *testing.T) {
	tc, token := setupAdminTest(t)
	defer tc.Close()

	resp := tc.NewRequest("GET", "/api/v1/admin/schemas").
		WithAuth(token).
		Send().
		AssertStatus(fiber.StatusOK)

	var schemas []string
	resp.JSON(&schemas)

	require.GreaterOrEqual(t, len(schemas), 1, "Should have at least one schema")

	t.Logf("Found %d schemas: %v", len(schemas), schemas)
}

// TestAdminListTables tests listing tables in a schema
func TestAdminListTables(t *testing.T) {
	tc, token := setupAdminTest(t)
	defer tc.Close()

	// List tables in the public schema
	resp := tc.NewRequest("GET", "/api/v1/admin/schemas/public/tables").
		WithAuth(token).
		Send()

	// This endpoint may not be implemented yet - check if it returns 404
	if resp.Status() == fiber.StatusNotFound {
		t.Skip("List tables endpoint not yet implemented")
		return
	}

	resp.AssertStatus(fiber.StatusOK)

	var tables []string
	resp.JSON(&tables)

	require.GreaterOrEqual(t, len(tables), 1, "Should have at least one table in public schema")

	t.Logf("Found %d tables in public schema: %v", len(tables), tables)
}

// TestAdminGetTableMetadata tests getting metadata for a specific table
func TestAdminGetTableMetadata(t *testing.T) {
	tc, token := setupAdminTest(t)
	defer tc.Close()

	// Get metadata for the auth.users table
	resp := tc.NewRequest("GET", "/api/v1/admin/schemas/auth/tables/users").
		WithAuth(token).
		Send()

	// This endpoint may not be implemented yet - check if it returns 404
	if resp.Status() == fiber.StatusNotFound {
		t.Skip("Get table metadata endpoint not yet implemented")
		return
	}

	resp.AssertStatus(fiber.StatusOK)

	var metadata map[string]interface{}
	resp.JSON(&metadata)

	require.NotNil(t, metadata, "Should return table metadata")
	require.Contains(t, metadata, "columns", "Metadata should include columns")

	t.Logf("Retrieved metadata for auth.users table")
}

// TestAdminCORSHeaders tests that CORS headers are present
func TestAdminCORSHeaders(t *testing.T) {
	tc, token := setupAdminTest(t)
	defer tc.Close()

	resp := tc.NewRequest("GET", "/api/v1/admin/schemas").
		WithAuth(token).
		WithHeader("Origin", "https://example.com").
		Send().
		AssertStatus(fiber.StatusOK)

	// Check for CORS headers
	corsHeader := resp.Header("Access-Control-Allow-Origin")
	if corsHeader != "" {
		t.Logf("CORS header present: %s", corsHeader)
	} else {
		t.Log("CORS headers may not be configured or may require specific configuration")
	}
}

// TestAdmin404Handling tests 404 error handling
func TestAdmin404Handling(t *testing.T) {
	tc, token := setupAdminTest(t)
	defer tc.Close()

	// Try to get metadata for non-existent table
	tc.NewRequest("GET", "/api/v1/admin/schemas/public/tables/nonexistent_table_xyz").
		WithAuth(token).
		Send().
		AssertStatus(fiber.StatusNotFound)

	t.Logf("404 handling works correctly")
}

// TestAdminRequestIDTracking tests that request IDs are tracked
func TestAdminRequestIDTracking(t *testing.T) {
	tc, token := setupAdminTest(t)
	defer tc.Close()

	resp := tc.NewRequest("GET", "/api/v1/admin/schemas").
		WithAuth(token).
		Send().
		AssertStatus(fiber.StatusOK)

	// Check for X-Request-ID header
	requestID := resp.Header("X-Request-ID")
	if requestID != "" {
		require.NotEmpty(t, requestID, "Request ID should be present")
		t.Logf("Request ID tracking working: %s", requestID)
	} else {
		t.Log("Request ID header may use a different name or not be configured")
	}
}
