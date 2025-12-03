package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fluxbase-eu/fluxbase/test"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

// TestOAuthListEnabledProviders tests listing enabled OAuth providers
func TestOAuthListEnabledProviders(t *testing.T) {
	tc, adminToken := setupAdminTest(t)
	defer tc.Close()

	// Clean up any existing test providers
	cleanupOAuthProviders(t, tc)

	// First, create an OAuth provider
	createProvider(t, tc, adminToken, "github_test1", "test-client-id", "test-client-secret")

	// List enabled OAuth providers (public endpoint, no auth required)
	resp := tc.NewRequest("GET", "/api/v1/auth/oauth/providers").
		Send().
		AssertStatus(fiber.StatusOK)

	var result map[string]interface{}
	resp.JSON(&result)

	providers, ok := result["providers"].([]interface{})
	require.True(t, ok, "Should have providers array")
	require.GreaterOrEqual(t, len(providers), 1, "Should have at least one provider")

	t.Logf("OAuth providers listed: %d providers", len(providers))
}

// TestOAuthAuthorizeRedirect tests OAuth authorization redirect
func TestOAuthAuthorizeRedirect(t *testing.T) {
	tc, adminToken := setupAdminTest(t)
	defer tc.Close()

	// Clean up any existing test providers
	cleanupOAuthProviders(t, tc)

	// Create a mock OAuth provider
	createProvider(t, tc, adminToken, "github_test2", "test-client-id", "test-client-secret")

	// Test authorize endpoint - should redirect to OAuth provider
	resp := tc.NewRequest("GET", "/api/v1/auth/oauth/github_test2/authorize").
		Send()

	// Should redirect (302 or 307)
	status := resp.Status()
	require.True(t, status == fiber.StatusFound || status == fiber.StatusTemporaryRedirect,
		"Should redirect to OAuth provider, got status: %d", status)

	// Check Location header exists
	location := resp.Header("Location")
	require.NotEmpty(t, location, "Should have Location header")
	require.Contains(t, location, "client_id=test-client-id", "Should contain client_id")
	require.Contains(t, location, "state=", "Should contain state parameter")

	t.Logf("OAuth authorization redirects to: %s", location)
}

// TestOAuthCallbackSuccess tests successful OAuth callback
func TestOAuthCallbackSuccess(t *testing.T) {
	tc, adminToken := setupAdminTest(t)
	defer tc.Close()

	// Clean up any existing test providers
	cleanupOAuthProviders(t, tc)

	// Create a mock OAuth provider with our mock server
	mockOAuthServer := setupMockOAuthServer(t)
	defer mockOAuthServer.Close()

	// Create provider with mock server URLs
	createCustomProvider(t, tc, adminToken, "mockprovider_test1", mockOAuthServer.URL+"/authorize",
		mockOAuthServer.URL+"/token", mockOAuthServer.URL+"/userinfo")

	// Get a valid state by calling authorize first
	authorizeResp := tc.NewRequest("GET", "/api/v1/auth/oauth/mockprovider_test1/authorize").
		Send()

	location := authorizeResp.Header("Location")
	require.NotEmpty(t, location, "Should have redirect location")

	// Extract state from redirect URL
	state := extractStateFromURL(location)
	require.NotEmpty(t, state, "Should have state parameter")

	// Call callback with valid code and state
	resp := tc.NewRequest("GET", fmt.Sprintf("/api/v1/auth/oauth/mockprovider_test1/callback?code=mock_code&state=%s", state)).
		Send().
		AssertStatus(fiber.StatusOK)

	var result map[string]interface{}
	resp.JSON(&result)

	// Verify response structure
	require.Contains(t, result, "access_token", "Should have access_token")
	require.Contains(t, result, "refresh_token", "Should have refresh_token")
	require.Contains(t, result, "user", "Should have user")
	require.Contains(t, result, "is_new_user", "Should have is_new_user")

	user := result["user"].(map[string]interface{})
	require.Equal(t, "test@example.com", user["email"], "Should have correct email")
	require.True(t, user["email_verified"].(bool), "OAuth users should have verified email")

	t.Logf("OAuth callback successful, created user: %s", user["id"])
}

// TestOAuthCallbackInvalidState tests OAuth callback with invalid state
func TestOAuthCallbackInvalidState(t *testing.T) {
	tc, adminToken := setupAdminTest(t)
	defer tc.Close()

	// Clean up any existing test providers
	cleanupOAuthProviders(t, tc)

	// Create a mock OAuth provider
	createProvider(t, tc, adminToken, "github_test4", "test-client-id", "test-client-secret")

	// Call callback with invalid state
	resp := tc.NewRequest("GET", "/api/v1/auth/oauth/github_test4/callback?code=mock_code&state=invalid_state").
		Send().
		AssertStatus(fiber.StatusBadRequest)

	var result map[string]interface{}
	resp.JSON(&result)

	require.Contains(t, result, "error", "Should have error message")
	require.Contains(t, result["error"].(string), "state", "Error should mention state")

	t.Logf("OAuth callback correctly rejected invalid state")
}

