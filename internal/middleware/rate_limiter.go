package middleware

import (
	"fmt"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/fluxbase-eu/fluxbase/internal/ratelimit"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/storage/memory/v2"
)

// RateLimiterConfig holds configuration for rate limiting
type RateLimiterConfig struct {
	Max        int                     // Maximum number of requests
	Expiration time.Duration           // Time window for the rate limit
	KeyFunc    func(*fiber.Ctx) string // Function to generate the key for rate limiting
	Message    string                  // Custom error message
	Store      ratelimit.Store         // Optional: custom rate limit store (uses global store if nil)
}

// NewRateLimiter creates a new rate limiter middleware with custom configuration.
// If config.Store is nil, it uses the global rate limit store (configured via scaling.backend).
// For backwards compatibility, it falls back to in-memory storage if no global store is set.
func NewRateLimiter(config RateLimiterConfig) fiber.Handler {
	var storage fiber.Storage

	// Use the configured store, or fall back to global store, or use memory
	if config.Store != nil {
		storage = ratelimit.NewIncrementAdapter(config.Store, config.Expiration)
	} else if ratelimit.GlobalStore != nil {
		storage = ratelimit.NewIncrementAdapter(ratelimit.GlobalStore, config.Expiration)
	} else {
		// Fall back to in-memory storage for backwards compatibility
		storage = memory.New(memory.Config{
			GCInterval: 10 * time.Minute,
		})
	}

	// Default key function uses IP address
	if config.KeyFunc == nil {
		config.KeyFunc = func(c *fiber.Ctx) string {
			return c.IP()
		}
	}

	// Default error message
	if config.Message == "" {
		config.Message = fmt.Sprintf("Rate limit exceeded. Maximum %d requests per %s allowed.",
			config.Max, config.Expiration.String())
	}

	return limiter.New(limiter.Config{
		Max:          config.Max,
		Expiration:   config.Expiration,
		KeyGenerator: config.KeyFunc,
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":       "Rate limit exceeded",
				"message":     config.Message,
				"retry_after": int(config.Expiration.Seconds()),
			})
		},
		Storage: storage,
	})
}

// AuthLoginLimiter limits login attempts per IP
func AuthLoginLimiter() fiber.Handler {
	return NewRateLimiter(RateLimiterConfig{
		Max:        10,
		Expiration: 15 * time.Minute,
		KeyFunc: func(c *fiber.Ctx) string {
			return "login:" + c.IP()
		},
		Message: "Too many login attempts. Please try again in 15 minutes.",
	})
}

// AuthSignupLimiter limits signup attempts per IP
func AuthSignupLimiter() fiber.Handler {
	return NewRateLimiter(RateLimiterConfig{
		Max:        10,
		Expiration: 15 * time.Minute,
		KeyFunc: func(c *fiber.Ctx) string {
			return "signup:" + c.IP()
		},
		Message: "Too many signup attempts. Please try again in 15 minutes.",
	})
}

// AuthPasswordResetLimiter limits password reset requests per IP
func AuthPasswordResetLimiter() fiber.Handler {
	return NewRateLimiter(RateLimiterConfig{
		Max:        5,
		Expiration: 15 * time.Minute,
		KeyFunc: func(c *fiber.Ctx) string {
			return "password_reset:" + c.IP()
		},
		Message: "Too many password reset requests. Please try again in 15 minutes.",
	})
}

// Auth2FALimiter limits 2FA verification attempts per IP
// Strict rate limiting to prevent brute-force attacks on 6-digit TOTP codes
func Auth2FALimiter() fiber.Handler {
	return NewRateLimiter(RateLimiterConfig{
		Max:        5,
		Expiration: 5 * time.Minute,
		KeyFunc: func(c *fiber.Ctx) string {
			return "2fa:" + c.IP()
		},
		Message: "Too many 2FA verification attempts. Please try again in 5 minutes.",
	})
}

