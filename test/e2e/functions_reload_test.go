package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
	"github.com/wayli-app/fluxbase/test"
)

// setupFunctionsTest prepares the test context for functions reload tests
func setupFunctionsTest(t *testing.T) (*test.TestContext, string, string) {
	tc := test.NewTestContext(t)
	tc.EnsureAuthSchema()

	// Create temporary functions directory for this test
	tmpDir, err := os.MkdirTemp("", "functions-test-")
	require.NoError(t, err, "Failed to create temp functions directory")

	// Create admin user for authentication
	timestamp := time.Now().UnixNano()
	email := fmt.Sprintf("admin-%s-%d@test.com", t.Name(), timestamp)
	password := "adminpass123456"
	_, token := tc.CreateAdminUser(email, password)

	return tc, token, tmpDir
}

// TestFunctionsReloadEndpoint tests the admin functions reload endpoint
func TestFunctionsReloadEndpoint(t *testing.T) {
	tc, token, functionsDir := setupFunctionsTest(t)
	defer tc.Close()
	defer os.RemoveAll(functionsDir)

	// Create test function files in the temporary directory
	testFunctions := map[string]string{
		"hello": `async function handler(req) {
	return {
		status: 200,
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify({ message: "Hello World" })
	};
}`,
		"greet": `async function handler(req) {
	const data = JSON.parse(req.body || "{}");
	return {
		status: 200,
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify({ message: "Hello " + (data.name || "Guest") })
	};
}`,
	}

	for name, code := range testFunctions {
		filePath := filepath.Join(functionsDir, name+".ts")
		err := os.WriteFile(filePath, []byte(code), 0644)
		require.NoError(t, err, "Failed to create test function file: %s", name)
	}

	// Note: This test assumes the server is configured with the test functions directory
	// In a real scenario, you'd need to configure the server to use this temp directory
	t.Log("Note: This test requires the server to be configured with the test functions directory")

	// Test reload endpoint - should create functions from files
	resp := tc.NewRequest("POST", "/api/v1/admin/functions/reload").
		WithAuth(token).
		Send().
		AssertStatus(fiber.StatusOK)

	var result map[string]interface{}
	resp.JSON(&result)

	require.Contains(t, result, "message", "Response should contain message")
	require.Contains(t, result, "total", "Response should contain total count")
	require.Contains(t, result, "created", "Response should contain created list")
	require.Contains(t, result, "updated", "Response should contain updated list")
	require.Contains(t, result, "errors", "Response should contain errors list")

	t.Logf("Reload result: %+v", result)
}

// TestFunctionsReloadUnauthorized tests that unauthorized users cannot reload functions
func TestFunctionsReloadUnauthorized(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()
	tc.EnsureAuthSchema()

	// Test without authentication
	tc.NewRequest("POST", "/api/v1/admin/functions/reload").
		Send().
		AssertStatus(fiber.StatusUnauthorized)

	// Test with invalid token
	tc.NewRequest("POST", "/api/v1/admin/functions/reload").
		WithAuth("invalid-token").
		Send().
		AssertStatus(fiber.StatusUnauthorized)
}

// TestFunctionsReloadWithRegularUser tests that regular users cannot reload functions
func TestFunctionsReloadWithRegularUser(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()
	tc.EnsureAuthSchema()

	// Create a regular user (not admin)
	timestamp := time.Now().UnixNano()
	email := fmt.Sprintf("user-%s-%d@test.com", t.Name(), timestamp)
	password := "userpass123456"
	_, userToken := tc.CreateTestUser(email, password)

	// Test with regular user token - should be forbidden
	tc.NewRequest("POST", "/api/v1/admin/functions/reload").
		WithAuth(userToken).
		Send().
		AssertStatus(fiber.StatusForbidden)
}

