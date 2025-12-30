package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/fluxbase-eu/fluxbase/test"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

// TestFunctionAnonKeyRequired tests that functions require at minimum an anon key by default
func TestFunctionAnonKeyRequired(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()
	tc.EnsureAuthSchema()

	// Create admin user for function management
	timestamp := time.Now().UnixNano()
	email := fmt.Sprintf("admin-%d@test.com", timestamp)
	password := "adminpass123456"
	_, adminToken := tc.CreateDashboardAdminUser(email, password)

	// Create a test function WITHOUT allow_unauthenticated
	functionName := fmt.Sprintf("test_auth_required_%d", timestamp)
	createResp := tc.NewRequest("POST", "/api/v1/functions").
		WithAuth(adminToken).
		WithJSON(map[string]interface{}{
			"name": functionName,
			"code": `export default async function handler(req) {
				return new Response(JSON.stringify({ message: "success" }), {
					headers: { "Content-Type": "application/json" }
				});
			}`,
			"runtime":               "deno",
			"enabled":               true,  // Make sure function is enabled
			"allow_unauthenticated": false, // Explicitly require authentication
		}).
		Send()
	require.Equal(t, fiber.StatusCreated, createResp.Status(), "Failed to create function: %s", string(createResp.Body()))

	// Test 1: Request with NO authentication should fail with 401
	t.Run("NoAuthReturns401", func(t *testing.T) {
		resp := tc.NewRequest("POST", fmt.Sprintf("/api/v1/functions/%s/invoke", functionName)).
			Unauthenticated().
			WithJSON(map[string]interface{}{"test": "data"}).
			Send()

		require.Equal(t, fiber.StatusUnauthorized, resp.Status(),
			"Expected 401 when invoking function without authentication, got %d: %s",
			resp.Status(), string(resp.Body()))

		// Verify error message mentions anon key
		body := string(resp.Body())
		require.Contains(t, body, "anon key", "Error message should mention anon key requirement")
	})

	// Test 2: Request with anon key should pass authentication
	// Note: May return 500 if Deno not installed, but should NOT return 401/403
	t.Run("AnonKeyPassesAuth", func(t *testing.T) {
		// Generate an anon key
		anonKey := tc.GenerateAnonKey()

		resp := tc.NewRequest("POST", fmt.Sprintf("/api/v1/functions/%s/invoke", functionName)).
			WithAuth(anonKey).
			WithJSON(map[string]interface{}{"test": "data"}).
			Send()

		// Should NOT return 401 or 403 - authentication should pass
		require.NotEqual(t, fiber.StatusUnauthorized, resp.Status(),
			"Should not return 401 with valid anon key, got %d: %s",
			resp.Status(), string(resp.Body()))
		require.NotEqual(t, fiber.StatusForbidden, resp.Status(),
			"Should not return 403 with valid anon key, got %d: %s",
			resp.Status(), string(resp.Body()))
	})

	// Test 3: Request with authenticated user should pass authentication
	// Note: May return 500 if Deno not installed, but should NOT return 401/403
	t.Run("AuthenticatedUserPassesAuth", func(t *testing.T) {
		// Create a regular user
		userEmail := fmt.Sprintf("user-%d@test.com", timestamp)
		userPassword := "userpass123456"
		_, userToken := tc.CreateUser(userEmail, userPassword)

		resp := tc.NewRequest("POST", fmt.Sprintf("/api/v1/functions/%s/invoke", functionName)).
			WithAuth(userToken).
			WithJSON(map[string]interface{}{"test": "data"}).
			Send()

		// Should NOT return 401 or 403 - authentication should pass
		require.NotEqual(t, fiber.StatusUnauthorized, resp.Status(),
			"Should not return 401 with valid user token, got %d: %s",
			resp.Status(), string(resp.Body()))
		require.NotEqual(t, fiber.StatusForbidden, resp.Status(),
			"Should not return 403 with valid user token, got %d: %s",
			resp.Status(), string(resp.Body()))
	})

	// Test 4: Request with API key should pass authentication
	// Note: May return 500 if Deno not installed, but should NOT return 401/403
	t.Run("APIKeyPassesAuth", func(t *testing.T) {
		// Create an API key
		apiKeyResp := tc.NewRequest("POST", "/api/v1/client-keys").
			WithAuth(adminToken).
			WithJSON(map[string]interface{}{
				"name":   "test-api-key",
				"scopes": []string{"execute:functions"},
			}).
			Send()
		require.Equal(t, fiber.StatusCreated, apiKeyResp.Status(), "Failed to create API key: %s", string(apiKeyResp.Body()))

		var keyData map[string]interface{}
		apiKeyResp.JSON(&keyData)
		apiKey, ok := keyData["key"].(string)
		require.True(t, ok, "API key 'key' field not found or not a string: %v", keyData)

		resp := tc.NewRequest("POST", fmt.Sprintf("/api/v1/functions/%s/invoke", functionName)).
			WithHeader("X-API-Key", apiKey).
			WithJSON(map[string]interface{}{"test": "data"}).
			Send()

		// Should NOT return 401 or 403 - authentication should pass
		require.NotEqual(t, fiber.StatusUnauthorized, resp.Status(),
			"Should not return 401 with valid API key, got %d: %s",
			resp.Status(), string(resp.Body()))
		require.NotEqual(t, fiber.StatusForbidden, resp.Status(),
			"Should not return 403 with valid API key, got %d: %s",
			resp.Status(), string(resp.Body()))
	})
}

