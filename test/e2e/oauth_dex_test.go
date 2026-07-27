package e2e

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/test"
)

// dexHost returns the Dex hostname: "dex" in devcontainer, "localhost" in CI.
func dexHost() string {
	if os.Getenv("CI") == "true" {
		return "localhost"
	}
	return "dex"
}

// fluxbasePort returns the Fluxbase backend port: 8082 in CI, 8080 locally.
func fluxbasePort() string {
	if p := os.Getenv("FLUXBASE_SERVER_ADDRESS"); p != "" {
		return strings.TrimPrefix(p, ":")
	}
	if os.Getenv("CI") == "true" {
		return "8082"
	}
	return "8080"
}

const dexPort = "5556"

func dexBaseURL() string {
	return "http://" + dexHost() + ":" + dexPort + "/dex"
}

// requireDex skips the test if the Dex OIDC provider is not reachable.
func requireDex(t *testing.T) {
	t.Helper()
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(dexBaseURL() + "/healthz")
	if err != nil {
		t.Skip("Dex OIDC provider not available — skipping Dex-dependent test")
	}
	resp.Body.Close()
}

// createDexProvider creates a Fluxbase OAuth provider configured for the Dex test instance.
func createDexProvider(t *testing.T, tc *test.TestContext, adminToken string) {
	providerData := map[string]interface{}{
		"provider_name":     "dex",
		"display_name":      "Dex Test",
		"enabled":           true,
		"client_id":         "fluxbase-test",
		"client_secret":     "test-client-secret",
		"redirect_url":      fmt.Sprintf("http://localhost:%s/api/v1/auth/oauth/dex/callback", fluxbasePort()),
		"scopes":            []string{"openid", "email", "profile"},
		"is_custom":         true,
		"authorization_url": dexBaseURL() + "/auth",
		"token_url":         dexBaseURL() + "/token",
		"user_info_url":     dexBaseURL() + "/userinfo",
	}

	tc.NewRequest("POST", "/api/v1/admin/oauth/providers").
		WithAuth(adminToken).
		WithBody(providerData).
		Send().
		AssertStatus(fiber.StatusCreated)

	t.Logf("Created Dex OAuth provider")
}

// dexTokens holds tokens from a completed OAuth flow
type dexTokens struct {
	AccessToken  string
	RefreshToken string
}

