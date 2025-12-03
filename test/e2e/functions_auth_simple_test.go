package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/fluxbase-eu/fluxbase/test"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

// TestFunctionAuthenticationOnly tests ONLY the authentication logic without function execution
// This test verifies that the auth middleware correctly accepts/rejects different auth types
func TestFunctionAuthenticationOnly(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()
	tc.EnsureAuthSchema()

	// Create admin user for function management
	timestamp := time.Now().UnixNano()
	email := fmt.Sprintf("admin-%d@test.com", timestamp)
	password := "adminpass123456"
	_, adminToken := tc.CreateDashboardAdminUser(email, password)

	// Create a test function WITHOUT allow_unauthenticated
	functionName := fmt.Sprintf("test_auth_%d", timestamp)
	createResp := tc.NewRequest("POST", "/api/v1/functions").
		WithAuth(adminToken).
		WithJSON(map[string]interface{}{
			"name":                  functionName,
			"code":                  `export default async function handler(req) { return new Response("ok"); }`,
			"runtime":               "deno",
			"enabled":               true,
			"allow_unauthenticated": false, // Require auth
		}).
		Send()
	require.Equal(t, fiber.StatusCreated, createResp.Status(), "Failed to create function: %s", string(createResp.Body()))

	t.Run("NoAuth_Returns401", func(t *testing.T) {
		resp := tc.NewRequest("POST", fmt.Sprintf("/api/v1/functions/%s/invoke", functionName)).
			Unauthenticated().
			Send()

		// Should return 401 BEFORE trying to execute the function
		require.Equal(t, fiber.StatusUnauthorized, resp.Status(),
			"Expected 401 when invoking function without authentication")

		body := string(resp.Body())
		require.Contains(t, body, "anon key", "Error should mention anon key requirement")
	})

	t.Run("AnonKey_PassesAuth", func(t *testing.T) {
		anonKey := tc.GenerateAnonKey()

		resp := tc.NewRequest("POST", fmt.Sprintf("/api/v1/functions/%s/invoke", functionName)).
			WithAuth(anonKey).
			Send()

		// Should NOT return 401/403 - auth passed (may return 500 if Deno not installed, but that's after auth)
		require.NotEqual(t, fiber.StatusUnauthorized, resp.Status(),
			"Should not return 401 with valid anon key")
		require.NotEqual(t, fiber.StatusForbidden, resp.Status(),
			"Should not return 403 with valid anon key")
	})

	t.Run("UserJWT_PassesAuth", func(t *testing.T) {
		userEmail := fmt.Sprintf("user-%d@test.com", timestamp)
		_, userToken := tc.CreateUser(userEmail, "password123")

		resp := tc.NewRequest("POST", fmt.Sprintf("/api/v1/functions/%s/invoke", functionName)).
			WithAuth(userToken).
			Send()

		// Should NOT return 401/403 - auth passed
		require.NotEqual(t, fiber.StatusUnauthorized, resp.Status(),
			"Should not return 401 with valid user JWT")
		require.NotEqual(t, fiber.StatusForbidden, resp.Status(),
			"Should not return 403 with valid user JWT")
	})

	t.Run("APIKey_PassesAuth", func(t *testing.T) {
		// Create API key using the helper
		apiKey := tc.CreateAPIKey("test-key", []string{"execute:functions"})

		resp := tc.NewRequest("POST", fmt.Sprintf("/api/v1/functions/%s/invoke", functionName)).
			WithHeader("X-API-Key", apiKey).
			Send()

		// Should NOT return 401/403 - auth passed
		require.NotEqual(t, fiber.StatusUnauthorized, resp.Status(),
			"Should not return 401 with valid API key")
		require.NotEqual(t, fiber.StatusForbidden, resp.Status(),
			"Should not return 403 with valid API key")
	})
}

// TestFunctionUnauthenticatedFlag tests that allow_unauthenticated flag works
func TestFunctionUnauthenticatedFlag(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()
	tc.EnsureAuthSchema()

	// Create admin user
	timestamp := time.Now().UnixNano()
	email := fmt.Sprintf("admin-%d@test.com", timestamp)
	password := "adminpass123456"
	_, adminToken := tc.CreateDashboardAdminUser(email, password)

	// Create function WITH allow_unauthenticated=true
	functionName := fmt.Sprintf("test_public_%d", timestamp)
	createResp := tc.NewRequest("POST", "/api/v1/functions").
		WithAuth(adminToken).
		WithJSON(map[string]interface{}{
			"name":                  functionName,
			"code":                  `export default async function handler(req) { return new Response("ok"); }`,
			"runtime":               "deno",
			"enabled":               true,
			"allow_unauthenticated": true, // Allow unauthenticated
		}).
		Send()
	require.Equal(t, fiber.StatusCreated, createResp.Status(), "Failed to create function")

	t.Run("NoAuth_PassesWithFlag", func(t *testing.T) {
		resp := tc.NewRequest("POST", fmt.Sprintf("/api/v1/functions/%s/invoke", functionName)).
			Unauthenticated().
			Send()

		// Should NOT return 401 - function allows unauthenticated access
		// May return 500 if Deno not installed, but not 401
		require.NotEqual(t, fiber.StatusUnauthorized, resp.Status(),
			"Should not return 401 for function with allow_unauthenticated=true")
	})
}
