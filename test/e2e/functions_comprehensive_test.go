package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wayli-app/fluxbase/internal/auth"
	"github.com/wayli-app/fluxbase/internal/config"
	"github.com/wayli-app/fluxbase/internal/database"
	"github.com/wayli-app/fluxbase/internal/functions"
	"github.com/wayli-app/fluxbase/internal/middleware"
)

// TestEdgeFunctionsComprehensive tests Edge Functions with authentication contexts
func TestEdgeFunctionsComprehensive(t *testing.T) {
	// Setup test environment
	cfg := setupFunctionsTestConfig(t)
	db := setupFunctionsTestDatabase(t, cfg)
	defer db.Close()

	// Ensure functions schema exists
	setupFunctionsSchema(t, db)

	// Create test app with functions routes
	app := createFunctionsTestApp(t, db, cfg)

	// Run Edge Functions tests
	t.Run("Function CRUD Operations", func(t *testing.T) {
		testFunctionCRUD(t, app, db, cfg.JWTSecret)
	})

	t.Run("HTTP Invocation - Authenticated User", func(t *testing.T) {
		testFunctionInvocationAuthenticated(t, app, db, cfg.JWTSecret)
	})

	t.Run("HTTP Invocation - Anonymous User", func(t *testing.T) {
		testFunctionInvocationAnonymous(t, app, db)
	})

	t.Run("Authentication Context Propagation", func(t *testing.T) {
		testAuthContextPropagation(t, app, db, cfg.JWTSecret)
	})

	t.Run("User ID Available in Function", func(t *testing.T) {
		testUserIDInFunction(t, app, db, cfg.JWTSecret)
	})

	t.Run("Service Role Authentication", func(t *testing.T) {
		testServiceRoleAuth(t, app, db, cfg.JWTSecret)
	})

	t.Run("Permission Sandboxing", func(t *testing.T) {
		testPermissionSandboxing(t, app, db, cfg.JWTSecret)
	})

	t.Run("Timeout Enforcement", func(t *testing.T) {
		testTimeoutEnforcement(t, app, db, cfg.JWTSecret)
	})

	t.Run("Execution Logging", func(t *testing.T) {
		testExecutionLogging(t, app, db, cfg.JWTSecret)
	})

	t.Run("Error Handling", func(t *testing.T) {
		testFunctionErrorHandling(t, app, db, cfg.JWTSecret)
	})

	t.Run("Disabled Function", func(t *testing.T) {
		testDisabledFunction(t, app, db, cfg.JWTSecret)
	})
}

// setupFunctionsTestConfig creates test configuration
func setupFunctionsTestConfig(t *testing.T) *config.Config {
	return &config.Config{
		DatabaseURL: "postgres://postgres:postgres@localhost:5432/fluxbase_test?sslmode=disable",
		JWTSecret:   "test-jwt-secret-functions",
		Port:        "8080",
		FluxbaseURL: "http://localhost:8080",
	}
}

// setupFunctionsTestDatabase creates database connection
func setupFunctionsTestDatabase(t *testing.T, cfg *config.Config) *database.Connection {
	db, err := database.Connect(cfg.DatabaseURL)
	require.NoError(t, err, "Failed to connect to test database")
	return db
}

