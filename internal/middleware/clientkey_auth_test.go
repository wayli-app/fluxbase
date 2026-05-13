package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/keys"
)

// =============================================================================
// RequireScope Tests
// =============================================================================

func TestRequireScope_ClientKeyWithAllScopes(t *testing.T) {
	app := fiber.New()

	// Set up middleware chain
	app.Use(func(c fiber.Ctx) error {
		c.Locals("auth_type", "clientkey")
		c.Locals("client_key_scopes", []string{"read", "write", "delete"})
		return c.Next()
	})
	app.Use(RequireScope("read", "write"))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRequireScope_ClientKeyWithWildcard(t *testing.T) {
	app := fiber.New()

	app.Use(func(c fiber.Ctx) error {
		c.Locals("auth_type", "clientkey")
		c.Locals("client_key_scopes", []string{"*"})
		return c.Next()
	})
	app.Use(RequireScope("read", "write", "admin"))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRequireScope_ClientKeyMissingScope(t *testing.T) {
	app := fiber.New()

	app.Use(func(c fiber.Ctx) error {
		c.Locals("auth_type", "clientkey")
		c.Locals("client_key_scopes", []string{"read"})
		return c.Next()
	})
	app.Use(RequireScope("read", "write"))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Insufficient permissions")
	assert.Contains(t, string(body), "write")
}

func TestRequireScope_ClientKeyNoScopes(t *testing.T) {
	app := fiber.New()

	app.Use(func(c fiber.Ctx) error {
		c.Locals("auth_type", "clientkey")
		// No scopes set
		return c.Next()
	})
	app.Use(RequireScope("read"))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestRequireScope_ServiceKeyWithAllScopes(t *testing.T) {
	app := fiber.New()

	app.Use(func(c fiber.Ctx) error {
		c.Locals("auth_type", "service_key")
		c.Locals("service_key_scopes", []string{"api:read", "api:write"})
		return c.Next()
	})
	app.Use(RequireScope("api:read"))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRequireScope_ServiceKeyMissingScope(t *testing.T) {
	app := fiber.New()

	app.Use(func(c fiber.Ctx) error {
		c.Locals("auth_type", "service_key")
		c.Locals("service_key_scopes", []string{"api:read"})
		return c.Next()
	})
	app.Use(RequireScope("api:admin"))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestRequireScope_JWTAuthAllowed(t *testing.T) {
	// JWT auth doesn't use scopes yet, so should be allowed through
	app := fiber.New()

	app.Use(func(c fiber.Ctx) error {
		c.Locals("auth_type", "jwt")
		c.Locals("user_id", "user-123")
		return c.Next()
	})
	app.Use(RequireScope("read"))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRequireScope_NoAuthType(t *testing.T) {
	// If no auth_type is set, should pass through (no scopes to check)
	app := fiber.New()

	app.Use(RequireScope("read"))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// =============================================================================
// RequireAdmin Tests
// =============================================================================

func TestRequireAdmin_ServiceKey(t *testing.T) {
	app := fiber.New()

	app.Use(func(c fiber.Ctx) error {
		c.Locals("auth_type", "service_key")
		c.Locals("user_role", "service_role")
		return c.Next()
	})
	app.Use(RequireAdmin())
	app.Get("/admin", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRequireAdmin_ServiceRoleJWT(t *testing.T) {
	app := fiber.New()

	app.Use(func(c fiber.Ctx) error {
		c.Locals("auth_type", "service_role_jwt")
		c.Locals("user_role", "service_role")
		return c.Next()
	})
	app.Use(RequireAdmin())
	app.Get("/admin", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRequireAdmin_DashboardAdmin(t *testing.T) {
	app := fiber.New()

	app.Use(func(c fiber.Ctx) error {
		c.Locals("auth_type", "jwt")
		c.Locals("user_role", "instance_admin")
		return c.Next()
	})
	app.Use(RequireAdmin())
	app.Get("/admin", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRequireAdmin_RegularUser(t *testing.T) {
	app := fiber.New()

	app.Use(func(c fiber.Ctx) error {
		c.Locals("auth_type", "jwt")
		c.Locals("user_role", "authenticated")
		return c.Next()
	})
	app.Use(RequireAdmin())
	app.Get("/admin", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Admin role required")
}

func TestRequireAdmin_AnonUser(t *testing.T) {
	app := fiber.New()

	app.Use(func(c fiber.Ctx) error {
		c.Locals("auth_type", "service_role_jwt")
		c.Locals("user_role", "anon")
		return c.Next()
	})
	app.Use(RequireAdmin())
	app.Get("/admin", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestRequireAdmin_NoAuth(t *testing.T) {
	app := fiber.New()

	// No auth locals set
	app.Use(RequireAdmin())
	app.Get("/admin", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

// =============================================================================
// Context Locals Tests
// =============================================================================

func TestContextLocals_ClientKeyInfo(t *testing.T) {
	app := fiber.New()

	// Simulate authenticated client key
	app.Use(func(c fiber.Ctx) error {
		c.Locals("client_key_id", "ck-123")
		c.Locals("client_key_name", "Test Key")
		c.Locals("client_key_scopes", []string{"read", "write"})
		c.Locals("auth_type", "clientkey")
		c.Locals("user_id", "user-456")
		return c.Next()
	})
	app.Get("/test", func(c fiber.Ctx) error {
		keyID := c.Locals("client_key_id").(string)
		keyName := c.Locals("client_key_name").(string)
		scopes := c.Locals("client_key_scopes").([]string)
		userID := c.Locals("user_id").(string)

		return c.JSON(fiber.Map{
			"key_id":   keyID,
			"key_name": keyName,
			"scopes":   scopes,
			"user_id":  userID,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestContextLocals_JWTInfo(t *testing.T) {
	app := fiber.New()

	// Simulate authenticated JWT
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user_id", "user-123")
		c.Locals("user_email", "test@example.com")
		c.Locals("user_role", "authenticated")
		c.Locals("session_id", "session-456")
		c.Locals("auth_type", "jwt")
		c.Locals("is_anonymous", false)
		return c.Next()
	})
	app.Get("/test", func(c fiber.Ctx) error {
		userID := c.Locals("user_id").(string)
		email := c.Locals("user_email").(string)
		role := c.Locals("user_role").(string)
		sessionID := c.Locals("session_id").(string)
		isAnon := c.Locals("is_anonymous").(bool)

		return c.JSON(fiber.Map{
			"user_id":      userID,
			"email":        email,
			"role":         role,
			"session_id":   sessionID,
			"is_anonymous": isAnon,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// =============================================================================
// Header Parsing Tests
// =============================================================================

func TestHeaderParsing_BearerToken(t *testing.T) {
	app := fiber.New()

	app.Get("/test", func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")

		return c.JSON(fiber.Map{
			"auth_header": authHeader,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer test-token-123")
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHeaderParsing_XClientKey(t *testing.T) {
	app := fiber.New()

	app.Get("/test", func(c fiber.Ctx) error {
		clientKey := c.Get("X-Client-Key")

		return c.JSON(fiber.Map{
			"client_key": clientKey,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Client-Key", "ck_test_12345")
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHeaderParsing_XServiceKey(t *testing.T) {
	app := fiber.New()

	app.Get("/test", func(c fiber.Ctx) error {
		serviceKey := c.Get("X-Service-Key")

		return c.JSON(fiber.Map{
			"service_key": serviceKey,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Service-Key", "sk_test_12345")
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// =============================================================================
// AllowedNamespaces Tests
// =============================================================================

func TestAllowedNamespaces_Set(t *testing.T) {
	app := fiber.New()

	app.Use(func(c fiber.Ctx) error {
		c.Locals("allowed_namespaces", []string{"ns1", "ns2"})
		return c.Next()
	})
	app.Get("/test", func(c fiber.Ctx) error {
		namespaces := c.Locals("allowed_namespaces").([]string)
		return c.JSON(fiber.Map{
			"namespaces": namespaces,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAllowedNamespaces_NotSet(t *testing.T) {
	app := fiber.New()

	app.Get("/test", func(c fiber.Ctx) error {
		namespaces := c.Locals("allowed_namespaces")
		if namespaces == nil {
			return c.JSON(fiber.Map{
				"namespaces": "all_allowed",
			})
		}
		return c.JSON(fiber.Map{
			"namespaces": namespaces,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// =============================================================================
// RLS Context Tests
// =============================================================================

func TestRLSContext_ServiceRole(t *testing.T) {
	app := fiber.New()

	app.Use(func(c fiber.Ctx) error {
		c.Locals("rls_role", "service_role")
		c.Locals("rls_user_id", nil)
		return c.Next()
	})
	app.Get("/test", func(c fiber.Ctx) error {
		role := c.Locals("rls_role").(string)
		userID := c.Locals("rls_user_id")

		return c.JSON(fiber.Map{
			"rls_role":    role,
			"rls_user_id": userID,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRLSContext_AuthenticatedUser(t *testing.T) {
	app := fiber.New()

	app.Use(func(c fiber.Ctx) error {
		c.Locals("rls_role", "authenticated")
		c.Locals("rls_user_id", "user-123")
		return c.Next()
	})
	app.Get("/test", func(c fiber.Ctx) error {
		role := c.Locals("rls_role").(string)
		userID := c.Locals("rls_user_id").(string)

		return c.JSON(fiber.Map{
			"rls_role":    role,
			"rls_user_id": userID,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkRequireScope_SingleScope(b *testing.B) {
	app := fiber.New()

	app.Use(func(c fiber.Ctx) error {
		c.Locals("auth_type", "clientkey")
		c.Locals("client_key_scopes", []string{"read", "write", "delete"})
		return c.Next()
	})
	app.Use(RequireScope("read"))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = app.Test(req)
	}
}

func BenchmarkRequireScope_MultipleScopes(b *testing.B) {
	app := fiber.New()

	app.Use(func(c fiber.Ctx) error {
		c.Locals("auth_type", "clientkey")
		c.Locals("client_key_scopes", []string{"read", "write", "delete", "admin"})
		return c.Next()
	})
	app.Use(RequireScope("read", "write", "admin"))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = app.Test(req)
	}
}

func BenchmarkRequireAdmin(b *testing.B) {
	app := fiber.New()

	app.Use(func(c fiber.Ctx) error {
		c.Locals("auth_type", "service_key")
		c.Locals("user_role", "service_role")
		return c.Next()
	})
	app.Use(RequireAdmin())
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = app.Test(req)
	}
}

// =============================================================================
// mapKeyTypetoRole Tests
// =============================================================================

func TestMapKeyTypetoRole(t *testing.T) {
	tests := []struct {
		name     string
		keyType  string
		expected string
	}{
		{"anon constant", keys.KeyTypeAnon, "anon"},
		{"tenant_service constant", keys.KeyTypeTenantService, "tenant_service"},
		{"global_service constant", keys.KeyTypeGlobalService, "service_role"},
		{"publishable constant", keys.KeyTypePublishable, "authenticated"},
		{"legacy service type", "service", "service_role"},
		{"unknown type defaults to anon", "unknown", "anon"},
		{"empty type defaults to anon", "", "anon"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapKeyTypetoRole(tt.keyType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Key Prefix Routing Tests
// =============================================================================

func TestKeyPrefixRouting_PlatformKeys(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
	}{
		{"tenant_service prefix", keys.KeyPrefixTenantService},
		{"anon prefix", keys.KeyPrefixAnon},
		{"global_service prefix", keys.KeyPrefixGlobalService},
		{"publishable prefix", keys.KeyPrefixPublishable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := tt.prefix + "abcdefgh1234567890abcdefghijklmnop"
			extracted := keys.ExtractPrefix(key)
			assert.Equal(t, tt.prefix, extracted, "ExtractPrefix should return the correct prefix")
			assert.True(t, len(key) >= 8, "Key should be at least 8 chars")
		})
	}
}

func TestKeyPrefixRouting_LegacyKeys(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		isSk    bool
		isFbKey bool
	}{
		{"sk_ prefix is legacy", "sk_test_1234567890abcdef", true, false},
		{"pk_ prefix is legacy", "pk_live_1234567890abcdef", false, false},
		{"fb_tsk_ is platform key", "fb_tsk_1234567890abcdefghij", false, true},
		{"fb_anon_ is platform key", "fb_anon_1234567890abcdefghij", false, true},
		{"fb_gsk_ is platform key", "fb_gsk_1234567890abcdefghij", false, true},
		{"random prefix is unrecognized", "xx_unknown_1234567890", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isSk, len(tt.key) >= 2 && tt.key[:3] == "sk_")
			extracted := keys.ExtractPrefix(tt.key)
			assert.Equal(t, tt.isFbKey, extracted != "", "ExtractPrefix should detect platform keys")
		})
	}
}

// =============================================================================
// min() helper Tests
// =============================================================================

func TestMin(t *testing.T) {
	assert.Equal(t, 3, min(3, 5))
	assert.Equal(t, 2, min(10, 2))
	assert.Equal(t, 5, min(5, 5))
	assert.Equal(t, 0, min(0, 10))
}
