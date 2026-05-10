package middleware

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/storage/memory/v2"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/observability"
)

var rateLimiterMetrics *observability.Metrics

var (
	serviceRoleLimiter     fiber.Handler
	serviceRoleLimiterCfg  RateLimiterConfig
	serviceRoleLimiterMu   sync.Mutex
)

// SetRateLimiterMetrics sets the metrics instance for rate limiter
func SetRateLimiterMetrics(m *observability.Metrics) {
	rateLimiterMetrics = m
}

// RateLimiterConfig holds configuration for rate limiting
type RateLimiterConfig struct {
	Name       string                 // Name of the rate limiter (for metrics)
	Max        int                    // Maximum number of requests
	Expiration time.Duration          // Time window for the rate limit
	KeyFunc    func(fiber.Ctx) string // Function to generate the key for rate limiting
	Message    string                 // Custom error message
	Storage    fiber.Storage          // Optional shared storage (if nil, creates new storage)
}

// NewRateLimiter creates a new rate limiter middleware with custom configuration.
//
// IMPORTANT: This middleware uses Fiber's native in-memory storage for rate limiting.
// Rate limiting is per-instance only. In multi-instance deployments, each instance
// maintains its own rate limit counters independently.
func NewRateLimiter(config RateLimiterConfig) fiber.Handler {
	// Use provided storage or create new one with test mode detection
	var storage fiber.Storage
	if config.Storage != nil {
		storage = config.Storage
	} else {
		// In test mode, use a very long GC interval to effectively disable GC
		// This prevents the GC goroutine from running frequently in tests
		gcInterval := 10 * time.Minute
		if os.Getenv("FLUXBASE_TEST_MODE") == "1" {
			// Set GC interval to 24 hours to effectively disable it during tests
			gcInterval = 24 * time.Hour
		}
		storage = memory.New(memory.Config{
			GCInterval: gcInterval,
		})
	}

	// Default key function uses IP address
	if config.KeyFunc == nil {
		config.KeyFunc = func(c fiber.Ctx) string {
			return c.IP()
		}
	}

	// Default error message
	if config.Message == "" {
		config.Message = fmt.Sprintf("Rate limit exceeded. Maximum %d requests per %s allowed.",
			config.Max, config.Expiration.String())
	}

	// Capture name for closure
	limiterName := config.Name
	if limiterName == "" {
		limiterName = "default"
	}

	return limiter.New(limiter.Config{
		Max:          config.Max,
		Expiration:   config.Expiration,
		KeyGenerator: config.KeyFunc,
		LimitReached: func(c fiber.Ctx) error {
			// Record rate limit hit metric
			if rateLimiterMetrics != nil {
				rateLimiterMetrics.RecordRateLimitHit(limiterName, c.IP())
			}

			retryAfter := int(config.Expiration.Seconds())
			c.Set("Retry-After", fmt.Sprintf("%d", retryAfter))
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"code":        "RATE_LIMIT_EXCEEDED",
				"error":       "Rate limit exceeded",
				"message":     config.Message,
				"retry_after": retryAfter,
			})
		},
		Storage: storage,
	})
}

// AuthLoginLimiter limits login attempts per IP
func AuthLoginLimiter(storage ...fiber.Storage) fiber.Handler {
	return AuthLoginLimiterWithConfig(10, 15*time.Minute, storage...)
}

// AuthLoginLimiterWithConfig creates an auth login rate limiter with custom limits
func AuthLoginLimiterWithConfig(max int, expiration time.Duration, storage ...fiber.Storage) fiber.Handler {
	cfg := RateLimiterConfig{
		Name:       "auth_login",
		Max:        max,
		Expiration: expiration,
		KeyFunc: func(c fiber.Ctx) string {
			return "login:" + c.IP()
		},
		Message: fmt.Sprintf("Too many login attempts. Please try again in %d minutes.", int(expiration.Minutes())),
	}
	if len(storage) > 0 && storage[0] != nil {
		cfg.Storage = storage[0]
	}
	return NewRateLimiter(cfg)
}

// AuthSignupLimiter limits signup attempts per IP
func AuthSignupLimiter(storage ...fiber.Storage) fiber.Handler {
	return AuthSignupLimiterWithConfig(10, 15*time.Minute, storage...)
}

