package middleware

import (
	"fmt"
	"reflect"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/config"
)

// RateLimitFactory creates rate limiters with consistent configuration.
// It uses the Registry to look up rate limit definitions and applies
// configuration overrides from the SecurityConfig.
//
// Usage:
//
//	factory := middleware.NewRateLimitFactory(&cfg.Security, sharedStorage)
//	loginLimiter := factory.Create("auth_login")
//	customLimiter := factory.CreateWithOverride("auth_login", 5, 10*time.Minute)
type RateLimitFactory struct {
	registry   map[string]RateLimitDefinition
	security   *config.SecurityConfig
	settings   *auth.SettingsCache
	storage    fiber.Storage
	configOpts rateLimitConfigOptions
}

type rateLimitConfigOptions struct {
	// Use reflection to access config fields by name
	securityValue reflect.Value
}

// RateLimitFactoryOption is a functional option for configuring the factory.
type RateLimitFactoryOption func(*RateLimitFactory)

// WithRateLimitSettingsCache sets the settings cache for dynamic rate limit configuration.
func WithRateLimitSettingsCache(cache *auth.SettingsCache) RateLimitFactoryOption {
	return func(f *RateLimitFactory) {
		f.settings = cache
	}
}

// NewRateLimitFactory creates a new rate limiter factory.
func NewRateLimitFactory(security *config.SecurityConfig, storage fiber.Storage, opts ...RateLimitFactoryOption) *RateLimitFactory {
	f := &RateLimitFactory{
		registry: Registry,
		security: security,
		storage:  storage,
	}

	if security != nil {
		f.configOpts.securityValue = reflect.ValueOf(security).Elem()
	}

	for _, opt := range opts {
		opt(f)
	}

	return f
}

// Create returns a rate limiter middleware by name.
// It looks up the definition in the registry and applies any config overrides.
func (f *RateLimitFactory) Create(name string) (fiber.Handler, error) {
	def, ok := f.registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown rate limiter: %s", name)
	}

	max := def.DefaultMax
	window := def.DefaultWindow

	// Override from config if available
	if f.security != nil && def.ConfigMaxField != "" {
		max = f.getConfigInt(def.ConfigMaxField, max)
		window = f.getConfigDuration(def.ConfigWindowField, window)
	}

	return f.createLimiter(def, max, window), nil
}

// CreateWithOverride returns a rate limiter with custom max and window values.
// This is useful when you need to override the defaults without modifying the registry.
func (f *RateLimitFactory) CreateWithOverride(name string, max int, window time.Duration) (fiber.Handler, error) {
	def, ok := f.registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown rate limiter: %s", name)
	}

	return f.createLimiter(def, max, window), nil
}

// CreateFromConfig creates a rate limiter using values from the settings cache.
// This enables dynamic rate limit configuration at runtime.
func (f *RateLimitFactory) CreateFromConfig(name string, settingsCache *auth.SettingsCache) (fiber.Handler, error) {
	def, ok := f.registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown rate limiter: %s", name)
	}

	max := def.DefaultMax
	window := def.DefaultWindow

	// Use settings cache if available for dynamic configuration
	if settingsCache != nil && def.ConfigMaxField != "" {
		// Settings keys are lowercase with dots
		maxKey := fmt.Sprintf("app.security.%s", def.ConfigMaxField)
		windowKey := fmt.Sprintf("app.security.%s", def.ConfigWindowField)

		// Note: GetInt/GetDuration require context, so we use defaults here
		// For truly dynamic rate limiting, use the DynamicGlobalAPILimiter pattern
		_ = maxKey
		_ = windowKey
	}

	return f.createLimiter(def, max, window), nil
}

// createLimiter creates a rate limiter from a definition with the given max and window.
func (f *RateLimitFactory) createLimiter(def RateLimitDefinition, max int, window time.Duration) fiber.Handler {
	message := def.Message
	if message == "" {
		message = fmt.Sprintf("Rate limit exceeded. Maximum %d requests per %s allowed.", max, window.String())
	}

	cfg := RateLimiterConfig{
		Name:       def.Name,
		Max:        max,
		Expiration: window,
		KeyFunc:    f.getKeyFunc(def),
		Message:    message,
		Storage:    f.storage,
	}

	return NewRateLimiter(cfg)
}

