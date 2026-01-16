package api

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// =============================================================================
// OAuthHandler Struct Tests
// =============================================================================

func TestNewOAuthHandler(t *testing.T) {
	t.Run("creates handler with valid encryption key", func(t *testing.T) {
		validKey := "12345678901234567890123456789012" // 32 bytes
		handler := NewOAuthHandler(nil, nil, nil, "https://example.com", validKey, nil)

		assert.NotNil(t, handler)
		assert.Equal(t, "https://example.com", handler.baseURL)
		assert.Equal(t, validKey, handler.encryptionKey)
		assert.NotNil(t, handler.stateStore)
		assert.NotNil(t, handler.logoutService)
	})

	t.Run("creates handler with empty encryption key", func(t *testing.T) {
		// Should warn but still create handler
		handler := NewOAuthHandler(nil, nil, nil, "https://example.com", "", nil)

		assert.NotNil(t, handler)
		assert.Empty(t, handler.encryptionKey)
	})

	t.Run("clears invalid encryption key", func(t *testing.T) {
		// Key must be exactly 32 bytes for AES-256
		invalidKey := "short-key"
		handler := NewOAuthHandler(nil, nil, nil, "https://example.com", invalidKey, nil)

		assert.NotNil(t, handler)
		assert.Empty(t, handler.encryptionKey, "invalid key should be cleared")
	})

	t.Run("creates handler with nil dependencies", func(t *testing.T) {
		handler := NewOAuthHandler(nil, nil, nil, "", "", nil)
		assert.NotNil(t, handler)
	})
}

// =============================================================================
// extractEmail Tests
// =============================================================================