// AuthSignupLimiterWithConfig creates an auth signup rate limiter with custom limits
func AuthSignupLimiterWithConfig(max int, expiration time.Duration, storage ...fiber.Storage) fiber.Handler {
	cfg := RateLimiterConfig{
		Name:       "auth_signup",
		Max:        max,
		Expiration: expiration,
		KeyFunc: func(c fiber.Ctx) string {
			return "signup:" + c.IP()
		},
		Message: fmt.Sprintf("Too many signup attempts. Please try again in %d minutes.", int(expiration.Minutes())),
	}
	if len(storage) > 0 && storage[0] != nil {
		cfg.Storage = storage[0]
	}
	return NewRateLimiter(cfg)
}

// AuthPasswordResetLimiter limits password reset requests per IP
func AuthPasswordResetLimiter() fiber.Handler {
	return AuthPasswordResetLimiterWithConfig(5, 15*time.Minute)
}

// AuthPasswordResetLimiterWithConfig creates an auth password reset rate limiter with custom limits
func AuthPasswordResetLimiterWithConfig(max int, expiration time.Duration, storage ...fiber.Storage) fiber.Handler {
	cfg := RateLimiterConfig{
		Name:       "auth_password_reset",
		Max:        max,
		Expiration: expiration,
		KeyFunc: func(c fiber.Ctx) string {
			return "password_reset:" + c.IP()
		},
		Message: fmt.Sprintf("Too many password reset requests. Please try again in %d minutes.", int(expiration.Minutes())),
	}
	if len(storage) > 0 && storage[0] != nil {
		cfg.Storage = storage[0]
	}
	return NewRateLimiter(cfg)
}

// Auth2FALimiter limits 2FA verification attempts per IP
// Strict rate limiting to prevent brute-force attacks on 6-digit TOTP codes
func Auth2FALimiter(storage ...fiber.Storage) fiber.Handler {
	return Auth2FALimiterWithConfig(5, 5*time.Minute, storage...)
}

// Auth2FALimiterWithConfig creates an auth 2FA rate limiter with custom limits
func Auth2FALimiterWithConfig(max int, expiration time.Duration, storage ...fiber.Storage) fiber.Handler {
	cfg := RateLimiterConfig{
		Name:       "auth_2fa",
		Max:        max,
		Expiration: expiration,
		KeyFunc: func(c fiber.Ctx) string {
			return "2fa:" + c.IP()
		},
		Message: fmt.Sprintf("Too many 2FA verification attempts. Please try again in %d minutes.", int(expiration.Minutes())),
	}
	if len(storage) > 0 && storage[0] != nil {
		cfg.Storage = storage[0]
	}
	return NewRateLimiter(cfg)
}

// AuthRefreshLimiter limits token refresh attempts per token
func AuthRefreshLimiter(storage ...fiber.Storage) fiber.Handler {
	return AuthRefreshLimiterWithConfig(10, 1*time.Minute, storage...)
}

// AuthRefreshLimiterWithConfig creates an auth token refresh rate limiter with custom limits
func AuthRefreshLimiterWithConfig(max int, expiration time.Duration, storage ...fiber.Storage) fiber.Handler {
	cfg := RateLimiterConfig{
		Name:       "auth_refresh",
		Max:        max,
		Expiration: expiration,
		KeyFunc: func(c fiber.Ctx) string {
			// Try to get token from request body
			var req struct {
				RefreshToken string `json:"refresh_token"`
			}
			if err := c.Bind().Body(&req); err == nil && req.RefreshToken != "" {
				prefix := req.RefreshToken
				if len(prefix) > 20 {
					prefix = prefix[:20]
				}
				return "refresh:" + prefix
			}
			// Fallback to IP if no token found
			return "refresh:" + c.IP()
		},
		Message: fmt.Sprintf("Too many token refresh attempts. Please wait %d minute(s).", int(expiration.Minutes())),
	}
	if len(storage) > 0 && storage[0] != nil {
		cfg.Storage = storage[0]
	}
	return NewRateLimiter(cfg)
}

// AuthMagicLinkLimiter limits magic link requests per IP
func AuthMagicLinkLimiter(storage ...fiber.Storage) fiber.Handler {
	return AuthMagicLinkLimiterWithConfig(5, 15*time.Minute, storage...)
}

// AuthMagicLinkLimiterWithConfig creates an auth magic link rate limiter with custom limits
func AuthMagicLinkLimiterWithConfig(max int, expiration time.Duration, storage ...fiber.Storage) fiber.Handler {
	cfg := RateLimiterConfig{
		Name:       "auth_magic_link",
		Max:        max,
		Expiration: expiration,
		KeyFunc: func(c fiber.Ctx) string {
			return "magiclink:" + c.IP()
		},
		Message: fmt.Sprintf("Too many magic link requests. Please try again in %d minutes.", int(expiration.Minutes())),
	}
	if len(storage) > 0 && storage[0] != nil {
		cfg.Storage = storage[0]
	}
	return NewRateLimiter(cfg)
}

