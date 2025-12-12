// Package e2e tests the RPC (Remote Procedure Call) functionality
package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/fluxbase-eu/fluxbase/test"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

// TestRPCAdminSync tests the admin sync endpoint for RPC procedures
func TestRPCAdminSync(t *testing.T) {
	// Skip if RPC feature is not enabled
	t.Skip("RPC feature requires database setup - skipping integration test")

	tc := test.NewTestContext(t)
	defer tc.Close()
	tc.EnsureAuthSchema()

	adminToken := tc.GetDashboardAuthToken("admin@test.com", "SecurePassword123!")

	// Test syncing with a procedure via API payload
	resp := tc.NewRequest("POST", "/api/v1/admin/rpc/sync").
		WithAuth(adminToken).
		WithJSON(map[string]interface{}{
			"namespace": "default",
			"procedures": []map[string]interface{}{
				{
					"name": "get-test-data",
					"code": `-- @fluxbase:name get-test-data
-- @fluxbase:description Get test data
-- @fluxbase:input {"limit?": "number"}
-- @fluxbase:public true
SELECT json_build_object('message', 'hello', 'count', COALESCE($limit, 10));`,
				},
			},
			"options": map[string]interface{}{
				"delete_missing": false,
				"dry_run":        false,
			},
		}).
		Send()

	resp.AssertStatus(fiber.StatusOK)

	var result map[string]interface{}
	resp.JSON(&result)

	require.NotNil(t, result["summary"], "Response should contain summary")
}

// TestRPCAdminSyncDryRun tests the dry run mode of sync
func TestRPCAdminSyncDryRun(t *testing.T) {
	t.Skip("RPC feature requires database setup - skipping integration test")

	tc := test.NewTestContext(t)
	defer tc.Close()
	tc.EnsureAuthSchema()

	adminToken := tc.GetDashboardAuthToken("admin@test.com", "SecurePassword123!")

	// Sync with dry_run=true
	resp := tc.NewRequest("POST", "/api/v1/admin/rpc/sync").
		WithAuth(adminToken).
		WithJSON(map[string]interface{}{
			"namespace": "default",
			"procedures": []map[string]interface{}{
				{
					"name": "dry-run-test",
					"code": `-- @fluxbase:name dry-run-test
SELECT 1;`,
				},
			},
			"options": map[string]interface{}{
				"dry_run": true,
			},
		}).
		Send()

	resp.AssertStatus(fiber.StatusOK)

	var result map[string]interface{}
	resp.JSON(&result)

	require.True(t, result["dry_run"].(bool), "Response should indicate dry run mode")
}

// TestRPCAdminListProcedures tests listing all procedures
func TestRPCAdminListProcedures(t *testing.T) {
	t.Skip("RPC feature requires database setup - skipping integration test")

	tc := test.NewTestContext(t)
	defer tc.Close()
	tc.EnsureAuthSchema()

	adminToken := tc.GetDashboardAuthToken("admin@test.com", "SecurePassword123!")

	resp := tc.NewRequest("GET", "/api/v1/admin/rpc/procedures").
		WithAuth(adminToken).
		Send()

	resp.AssertStatus(fiber.StatusOK)

	var result map[string]interface{}
	resp.JSON(&result)

	require.NotNil(t, result["procedures"], "Response should contain procedures array")
	require.NotNil(t, result["count"], "Response should contain count")
}

// TestRPCAdminListNamespaces tests listing all namespaces
func TestRPCAdminListNamespaces(t *testing.T) {
	t.Skip("RPC feature requires database setup - skipping integration test")

	tc := test.NewTestContext(t)
	defer tc.Close()
	tc.EnsureAuthSchema()

	adminToken := tc.GetDashboardAuthToken("admin@test.com", "SecurePassword123!")

	resp := tc.NewRequest("GET", "/api/v1/admin/rpc/namespaces").
		WithAuth(adminToken).
		Send()

	resp.AssertStatus(fiber.StatusOK)

	var result map[string]interface{}
	resp.JSON(&result)

	require.NotNil(t, result["namespaces"], "Response should contain namespaces array")
}

// TestRPCPublicListProcedures tests listing public procedures
func TestRPCPublicListProcedures(t *testing.T) {
	t.Skip("RPC feature requires database setup - skipping integration test")

	tc := test.NewTestContext(t)
	defer tc.Close()

	apiKey := tc.CreateAPIKey("RPC Test Key", nil)

	resp := tc.NewRequest("GET", "/api/v1/rpc/procedures").
		WithAPIKey(apiKey).
		Send()

	resp.AssertStatus(fiber.StatusOK)

	var result map[string]interface{}
	resp.JSON(&result)

	require.NotNil(t, result["procedures"], "Response should contain procedures array")
}

