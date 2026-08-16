package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Constants Tests
// =============================================================================

func TestCookieNames_Constants(t *testing.T) {
	assert.Equal(t, "fluxbase_access_token", AccessTokenCookieName)
	assert.Equal(t, "fluxbase_refresh_token", RefreshTokenCookieName)
}

func TestCookieNames_NotEmpty(t *testing.T) {
	assert.NotEmpty(t, AccessTokenCookieName)
	assert.NotEmpty(t, RefreshTokenCookieName)
}

func TestCookieNames_NoPrefixConflicts(t *testing.T) {
	// Cookies should not have the same prefix to avoid confusion
	assert.NotEqual(t, AccessTokenCookieName, RefreshTokenCookieName)
}

// =============================================================================
// AuthConfigResponse Tests
// =============================================================================

func TestAuthConfigResponse_Fields(t *testing.T) {
	config := AuthConfigResponse{
		SignupEnabled:            true,
		RequireEmailVerification: true,
		MagicLinkEnabled:         true,
		PasswordLoginEnabled:     true,
		MFAAvailable:             true,
		PasswordMinLength:        12,
		PasswordRequireUppercase: true,
		PasswordRequireLowercase: true,
		PasswordRequireNumber:    true,
		PasswordRequireSpecial:   true,
		OAuthProviders:           []OAuthProviderPublic{},
		SAMLProviders:            []SAMLProviderPublic{},
		Captcha:                  nil,
	}

	assert.True(t, config.SignupEnabled)
	assert.True(t, config.RequireEmailVerification)
	assert.True(t, config.MagicLinkEnabled)
	assert.True(t, config.PasswordLoginEnabled)
	assert.True(t, config.MFAAvailable)
	assert.Equal(t, 12, config.PasswordMinLength)
	assert.True(t, config.PasswordRequireUppercase)
	assert.True(t, config.PasswordRequireLowercase)
	assert.True(t, config.PasswordRequireNumber)
	assert.True(t, config.PasswordRequireSpecial)
	assert.Empty(t, config.OAuthProviders)
	assert.Empty(t, config.SAMLProviders)
	assert.Nil(t, config.Captcha)
}

func TestAuthConfigResponse_DefaultValues(t *testing.T) {
	config := AuthConfigResponse{}

	assert.False(t, config.SignupEnabled)
	assert.False(t, config.RequireEmailVerification)
	assert.False(t, config.MagicLinkEnabled)
	assert.False(t, config.PasswordLoginEnabled)
	assert.False(t, config.MFAAvailable)
	assert.Equal(t, 0, config.PasswordMinLength)
	assert.False(t, config.PasswordRequireUppercase)
	assert.False(t, config.PasswordRequireLowercase)
	assert.False(t, config.PasswordRequireNumber)
	assert.False(t, config.PasswordRequireSpecial)
}

func TestAuthConfigResponse_WithProviders(t *testing.T) {
	config := AuthConfigResponse{
		OAuthProviders: []OAuthProviderPublic{
			{Provider: "google", DisplayName: "Google", AuthorizeURL: "/oauth/google"},
			{Provider: "github", DisplayName: "GitHub", AuthorizeURL: "/oauth/github"},
		},
		SAMLProviders: []SAMLProviderPublic{
			{Provider: "okta", DisplayName: "Okta"},
		},
	}

	assert.Len(t, config.OAuthProviders, 2)
	assert.Len(t, config.SAMLProviders, 1)
	assert.Equal(t, "google", config.OAuthProviders[0].Provider)
	assert.Equal(t, "okta", config.SAMLProviders[0].Provider)
}

// =============================================================================
// OAuthProviderPublic Tests
// =============================================================================

