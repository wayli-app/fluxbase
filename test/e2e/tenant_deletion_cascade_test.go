//go:build integration

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/test"
)

func TestTenantDeletion_CascadeCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test")
	}

	tc := test.NewTestContext(t)

	// Create a unique tenant via API
	email := test.E2ETestEmailWithSuffix("cascade-admin")
	_, token := tc.CreateDashboardAdminUser(email, "Test-password-32chars!!")

	tenantSlug := "e2e-cascade-" + test.E2ETestEmail()[:8]

	// Create tenant via API
	resp := tc.NewRequest("POST", "/api/v1/admin/tenants").
		WithAuth(token).
		WithBody(map[string]interface{}{
			"name":               "Cascade Test Tenant",
			"slug":               tenantSlug,
			"auto_generate_keys": true,
		}).Send()
	require.Equal(t, 201, resp.Status())

	var tenantResp map[string]interface{}
	resp.JSON(&tenantResp)
	tenantData := tenantResp["tenant"].(map[string]interface{})
	tenantID := tenantData["id"].(string)

	// Insert data in multiple schemas for this tenant
	// Jobs function
	tc.ExecuteSQLAsSuperuser(`
		INSERT INTO jobs.functions (id, name, namespace, code, enabled, tenant_id)
		VALUES ($1, 'cascade-job', 'test', 'export default {}', true, $2::uuid)
	`, "30000000-0000-0000-0000-000000000001", tenantID)

	// RPC procedure
	tc.ExecuteSQLAsSuperuser(`
		INSERT INTO rpc.procedures (id, name, namespace, sql_query, enabled, tenant_id)
		VALUES ($1, 'cascade_proc', 'test', 'SELECT 1', true, $2::uuid)
	`, "30000000-0000-0000-0000-000000000002", tenantID)

	// Verify data exists before deletion
	rowsBefore := tc.QuerySQLAsSuperuser(
		"SELECT count(*) as cnt FROM jobs.functions WHERE tenant_id = $1::uuid", tenantID,
	)
	assert.Equal(t, int64(1), rowsBefore[0]["cnt"].(int64), "job function should exist before deletion")

	// Delete the tenant via API
	delResp := tc.NewRequest("DELETE", "/api/v1/admin/tenants/"+tenantID+"?hard=true").
		WithAuth(token).
		Send()
	require.True(t, delResp.Status() < 300,
		"tenant deletion should succeed, got status %d", delResp.Status())

	// Verify data is cleaned up
	rowsAfterJobs := tc.QuerySQLAsSuperuser(
		"SELECT count(*) as cnt FROM jobs.functions WHERE tenant_id = $1::uuid", tenantID,
	)
	assert.Equal(t, int64(0), rowsAfterJobs[0]["cnt"].(int64),
		"job functions should be deleted when tenant is deleted")

	rowsAfterRPC := tc.QuerySQLAsSuperuser(
		"SELECT count(*) as cnt FROM rpc.procedures WHERE tenant_id = $1::uuid", tenantID,
	)
	assert.Equal(t, int64(0), rowsAfterRPC[0]["cnt"].(int64),
		"rpc procedures should be deleted when tenant is deleted")
}