// TestRPCInvokeSync tests synchronous RPC invocation
func TestRPCInvokeSync(t *testing.T) {
	t.Skip("RPC feature requires database setup - skipping integration test")

	tc := test.NewTestContext(t)
	defer tc.Close()
	tc.EnsureAuthSchema()

	adminToken := tc.GetDashboardAuthToken("admin@test.com", "SecurePassword123!")

	// First, sync a test procedure
	syncResp := tc.NewRequest("POST", "/api/v1/admin/rpc/sync").
		WithAuth(adminToken).
		WithJSON(map[string]interface{}{
			"namespace": "default",
			"procedures": []map[string]interface{}{
				{
					"name": "sync-invoke-test",
					"code": `-- @fluxbase:name sync-invoke-test
-- @fluxbase:description Test sync invocation
-- @fluxbase:public true
SELECT json_build_object('result', 'success', 'timestamp', NOW());`,
				},
			},
		}).
		Send()
	syncResp.AssertStatus(fiber.StatusOK)

	// Now invoke the procedure
	apiKey := tc.CreateAPIKey("RPC Invoke Key", nil)

	resp := tc.NewRequest("POST", "/api/v1/rpc/default/sync-invoke-test").
		WithAPIKey(apiKey).
		WithJSON(map[string]interface{}{
			"params": map[string]interface{}{},
		}).
		Send()

	resp.AssertStatus(fiber.StatusOK)

	var result map[string]interface{}
	resp.JSON(&result)

	require.Equal(t, "completed", result["status"], "Status should be completed for sync invocation")
	require.NotNil(t, result["result"], "Response should contain result")
	require.NotNil(t, result["execution_id"], "Response should contain execution_id")
}

// TestRPCInvokeAsync tests asynchronous RPC invocation
func TestRPCInvokeAsync(t *testing.T) {
	t.Skip("RPC feature requires database setup - skipping integration test")

	tc := test.NewTestContext(t)
	defer tc.Close()
	tc.EnsureAuthSchema()

	adminToken := tc.GetDashboardAuthToken("admin@test.com", "SecurePassword123!")

	// Sync a test procedure
	syncResp := tc.NewRequest("POST", "/api/v1/admin/rpc/sync").
		WithAuth(adminToken).
		WithJSON(map[string]interface{}{
			"namespace": "default",
			"procedures": []map[string]interface{}{
				{
					"name": "async-invoke-test",
					"code": `-- @fluxbase:name async-invoke-test
-- @fluxbase:description Test async invocation
-- @fluxbase:public true
SELECT pg_sleep(0.1), json_build_object('result', 'async-success');`,
				},
			},
		}).
		Send()
	syncResp.AssertStatus(fiber.StatusOK)

	apiKey := tc.CreateAPIKey("RPC Async Key", nil)

	// Invoke asynchronously
	resp := tc.NewRequest("POST", "/api/v1/rpc/default/async-invoke-test").
		WithAPIKey(apiKey).
		WithJSON(map[string]interface{}{
			"params": map[string]interface{}{},
			"async":  true,
		}).
		Send()

	resp.AssertStatus(fiber.StatusOK)

	var result map[string]interface{}
	resp.JSON(&result)

	require.Equal(t, "pending", result["status"], "Status should be pending for async invocation")
	require.NotNil(t, result["execution_id"], "Response should contain execution_id")

	executionID := result["execution_id"].(string)

	// Poll for completion
	time.Sleep(200 * time.Millisecond)

	statusResp := tc.NewRequest("GET", fmt.Sprintf("/api/v1/rpc/executions/%s", executionID)).
		WithAPIKey(apiKey).
		Send()

	statusResp.AssertStatus(fiber.StatusOK)

	var statusResult map[string]interface{}
	statusResp.JSON(&statusResult)

	require.Contains(t, []string{"running", "completed"}, statusResult["status"],
		"Status should be running or completed")
}

