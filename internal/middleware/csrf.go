package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/storage/memory/v2"
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
	Storage fiber.Storage
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
		Storage: memory.New(memory.Config{
			GCInterval: 10 * time.Minute,
		}),
	}
}

// CSRF returns a middleware that protects against Cross-Site Request Forgery attacks
func CSRF(config ...CSRFConfig) fiber.Handler {
	// Use default config if none provided
	cfg := DefaultCSRFConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	// Initialize storage if not provided
	if cfg.Storage == nil {
		cfg.Storage = memory.New(memory.Config{
			GCInterval: 10 * time.Minute,
		})
	}

	return func(c *fiber.Ctx) error {
		// Skip CSRF for safe methods (GET, HEAD, OPTIONS)
		method := c.Method()
		if method == fiber.MethodGet || method == fiber.MethodHead || method == fiber.MethodOptions {
			return c.Next()
		}

		// Skip CSRF for certain paths (WebSocket, health checks)
		path := c.Path()
		if path == "/realtime" || path == "/health" || path == "/ready" || path == "/metrics" {
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
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error":   "Internal server error",
					"message": "Failed to generate CSRF token",
				})
			}

			// Store token
			if err := cfg.Storage.Set(token, []byte("1"), cfg.Expiration); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error":   "Internal server error",
					"message": "Failed to store CSRF token",
				})
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

			// First request, allow it through
			return c.Next()
		}

		// Validate tokens match
		if cookieToken != requestToken || requestToken == "" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "CSRF token validation failed",
				"message": "Invalid or missing CSRF token. Please refresh the page and try again.",
			})
		}

		// Check if token exists in storage
		_, err := cfg.Storage.Get(cookieToken)
		if err != nil {
			// Token expired or doesn't exist, generate new one
			token, err := generateCSRFToken(cfg.TokenLength)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error":   "Internal server error",
					"message": "Failed to generate CSRF token",
				})
			}

			// Store new token
			if err := cfg.Storage.Set(token, []byte("1"), cfg.Expiration); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error":   "Internal server error",
					"message": "Failed to store CSRF token",
				})
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

			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "CSRF token expired",
				"message": "CSRF token has expired. Please refresh the page and try again.",
			})
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
func GetCSRFToken(c *fiber.Ctx) string {
	return c.Cookies("csrf_token")
}