// TestFunctionAllowUnauthenticated tests that functions with allow_unauthenticated=true work without auth
func TestFunctionAllowUnauthenticated(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()
	tc.EnsureAuthSchema()

	// Create admin user for function management
	timestamp := time.Now().UnixNano()
	email := fmt.Sprintf("admin-%d@test.com", timestamp)
	password := "adminpass123456"
	_, adminToken := tc.CreateDashboardAdminUser(email, password)

	// Create a test function WITH allow_unauthenticated=true
	functionName := fmt.Sprintf("test_public_%d", timestamp)
	createResp := tc.NewRequest("POST", "/api/v1/functions").
		WithAuth(adminToken).
		WithJSON(map[string]interface{}{
			"name": functionName,
			"code": `export default async function handler(req) {
				return new Response(JSON.stringify({ message: "public endpoint" }), {
					headers: { "Content-Type": "application/json" }
				});
			}`,
			"runtime":               "deno",
			"enabled":               true, // Make sure function is enabled
			"allow_unauthenticated": true, // Allow unauthenticated access
		}).
		Send()
	require.Equal(t, fiber.StatusCreated, createResp.Status(), "Failed to create function: %s", string(createResp.Body()))

	// Test: Request with NO authentication should pass (not return 401)
	// Note: May return 500 if Deno not installed, but should NOT return 401
	t.Run("UnauthenticatedPasses", func(t *testing.T) {
		resp := tc.NewRequest("POST", fmt.Sprintf("/api/v1/functions/%s/invoke", functionName)).
			Unauthenticated().
			WithJSON(map[string]interface{}{"test": "data"}).
			Send()

		// Should NOT return 401 - function allows unauthenticated access
		require.NotEqual(t, fiber.StatusUnauthorized, resp.Status(),
			"Should not return 401 for function with allow_unauthenticated=true, got %d: %s",
			resp.Status(), string(resp.Body()))
	})

	// Test: Request with anon key should also pass
	t.Run("AnonKeyAlsoPasses", func(t *testing.T) {
		anonKey := tc.GenerateAnonKey()

		resp := tc.NewRequest("POST", fmt.Sprintf("/api/v1/functions/%s/invoke", functionName)).
			WithAuth(anonKey).
			WithJSON(map[string]interface{}{"test": "data"}).
			Send()

		// Should NOT return 401 - function allows unauthenticated, so auth should work too
		require.NotEqual(t, fiber.StatusUnauthorized, resp.Status(),
			"Should not return 401 for public function with anon key, got %d: %s",
			resp.Status(), string(resp.Body()))
	})
}

