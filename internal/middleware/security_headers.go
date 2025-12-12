package middleware

import (
	"github.com/gofiber/fiber/v2"
)

// SecurityHeadersConfig holds configuration for security headers
type SecurityHeadersConfig struct {
	// Content Security Policy
	ContentSecurityPolicy string
	// X-Frame-Options
	XFrameOptions string
	// X-Content-Type-Options
	XContentTypeOptions string
	// X-XSS-Protection
	XXSSProtection string
	// Strict-Transport-Security (HSTS)
	StrictTransportSecurity string
	// Referrer-Policy
	ReferrerPolicy string
	// Permissions-Policy
	PermissionsPolicy string
}

// DefaultSecurityHeadersConfig returns secure default configuration
// This is the STRICT configuration for API endpoints - no unsafe-inline or unsafe-eval
func DefaultSecurityHeadersConfig() SecurityHeadersConfig {
	return SecurityHeadersConfig{
		// CSP: Strict policy for API endpoints - no inline scripts or eval
		// Admin UI has its own relaxed policy via AdminUISecurityHeaders()
		ContentSecurityPolicy: "default-src 'self'; " +
			"script-src 'self'; " + // No unsafe-inline/eval for API
			"style-src 'self'; " +  // No unsafe-inline for API
			"img-src 'self' data: blob:; " +
			"font-src 'self' data:; " +
			"connect-src 'self' ws: wss:; " + // Allow WebSocket connections
			"frame-ancestors 'none'",
		XFrameOptions:           "DENY",
		XContentTypeOptions:     "nosniff",
		XXSSProtection:          "1; mode=block",
		StrictTransportSecurity: "max-age=31536000; includeSubDomains", // 1 year
		ReferrerPolicy:          "strict-origin-when-cross-origin",
		PermissionsPolicy:       "geolocation=(), microphone=(), camera=()",
	}
}

// SecurityHeaders returns a middleware that adds security headers to all responses
func SecurityHeaders(config ...SecurityHeadersConfig) fiber.Handler {
	// Use default config if none provided
	cfg := DefaultSecurityHeadersConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *fiber.Ctx) error {
		// Content Security Policy
		if cfg.ContentSecurityPolicy != "" {
			c.Set("Content-Security-Policy", cfg.ContentSecurityPolicy)
		}

		// X-Frame-Options
		if cfg.XFrameOptions != "" {
			c.Set("X-Frame-Options", cfg.XFrameOptions)
		}

		// X-Content-Type-Options
		if cfg.XContentTypeOptions != "" {
			c.Set("X-Content-Type-Options", cfg.XContentTypeOptions)
		}

		// X-XSS-Protection
		if cfg.XXSSProtection != "" {
			c.Set("X-XSS-Protection", cfg.XXSSProtection)
		}

		// Strict-Transport-Security (only on HTTPS)
		if cfg.StrictTransportSecurity != "" && c.Protocol() == "https" {
			c.Set("Strict-Transport-Security", cfg.StrictTransportSecurity)
		}

		// Referrer-Policy
		if cfg.ReferrerPolicy != "" {
			c.Set("Referrer-Policy", cfg.ReferrerPolicy)
		}

		// Permissions-Policy
		if cfg.PermissionsPolicy != "" {
			c.Set("Permissions-Policy", cfg.PermissionsPolicy)
		}

		// Remove server header to avoid information disclosure
		c.Set("Server", "")

		return c.Next()
	}
}

// AdminUISecurityHeaders returns relaxed security headers for Admin UI
// Admin UI needs 'unsafe-inline' and 'unsafe-eval' for React
// Also allows Google Fonts from googleapis.com and gstatic.com
func AdminUISecurityHeaders() fiber.Handler {
	cfg := SecurityHeadersConfig{
		ContentSecurityPolicy: "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.jsdelivr.net; " +
			"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://cdn.jsdelivr.net; " +
			"img-src 'self' data: blob: https:; " +
			"font-src 'self' data: https://fonts.gstatic.com; " +
			"connect-src 'self' ws: wss: http: https:; " +
			"worker-src 'self' blob:; " +
			"frame-ancestors 'none'",
		XFrameOptions:       "DENY",
		XContentTypeOptions: "nosniff",
		XXSSProtection:      "1; mode=block",
		ReferrerPolicy:      "strict-origin-when-cross-origin",
	}

	return SecurityHeaders(cfg)
}
