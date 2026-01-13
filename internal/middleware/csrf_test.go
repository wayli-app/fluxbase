package middleware

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/storage/memory/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultCSRFConfig(t *testing.T) {
	cfg := DefaultCSRFConfig()

	assert.Equal(t, 32, cfg.TokenLength)
	assert.Equal(t, "header:X-CSRF-Token", cfg.TokenLookup)
	assert.Equal(t, "csrf_token", cfg.CookieName)
	assert.Equal(t, "/", cfg.CookiePath)
	assert.False(t, cfg.CookieSecure)
	assert.True(t, cfg.CookieHTTPOnly)
	assert.Equal(t, "Strict", cfg.CookieSameSite)
	assert.Equal(t, 24*time.Hour, cfg.Expiration)
	assert.NotNil(t, cfg.Storage)
}

func TestCSRF_SkipsSafeMethods(t *testing.T) {
	app := fiber.New()
	app.Use(CSRF())

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})
	app.Head("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})
	app.Options("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// GET should pass without CSRF token
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// HEAD should pass without CSRF token
	req = httptest.NewRequest("HEAD", "/test", nil)
	resp, err = app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// OPTIONS should pass without CSRF token
	req = httptest.NewRequest("OPTIONS", "/test", nil)
	resp, err = app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestCSRF_SkipsSpecialPaths(t *testing.T) {
	app := fiber.New()
	app.Use(CSRF())

	// Set up routes for special paths
	specialPaths := []string{"/realtime", "/health", "/ready", "/metrics"}
	for _, path := range specialPaths {
		p := path
		app.Post(p, func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})
	}

	for _, path := range specialPaths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest("POST", path, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, 200, resp.StatusCode)
		})
	}
}

func TestCSRF_SkipsPublicAuthEndpoints(t *testing.T) {
	app := fiber.New()
	app.Use(CSRF())

	publicPaths := []string{
		"/api/v1/auth/signup",
		"/api/v1/auth/signin",
		"/api/v1/auth/signout",
		"/api/v1/auth/refresh",
		"/api/v1/auth/password/reset",
		"/api/v1/admin/setup",
		"/api/v1/admin/login",
	}

	for _, path := range publicPaths {
		p := path
		app.Post(p, func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})
	}

	for _, path := range publicPaths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest("POST", path, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, 200, resp.StatusCode)
		})
	}
}

func TestCSRF_SkipsBearerAuth(t *testing.T) {
	app := fiber.New()
	app.Use(CSRF())
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Authorization", "Bearer some-jwt-token")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestCSRF_SkipsClientKeyAuth(t *testing.T) {
	app := fiber.New()
	app.Use(CSRF())
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("clientkey", "some-client-key")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestCSRF_RejectsMissingToken(t *testing.T) {
	app := fiber.New()
	app.Use(CSRF())
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// First POST without any token should fail and set a cookie
	req := httptest.NewRequest("POST", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode)

	// Should have set a CSRF cookie
	cookies := resp.Header.Values("Set-Cookie")
	assert.NotEmpty(t, cookies)
	var hasCSRFCookie bool
	for _, cookie := range cookies {
		if strings.Contains(cookie, "csrf_token=") {
			hasCSRFCookie = true
			break
		}
	}
	assert.True(t, hasCSRFCookie)
}

func TestCSRF_RejectsInvalidToken(t *testing.T) {
	storage := memory.New()
	app := fiber.New()
	app.Use(CSRF(CSRFConfig{
		TokenLength: 32,
		TokenLookup: "header:X-CSRF-Token",
		CookieName:  "csrf_token",
		Storage:     storage,
		Expiration:  time.Hour,
	}))
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Set up a valid token in storage
	validToken := "valid-token-12345678901234567890"
	storage.Set(validToken, []byte("1"), time.Hour)

	// Request with cookie but wrong header token
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Cookie", "csrf_token="+validToken)
	req.Header.Set("X-CSRF-Token", "wrong-token")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode)
}