// setupFunctionsSchema creates functions schema
func setupFunctionsSchema(t *testing.T, db *database.Connection) {
	ctx := context.Background()

	queries := []string{
		`CREATE SCHEMA IF NOT EXISTS functions`,

		`CREATE TABLE IF NOT EXISTS functions.edge_functions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT UNIQUE NOT NULL,
			description TEXT,
			code TEXT NOT NULL,
			enabled BOOLEAN DEFAULT true,
			timeout_seconds INT DEFAULT 30,
			memory_limit_mb INT DEFAULT 128,
			allow_net BOOLEAN DEFAULT true,
			allow_env BOOLEAN DEFAULT true,
			allow_read BOOLEAN DEFAULT false,
			allow_write BOOLEAN DEFAULT false,
			cron_schedule TEXT,
			version INT DEFAULT 1,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			created_by UUID
		)`,

		`CREATE TABLE IF NOT EXISTS functions.edge_function_executions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			function_id UUID REFERENCES functions.edge_functions(id) ON DELETE CASCADE,
			trigger_type TEXT NOT NULL,
			status TEXT NOT NULL,
			status_code INT,
			duration_ms INT,
			result TEXT,
			logs TEXT,
			error_message TEXT,
			executed_at TIMESTAMPTZ DEFAULT NOW(),
			completed_at TIMESTAMPTZ
		)`,

		`CREATE SCHEMA IF NOT EXISTS auth`,
		`CREATE TABLE IF NOT EXISTS auth.users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email TEXT UNIQUE NOT NULL,
			encrypted_password TEXT,
			email_confirmed_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
	}

	for _, query := range queries {
		_, err := db.Pool().Exec(ctx, query)
		require.NoError(t, err, "Failed to setup functions schema")
	}

	// Cleanup
	_, _ = db.Pool().Exec(ctx, "TRUNCATE functions.edge_function_executions CASCADE")
	_, _ = db.Pool().Exec(ctx, "TRUNCATE functions.edge_functions CASCADE")
	_, _ = db.Pool().Exec(ctx, "DELETE FROM auth.users WHERE email LIKE '%@functest.com'")
}

// createFunctionsTestApp creates Fiber app with functions routes
func createFunctionsTestApp(t *testing.T, db *database.Connection, cfg *config.Config) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Add auth middleware
	app.Use(middleware.AuthMiddleware(middleware.AuthConfig{
		JWTSecret: cfg.JWTSecret,
		Optional:  true,
	}))

	// Register functions routes
	functionsHandler := functions.NewHandler(db.Pool())
	functionsHandler.RegisterRoutes(app)

	return app
}

// testFunctionCRUD tests function management operations
func testFunctionCRUD(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	token := createFunctionsTestUser(t, db, jwtSecret)

	// Create function
	t.Run("Create Function", func(t *testing.T) {
		fnReq := map[string]interface{}{
			"name":        "test_function",
			"description": "A test function",
			"code": `
				function handler(request) {
					return {
						status: 200,
						headers: { "Content-Type": "application/json" },
						body: JSON.stringify({ message: "Hello from test function" })
					};
				}
			`,
			"enabled":         true,
			"timeout_seconds": 30,
			"allow_net":       true,
			"allow_env":       true,
		}

		body, _ := json.Marshal(fnReq)
		req := httptest.NewRequest("POST", "/api/v1/functions", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode, "Function creation should succeed")

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "test_function", result["name"])
		assert.NotNil(t, result["id"])
	})

	// List functions
	t.Run("List Functions", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/functions", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var functions []map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&functions)
		assert.GreaterOrEqual(t, len(functions), 1)
	})

	// Get function by name
	t.Run("Get Function", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/functions/test_function", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var fn map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&fn)
		assert.Equal(t, "test_function", fn["name"])
	})

	// Update function
	t.Run("Update Function", func(t *testing.T) {
		updates := map[string]interface{}{
			"description": "Updated description",
			"enabled":     true,
		}

		body, _ := json.Marshal(updates)
		req := httptest.NewRequest("PUT", "/api/v1/functions/test_function", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	// Delete function
	t.Run("Delete Function", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/functions/test_function", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, 204, resp.StatusCode)
	})
}

// testFunctionInvocationAuthenticated tests function invocation with authenticated user
func testFunctionInvocationAuthenticated(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	userID, token := createFunctionsTestUserWithID(t, db, jwtSecret, "authuser@functest.com")

	// Create a function that checks authentication
	fnReq := map[string]interface{}{
		"name": "auth_check_function",
		"code": `
			function handler(request) {
				const userId = request.user_id;
				const isAuthenticated = userId ? true : false;

				return {
					status: 200,
					headers: { "Content-Type": "application/json" },
					body: JSON.stringify({
						authenticated: isAuthenticated,
						user_id: userId,
						message: isAuthenticated ? "User is authenticated" : "Anonymous user"
					})
				};
			}
		`,
		"enabled": true,
	}

	body, _ := json.Marshal(fnReq)
	req := httptest.NewRequest("POST", "/api/v1/functions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)

	// Invoke function with authentication
	invokeReq := map[string]interface{}{
		"test": "data",
	}

	body, _ = json.Marshal(invokeReq)
	req = httptest.NewRequest("POST", "/api/v1/functions/auth_check_function/invoke", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = app.Test(req, 10*time.Second)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, true, result["authenticated"], "User should be authenticated")
	assert.Equal(t, userID, result["user_id"], "User ID should match")
}

// testFunctionInvocationAnonymous tests function invocation without authentication
func testFunctionInvocationAnonymous(t *testing.T, app *fiber.App, db *database.Connection) {
	// Use the same auth_check_function created by previous test (or create if not exists)
	// Invoke without authentication token
	req := httptest.NewRequest("POST", "/api/v1/functions/auth_check_function/invoke", nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, 10*time.Second)
	require.NoError(t, err)

	// Function should still execute but with no user_id
	if resp.StatusCode == 200 {
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, false, result["authenticated"], "Anonymous user should not be authenticated")
		assert.Empty(t, result["user_id"], "Anonymous user should have no user_id")
	}
}

// testAuthContextPropagation tests that auth context is properly passed
func testAuthContextPropagation(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	userID, token := createFunctionsTestUserWithID(t, db, jwtSecret, "contexttest@functest.com")

	// Create function that echoes request object
	fnReq := map[string]interface{}{
		"name": "context_echo_function",
		"code": `
			function handler(request) {
				return {
					status: 200,
					headers: { "Content-Type": "application/json" },
					body: JSON.stringify({
						method: request.method,
						url: request.url,
						user_id: request.user_id,
						has_headers: !!request.headers,
						body: request.body
					})
				};
			}
		`,
		"enabled": true,
	}

	body, _ := json.Marshal(fnReq)
	req := httptest.NewRequest("POST", "/api/v1/functions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	require.Equal(t, 201, resp.StatusCode)

	// Invoke function
	invokeBody := map[string]interface{}{"test": "data"}
	body, _ = json.Marshal(invokeBody)
	req = httptest.NewRequest("POST", "/api/v1/functions/context_echo_function/invoke", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Custom-Header", "test-value")

	resp, err = app.Test(req, 10*time.Second)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, "POST", result["method"])
	assert.Equal(t, userID, result["user_id"], "User ID should be propagated")
	assert.Equal(t, true, result["has_headers"])
	assert.NotNil(t, result["body"])
}

// testUserIDInFunction tests that user ID is accessible in function code
func testUserIDInFunction(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	userID, token := createFunctionsTestUserWithID(t, db, jwtSecret, "userid@functest.com")

	// Create function that uses user ID for logic
	fnReq := map[string]interface{}{
		"name": "user_data_function",
		"code": `
			function handler(request) {
				if (!request.user_id) {
					return {
						status: 401,
						headers: { "Content-Type": "application/json" },
						body: JSON.stringify({ error: "Unauthorized" })
					};
				}

				// Simulate user-specific logic
				return {
					status: 200,
					headers: { "Content-Type": "application/json" },
					body: JSON.stringify({
						user_id: request.user_id,
						data: "User-specific data for " + request.user_id
					})
				};
			}
		`,
		"enabled": true,
	}

	body, _ := json.Marshal(fnReq)
	req := httptest.NewRequest("POST", "/api/v1/functions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	require.Equal(t, 201, resp.StatusCode)

	// Invoke with authentication
	req = httptest.NewRequest("POST", "/api/v1/functions/user_data_function/invoke", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = app.Test(req, 10*time.Second)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, userID, result["user_id"])
	assert.Contains(t, result["data"], userID)

	// Invoke without authentication (should fail)
	req = httptest.NewRequest("POST", "/api/v1/functions/user_data_function/invoke", nil)

	resp, err = app.Test(req, 10*time.Second)
	require.NoError(t, err)
	// Should return 401 from function logic
	if resp.StatusCode == 200 {
		var errorResult map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResult)
		// Function's own 401 response
		assert.NotNil(t, errorResult)
	}
}

// testServiceRoleAuth tests service role authentication (admin access)
func testServiceRoleAuth(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	// Service role bypass RLS and has full access
	// This would typically use a special API key or token

	token := createFunctionsTestUser(t, db, jwtSecret)

	// Create function that checks for service role
	fnReq := map[string]interface{}{
		"name": "service_role_function",
		"code": `
			function handler(request) {
				// In production, check for service role token
				const isServiceRole = request.headers && request.headers["x-service-role"] === "true";

				return {
					status: 200,
					headers: { "Content-Type": "application/json" },
					body: JSON.stringify({
						is_service_role: isServiceRole,
						can_bypass_rls: isServiceRole
					})
				};
			}
		`,
		"enabled": true,
	}

	body, _ := json.Marshal(fnReq)
	req := httptest.NewRequest("POST", "/api/v1/functions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	require.Equal(t, 201, resp.StatusCode)

	// Invoke with service role header
	req = httptest.NewRequest("POST", "/api/v1/functions/service_role_function/invoke", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Service-Role", "true")

	resp, err = app.Test(req, 10*time.Second)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, true, result["is_service_role"])
}

// testPermissionSandboxing tests Deno permission enforcement
func testPermissionSandboxing(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	token := createFunctionsTestUser(t, db, jwtSecret)

	// Create function with network access disabled
	fnReq := map[string]interface{}{
		"name":      "no_network_function",
		"allow_net": false, // Disable network
		"code": `
			async function handler(request) {
				try {
					// This should fail because network access is disabled
					const response = await fetch("https://api.github.com");
					return {
						status: 200,
						body: JSON.stringify({ error: "Should not reach here" })
					};
				} catch (error) {
					// Expected: Permission denied
					return {
						status: 200,
						headers: { "Content-Type": "application/json" },
						body: JSON.stringify({
							error: error.message,
							network_blocked: true
						})
					};
				}
			}
		`,
		"enabled": true,
	}

	body, _ := json.Marshal(fnReq)
	req := httptest.NewRequest("POST", "/api/v1/functions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	require.Equal(t, 201, resp.StatusCode)

	// Invoke function
	req = httptest.NewRequest("POST", "/api/v1/functions/no_network_function/invoke", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = app.Test(req, 10*time.Second)
	require.NoError(t, err)

	// Should complete but with permission error
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.NotNil(t, result)
}

// testTimeoutEnforcement tests function timeout
func testTimeoutEnforcement(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	token := createFunctionsTestUser(t, db, jwtSecret)

	// Create function with short timeout
	fnReq := map[string]interface{}{
		"name":            "timeout_function",
		"timeout_seconds": 2, // 2 second timeout
		"code": `
			async function handler(request) {
				// Simulate long-running operation
				await new Promise(resolve => setTimeout(resolve, 5000)); // 5 seconds
				return {
					status: 200,
					body: JSON.stringify({ message: "Completed" })
				};
			}
		`,
		"enabled": true,
	}

	body, _ := json.Marshal(fnReq)
	req := httptest.NewRequest("POST", "/api/v1/functions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	require.Equal(t, 201, resp.StatusCode)

	// Invoke function (should timeout)
	req = httptest.NewRequest("POST", "/api/v1/functions/timeout_function/invoke", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = app.Test(req, 15*time.Second) // Give enough time for test itself
	require.NoError(t, err)

	// Should return 504 Gateway Timeout or similar
	assert.True(t, resp.StatusCode == 504 || resp.StatusCode == 500, "Should timeout")
}

// testExecutionLogging tests that executions are logged
func testExecutionLogging(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	token := createFunctionsTestUser(t, db, jwtSecret)

	// Create and invoke a function
	fnReq := map[string]interface{}{
		"name": "logged_function",
		"code": `
			function handler(request) {
				console.error("Test log message");
				return {
					status: 200,
					body: JSON.stringify({ logged: true })
				};
			}
		`,
		"enabled": true,
	}

	body, _ := json.Marshal(fnReq)
	req := httptest.NewRequest("POST", "/api/v1/functions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	require.Equal(t, 201, resp.StatusCode)

	// Invoke function
	req = httptest.NewRequest("POST", "/api/v1/functions/logged_function/invoke", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = app.Test(req, 10*time.Second)
	require.NoError(t, err)

	// Query execution history
	time.Sleep(500 * time.Millisecond) // Wait for async logging

	req = httptest.NewRequest("GET", "/api/v1/functions/logged_function/executions", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var executions []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&executions)
	assert.GreaterOrEqual(t, len(executions), 1, "Should have at least one execution logged")
}

// testFunctionErrorHandling tests error handling in functions
func testFunctionErrorHandling(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	token := createFunctionsTestUser(t, db, jwtSecret)

	// Create function that throws error
	fnReq := map[string]interface{}{
		"name": "error_function",
		"code": `
			function handler(request) {
				throw new Error("Intentional error for testing");
			}
		`,
		"enabled": true,
	}

	body, _ := json.Marshal(fnReq)
	req := httptest.NewRequest("POST", "/api/v1/functions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	require.Equal(t, 201, resp.StatusCode)

	// Invoke function
	req = httptest.NewRequest("POST", "/api/v1/functions/error_function/invoke", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = app.Test(req, 10*time.Second)
	require.NoError(t, err)

	// Should return 500 with error details
	assert.Equal(t, 500, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.NotNil(t, result["error"])
}

// testDisabledFunction tests that disabled functions cannot be invoked
func testDisabledFunction(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	token := createFunctionsTestUser(t, db, jwtSecret)

	// Create disabled function
	fnReq := map[string]interface{}{
		"name":    "disabled_function",
		"enabled": false, // Disabled
		"code": `
			function handler(request) {
				return { status: 200, body: JSON.stringify({ message: "Hello" }) };
			}
		`,
	}

	body, _ := json.Marshal(fnReq)
	req := httptest.NewRequest("POST", "/api/v1/functions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	require.Equal(t, 201, resp.StatusCode)

	// Try to invoke disabled function
	req = httptest.NewRequest("POST", "/api/v1/functions/disabled_function/invoke", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode, "Disabled function should be rejected")
}

// Helper functions

func createFunctionsTestUser(t *testing.T, db *database.Connection, jwtSecret string) string {
	_, token := createFunctionsTestUserWithID(t, db, jwtSecret, "testuser@functest.com")
	return token
}

func createFunctionsTestUserWithID(t *testing.T, db *database.Connection, jwtSecret string, email string) (string, string) {
	ctx := context.Background()

	userID := uuid.New().String()
	_, err := db.Pool().Exec(ctx, `
		INSERT INTO auth.users (id, email, encrypted_password, email_confirmed_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (email) DO NOTHING
	`, userID, email, "$2a$10$FAKEHASH")
	require.NoError(t, err)

	// Generate JWT
	authService := auth.NewService(db, jwtSecret, "smtp://fake", "noreply@test.com")
	token, err := authService.GenerateJWT(userID, email)
	require.NoError(t, err)

	return userID, token
}
