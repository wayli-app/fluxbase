package api

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// TestRouteGroupIsolation ensures that middleware applied to one route group
// doesn't leak to other routes with overlapping path prefixes.
// This test prevents the bug where GitHubWebhookLimiter was affecting admin routes.
func TestRouteGroupIsolation(t *testing.T) {
	app := fiber.New()

	// Create main /api/v1 group without rate limiting
	v1 := app.Group("/api/v1")
	v1.Get("/admin/test", func(c fiber.Ctx) error {
		return c.SendString("admin ok")
	})

	// Register webhook route with rate limiting directly (not as a group)
	// This is the correct way to avoid middleware leakage
	app.Post(
		"/api/v1/webhooks/github",
		middleware.GitHubWebhookLimiter(),
		func(c fiber.Ctx) error {
			return c.SendString("webhook ok")
		},
	)

	t.Run("admin route should not be rate limited by webhook limiter", func(t *testing.T) {
		// Make 50 requests to admin endpoint (exceeds webhook limit of 30/min)
		for i := 0; i < 50; i++ {
			req := httptest.NewRequest("GET", "/api/v1/admin/test", nil)
			resp, err := app.Test(req, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
			require.NoError(t, err)

			// All requests should succeed (200) because webhook rate limiter
			// should NOT affect admin routes
			assert.Equal(t, 200, resp.StatusCode,
				"Request %d should succeed - admin routes should not be affected by webhook rate limiter", i+1)
		}
	})

	t.Run("webhook route should be rate limited", func(t *testing.T) {
		// Create a fresh app to reset rate limit counters
		app2 := fiber.New()

		// Register webhook with rate limiter
		app2.Post(
			"/api/v1/webhooks/github",
			middleware.GitHubWebhookLimiter(),
			func(c fiber.Ctx) error {
				return c.SendString("webhook ok")
			},
		)

		// Make 35 requests (exceeds the 30/min limit)
		rateLimitHit := false
		for i := 0; i < 35; i++ {
			req := httptest.NewRequest("POST", "/api/v1/webhooks/github", nil)
			resp, err := app2.Test(req, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
			require.NoError(t, err)

			if resp.StatusCode == 429 {
				rateLimitHit = true
				break
			}
		}

		// Webhook should hit rate limit
		assert.True(t, rateLimitHit, "Webhook endpoint should be rate limited after 30 requests")
	})
}

// TestRouteGroupMiddlewareLeakage is a regression test for the bug where
// creating multiple groups with the same path prefix caused middleware leakage.
// This was the original issue where GitHubWebhookLimiter affected admin routes.
func TestRouteGroupMiddlewareLeakage(t *testing.T) {
	t.Run("ANTI-PATTERN: multiple groups with same prefix causes middleware leakage", func(t *testing.T) {
		app := fiber.New()

		// Main group without middleware
		v1 := app.Group("/api/v1")
		v1.Get("/admin/test", func(c fiber.Ctx) error {
			return c.SendString("admin ok")
		})

		// Second group with same prefix but WITH middleware
		// This is the ANTI-PATTERN that caused the bug
		webhookGroup := app.Group("/api/v1", middleware.GitHubWebhookLimiter())
		webhookGroup.Post("/webhooks/github", func(c fiber.Ctx) error {
			return c.SendString("webhook ok")
		})

		// Make 35 requests to admin endpoint (exceeds webhook limit of 30/min)
		// These SHOULD all succeed, but with the anti-pattern they might not
		successCount := 0
		for i := 0; i < 35; i++ {
			req := httptest.NewRequest("GET", "/api/v1/admin/test", nil)
			resp, err := app.Test(req, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
			require.NoError(t, err)

			if resp.StatusCode == 200 {
				successCount++
			}
		}

		// With the bug, this test would fail because admin routes would be rate limited
		// This test documents the anti-pattern to avoid in the future
		if successCount < 35 {
			t.Logf("WARNING: Middleware leakage detected! Only %d/35 admin requests succeeded", successCount)
			t.Logf("This happens when multiple groups share the same path prefix")
			t.Logf("SOLUTION: Register rate-limited routes directly, not via a group with shared prefix")
		}
	})
}
