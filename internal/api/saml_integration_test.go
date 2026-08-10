//go:build integration
// +build integration

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nimbleflux/fluxbase/internal/testutil/e2e"
)

// TestSAMLProvider_ListProviders_Integration tests listing all SAML providers
func TestSAMLProvider_ListProviders_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Create dashboard admin user
	adminEmail := randomEmail()
	_, adminToken := tc.CreateDashboardAdminUser(adminEmail, "password123")

	// List SAML providers
	resp := tc.NewRequest("GET", "/api/v1/admin/saml/providers").
		WithAuth(adminToken).
		Send().
		AssertStatus(200)

	var providers []map[string]interface{}
	resp.JSON(&providers)

	// Should return an array (possibly empty)
	assert.NotNil(t, providers)
}

// TestSAMLProvider_ValidateMetadata_Integration tests validating SAML metadata
func TestSAMLProvider_ValidateMetadata_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Create dashboard admin user
	adminEmail := randomEmail()
	_, adminToken := tc.CreateDashboardAdminUser(adminEmail, "password123")

	// Test with invalid XML
	resp := tc.NewRequest("POST", "/api/v1/admin/saml/validate-metadata").
		WithAuth(adminToken).
		WithBody(map[string]interface{}{
			"metadata_xml": "<not-valid-xml>",
		}).
		Send()

	// Should return an error (invalid XML)
	var result map[string]interface{}
	resp.JSON(&result)
	assert.Equal(t, false, result["valid"])
	assert.NotNil(t, result["error"])
}

// TestSAMLProvider_ValidateMetadata_MissingInput_Integration tests validating SAML metadata with missing input
func TestSAMLProvider_ValidateMetadata_MissingInput_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Create dashboard admin user
	adminEmail := randomEmail()
	_, adminToken := tc.CreateDashboardAdminUser(adminEmail, "password123")

	// Test with no metadata input
	resp := tc.NewRequest("POST", "/api/v1/admin/saml/validate-metadata").
		WithAuth(adminToken).
		WithBody(map[string]interface{}{
			// Missing both metadata_url and metadata_xml
		}).
		Send()

	// Should return 400 error
	assert.Contains(t, []int{400, 422}, resp.Status(), "Should reject request without metadata input")
}

// TestSAMLProvider_CreateProvider_Integration tests creating a SAML provider
func TestSAMLProvider_CreateProvider_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Create dashboard admin user
	adminEmail := randomEmail()
	_, adminToken := tc.CreateDashboardAdminUser(adminEmail, "password123")

	// Test with invalid provider name (doesn't match pattern)
	resp := tc.NewRequest("POST", "/api/v1/admin/saml/providers").
		WithAuth(adminToken).
		WithBody(map[string]interface{}{
			"name":             "INVALID-NAME!", // Invalid - contains special chars
			"idp_metadata_xml": "<some>xml</some>",
		}).
		Send()

	// Should return 400 error due to invalid name
	assert.Contains(t, []int{400, 422}, resp.Status(), "Should reject invalid provider name")

	var result map[string]interface{}
	resp.JSON(&result)
	assert.Contains(t, result["error"], "Provider name must start with a letter")
}

// TestSAMLProvider_CreateProvider_MissingMetadata_Integration tests creating a SAML provider without metadata
func TestSAMLProvider_CreateProvider_MissingMetadata_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Create dashboard admin user
	adminEmail := randomEmail()
	_, adminToken := tc.CreateDashboardAdminUser(adminEmail, "password123")

	// Test without metadata URL or XML
	resp := tc.NewRequest("POST", "/api/v1/admin/saml/providers").
		WithAuth(adminToken).
		WithBody(map[string]interface{}{
			"name": "test-provider",
			// Missing idp_metadata_url and idp_metadata_xml
		}).
		Send()

	// Should return 400 error
	assert.Contains(t, []int{400, 422}, resp.Status(), "Should reject request without metadata")

	var result map[string]interface{}
	resp.JSON(&result)
	assert.Contains(t, result["error"], "metadata")
}

// TestSAMLProvider_Unauthorized_Integration tests that non-admin users cannot access SAML endpoints
func TestSAMLProvider_Unauthorized_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Create a regular user (not admin)
	userEmail := randomEmail()
	_, userToken := tc.CreateTestUser(userEmail, "password123")

	// Try to list SAML providers (should fail - user is not admin)
	resp := tc.NewRequest("GET", "/api/v1/admin/saml/providers").
		WithAuth(userToken).
		Send()
	assert.Contains(t, []int{401, 403}, resp.Status(), "Regular user should not be able to list SAML providers")

	// Try to validate metadata (should fail)
	resp2 := tc.NewRequest("POST", "/api/v1/admin/saml/validate-metadata").
		WithAuth(userToken).
		WithBody(map[string]interface{}{
			"metadata_xml": "<xml></xml>",
		}).
		Send()
	assert.Contains(t, []int{401, 403}, resp2.Status(), "Regular user should not be able to validate SAML metadata")

	// Try to create provider (should fail)
	resp3 := tc.NewRequest("POST", "/api/v1/admin/saml/providers").
		WithAuth(userToken).
		WithBody(map[string]interface{}{
			"name":             "test-provider",
			"idp_metadata_xml": "<xml></xml>",
		}).
		Send()
	assert.Contains(t, []int{401, 403}, resp3.Status(), "Regular user should not be able to create SAML providers")
}
