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
// RequireGlobalIPAllowlist Tests
// =============================================================================

func TestRequireGlobalIPAllowlist_AllowsAllWhenEmpty(t *testing.T) {
	app := fiber.New()

	cfg := &config.ServerConfig{
		AllowedIPRanges: []string{}, // Empty - allow all
	}

	app.Use(RequireGlobalIPAllowlist(cfg))
	app.Get("/api/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
}

func TestRequireGlobalIPAllowlist_AllowsIPInRange(t *testing.T) {
	tests := []struct {
		name        string
		ipRanges    []string
		clientIP    string
		shouldAllow bool
	}{
		{
			name:        "allows IP in /8 network",
			ipRanges:    []string{"10.0.0.0/8"},
			clientIP:    "10.255.255.255",
			shouldAllow: true,
		},
		{
			name:        "allows IP in /16 network",
			ipRanges:    []string{"192.168.0.0/16"},
			clientIP:    "192.168.100.50",
			shouldAllow: true,
		},
		{
			name:        "allows IP in /24 network",
			ipRanges:    []string{"172.16.1.0/24"},
			clientIP:    "172.16.1.100",
			shouldAllow: true,
		},
		{
			name:        "allows exact IP with /32",
			ipRanges:    []string{"203.0.113.42/32"},
			clientIP:    "203.0.113.42",
			shouldAllow: true,
		},
		{
			name:        "denies IP outside range",
			ipRanges:    []string{"10.0.0.0/8"},
			clientIP:    "192.168.1.1",
			shouldAllow: false,
		},
		{
			name:        "denies nearby IP outside /32",
			ipRanges:    []string{"203.0.113.42/32"},
			clientIP:    "203.0.113.43",
			shouldAllow: false,
		},
		{
			name:        "allows IPv6 address",
			ipRanges:    []string{"2001:db8::/32"},
			clientIP:    "2001:db8::1",
			shouldAllow: true,
		},
		{
			name:        "denies IPv6 outside range",
			ipRanges:    []string{"2001:db8::/32"},
			clientIP:    "2001:db9::1",
			shouldAllow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			cfg := &config.ServerConfig{
				AllowedIPRanges: tt.ipRanges,
			}

			app.Use(RequireGlobalIPAllowlist(cfg))
			app.Get("/api/test", func(c *fiber.Ctx) error {
				return c.SendString("OK")
			})

			req := httptest.NewRequest("GET", "/api/test", nil)
			req.Header.Set("X-Forwarded-For", tt.clientIP)

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			if tt.shouldAllow {
				assert.Equal(t, 200, resp.StatusCode)
			} else {
				assert.Equal(t, 403, resp.StatusCode)
			}
		})
	}
}

func TestRequireGlobalIPAllowlist_MultipleRanges(t *testing.T) {
	app := fiber.New()

	cfg := &config.ServerConfig{
		AllowedIPRanges: []string{
			"10.0.0.0/8",
			"192.168.0.0/16",
			"172.16.0.0/12",
			"203.0.113.0/24",
		},
	}

	app.Use(RequireGlobalIPAllowlist(cfg))
	app.Get("/api/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Test IP in each range
	allowedIPs := []string{"10.1.2.3", "192.168.50.100", "172.20.30.40", "203.0.113.99"}
	for _, ip := range allowedIPs {
		t.Run("allows "+ip, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/test", nil)
			req.Header.Set("X-Forwarded-For", ip)

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, 200, resp.StatusCode)
		})
	}

	// Test IPs not in any range
	deniedIPs := []string{"8.8.8.8", "1.1.1.1", "198.51.100.1"}
	for _, ip := range deniedIPs {
		t.Run("denies "+ip, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/test", nil)
			req.Header.Set("X-Forwarded-For", ip)

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, 403, resp.StatusCode)
		})
	}
}

func TestRequireGlobalIPAllowlist_ErrorResponse(t *testing.T) {
	app := fiber.New()

	cfg := &config.ServerConfig{
		AllowedIPRanges: []string{"10.0.0.0/8"},
	}

	app.Use(RequireGlobalIPAllowlist(cfg))
	app.Get("/api/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.1") // Not in 10.0.0.0/8

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 403, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "IP not allowlisted")
}

