package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/database"
)

func TestErrTenantNotFound(t *testing.T) {
	assert.EqualError(t, ErrTenantNotFound, "tenant not found")
}

func TestErrNotTenantMember(t *testing.T) {
	assert.EqualError(t, ErrNotTenantMember, "user is not a member of this tenant")
}

func TestTenantConfig_Fields(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		cfg := TenantConfig{}
		assert.Nil(t, cfg.DB)
		assert.Nil(t, cfg.ConfigLoader)
	})

	t.Run("config with loader", func(t *testing.T) {
		cfg := TenantConfig{
			ConfigLoader: &config.TenantConfigLoader{},
		}
		assert.Nil(t, cfg.DB)
		assert.NotNil(t, cfg.ConfigLoader)
	})
}

func TestGetTenantIDFromContext(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(c fiber.Ctx)
		expected string
	}{
		{
			name:     "returns empty when not set",
			setup:    func(c fiber.Ctx) {},
			expected: "",
		},
		{
			name: "returns tenant ID when set",
			setup: func(c fiber.Ctx) {
				c.Locals("tenant_id", "t-123")
			},
			expected: "t-123",
		},
		{
			name: "returns empty when wrong type",
			setup: func(c fiber.Ctx) {
				c.Locals("tenant_id", 12345)
			},
			expected: "",
		},
		{
			name: "returns UUID tenant ID",
			setup: func(c fiber.Ctx) {
				c.Locals("tenant_id", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
			},
			expected: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			var result string
			app.Get("/test", func(c fiber.Ctx) error {
				tt.setup(c)
				result = GetTenantIDFromContext(c)
				return c.SendString("OK")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetTenantSourceFromContext(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(c fiber.Ctx)
		expected string
	}{
		{
			name:     "returns empty when not set",
			setup:    func(c fiber.Ctx) {},
			expected: "",
		},
		{
			name: "returns header source",
			setup: func(c fiber.Ctx) {
				c.Locals("tenant_source", "header")
			},
			expected: "header",
		},
		{
			name: "returns jwt source",
			setup: func(c fiber.Ctx) {
				c.Locals("tenant_source", "jwt")
			},
			expected: "jwt",
		},
		{
			name: "returns default source",
			setup: func(c fiber.Ctx) {
				c.Locals("tenant_source", "default")
			},
			expected: "default",
		},
		{
			name: "returns empty when wrong type",
			setup: func(c fiber.Ctx) {
				c.Locals("tenant_source", 42)
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			var result string
			app.Get("/test", func(c fiber.Ctx) error {
				tt.setup(c)
				result = GetTenantSourceFromContext(c)
				return c.SendString("OK")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetTenantRoleFromContext(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(c fiber.Ctx)
		expected string
	}{
		{
			name:     "returns empty when not set",
			setup:    func(c fiber.Ctx) {},
			expected: "",
		},
		{
			name: "returns tenant_admin role",
			setup: func(c fiber.Ctx) {
				c.Locals("tenant_role", "tenant_admin")
			},
			expected: "tenant_admin",
		},
		{
			name: "returns instance_admin role",
			setup: func(c fiber.Ctx) {
				c.Locals("tenant_role", "instance_admin")
			},
			expected: "instance_admin",
		},
		{
			name: "returns tenant_service role",
			setup: func(c fiber.Ctx) {
				c.Locals("tenant_role", "tenant_service")
			},
			expected: "tenant_service",
		},
		{
			name: "returns empty when wrong type",
			setup: func(c fiber.Ctx) {
				c.Locals("tenant_role", true)
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			var result string
			app.Get("/test", func(c fiber.Ctx) error {
				tt.setup(c)
				result = GetTenantRoleFromContext(c)
				return c.SendString("OK")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsInstanceAdminFromContext(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(c fiber.Ctx)
		expected bool
	}{
		{
			name:     "returns false when not set",
			setup:    func(c fiber.Ctx) {},
			expected: false,
		},
		{
			name: "returns true when set to true",
			setup: func(c fiber.Ctx) {
				c.Locals("is_instance_admin", true)
			},
			expected: true,
		},
		{
			name: "returns false when set to false",
			setup: func(c fiber.Ctx) {
				c.Locals("is_instance_admin", false)
			},
			expected: false,
		},
		{
			name: "returns false when wrong type",
			setup: func(c fiber.Ctx) {
				c.Locals("is_instance_admin", "yes")
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			var result bool
			app.Get("/test", func(c fiber.Ctx) error {
				tt.setup(c)
				result = IsInstanceAdminFromContext(c)
				return c.SendString("OK")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsDefaultTenantFromContext(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(c fiber.Ctx)
		expected bool
	}{
		{
			name:     "returns false when not set",
			setup:    func(c fiber.Ctx) {},
			expected: false,
		},
		{
			name: "returns true for default tenant",
			setup: func(c fiber.Ctx) {
				c.Locals("is_default_tenant", true)
			},
			expected: true,
		},
		{
			name: "returns false for non-default tenant",
			setup: func(c fiber.Ctx) {
				c.Locals("is_default_tenant", false)
			},
			expected: false,
		},
		{
			name: "returns false when wrong type",
			setup: func(c fiber.Ctx) {
				c.Locals("is_default_tenant", "true")
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			var result bool
			app.Get("/test", func(c fiber.Ctx) error {
				tt.setup(c)
				result = IsDefaultTenantFromContext(c)
				return c.SendString("OK")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRequireTenantRole(t *testing.T) {
	t.Run("instance admin without tenant context passes", func(t *testing.T) {
		app := fiber.New()
		app.Use(func(c fiber.Ctx) error {
			c.Locals("is_instance_admin", true)
			return c.Next()
		})
		app.Use(RequireTenantRole("tenant_admin"))
		app.Get("/test", func(c fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("instance admin with default source passes", func(t *testing.T) {
		app := fiber.New()
		app.Use(func(c fiber.Ctx) error {
			c.Locals("is_instance_admin", true)
			c.Locals("tenant_id", "default-id")
			c.Locals("tenant_source", "default")
			return c.Next()
		})
		app.Use(RequireTenantRole("tenant_admin"))
		app.Get("/test", func(c fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("instance admin with header source needs role", func(t *testing.T) {
		app := fiber.New()
		app.Use(func(c fiber.Ctx) error {
			c.Locals("is_instance_admin", true)
			c.Locals("tenant_id", "t-123")
			c.Locals("tenant_source", "header")
			return c.Next()
		})
		app.Use(RequireTenantRole("tenant_admin"))
		app.Get("/test", func(c fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "tenant membership required")
	})

	t.Run("instance admin with jwt source needs role", func(t *testing.T) {
		app := fiber.New()
		app.Use(func(c fiber.Ctx) error {
			c.Locals("is_instance_admin", true)
			c.Locals("tenant_id", "t-123")
			c.Locals("tenant_source", "jwt")
			return c.Next()
		})
		app.Use(RequireTenantRole("tenant_admin"))
		app.Get("/test", func(c fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("tenant admin with header source passes", func(t *testing.T) {
		app := fiber.New()
		app.Use(func(c fiber.Ctx) error {
			c.Locals("is_instance_admin", false)
			c.Locals("tenant_id", "t-123")
			c.Locals("tenant_source", "header")
			c.Locals("tenant_role", "tenant_admin")
			return c.Next()
		})
		app.Use(RequireTenantRole("tenant_admin"))
		app.Get("/test", func(c fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("tenant admin passes regardless of required role", func(t *testing.T) {
		app := fiber.New()
		app.Use(func(c fiber.Ctx) error {
			c.Locals("is_instance_admin", false)
			c.Locals("tenant_id", "t-123")
			c.Locals("tenant_source", "jwt")
			c.Locals("tenant_role", "tenant_admin")
			return c.Next()
		})
		app.Use(RequireTenantRole("some_other_role"))
		app.Get("/test", func(c fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("matching role passes", func(t *testing.T) {
		app := fiber.New()
		app.Use(func(c fiber.Ctx) error {
			c.Locals("is_instance_admin", false)
			c.Locals("tenant_id", "t-123")
			c.Locals("tenant_source", "header")
			c.Locals("tenant_role", "custom_role")
			return c.Next()
		})
		app.Use(RequireTenantRole("custom_role"))
		app.Get("/test", func(c fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("no tenant role returns forbidden", func(t *testing.T) {
		app := fiber.New()
		app.Use(func(c fiber.Ctx) error {
			c.Locals("is_instance_admin", false)
			c.Locals("tenant_id", "t-123")
			c.Locals("tenant_source", "header")
			return c.Next()
		})
		app.Use(RequireTenantRole("tenant_admin"))
		app.Get("/test", func(c fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "tenant membership required")
	})

	t.Run("wrong role returns forbidden with role name", func(t *testing.T) {
		app := fiber.New()
		app.Use(func(c fiber.Ctx) error {
			c.Locals("is_instance_admin", false)
			c.Locals("tenant_id", "t-123")
			c.Locals("tenant_source", "jwt")
			c.Locals("tenant_role", "viewer")
			return c.Next()
		})
		app.Use(RequireTenantRole("tenant_admin"))
		app.Get("/test", func(c fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "tenant_admin role required")
	})

	t.Run("no auth at all returns forbidden", func(t *testing.T) {
		app := fiber.New()
		app.Use(RequireTenantRole("tenant_admin"))
		app.Get("/test", func(c fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
}

func TestRequireInstanceAdmin(t *testing.T) {
	t.Run("instance admin without tenant context passes", func(t *testing.T) {
		app := fiber.New()
		app.Use(func(c fiber.Ctx) error {
			c.Locals("is_instance_admin", true)
			return c.Next()
		})
		app.Use(RequireInstanceAdmin())
		app.Get("/test", func(c fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("instance admin with default tenant passes", func(t *testing.T) {
		app := fiber.New()
		app.Use(func(c fiber.Ctx) error {
			c.Locals("is_instance_admin", true)
			c.Locals("tenant_id", "default-id")
			c.Locals("tenant_source", "default")
			return c.Next()
		})
		app.Use(RequireInstanceAdmin())
		app.Get("/test", func(c fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("instance admin with header tenant context is denied", func(t *testing.T) {
		app := fiber.New()
		app.Use(func(c fiber.Ctx) error {
			c.Locals("is_instance_admin", true)
			c.Locals("tenant_id", "t-123")
			c.Locals("tenant_source", "header")
			return c.Next()
		})
		app.Use(RequireInstanceAdmin())
		app.Get("/test", func(c fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "instance admin access not available when acting as tenant admin")
	})

	t.Run("instance admin with jwt tenant context is denied", func(t *testing.T) {
		app := fiber.New()
		app.Use(func(c fiber.Ctx) error {
			c.Locals("is_instance_admin", true)
			c.Locals("tenant_id", "t-123")
			c.Locals("tenant_source", "jwt")
			return c.Next()
		})
		app.Use(RequireInstanceAdmin())
		app.Get("/test", func(c fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "instance admin access not available when acting as tenant admin")
	})

	t.Run("non-instance admin is denied", func(t *testing.T) {
		app := fiber.New()
		app.Use(func(c fiber.Ctx) error {
			c.Locals("is_instance_admin", false)
			return c.Next()
		})
		app.Use(RequireInstanceAdmin())
		app.Get("/test", func(c fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "instance admin role required")
	})

	t.Run("no auth at all is denied", func(t *testing.T) {
		app := fiber.New()
		app.Use(RequireInstanceAdmin())
		app.Get("/test", func(c fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "instance admin role required")
	})

	t.Run("tenant admin without instance admin is denied", func(t *testing.T) {
		app := fiber.New()
		app.Use(func(c fiber.Ctx) error {
			c.Locals("is_instance_admin", false)
			c.Locals("tenant_id", "t-123")
			c.Locals("tenant_source", "header")
			c.Locals("tenant_role", "tenant_admin")
			return c.Next()
		})
		app.Use(RequireInstanceAdmin())
		app.Get("/test", func(c fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
}

func TestCtxWithTenant(t *testing.T) {
	t.Run("uses tenant_id from locals when source is header", func(t *testing.T) {
		app := fiber.New()

		var result string
		app.Get("/test", func(c fiber.Ctx) error {
			c.Locals("tenant_id", "header-tenant-id")
			c.Locals("tenant_source", "header")
			ctx := CtxWithTenant(c)
			result = database.TenantFromContext(ctx)
			return c.SendString("OK")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, "header-tenant-id", result)
	})

	t.Run("uses tenant_id from locals when source is jwt", func(t *testing.T) {
		app := fiber.New()

		var result string
		app.Get("/test", func(c fiber.Ctx) error {
			c.Locals("tenant_id", "jwt-tenant-id")
			c.Locals("tenant_source", "jwt")
			ctx := CtxWithTenant(c)
			result = database.TenantFromContext(ctx)
			return c.SendString("OK")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, "jwt-tenant-id", result)
	})

	t.Run("prefers JWT claims over default source", func(t *testing.T) {
		app := fiber.New()

		jwtTenantID := "claims-tenant-id"
		var result string
		app.Get("/test", func(c fiber.Ctx) error {
			c.Locals("tenant_id", "default-tenant-id")
			c.Locals("tenant_source", "default")
			c.Locals("claims", &auth.TokenClaims{TenantID: &jwtTenantID})
			ctx := CtxWithTenant(c)
			result = database.TenantFromContext(ctx)
			return c.SendString("OK")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, "claims-tenant-id", result)
	})

	t.Run("prefers jwt_claims over default source", func(t *testing.T) {
		app := fiber.New()

		jwtTenantID := "jwt-claims-tenant-id"
		var result string
		app.Get("/test", func(c fiber.Ctx) error {
			c.Locals("tenant_id", "default-tenant-id")
			c.Locals("tenant_source", "default")
			c.Locals("jwt_claims", &auth.TokenClaims{TenantID: &jwtTenantID})
			ctx := CtxWithTenant(c)
			result = database.TenantFromContext(ctx)
			return c.SendString("OK")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, "jwt-claims-tenant-id", result)
	})

	t.Run("uses claims when claims key has nil TenantID", func(t *testing.T) {
		app := fiber.New()

		var result string
		app.Get("/test", func(c fiber.Ctx) error {
			c.Locals("tenant_id", "default-tenant-id")
			c.Locals("tenant_source", "default")
			c.Locals("claims", &auth.TokenClaims{TenantID: nil})
			ctx := CtxWithTenant(c)
			result = database.TenantFromContext(ctx)
			return c.SendString("OK")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, "default-tenant-id", result)
	})

	t.Run("returns empty when no tenant set at all", func(t *testing.T) {
		app := fiber.New()

		var result string
		app.Get("/test", func(c fiber.Ctx) error {
			ctx := CtxWithTenant(c)
			result = database.TenantFromContext(ctx)
			return c.SendString("OK")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, "", result)
	})

	t.Run("prefers claims locals over jwt_claims locals", func(t *testing.T) {
		app := fiber.New()

		claimsTenantID := "claims-tenant-id"
		jwtClaimsTenantID := "jwt-claims-tenant-id"
		var result string
		app.Get("/test", func(c fiber.Ctx) error {
			c.Locals("tenant_id", "default-tenant-id")
			c.Locals("tenant_source", "default")
			c.Locals("claims", &auth.TokenClaims{TenantID: &claimsTenantID})
			c.Locals("jwt_claims", &auth.TokenClaims{TenantID: &jwtClaimsTenantID})
			ctx := CtxWithTenant(c)
			result = database.TenantFromContext(ctx)
			return c.SendString("OK")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, "claims-tenant-id", result)
	})

	t.Run("uses empty tenant when tenant_id is empty and no claims", func(t *testing.T) {
		app := fiber.New()

		var result string
		app.Get("/test", func(c fiber.Ctx) error {
			c.Locals("tenant_id", "")
			c.Locals("tenant_source", "")
			ctx := CtxWithTenant(c)
			result = database.TenantFromContext(ctx)
			return c.SendString("OK")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, "", result)
	})
}

func TestRequireTenantRole_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		isAdmin        bool
		tenantID       string
		tenantSource   string
		tenantRole     string
		requiredRole   string
		expectedStatus int
	}{
		{
			name:           "instance admin no tenant context passes",
			isAdmin:        true,
			tenantID:       "",
			tenantSource:   "",
			tenantRole:     "",
			requiredRole:   "tenant_admin",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "instance admin default source passes",
			isAdmin:        true,
			tenantID:       "default-id",
			tenantSource:   "default",
			tenantRole:     "",
			requiredRole:   "tenant_admin",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "instance admin header source without role forbidden",
			isAdmin:        true,
			tenantID:       "t-123",
			tenantSource:   "header",
			tenantRole:     "",
			requiredRole:   "tenant_admin",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "instance admin jwt source without role forbidden",
			isAdmin:        true,
			tenantID:       "t-123",
			tenantSource:   "jwt",
			tenantRole:     "",
			requiredRole:   "tenant_admin",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "tenant admin with header source passes",
			isAdmin:        false,
			tenantID:       "t-123",
			tenantSource:   "header",
			tenantRole:     "tenant_admin",
			requiredRole:   "tenant_admin",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "tenant admin passes different required role",
			isAdmin:        false,
			tenantID:       "t-123",
			tenantSource:   "header",
			tenantRole:     "tenant_admin",
			requiredRole:   "custom_role",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "matching non-admin role passes",
			isAdmin:        false,
			tenantID:       "t-123",
			tenantSource:   "jwt",
			tenantRole:     "editor",
			requiredRole:   "editor",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "non-matching role forbidden",
			isAdmin:        false,
			tenantID:       "t-123",
			tenantSource:   "header",
			tenantRole:     "viewer",
			requiredRole:   "editor",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "no role with tenant context forbidden",
			isAdmin:        false,
			tenantID:       "t-123",
			tenantSource:   "header",
			tenantRole:     "",
			requiredRole:   "tenant_admin",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "non-admin no tenant context forbidden",
			isAdmin:        false,
			tenantID:       "",
			tenantSource:   "",
			tenantRole:     "",
			requiredRole:   "tenant_admin",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "instance admin with header source and matching role passes",
			isAdmin:        true,
			tenantID:       "t-123",
			tenantSource:   "header",
			tenantRole:     "tenant_admin",
			requiredRole:   "tenant_admin",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Use(func(c fiber.Ctx) error {
				c.Locals("is_instance_admin", tt.isAdmin)
				if tt.tenantID != "" {
					c.Locals("tenant_id", tt.tenantID)
				}
				if tt.tenantSource != "" {
					c.Locals("tenant_source", tt.tenantSource)
				}
				if tt.tenantRole != "" {
					c.Locals("tenant_role", tt.tenantRole)
				}
				return c.Next()
			})
			app.Use(RequireTenantRole(tt.requiredRole))
			app.Get("/test", func(c fiber.Ctx) error {
				return c.SendString("OK")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestRequireInstanceAdmin_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		isAdmin        bool
		tenantID       string
		tenantSource   string
		expectedStatus int
	}{
		{
			name:           "instance admin no tenant context",
			isAdmin:        true,
			tenantID:       "",
			tenantSource:   "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "instance admin default source",
			isAdmin:        true,
			tenantID:       "default-id",
			tenantSource:   "default",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "instance admin header source denied",
			isAdmin:        true,
			tenantID:       "t-123",
			tenantSource:   "header",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "instance admin jwt source denied",
			isAdmin:        true,
			tenantID:       "t-123",
			tenantSource:   "jwt",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "non-instance admin denied",
			isAdmin:        false,
			tenantID:       "",
			tenantSource:   "",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "no auth denied",
			isAdmin:        false,
			tenantID:       "",
			tenantSource:   "",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "instance admin empty tenant_id ok",
			isAdmin:        true,
			tenantID:       "",
			tenantSource:   "header",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Use(func(c fiber.Ctx) error {
				c.Locals("is_instance_admin", tt.isAdmin)
				if tt.tenantID != "" {
					c.Locals("tenant_id", tt.tenantID)
				}
				if tt.tenantSource != "" {
					c.Locals("tenant_source", tt.tenantSource)
				}
				return c.Next()
			})
			app.Use(RequireInstanceAdmin())
			app.Get("/test", func(c fiber.Ctx) error {
				return c.SendString("OK")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func BenchmarkGetTenantIDFromContext_Set(b *testing.B) {
	app := fiber.New()

	app.Get("/test", func(c fiber.Ctx) error {
		c.Locals("tenant_id", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
		for i := 0; i < b.N; i++ {
			_ = GetTenantIDFromContext(c)
		}
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, _ := app.Test(req)
	_ = resp.Body.Close()
}

func BenchmarkGetTenantIDFromContext_NotSet(b *testing.B) {
	app := fiber.New()

	app.Get("/test", func(c fiber.Ctx) error {
		for i := 0; i < b.N; i++ {
			_ = GetTenantIDFromContext(c)
		}
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, _ := app.Test(req)
	_ = resp.Body.Close()
}

func BenchmarkGetTenantSourceFromContext(b *testing.B) {
	app := fiber.New()

	app.Get("/test", func(c fiber.Ctx) error {
		c.Locals("tenant_source", "header")
		for i := 0; i < b.N; i++ {
			_ = GetTenantSourceFromContext(c)
		}
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, _ := app.Test(req)
	_ = resp.Body.Close()
}

func BenchmarkGetTenantRoleFromContext(b *testing.B) {
	app := fiber.New()

	app.Get("/test", func(c fiber.Ctx) error {
		c.Locals("tenant_role", "tenant_admin")
		for i := 0; i < b.N; i++ {
			_ = GetTenantRoleFromContext(c)
		}
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, _ := app.Test(req)
	_ = resp.Body.Close()
}

func BenchmarkIsInstanceAdminFromContext(b *testing.B) {
	app := fiber.New()

	app.Get("/test", func(c fiber.Ctx) error {
		c.Locals("is_instance_admin", true)
		for i := 0; i < b.N; i++ {
			_ = IsInstanceAdminFromContext(c)
		}
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, _ := app.Test(req)
	_ = resp.Body.Close()
}

func BenchmarkIsDefaultTenantFromContext(b *testing.B) {
	app := fiber.New()

	app.Get("/test", func(c fiber.Ctx) error {
		c.Locals("is_default_tenant", true)
		for i := 0; i < b.N; i++ {
			_ = IsDefaultTenantFromContext(c)
		}
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, _ := app.Test(req)
	_ = resp.Body.Close()
}

func BenchmarkCtxWithTenant(b *testing.B) {
	app := fiber.New()

	app.Get("/test", func(c fiber.Ctx) error {
		c.Locals("tenant_id", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
		c.Locals("tenant_source", "header")
		for i := 0; i < b.N; i++ {
			_ = CtxWithTenant(c)
		}
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, _ := app.Test(req)
	_ = resp.Body.Close()
}

func BenchmarkRequireTenantRole(b *testing.B) {
	app := fiber.New()

	app.Use(func(c fiber.Ctx) error {
		c.Locals("is_instance_admin", false)
		c.Locals("tenant_id", "t-123")
		c.Locals("tenant_source", "header")
		c.Locals("tenant_role", "tenant_admin")
		return c.Next()
	})
	app.Use(RequireTenantRole("tenant_admin"))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, _ := app.Test(req)
		_ = resp.Body.Close()
	}
}

func BenchmarkRequireInstanceAdmin(b *testing.B) {
	app := fiber.New()

	app.Use(func(c fiber.Ctx) error {
		c.Locals("is_instance_admin", true)
		return c.Next()
	})
	app.Use(RequireInstanceAdmin())
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, _ := app.Test(req)
		_ = resp.Body.Close()
	}
}
