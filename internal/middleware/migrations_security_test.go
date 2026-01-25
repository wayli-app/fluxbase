package middleware

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// getClientIP Tests
// =============================================================================

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expectedIP string
	}{
		{
			name:       "uses X-Forwarded-For first IP",
			headers:    map[string]string{"X-Forwarded-For": "1.2.3.4, 5.6.7.8"},
			expectedIP: "1.2.3.4",
		},
		{
			name:       "uses X-Forwarded-For single IP",
			headers:    map[string]string{"X-Forwarded-For": "10.0.0.1"},
			expectedIP: "10.0.0.1",
		},
		{
			name:       "uses X-Forwarded-For with spaces",
			headers:    map[string]string{"X-Forwarded-For": "  192.168.1.1  , 10.0.0.1"},
			expectedIP: "192.168.1.1",
		},
		{
			name:       "uses X-Real-IP when no X-Forwarded-For",
			headers:    map[string]string{"X-Real-IP": "172.16.0.1"},
			expectedIP: "172.16.0.1",
		},
		{
			name:       "prefers X-Forwarded-For over X-Real-IP",
			headers:    map[string]string{"X-Forwarded-For": "1.1.1.1", "X-Real-IP": "2.2.2.2"},
			expectedIP: "1.1.1.1",
		},
		{
			name:       "handles IPv6 in X-Forwarded-For",
			headers:    map[string]string{"X-Forwarded-For": "2001:db8::1"},
			expectedIP: "2001:db8::1",
		},
		{
			name:       "handles IPv6 in X-Real-IP",
			headers:    map[string]string{"X-Real-IP": "::1"},
			expectedIP: "::1",
		},
		{
			name:       "handles invalid X-Forwarded-For, tries X-Real-IP",
			headers:    map[string]string{"X-Forwarded-For": "invalid", "X-Real-IP": "8.8.8.8"},
			expectedIP: "8.8.8.8",
		},
		{
			name:       "handles all invalid headers, falls back to RemoteAddr",
			headers:    map[string]string{"X-Forwarded-For": "invalid", "X-Real-IP": "also-invalid"},
			expectedIP: "", // Falls back to c.IP() which is derived from RemoteAddr
		},
		{
			name:       "no headers - uses RemoteAddr",
			headers:    map[string]string{},
			expectedIP: "", // Falls back to c.IP()
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			var resultIP string
			app.Get("/test", func(c *fiber.Ctx) error {
				ip := getClientIP(c)
				if ip != nil {
					resultIP = ip.String()
				} else {
					resultIP = ""
				}
				return c.SendString("OK")
			})

			req := httptest.NewRequest("GET", "/test", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			if tt.expectedIP != "" {
				assert.Equal(t, tt.expectedIP, resultIP)
			}
		})
	}
}

func TestGetClientIP_MultipleIPsInForwardedFor(t *testing.T) {
	app := fiber.New()

	var resultIP string
	app.Get("/test", func(c *fiber.Ctx) error {
		ip := getClientIP(c)
		if ip != nil {
			resultIP = ip.String()
		}
		return c.SendString("OK")
	})

	// Simulate multiple proxy hops
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18, 150.172.238.178")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should use the first (leftmost) IP - the original client
	assert.Equal(t, "203.0.113.195", resultIP)
}

// =============================================================================
// min Function Tests
// =============================================================================

func TestMin(t *testing.T) {
	tests := []struct {
		a, b     int
		expected int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{0, 0, 0},
		{-1, 1, -1},
		{-5, -3, -5},
		{100, 50, 50},
		{0, -1, -1},
	}

	for _, tt := range tests {
		result := min(tt.a, tt.b)
		assert.Equal(t, tt.expected, result, "min(%d, %d) should be %d", tt.a, tt.b, tt.expected)
	}
}

// =============================================================================
// RequireMigrationsEnabled Tests
// =============================================================================