func TestOAuthProviderPublic_Fields(t *testing.T) {
	provider := OAuthProviderPublic{
		Provider:     "google",
		DisplayName:  "Sign in with Google",
		AuthorizeURL: "https://accounts.google.com/oauth",
	}

	assert.Equal(t, "google", provider.Provider)
	assert.Equal(t, "Sign in with Google", provider.DisplayName)
	assert.Equal(t, "https://accounts.google.com/oauth", provider.AuthorizeURL)
}

func TestOAuthProviderPublic_CommonProviders(t *testing.T) {
	providers := []OAuthProviderPublic{
		{Provider: "google", DisplayName: "Google"},
		{Provider: "github", DisplayName: "GitHub"},
		{Provider: "microsoft", DisplayName: "Microsoft"},
		{Provider: "facebook", DisplayName: "Facebook"},
		{Provider: "apple", DisplayName: "Apple"},
	}

	for _, p := range providers {
		assert.NotEmpty(t, p.Provider)
		assert.NotEmpty(t, p.DisplayName)
	}
}

// =============================================================================
// SAMLProviderPublic Tests
// =============================================================================

func TestSAMLProviderPublic_Fields(t *testing.T) {
	provider := SAMLProviderPublic{
		Provider:    "okta",
		DisplayName: "Okta SSO",
	}

	assert.Equal(t, "okta", provider.Provider)
	assert.Equal(t, "Okta SSO", provider.DisplayName)
}

func TestSAMLProviderPublic_CommonProviders(t *testing.T) {
	providers := []SAMLProviderPublic{
		{Provider: "okta", DisplayName: "Okta"},
		{Provider: "azure", DisplayName: "Azure AD"},
		{Provider: "onelogin", DisplayName: "OneLogin"},
		{Provider: "auth0", DisplayName: "Auth0"},
	}

	for _, p := range providers {
		assert.NotEmpty(t, p.Provider)
		assert.NotEmpty(t, p.DisplayName)
	}
}

// =============================================================================
// AuthHandler Construction Tests
// =============================================================================

func TestNewAuthHandler_NilDependencies(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "https://example.com", "")

	assert.NotNil(t, handler)
	assert.Nil(t, handler.db)
	assert.Nil(t, handler.authService)
	assert.Nil(t, handler.captchaService)
	assert.Equal(t, "https://example.com", handler.baseURL)
	assert.False(t, handler.secureCookie) // Default is false
}

func TestNewAuthHandler_BaseURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
	}{
		{"with trailing slash", "https://example.com/"},
		{"without trailing slash", "https://example.com"},
		{"with port", "http://localhost:3000"},
		{"localhost", "http://127.0.0.1:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewAuthHandler(nil, nil, nil, tt.baseURL, "")
			assert.Equal(t, tt.baseURL, handler.baseURL)
		})
	}
}

func TestAuthHandler_SetSecureCookie(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "https://example.com", "")

	// Default is false
	assert.False(t, handler.secureCookie)

	// Set to true
	handler.SetSecureCookie(true)
	assert.True(t, handler.secureCookie)

	// Set back to false
	handler.SetSecureCookie(false)
	assert.False(t, handler.secureCookie)
}

func TestAuthHandler_SetSAMLService(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "https://example.com", "")

	assert.Nil(t, handler.samlService)

	// SetSAMLService is tested for nil safety
	handler.SetSAMLService(nil)
	assert.Nil(t, handler.samlService)
}

// =============================================================================
// getAccessToken Tests
// =============================================================================

func TestGetAccessToken_FromCookie(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Get("/test", func(c fiber.Ctx) error {
		token := handler.getAccessToken(c)
		return c.SendString(token)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  AccessTokenCookieName,
		Value: "cookie_token_123",
	})

	resp, err := app.Test(req)
	require.NoError(t, err)

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "cookie_token_123", string(body))
}