// AuthEmailBasedLimiter limits requests per email address (for sensitive operations)
func AuthEmailBasedLimiter(prefix string, max int, expiration time.Duration) fiber.Handler {
	return NewRateLimiter(RateLimiterConfig{
		Max:        max,
		Expiration: expiration,
		KeyFunc: func(c fiber.Ctx) string {
			var req struct {
				Email string `json:"email"`
			}
			if err := c.Bind().Body(&req); err == nil && req.Email != "" {
				return prefix + ":" + req.Email
			}
			// Fallback to IP if no email found
			return prefix + ":" + c.IP()
		},
		Message: "Too many requests. Please try again later.",
	})
}

// GlobalAPILimiter is a general rate limiter for all API endpoints
// Uses per-IP rate limiting by default, can use per-user rate limiting if enabled
func GlobalAPILimiter(storage ...fiber.Storage) fiber.Handler {
	// Build config with optional shared storage
	cfg := RateLimiterConfig{
		Name:       "global",
		Max:        100,
		Expiration: 1 * time.Minute,
		KeyFunc: func(c fiber.Ctx) string {
			// Try to get user ID from locals (set by auth middleware)
			userID := c.Locals("user_id")
			if userID != nil {
				if uid, ok := userID.(string); ok && uid != "" && uid != "anonymous" {
					return "global_user:" + uid
				}
			}
			// Fallback to IP for anonymous users or when user ID not available
			return "global_ip:" + c.IP()
		},
		Message: "API rate limit exceeded. Maximum 100 requests per minute allowed.",
	}

	// Add shared storage if provided
	if len(storage) > 0 && storage[0] != nil {
		cfg.Storage = storage[0]
	}

	return NewRateLimiter(cfg)
}

// DynamicGlobalAPILimiter creates a rate limiter that respects the dynamic setting
// It checks the settings cache on each request, allowing real-time toggling of rate limiting
// without server restart
// Admin users (admin, instance_admin) are exempt from rate limiting
// service_role users can be rate-limited if service_role_rate_limit > 0
func DynamicGlobalAPILimiter(settingsCache *auth.SettingsCache, storage ...fiber.Storage) fiber.Handler {
	// Create the actual rate limiter once with optional shared storage
	rateLimiter := GlobalAPILimiter(storage...)

	return func(c fiber.Ctx) error {
		// Only apply rate limiting to API endpoints
		// Skip for static files, health checks, admin UI, favicon, etc.
		if !strings.HasPrefix(c.Path(), "/api/") {
			return c.Next()
		}

		// Only trust roles that have been validated by auth middleware
		// We do NOT extract roles from unvalidated JWT tokens to prevent bypass attacks
		role := c.Locals("user_role")
		if role != nil {
			roleStr, ok := role.(string)
			if ok && (roleStr == "admin" || roleStr == "instance_admin" || roleStr == "tenant_service") {
				log.Debug().
					Str("role", roleStr).
					Str("path", c.Path()).
					Str("method", c.Method()).
					Msg("Rate limiter: bypassing for admin user")
				return c.Next()
			}

			// For service_role, check if rate limiting is configured
			if ok && roleStr == "service_role" {
				ctx := c.RequestCtx()
				serviceRoleRateLimit := settingsCache.GetInt(ctx, "app.security.service_role_rate_limit", 0)
				if serviceRoleRateLimit <= 0 {
					return c.Next()
				}
				rateWindow := settingsCache.GetDuration(ctx, "app.security.service_role_rate_window", 1*time.Minute)
				cfg := RateLimiterConfig{
					Name:       "service_role",
					Max:        serviceRoleRateLimit,
					Expiration: rateWindow,
					KeyFunc: func(c fiber.Ctx) string {
						return "service_role:" + c.IP()
					},
					Message: fmt.Sprintf("Service role rate limit exceeded. Maximum %d requests per %s allowed.", serviceRoleRateLimit, rateWindow.String()),
				}
				if len(storage) > 0 && storage[0] != nil {
					cfg.Storage = storage[0]
				}

				serviceRoleLimiterMu.Lock()
				if serviceRoleLimiter == nil || serviceRoleLimiterCfg.Max != cfg.Max || serviceRoleLimiterCfg.Expiration != cfg.Expiration {
					serviceRoleLimiter = NewRateLimiter(cfg)
					serviceRoleLimiterCfg = cfg
				}
				limiter := serviceRoleLimiter
				serviceRoleLimiterMu.Unlock()

				return limiter(c)
			}
		}

		// Check if rate limiting is enabled via settings cache
		ctx := c.RequestCtx()
		isEnabled := settingsCache.GetBool(ctx, "app.security.enable_global_rate_limit", false)

		if !isEnabled {
			log.Debug().Msg("Rate limiter: disabled via settings, skipping")
			return c.Next() // Skip rate limiting
		}

		log.Debug().
			Str("path", c.Path()).
			Str("method", c.Method()).
			Str("ip", c.IP()).
			Msg("Rate limiter: applying global rate limit")
		return rateLimiter(c)
	}
}