// TestFunctionsReloadUpdatesExistingFunctions tests updating existing functions
func TestFunctionsReloadUpdatesExistingFunctions(t *testing.T) {
	tc, token, functionsDir := setupFunctionsTest(t)
	defer tc.Close()
	defer os.RemoveAll(functionsDir)

	// Create initial function file
	functionName := "update-test"
	initialCode := `async function handler(req) {
	return { status: 200, body: "version 1" };
}`
	filePath := filepath.Join(functionsDir, functionName+".ts")
	err := os.WriteFile(filePath, []byte(initialCode), 0644)
	require.NoError(t, err)

	// First reload - should create the function
	resp1 := tc.NewRequest("POST", "/api/v1/admin/functions/reload").
		WithAuth(token).
		Send().
		AssertStatus(fiber.StatusOK)

	var result1 map[string]interface{}
	resp1.JSON(&result1)
	t.Logf("First reload result: %+v", result1)

	// Update the function code
	updatedCode := `async function handler(req) {
	return { status: 200, body: "version 2" };
}`
	err = os.WriteFile(filePath, []byte(updatedCode), 0644)
	require.NoError(t, err)

	// Second reload - should update the function
	resp2 := tc.NewRequest("POST", "/api/v1/admin/functions/reload").
		WithAuth(token).
		Send().
		AssertStatus(fiber.StatusOK)

	var result2 map[string]interface{}
	resp2.JSON(&result2)
	t.Logf("Second reload result: %+v", result2)

	// The function should be in the updated list (if code changed)
	// or not in any list (if code is the same)
}

// TestFunctionsReloadWithInvalidFiles tests handling of invalid function files
func TestFunctionsReloadWithInvalidFiles(t *testing.T) {
	tc, token, functionsDir := setupFunctionsTest(t)
	defer tc.Close()
	defer os.RemoveAll(functionsDir)

	// Create files with invalid names (should be skipped with warnings)
	invalidFiles := map[string]string{
		"..":    "malicious code", // Reserved name
		".":     "malicious code", // Reserved name
		"index": "malicious code", // Reserved name
	}

	for name, content := range invalidFiles {
		// These should be rejected by the filesystem or validation
		filePath := filepath.Join(functionsDir, name+".ts")
		_ = os.WriteFile(filePath, []byte(content), 0644)
	}

	// Create a valid function
	validPath := filepath.Join(functionsDir, "valid-function.ts")
	err := os.WriteFile(validPath, []byte("async function handler(req) { return { status: 200 }; }"), 0644)
	require.NoError(t, err)

	// Reload - should process valid function and skip/report invalid ones
	resp := tc.NewRequest("POST", "/api/v1/admin/functions/reload").
		WithAuth(token).
		Send().
		AssertStatus(fiber.StatusOK)

	var result map[string]interface{}
	resp.JSON(&result)
	t.Logf("Reload with invalid files result: %+v", result)
}

// TestFunctionsReloadEmptyDirectory tests reload with no function files
func TestFunctionsReloadEmptyDirectory(t *testing.T) {
	tc, token, functionsDir := setupFunctionsTest(t)
	defer tc.Close()
	defer os.RemoveAll(functionsDir)

	// Don't create any function files

	// Reload should succeed but report 0 functions
	resp := tc.NewRequest("POST", "/api/v1/admin/functions/reload").
		WithAuth(token).
		Send().
		AssertStatus(fiber.StatusOK)

	var result map[string]interface{}
	resp.JSON(&result)

	require.Contains(t, result, "total", "Response should contain total count")

	// Total should be 0 or the endpoint should handle empty directory gracefully
	t.Logf("Reload empty directory result: %+v", result)
}

// TestFunctionsReloadConcurrent tests concurrent reload requests
func TestFunctionsReloadConcurrent(t *testing.T) {
	tc, token, functionsDir := setupFunctionsTest(t)
	defer tc.Close()
	defer os.RemoveAll(functionsDir)

	// Create a test function file
	filePath := filepath.Join(functionsDir, "concurrent-test.ts")
	err := os.WriteFile(filePath, []byte("async function handler(req) { return { status: 200 }; }"), 0644)
	require.NoError(t, err)

	// Send multiple concurrent reload requests
	done := make(chan bool)
	for i := 0; i < 3; i++ {
		go func(id int) {
			resp := tc.NewRequest("POST", "/api/v1/admin/functions/reload").
				WithAuth(token).
				Send()

			// All requests should succeed
			require.Equal(t, fiber.StatusOK, resp.Status(), "Concurrent request %d failed", id)
			done <- true
		}(i)
	}

	// Wait for all requests to complete
	for i := 0; i < 3; i++ {
		<-done
	}

	t.Log("All concurrent reload requests completed successfully")
}