// TestRPCInputValidation tests input schema validation
func TestRPCInputValidation(t *testing.T) {
	t.Skip("RPC feature requires database setup - skipping integration test")

	tc := test.NewTestContext(t)
	defer tc.Close()
	tc.EnsureAuthSchema()

	adminToken := tc.GetDashboardAuthToken("admin@test.com", "SecurePassword123!")

	// Sync a procedure with input schema
	syncResp := tc.NewRequest("POST", "/api/v1/admin/rpc/sync").
		WithAuth(adminToken).
		WithJSON(map[string]interface{}{
			"namespace": "default",
			"procedures": []map[string]interface{}{
				{
					"name": "validation-test",
					"code": `-- @fluxbase:name validation-test
-- @fluxbase:description Test input validation
-- @fluxbase:input {"user_id": "uuid", "limit": "number"}
-- @fluxbase:public true
SELECT json_build_object('user_id', $user_id, 'limit', $limit);`,
				},
			},
		}).
		Send()
	syncResp.AssertStatus(fiber.StatusOK)

	apiKey := tc.CreateAPIKey("Validation Test Key", nil)

	t.Run("MissingRequiredParam", func(t *testing.T) {
		// Invoke without required user_id
		resp := tc.NewRequest("POST", "/api/v1/rpc/default/validation-test").
			WithAPIKey(apiKey).
			WithJSON(map[string]interface{}{
				"params": map[string]interface{}{
					"limit": 10,
				},
			}).
			Send()

		// Should fail validation
		resp.AssertStatus(fiber.StatusBadRequest)

		var result map[string]interface{}
		resp.JSON(&result)
		require.Contains(t, result["error"].(string), "user_id", "Error should mention missing user_id")
	})

	t.Run("InvalidParamType", func(t *testing.T) {
		// Invoke with wrong type for limit
		resp := tc.NewRequest("POST", "/api/v1/rpc/default/validation-test").
			WithAPIKey(apiKey).
			WithJSON(map[string]interface{}{
				"params": map[string]interface{}{
					"user_id": "550e8400-e29b-41d4-a716-446655440000",
					"limit":   "not-a-number",
				},
			}).
			Send()

		// Should fail validation
		resp.AssertStatus(fiber.StatusBadRequest)
	})

	t.Run("ValidParams", func(t *testing.T) {
		// Invoke with valid params
		resp := tc.NewRequest("POST", "/api/v1/rpc/default/validation-test").
			WithAPIKey(apiKey).
			WithJSON(map[string]interface{}{
				"params": map[string]interface{}{
					"user_id": "550e8400-e29b-41d4-a716-446655440000",
					"limit":   10,
				},
			}).
			Send()

		resp.AssertStatus(fiber.StatusOK)
	})
}

// TestRPCRoleRestrictions tests role-based access control
func TestRPCRoleRestrictions(t *testing.T) {
	t.Skip("RPC feature requires database setup - skipping integration test")

	tc := test.NewTestContext(t)
	defer tc.Close()
	tc.EnsureAuthSchema()

	adminToken := tc.GetDashboardAuthToken("admin@test.com", "SecurePassword123!")

	// Sync a procedure requiring admin role
	syncResp := tc.NewRequest("POST", "/api/v1/admin/rpc/sync").
		WithAuth(adminToken).
		WithJSON(map[string]interface{}{
			"namespace": "default",
			"procedures": []map[string]interface{}{
				{
					"name": "admin-only-proc",
					"code": `-- @fluxbase:name admin-only-proc
-- @fluxbase:description Admin only procedure
-- @fluxbase:require-role admin
SELECT json_build_object('secret', 'admin-data');`,
				},
			},
		}).
		Send()
	syncResp.AssertStatus(fiber.StatusOK)

	t.Run("RegularUserDenied", func(t *testing.T) {
		timestamp := time.Now().UnixNano()
		userEmail := fmt.Sprintf("user-%d@test.com", timestamp)
		_, userToken := tc.CreateUser(userEmail, "password123")

		resp := tc.NewRequest("POST", "/api/v1/rpc/default/admin-only-proc").
			WithAuth(userToken).
			WithJSON(map[string]interface{}{
				"params": map[string]interface{}{},
			}).
			Send()

		resp.AssertStatus(fiber.StatusForbidden)
	})

	t.Run("ServiceRoleAllowed", func(t *testing.T) {
		serviceKey := tc.CreateServiceKey("test-service-key")

		resp := tc.NewRequest("POST", "/api/v1/rpc/default/admin-only-proc").
			WithAuth(serviceKey).
			WithJSON(map[string]interface{}{
				"params": map[string]interface{}{},
			}).
			Send()

		// Service role should bypass role restrictions
		resp.AssertStatus(fiber.StatusOK)
	})
}