// AuthenticatedUserLimiter limits requests per authenticated user (higher limits than IP-based)
// Should be applied AFTER authentication middleware
func AuthenticatedUserLimiter() fiber.Handler {
	return NewRateLimiter(RateLimiterConfig{
		Max:        500, // Higher limit for authenticated users
		Expiration: 1 * time.Minute,
		KeyFunc: func(c fiber.Ctx) string {
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

// ClientKeyLimiter limits requests per client key with configurable limits
// Should be applied AFTER client key authentication middleware
func ClientKeyLimiter(maxRequests int, duration time.Duration) fiber.Handler {
	return NewRateLimiter(RateLimiterConfig{
		Max:        maxRequests,
		Expiration: duration,
		KeyFunc: func(c fiber.Ctx) string {
			// Try to get client key ID from locals (set by client key auth middleware)
			keyID := c.Locals("client_key_id")
			if keyID != nil {
				if kid, ok := keyID.(string); ok && kid != "" {
					return "clientkey:" + kid
				}
			}
			// Fallback to IP if no client key ID
			return "clientkey:" + c.IP()
		},
		Message: fmt.Sprintf("Client key rate limit exceeded. Maximum %d requests per %s allowed.", maxRequests, duration.String()),
	})
}

// DefaultClientKeyLimiter returns a client key limiter with default limits (1000 req/min)
func DefaultClientKeyLimiter() fiber.Handler {
	return ClientKeyLimiter(1000, 1*time.Minute)
}

// PerUserOrIPLimiter implements tiered rate limiting:
// - Authenticated users: higher limit
// - Client keys: configurable limit
// - Anonymous (IP): lower limit
func PerUserOrIPLimiter(anonMax, userMax, clientKeyMax int, duration time.Duration) fiber.Handler {
	return NewRateLimiter(RateLimiterConfig{
		Max:        anonMax, // Base max (will be adjusted by key function)
		Expiration: duration,
		KeyFunc: func(c fiber.Ctx) string {
			// Priority 1: Check for client key
			clientKeyID := c.Locals("client_key_id")
			if clientKeyID != nil {
				if kid, ok := clientKeyID.(string); ok && kid != "" {
					// Use client key specific limit
					return fmt.Sprintf("clientkey:%s:%d", kid, clientKeyMax)
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
func AdminSetupLimiter(storage ...fiber.Storage) fiber.Handler {
	return AdminSetupLimiterWithConfig(5, 15*time.Minute, storage...)
}

// AdminSetupLimiterWithConfig creates an admin setup rate limiter with custom limits
func AdminSetupLimiterWithConfig(max int, expiration time.Duration, storage ...fiber.Storage) fiber.Handler {
	cfg := RateLimiterConfig{
		Max:        max,
		Expiration: expiration,
		KeyFunc: func(c fiber.Ctx) string {
			return "admin_setup:" + c.IP()
		},
		Message: fmt.Sprintf("Too many admin setup attempts. Please try again in %d minutes.", int(expiration.Minutes())),
	}
	if len(storage) > 0 && storage[0] != nil {
		cfg.Storage = storage[0]
	}
	return NewRateLimiter(cfg)
}

// AdminLoginLimiter limits admin login attempts per IP
// Max is set to 4 to trigger rate limiting before account lockout (which happens at 5 failed attempts)
func AdminLoginLimiter(storage ...fiber.Storage) fiber.Handler {
	return AdminLoginLimiterWithConfig(4, 1*time.Minute, storage...)
}

// AdminLoginLimiterWithConfig creates an admin login rate limiter with custom limits
func AdminLoginLimiterWithConfig(max int, expiration time.Duration, storage ...fiber.Storage) fiber.Handler {
	cfg := RateLimiterConfig{
		Max:        max,
		Expiration: expiration,
		KeyFunc: func(c fiber.Ctx) string {
			return "admin_login:" + c.IP()
		},
		Message: fmt.Sprintf("Too many admin login attempts. Please try again in %d minutes.", int(expiration.Minutes())),
	}
	if len(storage) > 0 && storage[0] != nil {
		cfg.Storage = storage[0]
	}
	return NewRateLimiter(cfg)
}

// GitHubWebhookLimiter limits GitHub webhook requests per IP and repository
// Prevents abuse of the webhook endpoint for branch creation/deletion
func GitHubWebhookLimiter(storage ...fiber.Storage) fiber.Handler {
	cfg := RateLimiterConfig{
		Max:        30,              // 30 requests
		Expiration: 1 * time.Minute, // per minute per IP
		KeyFunc: func(c fiber.Ctx) string {
			return "github_webhook:" + c.IP()
		},
		Message: "GitHub webhook rate limit exceeded. Maximum 30 requests per minute allowed.",
	}
	if len(storage) > 0 && storage[0] != nil {
		cfg.Storage = storage[0]
	}
	return NewRateLimiter(cfg)
}

// MigrationAPILimiter limits migrations API requests per service key
// Very strict rate limiting due to powerful DDL operations
// Should be applied AFTER service key authentication middleware
// NOTE: service_role JWT tokens bypass rate limiting entirely (trusted keys)
// Service keys (sk_*) use per-key configurable rate limits from the database
//
// Deprecated: Use MigrationAPILimiterWithConfig for H-2 security fix
func MigrationAPILimiter(storage ...fiber.Storage) fiber.Handler {
	return MigrationAPILimiterWithConfig(0, 0, storage...)
}

// MigrationAPILimiterWithConfig creates a migrations API rate limiter with custom limits
// H-2: Enforces rate limiting for service_role tokens when configured
// serviceRoleRateLimit: Max requests for service_role tokens (0 = unlimited, for backward compatibility)
// serviceRoleRateWindow: Time window for service_role rate limiting
func MigrationAPILimiterWithConfig(serviceRoleRateLimit int, serviceRoleRateWindow time.Duration, storage ...fiber.Storage) fiber.Handler {
	// Default rate limiter for service keys without custom limits
	defaultRateLimiterCfg := RateLimiterConfig{
		Max:        10,            // 10 requests
		Expiration: 1 * time.Hour, // per hour
		KeyFunc: func(c fiber.Ctx) string {
			keyID := c.Locals("service_key_id")
			if keyID != nil {
				if kid, ok := keyID.(string); ok && kid != "" {
					return "migration_key:" + kid
				}
			}
			return "migration_ip:" + c.IP()
		},
		Message: "Migrations API rate limit exceeded. Maximum 10 requests per hour allowed.",
	}
	// Add shared storage if provided
	if len(storage) > 0 && storage[0] != nil {
		defaultRateLimiterCfg.Storage = storage[0]
	}
	defaultRateLimiter := NewRateLimiter(defaultRateLimiterCfg)

	// H-2: Service role rate limiter (if configured)
	var serviceRoleLimiter fiber.Handler
	if serviceRoleRateLimit > 0 && serviceRoleRateWindow > 0 {
		serviceRoleLimiterCfg := RateLimiterConfig{
			Max:        serviceRoleRateLimit,
			Expiration: serviceRoleRateWindow,
			KeyFunc: func(c fiber.Ctx) string {
				// Rate limit by JWT ID (jti) for service_role tokens
				if jti := c.Locals("jti"); jti != nil {
					if jtiStr, ok := jti.(string); ok && jtiStr != "" {
						return "service_role:" + jtiStr
					}
				}
				// Fallback to service key ID
				if keyID := c.Locals("service_key_id"); keyID != nil {
					if kid, ok := keyID.(string); ok && kid != "" {
						return "service_role_key:" + kid
					}
				}
				return "service_role_ip:" + c.IP()
			},
			Message: fmt.Sprintf("Service role rate limit exceeded. Maximum %d requests per %v allowed.", serviceRoleRateLimit, serviceRoleRateWindow),
		}
		// Add shared storage if provided
		if len(storage) > 0 && storage[0] != nil {
			serviceRoleLimiterCfg.Storage = storage[0]
		}
		serviceRoleLimiter = NewRateLimiter(serviceRoleLimiterCfg)
	}

	// Cache for per-key rate limiters (keyed by key ID + limit config)
	perKeyLimiters := make(map[string]fiber.Handler)
	perKeyLimiterTimes := make(map[string]time.Time)
	var limiterMu sync.RWMutex

	// Periodically evict stale per-key limiters to prevent unbounded growth
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			limiterMu.Lock()
			now := time.Now()
			for k, t := range perKeyLimiterTimes {
				if now.Sub(t) > time.Hour {
					delete(perKeyLimiters, k)
					delete(perKeyLimiterTimes, k)
				}
			}
			limiterMu.Unlock()
		}
	}()

	return func(c fiber.Ctx) error {
		role := c.Locals("user_role")
		if role == "service_role" {
			// H-2: Apply rate limiting to service_role tokens if configured
			if serviceRoleLimiter != nil {
				return serviceRoleLimiter(c)
			}
			// Backward compatibility: bypass if no rate limit configured
			return c.Next()
		}

		// Check for per-key rate limits from service key context
		rateLimitPerHour := c.Locals("service_key_rate_limit_per_hour")

		// If no custom rate limit is set (nil), use the default
		if rateLimitPerHour == nil {
			return defaultRateLimiter(c)
		}

		// Get the rate limit value
		limitPtr, ok := rateLimitPerHour.(*int)
		if !ok || limitPtr == nil {
			return defaultRateLimiter(c)
		}
		limit := *limitPtr

		// Get key ID for cache lookup
		keyID := c.Locals("service_key_id")
		keyIDStr := ""
		if keyID != nil {
			if kid, ok := keyID.(string); ok {
				keyIDStr = kid
			}
		}

		// Create cache key based on key ID and limit
		cacheKey := fmt.Sprintf("%s:%d", keyIDStr, limit)

		// Try to get cached limiter
		limiterMu.RLock()
		limiter, exists := perKeyLimiters[cacheKey]
		limiterMu.RUnlock()

		if !exists {
			// Create new limiter for this key's rate limit with shared storage if available
			limiterCfg := RateLimiterConfig{
				Max:        limit,
				Expiration: 1 * time.Hour,
				KeyFunc: func(c fiber.Ctx) string {
					keyID := c.Locals("service_key_id")
					if keyID != nil {
						if kid, ok := keyID.(string); ok && kid != "" {
							return "migration_key:" + kid
						}
					}
					return "migration_ip:" + c.IP()
				},
				Message: fmt.Sprintf("Migrations API rate limit exceeded. Maximum %d requests per hour allowed.", limit),
			}
			// Add shared storage if provided
			if len(storage) > 0 && storage[0] != nil {
				limiterCfg.Storage = storage[0]
			}
			limiter = NewRateLimiter(limiterCfg)

			// Cache the limiter
			limiterMu.Lock()
			perKeyLimiters[cacheKey] = limiter
			perKeyLimiterTimes[cacheKey] = time.Now()
			limiterMu.Unlock()
		}

		return limiter(c)
	}
}

// StorageUploadLimiter limits file upload requests per user/IP
// Prevents abuse of storage upload endpoints including streaming uploads
func StorageUploadLimiter(storage ...fiber.Storage) fiber.Handler {
	return StorageUploadLimiterWithConfig(60, 1*time.Minute, storage...)
}

// StorageUploadLimiterWithConfig creates a storage upload rate limiter with custom limits
func StorageUploadLimiterWithConfig(max int, expiration time.Duration, storage ...fiber.Storage) fiber.Handler {
	cfg := RateLimiterConfig{
		Name:       "storage_upload",
		Max:        max,
		Expiration: expiration,
		KeyFunc: func(c fiber.Ctx) string {
			// Try to get user ID from locals (set by auth middleware)
			userID := c.Locals("user_id")
			if userID != nil {
				if uid, ok := userID.(string); ok && uid != "" && uid != "anonymous" {
					return "storage_upload_user:" + uid
				}
			}
			// Fallback to IP for anonymous users
			return "storage_upload_ip:" + c.IP()
		},
		Message: fmt.Sprintf("Storage upload rate limit exceeded. Maximum %d requests per %s allowed.", max, expiration.String()),
	}
	if len(storage) > 0 && storage[0] != nil {
		cfg.Storage = storage[0]
	}
	return NewRateLimiter(cfg)
}

// fiber:context-methods migrated
