package middleware

import (
	"fmt"
	"time"

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
}

// NewRateLimiter creates a new rate limiter middleware with custom configuration
func NewRateLimiter(config RateLimiterConfig) fiber.Handler {
	// Use in-memory storage (can be replaced with Redis for distributed systems)
	storage := memory.New(memory.Config{
		GCInterval: 10 * time.Minute,
	})

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
		Max:        5,
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
		Max:        3,
		Expiration: 1 * time.Hour,
		KeyFunc: func(c *fiber.Ctx) string {
			return "signup:" + c.IP()
		},
		Message: "Too many signup attempts. Please try again in 1 hour.",
	})
}

// AuthPasswordResetLimiter limits password reset requests per IP
func AuthPasswordResetLimiter() fiber.Handler {
	return NewRateLimiter(RateLimiterConfig{
		Max:        3,
		Expiration: 1 * time.Hour,
		KeyFunc: func(c *fiber.Ctx) string {
			return "password_reset:" + c.IP()
		},
		Message: "Too many password reset requests. Please try again in 1 hour.",
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
		Max:        3,
		Expiration: 1 * time.Hour,
		KeyFunc: func(c *fiber.Ctx) string {
			return "magiclink:" + c.IP()
		},
		Message: "Too many magic link requests. Please try again in 1 hour.",
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