// AuthRefreshLimiter limits token refresh attempts per token
func AuthRefreshLimiter() fiber.Handler {
	return NewRateLimiter(RateLimiterConfig{
		Max:        10,
		Expiration: 1 * time.Minute,
		KeyFunc: func(c *fiber.Ctx) string {
			// Try to get token from request body
			var req struct {
				RefreshToken string `json:"refresh_token"`
			}
			if err := c.BodyParser(&req); err == nil && req.RefreshToken != "" {
				return "refresh:" + req.RefreshToken[:20] // Use first 20 chars as key
			}
			// Fallback to IP if no token found
			return "refresh:" + c.IP()
		},
		Message: "Too many token refresh attempts. Please wait 1 minute.",
	})
}

// AuthMagicLinkLimiter limits magic link requests per IP
func AuthMagicLinkLimiter() fiber.Handler {
	return NewRateLimiter(RateLimiterConfig{
		Max:        5,
		Expiration: 15 * time.Minute,
		KeyFunc: func(c *fiber.Ctx) string {
			return "magiclink:" + c.IP()
		},
		Message: "Too many magic link requests. Please try again in 15 minutes.",
	})
}

// AuthEmailBasedLimiter limits requests per email address (for sensitive operations)
func AuthEmailBasedLimiter(prefix string, max int, expiration time.Duration) fiber.Handler {
	return NewRateLimiter(RateLimiterConfig{
		Max:        max,
		Expiration: expiration,
		KeyFunc: func(c *fiber.Ctx) string {
			var req struct {
				Email string `json:"email"`
			}
			if err := c.BodyParser(&req); err == nil && req.Email != "" {
				return prefix + ":" + req.Email
			}
			// Fallback to IP if no email found
			return prefix + ":" + c.IP()
		},
		Message: "Too many requests. Please try again later.",
	})
}

// GlobalAPILimiter is a general rate limiter for all API endpoints
func GlobalAPILimiter() fiber.Handler {
	return NewRateLimiter(RateLimiterConfig{
		Max:        100,
		Expiration: 1 * time.Minute,
		KeyFunc: func(c *fiber.Ctx) string {
			return "global:" + c.IP()
		},
		Message: "API rate limit exceeded. Maximum 100 requests per minute allowed.",
	})
}

// DynamicGlobalAPILimiter creates a rate limiter that respects the dynamic setting
// It checks the settings cache on each request, allowing real-time toggling of rate limiting
// without server restart
func DynamicGlobalAPILimiter(settingsCache *auth.SettingsCache) fiber.Handler {
	// Create the actual rate limiter once
	rateLimiter := GlobalAPILimiter()

	return func(c *fiber.Ctx) error {
		// Check if rate limiting is enabled via settings cache
		ctx := c.Context()
		isEnabled := settingsCache.GetBool(ctx, "app.security.enable_global_rate_limit", false)

		if !isEnabled {
			return c.Next() // Skip rate limiting
		}

		return rateLimiter(c)
	}
}

// AuthenticatedUserLimiter limits requests per authenticated user (higher limits than IP-based)
// Should be applied AFTER authentication middleware
func AuthenticatedUserLimiter() fiber.Handler {
	return NewRateLimiter(RateLimiterConfig{
		Max:        500, // Higher limit for authenticated users
		Expiration: 1 * time.Minute,
		KeyFunc: func(c *fiber.Ctx) string {
			// Try to get user ID from locals (set by auth middleware)
			userID := c.Locals("user_id")
			if userID != nil {
				if uid, ok := userID.(string); ok && uid != "" {
					return "user:" + uid
				}
			}
			// Fallback to IP if no user ID (shouldn't happen if auth middleware ran)
			return "user:" + c.IP()
		},
		Message: "Rate limit exceeded for your account. Maximum 500 requests per minute allowed.",
	})
}