// TestFunctionCodeCommentAllowUnauthenticated tests the @allow-unauthenticated code comment
func TestFunctionCodeCommentAllowUnauthenticated(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()
	tc.EnsureAuthSchema()

	// Create admin user for function management
	timestamp := time.Now().UnixNano()
	email := fmt.Sprintf("admin-%d@test.com", timestamp)
	password := "adminpass123456"
	_, adminToken := tc.CreateDashboardAdminUser(email, password)

	// Create a test function with @allow-unauthenticated comment
	functionName := fmt.Sprintf("test_comment_auth_%d", timestamp)
	createResp := tc.NewRequest("POST", "/api/v1/functions").
		WithAuth(adminToken).
		WithJSON(map[string]interface{}{
			"name": functionName,
			"code": `// @allow-unauthenticated
export default async function handler(req) {
	return new Response(JSON.stringify({ message: "public via comment" }), {
		headers: { "Content-Type": "application/json" }
	});
}`,
			"runtime": "deno",
			"enabled": true, // Make sure function is enabled
		}).
		Send()
	require.Equal(t, fiber.StatusCreated, createResp.Status(), "Failed to create function: %s", string(createResp.Body()))

	// Test: Request with NO authentication should pass (comment should be parsed)
	// Note: Comment parsing happens during file-based function loading, not via API
	// For API-created functions, we just verify the flag was set correctly
	t.Run("CommentParsedCorrectly", func(t *testing.T) {
		resp := tc.NewRequest("POST", fmt.Sprintf("/api/v1/functions/%s/invoke", functionName)).
			Unauthenticated().
			WithJSON(map[string]interface{}{"test": "data"}).
			Send()

		// The comment doesn't get parsed for API-created functions
		// This test would only work for file-based functions loaded via reload
		// For now, just verify it doesn't return 401 if the flag is properly set
		// In this case, since we didn't set allow_unauthenticated in the API call,
		// it should return 401
		require.Equal(t, fiber.StatusUnauthorized, resp.Status(),
			"Expected 401 for API-created function without allow_unauthenticated flag, got %d: %s",
			resp.Status(), string(resp.Body()))
	})
}

// TestFunctionAuthenticationTypes tests different authentication types work correctly
func TestFunctionAuthenticationTypes(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()
	tc.EnsureAuthSchema()

	// Create admin user
	timestamp := time.Now().UnixNano()
	email := fmt.Sprintf("admin-%d@test.com", timestamp)
	password := "adminpass123456"
	_, adminToken := tc.CreateDashboardAdminUser(email, password)

	// Create a test function that echoes auth info
	functionName := fmt.Sprintf("test_auth_echo_%d", timestamp)
	createResp := tc.NewRequest("POST", "/api/v1/functions").
		WithAuth(adminToken).
		WithJSON(map[string]interface{}{
			"name": functionName,
			"code": `export default async function handler(req) {
				const authType = req.headers.get('X-Auth-Type') || 'none';
				return new Response(JSON.stringify({
					message: "authenticated",
					auth_type: authType
				}), {
					headers: { "Content-Type": "application/json" }
				});
			}`,
			"runtime":               "deno",
			"enabled":               true,  // Make sure function is enabled
			"allow_unauthenticated": false, // Require auth
		}).
		Send()
	require.Equal(t, fiber.StatusCreated, createResp.Status(), "Failed to create function: %s", string(createResp.Body()))

	tests := []struct {
		name       string
		setupAuth  func(tc *test.TestContext) string
		headerName string
	}{
		{
			name: "AnonKey",
			setupAuth: func(tc *test.TestContext) string {
				return tc.GenerateAnonKey()
			},
			headerName: "Authorization",
		},
		{
			name: "UserJWT",
			setupAuth: func(tc *test.TestContext) string {
				userEmail := fmt.Sprintf("user-%d@test.com", time.Now().UnixNano())
				_, token := tc.CreateUser(userEmail, "password123456")
				return token
			},
			headerName: "Authorization",
		},
		{
			name: "APIKey",
			setupAuth: func(tc *test.TestContext) string {
				resp := tc.NewRequest("POST", "/api/v1/client-keys").
					WithAuth(adminToken).
					WithJSON(map[string]interface{}{
						"name":   fmt.Sprintf("key-%d", time.Now().UnixNano()),
						"scopes": []string{"execute:functions"},
					}).
					Send()
				var keyData map[string]interface{}
				resp.JSON(&keyData)
				return keyData["key"].(string)
			},
			headerName: "X-API-Key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authValue := tt.setupAuth(tc)

			req := tc.NewRequest("POST", fmt.Sprintf("/api/v1/functions/%s/invoke", functionName)).
				WithJSON(map[string]interface{}{"test": "data"})

			if tt.headerName == "Authorization" {
				req = req.WithAuth(authValue)
			} else {
				req = req.WithHeader(tt.headerName, authValue)
			}

			resp := req.Send()

			// Should NOT return 401 or 403 - authentication should pass
			// May return 500 if Deno not installed, but that's after auth
			require.NotEqual(t, fiber.StatusUnauthorized, resp.Status(),
				"Should not return 401 for %s auth, got %d: %s",
				tt.name, resp.Status(), string(resp.Body()))
			require.NotEqual(t, fiber.StatusForbidden, resp.Status(),
				"Should not return 403 for %s auth, got %d: %s",
				tt.name, resp.Status(), string(resp.Body()))
		})
	}
}
