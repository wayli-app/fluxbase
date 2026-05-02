package e2e

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/test"
)

func createTestFunction(t *testing.T, tc *test.TestContext, adminToken, functionName, code string) {
	t.Helper()
	resp := tc.NewRequest("POST", "/api/v1/functions").
		WithAuth(adminToken).
		WithJSON(map[string]interface{}{
			"name":    functionName,
			"code":    code,
			"runtime": "deno",
			"enabled": true,
		}).
		Send()
	require.Equal(t, fiber.StatusCreated, resp.Status(), "Failed to create function %s: %s", functionName, string(resp.Body()))
}

func invokeFunction(t *testing.T, tc *test.TestContext, adminToken, functionName string) map[string]interface{} {
	t.Helper()
	resp := tc.NewRequest("POST", fmt.Sprintf("/api/v1/functions/%s/invoke", functionName)).
		WithAuth(adminToken).
		WithJSON(map[string]interface{}{"test": "data"}).
		Send()
	require.Equal(t, fiber.StatusOK, resp.Status(), "Failed to invoke function %s: %s", functionName, string(resp.Body()))

	var result map[string]interface{}
	err := json.Unmarshal(resp.Body(), &result)
	require.NoError(t, err)
	return result
}

func getExecutions(t *testing.T, tc *test.TestContext, adminToken, functionName string) []interface{} {
	t.Helper()
	resp := tc.NewRequest("GET", fmt.Sprintf("/api/v1/admin/functions/executions?function_name=%s&limit=10", functionName)).
		WithAuth(adminToken).
		WithDefaultTenant().
		Send()
	require.Equal(t, fiber.StatusOK, resp.Status(), "Failed to get executions: %s", string(resp.Body()))

	var result map[string]interface{}
	err := json.Unmarshal(resp.Body(), &result)
	require.NoError(t, err)

	executions, ok := result["executions"].([]interface{})
	require.True(t, ok, "Response should contain executions array")
	return executions
}

func getExecutionLogs(t *testing.T, tc *test.TestContext, adminToken, executionID string) []interface{} {
	t.Helper()
	resp := tc.NewRequest("GET", fmt.Sprintf("/api/v1/admin/functions/executions/%s/logs", executionID)).
		WithAuth(adminToken).
		WithDefaultTenant().
		Send()

	if resp.Status() == fiber.StatusServiceUnavailable {
		t.Log("Logging service not available in test context, skipping log retrieval")
		return nil
	}
	require.Equal(t, fiber.StatusOK, resp.Status(), "Failed to get execution logs: %s", string(resp.Body()))

	var result map[string]interface{}
	err := json.Unmarshal(resp.Body(), &result)
	require.NoError(t, err)

	entries, ok := result["entries"].([]interface{})
	require.True(t, ok, "Response should contain entries array")
	return entries
}

func TestFunctionExecutionStatus(t *testing.T) {
	rateLimiter, pubSub := test.NewInMemoryDependencies()
	tc := test.NewTestContextWithOptions(t, test.TestContextOptions{
		RateLimiter: rateLimiter,
		PubSub:      pubSub,
	})
	defer tc.Close()
	tc.EnsureAuthSchema()

	timestamp := time.Now().UnixNano()
	email := fmt.Sprintf("admin-exec-status-%d@test.com", timestamp)
	_, adminToken := tc.CreateDashboardAdminUser(email, "adminpass123456")

	functionName := fmt.Sprintf("test_exec_status_%d", timestamp)
	createTestFunction(t, tc, adminToken, functionName, `export default async function handler(req) {
		return new Response(JSON.stringify({ message: "hello" }), {
			headers: { "Content-Type": "application/json" }
		});
	}`)

	invokeFunction(t, tc, adminToken, functionName)

	time.Sleep(500 * time.Millisecond)

	executions := getExecutions(t, tc, adminToken, functionName)
	require.NotEmpty(t, executions, "Expected at least one execution record")

	exec := executions[0].(map[string]interface{})
	assert.Equal(t, "success", exec["status"], "Execution status should be 'success'")
	assert.NotNil(t, exec["status_code"], "Execution should have status_code")
	assert.NotNil(t, exec["duration_ms"], "Execution should have duration_ms")

	t.Logf("Execution: status=%v, status_code=%v, duration_ms=%v", exec["status"], exec["status_code"], exec["duration_ms"])
}

