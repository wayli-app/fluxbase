package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultSecurityHeadersConfig(t *testing.T) {
	cfg := DefaultSecurityHeadersConfig()

	assert.Contains(t, cfg.ContentSecurityPolicy, "default-src 'self'")
	assert.Contains(t, cfg.ContentSecurityPolicy, "frame-ancestors 'none'")
	assert.Equal(t, "DENY", cfg.XFrameOptions)
	assert.Equal(t, "nosniff", cfg.XContentTypeOptions)
	assert.Equal(t, "1; mode=block", cfg.XXSSProtection)
	assert.Contains(t, cfg.StrictTransportSecurity, "max-age=31536000")
	assert.Equal(t, "strict-origin-when-cross-origin", cfg.ReferrerPolicy)
	assert.Contains(t, cfg.PermissionsPolicy, "geolocation=()")
}

func TestSecurityHeaders(t *testing.T) {
	t.Run("applies default headers", func(t *testing.T) {
		app := fiber.New()
		app.Use(SecurityHeaders())
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)

		assert.Equal(t, 200, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Security-Policy"), "default-src 'self'")
		assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
		assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
		assert.Equal(t, "1; mode=block", resp.Header.Get("X-XSS-Protection"))
		assert.Equal(t, "strict-origin-when-cross-origin", resp.Header.Get("Referrer-Policy"))
		assert.Contains(t, resp.Header.Get("Permissions-Policy"), "geolocation=()")
	})

	t.Run("applies custom headers", func(t *testing.T) {
		app := fiber.New()
		app.Use(SecurityHeaders(SecurityHeadersConfig{
			ContentSecurityPolicy: "default-src 'none'",
			XFrameOptions:         "SAMEORIGIN",
			XContentTypeOptions:   "nosniff",
			XXSSProtection:        "0",
			ReferrerPolicy:        "no-referrer",
			PermissionsPolicy:     "camera=()",
		}))
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)

		assert.Equal(t, "default-src 'none'", resp.Header.Get("Content-Security-Policy"))
		assert.Equal(t, "SAMEORIGIN", resp.Header.Get("X-Frame-Options"))
		assert.Equal(t, "0", resp.Header.Get("X-XSS-Protection"))
		assert.Equal(t, "no-referrer", resp.Header.Get("Referrer-Policy"))
		assert.Equal(t, "camera=()", resp.Header.Get("Permissions-Policy"))
	})

	t.Run("skips empty headers", func(t *testing.T) {
		app := fiber.New()
		app.Use(SecurityHeaders(SecurityHeadersConfig{
			ContentSecurityPolicy: "",
			XFrameOptions:         "",
		}))
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)

		assert.Empty(t, resp.Header.Get("Content-Security-Policy"))
		assert.Empty(t, resp.Header.Get("X-Frame-Options"))
	})

	t.Run("removes server header", func(t *testing.T) {
		app := fiber.New()
		app.Use(SecurityHeaders())
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Server header should be empty (removed)
		assert.Empty(t, resp.Header.Get("Server"))
	})

	t.Run("does not add HSTS on non-HTTPS", func(t *testing.T) {
		app := fiber.New()
		app.Use(SecurityHeaders())
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "http://example.com/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)

		// HSTS should not be present on HTTP
		assert.Empty(t, resp.Header.Get("Strict-Transport-Security"))
	})
}

func TestAdminUISecurityHeaders(t *testing.T) {
	app := fiber.New()
	app.Use(AdminUISecurityHeaders())
	app.Get("/admin", func(c *fiber.Ctx) error {
		return c.SendString("Admin UI")
	})

	req := httptest.NewRequest("GET", "/admin", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, 200, resp.StatusCode)

	csp := resp.Header.Get("Content-Security-Policy")
	// Admin UI should have relaxed CSP for React
	assert.Contains(t, csp, "'unsafe-inline'")
	assert.Contains(t, csp, "'unsafe-eval'")
	assert.Contains(t, csp, "fonts.googleapis.com")
	assert.Contains(t, csp, "fonts.gstatic.com")

	// But still have basic security headers
	assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
}

func TestSecurityHeadersConfig_AllFields(t *testing.T) {
	cfg := SecurityHeadersConfig{
		ContentSecurityPolicy:   "test-csp",
		XFrameOptions:           "test-frame",
		XContentTypeOptions:     "test-content",
		XXSSProtection:          "test-xss",
		StrictTransportSecurity: "test-hsts",
		ReferrerPolicy:          "test-referrer",
		PermissionsPolicy:       "test-permissions",
	}

	assert.Equal(t, "test-csp", cfg.ContentSecurityPolicy)
	assert.Equal(t, "test-frame", cfg.XFrameOptions)
	assert.Equal(t, "test-content", cfg.XContentTypeOptions)
	assert.Equal(t, "test-xss", cfg.XXSSProtection)
	assert.Equal(t, "test-hsts", cfg.StrictTransportSecurity)
	assert.Equal(t, "test-referrer", cfg.ReferrerPolicy)
	assert.Equal(t, "test-permissions", cfg.PermissionsPolicy)
}
