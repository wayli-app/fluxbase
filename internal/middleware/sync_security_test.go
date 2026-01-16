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
// RequireSyncIPAllowlist Tests - Empty Config
// =============================================================================

func TestRequireSyncIPAllowlist_EmptyConfig(t *testing.T) {
	app := fiber.New()

	// Empty ranges = allow all
	app.Use(RequireSyncIPAllowlist([]string{}, "functions"))
	app.Get("/sync", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/sync", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
}

func TestRequireSyncIPAllowlist_NilConfig(t *testing.T) {
	app := fiber.New()

	// Nil slice = allow all
	app.Use(RequireSyncIPAllowlist(nil, "jobs"))
	app.Get("/sync", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/sync", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
}

// =============================================================================
// RequireSyncIPAllowlist Tests - IP Matching
// =============================================================================

func TestRequireSyncIPAllowlist_IPMatching(t *testing.T) {
	tests := []struct {
		name        string
		ipRanges    []string
		clientIP    string
		featureName string
		shouldAllow bool
	}{
		{
			name:        "allows IP in /8 range",
			ipRanges:    []string{"10.0.0.0/8"},
			clientIP:    "10.1.2.3",
			featureName: "functions",
			shouldAllow: true,
		},
		{
			name:        "allows IP in /16 range",
			ipRanges:    []string{"192.168.0.0/16"},
			clientIP:    "192.168.100.50",
			featureName: "jobs",
			shouldAllow: true,
		},
		{
			name:        "allows IP in /24 range",
			ipRanges:    []string{"172.16.1.0/24"},
			clientIP:    "172.16.1.100",
			featureName: "functions",
			shouldAllow: true,
		},
		{
			name:        "allows exact IP with /32",
			ipRanges:    []string{"203.0.113.50/32"},
			clientIP:    "203.0.113.50",
			featureName: "functions",
			shouldAllow: true,
		},
		{
			name:        "denies IP outside range",
			ipRanges:    []string{"10.0.0.0/8"},
			clientIP:    "192.168.1.1",
			featureName: "functions",
			shouldAllow: false,
		},
		{
			name:        "denies IP near /32 boundary",
			ipRanges:    []string{"203.0.113.50/32"},
			clientIP:    "203.0.113.51",
			featureName: "functions",
			shouldAllow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			app.Use(RequireSyncIPAllowlist(tt.ipRanges, tt.featureName))
			app.Get("/sync", func(c *fiber.Ctx) error {
				return c.SendString("OK")
			})

			req := httptest.NewRequest("GET", "/sync", nil)
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

// =============================================================================
// RequireSyncIPAllowlist Tests - Multiple Ranges
// =============================================================================

func TestRequireSyncIPAllowlist_MultipleRanges(t *testing.T) {
	app := fiber.New()

	ranges := []string{
		"10.0.0.0/8",
		"192.168.0.0/16",
		"172.16.0.0/12",
	}

	app.Use(RequireSyncIPAllowlist(ranges, "functions"))
	app.Get("/sync", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	t.Run("allows IP in first range", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/sync", nil)
		req.Header.Set("X-Forwarded-For", "10.1.2.3")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("allows IP in second range", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/sync", nil)
		req.Header.Set("X-Forwarded-For", "192.168.50.100")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("allows IP in third range", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/sync", nil)
		req.Header.Set("X-Forwarded-For", "172.20.30.40")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("denies IP not in any range", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/sync", nil)
		req.Header.Set("X-Forwarded-For", "8.8.8.8")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 403, resp.StatusCode)
	})
}

// =============================================================================
// RequireSyncIPAllowlist Tests - Error Message
// =============================================================================

func TestRequireSyncIPAllowlist_ErrorMessage(t *testing.T) {
	tests := []struct {
		featureName     string
		expectedInError string
	}{
		{"functions", "functions sync"},
		{"jobs", "jobs sync"},
		{"custom-feature", "custom-feature sync"},
	}

	for _, tt := range tests {
		t.Run(tt.featureName, func(t *testing.T) {
			app := fiber.New()

			app.Use(RequireSyncIPAllowlist([]string{"10.0.0.0/8"}, tt.featureName))
			app.Get("/sync", func(c *fiber.Ctx) error {
				return c.SendString("OK")
			})

			req := httptest.NewRequest("GET", "/sync", nil)
			req.Header.Set("X-Forwarded-For", "192.168.1.1") // Not in range

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, 403, resp.StatusCode)

			body, _ := io.ReadAll(resp.Body)
			assert.Contains(t, string(body), tt.expectedInError)
		})
	}
}

// =============================================================================
// RequireSyncIPAllowlist Tests - Invalid CIDR
// =============================================================================

func TestRequireSyncIPAllowlist_InvalidCIDR(t *testing.T) {
	t.Run("ignores invalid CIDR", func(t *testing.T) {
		app := fiber.New()

		ranges := []string{
			"invalid-cidr",
			"10.0.0.0/8", // Valid one
		}

		app.Use(RequireSyncIPAllowlist(ranges, "functions"))
		app.Get("/sync", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/sync", nil)
		req.Header.Set("X-Forwarded-For", "10.1.2.3")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("all invalid CIDRs allows all", func(t *testing.T) {
		app := fiber.New()

		ranges := []string{
			"invalid1",
			"invalid2",
			"also-invalid",
		}

		app.Use(RequireSyncIPAllowlist(ranges, "functions"))
		app.Get("/sync", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/sync", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// All invalid = empty valid ranges = allow all
		assert.Equal(t, 200, resp.StatusCode)
	})
}

// =============================================================================
// RequireSyncIPAllowlist Tests - IPv6
// =============================================================================

func TestRequireSyncIPAllowlist_IPv6(t *testing.T) {
	t.Run("allows IPv6 in range", func(t *testing.T) {
		app := fiber.New()

		app.Use(RequireSyncIPAllowlist([]string{"2001:db8::/32"}, "functions"))
		app.Get("/sync", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/sync", nil)
		req.Header.Set("X-Forwarded-For", "2001:db8::1")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("denies IPv6 outside range", func(t *testing.T) {
		app := fiber.New()

		app.Use(RequireSyncIPAllowlist([]string{"2001:db8::/32"}, "functions"))
		app.Get("/sync", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/sync", nil)
		req.Header.Set("X-Forwarded-For", "2001:db9::1") // Different prefix

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 403, resp.StatusCode)
	})
}

// =============================================================================
// RequireSyncIPAllowlist Tests - Proxy Chain
// =============================================================================

func TestRequireSyncIPAllowlist_ProxyChain(t *testing.T) {
	t.Run("uses first IP from X-Forwarded-For", func(t *testing.T) {
		app := fiber.New()

		app.Use(RequireSyncIPAllowlist([]string{"203.0.113.0/24"}, "functions"))
		app.Get("/sync", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/sync", nil)
		// First IP is the original client
		req.Header.Set("X-Forwarded-For", "203.0.113.50, 10.0.0.1, 172.16.0.1")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("denies when original client not in range", func(t *testing.T) {
		app := fiber.New()

		app.Use(RequireSyncIPAllowlist([]string{"10.0.0.0/8"}, "functions"))
		app.Get("/sync", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/sync", nil)
		// First IP (original client) is NOT in range
		req.Header.Set("X-Forwarded-For", "192.168.1.1, 10.0.0.1")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 403, resp.StatusCode)
	})
}

// =============================================================================
// RequireSyncIPAllowlist Tests - X-Real-IP
// =============================================================================

func TestRequireSyncIPAllowlist_XRealIP(t *testing.T) {
	t.Run("uses X-Real-IP when no X-Forwarded-For", func(t *testing.T) {
		app := fiber.New()

		app.Use(RequireSyncIPAllowlist([]string{"10.0.0.0/8"}, "jobs"))
		app.Get("/sync", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/sync", nil)
		req.Header.Set("X-Real-IP", "10.50.100.200")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
	})
}

// =============================================================================
// RequireSyncIPAllowlist Tests - Different Feature Names
// =============================================================================

func TestRequireSyncIPAllowlist_FeatureNames(t *testing.T) {
	features := []string{"functions", "jobs", "realtime", "storage"}

	for _, feature := range features {
		t.Run("logs feature: "+feature, func(t *testing.T) {
			app := fiber.New()

			app.Use(RequireSyncIPAllowlist([]string{"10.0.0.0/8"}, feature))
			app.Get("/sync", func(c *fiber.Ctx) error {
				return c.SendString("OK")
			})

			req := httptest.NewRequest("GET", "/sync", nil)
			req.Header.Set("X-Forwarded-For", "10.1.2.3")

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

func BenchmarkRequireSyncIPAllowlist_EmptyConfig(b *testing.B) {
	app := fiber.New()

	app.Use(RequireSyncIPAllowlist([]string{}, "functions"))
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

func BenchmarkRequireSyncIPAllowlist_SingleRange(b *testing.B) {
	app := fiber.New()

	app.Use(RequireSyncIPAllowlist([]string{"10.0.0.0/8"}, "functions"))
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

func BenchmarkRequireSyncIPAllowlist_MultipleRanges(b *testing.B) {
	app := fiber.New()

	ranges := []string{
		"10.0.0.0/8",
		"192.168.0.0/16",
		"172.16.0.0/12",
		"203.0.113.0/24",
		"198.51.100.0/24",
	}

	app.Use(RequireSyncIPAllowlist(ranges, "functions"))
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

func BenchmarkRequireSyncIPAllowlist_Denied(b *testing.B) {
	app := fiber.New()

	app.Use(RequireSyncIPAllowlist([]string{"10.0.0.0/8"}, "functions"))
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