// TestRPCPublicAccess tests public procedure access
func TestRPCPublicAccess(t *testing.T) {
	t.Skip("RPC feature requires database setup - skipping integration test")

	tc := test.NewTestContext(t)
	defer tc.Close()
	tc.EnsureAuthSchema()

	adminToken := tc.GetDashboardAuthToken("admin@test.com", "SecurePassword123!")

	// Sync both public and private procedures
	syncResp := tc.NewRequest("POST", "/api/v1/admin/rpc/sync").
		WithAuth(adminToken).
		WithJSON(map[string]interface{}{
			"namespace": "default",
			"procedures": []map[string]interface{}{
				{
					"name": "public-proc",
					"code": `-- @fluxbase:name public-proc
-- @fluxbase:public true
SELECT json_build_object('message', 'public');`,
				},
				{
					"name": "private-proc",
					"code": `-- @fluxbase:name private-proc
-- @fluxbase:public false
SELECT json_build_object('message', 'private');`,
				},
			},
		}).
		Send()
	syncResp.AssertStatus(fiber.StatusOK)

	anonKey := tc.GenerateAnonKey()

	t.Run("PublicProcAccessible", func(t *testing.T) {
		resp := tc.NewRequest("POST", "/api/v1/rpc/default/public-proc").
			WithAuth(anonKey).
			WithJSON(map[string]interface{}{
				"params": map[string]interface{}{},
			}).
			Send()

		resp.AssertStatus(fiber.StatusOK)
	})

	t.Run("PrivateProcRequiresAuth", func(t *testing.T) {
		resp := tc.NewRequest("POST", "/api/v1/rpc/default/private-proc").
			WithAuth(anonKey).
			WithJSON(map[string]interface{}{
				"params": map[string]interface{}{},
			}).
			Send()

		resp.AssertStatus(fiber.StatusUnauthorized)
	})
}

// TestRPCExecutionHistory tests execution history tracking
func TestRPCExecutionHistory(t *testing.T) {
	t.Skip("RPC feature requires database setup - skipping integration test")

	tc := test.NewTestContext(t)
	defer tc.Close()
	tc.EnsureAuthSchema()

	adminToken := tc.GetDashboardAuthToken("admin@test.com", "SecurePassword123!")

	// Sync a test procedure
	syncResp := tc.NewRequest("POST", "/api/v1/admin/rpc/sync").
		WithAuth(adminToken).
		WithJSON(map[string]interface{}{
			"namespace": "default",
			"procedures": []map[string]interface{}{
				{
					"name": "history-test",
					"code": `-- @fluxbase:name history-test
-- @fluxbase:public true
SELECT json_build_object('timestamp', NOW());`,
				},
			},
		}).
		Send()
	syncResp.AssertStatus(fiber.StatusOK)

	apiKey := tc.CreateAPIKey("History Test Key", nil)

	// Invoke the procedure a few times
	for i := 0; i < 3; i++ {
		tc.NewRequest("POST", "/api/v1/rpc/default/history-test").
			WithAPIKey(apiKey).
			WithJSON(map[string]interface{}{"params": map[string]interface{}{}}).
			Send()
	}

	// Check admin execution history
	resp := tc.NewRequest("GET", "/api/v1/admin/rpc/executions?procedure_name=history-test").
		WithAuth(adminToken).
		Send()

	resp.AssertStatus(fiber.StatusOK)

	var result map[string]interface{}
	resp.JSON(&result)

	executions := result["executions"].([]interface{})
	require.GreaterOrEqual(t, len(executions), 3, "Should have at least 3 executions")
}