func TestCSRF_AcceptsValidToken(t *testing.T) {
	storage := memory.New()
	app := fiber.New()
	app.Use(CSRF(CSRFConfig{
		TokenLength: 32,
		TokenLookup: "header:X-CSRF-Token",
		CookieName:  "csrf_token",
		Storage:     storage,
		Expiration:  time.Hour,
	}))
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Set up a valid token in storage
	validToken := "valid-token-12345678901234567890"
	storage.Set(validToken, []byte("1"), time.Hour)

	// Request with matching cookie and header token
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Cookie", "csrf_token="+validToken)
	req.Header.Set("X-CSRF-Token", validToken)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestGenerateCSRFToken(t *testing.T) {
	t.Run("generates token of correct length", func(t *testing.T) {
		token, err := generateCSRFToken(32)
		require.NoError(t, err)
		// Base64 encoding produces ~4/3 the length
		assert.True(t, len(token) > 32)
	})

	t.Run("generates unique tokens", func(t *testing.T) {
		token1, err := generateCSRFToken(32)
		require.NoError(t, err)
		token2, err := generateCSRFToken(32)
		require.NoError(t, err)
		assert.NotEqual(t, token1, token2)
	})
}

func TestGetCSRFToken(t *testing.T) {
	app := fiber.New()

	var tokenFromHelper string
	app.Get("/test", func(c *fiber.Ctx) error {
		tokenFromHelper = GetCSRFToken(c)
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Cookie", "csrf_token=test-token-value")
	_, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, "test-token-value", tokenFromHelper)
}

func TestIsPublicAuthEndpoint(t *testing.T) {
	testCases := []struct {
		path     string
		expected bool
	}{
		{"/api/v1/auth/signup", true},
		{"/api/v1/auth/signin", true},
		{"/api/v1/auth/signout", true},
		{"/api/v1/auth/refresh", true},
		{"/api/v1/auth/password/reset", true},
		{"/api/v1/auth/oauth", true},
		{"/api/v1/auth/oauth/google", true},
		{"/api/v1/auth/oauth/google/callback", true},
		{"/api/v1/admin/setup", true},
		{"/api/v1/admin/login", true},
		{"/api/v1/admin/login/2fa", true},
		{"/api/v1/users", false},
		{"/api/v1/data/users", false},
		{"/some/other/path", false},
		{"/dashboard/auth/login", true},
		{"/dashboard/auth/signup", true},
		{"/dashboard/data", false},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			result := isPublicAuthEndpoint(tc.path)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCSRFConfig_CustomConfig(t *testing.T) {
	cfg := CSRFConfig{
		TokenLength:    64,
		TokenLookup:    "form:_csrf",
		CookieName:     "my_csrf",
		CookieDomain:   "example.com",
		CookiePath:     "/api",
		CookieSecure:   true,
		CookieHTTPOnly: false,
		CookieSameSite: "Lax",
		Expiration:     12 * time.Hour,
	}

	assert.Equal(t, 64, cfg.TokenLength)
	assert.Equal(t, "form:_csrf", cfg.TokenLookup)
	assert.Equal(t, "my_csrf", cfg.CookieName)
	assert.Equal(t, "example.com", cfg.CookieDomain)
	assert.Equal(t, "/api", cfg.CookiePath)
	assert.True(t, cfg.CookieSecure)
	assert.False(t, cfg.CookieHTTPOnly)
	assert.Equal(t, "Lax", cfg.CookieSameSite)
	assert.Equal(t, 12*time.Hour, cfg.Expiration)
}

// =============================================================================
// Additional Security Tests
// =============================================================================

func TestCSRF_RejectsExpiredToken(t *testing.T) {
	storage := memory.New()
	app := fiber.New()
	app.Use(CSRF(CSRFConfig{
		TokenLength: 32,
		TokenLookup: "header:X-CSRF-Token",
		CookieName:  "csrf_token",
		Storage:     storage,
		Expiration:  time.Hour,
	}))
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Token NOT in storage (simulates expiration)
	expiredToken := "expired-token-12345678901234567890"

	// Request with matching cookie and header but token not in storage
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Cookie", "csrf_token="+expiredToken)
	req.Header.Set("X-CSRF-Token", expiredToken)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode)
}

func TestCSRF_RejectsEmptyHeaderToken(t *testing.T) {
	storage := memory.New()
	app := fiber.New()
	app.Use(CSRF(CSRFConfig{
		TokenLength: 32,
		TokenLookup: "header:X-CSRF-Token",
		CookieName:  "csrf_token",
		Storage:     storage,
		Expiration:  time.Hour,
	}))
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	validToken := "valid-token-12345678901234567890"
	storage.Set(validToken, []byte("1"), time.Hour)

	// Request with cookie but empty header token
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Cookie", "csrf_token="+validToken)
	req.Header.Set("X-CSRF-Token", "")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode)
}

func TestCSRF_FormTokenLookup(t *testing.T) {
	storage := memory.New()
	app := fiber.New()
	app.Use(CSRF(CSRFConfig{
		TokenLength: 32,
		TokenLookup: "form:_csrf",
		CookieName:  "csrf_token",
		Storage:     storage,
		Expiration:  time.Hour,
	}))
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	validToken := "valid-token-form-12345678901234567890"
	storage.Set(validToken, []byte("1"), time.Hour)

	// Request with form data
	req := httptest.NewRequest("POST", "/test", strings.NewReader("_csrf="+validToken))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", "csrf_token="+validToken)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestCSRF_PreventsAttackWithoutToken(t *testing.T) {
	app := fiber.New()
	app.Use(CSRF())

	handlerCalled := false
	app.Post("/api/transfer", func(c *fiber.Ctx) error {
		handlerCalled = true
		return c.SendString("Transfer completed")
	})

	// Attacker tries to submit without CSRF token
	req := httptest.NewRequest("POST", "/api/transfer", strings.NewReader("amount=1000&to=attacker"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode)
	assert.False(t, handlerCalled, "Handler should not be called without CSRF token")
}

func TestCSRF_ShortAuthorizationHeader(t *testing.T) {
	app := fiber.New()
	app.Use(CSRF())
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Authorization header too short to be "Bearer X"
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Authorization", "Bear")
	resp, err := app.Test(req)
	require.NoError(t, err)
	// Should reject because it's not a valid Bearer token
	assert.Equal(t, 403, resp.StatusCode)
}

func TestCSRF_DefaultStorageInitialized(t *testing.T) {
	// Test that CSRF middleware with no config initializes storage properly
	app := fiber.New()
	app.Use(CSRF())

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Should not panic
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestCSRF_NilStorageInitialized(t *testing.T) {
	// Test that nil storage in config gets initialized
	app := fiber.New()
	app.Use(CSRF(CSRFConfig{
		TokenLength: 32,
		Storage:     nil, // Explicitly nil
	}))

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Should not panic
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

// =============================================================================
// Token Generation Edge Cases
// =============================================================================

func TestGenerateCSRFToken_VariousLengths(t *testing.T) {
	lengths := []int{8, 16, 32, 64, 128, 256}

	for _, length := range lengths {
		t.Run("length_"+string(rune('0'+length/10))+string(rune('0'+length%10)), func(t *testing.T) {
			token, err := generateCSRFToken(length)
			require.NoError(t, err)
			assert.NotEmpty(t, token)
		})
	}
}

func TestGenerateCSRFToken_ZeroLength(t *testing.T) {
	token, err := generateCSRFToken(0)
	require.NoError(t, err)
	assert.Empty(t, token)
}

func TestGenerateCSRFToken_URLSafeCharacters(t *testing.T) {
	token, err := generateCSRFToken(32)
	require.NoError(t, err)

	// URL-safe base64 should only contain: A-Z, a-z, 0-9, -, _, =
	for _, char := range token {
		valid := (char >= 'A' && char <= 'Z') ||
			(char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_' || char == '='
		assert.True(t, valid, "Invalid character in token: %c", char)
	}
}

// =============================================================================
// Public Auth Endpoint Edge Cases
// =============================================================================

func TestIsPublicAuthEndpoint_OAuthPaths(t *testing.T) {
	oauthPaths := []string{
		"/api/v1/auth/oauth/google",
		"/api/v1/auth/oauth/github",
		"/api/v1/auth/oauth/facebook",
		"/api/v1/auth/oauth/microsoft",
		"/api/v1/auth/oauth/apple",
	}

	for _, path := range oauthPaths {
		t.Run(path, func(t *testing.T) {
			assert.True(t, isPublicAuthEndpoint(path), "OAuth path should be public: %s", path)
		})
	}
}

func TestIsPublicAuthEndpoint_AllPublicPaths(t *testing.T) {
	publicPaths := []string{
		"/api/v1/auth/signup",
		"/api/v1/auth/signin",
		"/api/v1/auth/signout",
		"/api/v1/auth/refresh",
		"/api/v1/auth/password/reset",
		"/api/v1/auth/password/reset/confirm",
		"/api/v1/auth/password/reset/verify",
		"/api/v1/auth/magic-link",
		"/api/v1/auth/magic-link/verify",
		"/api/v1/auth/magiclink",
		"/api/v1/auth/magiclink/verify",
		"/api/v1/auth/verify-email",
		"/api/v1/auth/oauth",
		"/api/v1/auth/2fa/verify",
		"/api/v1/admin/setup",
		"/api/v1/admin/setup/status",
		"/api/v1/admin/login",
		"/api/v1/admin/login/2fa",
		"/api/v1/admin/2fa/verify",
		"/api/v1/admin/refresh",
		"/dashboard/auth/signup",
		"/dashboard/auth/login",
		"/dashboard/auth/2fa/verify",
	}

	for _, path := range publicPaths {
		t.Run(path, func(t *testing.T) {
			assert.True(t, isPublicAuthEndpoint(path), "Should be public: %s", path)
		})
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkGenerateCSRFToken(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = generateCSRFToken(32)
	}
}

func BenchmarkIsPublicAuthEndpoint_Public(b *testing.B) {
	path := "/api/v1/auth/signin"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = isPublicAuthEndpoint(path)
	}
}

func BenchmarkIsPublicAuthEndpoint_Private(b *testing.B) {
	path := "/api/v1/users/data"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = isPublicAuthEndpoint(path)
	}
}

func BenchmarkCSRF_SafeMethod(b *testing.B) {
	app := fiber.New()
	app.Use(CSRF())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		_, _ = app.Test(req)
	}
}

func BenchmarkCSRF_BearerSkip(b *testing.B) {
	app := fiber.New()
	app.Use(CSRF())
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/test", nil)
		req.Header.Set("Authorization", "Bearer token_value")
		_, _ = app.Test(req)
	}
}

func BenchmarkCSRF_ValidToken(b *testing.B) {
	storage := memory.New()
	token := "benchmark-token-12345678901234567890"
	_ = storage.Set(token, []byte("1"), 24*time.Hour)

	app := fiber.New()
	app.Use(CSRF(CSRFConfig{
		TokenLength: 32,
		TokenLookup: "header:X-CSRF-Token",
		CookieName:  "csrf_token",
		Storage:     storage,
		Expiration:  time.Hour,
	}))
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/test", nil)
		req.Header.Set("Cookie", "csrf_token="+token)
		req.Header.Set("X-CSRF-Token", token)
		_, _ = app.Test(req)
	}
}
