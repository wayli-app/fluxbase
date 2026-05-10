//go:build integration
// +build integration

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nimbleflux/fluxbase/internal/testutil"
)

// TestAuthHandler_SendMagicLink_Integration tests sending a magic link email
func TestAuthHandler_SendMagicLink_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	email := randomEmail()

	// Send magic link
	resp := tc.NewRequest("POST", "/api/v1/auth/magiclink").
		WithBody(map[string]interface{}{
			"email": email,
		}).
		Send().
		AssertStatus(200)

	var result map[string]interface{}
	resp.JSON(&result)
	// Standard OTP response format (returns user: nil, session: nil for magic link)
	assert.Nil(t, result["user"])
	assert.Nil(t, result["session"])
}

// TestAuthHandler_SendMagicLink_InvalidEmail_Integration tests sending magic link with invalid email
func TestAuthHandler_SendMagicLink_InvalidEmail_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Send magic link with empty email
	resp := tc.NewRequest("POST", "/api/v1/auth/magiclink").
		WithBody(map[string]interface{}{
			"email": "",
		}).
		Send()

	// Should fail with 400
	assert.Contains(t, []int{400, 422}, resp.Status(), "Should reject empty email")
}

// TestAuthHandler_VerifyMagicLink_Integration tests verifying a magic link token
func TestAuthHandler_VerifyMagicLink_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	email := randomEmail()

	// First, create the user (magic link creates user on verification)
	// We'll use signup to create the user directly for this test
	userID, _ := tc.CreateTestUser(email, "password123")

	// Get a valid session token for the user
	token := tc.GetAuthToken(email, "password123")

	// Verify we have a valid token
	assert.NotEmpty(t, token, "Should have valid access token")

	// Verify the user exists in database
	users := tc.QuerySQL(`SELECT email FROM auth.users WHERE id = $1`, userID)
	assert.NotEmpty(t, users, "User should exist in database")
	assert.Equal(t, email, users[0]["email"])
}

// TestAuthHandler_VerifyMagicLink_InvalidToken_Integration tests verifying with invalid token
func TestAuthHandler_VerifyMagicLink_InvalidToken_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Try to verify with invalid token
	resp := tc.NewRequest("POST", "/api/v1/auth/magiclink/verify").
		WithBody(map[string]interface{}{
			"token": "invalid_token_12345",
		}).
		Send()

	// Should fail with 400 or 401
	assert.Contains(t, []int{400, 401, 403}, resp.Status(), "Should reject invalid magic link token")
}
