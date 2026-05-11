package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// OAuthProviderHandler Construction Tests
// =============================================================================

func TestNewOAuthProviderHandler(t *testing.T) {
	t.Run("creates handler with nil dependencies", func(t *testing.T) {
		handler := NewOAuthProviderHandler(nil, nil, []byte(""), "", nil)
		assert.NotNil(t, handler)
		assert.Nil(t, handler.db)
		assert.Nil(t, handler.settingsCache)
		assert.Empty(t, handler.encryptionKey)
		assert.Empty(t, handler.baseURL)
		assert.Nil(t, handler.configProviders)
	})

	t.Run("creates handler with values", func(t *testing.T) {
		handler := NewOAuthProviderHandler(nil, nil, []byte("encryption-key"), "https://example.com", nil)
		assert.NotNil(t, handler)
		assert.Equal(t, []byte("encryption-key"), handler.encryptionKey)
		assert.Equal(t, "https://example.com", handler.baseURL)
	})
}

// =============================================================================
// OAuthProvider Struct Tests
// =============================================================================

func TestOAuthProvider_Struct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		id := uuid.New()
		now := time.Now()
		authURL := "https://provider.example.com/authorize"
		tokenURL := "https://provider.example.com/token"
		userInfoURL := "https://provider.example.com/userinfo"
		revokeURL := "https://provider.example.com/revoke"
		logoutURL := "https://provider.example.com/logout"

		provider := OAuthProvider{
			ID:                  id,
			ProviderName:        "custom_oauth",
			DisplayName:         "Custom OAuth Provider",
			Enabled:             true,
			ClientID:            "client-id-123",
			ClientSecret:        "",
			HasSecret:           true,
			RedirectURL:         "https://app.example.com/callback",
			Scopes:              []string{"openid", "profile", "email"},
			IsCustom:            true,
			AuthorizationURL:    &authURL,
			TokenURL:            &tokenURL,
			UserInfoURL:         &userInfoURL,
			RevocationEndpoint:  &revokeURL,
			EndSessionEndpoint:  &logoutURL,
			AllowDashboardLogin: true,
			AllowAppLogin:       true,
			RequiredClaims:      map[string][]string{"groups": {"admin"}},
			DeniedClaims:        map[string][]string{"status": {"banned"}},
			Source:              "database",
			CreatedAt:           now,
			UpdatedAt:           now,
		}

		assert.Equal(t, id, provider.ID)
		assert.Equal(t, "custom_oauth", provider.ProviderName)
		assert.Equal(t, "Custom OAuth Provider", provider.DisplayName)
		assert.True(t, provider.Enabled)
		assert.Equal(t, "client-id-123", provider.ClientID)
		assert.Empty(t, provider.ClientSecret) // Should be omitted in responses
		assert.True(t, provider.HasSecret)
		assert.Equal(t, "https://app.example.com/callback", provider.RedirectURL)
		assert.Len(t, provider.Scopes, 3)
		assert.True(t, provider.IsCustom)
		assert.Equal(t, &authURL, provider.AuthorizationURL)
		assert.Equal(t, &tokenURL, provider.TokenURL)
		assert.Equal(t, &userInfoURL, provider.UserInfoURL)
		assert.Equal(t, &revokeURL, provider.RevocationEndpoint)
		assert.Equal(t, &logoutURL, provider.EndSessionEndpoint)
		assert.True(t, provider.AllowDashboardLogin)
		assert.True(t, provider.AllowAppLogin)
		assert.Contains(t, provider.RequiredClaims["groups"], "admin")
		assert.Contains(t, provider.DeniedClaims["status"], "banned")
		assert.Equal(t, "database", provider.Source)
	})

	t.Run("JSON serialization", func(t *testing.T) {
		provider := OAuthProvider{
			ID:           uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			ProviderName: "google",
			DisplayName:  "Google",
			Enabled:      true,
			ClientID:     "google-client-id",
			HasSecret:    true,
			Scopes:       []string{"openid", "email"},
			Source:       "database",
		}

		data, err := json.Marshal(provider)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"id":"550e8400-e29b-41d4-a716-446655440000"`)
		assert.Contains(t, string(data), `"provider_name":"google"`)
		assert.Contains(t, string(data), `"display_name":"Google"`)
		assert.Contains(t, string(data), `"enabled":true`)
		assert.Contains(t, string(data), `"has_secret":true`)
	})

	t.Run("JSON deserialization", func(t *testing.T) {
		jsonData := `{
			"id": "550e8400-e29b-41d4-a716-446655440000",
			"provider_name": "github",
			"display_name": "GitHub",
			"enabled": true,
			"client_id": "github-client-id",
			"has_secret": true,
			"redirect_url": "https://app.example.com/auth/github/callback",
			"scopes": ["read:user", "user:email"],
			"is_custom": false,
			"allow_dashboard_login": true,
			"allow_app_login": true,
			"source": "database"
		}`

		var provider OAuthProvider
		err := json.Unmarshal([]byte(jsonData), &provider)
		require.NoError(t, err)

		assert.Equal(t, "github", provider.ProviderName)
		assert.Equal(t, "GitHub", provider.DisplayName)
		assert.True(t, provider.Enabled)
		assert.True(t, provider.HasSecret)
		assert.False(t, provider.IsCustom)
	})
}

// =============================================================================
// CreateOAuthProviderRequest Struct Tests
// =============================================================================

func TestCreateOAuthProviderRequest_Struct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		authURL := "https://custom.example.com/authorize"
		tokenURL := "https://custom.example.com/token"
		userInfoURL := "https://custom.example.com/userinfo"
		revokeURL := "https://custom.example.com/revoke"
		logoutURL := "https://custom.example.com/logout"
		allowDash := true
		allowApp := true

		req := CreateOAuthProviderRequest{
			ProviderName:        "custom_provider",
			DisplayName:         "Custom Provider",
			Enabled:             true,
			ClientID:            "client-123",
			ClientSecret:        "secret-456",
			RedirectURL:         "https://app.example.com/callback",
			Scopes:              []string{"openid", "profile"},
			IsCustom:            true,
			AuthorizationURL:    &authURL,
			TokenURL:            &tokenURL,
			UserInfoURL:         &userInfoURL,
			RevocationEndpoint:  &revokeURL,
			EndSessionEndpoint:  &logoutURL,
			AllowDashboardLogin: &allowDash,
			AllowAppLogin:       &allowApp,
			RequiredClaims:      map[string][]string{"org": {"acme"}},
			DeniedClaims:        map[string][]string{"banned": {"true"}},
		}

		assert.Equal(t, "custom_provider", req.ProviderName)
		assert.Equal(t, "Custom Provider", req.DisplayName)
		assert.True(t, req.Enabled)
		assert.Equal(t, "client-123", req.ClientID)
		assert.Equal(t, "secret-456", req.ClientSecret)
		assert.True(t, req.IsCustom)
		assert.NotNil(t, req.AuthorizationURL)
	})

	t.Run("JSON deserialization for standard provider", func(t *testing.T) {
		jsonData := `{
			"provider_name": "google",
			"display_name": "Google",
			"enabled": true,
			"client_id": "google-client-id.apps.googleusercontent.com",
			"client_secret": "GOCSPX-secret",
			"redirect_url": "https://app.example.com/auth/google/callback",
			"scopes": ["openid", "email", "profile"],
			"is_custom": false
		}`

		var req CreateOAuthProviderRequest
		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Equal(t, "google", req.ProviderName)
		assert.Equal(t, "Google", req.DisplayName)
		assert.False(t, req.IsCustom)
		assert.Nil(t, req.AuthorizationURL)
	})

	t.Run("JSON deserialization for custom provider", func(t *testing.T) {
		jsonData := `{
			"provider_name": "custom_oidc",
			"display_name": "Custom OIDC",
			"enabled": true,
			"client_id": "client-id",
			"client_secret": "client-secret",
			"redirect_url": "https://app.example.com/callback",
			"scopes": ["openid"],
			"is_custom": true,
			"authorization_url": "https://auth.example.com/authorize",
			"token_url": "https://auth.example.com/token",
			"user_info_url": "https://auth.example.com/userinfo"
		}`

		var req CreateOAuthProviderRequest
		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Equal(t, "custom_oidc", req.ProviderName)
		assert.True(t, req.IsCustom)
		assert.NotNil(t, req.AuthorizationURL)
		assert.NotNil(t, req.TokenURL)
		assert.NotNil(t, req.UserInfoURL)
	})
}

// =============================================================================
// UpdateOAuthProviderRequest Struct Tests
// =============================================================================

func TestUpdateOAuthProviderRequest_Struct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		displayName := "Updated Name"
		enabled := false
		clientID := "new-client-id"
		clientSecret := "new-secret"
		redirectURL := "https://new.example.com/callback"
		authURL := "https://new.example.com/authorize"
		tokenURL := "https://new.example.com/token"
		userInfoURL := "https://new.example.com/userinfo"
		revokeURL := "https://new.example.com/revoke"
		logoutURL := "https://new.example.com/logout"
		allowDash := true
		allowApp := false

		req := UpdateOAuthProviderRequest{
			DisplayName:         &displayName,
			Enabled:             &enabled,
			ClientID:            &clientID,
			ClientSecret:        &clientSecret,
			RedirectURL:         &redirectURL,
			Scopes:              []string{"new_scope"},
			AuthorizationURL:    &authURL,
			TokenURL:            &tokenURL,
			UserInfoURL:         &userInfoURL,
			RevocationEndpoint:  &revokeURL,
			EndSessionEndpoint:  &logoutURL,
			AllowDashboardLogin: &allowDash,
			AllowAppLogin:       &allowApp,
			RequiredClaims:      map[string][]string{"role": {"user"}},
			DeniedClaims:        map[string][]string{"blocked": {"yes"}},
		}

		assert.Equal(t, "Updated Name", *req.DisplayName)
		assert.False(t, *req.Enabled)
		assert.Equal(t, "new-client-id", *req.ClientID)
	})

	t.Run("JSON deserialization partial update", func(t *testing.T) {
		jsonData := `{
			"display_name": "New Display Name",
			"enabled": true
		}`

		var req UpdateOAuthProviderRequest
		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.NotNil(t, req.DisplayName)
		assert.Equal(t, "New Display Name", *req.DisplayName)
		assert.NotNil(t, req.Enabled)
		assert.True(t, *req.Enabled)
		assert.Nil(t, req.ClientID)
		assert.Nil(t, req.ClientSecret)
	})

	t.Run("empty update request", func(t *testing.T) {
		jsonData := `{}`

		var req UpdateOAuthProviderRequest
		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Nil(t, req.DisplayName)
		assert.Nil(t, req.Enabled)
	})
}

// =============================================================================
// AuthSettings Struct Tests
// =============================================================================

func TestAuthSettings_Struct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		settings := AuthSettings{
			SignupEnabled:                 true,
			RequireEmailVerification:      true,
			MagicLinkEnabled:              true,
			PasswordMinLength:             12,
			PasswordRequireUppercase:      true,
			PasswordRequireLowercase:      true,
			PasswordRequireNumber:         true,
			PasswordRequireSpecial:        true,
			SessionTimeoutMinutes:         30,
			MaxSessionsPerUser:            3,
			DisableDashboardPasswordLogin: false,
			DisableAppPasswordLogin:       false,
			Overrides:                     make(map[string]SettingOverride),
		}

		assert.True(t, settings.SignupEnabled)
		assert.True(t, settings.RequireEmailVerification)
		assert.True(t, settings.MagicLinkEnabled)
		assert.Equal(t, 12, settings.PasswordMinLength)
		assert.True(t, settings.PasswordRequireUppercase)
		assert.True(t, settings.PasswordRequireLowercase)
		assert.True(t, settings.PasswordRequireNumber)
		assert.True(t, settings.PasswordRequireSpecial)
		assert.Equal(t, 30, settings.SessionTimeoutMinutes)
		assert.Equal(t, 3, settings.MaxSessionsPerUser)
		assert.False(t, settings.DisableDashboardPasswordLogin)
		assert.False(t, settings.DisableAppPasswordLogin)
	})

	t.Run("JSON serialization", func(t *testing.T) {
		settings := AuthSettings{
			SignupEnabled:         true,
			MagicLinkEnabled:      true,
			PasswordMinLength:     8,
			SessionTimeoutMinutes: 60,
			MaxSessionsPerUser:    5,
			Overrides:             make(map[string]SettingOverride),
		}

		data, err := json.Marshal(settings)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"enable_signup":true`)
		assert.Contains(t, string(data), `"enable_magic_link":true`)
		assert.Contains(t, string(data), `"password_min_length":8`)
	})

	t.Run("JSON serialization with overrides", func(t *testing.T) {
		settings := AuthSettings{
			SignupEnabled: true,
			Overrides: map[string]SettingOverride{
				"enable_signup": {
					IsOverridden: true,
					EnvVar:       "FLUXBASE_AUTH_SIGNUP_ENABLED",
				},
			},
		}

		data, err := json.Marshal(settings)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"_overrides"`)
		assert.Contains(t, string(data), `"is_overridden":true`)
	})

	t.Run("JSON deserialization", func(t *testing.T) {
		jsonData := `{
			"enable_signup": true,
			"require_email_verification": true,
			"enable_magic_link": false,
			"password_min_length": 10,
			"password_require_uppercase": true,
			"password_require_lowercase": true,
			"password_require_number": true,
			"password_require_special": false,
			"session_timeout_minutes": 120,
			"max_sessions_per_user": 10,
			"disable_dashboard_password_login": true,
			"disable_app_password_login": false
		}`

		var settings AuthSettings
		err := json.Unmarshal([]byte(jsonData), &settings)
		require.NoError(t, err)

		assert.True(t, settings.SignupEnabled)
		assert.True(t, settings.RequireEmailVerification)
		assert.False(t, settings.MagicLinkEnabled)
		assert.Equal(t, 10, settings.PasswordMinLength)
		assert.True(t, settings.DisableDashboardPasswordLogin)
	})
}

// =============================================================================
// SettingOverride Struct Tests
// =============================================================================

func TestSettingOverride_Struct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		override := SettingOverride{
			IsOverridden: true,
			EnvVar:       "FLUXBASE_AUTH_SIGNUP_ENABLED",
		}

		assert.True(t, override.IsOverridden)
		assert.Equal(t, "FLUXBASE_AUTH_SIGNUP_ENABLED", override.EnvVar)
	})

	t.Run("JSON serialization", func(t *testing.T) {
		override := SettingOverride{
			IsOverridden: true,
			EnvVar:       "FLUXBASE_AUTH_PASSWORD_MIN_LENGTH",
		}

		data, err := json.Marshal(override)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"is_overridden":true`)
		assert.Contains(t, string(data), `"env_var":"FLUXBASE_AUTH_PASSWORD_MIN_LENGTH"`)
	})
}

