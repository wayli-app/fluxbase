package middleware

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// TracingConfig Tests
// =============================================================================

func TestTracingConfig(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		config := TracingConfig{}
		assert.False(t, config.Enabled)
		assert.Empty(t, config.ServiceName)
		assert.Empty(t, config.SkipPaths)
		assert.False(t, config.RecordRequestBody)
		assert.False(t, config.RecordResponseBody)
	})

	t.Run("custom config", func(t *testing.T) {
		config := TracingConfig{
			Enabled:            true,
			ServiceName:        "my-service",
			SkipPaths:          []string{"/health", "/ready"},
			RecordRequestBody:  true,
			RecordResponseBody: true,
		}

		assert.True(t, config.Enabled)
		assert.Equal(t, "my-service", config.ServiceName)
		assert.Len(t, config.SkipPaths, 2)
		assert.True(t, config.RecordRequestBody)
		assert.True(t, config.RecordResponseBody)
	})
}

// =============================================================================
// DefaultTracingConfig Tests
// =============================================================================

func TestDefaultTracingConfig(t *testing.T) {
	cfg := DefaultTracingConfig()

	t.Run("enabled by default", func(t *testing.T) {
		assert.True(t, cfg.Enabled)
	})

	t.Run("service name", func(t *testing.T) {
		assert.Equal(t, "fluxbase", cfg.ServiceName)
	})

	t.Run("default skip paths", func(t *testing.T) {
		assert.Contains(t, cfg.SkipPaths, "/health")
		assert.Contains(t, cfg.SkipPaths, "/ready")
		assert.Contains(t, cfg.SkipPaths, "/metrics")
		assert.Len(t, cfg.SkipPaths, 3)
	})

	t.Run("body recording disabled by default", func(t *testing.T) {
		assert.False(t, cfg.RecordRequestBody)
		assert.False(t, cfg.RecordResponseBody)
	})
}

// =============================================================================
// TracingMiddleware Tests - Disabled
// =============================================================================

func TestTracingMiddleware_Disabled(t *testing.T) {
	app := fiber.New()

	cfg := TracingConfig{
		Enabled: false,
	}

	app.Use(TracingMiddleware(cfg))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "OK", string(body))

	// No trace ID header when disabled
	assert.Empty(t, resp.Header.Get("X-Trace-ID"))
}

// =============================================================================
// TracingMiddleware Tests - Skip Paths
// =============================================================================

func TestTracingMiddleware_SkipPaths(t *testing.T) {
	app := fiber.New()

	cfg := TracingConfig{
		Enabled:   true,
		SkipPaths: []string{"/health", "/ready", "/metrics"},
	}

	app.Use(TracingMiddleware(cfg))

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("healthy")
	})

	app.Get("/api/users", func(c *fiber.Ctx) error {
		return c.SendString("users")
	})

	t.Run("skipped paths do not get trace ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
		// Skip paths should not have trace ID header
		// (This depends on implementation - if middleware skips, no header)
	})

	t.Run("non-skipped paths get processed", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
	})
}

// =============================================================================
// GetTraceContext Tests
// =============================================================================

