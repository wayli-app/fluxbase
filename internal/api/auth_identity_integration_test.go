//go:build integration
// +build integration

package api_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/nimbleflux/fluxbase/internal/testutil/e2e"
)

// TestAuthHandler_GetUserIdentities_Integration tests getting user identities
func TestAuthHandler_GetUserIdentities_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Create a regular user
	email := randomEmail()
	_, token := tc.CreateTestUser(email, "password123")

	// Get user identities (should be empty for new user)
	resp := tc.NewRequest("GET", "/api/v1/auth/user/identities").
		WithAuth(token).
		Send().
		AssertStatus(200)

	var result map[string]interface{}
	resp.JSON(&result)

	// Should have an identities array (may be empty or null)
	assert.Contains(t, result, "identities")
	// Identities can be nil (empty) or an empty array
	identities := result["identities"]
	if identities != nil {
		identityArray, ok := identities.([]interface{})
		assert.True(t, ok, "identities should be an array")
		// New user should have no linked identities
		assert.Empty(t, identityArray, "New user should have no linked identities")
	}
}

// TestAuthHandler_GetUserIdentities_Unauthenticated_Integration tests getting identities without auth
func TestAuthHandler_GetUserIdentities_Unauthenticated_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Try to get identities without authentication
	resp := tc.NewRequest("GET", "/api/v1/auth/user/identities").
		Send()

	// Should fail with 401
	assert.Equal(t, 401, resp.Status(), "Should require authentication")
}

// TestAuthHandler_LinkIdentity_Validation_Integration tests identity link validation
func TestAuthHandler_LinkIdentity_Validation_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Create a regular user
	email := randomEmail()
	_, token := tc.CreateTestUser(email, "password123")

	// Try to link identity without provider
	resp := tc.NewRequest("POST", "/api/v1/auth/user/identities").
		WithAuth(token).
		WithBody(map[string]interface{}{
			// Missing provider field
		}).
		Send()

	// Should fail with 400
	assert.Contains(t, []int{400, 422}, resp.Status(), "Should reject request without provider")
}

// TestAuthHandler_LinkIdentity_InvalidProvider_Integration tests linking with invalid provider
func TestAuthHandler_LinkIdentity_InvalidProvider_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Create a regular user
	email := randomEmail()
	_, token := tc.CreateTestUser(email, "password123")

	// Try to link identity with non-existent provider
	resp := tc.NewRequest("POST", "/api/v1/auth/user/identities").
		WithAuth(token).
		WithBody(map[string]interface{}{
			"provider": "nonexistent_provider",
			"options": map[string]interface{}{
				"redirect_to": "http://localhost:3000",
			},
		}).
		Send()

	// Should fail with 400 or 404
	assert.Contains(t, []int{400, 404}, resp.Status(), "Should reject non-existent provider")
}

// TestAuthHandler_UnlinkIdentity_NotFound_Integration tests unlinking non-existent identity
func TestAuthHandler_UnlinkIdentity_NotFound_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Create a regular user
	email := randomEmail()
	_, token := tc.CreateTestUser(email, "password123")

	// Try to unlink non-existent identity
	fakeIdentityID := uuid.New().String()
	resp := tc.NewRequest("DELETE", "/api/v1/auth/user/identities/"+fakeIdentityID).
		WithAuth(token).
		Send()

	// Should fail with 400 (identity not found returns bad request)
	assert.Equal(t, 400, resp.Status(), "Should return 400 for non-existent identity")
}

// TestAuthHandler_UnlinkIdentity_Unauthenticated_Integration tests unlinking without auth
func TestAuthHandler_UnlinkIdentity_Unauthenticated_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Try to unlink identity without authentication
	resp := tc.NewRequest("DELETE", "/api/v1/auth/user/identities/"+uuid.New().String()).
		Send()

	// Should fail with 401 or 403 (forbidden)
	assert.Contains(t, []int{401, 403}, resp.Status(), "Should require authentication")
}