func TestExtractEmail(t *testing.T) {
	handler := NewOAuthHandler(nil, nil, nil, "", "", nil)

	tests := []struct {
		name         string
		providerName string
		userInfo     map[string]interface{}
		expected     string
	}{
		{
			name:         "email field present",
			providerName: "google",
			userInfo: map[string]interface{}{
				"email": "user@example.com",
			},
			expected: "user@example.com",
		},
		{
			name:         "email field empty",
			providerName: "google",
			userInfo: map[string]interface{}{
				"email": "",
			},
			expected: "",
		},
		{
			name:         "email field missing",
			providerName: "google",
			userInfo:     map[string]interface{}{},
			expected:     "",
		},
		{
			name:         "GitHub with no email uses login",
			providerName: "github",
			userInfo: map[string]interface{}{
				"login": "octocat",
			},
			expected: "octocat@users.noreply.github.com",
		},
		{
			name:         "GitHub with email present uses email",
			providerName: "github",
			userInfo: map[string]interface{}{
				"email": "octocat@github.com",
				"login": "octocat",
			},
			expected: "octocat@github.com",
		},
		{
			name:         "GitHub with no email and no login",
			providerName: "github",
			userInfo:     map[string]interface{}{},
			expected:     "",
		},
		{
			name:         "nil userInfo",
			providerName: "google",
			userInfo:     nil,
			expected:     "",
		},
		{
			name:         "email as non-string type",
			providerName: "google",
			userInfo: map[string]interface{}{
				"email": 12345,
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.extractEmail(tt.providerName, tt.userInfo)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// extractProviderUserID Tests
// =============================================================================

func TestExtractProviderUserID(t *testing.T) {
	handler := NewOAuthHandler(nil, nil, nil, "", "", nil)

	tests := []struct {
		name         string
		providerName string
		userInfo     map[string]interface{}
		expected     string
	}{
		{
			name:         "id as string",
			providerName: "google",
			userInfo: map[string]interface{}{
				"id": "12345",
			},
			expected: "12345",
		},
		{
			name:         "id as float64 (GitHub/Facebook)",
			providerName: "github",
			userInfo: map[string]interface{}{
				"id": float64(12345678),
			},
			expected: "12345678",
		},
		{
			name:         "sub field (OIDC)",
			providerName: "google",
			userInfo: map[string]interface{}{
				"sub": "openid-subject-123",
			},
			expected: "openid-subject-123",
		},
		{
			name:         "id takes precedence over sub",
			providerName: "google",
			userInfo: map[string]interface{}{
				"id":  "id-value",
				"sub": "sub-value",
			},
			expected: "id-value",
		},
		{
			name:         "no id or sub",
			providerName: "google",
			userInfo:     map[string]interface{}{},
			expected:     "",
		},
		{
			name:         "nil userInfo",
			providerName: "google",
			userInfo:     nil,
			expected:     "",
		},
		{
			name:         "id as other type",
			providerName: "google",
			userInfo: map[string]interface{}{
				"id": []string{"invalid"},
			},
			expected: "",
		},
		{
			name:         "large numeric id",
			providerName: "github",
			userInfo: map[string]interface{}{
				// Note: float64 can't accurately represent int64 max (9223372036854775807)
				// It rounds to 9223372036854775808 due to precision limits
				"id": float64(9223372036854775807),
			},
			expected: "9223372036854775808", // Rounded due to float64 precision
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.extractProviderUserID(tt.providerName, tt.userInfo)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// getStandardEndpoint Tests
// =============================================================================

func TestGetStandardEndpoint(t *testing.T) {
	handler := NewOAuthHandler(nil, nil, nil, "", "", nil)

	t.Run("Google provider", func(t *testing.T) {
		endpoint := handler.getStandardEndpoint("google")
		assert.Equal(t, "https://accounts.google.com/o/oauth2/auth", endpoint.AuthURL)
		assert.Equal(t, "https://oauth2.googleapis.com/token", endpoint.TokenURL)
	})

	t.Run("GitHub provider", func(t *testing.T) {
		endpoint := handler.getStandardEndpoint("github")
		assert.Equal(t, "https://github.com/login/oauth/authorize", endpoint.AuthURL)
		assert.Equal(t, "https://github.com/login/oauth/access_token", endpoint.TokenURL)
	})

	t.Run("Microsoft provider", func(t *testing.T) {
		endpoint := handler.getStandardEndpoint("microsoft")
		assert.Equal(t, "https://login.microsoftonline.com/common/oauth2/v2.0/authorize", endpoint.AuthURL)
		assert.Equal(t, "https://login.microsoftonline.com/common/oauth2/v2.0/token", endpoint.TokenURL)
	})

	t.Run("GitLab provider", func(t *testing.T) {
		endpoint := handler.getStandardEndpoint("gitlab")
		assert.Contains(t, endpoint.AuthURL, "gitlab")
	})

	t.Run("unknown provider returns empty endpoint", func(t *testing.T) {
		endpoint := handler.getStandardEndpoint("unknown")
		// Unknown providers should return empty endpoint
		assert.Empty(t, endpoint.AuthURL)
		assert.Empty(t, endpoint.TokenURL)
	})
}

// =============================================================================
// Handler Endpoint Tests - Authorize
// =============================================================================

// NOTE: TestOAuthHandler_Authorize_NoDatabase was removed because it requires
// a database connection. The handler's Authorize method calls getProviderConfig()
// which queries the database before returning any response. Tests that exercise
// the full handler flow should use integration tests with a test database.

// =============================================================================
// Handler Endpoint Tests - Callback
// =============================================================================

func TestOAuthHandler_Callback_Validation(t *testing.T) {
	app := fiber.New()
	handler := NewOAuthHandler(nil, nil, nil, "https://example.com", "", nil)

	app.Get("/api/v1/auth/oauth/:provider/callback", handler.Callback)

	t.Run("OAuth error from provider", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/auth/oauth/google/callback?error=access_denied&error_description=User+denied+access", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, 400, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "OAuth authentication failed")
		assert.Contains(t, string(body), "User denied access")
	})

	t.Run("invalid state", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/auth/oauth/google/callback?code=abc123&state=invalid-state", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, 400, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "Invalid OAuth state parameter")
	})

	t.Run("missing state", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/auth/oauth/google/callback?code=abc123", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, 400, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "Invalid OAuth state")
	})
}

// =============================================================================
// Handler Endpoint Tests - ListEnabledProviders
// =============================================================================

// NOTE: TestOAuthHandler_ListEnabledProviders_NoDatabase was removed because
// it requires a database connection. The handler panics with nil db when
// calling ListEnabledProviders. Use integration tests with a test database.

// =============================================================================
// Handler Endpoint Tests - Logout
// =============================================================================

func TestOAuthHandler_Logout_Validation(t *testing.T) {
	app := fiber.New()
	handler := NewOAuthHandler(nil, nil, nil, "https://example.com", "", nil)

	app.Post("/api/v1/auth/oauth/:provider/logout", handler.Logout)

	t.Run("missing authentication", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/auth/oauth/google/logout", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, 401, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "Authentication required")
	})

	t.Run("empty user_id", func(t *testing.T) {
		app := fiber.New()
		app.Post("/api/v1/auth/oauth/:provider/logout", func(c *fiber.Ctx) error {
			c.Locals("user_id", "")
			return handler.Logout(c)
		})

		req := httptest.NewRequest("POST", "/api/v1/auth/oauth/google/logout", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, 401, resp.StatusCode)
	})
}

// =============================================================================
// Handler Endpoint Tests - LogoutCallback
// =============================================================================

func TestOAuthHandler_LogoutCallback_Validation(t *testing.T) {
	app := fiber.New()
	handler := NewOAuthHandler(nil, nil, nil, "https://example.com", "", nil)

	app.Get("/api/v1/auth/oauth/:provider/logout/callback", handler.LogoutCallback)

	t.Run("missing state parameter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/auth/oauth/google/logout/callback", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, 400, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "Missing state parameter")
	})

	// NOTE: "invalid state" test removed because it calls ValidateLogoutState
	// which requires a database connection. Use integration tests for that path.
}

// =============================================================================
// GetAndValidateState Tests
// =============================================================================

func TestOAuthHandler_GetAndValidateState(t *testing.T) {
	handler := NewOAuthHandler(nil, nil, nil, "https://example.com", "", nil)

	t.Run("valid state returns metadata", func(t *testing.T) {
		// First, set a state
		state, err := auth.GenerateState()
		require.NoError(t, err)

		handler.stateStore.Set(state, "/callback")

		// Now validate it
		metadata, valid := handler.GetAndValidateState(state)
		assert.True(t, valid)
		assert.NotNil(t, metadata)
		assert.Equal(t, "/callback", metadata.RedirectURI)
	})

	t.Run("invalid state returns false", func(t *testing.T) {
		metadata, valid := handler.GetAndValidateState("non-existent-state")
		assert.False(t, valid)
		assert.Nil(t, metadata)
	})

	t.Run("state can only be used once", func(t *testing.T) {
		state, err := auth.GenerateState()
		require.NoError(t, err)

		handler.stateStore.Set(state, "/callback")

		// First validation succeeds
		_, valid := handler.GetAndValidateState(state)
		assert.True(t, valid)

		// Second validation fails (consumed)
		_, valid = handler.GetAndValidateState(state)
		assert.False(t, valid)
	})
}

// =============================================================================
// OAuth2 Config Tests
// =============================================================================

func TestOAuth2ConfigConstruction(t *testing.T) {
	t.Run("config with all fields", func(t *testing.T) {
		config := &oauth2.Config{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "https://example.com/callback",
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://auth.example.com/authorize",
				TokenURL: "https://auth.example.com/token",
			},
		}

		assert.Equal(t, "test-client-id", config.ClientID)
		assert.Equal(t, "test-client-secret", config.ClientSecret)
		assert.Equal(t, "https://example.com/callback", config.RedirectURL)
		assert.Len(t, config.Scopes, 3)
	})

	t.Run("AuthCodeURL generation", func(t *testing.T) {
		config := &oauth2.Config{
			ClientID:    "test-client-id",
			RedirectURL: "https://example.com/callback",
			Scopes:      []string{"openid", "email"},
			Endpoint: oauth2.Endpoint{
				AuthURL: "https://auth.example.com/authorize",
			},
		}

		state := "test-state-123"
		authURL := config.AuthCodeURL(state)

		assert.Contains(t, authURL, "https://auth.example.com/authorize")
		assert.Contains(t, authURL, "client_id=test-client-id")
		assert.Contains(t, authURL, "state=test-state-123")
		assert.Contains(t, authURL, "redirect_uri=")
	})
}

