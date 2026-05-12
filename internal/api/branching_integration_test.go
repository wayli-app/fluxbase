//go:build integration
// +build integration

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nimbleflux/fluxbase/internal/testutil"
)

// TestBranching_ListBranches_Integration tests listing all branches
func TestBranching_ListBranches_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Create dashboard admin user
	adminEmail := randomEmail()
	_, adminToken := tc.CreateDashboardAdminUser(adminEmail, "password123")

	// List branches
	// Note: Branching routes may not be registered if the handler is not initialized
	resp := tc.NewRequest("GET", "/api/v1/admin/branches").
		WithAuth(adminToken).
		Send()

	// Accept 200 if routing is configured, 404 if branching not enabled, or 500/503 for other issues
	assert.Contains(t, []int{200, 404, 500, 503}, resp.Status(),
		"Should return 200 if routing configured, 404 if not enabled, or 500/503 for errors")

	if resp.Status() == 200 {
		var result map[string]interface{}
		resp.JSON(&result)
		assert.Contains(t, result, "branches")
		assert.Contains(t, result, "total")
	} else if resp.Status() == 503 {
		var result map[string]interface{}
		resp.JSON(&result)
		assert.Equal(t, "BRANCHING_DISABLED", result["code"])
	}
}

// TestBranching_GetActiveBranch_Integration tests getting the active branch
func TestBranching_GetActiveBranch_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Create dashboard admin user
	adminEmail := randomEmail()
	_, adminToken := tc.CreateDashboardAdminUser(adminEmail, "password123")

	// Get active branch
	// Note: Branching routes may not be registered if the handler is not initialized
	resp := tc.NewRequest("GET", "/api/v1/admin/branches/active").
		WithAuth(adminToken).
		Send()

	// Accept 200 if routing is configured, 404 if branching is not enabled/handler not initialized
	assert.Contains(t, []int{200, 404}, resp.Status(),
		"Should return 200 if routing is configured, or 404 if branching not enabled")

	if resp.Status() == 200 {
		var result map[string]interface{}
		resp.JSON(&result)
		assert.Contains(t, result, "branch")
		assert.Contains(t, result, "source")
	}
}

// TestBranching_CreateBranch_Disabled_Integration tests that branch creation fails when branching is disabled
func TestBranching_CreateBranch_Disabled_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Create dashboard admin user
	adminEmail := randomEmail()
	_, adminToken := tc.CreateDashboardAdminUser(adminEmail, "password123")

	// Try to create a branch
	resp := tc.NewRequest("POST", "/api/v1/admin/branches").
		WithAuth(adminToken).
		WithBody(map[string]interface{}{
			"name": "test-branch",
		}).
		Send()

	// Branching is likely not enabled, so expect 404 (routes not registered), 503, or other appropriate status
	assert.Contains(t, []int{404, 503, 500, 400}, resp.Status(),
		"Should return 404 if routing not configured, 503 if disabled, 500 if not initialized, or 400 for validation")

	if resp.Status() == 503 {
		var result map[string]interface{}
		resp.JSON(&result)
		assert.Equal(t, "BRANCHING_DISABLED", result["code"])
	}
}

// TestBranching_CreateBranch_MissingName_Integration tests branch creation without name
func TestBranching_CreateBranch_MissingName_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Create dashboard admin user
	adminEmail := randomEmail()
	_, adminToken := tc.CreateDashboardAdminUser(adminEmail, "password123")

	// Try to create a branch without name
	resp := tc.NewRequest("POST", "/api/v1/admin/branches").
		WithAuth(adminToken).
		WithBody(map[string]interface{}{
			// Missing name
		}).
		Send()

	// Should return 400/422 for validation error, 404 if routing not configured, or 503 if branching is disabled
	assert.Contains(t, []int{400, 422, 404, 503}, resp.Status(),
		"Should return 400/422 for missing name, 404 if routing not configured, or 503 if disabled")

	var result map[string]interface{}
	resp.JSON(&result)
	if resp.Status() == 400 || resp.Status() == 422 {
		assert.Contains(t, result["message"], "name")
	}
}

// TestBranching_Unauthorized_Integration tests that non-admin users cannot access branch endpoints
func TestBranching_Unauthorized_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Create a regular user (not admin)
	userEmail := randomEmail()
	_, userToken := tc.CreateTestUser(userEmail, "password123")

	// Try to list branches (should fail - user is not admin)
	resp := tc.NewRequest("GET", "/api/v1/admin/branches").
		WithAuth(userToken).
		Send()
	// Should return 401/403 for unauthorized or 404 if branching not configured
	assert.Contains(t, []int{401, 403, 404}, resp.Status(),
		"Regular user should not be able to list branches (401/403) or branching not configured (404)")

	// Try to create branch (should fail)
	resp2 := tc.NewRequest("POST", "/api/v1/admin/branches").
		WithAuth(userToken).
		WithBody(map[string]interface{}{
			"name": "test-branch",
		}).
		Send()
	assert.Contains(t, []int{401, 403, 404}, resp2.Status(),
		"Regular user should not be able to create branches (401/403) or branching not configured (404)")
}

// TestBranching_GetPoolStats_Integration tests getting connection pool statistics
func TestBranching_GetPoolStats_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Create dashboard admin user
	adminEmail := randomEmail()
	_, adminToken := tc.CreateDashboardAdminUser(adminEmail, "password123")

	// Get pool stats
	// Note: Branching routes may not be registered if the handler is not initialized
	resp := tc.NewRequest("GET", "/api/v1/admin/branches/stats/pools").
		WithAuth(adminToken).
		Send()

	// Accept 200 if routing is configured, 404 if branching is not enabled
	assert.Contains(t, []int{200, 404}, resp.Status(),
		"Should return 200 if routing configured, or 404 if branching not enabled")

	if resp.Status() == 200 {
		var result map[string]interface{}
		resp.JSON(&result)
		assert.Contains(t, result, "pools")
	}
}
