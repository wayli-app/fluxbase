package api

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
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
	handler := NewAuthHandler(nil, nil, nil, "https://example.com")

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
			handler := NewAuthHandler(nil, nil, nil, tt.baseURL)
			assert.Equal(t, tt.baseURL, handler.baseURL)
		})
	}
}

func TestAuthHandler_SetSecureCookie(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "https://example.com")

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
	handler := NewAuthHandler(nil, nil, nil, "https://example.com")

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

	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		token := handler.getAccessToken(c)
		return c.SendString(token)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&httptest.Cookie{
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

	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
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

	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		token := handler.getAccessToken(c)
		return c.SendString(token)
	})

	// Both cookie and header set - cookie should take priority
	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&httptest.Cookie{
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

	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
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

	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
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

	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
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

	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		token := handler.getRefreshToken(c)
		return c.SendString(token)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&httptest.Cookie{
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

	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
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

	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		handler.setAuthCookies(c, "access_token_test", "refresh_token_test", 3600)
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Check cookies are set
	cookies := resp.Cookies()

	var accessCookie, refreshCookie *httptest.Cookie
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

	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
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

	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		handler.clearAuthCookies(c)
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	cookies := resp.Cookies()
	for _, cookie := range cookies {
		if cookie.Name == AccessTokenCookieName || cookie.Name == RefreshTokenCookieName {
			// Cleared cookies should have empty value and negative MaxAge
			assert.Empty(t, cookie.Value)
			assert.Less(t, cookie.MaxAge, 0, "Cookie %s should expire immediately", cookie.Name)
		}
	}
}

// =============================================================================
// SignInAnonymous Tests (Deprecated)
// =============================================================================

func TestSignInAnonymous_Disabled(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := fiber.New()
	app.Post("/auth/signin/anonymous", handler.SignInAnonymous)

	req := httptest.NewRequest("POST", "/auth/signin/anonymous", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusGone, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "disabled")
}

// =============================================================================
// GetCSRFToken Tests
// =============================================================================

func TestGetCSRFToken_ReturnsToken(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := fiber.New()

	// Simulate CSRF middleware setting the cookie
	app.Use(func(c *fiber.Ctx) error {
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

func TestSignUp_EmptyEmail(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := fiber.New()
	app.Post("/auth/signup", handler.SignUp)

	body := `{"email": "", "password": "password123"}`
	req := httptest.NewRequest("POST", "/auth/signup", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	// Without a real authService, it might fail differently
	// but we're testing the handler structure works
	assert.True(t, resp.StatusCode >= 400)
}

func TestSignIn_EmptyCredentials(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := fiber.New()
	app.Post("/auth/signin", handler.SignIn)

	body := `{"email": "", "password": ""}`
	req := httptest.NewRequest("POST", "/auth/signin", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(respBody), "required")
}

func TestRefreshToken_EmptyToken(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := fiber.New()
	app.Post("/auth/refresh", handler.RefreshToken)

	body := `{"refresh_token": ""}`
	req := httptest.NewRequest("POST", "/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(respBody), "required")
}

func TestRequestPasswordReset_EmptyEmail(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := fiber.New()
	app.Post("/auth/password/reset", handler.RequestPasswordReset)

	body := `{"email": ""}`
	req := httptest.NewRequest("POST", "/auth/password/reset", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(respBody), "required")
}

func TestResetPassword_EmptyToken(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := fiber.New()
	app.Post("/auth/password/reset/confirm", handler.ResetPassword)

	body := `{"token": "", "new_password": "newpass123"}`
	req := httptest.NewRequest("POST", "/auth/password/reset/confirm", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(respBody), "required")
}

func TestResetPassword_EmptyPassword(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := fiber.New()
	app.Post("/auth/password/reset/confirm", handler.ResetPassword)

	body := `{"token": "valid_token", "new_password": ""}`
	req := httptest.NewRequest("POST", "/auth/password/reset/confirm", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(respBody), "required")
}

func TestVerifyEmail_EmptyToken(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := fiber.New()
	app.Post("/auth/verify-email", handler.VerifyEmail)

	body := `{"token": ""}`
	req := httptest.NewRequest("POST", "/auth/verify-email", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(respBody), "required")
}

func TestSendMagicLink_EmptyEmail(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := fiber.New()
	app.Post("/auth/magiclink", handler.SendMagicLink)

	body := `{"email": ""}`
	req := httptest.NewRequest("POST", "/auth/magiclink", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(respBody), "required")
}

func TestVerifyMagicLink_EmptyToken(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := fiber.New()
	app.Post("/auth/magiclink/verify", handler.VerifyMagicLink)

	body := `{"token": ""}`
	req := httptest.NewRequest("POST", "/auth/magiclink/verify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(respBody), "required")
}

func TestVerifyTOTP_EmptyFields(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := fiber.New()
	app.Post("/auth/2fa/verify", handler.VerifyTOTP)

	body := `{"user_id": "", "code": ""}`
	req := httptest.NewRequest("POST", "/auth/2fa/verify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(respBody), "required")
}

// =============================================================================
// Invalid JSON Body Tests
// =============================================================================

func TestSignUp_InvalidJSON(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := fiber.New()
	app.Post("/auth/signup", handler.SignUp)

	req := httptest.NewRequest("POST", "/auth/signup", strings.NewReader("not valid json"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestSignIn_InvalidJSON(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := fiber.New()
	app.Post("/auth/signin", handler.SignIn)

	req := httptest.NewRequest("POST", "/auth/signin", strings.NewReader("{{invalid json"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

// =============================================================================
// Protected Route Tests (No Auth)
// =============================================================================

func TestGetUser_NoAuth(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := fiber.New()
	app.Get("/auth/user", handler.GetUser)

	req := httptest.NewRequest("GET", "/auth/user", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestUpdateUser_NoAuth(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := fiber.New()
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

	app := fiber.New()
	app.Post("/auth/2fa/setup", handler.SetupTOTP)

	req := httptest.NewRequest("POST", "/auth/2fa/setup", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestEnableTOTP_NoAuth(t *testing.T) {
	handler := NewAuthHandler(nil, nil, nil, "")

	app := fiber.New()
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

	app := fiber.New()
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

	app := fiber.New()
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
	var captured *fiber.Ctx

	app.Get("/test", func(c *fiber.Ctx) error {
		captured = c
		return nil
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&httptest.Cookie{
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
	var captured *fiber.Ctx

	app.Get("/test", func(c *fiber.Ctx) error {
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

	app.Get("/test", func(c *fiber.Ctx) error {
		handler.setAuthCookies(c, "access", "refresh", 3600)
		return nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		_, _ = app.Test(req)
	}
}