func TestGetAccessToken_FromBearerHeader(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Get("/test", func(c fiber.Ctx) error {
		token := handler.getAccessToken(c)
		return c.SendString(token)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer header_token_456")

	resp, err := app.Test(req)
	require.NoError(t, err)

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "header_token_456", string(body))
}

func TestGetAccessToken_CookiePriority(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Get("/test", func(c fiber.Ctx) error {
		token := handler.getAccessToken(c)
		return c.SendString(token)
	})

	// Both cookie and header set - cookie should take priority
	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  AccessTokenCookieName,
		Value: "cookie_token",
	})
	req.Header.Set("Authorization", "Bearer header_token")

	resp, err := app.Test(req)
	require.NoError(t, err)

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "cookie_token", string(body))
}

func TestGetAccessToken_HeaderWithoutBearer(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Get("/test", func(c fiber.Ctx) error {
		token := handler.getAccessToken(c)
		return c.SendString(token)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "token_without_bearer")

	resp, err := app.Test(req)
	require.NoError(t, err)

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "token_without_bearer", string(body))
}

func TestGetAccessToken_Empty(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Get("/test", func(c fiber.Ctx) error {
		token := handler.getAccessToken(c)
		return c.SendString(token)
	})

	req := httptest.NewRequest("GET", "/test", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)

	body, _ := io.ReadAll(resp.Body)
	assert.Empty(t, string(body))
}

func TestGetAccessToken_ShortBearerHeader(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Get("/test", func(c fiber.Ctx) error {
		token := handler.getAccessToken(c)
		return c.SendString(token)
	})

	// "Bearer " is 7 chars, so this should not match the prefix check
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer")

	resp, err := app.Test(req)
	require.NoError(t, err)

	body, _ := io.ReadAll(resp.Body)
	// Should return the raw value since len <= 7
	assert.Equal(t, "Bearer", string(body))
}

// =============================================================================
// getRefreshToken Tests
// =============================================================================

func TestGetRefreshToken_FromCookie(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Get("/test", func(c fiber.Ctx) error {
		token := handler.getRefreshToken(c)
		return c.SendString(token)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  RefreshTokenCookieName,
		Value: "refresh_token_789",
	})

	resp, err := app.Test(req)
	require.NoError(t, err)

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "refresh_token_789", string(body))
}

func TestGetRefreshToken_Empty(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Get("/test", func(c fiber.Ctx) error {
		token := handler.getRefreshToken(c)
		return c.SendString(token)
	})

	req := httptest.NewRequest("GET", "/test", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)

	body, _ := io.ReadAll(resp.Body)
	assert.Empty(t, string(body))
}

// =============================================================================
// Cookie Setting Tests
// =============================================================================