// =============================================================================
// Provider Name Pattern Tests
// =============================================================================

func TestProviderNamePattern(t *testing.T) {
	validNames := []string{
		"google",
		"github",
		"azure_ad",
		"custom123",
		"provider_name_123",
		"aa", // minimum 2 chars
	}

	invalidNames := []string{
		"",             // empty
		"a",            // too short
		"1google",      // starts with number
		"Google",       // starts with uppercase
		"GOOGLE",       // all uppercase
		"google-oauth", // contains hyphen
		"google oauth", // contains space
		"google.oauth", // contains dot
		"google@oauth", // contains @
		"_google",      // starts with underscore
		"this_is_a_very_long_provider_name_that_exceeds_fifty_characters_limit", // too long
	}

	for _, name := range validNames {
		t.Run("valid: "+name, func(t *testing.T) {
			assert.True(t, providerNamePattern.MatchString(name), "Expected %q to be valid", name)
		})
	}

	for _, name := range invalidNames {
		t.Run("invalid: "+name, func(t *testing.T) {
			assert.False(t, providerNamePattern.MatchString(name), "Expected %q to be invalid", name)
		})
	}
}

// =============================================================================
// CreateOAuthProvider Handler Validation Tests
// =============================================================================

func TestCreateOAuthProvider_Validation(t *testing.T) {
	t.Run("invalid request body", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewOAuthProviderHandler(nil, nil, []byte(""), "", nil)

		app.Post("/oauth/providers", handler.CreateOAuthProvider)

		req := httptest.NewRequest(http.MethodPost, "/oauth/providers", bytes.NewReader([]byte("not json")))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(body, &result)
		assert.Contains(t, result["error"], "Invalid request body")
	})

	t.Run("invalid provider name format", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewOAuthProviderHandler(nil, nil, []byte(""), "", nil)

		app.Post("/oauth/providers", handler.CreateOAuthProvider)

		body := `{
			"provider_name": "Invalid-Name",
			"display_name": "Test",
			"client_id": "id",
			"client_secret": "secret",
			"redirect_url": "https://example.com"
		}`
		req := httptest.NewRequest(http.MethodPost, "/oauth/providers", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(respBody, &result)
		assert.Contains(t, result["error"], "Provider name must start with a letter")
	})

	t.Run("missing required fields", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewOAuthProviderHandler(nil, nil, []byte(""), "", nil)

		app.Post("/oauth/providers", handler.CreateOAuthProvider)

		body := `{
			"provider_name": "google",
			"enabled": true
		}`
		req := httptest.NewRequest(http.MethodPost, "/oauth/providers", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(respBody, &result)
		assert.Contains(t, result["error"], "Missing required fields")
	})

	t.Run("custom provider missing URLs", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewOAuthProviderHandler(nil, nil, []byte(""), "", nil)

		app.Post("/oauth/providers", handler.CreateOAuthProvider)

		body := `{
			"provider_name": "custom_provider",
			"display_name": "Custom Provider",
			"client_id": "id",
			"client_secret": "secret",
			"redirect_url": "https://example.com",
			"is_custom": true
		}`
		req := httptest.NewRequest(http.MethodPost, "/oauth/providers", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(respBody, &result)
		assert.Contains(t, result["error"], "Custom providers require")
	})
}

