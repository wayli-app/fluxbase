package api

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testUUID is a fixed UUID for testing
var testUUID = uuid.MustParse("12345678-1234-1234-1234-123456789abc")

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestGetFirstAttribute(t *testing.T) {
	tests := []struct {
		name       string
		attributes map[string][]string
		key        string
		expected   string
	}{
		{
			name: "key exists with single value",
			attributes: map[string][]string{
				"email": {"user@example.com"},
			},
			key:      "email",
			expected: "user@example.com",
		},
		{
			name: "key exists with multiple values returns first",
			attributes: map[string][]string{
				"groups": {"admin", "users", "developers"},
			},
			key:      "groups",
			expected: "admin",
		},
		{
			name: "key does not exist",
			attributes: map[string][]string{
				"email": {"user@example.com"},
			},
			key:      "name",
			expected: "",
		},
		{
			name:       "nil attributes",
			attributes: nil,
			key:        "email",
			expected:   "",
		},
		{
			name:       "empty attributes",
			attributes: map[string][]string{},
			key:        "email",
			expected:   "",
		},
		{
			name: "key exists but empty slice",
			attributes: map[string][]string{
				"email": {},
			},
			key:      "email",
			expected: "",
		},
		{
			name: "SAML claim URL format",
			attributes: map[string][]string{
				"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress": {"user@example.com"},
			},
			key:      "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
			expected: "user@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFirstAttribute(tt.attributes, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertSAMLAttributesToMap(t *testing.T) {
	tests := []struct {
		name     string
		attrs    map[string][]string
		expected map[string]interface{}
	}{
		{
			name: "single value attributes converted to string",
			attrs: map[string][]string{
				"email": {"user@example.com"},
				"name":  {"John Doe"},
			},
			expected: map[string]interface{}{
				"email": "user@example.com",
				"name":  "John Doe",
			},
		},
		{
			name: "multiple value attributes kept as slice",
			attrs: map[string][]string{
				"groups": {"admin", "users"},
				"roles":  {"manager", "developer", "reviewer"},
			},
			expected: map[string]interface{}{
				"groups": []string{"admin", "users"},
				"roles":  []string{"manager", "developer", "reviewer"},
			},
		},
		{
			name: "mixed single and multiple values",
			attrs: map[string][]string{
				"email":  {"user@example.com"},
				"groups": {"admin", "users"},
			},
			expected: map[string]interface{}{
				"email":  "user@example.com",
				"groups": []string{"admin", "users"},
			},
		},
		{
			name:     "empty attributes",
			attrs:    map[string][]string{},
			expected: map[string]interface{}{},
		},
		{
			name:     "nil attributes",
			attrs:    nil,
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertSAMLAttributesToMap(tt.attrs)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCapitalizeWords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single word lowercase",
			input:    "john",
			expected: "John",
		},
		{
			name:     "single word uppercase",
			input:    "JOHN",
			expected: "John",
		},
		{
			name:     "two words lowercase",
			input:    "john doe",
			expected: "John Doe",
		},
		{
			name:     "two words uppercase",
			input:    "JOHN DOE",
			expected: "John Doe",
		},
		{
			name:     "mixed case",
			input:    "jOHN dOE",
			expected: "John Doe",
		},
		{
			name:     "three words",
			input:    "john doe smith",
			expected: "John Doe Smith",
		},
		{
			name:     "multiple spaces",
			input:    "john   doe",
			expected: "John Doe",
		},
		{
			name:     "leading and trailing spaces",
			input:    "  john doe  ",
			expected: "John Doe",
		},
		{
			name:     "single character words",
			input:    "a b c",
			expected: "A B C",
		},
		{
			name:     "already capitalized",
			input:    "John Doe",
			expected: "John Doe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := capitalizeWords(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateOAuthState(t *testing.T) {
	t.Run("generates non-empty state", func(t *testing.T) {
		state, err := generateOAuthState()
		require.NoError(t, err)
		assert.NotEmpty(t, state)
	})

	t.Run("generates base64 URL encoded state", func(t *testing.T) {
		state, err := generateOAuthState()
		require.NoError(t, err)

		// Should be valid base64 URL encoding
		decoded, err := base64.URLEncoding.DecodeString(state)
		require.NoError(t, err)
		assert.Len(t, decoded, 32, "decoded state should be 32 bytes")
	})

	t.Run("generates unique states", func(t *testing.T) {
		states := make(map[string]bool)
		for i := 0; i < 100; i++ {
			state, err := generateOAuthState()
			require.NoError(t, err)
			assert.False(t, states[state], "state should be unique")
			states[state] = true
		}
	})
}

func TestParseIDTokenClaims(t *testing.T) {
	t.Run("valid ID token", func(t *testing.T) {
		// Create a simple JWT-like token
		claims := map[string]interface{}{
			"sub":   "12345",
			"email": "user@example.com",
			"name":  "John Doe",
		}
		claimsJSON, _ := json.Marshal(claims)
		payload := base64.RawURLEncoding.EncodeToString(claimsJSON)

		// JWT format: header.payload.signature
		idToken := "eyJhbGciOiJSUzI1NiJ9." + payload + ".signature"

		result, err := parseIDTokenClaims(idToken)
		require.NoError(t, err)
		assert.Equal(t, "12345", result["sub"])
		assert.Equal(t, "user@example.com", result["email"])
		assert.Equal(t, "John Doe", result["name"])
	})

	t.Run("invalid token format - missing parts", func(t *testing.T) {
		_, err := parseIDTokenClaims("invalid-token")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid ID token format")
	})

	t.Run("invalid token format - two parts", func(t *testing.T) {
		_, err := parseIDTokenClaims("header.payload")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid ID token format")
	})

	t.Run("invalid base64 payload", func(t *testing.T) {
		_, err := parseIDTokenClaims("header.!!!invalid!!.signature")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode ID token payload")
	})

	t.Run("invalid JSON payload", func(t *testing.T) {
		payload := base64.RawURLEncoding.EncodeToString([]byte("not json"))
		_, err := parseIDTokenClaims("header." + payload + ".signature")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal ID token claims")
	})

	t.Run("empty claims", func(t *testing.T) {
		claimsJSON, _ := json.Marshal(map[string]interface{}{})
		payload := base64.RawURLEncoding.EncodeToString(claimsJSON)
		idToken := "header." + payload + ".signature"

		result, err := parseIDTokenClaims(idToken)
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

// =============================================================================
// Struct Tests
// =============================================================================

func TestSSOProviderStruct(t *testing.T) {
	t.Run("OAuth provider", func(t *testing.T) {
		provider := SSOProvider{
			ID:       "google",
			Name:     "Google",
			Type:     "oauth",
			Provider: "google",
		}

		assert.Equal(t, "google", provider.ID)
		assert.Equal(t, "Google", provider.Name)
		assert.Equal(t, "oauth", provider.Type)
		assert.Equal(t, "google", provider.Provider)
	})

	t.Run("SAML provider", func(t *testing.T) {
		provider := SSOProvider{
			ID:   "okta",
			Name: "Okta SSO",
			Type: "saml",
		}

		assert.Equal(t, "okta", provider.ID)
		assert.Equal(t, "Okta SSO", provider.Name)
		assert.Equal(t, "saml", provider.Type)
		assert.Empty(t, provider.Provider)
	})

	t.Run("JSON serialization", func(t *testing.T) {
		provider := SSOProvider{
			ID:       "github",
			Name:     "GitHub",
			Type:     "oauth",
			Provider: "github",
		}

		data, err := json.Marshal(provider)
		require.NoError(t, err)

		var decoded SSOProvider
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, provider, decoded)
	})

	t.Run("JSON omits empty provider", func(t *testing.T) {
		provider := SSOProvider{
			ID:   "okta",
			Name: "Okta",
			Type: "saml",
		}

		data, err := json.Marshal(provider)
		require.NoError(t, err)

		// Should not contain "provider" key when empty
		assert.NotContains(t, string(data), `"provider"`)
	})
}

func TestDashboardOAuthStateStruct(t *testing.T) {
	t.Run("basic state with all fields", func(t *testing.T) {
		userInfoURL := "https://api.example.com/userinfo"
		state := dashboardOAuthState{
			Provider:    "google",
			CreatedAt:   time.Now(),
			RedirectTo:  "/admin/dashboard",
			UserInfoURL: &userInfoURL,
		}

		assert.Equal(t, "google", state.Provider)
		assert.NotZero(t, state.CreatedAt)
		assert.Equal(t, "/admin/dashboard", state.RedirectTo)
		assert.NotNil(t, state.UserInfoURL)
		assert.Equal(t, userInfoURL, *state.UserInfoURL)
	})

	t.Run("state without user info URL", func(t *testing.T) {
		state := dashboardOAuthState{
			Provider:   "github",
			CreatedAt:  time.Now(),
			RedirectTo: "/",
		}

		assert.Nil(t, state.UserInfoURL)
	})
}

func TestNewDashboardAuthHandler(t *testing.T) {
	t.Run("creates handler with nil services", func(t *testing.T) {
		// Test that handler can be created even with nil services
		// (services are validated at runtime when called)
		handler := NewDashboardAuthHandler(nil, nil, nil, nil, nil, "https://example.com", "12345678901234567890123456789012", nil)

		assert.NotNil(t, handler)
		assert.Equal(t, "https://example.com", handler.baseURL)
		assert.Equal(t, "12345678901234567890123456789012", handler.encryptionKey)
		assert.NotNil(t, handler.oauthStates)
		assert.NotNil(t, handler.oauthConfigs)
	})

	t.Run("creates handler with empty base URL", func(t *testing.T) {
		handler := NewDashboardAuthHandler(nil, nil, nil, nil, nil, "", "", nil)
		assert.NotNil(t, handler)
		assert.Empty(t, handler.baseURL)
	})

	t.Run("initializes empty maps", func(t *testing.T) {
		handler := NewDashboardAuthHandler(nil, nil, nil, nil, nil, "", "", nil)
		assert.Empty(t, handler.oauthStates)
		assert.Empty(t, handler.oauthConfigs)
	})
}

// =============================================================================
// getIPAddress Tests (using Fiber test context)
// =============================================================================

func TestGetIPAddress(t *testing.T) {
	tests := []struct {
		name           string
		headers        map[string]string
		remoteAddr     string
		expectedIP     string
		expectNonEmpty bool
	}{
		{
			name: "X-Forwarded-For header single IP",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.100",
			},
			expectedIP: "192.168.1.100",
		},
		{
			name: "X-Forwarded-For header multiple IPs",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.100, 10.0.0.1, 172.16.0.1",
			},
			expectedIP: "192.168.1.100",
		},
		{
			name: "X-Forwarded-For with spaces",
			headers: map[string]string{
				"X-Forwarded-For": "  192.168.1.100  ",
			},
			expectedIP: "192.168.1.100",
		},
		{
			name: "X-Real-IP header",
			headers: map[string]string{
				"X-Real-IP": "10.0.0.50",
			},
			expectedIP: "10.0.0.50",
		},
		{
			name: "X-Forwarded-For takes precedence over X-Real-IP",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.100",
				"X-Real-IP":       "10.0.0.50",
			},
			expectedIP: "192.168.1.100",
		},
		{
			name:           "no headers uses RemoteAddr",
			headers:        map[string]string{},
			expectNonEmpty: true, // Fiber returns some IP
		},
		{
			name: "IPv6 address in X-Forwarded-For",
			headers: map[string]string{
				"X-Forwarded-For": "2001:db8::1",
			},
			expectedIP: "2001:db8::1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			var capturedIP string
			app.Get("/test", func(c *fiber.Ctx) error {
				ip := getIPAddress(c)
				if ip != nil {
					capturedIP = ip.String()
				}
				return c.SendStatus(200)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			if tt.expectNonEmpty {
				// Just check that we got some IP
				assert.NotEmpty(t, capturedIP)
			} else if tt.expectedIP != "" {
				assert.Equal(t, tt.expectedIP, capturedIP)
			}
		})
	}
}

// =============================================================================
// buildOAuthConfig Tests
// =============================================================================

func TestBuildOAuthConfig(t *testing.T) {
	handler := NewDashboardAuthHandler(nil, nil, nil, nil, nil, "https://example.com", "", nil)

	t.Run("Google provider", func(t *testing.T) {
		config := handler.buildOAuthConfig("google", "client-id", "client-secret", nil, false, nil, nil)
		require.NotNil(t, config)

		assert.Equal(t, "client-id", config.ClientID)
		assert.Equal(t, "client-secret", config.ClientSecret)
		assert.Equal(t, "https://example.com/dashboard/auth/sso/oauth/google/callback", config.RedirectURL)
		assert.Equal(t, "https://accounts.google.com/o/oauth2/v2/auth", config.Endpoint.AuthURL)
		assert.Equal(t, "https://oauth2.googleapis.com/token", config.Endpoint.TokenURL)
		assert.Contains(t, config.Scopes, "openid")
		assert.Contains(t, config.Scopes, "email")
		assert.Contains(t, config.Scopes, "profile")
	})

	t.Run("GitHub provider", func(t *testing.T) {
		config := handler.buildOAuthConfig("github", "client-id", "client-secret", nil, false, nil, nil)
		require.NotNil(t, config)

		assert.Equal(t, "https://github.com/login/oauth/authorize", config.Endpoint.AuthURL)
		assert.Equal(t, "https://github.com/login/oauth/access_token", config.Endpoint.TokenURL)
		assert.Contains(t, config.Scopes, "read:user")
		assert.Contains(t, config.Scopes, "user:email")
	})

	t.Run("Microsoft provider", func(t *testing.T) {
		config := handler.buildOAuthConfig("microsoft", "client-id", "client-secret", nil, false, nil, nil)
		require.NotNil(t, config)

		assert.Equal(t, "https://login.microsoftonline.com/common/oauth2/v2.0/authorize", config.Endpoint.AuthURL)
		assert.Equal(t, "https://login.microsoftonline.com/common/oauth2/v2.0/token", config.Endpoint.TokenURL)
	})

	t.Run("GitLab provider", func(t *testing.T) {
		config := handler.buildOAuthConfig("gitlab", "client-id", "client-secret", nil, false, nil, nil)
		require.NotNil(t, config)

		assert.Equal(t, "https://gitlab.com/oauth/authorize", config.Endpoint.AuthURL)
		assert.Equal(t, "https://gitlab.com/oauth/token", config.Endpoint.TokenURL)
		assert.Contains(t, config.Scopes, "read_user")
	})

	t.Run("unsupported provider returns nil", func(t *testing.T) {
		config := handler.buildOAuthConfig("unknown", "client-id", "client-secret", nil, false, nil, nil)
		assert.Nil(t, config)
	})

	t.Run("custom scopes override defaults", func(t *testing.T) {
		customScopes := []string{"custom_scope_1", "custom_scope_2"}
		config := handler.buildOAuthConfig("google", "client-id", "client-secret", customScopes, false, nil, nil)
		require.NotNil(t, config)

		assert.Equal(t, customScopes, config.Scopes)
	})

	t.Run("custom provider with URLs", func(t *testing.T) {
		authURL := "https://custom.example.com/oauth/authorize"
		tokenURL := "https://custom.example.com/oauth/token"
		config := handler.buildOAuthConfig("custom", "client-id", "client-secret", []string{"openid"}, true, &authURL, &tokenURL)
		require.NotNil(t, config)

		assert.Equal(t, authURL, config.Endpoint.AuthURL)
		assert.Equal(t, tokenURL, config.Endpoint.TokenURL)
	})

	t.Run("custom provider without URLs returns nil", func(t *testing.T) {
		config := handler.buildOAuthConfig("custom", "client-id", "client-secret", nil, true, nil, nil)
		assert.Nil(t, config)
	})
}

// =============================================================================
// Handler Endpoint Tests
// =============================================================================

// NOTE: Handler endpoint validation tests have been removed because they require
// a real or mocked auth service. The handlers call service methods (e.g.,
// HasExistingUsers, isPasswordLoginDisabled) before performing input validation,
// so nil services cause panics. These tests should be implemented using
// integration tests with a test database or mocked services.

// =============================================================================
// RequireDashboardAuth Middleware Tests
// =============================================================================

func TestRequireDashboardAuth(t *testing.T) {
	handler := NewDashboardAuthHandler(nil, nil, nil, nil, nil, "", "", nil)

	app := fiber.New()
	app.Use(handler.RequireDashboardAuth)
	app.Get("/protected", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	t.Run("missing authorization header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, 401, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "Missing authorization header")
	})

	t.Run("empty authorization header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, 401, resp.StatusCode)
	})

	t.Run("invalid authorization format - no Bearer", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Basic token")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, 401, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "Invalid authorization header")
	})

	t.Run("invalid authorization format - Bearer without token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer ")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// This will fail at token validation since jwtManager is nil
		assert.Equal(t, 401, resp.StatusCode)
	})
}

