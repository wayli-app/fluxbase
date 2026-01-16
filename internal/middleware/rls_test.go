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
// RLSConfig Tests
// =============================================================================

func TestRLSConfig_Fields(t *testing.T) {
	config := RLSConfig{
		DB:               nil,
		SessionVarPrefix: "custom_app",
	}

	assert.Equal(t, "custom_app", config.SessionVarPrefix)
	assert.Nil(t, config.DB)
}

func TestRLSConfig_DefaultValues(t *testing.T) {
	config := RLSConfig{}

	assert.Empty(t, config.SessionVarPrefix)
	assert.Nil(t, config.DB)
}

// =============================================================================
// RLSContext Tests
// =============================================================================

func TestRLSContext_Struct(t *testing.T) {
	rlsCtx := RLSContext{
		UserID: "user-123",
		Role:   "authenticated",
	}

	assert.Equal(t, "user-123", rlsCtx.UserID)
	assert.Equal(t, "authenticated", rlsCtx.Role)
}

func TestRLSContext_NilUserID(t *testing.T) {
	rlsCtx := RLSContext{
		UserID: nil,
		Role:   "anon",
	}

	assert.Nil(t, rlsCtx.UserID)
	assert.Equal(t, "anon", rlsCtx.Role)
}

// =============================================================================
// mapAppRoleToDatabaseRole Tests
// =============================================================================

func TestMapAppRoleToDatabaseRole_ServiceRole(t *testing.T) {
	tests := []struct {
		appRole    string
		expectedDB string
	}{
		{"service_role", "service_role"},
		{"dashboard_admin", "service_role"},
	}

	for _, tt := range tests {
		t.Run(tt.appRole, func(t *testing.T) {
			result := mapAppRoleToDatabaseRole(tt.appRole)
			assert.Equal(t, tt.expectedDB, result)
		})
	}
}

func TestMapAppRoleToDatabaseRole_Anon(t *testing.T) {
	tests := []struct {
		appRole    string
		expectedDB string
	}{
		{"anon", "anon"},
		{"", "anon"},
	}

	for _, tt := range tests {
		t.Run("role_"+tt.appRole, func(t *testing.T) {
			result := mapAppRoleToDatabaseRole(tt.appRole)
			assert.Equal(t, tt.expectedDB, result)
		})
	}
}

func TestMapAppRoleToDatabaseRole_Authenticated(t *testing.T) {
	tests := []string{
		"authenticated",
		"admin",
		"user",
		"moderator",
		"editor",
		"viewer",
		"custom_role",
	}

	for _, appRole := range tests {
		t.Run(appRole, func(t *testing.T) {
			result := mapAppRoleToDatabaseRole(appRole)
			assert.Equal(t, "authenticated", result)
		})
	}
}

// =============================================================================
// splitTableName Tests
// =============================================================================

func TestSplitTableName_WithSchema(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"public.users", []string{"public", "users"}},
		{"auth.sessions", []string{"auth", "sessions"}},
		{"storage.buckets", []string{"storage", "buckets"}},
		{"my_schema.my_table", []string{"my_schema", "my_table"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := splitTableName(tt.input)
			assert.Equal(t, tt.expected, result)
			assert.Len(t, result, 2)
		})
	}
}

func TestSplitTableName_NoSchema(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"users", []string{"users"}},
		{"my_table", []string{"my_table"}},
		{"table_name_with_underscores", []string{"table_name_with_underscores"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := splitTableName(tt.input)
			assert.Equal(t, tt.expected, result)
			assert.Len(t, result, 1)
		})
	}
}

func TestSplitTableName_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty string", "", []string{""}},
		{"dot at start", ".table", []string{".table"}}, // dotIndex=0, not > 0, so returns whole string
		{"multiple dots", "a.b.c", []string{"a", "b.c"}}, // Only splits on first dot
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitTableName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// GetRLSContext Tests
// =============================================================================