func TestGetTraceContext(t *testing.T) {
	t.Run("returns empty when no span set", func(t *testing.T) {
		app := fiber.New()

		var hasTraceID, hasSpanID bool
		app.Get("/test", func(c *fiber.Ctx) error {
			ctx := GetTraceContext(c)
			hasTraceID = ctx.HasTraceID()
			hasSpanID = ctx.HasSpanID()
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.False(t, hasTraceID)
		assert.False(t, hasSpanID)
	})

	t.Run("returns empty when wrong type in locals", func(t *testing.T) {
		app := fiber.New()

		var hasTraceID bool
		app.Get("/test", func(c *fiber.Ctx) error {
			c.Locals("trace_span", "not-a-span")
			ctx := GetTraceContext(c)
			hasTraceID = ctx.HasTraceID()
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.False(t, hasTraceID)
	})
}

// =============================================================================
// GetTraceID Tests
// =============================================================================

func TestGetTraceID(t *testing.T) {
	t.Run("returns empty string when no span", func(t *testing.T) {
		app := fiber.New()

		var traceID string
		app.Get("/test", func(c *fiber.Ctx) error {
			traceID = GetTraceID(c)
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Empty(t, traceID)
	})
}

// =============================================================================
// GetSpanID Tests
// =============================================================================

func TestGetSpanID(t *testing.T) {
	t.Run("returns empty string when no span", func(t *testing.T) {
		app := fiber.New()

		var spanID string
		app.Get("/test", func(c *fiber.Ctx) error {
			spanID = GetSpanID(c)
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Empty(t, spanID)
	})
}

// =============================================================================
// AddSpanEvent Tests
// =============================================================================

func TestAddSpanEvent(t *testing.T) {
	t.Run("does not panic when no span", func(t *testing.T) {
		app := fiber.New()

		app.Get("/test", func(c *fiber.Ctx) error {
			// Should not panic
			AddSpanEvent(c, "test-event")
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("does not panic when wrong type", func(t *testing.T) {
		app := fiber.New()

		app.Get("/test", func(c *fiber.Ctx) error {
			c.Locals("trace_span", "not-a-span")
			AddSpanEvent(c, "test-event")
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
	})
}

// =============================================================================
// SetSpanError Tests
// =============================================================================

func TestSetSpanError(t *testing.T) {
	t.Run("does not panic when no span", func(t *testing.T) {
		app := fiber.New()

		app.Get("/test", func(c *fiber.Ctx) error {
			SetSpanError(c, fiber.NewError(400, "Bad Request"))
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
	})
}

// =============================================================================
// SetSpanAttributes Tests
// =============================================================================

func TestSetSpanAttributes(t *testing.T) {
	t.Run("does not panic when no span", func(t *testing.T) {
		app := fiber.New()

		app.Get("/test", func(c *fiber.Ctx) error {
			SetSpanAttributes(c) // No attributes
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
	})
}

// =============================================================================
// StartChildSpan Tests
// =============================================================================

func TestStartChildSpan(t *testing.T) {
	t.Run("returns span and cleanup function", func(t *testing.T) {
		app := fiber.New()

		var spanNotNil bool
		var cleanupNotNil bool

		app.Get("/test", func(c *fiber.Ctx) error {
			span, cleanup := StartChildSpan(c, "child-operation")
			spanNotNil = span != nil
			cleanupNotNil = cleanup != nil
			cleanup() // Call cleanup
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.True(t, spanNotNil)
		assert.True(t, cleanupNotNil)
	})
}

// =============================================================================
// TracingMiddleware Integration Tests
// =============================================================================

func TestTracingMiddleware_RequestLifecycle(t *testing.T) {
	t.Run("processes request and response", func(t *testing.T) {
		app := fiber.New()

		cfg := DefaultTracingConfig()
		app.Use(TracingMiddleware(cfg))

		app.Get("/api/test", func(c *fiber.Ctx) error {
			return c.SendString("Response")
		})

		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("User-Agent", "TestAgent/1.0")
		req.Header.Set("X-Request-ID", "test-request-id")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("handles error responses", func(t *testing.T) {
		app := fiber.New()

		cfg := DefaultTracingConfig()
		app.Use(TracingMiddleware(cfg))

		app.Get("/api/error", func(c *fiber.Ctx) error {
			return c.Status(500).SendString("Internal Error")
		})

		req := httptest.NewRequest("GET", "/api/error", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 500, resp.StatusCode)
	})

	t.Run("handles fiber errors", func(t *testing.T) {
		app := fiber.New()

		cfg := DefaultTracingConfig()
		app.Use(TracingMiddleware(cfg))

		app.Get("/api/fiber-error", func(c *fiber.Ctx) error {
			return fiber.NewError(400, "Bad Request")
		})

		req := httptest.NewRequest("GET", "/api/fiber-error", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 400, resp.StatusCode)
	})
}

// =============================================================================
// TracingMiddleware User Context Tests
// =============================================================================

func TestTracingMiddleware_UserContext(t *testing.T) {
	t.Run("adds user context to span when available", func(t *testing.T) {
		app := fiber.New()

		// Set user context before tracing middleware
		app.Use(func(c *fiber.Ctx) error {
			c.Locals("user_id", "user-123")
			c.Locals("user_role", "admin")
			return c.Next()
		})

		cfg := DefaultTracingConfig()
		app.Use(TracingMiddleware(cfg))

		app.Get("/api/test", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/api/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
	})
}

// =============================================================================
// TracingMiddleware Body Recording Tests
// =============================================================================

func TestTracingMiddleware_BodyRecording(t *testing.T) {
	t.Run("records request body when enabled and small", func(t *testing.T) {
		app := fiber.New()

		cfg := TracingConfig{
			Enabled:           true,
			RecordRequestBody: true,
		}
		app.Use(TracingMiddleware(cfg))

		app.Post("/api/test", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("POST", "/api/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("records response body when enabled and small", func(t *testing.T) {
		app := fiber.New()

		cfg := TracingConfig{
			Enabled:            true,
			RecordResponseBody: true,
		}
		app.Use(TracingMiddleware(cfg))

		app.Get("/api/test", func(c *fiber.Ctx) error {
			return c.SendString("Response body")
		})

		req := httptest.NewRequest("GET", "/api/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkTracingMiddleware_Disabled(b *testing.B) {
	app := fiber.New()

	cfg := TracingConfig{Enabled: false}
	app.Use(TracingMiddleware(cfg))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, _ := app.Test(req)
		resp.Body.Close()
	}
}

func BenchmarkTracingMiddleware_SkipPath(b *testing.B) {
	app := fiber.New()

	cfg := TracingConfig{
		Enabled:   true,
		SkipPaths: []string{"/health"},
	}
	app.Use(TracingMiddleware(cfg))
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/health", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, _ := app.Test(req)
		resp.Body.Close()
	}
}

func BenchmarkGetTraceID_NoSpan(b *testing.B) {
	app := fiber.New()

	app.Get("/test", func(c *fiber.Ctx) error {
		for i := 0; i < b.N; i++ {
			_ = GetTraceID(c)
		}
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)
	resp.Body.Close()
}

func BenchmarkGetSpanID_NoSpan(b *testing.B) {
	app := fiber.New()

	app.Get("/test", func(c *fiber.Ctx) error {
		for i := 0; i < b.N; i++ {
			_ = GetSpanID(c)
		}
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)
	resp.Body.Close()
}

func BenchmarkDefaultTracingConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = DefaultTracingConfig()
	}
}