// TestRPCExecutionLogs tests execution log retrieval
func TestRPCExecutionLogs(t *testing.T) {
	t.Skip("RPC feature requires database setup - skipping integration test")

	tc := test.NewTestContext(t)
	defer tc.Close()
	tc.EnsureAuthSchema()

	adminToken := tc.GetDashboardAuthToken("admin@test.com", "SecurePassword123!")

	// Sync a test procedure
	syncResp := tc.NewRequest("POST", "/api/v1/admin/rpc/sync").
		WithAuth(adminToken).
		WithJSON(map[string]interface{}{
			"namespace": "default",
			"procedures": []map[string]interface{}{
				{
					"name": "logs-test",
					"code": `-- @fluxbase:name logs-test
-- @fluxbase:public true
SELECT json_build_object('done', true);`,
				},
			},
		}).
		Send()
	syncResp.AssertStatus(fiber.StatusOK)

	apiKey := tc.CreateAPIKey("Logs Test Key", nil)

	// Invoke the procedure
	invokeResp := tc.NewRequest("POST", "/api/v1/rpc/default/logs-test").
		WithAPIKey(apiKey).
		WithJSON(map[string]interface{}{"params": map[string]interface{}{}}).
		Send()
	invokeResp.AssertStatus(fiber.StatusOK)

	var invokeResult map[string]interface{}
	invokeResp.JSON(&invokeResult)
	executionID := invokeResult["execution_id"].(string)

	// Get execution logs
	logsResp := tc.NewRequest("GET", fmt.Sprintf("/api/v1/admin/rpc/executions/%s/logs", executionID)).
		WithAuth(adminToken).
		Send()

	logsResp.AssertStatus(fiber.StatusOK)

	var logsResult map[string]interface{}
	logsResp.JSON(&logsResult)

	require.NotNil(t, logsResult["logs"], "Response should contain logs array")
	logs := logsResult["logs"].([]interface{})
	require.Greater(t, len(logs), 0, "Should have at least one log entry")
}

// TestRPCNamespaceIsolation tests that procedures are isolated by namespace
func TestRPCNamespaceIsolation(t *testing.T) {
	t.Skip("RPC feature requires database setup - skipping integration test")

	tc := test.NewTestContext(t)
	defer tc.Close()
	tc.EnsureAuthSchema()

	adminToken := tc.GetDashboardAuthToken("admin@test.com", "SecurePassword123!")

	// Sync procedures to different namespaces
	syncResp := tc.NewRequest("POST", "/api/v1/admin/rpc/sync").
		WithAuth(adminToken).
		WithJSON(map[string]interface{}{
			"namespace": "namespace-a",
			"procedures": []map[string]interface{}{
				{
					"name": "same-name",
					"code": `-- @fluxbase:name same-name
-- @fluxbase:public true
SELECT json_build_object('namespace', 'a');`,
				},
			},
		}).
		Send()
	syncResp.AssertStatus(fiber.StatusOK)

	syncResp2 := tc.NewRequest("POST", "/api/v1/admin/rpc/sync").
		WithAuth(adminToken).
		WithJSON(map[string]interface{}{
			"namespace": "namespace-b",
			"procedures": []map[string]interface{}{
				{
					"name": "same-name",
					"code": `-- @fluxbase:name same-name
-- @fluxbase:public true
SELECT json_build_object('namespace', 'b');`,
				},
			},
		}).
		Send()
	syncResp2.AssertStatus(fiber.StatusOK)

	apiKey := tc.CreateAPIKey("Namespace Test Key", nil)

	// Invoke from namespace-a
	respA := tc.NewRequest("POST", "/api/v1/rpc/namespace-a/same-name").
		WithAPIKey(apiKey).
		WithJSON(map[string]interface{}{"params": map[string]interface{}{}}).
		Send()
	respA.AssertStatus(fiber.StatusOK)

	var resultA map[string]interface{}
	respA.JSON(&resultA)
	dataA := resultA["result"].([]interface{})[0].(map[string]interface{})
	require.Equal(t, "a", dataA["namespace"], "Should return from namespace-a")

	// Invoke from namespace-b
	respB := tc.NewRequest("POST", "/api/v1/rpc/namespace-b/same-name").
		WithAPIKey(apiKey).
		WithJSON(map[string]interface{}{"params": map[string]interface{}{}}).
		Send()
	respB.AssertStatus(fiber.StatusOK)

	var resultB map[string]interface{}
	respB.JSON(&resultB)
	dataB := resultB["result"].([]interface{})[0].(map[string]interface{})
	require.Equal(t, "b", dataB["namespace"], "Should return from namespace-b")
}