// TestOAuthCallbackProviderError tests OAuth callback when provider returns error
func TestOAuthCallbackProviderError(t *testing.T) {
	tc, adminToken := setupAdminTest(t)
	defer tc.Close()

	// Clean up any existing test providers
	cleanupOAuthProviders(t, tc)

	// Create a mock OAuth provider
	createProvider(t, tc, adminToken, "github_test5", "test-client-id", "test-client-secret")

	// Call callback with OAuth error
	resp := tc.NewRequest("GET", "/api/v1/auth/oauth/github_test5/callback?error=access_denied&error_description=User+denied+access").
		Send().
		AssertStatus(fiber.StatusBadRequest)

	var result map[string]interface{}
	resp.JSON(&result)

	require.Contains(t, result, "error", "Should have error message")
	require.Contains(t, result, "description", "Should have error description")

	t.Logf("OAuth callback correctly handled provider error")
}

// TestOAuthUserLinking tests linking OAuth to existing user
func TestOAuthUserLinking(t *testing.T) {
	tc, adminToken := setupAdminTest(t)
	defer tc.Close()

	// Clean up any existing test providers
	cleanupOAuthProviders(t, tc)

	// Create a mock OAuth provider with our mock server
	mockOAuthServer := setupMockOAuthServer(t)
	defer mockOAuthServer.Close()

	// Create provider with mock server URLs
	createCustomProvider(t, tc, adminToken, "mockprovider_test2", mockOAuthServer.URL+"/authorize",
		mockOAuthServer.URL+"/token", mockOAuthServer.URL+"/userinfo")

	// First, create a regular user with the same email
	email := "test@example.com"
	password := "TestPass123!"

	signupReq := map[string]interface{}{
		"email":    email,
		"password": password,
	}

	tc.NewRequest("POST", "/api/v1/auth/signup").
		WithBody(signupReq).
		Send().
		AssertStatus(fiber.StatusCreated)

	t.Logf("Created regular user: %s", email)

	// Now authenticate via OAuth with the same email
	// Get a valid state
	authorizeResp := tc.NewRequest("GET", "/api/v1/auth/oauth/mockprovider_test2/authorize").
		Send()

	location := authorizeResp.Header("Location")
	state := extractStateFromURL(location)

	// Call callback
	resp := tc.NewRequest("GET", fmt.Sprintf("/api/v1/auth/oauth/mockprovider_test2/callback?code=mock_code&state=%s", state)).
		Send().
		AssertStatus(fiber.StatusOK)

	var result map[string]interface{}
	resp.JSON(&result)

	// Should link to existing user (is_new_user should be false)
	isNewUser, ok := result["is_new_user"].(bool)
	require.True(t, ok, "Should have is_new_user field")
	require.False(t, isNewUser, "Should link to existing user, not create new one")

	user := result["user"].(map[string]interface{})
	require.Equal(t, email, user["email"], "Should have same email")

	t.Logf("OAuth correctly linked to existing user")
}

// TestOAuthUnsupportedProvider tests OAuth with unsupported provider
func TestOAuthUnsupportedProvider(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()

	tc.EnsureAuthSchema()

	// Try to authorize with a provider that doesn't exist
	resp := tc.NewRequest("GET", "/api/v1/auth/oauth/nonexistent/authorize").
		Send().
		AssertStatus(fiber.StatusBadRequest)

	var result map[string]interface{}
	resp.JSON(&result)

	require.Contains(t, result, "error", "Should have error message")
	require.Contains(t, result["error"].(string), "not configured", "Error should mention provider not configured")

	t.Logf("OAuth correctly rejected unsupported provider")
}

// TestOAuthTokenStorage tests that OAuth tokens are stored
func TestOAuthTokenStorage(t *testing.T) {
	tc, adminToken := setupAdminTest(t)
	defer tc.Close()

	// Clean up any existing test providers
	cleanupOAuthProviders(t, tc)

	// Create a mock OAuth provider
	mockOAuthServer := setupMockOAuthServer(t)
	defer mockOAuthServer.Close()

	createCustomProvider(t, tc, adminToken, "mockprovider_test3", mockOAuthServer.URL+"/authorize",
		mockOAuthServer.URL+"/token", mockOAuthServer.URL+"/userinfo")

	// Complete OAuth flow
	authorizeResp := tc.NewRequest("GET", "/api/v1/auth/oauth/mockprovider_test3/authorize").Send()
	location := authorizeResp.Header("Location")
	state := extractStateFromURL(location)

	resp := tc.NewRequest("GET", fmt.Sprintf("/api/v1/auth/oauth/mockprovider_test3/callback?code=mock_code&state=%s", state)).
		Send().
		AssertStatus(fiber.StatusOK)

	var result map[string]interface{}
	resp.JSON(&result)

	user := result["user"].(map[string]interface{})
	userID := user["id"].(string)

	// Check that OAuth link was created
	var linkCount int
	err := tc.DB.Pool().QueryRow(context.Background(),
		"SELECT COUNT(*) FROM auth.oauth_links WHERE user_id = $1 AND provider = 'mockprovider_test3'",
		userID).Scan(&linkCount)
	require.NoError(t, err, "Should query oauth_links")
	require.Equal(t, 1, linkCount, "Should have one OAuth link")

	// Check that OAuth token was stored
	var tokenCount int
	err = tc.DB.Pool().QueryRow(context.Background(),
		"SELECT COUNT(*) FROM auth.oauth_tokens WHERE user_id = $1 AND provider = 'mockprovider_test3'",
		userID).Scan(&tokenCount)
	require.NoError(t, err, "Should query oauth_tokens")
	require.Equal(t, 1, tokenCount, "Should have one OAuth token")

	t.Logf("OAuth tokens correctly stored for user: %s", userID)
}