func TestGetRLSContext_WithUserAndRole(t *testing.T) {
	app := fiber.New()

	app.Get("/test", func(c *fiber.Ctx) error {
		c.Locals("rls_user_id", "user-123")
		c.Locals("rls_role", "authenticated")

		rlsCtx := GetRLSContext(c)

		assert.Equal(t, "user-123", rlsCtx.UserID)
		assert.Equal(t, "authenticated", rlsCtx.Role)

		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestGetRLSContext_NilRole_DefaultsToAnon(t *testing.T) {
	app := fiber.New()

	app.Get("/test", func(c *fiber.Ctx) error {
		c.Locals("rls_user_id", nil)
		// rls_role not set

		rlsCtx := GetRLSContext(c)

		assert.Nil(t, rlsCtx.UserID)
		assert.Equal(t, "anon", rlsCtx.Role)

		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestGetRLSContext_ServiceRole(t *testing.T) {
	app := fiber.New()

	app.Get("/test", func(c *fiber.Ctx) error {
		c.Locals("rls_role", "service_role")

		rlsCtx := GetRLSContext(c)

		assert.Equal(t, "service_role", rlsCtx.Role)

		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

// =============================================================================
// RLSMiddleware Tests
// =============================================================================

func TestRLSMiddleware_DefaultPrefix(t *testing.T) {
	config := RLSConfig{} // Empty prefix should default to "app"
	middleware := RLSMiddleware(config)

	assert.NotNil(t, middleware)
}

func TestRLSMiddleware_Anonymous(t *testing.T) {
	app := fiber.New()

	config := RLSConfig{SessionVarPrefix: "app"}
	app.Use(RLSMiddleware(config))

	app.Get("/test", func(c *fiber.Ctx) error {
		// user_id is not set, should be anonymous
		rlsCtx := GetRLSContext(c)
		assert.Equal(t, "anon", rlsCtx.Role)
		assert.Nil(t, rlsCtx.UserID)
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestRLSMiddleware_Authenticated(t *testing.T) {
	app := fiber.New()

	// Set user_id before RLS middleware
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", "user-123")
		return c.Next()
	})

	config := RLSConfig{SessionVarPrefix: "app"}
	app.Use(RLSMiddleware(config))

	app.Get("/test", func(c *fiber.Ctx) error {
		rlsCtx := GetRLSContext(c)
		assert.Equal(t, "authenticated", rlsCtx.Role)
		assert.Equal(t, "user-123", rlsCtx.UserID)
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestRLSMiddleware_PreservesExistingRole(t *testing.T) {
	app := fiber.New()

	// Pre-set rls_role (e.g., from service key auth)
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("rls_role", "service_role")
		return c.Next()
	})

	config := RLSConfig{SessionVarPrefix: "app"}
	app.Use(RLSMiddleware(config))

	app.Get("/test", func(c *fiber.Ctx) error {
		rlsCtx := GetRLSContext(c)
		assert.Equal(t, "service_role", rlsCtx.Role)
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestRLSMiddleware_WithUserRole(t *testing.T) {
	app := fiber.New()

	// Set both user_id and user_role
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", "admin-123")
		c.Locals("user_role", "admin")
		return c.Next()
	})

	config := RLSConfig{SessionVarPrefix: "app"}
	app.Use(RLSMiddleware(config))

	app.Get("/test", func(c *fiber.Ctx) error {
		rlsCtx := GetRLSContext(c)
		// user_role should override default "authenticated"
		assert.Equal(t, "admin", rlsCtx.Role)
		assert.Equal(t, "admin-123", rlsCtx.UserID)
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestRLSMiddleware_EmptyUserRoleNotOverwrite(t *testing.T) {
	app := fiber.New()

	// Set user_id but empty user_role
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", "user-123")
		c.Locals("user_role", "") // Empty string should NOT overwrite "authenticated"
		return c.Next()
	})

	config := RLSConfig{SessionVarPrefix: "app"}
	app.Use(RLSMiddleware(config))

	app.Get("/test", func(c *fiber.Ctx) error {
		rlsCtx := GetRLSContext(c)
		// Empty user_role should keep "authenticated"
		assert.Equal(t, "authenticated", rlsCtx.Role)
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestRLSMiddleware_CallsNext(t *testing.T) {
	app := fiber.New()

	config := RLSConfig{SessionVarPrefix: "app"}
	app.Use(RLSMiddleware(config))

	handlerCalled := false
	app.Get("/test", func(c *fiber.Ctx) error {
		handlerCalled = true
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.True(t, handlerCalled)

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "OK", string(body))
}

// =============================================================================
// Role Mapping Security Tests
// =============================================================================

func TestMapAppRoleToDatabaseRole_SQLInjection(t *testing.T) {
	// Test that role mapping doesn't allow SQL injection via role names
	maliciousRoles := []string{
		"admin'; DROP TABLE users;--",
		"authenticated OR 1=1",
		"service_role\n--",
		"anon\x00malicious",
	}

	for _, role := range maliciousRoles {
		t.Run("malicious_role", func(t *testing.T) {
			result := mapAppRoleToDatabaseRole(role)
			// Any unrecognized role should map to "authenticated"
			assert.Equal(t, "authenticated", result)
		})
	}
}

func TestMapAppRoleToDatabaseRole_ValidRolesOnly(t *testing.T) {
	// Only these three roles should be returned
	validDBRoles := map[string]bool{
		"anon":          true,
		"authenticated": true,
		"service_role":  true,
	}

	testRoles := []string{
		"service_role",
		"dashboard_admin",
		"anon",
		"",
		"authenticated",
		"admin",
		"user",
		"moderator",
		"arbitrary_role",
	}

	for _, role := range testRoles {
		t.Run(role, func(t *testing.T) {
			result := mapAppRoleToDatabaseRole(role)
			assert.True(t, validDBRoles[result], "Result %q is not a valid database role", result)
		})
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkMapAppRoleToDatabaseRole(b *testing.B) {
	roles := []string{
		"service_role",
		"dashboard_admin",
		"anon",
		"authenticated",
		"admin",
		"user",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, role := range roles {
			_ = mapAppRoleToDatabaseRole(role)
		}
	}
}

func BenchmarkSplitTableName(b *testing.B) {
	tables := []string{
		"public.users",
		"auth.sessions",
		"users",
		"schema.table_name",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, table := range tables {
			_ = splitTableName(table)
		}
	}
}

func BenchmarkGetRLSContext(b *testing.B) {
	app := fiber.New()
	var captured *fiber.Ctx

	app.Get("/test", func(c *fiber.Ctx) error {
		c.Locals("rls_user_id", "user-123")
		c.Locals("rls_role", "authenticated")
		captured = c
		return nil
	})

	req := httptest.NewRequest("GET", "/test", nil)
	_, _ = app.Test(req)

	if captured == nil {
		b.Fatal("Failed to capture context")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetRLSContext(captured)
	}
}

func BenchmarkRLSMiddleware(b *testing.B) {
	app := fiber.New()

	config := RLSConfig{SessionVarPrefix: "app"}
	app.Use(RLSMiddleware(config))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		_, _ = app.Test(req)
	}
}