// TestRPCCallerContext tests that caller context is injected
func TestRPCCallerContext(t *testing.T) {
	t.Skip("RPC feature requires database setup - skipping integration test")

	tc := test.NewTestContext(t)
	defer tc.Close()
	tc.EnsureAuthSchema()

	adminToken := tc.GetDashboardAuthToken("admin@test.com", "SecurePassword123!")

	// Sync a procedure that uses caller context
	syncResp := tc.NewRequest("POST", "/api/v1/admin/rpc/sync").
		WithAuth(adminToken).
		WithJSON(map[string]interface{}{
			"namespace": "default",
			"procedures": []map[string]interface{}{
				{
					"name": "caller-context-test",
					"code": `-- @fluxbase:name caller-context-test
-- @fluxbase:public true
SELECT json_build_object(
    'caller_id', $caller_id,
    'caller_role', $caller_role
);`,
				},
			},
		}).
		Send()
	syncResp.AssertStatus(fiber.StatusOK)

	// Create a user and invoke
	timestamp := time.Now().UnixNano()
	userEmail := fmt.Sprintf("caller-%d@test.com", timestamp)
	userID, userToken := tc.CreateUser(userEmail, "password123")

	resp := tc.NewRequest("POST", "/api/v1/rpc/default/caller-context-test").
		WithAuth(userToken).
		WithJSON(map[string]interface{}{"params": map[string]interface{}{}}).
		Send()

	resp.AssertStatus(fiber.StatusOK)

	var result map[string]interface{}
	resp.JSON(&result)

	data := result["result"].([]interface{})[0].(map[string]interface{})
	require.Equal(t, userID, data["caller_id"], "Caller ID should match user ID")
	require.NotEmpty(t, data["caller_role"], "Caller role should be set")
}

// TestRPCUnauthenticatedAccess tests that unauthenticated access is blocked for admin endpoints
func TestRPCUnauthenticatedAccess(t *testing.T) {
	t.Skip("RPC feature requires database setup - skipping integration test")

	tc := test.NewTestContext(t)
	defer tc.Close()

	t.Run("AdminSyncRequiresAuth", func(t *testing.T) {
		resp := tc.NewRequest("POST", "/api/v1/admin/rpc/sync").
			Unauthenticated().
			WithJSON(map[string]interface{}{}).
			Send()

		resp.AssertStatus(fiber.StatusUnauthorized)
	})

	t.Run("AdminListRequiresAuth", func(t *testing.T) {
		resp := tc.NewRequest("GET", "/api/v1/admin/rpc/procedures").
			Unauthenticated().
			Send()

		resp.AssertStatus(fiber.StatusUnauthorized)
	})

	t.Run("AdminExecutionsRequiresAuth", func(t *testing.T) {
		resp := tc.NewRequest("GET", "/api/v1/admin/rpc/executions").
			Unauthenticated().
			Send()

		resp.AssertStatus(fiber.StatusUnauthorized)
	})
}

// TestRPCDisabledProcedure tests that disabled procedures cannot be invoked
func TestRPCDisabledProcedure(t *testing.T) {
	t.Skip("RPC feature requires database setup - skipping integration test")

	tc := test.NewTestContext(t)
	defer tc.Close()
	tc.EnsureAuthSchema()

	adminToken := tc.GetDashboardAuthToken("admin@test.com", "SecurePassword123!")

	// Sync a procedure
	syncResp := tc.NewRequest("POST", "/api/v1/admin/rpc/sync").
		WithAuth(adminToken).
		WithJSON(map[string]interface{}{
			"namespace": "default",
			"procedures": []map[string]interface{}{
				{
					"name": "disable-test",
					"code": `-- @fluxbase:name disable-test
-- @fluxbase:public true
SELECT 1;`,
				},
			},
		}).
		Send()
	syncResp.AssertStatus(fiber.StatusOK)

	// Disable the procedure
	updateResp := tc.NewRequest("PUT", "/api/v1/admin/rpc/procedures/default/disable-test").
		WithAuth(adminToken).
		WithJSON(map[string]interface{}{
			"enabled": false,
		}).
		Send()
	updateResp.AssertStatus(fiber.StatusOK)

	// Try to invoke
	apiKey := tc.CreateAPIKey("Disable Test Key", nil)

	invokeResp := tc.NewRequest("POST", "/api/v1/rpc/default/disable-test").
		WithAPIKey(apiKey).
		WithJSON(map[string]interface{}{"params": map[string]interface{}{}}).
		Send()

	// Should be not found or forbidden
	require.True(t, invokeResp.Status() == fiber.StatusNotFound || invokeResp.Status() == fiber.StatusForbidden,
		"Disabled procedure should not be invokable")
}

// TestRPCProcedureNotFound tests 404 for non-existent procedures
func TestRPCProcedureNotFound(t *testing.T) {
	t.Skip("RPC feature requires database setup - skipping integration test")

	tc := test.NewTestContext(t)
	defer tc.Close()

	apiKey := tc.CreateAPIKey("NotFound Test Key", nil)

	resp := tc.NewRequest("POST", "/api/v1/rpc/default/nonexistent-procedure").
		WithAPIKey(apiKey).
		WithJSON(map[string]interface{}{"params": map[string]interface{}{}}).
		Send()

	resp.AssertStatus(fiber.StatusNotFound)
}