// =============================================================================
// GetOAuthProvider Handler Validation Tests
// =============================================================================

func TestGetOAuthProvider_Validation(t *testing.T) {
	t.Run("invalid provider ID", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewOAuthProviderHandler(nil, nil, []byte(""), "", nil)

		app.Get("/oauth/providers/:id", handler.GetOAuthProvider)

		req := httptest.NewRequest(http.MethodGet, "/oauth/providers/not-a-uuid", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(body, &result)
		assert.Equal(t, "Invalid provider ID", result["error"])
	})

	t.Run("valid UUID format accepted", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewOAuthProviderHandler(nil, nil, []byte(""), "", nil)

		app.Get("/oauth/providers/:id", handler.GetOAuthProvider)

		req := httptest.NewRequest(http.MethodGet, "/oauth/providers/550e8400-e29b-41d4-a716-446655440000", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Should not be 400 (validation passed)
		assert.NotEqual(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// =============================================================================
// UpdateOAuthProvider Handler Validation Tests
// =============================================================================

func TestUpdateOAuthProvider_Validation(t *testing.T) {
	t.Run("invalid provider ID", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewOAuthProviderHandler(nil, nil, []byte(""), "", nil)

		app.Put("/oauth/providers/:id", handler.UpdateOAuthProvider)

		body := `{"display_name": "Updated Name"}`
		req := httptest.NewRequest(http.MethodPut, "/oauth/providers/invalid-uuid", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("invalid request body", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewOAuthProviderHandler(nil, nil, []byte(""), "", nil)

		app.Put("/oauth/providers/:id", handler.UpdateOAuthProvider)

		req := httptest.NewRequest(http.MethodPut, "/oauth/providers/550e8400-e29b-41d4-a716-446655440000", bytes.NewReader([]byte("not json")))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("no fields to update", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewOAuthProviderHandler(nil, nil, []byte(""), "", nil)

		app.Put("/oauth/providers/:id", handler.UpdateOAuthProvider)

		body := `{}`
		req := httptest.NewRequest(http.MethodPut, "/oauth/providers/550e8400-e29b-41d4-a716-446655440000", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(respBody, &result)
		assert.Equal(t, "No fields to update", result["error"])
	})
}

// =============================================================================
// DeleteOAuthProvider Handler Validation Tests
// =============================================================================

func TestDeleteOAuthProvider_Validation(t *testing.T) {
	t.Run("invalid provider ID", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewOAuthProviderHandler(nil, nil, []byte(""), "", nil)

		app.Delete("/oauth/providers/:id", handler.DeleteOAuthProvider)

		req := httptest.NewRequest(http.MethodDelete, "/oauth/providers/bad-uuid", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// =============================================================================
// UpdateAuthSettings Handler Validation Tests
// =============================================================================

func TestUpdateAuthSettings_Validation(t *testing.T) {
	t.Run("invalid request body", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewOAuthProviderHandler(nil, nil, []byte(""), "", nil)

		app.Put("/auth/settings", handler.UpdateAuthSettings)

		req := httptest.NewRequest(http.MethodPut, "/auth/settings", bytes.NewReader([]byte("not json")))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(body, &result)
		assert.Contains(t, result["error"], "Invalid request body")
	})
}

// =============================================================================
// JSON Serialization Tests
// =============================================================================

func TestOAuthRequests_JSONSerialization(t *testing.T) {
	t.Run("CreateOAuthProviderRequest serializes correctly", func(t *testing.T) {
		authURL := "https://auth.example.com/authorize"

		req := CreateOAuthProviderRequest{
			ProviderName:     "test_provider",
			DisplayName:      "Test Provider",
			Enabled:          true,
			ClientID:         "client-id",
			ClientSecret:     "client-secret",
			RedirectURL:      "https://example.com/callback",
			Scopes:           []string{"openid"},
			IsCustom:         true,
			AuthorizationURL: &authURL,
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"provider_name":"test_provider"`)
		assert.Contains(t, string(data), `"display_name":"Test Provider"`)
		assert.Contains(t, string(data), `"client_id":"client-id"`)
		assert.Contains(t, string(data), `"client_secret":"client-secret"`)
		assert.Contains(t, string(data), `"is_custom":true`)
	})

	t.Run("UpdateOAuthProviderRequest serializes correctly", func(t *testing.T) {
		displayName := "Updated Provider"
		enabled := false

		req := UpdateOAuthProviderRequest{
			DisplayName: &displayName,
			Enabled:     &enabled,
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"display_name":"Updated Provider"`)
		assert.Contains(t, string(data), `"enabled":false`)
	})

	t.Run("AuthSettings serializes correctly", func(t *testing.T) {
		settings := AuthSettings{
			SignupEnabled:         true,
			PasswordMinLength:     8,
			SessionTimeoutMinutes: 60,
			MaxSessionsPerUser:    5,
		}

		data, err := json.Marshal(settings)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"enable_signup":true`)
		assert.Contains(t, string(data), `"password_min_length":8`)
	})
}

// =============================================================================
// Provider Source Tests
// =============================================================================

func TestOAuthProviderSource(t *testing.T) {
	t.Run("database source provider", func(t *testing.T) {
		provider := OAuthProvider{
			ProviderName: "db_provider",
			Source:       "database",
		}

		assert.Equal(t, "database", provider.Source)
	})

	t.Run("config source provider", func(t *testing.T) {
		provider := OAuthProvider{
			ProviderName: "config_provider",
			Source:       "config",
		}

		assert.Equal(t, "config", provider.Source)
	})
}

// =============================================================================
// Claims-Based Authorization Tests
// =============================================================================

func TestClaimsBasedAuthorization(t *testing.T) {
	t.Run("required claims", func(t *testing.T) {
		provider := OAuthProvider{
			RequiredClaims: map[string][]string{
				"groups": {"admin", "developers"},
				"org":    {"acme"},
			},
		}

		assert.Len(t, provider.RequiredClaims, 2)
		assert.Contains(t, provider.RequiredClaims["groups"], "admin")
		assert.Contains(t, provider.RequiredClaims["groups"], "developers")
	})

	t.Run("denied claims", func(t *testing.T) {
		provider := OAuthProvider{
			DeniedClaims: map[string][]string{
				"status": {"banned", "suspended"},
			},
		}

		assert.Len(t, provider.DeniedClaims, 1)
		assert.Contains(t, provider.DeniedClaims["status"], "banned")
	})

	t.Run("no claims restrictions", func(t *testing.T) {
		provider := OAuthProvider{
			RequiredClaims: nil,
			DeniedClaims:   nil,
		}

		assert.Nil(t, provider.RequiredClaims)
		assert.Nil(t, provider.DeniedClaims)
	})
}

// =============================================================================
// Login Permission Tests
// =============================================================================

func TestOAuthLoginPermissions(t *testing.T) {
	t.Run("dashboard only login", func(t *testing.T) {
		provider := OAuthProvider{
			AllowDashboardLogin: true,
			AllowAppLogin:       false,
		}

		assert.True(t, provider.AllowDashboardLogin)
		assert.False(t, provider.AllowAppLogin)
	})

	t.Run("app only login", func(t *testing.T) {
		provider := OAuthProvider{
			AllowDashboardLogin: false,
			AllowAppLogin:       true,
		}

		assert.False(t, provider.AllowDashboardLogin)
		assert.True(t, provider.AllowAppLogin)
	})

	t.Run("both logins allowed", func(t *testing.T) {
		provider := OAuthProvider{
			AllowDashboardLogin: true,
			AllowAppLogin:       true,
		}

		assert.True(t, provider.AllowDashboardLogin)
		assert.True(t, provider.AllowAppLogin)
	})
}

// =============================================================================
// OAuth 2.0 Endpoints Tests
// =============================================================================

func TestOAuth2Endpoints(t *testing.T) {
	t.Run("standard OAuth endpoints", func(t *testing.T) {
		authURL := "https://example.com/authorize"
		tokenURL := "https://example.com/token"
		userInfoURL := "https://example.com/userinfo"

		provider := OAuthProvider{
			AuthorizationURL: &authURL,
			TokenURL:         &tokenURL,
			UserInfoURL:      &userInfoURL,
		}

		assert.Equal(t, "https://example.com/authorize", *provider.AuthorizationURL)
		assert.Equal(t, "https://example.com/token", *provider.TokenURL)
		assert.Equal(t, "https://example.com/userinfo", *provider.UserInfoURL)
	})

	t.Run("OAuth 2.0 token revocation endpoint", func(t *testing.T) {
		revokeURL := "https://example.com/revoke"

		provider := OAuthProvider{
			RevocationEndpoint: &revokeURL,
		}

		assert.Equal(t, "https://example.com/revoke", *provider.RevocationEndpoint)
	})

	t.Run("OIDC end session endpoint", func(t *testing.T) {
		logoutURL := "https://example.com/logout"

		provider := OAuthProvider{
			EndSessionEndpoint: &logoutURL,
		}

		assert.Equal(t, "https://example.com/logout", *provider.EndSessionEndpoint)
	})
}

// =============================================================================
// getUserIDFromContext Helper Tests
// =============================================================================

func TestGetUserIDFromContext(t *testing.T) {
	t.Run("no user ID in context returns nil", func(t *testing.T) {
		app := newTestApp(t)
		var result *uuid.UUID

		app.Get("/test", func(c fiber.Ctx) error {
			result = getUserIDFromContext(c)
			return c.SendStatus(200)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		_, _ = app.Test(req)

		assert.Nil(t, result)
	})

	t.Run("valid UUID string in context", func(t *testing.T) {
		app := newTestApp(t)
		var result *uuid.UUID
		expectedID := "550e8400-e29b-41d4-a716-446655440000"

		app.Get("/test", func(c fiber.Ctx) error {
			c.Locals("user_id", expectedID)
			result = getUserIDFromContext(c)
			return c.SendStatus(200)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		_, _ = app.Test(req)

		assert.NotNil(t, result)
		assert.Equal(t, expectedID, result.String())
	})

	t.Run("invalid UUID string returns nil", func(t *testing.T) {
		app := newTestApp(t)
		var result *uuid.UUID

		app.Get("/test", func(c fiber.Ctx) error {
			c.Locals("user_id", "not-a-uuid")
			result = getUserIDFromContext(c)
			return c.SendStatus(200)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		_, _ = app.Test(req)

		assert.Nil(t, result)
	})
}