// APIKeyLimiter limits requests per API key with configurable limits
// Should be applied AFTER API key authentication middleware
func APIKeyLimiter(maxRequests int, duration time.Duration) fiber.Handler {
	return NewRateLimiter(RateLimiterConfig{
		Max:        maxRequests,
		Expiration: duration,
		KeyFunc: func(c *fiber.Ctx) string {
			// Try to get API key ID from locals (set by API key auth middleware)
			keyID := c.Locals("api_key_id")
			if keyID != nil {
				if kid, ok := keyID.(string); ok && kid != "" {
					return "apikey:" + kid
				}
			}
			// Fallback to IP if no API key ID
			return "apikey:" + c.IP()
		},
		Message: fmt.Sprintf("API key rate limit exceeded. Maximum %d requests per %s allowed.", maxRequests, duration.String()),
	})
}

// DefaultAPIKeyLimiter returns an API key limiter with default limits (1000 req/min)
func DefaultAPIKeyLimiter() fiber.Handler {
	return APIKeyLimiter(1000, 1*time.Minute)
}

// PerUserOrIPLimiter implements tiered rate limiting:
// - Authenticated users: higher limit
// - API keys: configurable limit
// - Anonymous (IP): lower limit
func PerUserOrIPLimiter(anonMax, userMax, apiKeyMax int, duration time.Duration) fiber.Handler {
	return NewRateLimiter(RateLimiterConfig{
		Max:        anonMax, // Base max (will be adjusted by key function)
		Expiration: duration,
		KeyFunc: func(c *fiber.Ctx) string {
			// Priority 1: Check for API key
			apiKeyID := c.Locals("api_key_id")
			if apiKeyID != nil {
				if kid, ok := apiKeyID.(string); ok && kid != "" {
					// Use API key specific limit
					return fmt.Sprintf("apikey:%s:%d", kid, apiKeyMax)
				}
			}

			// Priority 2: Check for authenticated user
			userID := c.Locals("user_id")
			if userID != nil {
				if uid, ok := userID.(string); ok && uid != "" {
					// Use user specific limit
					return fmt.Sprintf("user:%s:%d", uid, userMax)
				}
			}

			// Priority 3: Fallback to IP (anonymous)
			return fmt.Sprintf("ip:%s:%d", c.IP(), anonMax)
		},
		Message: "Rate limit exceeded. Please try again later.",
	})
}

// AdminSetupLimiter limits admin setup attempts per IP
// Very strict since this is a one-time operation
func AdminSetupLimiter() fiber.Handler {
	return NewRateLimiter(RateLimiterConfig{
		Max:        5,
		Expiration: 15 * time.Minute,
		KeyFunc: func(c *fiber.Ctx) string {
			return "admin_setup:" + c.IP()
		},
		Message: "Too many admin setup attempts. Please try again in 15 minutes.",
	})
}

// AdminLoginLimiter limits admin login attempts per IP
func AdminLoginLimiter() fiber.Handler {
	return NewRateLimiter(RateLimiterConfig{
		Max:        10,
		Expiration: 1 * time.Minute,
		KeyFunc: func(c *fiber.Ctx) string {
			return "admin_login:" + c.IP()
		},
		Message: "Too many admin login attempts. Please try again in 1 minute.",
	})
}

// MigrationAPILimiter limits migrations API requests per service key
// Very strict rate limiting due to powerful DDL operations
// Should be applied AFTER service key authentication middleware
func MigrationAPILimiter() fiber.Handler {
	return NewRateLimiter(RateLimiterConfig{
		Max:        10,            // 10 requests
		Expiration: 1 * time.Hour, // per hour
		KeyFunc: func(c *fiber.Ctx) string {
			// Rate limit by service key ID (set by auth middleware)
			keyID := c.Locals("service_key_id")
			if keyID != nil {
				if kid, ok := keyID.(string); ok && kid != "" {
					return "migration_key:" + kid
				}
			}
			// Fallback to IP if no service key (shouldn't happen if auth middleware ran)
			return "migration_ip:" + c.IP()
		},
		Message: "Migrations API rate limit exceeded. Maximum 10 requests per hour allowed.",
	})
}
