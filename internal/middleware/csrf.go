package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"os"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/storage/memory/v2"

	"github.com/nimbleflux/fluxbase/internal/errors"
)

// CSRFConfig holds configuration for CSRF protection
type CSRFConfig struct {
	// TokenLength is the length of the CSRF token in bytes
	TokenLength int
	// TokenLookup defines where to find the token (header:X-CSRF-Token or form:_csrf)
	TokenLookup string
	// CookieName is the name of the CSRF cookie
	CookieName string
	// CookieDomain is the domain of the CSRF cookie
	CookieDomain string
	// CookiePath is the path of the CSRF cookie
	CookiePath string
	// CookieSecure marks the cookie as secure (HTTPS only)
	CookieSecure bool
	// CookieHTTPOnly marks the cookie as HTTP only
	CookieHTTPOnly bool
	// CookieSameSite defines the SameSite attribute
	CookieSameSite string
	// Expiration is how long tokens are valid
	Expiration time.Duration
	// Storage is used to store tokens (default: in-memory)
	// If provided, this storage will be used instead of creating a new one
	// This allows sharing a single storage instance across multiple middleware
	Storage fiber.Storage
	// LazyStorage creates storage only if Storage is nil
	// Set to false if you want to force using the provided Storage
	LazyStorage bool
}

// DefaultCSRFConfig returns default CSRF configuration
func DefaultCSRFConfig() CSRFConfig {
	return CSRFConfig{
		TokenLength:    32,
		TokenLookup:    "header:X-CSRF-Token",
		CookieName:     "csrf_token",
		CookiePath:     "/",
		CookieSecure:   false, // Set to true in production with HTTPS
		CookieHTTPOnly: true,
		CookieSameSite: "Strict",
		Expiration:     24 * time.Hour,
		Storage:        nil,  // Will be initialized in CSRF() function unless provided
		LazyStorage:    true, // Create storage if nil
	}
}

// CSRF returns a middleware that protects against Cross-Site Request Forgery attacks
func CSRF(config ...CSRFConfig) fiber.Handler {
	// Use default config if none provided
	cfg := DefaultCSRFConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	// Initialize storage if not provided and LazyStorage is true
	if cfg.Storage == nil && cfg.LazyStorage {
		// In test mode, use a very long GC interval to effectively disable GC
		// This prevents the GC goroutine from running frequently in tests
		gcInterval := 10 * time.Minute
		if os.Getenv("FLUXBASE_TEST_MODE") == "1" {
			// Set GC interval to 24 hours to effectively disable it during tests
			gcInterval = 24 * time.Hour
		}
		cfg.Storage = memory.New(memory.Config{
			GCInterval: gcInterval,
		})
	}

	return func(c fiber.Ctx) error {
		// Skip CSRF for safe methods (GET, HEAD, OPTIONS)
		method := c.Method()
		if method == fiber.MethodGet || method == fiber.MethodHead || method == fiber.MethodOptions {
			return c.Next()
		}

		// Skip CSRF for certain paths (WebSocket, health checks, public auth endpoints)
		path := c.Path()
		if path == "/realtime" || path == "/health" || path == "/ready" || path == "/metrics" {
			return c.Next()
		}

		// Skip CSRF for public auth endpoints that don't require prior authentication
		// These endpoints are designed to be accessed without tokens
		if isPublicAuthEndpoint(path) {
			return c.Next()
		}

		// Skip CSRF for Bearer token or client key authentication
		// CSRF protection is for cookie-based sessions, not token-based auth
		authHeader := c.Get("Authorization")
		clientKey := c.Get("clientkey")
		if (authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer ") || clientKey != "" {
			return c.Next()
		}

		// Get token from cookie
		cookieToken := c.Cookies(cfg.CookieName)

		// Get token from request (header or form)
		var requestToken string
		if cfg.TokenLookup == "header:X-CSRF-Token" {
			requestToken = c.Get("X-CSRF-Token")
		} else {
			requestToken = c.FormValue("_csrf")
		}

		// If no cookie token exists, this is the first request - generate one
		if cookieToken == "" {
			token, err := generateCSRFToken(cfg.TokenLength)
			if err != nil {
				return errors.SendInternalError(c, "Failed to generate CSRF token")
			}

			// Store token
			if err := cfg.Storage.Set(token, []byte("1"), cfg.Expiration); err != nil {
				return errors.SendInternalError(c, "Failed to store CSRF token")
			}

			// Set cookie
			c.Cookie(&fiber.Cookie{
				Name:     cfg.CookieName,
				Value:    token,
				Path:     cfg.CookiePath,
				Domain:   cfg.CookieDomain,
				MaxAge:   int(cfg.Expiration.Seconds()),
				Secure:   cfg.CookieSecure,
				HTTPOnly: cfg.CookieHTTPOnly,
				SameSite: cfg.CookieSameSite,
			})

			// SECURITY FIX: Reject request - user must retry with the new token
			// The cookie is set, so the next request will include it
			// Previously this allowed the first request through, which was a vulnerability
			// (attacker could clear cookie and submit malicious POST)
			return errors.SendErrorWithCode(c, fiber.StatusForbidden, "CSRF token was not present in the request", errors.ErrCodeCsrfTokenRequired)
		}

		// Validate tokens match
		if cookieToken != requestToken || requestToken == "" {
			return errors.SendErrorWithCode(c, fiber.StatusForbidden, "Invalid or missing CSRF token. Please refresh the page and try again.", errors.ErrCodeCsrfTokenValidationFailed)
		}

		// Check if token exists in storage
		_, err := cfg.Storage.Get(cookieToken)
		if err != nil {
			// Token expired or doesn't exist, generate new one
			token, err := generateCSRFToken(cfg.TokenLength)
			if err != nil {
				return errors.SendInternalError(c, "Failed to generate CSRF token")
			}

			// Store new token
			if err := cfg.Storage.Set(token, []byte("1"), cfg.Expiration); err != nil {
				return errors.SendInternalError(c, "Failed to store CSRF token")
			}

			// Set new cookie
			c.Cookie(&fiber.Cookie{
				Name:     cfg.CookieName,
				Value:    token,
				Path:     cfg.CookiePath,
				Domain:   cfg.CookieDomain,
				MaxAge:   int(cfg.Expiration.Seconds()),
				Secure:   cfg.CookieSecure,
				HTTPOnly: cfg.CookieHTTPOnly,
				SameSite: cfg.CookieSameSite,
			})

			return errors.SendErrorWithCode(c, fiber.StatusForbidden, "CSRF token has expired. Please refresh the page and try again.", errors.ErrCodeCsrfTokenExpired)
		}

		// Token is valid, proceed
		return c.Next()
	}
}

// generateCSRFToken generates a random CSRF token
func generateCSRFToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GetCSRFToken is a helper to retrieve the CSRF token for the current request
func GetCSRFToken(c fiber.Ctx) string {
	return c.Cookies("csrf_token")
}

// isPublicAuthEndpoint checks if the path is a public auth endpoint that doesn't require CSRF
// These are endpoints that are designed to work without prior authentication
func isPublicAuthEndpoint(path string) bool {
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

	for _, p := range publicPaths {
		if path == p {
			return true
		}
		// Also match paths that start with OAuth callback paths
		if len(path) > len(p) && path[:len(p)] == p && path[len(p)] == '/' {
			return true
		}
	}

	// Match OAuth callback pattern
	if len(path) > 20 && path[:20] == "/api/v1/auth/oauth/" {
		return true
	}

	return false
}
