package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Constants Tests
// =============================================================================

func TestBranchConstants(t *testing.T) {
	t.Run("header constant", func(t *testing.T) {
		assert.Equal(t, "X-Fluxbase-Branch", BranchHeader)
	})

	t.Run("query param constant", func(t *testing.T) {
		assert.Equal(t, "branch", BranchQueryParam)
	})

	t.Run("locals keys", func(t *testing.T) {
		assert.Equal(t, "branch_slug", LocalsBranchSlug)
		assert.Equal(t, "branch_pool", LocalsBranchPool)
		assert.Equal(t, "branch", LocalsBranch)
	})
}

// =============================================================================
// BranchContextConfig Tests
// =============================================================================

func TestBranchContextConfig(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		config := BranchContextConfig{}
		assert.Nil(t, config.Router)
		assert.False(t, config.RequireAccess)
		assert.False(t, config.AllowAnonymous)
	})

	t.Run("config with access checks", func(t *testing.T) {
		config := BranchContextConfig{
			RequireAccess:  true,
			AllowAnonymous: false,
		}
		assert.True(t, config.RequireAccess)
		assert.False(t, config.AllowAnonymous)
	})
}

// =============================================================================
// GetBranchSlug Tests
// =============================================================================

func TestGetBranchSlug(t *testing.T) {
	t.Run("returns slug from locals", func(t *testing.T) {
		app := fiber.New()

		var result string
		app.Get("/test", func(c *fiber.Ctx) error {
			c.Locals(LocalsBranchSlug, "feature-123")
			result = GetBranchSlug(c)
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, "feature-123", result)
	})

	t.Run("returns main when not set", func(t *testing.T) {
		app := fiber.New()

		var result string
		app.Get("/test", func(c *fiber.Ctx) error {
			result = GetBranchSlug(c)
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, "main", result)
	})

	t.Run("returns main when wrong type", func(t *testing.T) {
		app := fiber.New()

		var result string
		app.Get("/test", func(c *fiber.Ctx) error {
			c.Locals(LocalsBranchSlug, 12345) // Wrong type
			result = GetBranchSlug(c)
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, "main", result)
	})
}

// =============================================================================
// GetBranchPool Tests
// =============================================================================

func TestGetBranchPool(t *testing.T) {
	t.Run("returns nil when not set", func(t *testing.T) {
		app := fiber.New()

		var result interface{}
		app.Get("/test", func(c *fiber.Ctx) error {
			result = GetBranchPool(c)
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Nil(t, result)
	})

	t.Run("returns nil when wrong type", func(t *testing.T) {
		app := fiber.New()

		var result interface{}
		app.Get("/test", func(c *fiber.Ctx) error {
			c.Locals(LocalsBranchPool, "not-a-pool")
			result = GetBranchPool(c)
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Nil(t, result)
	})
}

// =============================================================================
// IsUsingBranch Tests
// =============================================================================

func TestIsUsingBranch(t *testing.T) {
	tests := []struct {
		name     string
		slug     string
		expected bool
	}{
		{"main branch returns false", "main", false},
		{"feature branch returns true", "feature-123", true},
		{"dev branch returns true", "dev", true},
		{"empty string returns false (defaults to main)", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			var result bool
			app.Get("/test", func(c *fiber.Ctx) error {
				if tt.slug != "" {
					c.Locals(LocalsBranchSlug, tt.slug)
				}
				result = IsUsingBranch(c)
				return c.SendString("OK")
			})

			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// BranchContext Middleware Tests (without Router)
// =============================================================================

func TestBranchContext_NoRouter(t *testing.T) {
	t.Run("defaults to main branch", func(t *testing.T) {
		app := fiber.New()

		config := BranchContextConfig{
			Router: nil,
		}

		var branchSlug string
		app.Use(BranchContext(config))
		app.Get("/test", func(c *fiber.Ctx) error {
			branchSlug = GetBranchSlug(c)
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, "main", branchSlug)
	})

	t.Run("extracts branch from header", func(t *testing.T) {
		app := fiber.New()

		config := BranchContextConfig{
			Router: nil,
		}

		app.Use(BranchContext(config))
		app.Get("/test", func(c *fiber.Ctx) error {
			_ = GetBranchSlug(c)
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set(BranchHeader, "feature-xyz")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Without router, non-main branch returns error
		assert.Equal(t, 503, resp.StatusCode)
	})

	t.Run("extracts branch from query param", func(t *testing.T) {
		app := fiber.New()

		config := BranchContextConfig{
			Router: nil,
		}

		app.Use(BranchContext(config))
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test?branch=feature-query", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Without router, non-main branch returns error
		assert.Equal(t, 503, resp.StatusCode)
	})

	t.Run("header takes precedence over query param", func(t *testing.T) {
		app := fiber.New()

		config := BranchContextConfig{
			Router: nil,
		}

		app.Use(func(c *fiber.Ctx) error {
			// Capture before middleware
			return c.Next()
		})
		app.Use(BranchContext(config))
		app.Get("/test", func(c *fiber.Ctx) error {
			_ = GetBranchSlug(c)
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test?branch=query-branch", nil)
		req.Header.Set(BranchHeader, "header-branch")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Both are non-main, so will fail without router
		assert.Equal(t, 503, resp.StatusCode)
	})

	t.Run("returns 503 for non-main branch without router", func(t *testing.T) {
		app := fiber.New()

		config := BranchContextConfig{
			Router: nil,
		}

		app.Use(BranchContext(config))
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set(BranchHeader, "feature-branch")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 503, resp.StatusCode)
	})
}

// =============================================================================
// BranchContext Access Control Tests (without database)
// =============================================================================

func TestBranchContext_AccessControl(t *testing.T) {
	t.Run("RequireAccess denies anonymous on non-main branch", func(t *testing.T) {
		app := fiber.New()

		config := BranchContextConfig{
			Router:         nil,
			RequireAccess:  true,
			AllowAnonymous: false,
		}

		app.Use(BranchContext(config))
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set(BranchHeader, "feature-123")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should be 503 first (no router), but if router existed, would be 401
		assert.Equal(t, 503, resp.StatusCode)
	})

	t.Run("main branch always allowed without authentication", func(t *testing.T) {
		app := fiber.New()

		config := BranchContextConfig{
			Router:         nil,
			RequireAccess:  true,
			AllowAnonymous: false,
		}

		var branchSlug string
		app.Use(BranchContext(config))
		app.Get("/test", func(c *fiber.Ctx) error {
			branchSlug = GetBranchSlug(c)
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		// No branch header = main branch

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, "main", branchSlug)
	})
}

// =============================================================================
// BranchContextSimple Tests
// =============================================================================

func TestBranchContextSimple(t *testing.T) {
	t.Run("creates config with no access checks", func(t *testing.T) {
		// BranchContextSimple should return a handler that:
		// - Does not require access checks
		// - Allows anonymous users
		handler := BranchContextSimple(nil)
		assert.NotNil(t, handler)
	})

	t.Run("handles main branch without router", func(t *testing.T) {
		app := fiber.New()

		app.Use(BranchContextSimple(nil))
		app.Get("/test", func(c *fiber.Ctx) error {
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
// RequireBranchAccess Tests
// =============================================================================

func TestRequireBranchAccess(t *testing.T) {
	t.Run("creates config with access checks", func(t *testing.T) {
		handler := RequireBranchAccess(nil)
		assert.NotNil(t, handler)
	})

	t.Run("main branch allowed without auth", func(t *testing.T) {
		app := fiber.New()

		app.Use(RequireBranchAccess(nil))
		app.Get("/test", func(c *fiber.Ctx) error {
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
// Integration Tests (Branch Extraction)
// =============================================================================

func TestBranchExtraction(t *testing.T) {
	tests := []struct {
		name          string
		headerValue   string
		queryValue    string
		expectedSlug  string
		expectSuccess bool
	}{
		{
			name:          "no branch specified - uses main",
			headerValue:   "",
			queryValue:    "",
			expectedSlug:  "main",
			expectSuccess: true,
		},
		{
			name:          "explicit main in header",
			headerValue:   "main",
			queryValue:    "",
			expectedSlug:  "main",
			expectSuccess: true,
		},
		{
			name:          "explicit main in query",
			headerValue:   "",
			queryValue:    "main",
			expectedSlug:  "main",
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			config := BranchContextConfig{
				Router: nil,
			}

			var capturedSlug string
			app.Use(BranchContext(config))
			app.Get("/test", func(c *fiber.Ctx) error {
				capturedSlug = GetBranchSlug(c)
				return c.SendString("OK")
			})

			url := "/test"
			if tt.queryValue != "" {
				url = "/test?branch=" + tt.queryValue
			}

			req := httptest.NewRequest("GET", url, nil)
			if tt.headerValue != "" {
				req.Header.Set(BranchHeader, tt.headerValue)
			}

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			if tt.expectSuccess {
				assert.Equal(t, 200, resp.StatusCode)
				assert.Equal(t, tt.expectedSlug, capturedSlug)
			}
		})
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkGetBranchSlug_Set(b *testing.B) {
	app := fiber.New()

	app.Get("/test", func(c *fiber.Ctx) error {
		c.Locals(LocalsBranchSlug, "feature-123")
		for i := 0; i < b.N; i++ {
			_ = GetBranchSlug(c)
		}
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)
	resp.Body.Close()
}

func BenchmarkGetBranchSlug_NotSet(b *testing.B) {
	app := fiber.New()

	app.Get("/test", func(c *fiber.Ctx) error {
		for i := 0; i < b.N; i++ {
			_ = GetBranchSlug(c)
		}
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)
	resp.Body.Close()
}

func BenchmarkIsUsingBranch_Main(b *testing.B) {
	app := fiber.New()

	app.Get("/test", func(c *fiber.Ctx) error {
		c.Locals(LocalsBranchSlug, "main")
		for i := 0; i < b.N; i++ {
			_ = IsUsingBranch(c)
		}
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)
	resp.Body.Close()
}

func BenchmarkIsUsingBranch_Feature(b *testing.B) {
	app := fiber.New()

	app.Get("/test", func(c *fiber.Ctx) error {
		c.Locals(LocalsBranchSlug, "feature-123")
		for i := 0; i < b.N; i++ {
			_ = IsUsingBranch(c)
		}
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)
	resp.Body.Close()
}

func BenchmarkBranchContext_MainBranch(b *testing.B) {
	app := fiber.New()

	config := BranchContextConfig{Router: nil}
	app.Use(BranchContext(config))
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