// Helper functions

func setupMockOAuthServer(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()

	// Mock authorization endpoint (not actually used in tests, but needed for completeness)
	mux.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		// In real flow, this would show login page and redirect back
		state := r.URL.Query().Get("state")
		redirectURI := r.URL.Query().Get("redirect_uri")
		http.Redirect(w, r, fmt.Sprintf("%s?code=mock_code&state=%s", redirectURI, state), http.StatusFound)
	})

	// Mock token endpoint
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "mock_access_token",
			"refresh_token": "mock_refresh_token",
			"token_type":    "Bearer",
			"expires_in":    3600,
		})
	})

	// Mock userinfo endpoint
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "mock_user_123",
			"email": "test@example.com",
			"name":  "Test User",
		})
	})

	server := httptest.NewServer(mux)
	t.Logf("Mock OAuth server started at: %s", server.URL)
	return server
}

func createProvider(t *testing.T, tc *test.TestContext, adminToken, provider, clientID, clientSecret string) {
	providerData := map[string]interface{}{
		"provider_name": provider,
		"display_name":  provider,
		"enabled":       true,
		"client_id":     clientID,
		"client_secret": clientSecret,
		"redirect_url":  fmt.Sprintf("http://localhost:8080/api/v1/auth/oauth/%s/callback", provider),
		"scopes":        []string{"openid", "email", "profile"},
		"is_custom":     false,
	}

	tc.NewRequest("POST", "/api/v1/admin/oauth/providers").
		WithAuth(adminToken).
		WithBody(providerData).
		Send().
		AssertStatus(fiber.StatusCreated)

	t.Logf("Created OAuth provider: %s", provider)
}

func createCustomProvider(t *testing.T, tc *test.TestContext, adminToken, providerName, authURL, tokenURL, userInfoURL string) {
	providerData := map[string]interface{}{
		"provider_name":     providerName,
		"display_name":      providerName,
		"enabled":           true,
		"client_id":         "test-client-id",
		"client_secret":     "test-client-secret",
		"redirect_url":      fmt.Sprintf("http://localhost:8080/api/v1/auth/oauth/%s/callback", providerName),
		"scopes":            []string{"openid", "email", "profile"},
		"is_custom":         true,
		"authorization_url": authURL,
		"token_url":         tokenURL,
		"user_info_url":     userInfoURL,
	}

	tc.NewRequest("POST", "/api/v1/admin/oauth/providers").
		WithAuth(adminToken).
		WithBody(providerData).
		Send().
		AssertStatus(fiber.StatusCreated)

	t.Logf("Created custom OAuth provider: %s", providerName)
}

func extractStateFromURL(url string) string {
	// Extract state parameter from URL
	// Format: https://provider.com/auth?client_id=xxx&state=yyy&...
	parts := strings.Split(url, "state=")
	if len(parts) < 2 {
		return ""
	}
	state := strings.Split(parts[1], "&")[0]
	return state
}

func cleanupOAuthProviders(t *testing.T, tc *test.TestContext) {
	// Delete OAuth tokens first (has foreign key to users)
	_, err := tc.DB.Pool().Exec(context.Background(), "DELETE FROM auth.oauth_tokens")
	if err != nil {
		t.Logf("Warning: Failed to cleanup OAuth tokens: %v", err)
	}

	// Delete OAuth links (has foreign key to users)
	_, err = tc.DB.Pool().Exec(context.Background(), "DELETE FROM auth.oauth_links")
	if err != nil {
		t.Logf("Warning: Failed to cleanup OAuth links: %v", err)
	}

	// Delete test users (those created via OAuth with test@example.com)
	_, err = tc.DB.Pool().Exec(context.Background(), "DELETE FROM auth.users WHERE email = 'test@example.com'")
	if err != nil {
		t.Logf("Warning: Failed to cleanup test users: %v", err)
	}

	// Delete all OAuth providers
	_, err = tc.DB.Pool().Exec(context.Background(), "DELETE FROM dashboard.oauth_providers")
	if err != nil {
		t.Logf("Warning: Failed to cleanup OAuth providers: %v", err)
	}
}