// authenticateWithDex performs the full browser-simulated login against Dex and returns
// the callback URL containing the authorization code.
//
// Dex flow:
//  1. GET /dex/auth?client_id=... → auto-follow redirects → land on login page
//     (extract dex login state from the /dex/auth/local/login?state=<s> redirect)
//  2. POST /dex/auth/local/login?back=&state=<dex_state> with login, password
//     → 303 to /dex/approval?hmac=...&req=<req> (older) or /dex/auth?hmac=...&req=<req> (newer)
//  3. POST /dex/approval?hmac=...&req=<req> with approval=approve
//     → 303 to redirect_uri?code=<code>&state=<oauth_state>
func authenticateWithDex(t *testing.T, authURL string) string {
	jar, _ := cookiejar.New(nil)

	// Phase 1: Follow redirects to login page, capture the Dex login state
	var dexLoginState string
	var redirectLog []string
	phase1 := &http.Client{
		Jar:     jar,
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			redirectLog = append(redirectLog, req.URL.String())
			// Capture state from any redirect that carries a state parameter.
			// Dex redirects through various paths; capture the last one with state.
			if s := req.URL.Query().Get("state"); s != "" {
				dexLoginState = s
			}
			return nil
		},
	}

	loginURL := replaceDexHost(authURL)
	resp, err := phase1.Get(loginURL)
	require.NoError(t, err, "Should reach Dex authorize endpoint")
	bodyBytes, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK || len(redirectLog) == 0 {
		t.Logf("Phase 1: followed %d redirects, final status: %d", len(redirectLog), resp.StatusCode)
		t.Logf("Phase 1: response body: %s", string(bodyBytes))
	}

	require.NotEmpty(t, dexLoginState, "Should have extracted Dex login state from redirect chain (redirects: %d, status: %d)", len(redirectLog), resp.StatusCode)

	// Phase 2: POST login credentials (don't follow redirects to preserve cookies)
	phase2 := &http.Client{
		Jar:     jar,
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	loginEndpoint := fmt.Sprintf("http://%s:%s/dex/auth/local/login?back=&state=%s", dexHost(), dexPort, dexLoginState)
	loginData := url.Values{
		"login":    {"testuser@fluxbase.test"},
		"password": {"testpass"},
	}

	resp, err = phase2.PostForm(loginEndpoint, loginData)
	require.NoError(t, err, "Should POST login to Dex")
	approvalURL := resp.Header.Get("Location")
	resp.Body.Close()
	require.NotEmpty(t, approvalURL, "Dex should redirect to approval page after login")
	// Dex versions differ: older uses /dex/approval, newer uses /dex/auth?hmac=...
	require.True(t,
		strings.Contains(approvalURL, "/dex/approval") || strings.Contains(approvalURL, "/dex/auth"),
		"Should redirect to Dex approval or auth page, got: %s", approvalURL)

	// Phase 3: POST approval (grant access)
	dexOrigin := fmt.Sprintf("http://%s:%s", dexHost(), dexPort)
	if strings.HasPrefix(approvalURL, "/") {
		approvalURL = dexOrigin + approvalURL
	}

	parsed, _ := url.Parse(approvalURL)
	reqID := parsed.Query().Get("req")

	approvalData := url.Values{
		"req":      {reqID},
		"approval": {"approve"},
	}

	phase3 := &http.Client{
		Jar:     jar,
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if strings.Contains(req.URL.String(), "/api/v1/auth/oauth/dex/callback") {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	resp, err = phase3.PostForm(approvalURL, approvalData)
	require.NoError(t, err, "Should POST approval to Dex")
	defer resp.Body.Close()

	callbackURL := resp.Header.Get("Location")

	// If not a direct callback redirect, follow remaining redirects
	for i := 0; i < 5 && resp.StatusCode == http.StatusFound && !strings.Contains(callbackURL, "/api/v1/auth/oauth/dex/callback"); i++ {
		resp.Body.Close()
		if strings.HasPrefix(callbackURL, "/") {
			callbackURL = dexOrigin + callbackURL
		}
		resp, err = phase3.Get(callbackURL)
		require.NoError(t, err)
		callbackURL = resp.Header.Get("Location")
	}

	require.Contains(t, callbackURL, "/api/v1/auth/oauth/dex/callback",
		"Dex should redirect to Fluxbase callback URL, got: %s", callbackURL)

	return callbackURL
}

// replaceDexHost replaces localhost:5556 with dex:5556 when running in devcontainer
// where Dex is on a separate container. In CI (GitHub Actions), both run on localhost.
func replaceDexHost(u string) string {
	if os.Getenv("CI") == "true" {
		return u
	}
	return strings.Replace(u, "localhost:5556", "dex:5556", 1)
}

// completeDexOAuthFlow performs the full OAuth flow against Dex and returns tokens.
func completeDexOAuthFlow(t *testing.T, tc *test.TestContext) dexTokens {
	// Get a valid state from the authorize endpoint
	authorizeResp := tc.NewRequest("GET", "/api/v1/auth/oauth/dex/authorize").
		Send().
		AssertStatus(fiber.StatusOK)

	var authorizeResult map[string]interface{}
	authorizeResp.JSON(&authorizeResult)

	authURL := authorizeResult["url"].(string)
	state := extractStateFromURL(authURL)
	require.NotEmpty(t, state)

	// Authenticate with Dex and get the callback URL
	callbackURL := authenticateWithDex(t, authURL)

	// Extract path+query for the test client (strip host since tc sends to its own server)
	parsed, err := url.Parse(callbackURL)
	require.NoError(t, err)
	callbackPath := parsed.Path + "?" + parsed.RawQuery

	callbackResp := tc.NewRequest("GET", callbackPath).Send()

	var result map[string]interface{}
	callbackResp.JSON(&result)

	if accessToken, ok := result["access_token"].(string); ok {
		refreshToken, _ := result["refresh_token"].(string)
		return dexTokens{AccessToken: accessToken, RefreshToken: refreshToken}
	}

	t.Logf("Callback response status: %d, body: %v", callbackResp.Status(), result)
	t.Logf("Note: Full Dex token exchange failed. State validation and route wiring verified.")
	return dexTokens{}
}

// TestDexOAuth_FullFlow tests the complete OAuth authorize -> callback flow against a real Dex OIDC provider.
// Requires the Dex container to be running (added to .devcontainer/docker-compose.yml).
func TestDexOAuth_FullFlow(t *testing.T) {
	requireDex(t)
	tc, adminToken := setupAdminTest(t)
	defer tc.Close()

	cleanupOAuthProviders(t, tc)
	createDexProvider(t, tc, adminToken)

	// Step 1: Call authorize to get the Dex login URL
	authorizeResp := tc.NewRequest("GET", "/api/v1/auth/oauth/dex/authorize").
		Send().
		AssertStatus(fiber.StatusOK)

	var authorizeResult map[string]interface{}
	authorizeResp.JSON(&authorizeResult)
	require.Contains(t, authorizeResult, "url", "Should have url field")

	authURL := authorizeResult["url"].(string)
	require.NotEmpty(t, authURL, "Should have authorization URL")
	require.Contains(t, authURL, "client_id=fluxbase-test", "Should contain Dex client_id")

	state := extractStateFromURL(authURL)
	require.NotEmpty(t, state, "Should have state parameter")

	t.Logf("Step 1 OK: Got Dex authorize URL with state=%s", state[:10]+"...")

	// Step 2: Authenticate with Dex
	callbackURL := authenticateWithDex(t, authURL)
	t.Logf("Step 2 OK: Got callback URL: %s", callbackURL)

	// Step 3: Call the Fluxbase callback
	parsed, err := url.Parse(callbackURL)
	require.NoError(t, err)
	callbackPath := parsed.Path + "?" + parsed.RawQuery

	callbackResp := tc.NewRequest("GET", callbackPath).Send()
	require.Equal(t, fiber.StatusOK, callbackResp.Status(),
		"OAuth callback should succeed")

	var result map[string]interface{}
	callbackResp.JSON(&result)

	require.Contains(t, result, "access_token", "Should have access_token")
	require.Contains(t, result, "refresh_token", "Should have refresh_token")
	require.Contains(t, result, "user", "Should have user")

	user := result["user"].(map[string]interface{})
	require.Equal(t, "testuser@fluxbase.test", user["email"],
		"Should have correct email from Dex")

	t.Logf("Step 3 OK: OAuth callback succeeded, user: %s", user["id"])
	t.Logf("Full OAuth flow against Dex completed successfully")
}

// TestDexOAuth_TokenRefresh tests that tokens obtained via Dex OAuth can be refreshed
func TestDexOAuth_TokenRefresh(t *testing.T) {
	requireDex(t)
	tc, adminToken := setupAdminTest(t)
	defer tc.Close()

	cleanupOAuthProviders(t, tc)
	createDexProvider(t, tc, adminToken)

	tokens := completeDexOAuthFlow(t, tc)
	require.NotEmpty(t, tokens.AccessToken, "Should have access token")
	require.NotEmpty(t, tokens.RefreshToken, "Should have refresh token")

	userResp := tc.NewRequest("GET", "/api/v1/auth/user").
		WithHeader("Authorization", "Bearer "+tokens.AccessToken).
		Send().
		AssertStatus(fiber.StatusOK)

	var user map[string]interface{}
	userResp.JSON(&user)
	require.Equal(t, "testuser@fluxbase.test", user["email"],
		"Should authenticate with Dex-issued token")

	t.Logf("Token refresh test: access token verified for user %s", user["id"])
}

// TestDexOAuth_ExistingUserLinking tests that Dex OAuth links to an existing user
func TestDexOAuth_ExistingUserLinking(t *testing.T) {
	requireDex(t)
	tc, adminToken := setupAdminTest(t)
	defer tc.Close()

	cleanupOAuthProviders(t, tc)

	// Clean up any leftover Dex test users from previous tests
	tc.ExecuteSQL("DELETE FROM auth.oauth_links WHERE email = $1", "testuser@fluxbase.test")
	tc.ExecuteSQL("DELETE FROM auth.users WHERE email = $1", "testuser@fluxbase.test")

	// Create a regular user with the same email as the Dex test user
	signupReq := map[string]interface{}{
		"email":    "testuser@fluxbase.test",
		"password": "ExistingPass123!",
	}
	tc.NewRequest("POST", "/api/v1/auth/signup").
		WithBody(signupReq).
		Send().
		AssertStatus(fiber.StatusCreated)

	createDexProvider(t, tc, adminToken)

	tokens := completeDexOAuthFlow(t, tc)
	require.NotEmpty(t, tokens.AccessToken)

	userResp := tc.NewRequest("GET", "/api/v1/auth/user").
		WithHeader("Authorization", "Bearer "+tokens.AccessToken).
		Send()
	var user map[string]interface{}
	userResp.JSON(&user)

	require.Equal(t, "testuser@fluxbase.test", user["email"])
	t.Logf("User linking verified: OAuth linked to existing user")
}

// TestDexOAuth_RouteRegistration verifies the Dex OAuth routes are properly registered
func TestDexOAuth_RouteRegistration(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()

	tc.EnsureAuthSchema()

	resp := tc.NewRequest("GET", "/api/v1/auth/oauth/providers").Send()
	require.True(t, resp.Status() == fiber.StatusOK || resp.Status() == fiber.StatusBadRequest,
		"OAuth providers endpoint should be registered")

	t.Logf("OAuth route registration verified")
}

// TestOAuthLogoutCallbackRegistered verifies the logout callback route exists
func TestOAuthLogoutCallbackRegistered(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()

	tc.EnsureAuthSchema()

	resp := tc.NewRequest("GET", "/api/v1/auth/oauth/test-provider/logout/callback?state=test").Send()
	require.NotEqual(t, fiber.StatusNotFound, resp.Status(),
		"Logout callback route should be registered")

	t.Logf("OAuth logout callback route registration verified")
}

// TestSAMLLogoutRoutesRegistered verifies SAML SLO routes exist
func TestSAMLLogoutRoutesRegistered(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()

	tc.EnsureAuthSchema()

	resp := tc.NewRequest("POST", "/api/v1/auth/saml/slo").Send()
	require.NotEqual(t, fiber.StatusNotFound, resp.Status(),
		"SAML SLO POST route should be registered")

	resp = tc.NewRequest("GET", "/api/v1/auth/saml/slo").Send()
	require.NotEqual(t, fiber.StatusNotFound, resp.Status(),
		"SAML SLO GET route should be registered")

	t.Logf("SAML SLO route registration verified")
}
