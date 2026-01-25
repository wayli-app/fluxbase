package middleware

import (
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// RateLimiterConfig Tests
// =============================================================================

func TestRateLimiterConfig_Fields(t *testing.T) {
	config := RateLimiterConfig{
		Name:       "test_limiter",
		Max:        100,
		Expiration: time.Minute,
		KeyFunc: func(c *fiber.Ctx) string {
			return "test:" + c.IP()
		},
		Message: "Custom rate limit message",
	}

	assert.Equal(t, "test_limiter", config.Name)
	assert.Equal(t, 100, config.Max)
	assert.Equal(t, time.Minute, config.Expiration)
	assert.NotNil(t, config.KeyFunc)
	assert.Equal(t, "Custom rate limit message", config.Message)
}

func TestRateLimiterConfig_EmptyFields(t *testing.T) {
	config := RateLimiterConfig{}

	assert.Empty(t, config.Name)
	assert.Equal(t, 0, config.Max)
	assert.Equal(t, time.Duration(0), config.Expiration)
	assert.Nil(t, config.KeyFunc)
	assert.Empty(t, config.Message)
}

// =============================================================================
// NewRateLimiter Tests
// =============================================================================

func TestNewRateLimiter_NotNil(t *testing.T) {
	limiter := NewRateLimiter(RateLimiterConfig{
		Max:        10,
		Expiration: time.Minute,
	})

	assert.NotNil(t, limiter)
}

func TestNewRateLimiter_DefaultKeyFunc(t *testing.T) {
	// Config without KeyFunc should use IP-based default
	limiter := NewRateLimiter(RateLimiterConfig{
		Max:        10,
		Expiration: time.Minute,
	})

	app := fiber.New()
	app.Use(limiter)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestNewRateLimiter_CustomMessage(t *testing.T) {
	customMessage := "Custom rate limit error message"

	limiter := NewRateLimiter(RateLimiterConfig{
		Max:        1, // Very low to trigger quickly
		Expiration: time.Hour,
		Message:    customMessage,
	})

	app := fiber.New()
	app.Use(limiter)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// First request should succeed
	req1 := httptest.NewRequest("GET", "/test", nil)
	resp1, err := app.Test(req1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp1.StatusCode)

	// Second request should be rate limited
	req2 := httptest.NewRequest("GET", "/test", nil)
	resp2, err := app.Test(req2)
	require.NoError(t, err)
	assert.Equal(t, 429, resp2.StatusCode)

	// Check response body contains custom message
	body, _ := io.ReadAll(resp2.Body)
	assert.Contains(t, string(body), customMessage)
}

func TestNewRateLimiter_RetryAfterHeader(t *testing.T) {
	limiter := NewRateLimiter(RateLimiterConfig{
		Max:        1,
		Expiration: 30 * time.Second,
	})

	app := fiber.New()
	app.Use(limiter)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// First request succeeds
	req1 := httptest.NewRequest("GET", "/test", nil)
	_, _ = app.Test(req1)

	// Second request should have Retry-After header
	req2 := httptest.NewRequest("GET", "/test", nil)
	resp2, err := app.Test(req2)
	require.NoError(t, err)
	assert.Equal(t, 429, resp2.StatusCode)
	assert.Equal(t, "30", resp2.Header.Get("Retry-After"))
}

// =============================================================================
// Preset Limiter Tests
// =============================================================================

func TestAuthLoginLimiter(t *testing.T) {
	limiter := AuthLoginLimiter()
	assert.NotNil(t, limiter)
}

func TestAuthSignupLimiter(t *testing.T) {
	limiter := AuthSignupLimiter()
	assert.NotNil(t, limiter)
}

func TestAuthPasswordResetLimiter(t *testing.T) {
	limiter := AuthPasswordResetLimiter()
	assert.NotNil(t, limiter)
}

func TestAuth2FALimiter(t *testing.T) {
	limiter := Auth2FALimiter()
	assert.NotNil(t, limiter)
}

func TestAuthRefreshLimiter(t *testing.T) {
	limiter := AuthRefreshLimiter()
	assert.NotNil(t, limiter)
}

func TestAuthMagicLinkLimiter(t *testing.T) {
	limiter := AuthMagicLinkLimiter()
	assert.NotNil(t, limiter)
}

func TestGlobalAPILimiter(t *testing.T) {
	limiter := GlobalAPILimiter()
	assert.NotNil(t, limiter)
}

func TestAuthenticatedUserLimiter(t *testing.T) {
	limiter := AuthenticatedUserLimiter()
	assert.NotNil(t, limiter)
}

func TestDefaultClientKeyLimiter(t *testing.T) {
	limiter := DefaultClientKeyLimiter()
	assert.NotNil(t, limiter)
}

func TestAdminSetupLimiter(t *testing.T) {
	limiter := AdminSetupLimiter()
	assert.NotNil(t, limiter)
}

func TestAdminLoginLimiter(t *testing.T) {
	limiter := AdminLoginLimiter()
	assert.NotNil(t, limiter)
}

func TestGitHubWebhookLimiter(t *testing.T) {
	limiter := GitHubWebhookLimiter()
	assert.NotNil(t, limiter)
}

func TestMigrationAPILimiter(t *testing.T) {
	limiter := MigrationAPILimiter()
	assert.NotNil(t, limiter)
}

// =============================================================================
// ClientKeyLimiter Tests
// =============================================================================

func TestClientKeyLimiter_CustomLimits(t *testing.T) {
	limits := []struct {
		max      int
		duration time.Duration
	}{
		{100, time.Minute},
		{500, time.Minute},
		{1000, time.Hour},
		{10, time.Second},
	}

	for _, limit := range limits {
		limiter := ClientKeyLimiter(limit.max, limit.duration)
		assert.NotNil(t, limiter)
	}
}

// =============================================================================
// AuthEmailBasedLimiter Tests
// =============================================================================

func TestAuthEmailBasedLimiter(t *testing.T) {
	limiter := AuthEmailBasedLimiter("test", 5, 15*time.Minute)
	assert.NotNil(t, limiter)
}

func TestAuthEmailBasedLimiter_DifferentPrefixes(t *testing.T) {
	prefixes := []string{
		"password_reset",
		"email_change",
		"verification",
	}

	for _, prefix := range prefixes {
		limiter := AuthEmailBasedLimiter(prefix, 10, time.Hour)
		assert.NotNil(t, limiter)
	}
}

// =============================================================================
// PerUserOrIPLimiter Tests
// =============================================================================

func TestPerUserOrIPLimiter(t *testing.T) {
	limiter := PerUserOrIPLimiter(10, 100, 500, time.Minute)
	assert.NotNil(t, limiter)
}

func TestPerUserOrIPLimiter_DifferentLimits(t *testing.T) {
	configs := []struct {
		anonMax      int
		userMax      int
		clientKeyMax int
		duration     time.Duration
	}{
		{10, 100, 500, time.Minute},
		{50, 500, 1000, time.Minute},
		{5, 50, 100, time.Second},
	}

	for _, cfg := range configs {
		limiter := PerUserOrIPLimiter(cfg.anonMax, cfg.userMax, cfg.clientKeyMax, cfg.duration)
		assert.NotNil(t, limiter)
	}
}

// =============================================================================
// SetRateLimiterMetrics Tests
// =============================================================================

func TestSetRateLimiterMetrics(t *testing.T) {
	// Setting to nil should not panic
	SetRateLimiterMetrics(nil)
	assert.Nil(t, rateLimiterMetrics)
}

// =============================================================================
// Rate Limit Response Format Tests
// =============================================================================

func TestRateLimitResponse_Format(t *testing.T) {
	limiter := NewRateLimiter(RateLimiterConfig{
		Max:        1,
		Expiration: time.Minute,
		Message:    "Rate limit exceeded",
	})

	app := fiber.New()
	app.Use(limiter)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Trigger rate limit
	req1 := httptest.NewRequest("GET", "/test", nil)
	_, _ = app.Test(req1)

	req2 := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req2)
	require.NoError(t, err)

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Check JSON response structure
	assert.Contains(t, bodyStr, "RATE_LIMIT_EXCEEDED")
	assert.Contains(t, bodyStr, "error")
	assert.Contains(t, bodyStr, "message")
	assert.Contains(t, bodyStr, "retry_after")
}

// =============================================================================
// Key Function Tests
// =============================================================================

func TestKeyFunc_IPBased(t *testing.T) {
	app := fiber.New()

	var capturedKey string
	limiter := NewRateLimiter(RateLimiterConfig{
		Max:        100,
		Expiration: time.Minute,
		KeyFunc: func(c *fiber.Ctx) string {
			capturedKey = "custom:" + c.IP()
			return capturedKey
		},
	})

	app.Use(limiter)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.42")
	_, err := app.Test(req)

	require.NoError(t, err)
	assert.Contains(t, capturedKey, "custom:")
}

// =============================================================================
// Limiter Integration Tests
// =============================================================================

func TestAuthLoginLimiter_Integration(t *testing.T) {
	app := fiber.New()
	app.Use(AuthLoginLimiter())
	app.Post("/auth/login", func(c *fiber.Ctx) error {
		return c.SendString("Login successful")
	})

	// First request should succeed
	req := httptest.NewRequest("POST", "/auth/login", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestGlobalAPILimiter_Integration(t *testing.T) {
	app := fiber.New()
	app.Use(GlobalAPILimiter())
	app.Get("/api/data", func(c *fiber.Ctx) error {
		return c.SendString("Data")
	})

	// First request should succeed
	req := httptest.NewRequest("GET", "/api/data", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestAuthenticatedUserLimiter_WithUserID(t *testing.T) {
	app := fiber.New()

	// Middleware to simulate authenticated user
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", "user-123")
		return c.Next()
	})

	app.Use(AuthenticatedUserLimiter())
	app.Get("/api/user", func(c *fiber.Ctx) error {
		return c.SendString("User data")
	})

	req := httptest.NewRequest("GET", "/api/user", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestClientKeyLimiter_WithClientKeyID(t *testing.T) {
	app := fiber.New()

	// Middleware to simulate client key authentication
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("client_key_id", "key-abc123")
		return c.Next()
	})

	app.Use(DefaultClientKeyLimiter())
	app.Get("/api/data", func(c *fiber.Ctx) error {
		return c.SendString("Data")
	})

	req := httptest.NewRequest("GET", "/api/data", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

// =============================================================================
// MigrationAPILimiter Service Role Bypass Tests
// =============================================================================

func TestMigrationAPILimiter_ServiceRoleBypass(t *testing.T) {
	app := fiber.New()

	// Simulate service_role JWT authentication
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_role", "service_role")
		return c.Next()
	})

	app.Use(MigrationAPILimiter())
	app.Post("/migrations/up", func(c *fiber.Ctx) error {
		return c.SendString("Migration successful")
	})

	// Make many requests - should not be rate limited
	for i := 0; i < 20; i++ {
		req := httptest.NewRequest("POST", "/migrations/up", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	}
}

func TestMigrationAPILimiter_NonServiceRole(t *testing.T) {
	app := fiber.New()

	// Simulate non-service_role authentication
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_role", "authenticated")
		return c.Next()
	})

	app.Use(MigrationAPILimiter())
	app.Post("/migrations/up", func(c *fiber.Ctx) error {
		return c.SendString("Migration successful")
	})

	// First request should succeed
	req := httptest.NewRequest("POST", "/migrations/up", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkNewRateLimiter(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewRateLimiter(RateLimiterConfig{
			Max:        100,
			Expiration: time.Minute,
		})
	}
}

func BenchmarkAuthLoginLimiter(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = AuthLoginLimiter()
	}
}

func BenchmarkGlobalAPILimiter(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GlobalAPILimiter()
	}
}

func BenchmarkRateLimiter_Request(b *testing.B) {
	app := fiber.New()
	app.Use(NewRateLimiter(RateLimiterConfig{
		Max:        1000000, // High limit to avoid rate limiting during benchmark
		Expiration: time.Minute,
	}))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		_, _ = app.Test(req)
	}
}

// =============================================================================
// Concurrent Request Tests
// =============================================================================

func TestRateLimiter_ConcurrentRequests(t *testing.T) {
	app := fiber.New()
	app.Use(NewRateLimiter(RateLimiterConfig{
		Max:        1000,
		Expiration: time.Minute,
	}))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				req := httptest.NewRequest("GET", "/test", nil)
				resp, err := app.Test(req)
				if err == nil {
					resp.Body.Close()
				}
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// No panics means success
}