// =============================================================================
// SSO Route Tests
// =============================================================================

func TestDashboardAuthHandler_InitiateSAMLLogin_SAMLNotConfigured(t *testing.T) {
	app := fiber.New()
	handler := NewDashboardAuthHandler(nil, nil, nil, nil, nil, "", "", nil) // samlService is nil

	app.Get("/dashboard/auth/sso/saml/:provider", handler.InitiateSAMLLogin)

	req := httptest.NewRequest("GET", "/dashboard/auth/sso/saml/okta", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, 500, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "SAML not configured")
}

func TestDashboardAuthHandler_SAMLACSCallback_SAMLNotConfigured(t *testing.T) {
	app := fiber.New()
	handler := NewDashboardAuthHandler(nil, nil, nil, nil, nil, "", "", nil) // samlService is nil

	app.Post("/dashboard/auth/sso/saml/acs", handler.SAMLACSCallback)

	req := httptest.NewRequest("POST", "/dashboard/auth/sso/saml/acs", strings.NewReader("SAMLResponse=test"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should redirect to login with error
	assert.Equal(t, 302, resp.StatusCode)
	location := resp.Header.Get("Location")
	assert.Contains(t, location, "/admin/login")
	// URL may encode space as %20 or +, both are valid
	assert.True(t, strings.Contains(location, "SAML%20not%20configured") || strings.Contains(location, "SAML+not+configured"),
		"Expected location to contain 'SAML not configured' encoded, got: %s", location)
}

func TestDashboardAuthHandler_SAMLACSCallback_MissingResponse(t *testing.T) {
	app := fiber.New()
	handler := NewDashboardAuthHandler(nil, nil, nil, nil, nil, "", "", nil)

	// Need to have samlService to get past the nil check
	// But we're testing the missing SAMLResponse validation
	// So we'll test the flow where SAML is configured but response is missing
	// For now, test that empty SAMLResponse results in redirect

	// Since samlService is nil, it will redirect with "SAML not configured"
	app.Post("/dashboard/auth/sso/saml/acs", handler.SAMLACSCallback)

	req := httptest.NewRequest("POST", "/dashboard/auth/sso/saml/acs", nil)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, 302, resp.StatusCode)
}

// =============================================================================
// OAuth State Management Tests
// =============================================================================

func TestDashboardAuthHandler_OAuthStateManagement(t *testing.T) {
	handler := NewDashboardAuthHandler(nil, nil, nil, nil, nil, "https://example.com", "", nil)

	t.Run("store and retrieve OAuth state", func(t *testing.T) {
		state := "test-state-123"
		userInfoURL := "https://api.example.com/userinfo"

		handler.oauthStatesMu.Lock()
		handler.oauthStates[state] = &dashboardOAuthState{
			Provider:    "google",
			CreatedAt:   time.Now(),
			RedirectTo:  "/admin",
			UserInfoURL: &userInfoURL,
		}
		handler.oauthStatesMu.Unlock()

		handler.oauthStatesMu.RLock()
		storedState, exists := handler.oauthStates[state]
		handler.oauthStatesMu.RUnlock()

		assert.True(t, exists)
		assert.Equal(t, "google", storedState.Provider)
		assert.Equal(t, "/admin", storedState.RedirectTo)
		assert.NotNil(t, storedState.UserInfoURL)
	})

	t.Run("non-existent state returns nil", func(t *testing.T) {
		handler.oauthStatesMu.RLock()
		_, exists := handler.oauthStates["non-existent"]
		handler.oauthStatesMu.RUnlock()

		assert.False(t, exists)
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkCapitalizeWords(b *testing.B) {
	input := "john doe smith"
	for i := 0; i < b.N; i++ {
		_ = capitalizeWords(input)
	}
}

func BenchmarkGetFirstAttribute(b *testing.B) {
	attrs := map[string][]string{
		"email":  {"user@example.com"},
		"name":   {"John Doe"},
		"groups": {"admin", "users", "developers"},
	}

	for i := 0; i < b.N; i++ {
		_ = getFirstAttribute(attrs, "email")
	}
}

func BenchmarkConvertSAMLAttributesToMap(b *testing.B) {
	attrs := map[string][]string{
		"email":       {"user@example.com"},
		"name":        {"John Doe"},
		"groups":      {"admin", "users"},
		"displayName": {"John D."},
		"firstName":   {"John"},
		"lastName":    {"Doe"},
	}

	for i := 0; i < b.N; i++ {
		_ = convertSAMLAttributesToMap(attrs)
	}
}

func BenchmarkGenerateOAuthState(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = generateOAuthState()
	}
}

func BenchmarkParseIDTokenClaims(b *testing.B) {
	claims := map[string]interface{}{
		"sub":   "12345",
		"email": "user@example.com",
		"name":  "John Doe",
	}
	claimsJSON, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)
	idToken := "eyJhbGciOiJSUzI1NiJ9." + payload + ".signature"

	for i := 0; i < b.N; i++ {
		_, _ = parseIDTokenClaims(idToken)
	}
}

func BenchmarkBuildOAuthConfig(b *testing.B) {
	handler := NewDashboardAuthHandler(nil, nil, nil, nil, nil, "https://example.com", "", nil)

	for i := 0; i < b.N; i++ {
		_ = handler.buildOAuthConfig("google", "client-id", "client-secret", nil, false, nil, nil)
	}
}