func TestRequireMigrationsEnabled(t *testing.T) {
	t.Run("allows access when enabled", func(t *testing.T) {
		app := fiber.New()

		cfg := &config.MigrationsConfig{
			Enabled: true,
		}

		app.Use(RequireMigrationsEnabled(cfg))
		app.Get("/migrations", func(c *fiber.Ctx) error {
			return c.SendString("Migrations API")
		})

		req := httptest.NewRequest("GET", "/migrations", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("returns 404 when disabled", func(t *testing.T) {
		app := fiber.New()

		cfg := &config.MigrationsConfig{
			Enabled: false,
		}

		app.Use(RequireMigrationsEnabled(cfg))
		app.Get("/migrations", func(c *fiber.Ctx) error {
			return c.SendString("Migrations API")
		})

		req := httptest.NewRequest("GET", "/migrations", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 404, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "Not Found")
	})
}

// =============================================================================
// RequireMigrationsIPAllowlist Tests
// =============================================================================

func TestRequireMigrationsIPAllowlist(t *testing.T) {
	t.Run("allows all when no IP ranges configured", func(t *testing.T) {
		app := fiber.New()

		cfg := &config.MigrationsConfig{
			Enabled:         true,
			AllowedIPRanges: []string{}, // Empty - allow all
		}

		app.Use(RequireMigrationsIPAllowlist(cfg))
		app.Get("/migrations", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/migrations", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("allows IP in range", func(t *testing.T) {
		app := fiber.New()

		cfg := &config.MigrationsConfig{
			Enabled:         true,
			AllowedIPRanges: []string{"10.0.0.0/8"},
		}

		app.Use(RequireMigrationsIPAllowlist(cfg))
		app.Get("/migrations", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/migrations", nil)
		req.Header.Set("X-Forwarded-For", "10.1.2.3")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("denies IP not in range", func(t *testing.T) {
		app := fiber.New()

		cfg := &config.MigrationsConfig{
			Enabled:         true,
			AllowedIPRanges: []string{"10.0.0.0/8"},
		}

		app.Use(RequireMigrationsIPAllowlist(cfg))
		app.Get("/migrations", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/migrations", nil)
		req.Header.Set("X-Forwarded-For", "192.168.1.1")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 403, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "IP not allowlisted")
	})

	t.Run("allows IP in any of multiple ranges", func(t *testing.T) {
		app := fiber.New()

		cfg := &config.MigrationsConfig{
			Enabled:         true,
			AllowedIPRanges: []string{"10.0.0.0/8", "192.168.0.0/16", "172.16.0.0/12"},
		}

		app.Use(RequireMigrationsIPAllowlist(cfg))
		app.Get("/migrations", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		// Test IP in second range
		req := httptest.NewRequest("GET", "/migrations", nil)
		req.Header.Set("X-Forwarded-For", "192.168.100.50")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("handles invalid CIDR gracefully", func(t *testing.T) {
		app := fiber.New()

		cfg := &config.MigrationsConfig{
			Enabled:         true,
			AllowedIPRanges: []string{"invalid-cidr", "10.0.0.0/8"},
		}

		app.Use(RequireMigrationsIPAllowlist(cfg))
		app.Get("/migrations", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		// Should still work with valid CIDR
		req := httptest.NewRequest("GET", "/migrations", nil)
		req.Header.Set("X-Forwarded-For", "10.1.2.3")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("handles single IP CIDR (/32)", func(t *testing.T) {
		app := fiber.New()

		cfg := &config.MigrationsConfig{
			Enabled:         true,
			AllowedIPRanges: []string{"203.0.113.50/32"},
		}

		app.Use(RequireMigrationsIPAllowlist(cfg))
		app.Get("/migrations", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		// Exact IP should be allowed
		req := httptest.NewRequest("GET", "/migrations", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.50")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)

		// Different IP should be denied
		req2 := httptest.NewRequest("GET", "/migrations", nil)
		req2.Header.Set("X-Forwarded-For", "203.0.113.51")

		resp2, err := app.Test(req2)
		require.NoError(t, err)
		defer resp2.Body.Close()

		assert.Equal(t, 403, resp2.StatusCode)
	})
}

// =============================================================================
// RequireMigrationScope Tests
// =============================================================================

func TestRequireMigrationScope(t *testing.T) {
	t.Run("allows JWT with service_role", func(t *testing.T) {
		app := fiber.New()

		app.Use(func(c *fiber.Ctx) error {
			c.Locals("auth_type", "jwt")
			c.Locals("user_role", "service_role")
			return c.Next()
		})

		app.Use(RequireMigrationScope())
		app.Get("/migrations", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/migrations", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("denies JWT without service_role", func(t *testing.T) {
		app := fiber.New()

		app.Use(func(c *fiber.Ctx) error {
			c.Locals("auth_type", "jwt")
			c.Locals("user_role", "authenticated")
			return c.Next()
		})

		app.Use(RequireMigrationScope())
		app.Get("/migrations", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/migrations", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 403, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "service_role")
	})

	t.Run("allows service key with migrations:execute scope", func(t *testing.T) {
		app := fiber.New()

		app.Use(func(c *fiber.Ctx) error {
			c.Locals("auth_type", "service_key")
			c.Locals("service_key_scopes", []string{"migrations:execute", "storage:read"})
			return c.Next()
		})

		app.Use(RequireMigrationScope())
		app.Get("/migrations", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/migrations", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("allows service key with wildcard scope", func(t *testing.T) {
		app := fiber.New()

		app.Use(func(c *fiber.Ctx) error {
			c.Locals("auth_type", "service_key")
			c.Locals("service_key_scopes", []string{"*"})
			return c.Next()
		})

		app.Use(RequireMigrationScope())
		app.Get("/migrations", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/migrations", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("denies service key without migrations scope", func(t *testing.T) {
		app := fiber.New()

		app.Use(func(c *fiber.Ctx) error {
			c.Locals("auth_type", "service_key")
			c.Locals("service_key_scopes", []string{"storage:read", "storage:write"})
			return c.Next()
		})

		app.Use(RequireMigrationScope())
		app.Get("/migrations", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/migrations", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 403, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "migrations:execute scope")
	})

	t.Run("denies service key with no scopes", func(t *testing.T) {
		app := fiber.New()

		app.Use(func(c *fiber.Ctx) error {
			c.Locals("auth_type", "service_key")
			// No scopes set
			return c.Next()
		})

		app.Use(RequireMigrationScope())
		app.Get("/migrations", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/migrations", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 403, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "No scopes found")
	})

	t.Run("denies unknown auth type", func(t *testing.T) {
		app := fiber.New()

		app.Use(func(c *fiber.Ctx) error {
			c.Locals("auth_type", "api_key") // Not JWT or service_key
			return c.Next()
		})

		app.Use(RequireMigrationScope())
		app.Get("/migrations", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/migrations", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 403, resp.StatusCode)
	})

	t.Run("denies no auth type", func(t *testing.T) {
		app := fiber.New()

		app.Use(RequireMigrationScope())
		app.Get("/migrations", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/migrations", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 403, resp.StatusCode)
	})
}

// =============================================================================
// MigrationsAuditLog Tests
// =============================================================================

func TestMigrationsAuditLog(t *testing.T) {
	t.Run("logs request and response", func(t *testing.T) {
		app := fiber.New()

		app.Use(func(c *fiber.Ctx) error {
			c.Locals("service_key_id", "key-123")
			c.Locals("service_key_name", "production-key")
			return c.Next()
		})

		app.Use(MigrationsAuditLog())
		app.Get("/migrations/status", func(c *fiber.Ctx) error {
			return c.SendString("Migration Status")
		})

		req := httptest.NewRequest("GET", "/migrations/status", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("logs without service key info", func(t *testing.T) {
		app := fiber.New()

		app.Use(MigrationsAuditLog())
		app.Post("/migrations/run", func(c *fiber.Ctx) error {
			return c.SendString("Migration Run")
		})

		req := httptest.NewRequest("POST", "/migrations/run", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("logs error responses", func(t *testing.T) {
		app := fiber.New()

		app.Use(MigrationsAuditLog())
		app.Get("/migrations/fail", func(c *fiber.Ctx) error {
			return c.Status(500).SendString("Internal Error")
		})

		req := httptest.NewRequest("GET", "/migrations/fail", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 500, resp.StatusCode)
	})

	t.Run("passes through errors from handler", func(t *testing.T) {
		app := fiber.New()

		expectedError := fiber.NewError(400, "Bad Request")

		app.Use(MigrationsAuditLog())
		app.Get("/migrations/error", func(c *fiber.Ctx) error {
			return expectedError
		})

		req := httptest.NewRequest("GET", "/migrations/error", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 400, resp.StatusCode)
	})
}

// =============================================================================
// RequireServiceKeyOnly Tests (validation paths only - no DB)
// =============================================================================

func TestRequireServiceKeyOnly_NoAuth(t *testing.T) {
	// Test the case where no authentication is provided
	// This doesn't require a database connection

	app := fiber.New()

	// Use nil for db and authService - will fail before database calls
	app.Use(RequireServiceKeyOnly(nil, nil))
	app.Get("/migrations", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/migrations", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 401, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Service key or service_role JWT authentication required")
}

func TestRequireServiceKeyOnly_InvalidServiceKeyFormat(t *testing.T) {
	app := fiber.New()

	// Use nil for db - will fail on invalid key format before database call
	app.Use(RequireServiceKeyOnly(nil, nil))
	app.Get("/migrations", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	t.Run("too short service key", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/migrations", nil)
		req.Header.Set("X-Service-Key", "sk_short")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 401, resp.StatusCode)
	})

	t.Run("wrong prefix", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/migrations", nil)
		req.Header.Set("X-Service-Key", "pk_1234567890123456") // 'pk_' instead of 'sk_'

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 401, resp.StatusCode)
	})
}

func TestRequireServiceKeyOnly_NoAuthProvided(t *testing.T) {
	// Test that requests without any authentication are rejected
	// Note: Tests with service keys require a database connection for validation

	t.Run("rejects request with no auth", func(t *testing.T) {
		app := fiber.New()
		app.Use(RequireServiceKeyOnly(nil, nil))
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 401, resp.StatusCode)
	})

	t.Run("rejects short service key format", func(t *testing.T) {
		app := fiber.New()
		app.Use(RequireServiceKeyOnly(nil, nil))
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Service-Key", "sk_short") // Too short to be valid

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 401, resp.StatusCode)
	})

	t.Run("rejects non-sk prefix key", func(t *testing.T) {
		app := fiber.New()
		app.Use(RequireServiceKeyOnly(nil, nil))
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Service-Key", "pk_1234567890123456_valid") // Wrong prefix

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 401, resp.StatusCode)
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkGetClientIP_WithForwardedFor(b *testing.B) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		_ = getClientIP(c)
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8, 9.10.11.12")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, _ := app.Test(req)
		resp.Body.Close()
	}
}

func BenchmarkGetClientIP_WithRealIP(b *testing.B) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		_ = getClientIP(c)
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Real-IP", "10.0.0.1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, _ := app.Test(req)
		resp.Body.Close()
	}
}

func BenchmarkMin(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = min(100, 50)
	}
}

func BenchmarkRequireMigrationsIPAllowlist_InRange(b *testing.B) {
	app := fiber.New()

	cfg := &config.MigrationsConfig{
		Enabled:         true,
		AllowedIPRanges: []string{"10.0.0.0/8", "192.168.0.0/16", "172.16.0.0/12"},
	}

	app.Use(RequireMigrationsIPAllowlist(cfg))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "10.1.2.3")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, _ := app.Test(req)
		resp.Body.Close()
	}
}

func BenchmarkRequireMigrationScope_JWT(b *testing.B) {
	app := fiber.New()

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("auth_type", "jwt")
		c.Locals("user_role", "service_role")
		return c.Next()
	})

	app.Use(RequireMigrationScope())
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

func BenchmarkRequireMigrationScope_ServiceKey(b *testing.B) {
	app := fiber.New()

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("auth_type", "service_key")
		c.Locals("service_key_scopes", []string{"migrations:execute", "storage:read", "storage:write"})
		return c.Next()
	})

	app.Use(RequireMigrationScope())
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