// =============================================================================
// State Store Integration Tests
// =============================================================================

func TestOAuthHandler_StateStoreIntegration(t *testing.T) {
	handler := NewOAuthHandler(nil, nil, nil, "https://example.com", "", nil)

	t.Run("multiple states can be stored", func(t *testing.T) {
		states := make([]string, 5)
		for i := 0; i < 5; i++ {
			state, err := auth.GenerateState()
			require.NoError(t, err)
			states[i] = state
			handler.stateStore.Set(state, "/callback"+string(rune('0'+i)))
		}

		// All states should be valid
		for i, state := range states {
			metadata, valid := handler.GetAndValidateState(state)
			assert.True(t, valid, "state %d should be valid", i)
			assert.NotNil(t, metadata)
		}
	})

	t.Run("state with empty redirect URI", func(t *testing.T) {
		state, err := auth.GenerateState()
		require.NoError(t, err)

		handler.stateStore.Set(state, "")

		metadata, valid := handler.GetAndValidateState(state)
		assert.True(t, valid)
		assert.NotNil(t, metadata)
		assert.Empty(t, metadata.RedirectURI)
	})
}

// =============================================================================
// Request Body Parsing Tests
// =============================================================================

// NOTE: TestOAuthHandler_Logout_BodyParsing was removed because the Logout
// handler calls database methods (getTokenByUserAndProvider) before any body
// parsing occurs. Tests for body parsing should use integration tests with
// a test database.