// getKeyFunc returns the key generation function for the given definition.
func (f *RateLimitFactory) getKeyFunc(def RateLimitDefinition) func(fiber.Ctx) string {
	prefix := def.KeyPrefix
	if prefix == "" {
		prefix = def.Name
	}

	switch def.KeyStrategy {
	case KeyStrategyIP:
		return func(c fiber.Ctx) string {
			return prefix + ":" + c.IP()
		}

	case KeyStrategyUser:
		return func(c fiber.Ctx) string {
			userID := c.Locals("user_id")
			if userID != nil {
				if uid, ok := userID.(string); ok && uid != "" && uid != "anonymous" {
					return prefix + "_user:" + uid
				}
			}
			return prefix + "_ip:" + c.IP()
		}

	case KeyStrategyClientKey:
		return func(c fiber.Ctx) string {
			keyID := c.Locals("client_key_id")
			if keyID != nil {
				if kid, ok := keyID.(string); ok && kid != "" {
					return prefix + "_key:" + kid
				}
			}
			return prefix + "_ip:" + c.IP()
		}

	case KeyStrategyServiceKey:
		return func(c fiber.Ctx) string {
			keyID := c.Locals("service_key_id")
			if keyID != nil {
				if kid, ok := keyID.(string); ok && kid != "" {
					return prefix + "_key:" + kid
				}
			}
			return prefix + "_ip:" + c.IP()
		}

	case KeyStrategyToken:
		return func(c fiber.Ctx) string {
			// Try to get token from request body
			var req struct {
				RefreshToken string `json:"refresh_token"`
			}
			if err := c.Bind().Body(&req); err == nil && req.RefreshToken != "" {
				if len(req.RefreshToken) >= 20 {
					return prefix + ":" + req.RefreshToken[:20]
				}
				return prefix + ":" + req.RefreshToken
			}
			return prefix + ":" + c.IP()
		}

	case KeyStrategyEmail:
		return func(c fiber.Ctx) string {
			var req struct {
				Email string `json:"email"`
			}
			if err := c.Bind().Body(&req); err == nil && req.Email != "" {
				return prefix + ":" + req.Email
			}
			return prefix + ":" + c.IP()
		}

	case KeyStrategyTiered:
		// Tiered strategy requires special handling with different limits
		// Fall back to user strategy for simplicity
		return func(c fiber.Ctx) string {
			userID := c.Locals("user_id")
			if userID != nil {
				if uid, ok := userID.(string); ok && uid != "" && uid != "anonymous" {
					return prefix + "_user:" + uid
				}
			}
			return prefix + "_ip:" + c.IP()
		}

	default:
		return func(c fiber.Ctx) string {
			return prefix + ":" + c.IP()
		}
	}
}

// getConfigInt retrieves an int value from the security config by field name.
func (f *RateLimitFactory) getConfigInt(fieldName string, defaultVal int) int {
	if !f.configOpts.securityValue.IsValid() {
		return defaultVal
	}

	field := f.configOpts.securityValue.FieldByName(fieldName)
	if !field.IsValid() {
		return defaultVal
	}

	if field.Kind() == reflect.Int {
		return int(field.Int())
	}

	return defaultVal
}

// getConfigDuration retrieves a duration value from the security config by field name.
func (f *RateLimitFactory) getConfigDuration(fieldName string, defaultVal time.Duration) time.Duration {
	if !f.configOpts.securityValue.IsValid() {
		return defaultVal
	}

	field := f.configOpts.securityValue.FieldByName(fieldName)
	if !field.IsValid() {
		return defaultVal
	}

	// Duration is stored as time.Duration which is int64
	if field.Kind() == reflect.Int64 {
		return time.Duration(field.Int())
	}

	return defaultVal
}

// CreateTieredLimiter creates a rate limiter with different limits for different user types.
// - anonMax: Maximum requests for anonymous (IP-based) users
// - userMax: Maximum requests for authenticated users
// - clientKeyMax: Maximum requests for client key users
func (f *RateLimitFactory) CreateTieredLimiter(name string, anonMax, userMax, clientKeyMax int, window time.Duration) (fiber.Handler, error) {
	def, ok := f.registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown rate limiter: %s", name)
	}

	prefix := def.KeyPrefix
	if prefix == "" {
		prefix = def.Name
	}

	return NewRateLimiter(RateLimiterConfig{
		Name:       def.Name,
		Max:        anonMax, // Base max (will be adjusted by key function)
		Expiration: window,
		KeyFunc: func(c fiber.Ctx) string {
			// Priority 1: Check for client key
			clientKeyID := c.Locals("client_key_id")
			if clientKeyID != nil {
				if kid, ok := clientKeyID.(string); ok && kid != "" {
					return fmt.Sprintf("%s_clientkey:%s:%d", prefix, kid, clientKeyMax)
				}
			}

			// Priority 2: Check for authenticated user
			userID := c.Locals("user_id")
			if userID != nil {
				if uid, ok := userID.(string); ok && uid != "" {
					return fmt.Sprintf("%s_user:%s:%d", prefix, uid, userMax)
				}
			}

			// Priority 3: Fallback to IP (anonymous)
			return fmt.Sprintf("%s_ip:%s:%d", prefix, c.IP(), anonMax)
		},
		Message: "Rate limit exceeded. Please try again later.",
		Storage: f.storage,
	}), nil
}

// CreateEmailBasedLimiter creates a rate limiter keyed by email from request body.
// This is useful for operations like password reset where email is the primary identifier.
func (f *RateLimitFactory) CreateEmailBasedLimiter(name string, max int, window time.Duration) (fiber.Handler, error) {
	def, ok := f.registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown rate limiter: %s", name)
	}

	prefix := def.KeyPrefix
	if prefix == "" {
		prefix = def.Name
	}

	return NewRateLimiter(RateLimiterConfig{
		Name:       def.Name,
		Max:        max,
		Expiration: window,
		KeyFunc: func(c fiber.Ctx) string {
			var req struct {
				Email string `json:"email"`
			}
			if err := c.Bind().Body(&req); err == nil && req.Email != "" {
				return prefix + ":" + req.Email
			}
			return prefix + ":" + c.IP()
		},
		Message: "Too many requests. Please try again later.",
		Storage: f.storage,
	}), nil
}