func TestRequireGlobalIPAllowlist_InvalidCIDR(t *testing.T) {
	app := fiber.New()

	cfg := &config.ServerConfig{
		AllowedIPRanges: []string{
			"invalid-cidr",
			"not-a-network",
			"10.0.0.0/8", // Valid one should still work
		},
	}

	app.Use(RequireGlobalIPAllowlist(cfg))
	app.Get("/api/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// IP in the valid range should still be allowed
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-Forwarded-For", "10.1.2.3")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
}

func TestRequireGlobalIPAllowlist_AllInvalidCIDRs(t *testing.T) {
	app := fiber.New()

	cfg := &config.ServerConfig{
		AllowedIPRanges: []string{
			"invalid-cidr",
			"not-a-network",
			"also-invalid",
		},
	}

	app.Use(RequireGlobalIPAllowlist(cfg))
	app.Get("/api/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// With all invalid CIDRs, no networks are parsed, so allowedNets is empty
	// Empty allowedNets means allow all (backward compatible)
	req := httptest.NewRequest("GET", "/api/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
}

func TestRequireGlobalIPAllowlist_XRealIP(t *testing.T) {
	app := fiber.New()

	cfg := &config.ServerConfig{
		AllowedIPRanges: []string{"10.0.0.0/8"},
	}

	app.Use(RequireGlobalIPAllowlist(cfg))
	app.Get("/api/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Test with X-Real-IP instead of X-Forwarded-For
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-Real-IP", "10.50.100.200")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
}

func TestRequireGlobalIPAllowlist_ProxyChain(t *testing.T) {
	app := fiber.New()

	cfg := &config.ServerConfig{
		AllowedIPRanges: []string{"203.0.113.0/24"},
	}

	app.Use(RequireGlobalIPAllowlist(cfg))
	app.Get("/api/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Simulate a proxy chain - should use the first (original client) IP
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.50, 10.0.0.1, 172.16.0.1")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
}

func TestRequireGlobalIPAllowlist_ProxyChain_DeniedClient(t *testing.T) {
	app := fiber.New()

	cfg := &config.ServerConfig{
		AllowedIPRanges: []string{"10.0.0.0/8"}, // Only 10.x.x.x allowed
	}

	app.Use(RequireGlobalIPAllowlist(cfg))
	app.Get("/api/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Original client (first IP) is not in allowed range
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.50, 10.0.0.1")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should deny based on original client IP
	assert.Equal(t, 403, resp.StatusCode)
}

func TestRequireGlobalIPAllowlist_LocalhostIPv6(t *testing.T) {
	app := fiber.New()

	cfg := &config.ServerConfig{
		AllowedIPRanges: []string{"::1/128"}, // Localhost IPv6
	}

	app.Use(RequireGlobalIPAllowlist(cfg))
	app.Get("/api/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-Forwarded-For", "::1")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
}

func TestRequireGlobalIPAllowlist_MixedIPv4IPv6(t *testing.T) {
	app := fiber.New()

	cfg := &config.ServerConfig{
		AllowedIPRanges: []string{
			"10.0.0.0/8",
			"2001:db8::/32",
		},
	}

	app.Use(RequireGlobalIPAllowlist(cfg))
	app.Get("/api/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	t.Run("allows IPv4", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("X-Forwarded-For", "10.1.2.3")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("allows IPv6", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("X-Forwarded-For", "2001:db8::abcd")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("denies non-matching IPv4", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("X-Forwarded-For", "192.168.1.1")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 403, resp.StatusCode)
	})

	t.Run("denies non-matching IPv6", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("X-Forwarded-For", "2001:db9::1")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 403, resp.StatusCode)
	})
}

func TestRequireGlobalIPAllowlist_LargeNetwork(t *testing.T) {
	app := fiber.New()

	// Allow entire 0.0.0.0/0 (all IPv4)
	cfg := &config.ServerConfig{
		AllowedIPRanges: []string{"0.0.0.0/0"},
	}

	app.Use(RequireGlobalIPAllowlist(cfg))
	app.Get("/api/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	ips := []string{"1.2.3.4", "10.0.0.1", "192.168.1.1", "255.255.255.255"}
	for _, ip := range ips {
		t.Run("allows "+ip, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/test", nil)
			req.Header.Set("X-Forwarded-For", ip)

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, 200, resp.StatusCode)
		})
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkRequireGlobalIPAllowlist_EmptyConfig(b *testing.B) {
	app := fiber.New()

	cfg := &config.ServerConfig{
		AllowedIPRanges: []string{},
	}

	app.Use(RequireGlobalIPAllowlist(cfg))
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

func BenchmarkRequireGlobalIPAllowlist_SingleRange(b *testing.B) {
	app := fiber.New()

	cfg := &config.ServerConfig{
		AllowedIPRanges: []string{"10.0.0.0/8"},
	}

	app.Use(RequireGlobalIPAllowlist(cfg))
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

func BenchmarkRequireGlobalIPAllowlist_MultipleRanges(b *testing.B) {
	app := fiber.New()

	cfg := &config.ServerConfig{
		AllowedIPRanges: []string{
			"10.0.0.0/8",
			"192.168.0.0/16",
			"172.16.0.0/12",
			"203.0.113.0/24",
			"198.51.100.0/24",
		},
	}

	app.Use(RequireGlobalIPAllowlist(cfg))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.50") // Match on 4th range

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, _ := app.Test(req)
		resp.Body.Close()
	}
}

func BenchmarkRequireGlobalIPAllowlist_Denied(b *testing.B) {
	app := fiber.New()

	cfg := &config.ServerConfig{
		AllowedIPRanges: []string{"10.0.0.0/8"},
	}

	app.Use(RequireGlobalIPAllowlist(cfg))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.1") // Not in range

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, _ := app.Test(req)
		resp.Body.Close()
	}
}