// =============================================================================
// Error Description Extraction Tests
// =============================================================================

func TestOAuthHandler_ErrorDescriptionExtraction(t *testing.T) {
	app := fiber.New()
	handler := NewOAuthHandler(nil, nil, nil, "https://example.com", "", nil)

	app.Get("/api/v1/auth/oauth/:provider/callback", handler.Callback)

	tests := []struct {
		name            string
		queryParams     string
		expectedMessage string
	}{
		{
			name:            "with error_description",
			queryParams:     "error=access_denied&error_description=User+denied+access",
			expectedMessage: "User denied access",
		},
		{
			name:            "without error_description",
			queryParams:     "error=access_denied",
			expectedMessage: "access_denied",
		},
		{
			name:            "server error",
			queryParams:     "error=server_error&error_description=Internal+server+error",
			expectedMessage: "Internal server error",
		},
		{
			name:            "invalid_request",
			queryParams:     "error=invalid_request&error_description=Missing+required+parameter",
			expectedMessage: "Missing required parameter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/auth/oauth/google/callback?"+tt.queryParams, nil)

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, 400, resp.StatusCode)

			body, _ := io.ReadAll(resp.Body)
			assert.Contains(t, string(body), tt.expectedMessage)
		})
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkExtractEmail(b *testing.B) {
	handler := NewOAuthHandler(nil, nil, nil, "", "", nil)
	userInfo := map[string]interface{}{
		"email": "user@example.com",
		"name":  "Test User",
		"id":    "12345",
	}

	for i := 0; i < b.N; i++ {
		_ = handler.extractEmail("google", userInfo)
	}
}

func BenchmarkExtractProviderUserID(b *testing.B) {
	handler := NewOAuthHandler(nil, nil, nil, "", "", nil)
	userInfo := map[string]interface{}{
		"email": "user@example.com",
		"id":    float64(12345678),
	}

	for i := 0; i < b.N; i++ {
		_ = handler.extractProviderUserID("github", userInfo)
	}
}

func BenchmarkGetStandardEndpoint(b *testing.B) {
	handler := NewOAuthHandler(nil, nil, nil, "", "", nil)

	for i := 0; i < b.N; i++ {
		_ = handler.getStandardEndpoint("google")
	}
}

func BenchmarkGenerateAndValidateState(b *testing.B) {
	handler := NewOAuthHandler(nil, nil, nil, "", "", nil)

	for i := 0; i < b.N; i++ {
		state, _ := auth.GenerateState()
		handler.stateStore.Set(state, "/callback")
		_, _ = handler.GetAndValidateState(state)
	}
}