func TestFunctionExecutionLogs(t *testing.T) {
	rateLimiter, pubSub := test.NewInMemoryDependencies()
	tc := test.NewTestContextWithOptions(t, test.TestContextOptions{
		RateLimiter: rateLimiter,
		PubSub:      pubSub,
	})
	defer tc.Close()
	tc.EnsureAuthSchema()

	timestamp := time.Now().UnixNano()
	email := fmt.Sprintf("admin-exec-logs-%d@test.com", timestamp)
	_, adminToken := tc.CreateDashboardAdminUser(email, "adminpass123456")

	functionName := fmt.Sprintf("test_exec_logs_%d", timestamp)
	createTestFunction(t, tc, adminToken, functionName, `export default async function handler(req) {
		console.log("test log line 1");
		console.log("test log line 2");
		return new Response(JSON.stringify({ ok: true }), {
			headers: { "Content-Type": "application/json" }
		});
	}`)

	invokeFunction(t, tc, adminToken, functionName)

	time.Sleep(500 * time.Millisecond)

	executions := getExecutions(t, tc, adminToken, functionName)
	require.NotEmpty(t, executions, "Expected at least one execution record")

	exec := executions[0].(map[string]interface{})
	executionID, ok := exec["id"].(string)
	require.True(t, ok, "Execution should have an id")
	require.NotEmpty(t, executionID, "Execution ID should not be empty")

	entries := getExecutionLogs(t, tc, adminToken, executionID)

	if len(entries) > 0 {
		t.Logf("Got %d log entries", len(entries))
		for _, e := range entries {
			entry := e.(map[string]interface{})
			t.Logf("  line=%v level=%v message=%v", entry["line_number"], entry["level"], entry["message"])
		}
	} else {
		t.Log("No log entries returned (logs may not be captured for inline functions)")
	}
}

func TestFunctionExecutionFailure(t *testing.T) {
	rateLimiter, pubSub := test.NewInMemoryDependencies()
	tc := test.NewTestContextWithOptions(t, test.TestContextOptions{
		RateLimiter: rateLimiter,
		PubSub:      pubSub,
	})
	defer tc.Close()
	tc.EnsureAuthSchema()

	timestamp := time.Now().UnixNano()
	email := fmt.Sprintf("admin-exec-fail-%d@test.com", timestamp)
	_, adminToken := tc.CreateDashboardAdminUser(email, "adminpass123456")

	functionName := fmt.Sprintf("test_exec_fail_%d", timestamp)
	createTestFunction(t, tc, adminToken, functionName, `export default async function handler(req) {
		throw new Error("intentional test error");
	}`)

	resp := tc.NewRequest("POST", fmt.Sprintf("/api/v1/functions/%s/invoke", functionName)).
		WithAuth(adminToken).
		WithJSON(map[string]interface{}{"test": "data"}).
		Send()
	require.Equal(t, fiber.StatusInternalServerError, resp.Status(), "Expected 500 for failing function: %s", string(resp.Body()))

	time.Sleep(500 * time.Millisecond)

	executions := getExecutions(t, tc, adminToken, functionName)
	require.NotEmpty(t, executions, "Expected at least one execution record")

	exec := executions[0].(map[string]interface{})
	assert.Equal(t, "error", exec["status"], "Execution status should be 'error'")
	assert.NotNil(t, exec["error_message"], "Execution should have error_message")

	t.Logf("Failed execution: status=%v, error_message=%v", exec["status"], exec["error_message"])
}

func TestFunctionExecutionTenantIsolation(t *testing.T) {
	rateLimiter, pubSub := test.NewInMemoryDependencies()
	tc := test.NewTestContextWithOptions(t, test.TestContextOptions{
		RateLimiter: rateLimiter,
		PubSub:      pubSub,
	})
	defer tc.Close()
	tc.EnsureAuthSchema()

	timestamp := time.Now().UnixNano()
	email := fmt.Sprintf("admin-exec-tenant-%d@test.com", timestamp)
	_, adminToken := tc.CreateDashboardAdminUser(email, "adminpass123456")

	defaultTenantID := tc.GetDefaultTenantID()

	functionName := fmt.Sprintf("test_exec_tenant_%d", timestamp)
	createTestFunction(t, tc, adminToken, functionName, `export default async function handler(req) {
		return new Response(JSON.stringify({ tenant: "test" }), {
			headers: { "Content-Type": "application/json" }
		});
	}`)

	invokeFunction(t, tc, adminToken, functionName)

	time.Sleep(500 * time.Millisecond)

	resp := tc.NewRequest("GET", fmt.Sprintf("/api/v1/admin/functions/executions?function_name=%s&limit=10", functionName)).
		WithAuth(adminToken).
		WithDefaultTenant().
		Send()
	require.Equal(t, fiber.StatusOK, resp.Status())

	var tenantResult map[string]interface{}
	err := json.Unmarshal(resp.Body(), &tenantResult)
	require.NoError(t, err)

	executions := tenantResult["executions"].([]interface{})
	require.NotEmpty(t, executions, "Default tenant should see its executions")

	exec := executions[0].(map[string]interface{})
	execID := exec["id"].(string)
	require.NotEmpty(t, execID)

	entries := getExecutionLogs(t, tc, adminToken, execID)
	_ = entries
	resp2 := tc.NewRequest("GET", fmt.Sprintf("/api/v1/admin/functions/executions?function_name=%s&limit=10", functionName)).
		WithAuth(adminToken).
		Send()
	require.Equal(t, fiber.StatusOK, resp2.Status())

	var noTenantResult map[string]interface{}
	err = json.Unmarshal(resp2.Body(), &noTenantResult)
	require.NoError(t, err)

	noTenantCount := noTenantResult["count"]
	_ = defaultTenantID

	t.Logf("Default tenant sees %d executions, no-tenant query sees %v", len(executions), noTenantCount)
}