func TestSetAuthCookies(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Get("/test", func(c fiber.Ctx) error {
		handler.setAuthCookies(c, "access_token_test", "refresh_token_test", 3600)
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Check cookies are set
	cookies := resp.Cookies()

	var accessCookie, refreshCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == AccessTokenCookieName {
			accessCookie = cookie
		}
		if cookie.Name == RefreshTokenCookieName {
			refreshCookie = cookie
		}
	}

	require.NotNil(t, accessCookie, "Access token cookie should be set")
	assert.Equal(t, "access_token_test", accessCookie.Value)
	assert.True(t, accessCookie.HttpOnly)
	assert.Equal(t, "/", accessCookie.Path)

	require.NotNil(t, refreshCookie, "Refresh token cookie should be set")
	assert.Equal(t, "refresh_token_test", refreshCookie.Value)
	assert.True(t, refreshCookie.HttpOnly)
	assert.Equal(t, "/api/v1/auth", refreshCookie.Path)
}

func TestSetAuthCookies_Secure(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")
	handler.SetSecureCookie(true)

	app := newTestApp(t)
	app.Get("/test", func(c fiber.Ctx) error {
		handler.setAuthCookies(c, "token", "refresh", 3600)
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	cookies := resp.Cookies()
	for _, cookie := range cookies {
		if cookie.Name == AccessTokenCookieName || cookie.Name == RefreshTokenCookieName {
			assert.True(t, cookie.Secure, "Cookie %s should be secure", cookie.Name)
		}
	}
}

func TestClearAuthCookies(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Get("/test", func(c fiber.Ctx) error {
		handler.clearAuthCookies(c)
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	cookies := resp.Cookies()
	for _, cookie := range cookies {
		if cookie.Name == AccessTokenCookieName || cookie.Name == RefreshTokenCookieName {
			// Cleared cookies should have empty value and MaxAge <= 0
			// Note: Go's http.Cookie parsing may return 0 for immediate expiration
			assert.Empty(t, cookie.Value)
			assert.LessOrEqual(t, cookie.MaxAge, 0, "Cookie %s should expire immediately", cookie.Name)
		}
	}
}

// =============================================================================
// GetCSRFToken Tests
// =============================================================================

func TestGetCSRFToken_ReturnsToken(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)

	// Simulate CSRF middleware setting the cookie
	app.Use(func(c fiber.Ctx) error {
		c.Cookie(&fiber.Cookie{
			Name:  "csrf_token",
			Value: "test_csrf_token",
		})
		return c.Next()
	})

	app.Get("/auth/csrf", handler.GetCSRFToken)

	req := httptest.NewRequest("GET", "/auth/csrf", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, 200, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "csrf_token")
}

// =============================================================================
// Request Validation Tests
// =============================================================================
// NOTE: These tests require a real auth service to test validation that happens
// after request parsing. Tests that validate input before service calls are kept.

// =============================================================================
// Invalid JSON Body Tests
// =============================================================================
// NOTE: Invalid JSON tests require service layer because JSON parsing doesn't
// fail early - it's only caught when the handler processes the request.

// =============================================================================
// Protected Route Tests (No Auth)
// =============================================================================

func TestGetUser_NoAuth(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Get("/auth/user", handler.GetUser)

	req := httptest.NewRequest("GET", "/auth/user", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestUpdateUser_NoAuth(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Patch("/auth/user", handler.UpdateUser)

	body := `{"name": "Test"}`
	req := httptest.NewRequest("PATCH", "/auth/user", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestSetupTOTP_NoAuth(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Post("/auth/2fa/setup", handler.SetupTOTP)

	req := httptest.NewRequest("POST", "/auth/2fa/setup", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestEnableTOTP_NoAuth(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Post("/auth/2fa/enable", handler.EnableTOTP)

	body := `{"code": "123456"}`
	req := httptest.NewRequest("POST", "/auth/2fa/enable", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestDisableTOTP_NoAuth(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Post("/auth/2fa/disable", handler.DisableTOTP)

	body := `{"password": "secret"}`
	req := httptest.NewRequest("POST", "/auth/2fa/disable", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestGetTOTPStatus_NoAuth(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Get("/auth/2fa/status", handler.GetTOTPStatus)

	req := httptest.NewRequest("GET", "/auth/2fa/status", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkGetAccessToken_Cookie(b *testing.B) {
	handler := NewAuthHandler(nil, nil, nil, "")
	app := fiber.New()
	var captured fiber.Ctx

	app.Get("/test", func(c fiber.Ctx) error {
		captured = c
		return nil
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  AccessTokenCookieName,
		Value: "test_token",
	})
	_, _ = app.Test(req)

	if captured == nil {
		b.Fatal("Failed to capture context")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = handler.getAccessToken(captured)
	}
}

func BenchmarkGetAccessToken_Header(b *testing.B) {
	handler := NewAuthHandler(nil, nil, nil, "")
	app := fiber.New()
	var captured fiber.Ctx

	app.Get("/test", func(c fiber.Ctx) error {
		captured = c
		return nil
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer test_token_123")
	_, _ = app.Test(req)

	if captured == nil {
		b.Fatal("Failed to capture context")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = handler.getAccessToken(captured)
	}
}

func BenchmarkSetAuthCookies(b *testing.B) {
	handler := NewAuthHandler(nil, nil, nil, "")
	app := fiber.New()

	app.Get("/test", func(c fiber.Ctx) error {
		handler.setAuthCookies(c, "access", "refresh", 3600)
		return nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		_, _ = app.Test(req)
	}
}

// =============================================================================
// Handler Tests with 0% Coverage
// =============================================================================

func TestSignOut_NoToken(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Post("/auth/signout", handler.SignOut)

	req := httptest.NewRequest("POST", "/auth/signout", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)

	// Should return 400 when no token is provided
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestRefreshToken_NoToken(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Post("/auth/token/refresh", handler.RefreshToken)

	// Test with no body and no cookies
	req := httptest.NewRequest("POST", "/auth/token/refresh", strings.NewReader(""))

	resp, err := app.Test(req)
	require.NoError(t, err)

	// Should return 400 when no refresh token is provided
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestSendMagicLink_Validation(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Post("/auth/magiclink", handler.SendMagicLink)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "empty body",
			body:       "",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing email",
			body:       `{}`,
			wantStatus: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/auth/magiclink", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

// VerifyMagicLink requires auth service - skipped for unit testing

func TestRequestPasswordReset_Validation(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Post("/auth/password/reset", handler.RequestPasswordReset)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "empty body",
			body:       "",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing email",
			body:       `{}`,
			wantStatus: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/auth/password/reset", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestResetPassword_Validation(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Post("/auth/password/reset/confirm", handler.ResetPassword)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "empty body",
			body:       "",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing token",
			body:       `{"password": "newpass123"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing password",
			body:       `{"token": "test-token"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "short password",
			body:       `{"token": "test", "password": "short"}`,
			wantStatus: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/auth/password/reset/confirm", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestVerifyPasswordResetToken_Validation(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Get("/auth/password/reset/verify", handler.VerifyPasswordResetToken)

	tests := []struct {
		name       string
		url        string
		wantStatus int
	}{
		{
			name:       "missing token",
			url:        "/auth/password/reset/verify",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "empty token",
			url:        "/auth/password/reset/verify?token=",
			wantStatus: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestVerifyEmail_Validation(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Post("/auth/verify", handler.VerifyEmail)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "empty body",
			body:       "",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing token",
			body:       `{}`,
			wantStatus: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/auth/verify", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestResendVerificationEmail_Validation(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Post("/auth/resend-verification", handler.ResendVerificationEmail)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "empty body",
			body:       "",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing email",
			body:       `{}`,
			wantStatus: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/auth/resend-verification", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestStartImpersonation_NoAuth(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Post("/auth/impersonation/start", handler.StartImpersonation)

	body := `{"user_id": "user-123", "reason": "Testing"}`
	req := httptest.NewRequest("POST", "/auth/impersonation/start", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	// Should fail without auth/admin privileges
	assert.NotEqual(t, fiber.StatusOK, resp.StatusCode)
}

func TestStopImpersonation_NoAuth(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Post("/auth/impersonation/stop", handler.StopImpersonation)

	req := httptest.NewRequest("POST", "/auth/impersonation/stop", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)

	// Should fail without auth
	assert.NotEqual(t, fiber.StatusOK, resp.StatusCode)
}

func TestGetActiveImpersonation_NoAuth(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Get("/auth/impersonation/active", handler.GetActiveImpersonation)

	req := httptest.NewRequest("GET", "/auth/impersonation/active", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)

	// Should fail without auth
	assert.NotEqual(t, fiber.StatusOK, resp.StatusCode)
}

func TestListImpersonationSessions_NoAuth(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Get("/auth/impersonation/sessions", handler.ListImpersonationSessions)

	req := httptest.NewRequest("GET", "/auth/impersonation/sessions", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)

	// Should fail without auth
	assert.NotEqual(t, fiber.StatusOK, resp.StatusCode)
}

func TestGetCSRFToken_Handler(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Get("/auth/csrf-token", handler.GetCSRFToken)

	req := httptest.NewRequest("GET", "/auth/csrf-token", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)

	// Should return a token even without auth
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.NotEmpty(t, body)
}

// TestSignUp_Validation removed - SignUp handler calls IsSignupEnabled() before any validation,
// which requires a non-nil authService. Cannot test validation without mocking the full service.

// TestSignIn_Validation removed - SignIn handler calls isPasswordLoginDisabled() before any validation,
// which requires a non-nil authService. Cannot test validation without mocking the full service.

func TestUpdateUser_AuthHandlerValidation(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Patch("/auth/user", handler.UpdateUser)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "empty body - returns 401 (auth checked before body parsing)",
			body:       "",
			wantStatus: fiber.StatusUnauthorized,
		},
		{
			name:       "invalid email format - returns 401 (auth checked before body parsing)",
			body:       `{"email": "invalid-email"}`,
			wantStatus: fiber.StatusUnauthorized,
		},
		{
			name:       "valid empty update - returns 401 (no auth middleware)",
			body:       `{}`,
			wantStatus: fiber.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("PATCH", "/auth/user", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

// RegisterRoutes testing skipped - requires valid app state

// =============================================================================
// Additional Handler Tests for Improved Coverage
// =============================================================================

func TestStartAnonImpersonation_NoAuth(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Post("/auth/impersonate/anon", handler.StartAnonImpersonation)

	body := `{"reason": "Testing anonymous impersonation"}`
	req := httptest.NewRequest("POST", "/auth/impersonate/anon", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	// Should fail without auth
	assert.NotEqual(t, fiber.StatusOK, resp.StatusCode)
}

func TestStartAnonImpersonation_Validation(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Post("/auth/impersonate/anon", handler.StartAnonImpersonation)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "empty body",
			body:       "",
			wantStatus: fiber.StatusUnauthorized,
		},
		{
			name:       "missing reason",
			body:       `{}`,
			wantStatus: fiber.StatusUnauthorized,
		},
		{
			name:       "empty reason",
			body:       `{"reason": ""}`,
			wantStatus: fiber.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/auth/impersonate/anon", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestStartServiceImpersonation_NoAuth(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Post("/auth/impersonate/service", handler.StartServiceImpersonation)

	body := `{"reason": "Testing service role impersonation"}`
	req := httptest.NewRequest("POST", "/auth/impersonate/service", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	// Should fail without auth
	assert.NotEqual(t, fiber.StatusOK, resp.StatusCode)
}

func TestStartServiceImpersonation_Validation(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Post("/auth/impersonate/service", handler.StartServiceImpersonation)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "empty body",
			body:       "",
			wantStatus: fiber.StatusUnauthorized,
		},
		{
			name:       "missing reason",
			body:       `{}`,
			wantStatus: fiber.StatusUnauthorized,
		},
		{
			name:       "empty reason",
			body:       `{"reason": ""}`,
			wantStatus: fiber.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/auth/impersonate/service", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestGetCaptchaConfig_NilService(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Get("/auth/captcha/config", handler.GetCaptchaConfig)

	req := httptest.NewRequest("GET", "/auth/captcha/config", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), `"enabled":false`)
}

func TestCheckCaptcha_Validation(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Post("/auth/captcha/check", handler.CheckCaptcha)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "empty body",
			body:       "",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "invalid endpoint",
			body:       `{"endpoint": "invalid"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing endpoint",
			body:       `{}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "valid endpoint - returns disabled",
			body:       `{"endpoint": "signup"}`,
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/auth/captcha/check", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestVerifyMagicLink_Validation(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Post("/auth/magiclink/verify", handler.VerifyMagicLink)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "empty body",
			body:       "",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing token",
			body:       `{}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "empty token",
			body:       `{"token": ""}`,
			wantStatus: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/auth/magiclink/verify", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestVerifyTOTP_Validation(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Post("/auth/2fa/verify", handler.VerifyTOTP)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "empty body",
			body:       "",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing user_id",
			body:       `{"code": "123456"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing code",
			body:       `{"user_id": "user-123"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "empty code",
			body:       `{"user_id": "user-123", "code": ""}`,
			wantStatus: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/auth/2fa/verify", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestSendOTP_Validation(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Post("/auth/otp/signin", handler.SendOTP)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "empty body",
			body:       "",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing both email and phone",
			body:       `{}`,
			wantStatus: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/auth/otp/signin", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestVerifyOTP_Validation(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Post("/auth/otp/verify", handler.VerifyOTP)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "empty body",
			body:       "",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing token",
			body:       `{"email": "test@example.com"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing both email and phone",
			body:       `{"token": "123456"}`,
			wantStatus: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/auth/otp/verify", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestResendOTP_Validation(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Post("/auth/otp/resend", handler.ResendOTP)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "empty body",
			body:       "",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing both email and phone",
			body:       `{}`,
			wantStatus: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/auth/otp/resend", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestGetUserIdentities_NoAuth(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Get("/auth/user/identities", handler.GetUserIdentities)

	req := httptest.NewRequest("GET", "/auth/user/identities", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestLinkIdentity_NoAuth(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Post("/auth/user/identities", handler.LinkIdentity)

	body := `{"provider": "google"}`
	req := httptest.NewRequest("POST", "/auth/user/identities", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestLinkIdentity_Validation(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Post("/auth/user/identities", handler.LinkIdentity)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "empty body",
			body:       "",
			wantStatus: fiber.StatusUnauthorized,
		},
		{
			name:       "missing provider",
			body:       `{}`,
			wantStatus: fiber.StatusUnauthorized,
		},
		{
			name:       "empty provider",
			body:       `{"provider": ""}`,
			wantStatus: fiber.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/auth/user/identities", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestUnlinkIdentity_NoAuth(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Delete("/auth/user/identities/identity-123", handler.UnlinkIdentity)

	req := httptest.NewRequest("DELETE", "/auth/user/identities/identity-123", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestReauthenticate_NoAuth(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Post("/auth/reauthenticate", handler.Reauthenticate)

	req := httptest.NewRequest("POST", "/auth/reauthenticate", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestSignInWithIDToken_Validation(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Post("/auth/signin/idtoken", handler.SignInWithIDToken)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "empty body",
			body:       "",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing provider",
			body:       `{"token": "id-token-123"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing token",
			body:       `{"provider": "google"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "empty provider",
			body:       `{"provider": "", "token": "id-token"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "empty token",
			body:       `{"provider": "google", "token": ""}`,
			wantStatus: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/auth/signin/idtoken", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

// =============================================================================
// Impersonation Session Query Parameter Tests
// =============================================================================

func TestListImpersonationSessions_QueryParameters(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := newTestApp(t)
	app.Get("/auth/impersonation/sessions", handler.ListImpersonationSessions)

	tests := []struct {
		name       string
		url        string
		wantStatus int
	}{
		{
			name:       "no parameters",
			url:        "/auth/impersonation/sessions",
			wantStatus: fiber.StatusUnauthorized,
		},
		{
			name:       "with limit",
			url:        "/auth/impersonation/sessions?limit=10",
			wantStatus: fiber.StatusUnauthorized,
		},
		{
			name:       "with offset",
			url:        "/auth/impersonation/sessions?offset=5",
			wantStatus: fiber.StatusUnauthorized,
		},
		{
			name:       "with limit and offset",
			url:        "/auth/impersonation/sessions?limit=10&offset=5",
			wantStatus: fiber.StatusUnauthorized,
		},
		{
			name:       "with invalid limit",
			url:        "/auth/impersonation/sessions?limit=invalid",
			wantStatus: fiber.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}
