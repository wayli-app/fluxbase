package api

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"

	mw "github.com/nimbleflux/fluxbase/internal/middleware"
)

// =============================================================================
// Helper Functions for Testing
// =============================================================================

// createTestFiberCtx creates a Fiber context for testing
func createTestFiberCtx() fiber.Ctx {
	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	return ctx
}

// =============================================================================
// GetUserID Tests
// =============================================================================

func TestGetUserID(t *testing.T) {
	t.Run("returns empty when user_id not set", func(t *testing.T) {
		ctx := createTestFiberCtx()
		defer func() {
			app := ctx.App()
			app.ReleaseCtx(ctx)
		}()

		userID := mw.GetUserID(ctx)

		assert.Empty(t, userID)
	})

	t.Run("returns user_id when set as string", func(t *testing.T) {
		ctx := createTestFiberCtx()
		defer func() {
			app := ctx.App()
			app.ReleaseCtx(ctx)
		}()

		ctx.Locals("user_id", "user-123-abc")

		userID := mw.GetUserID(ctx)

		assert.Equal(t, "user-123-abc", userID)
	})

	t.Run("returns empty when user_id is wrong type", func(t *testing.T) {
		ctx := createTestFiberCtx()
		defer func() {
			app := ctx.App()
			app.ReleaseCtx(ctx)
		}()

		ctx.Locals("user_id", 12345)

		userID := mw.GetUserID(ctx)

		assert.Empty(t, userID)
	})
}

// =============================================================================
// GetUserEmail Tests
// =============================================================================

func TestGetUserEmail(t *testing.T) {
	t.Run("returns empty when user_email not set", func(t *testing.T) {
		ctx := createTestFiberCtx()
		defer func() {
			app := ctx.App()
			app.ReleaseCtx(ctx)
		}()

		email, ok := GetUserEmail(ctx)

		assert.False(t, ok)
		assert.Empty(t, email)
	})

	t.Run("returns user_email when set as string", func(t *testing.T) {
		ctx := createTestFiberCtx()
		defer func() {
			app := ctx.App()
			app.ReleaseCtx(ctx)
		}()

		ctx.Locals("user_email", "test@example.com")

		email, ok := GetUserEmail(ctx)

		assert.True(t, ok)
		assert.Equal(t, "test@example.com", email)
	})

	t.Run("returns empty when user_email is wrong type", func(t *testing.T) {
		ctx := createTestFiberCtx()
		defer func() {
			app := ctx.App()
			app.ReleaseCtx(ctx)
		}()

		ctx.Locals("user_email", []byte("test@example.com")) // Wrong type

		email, ok := GetUserEmail(ctx)

		assert.False(t, ok)
		assert.Empty(t, email)
	})
}

// =============================================================================
// GetUserRole Tests
// =============================================================================

func TestGetUserRole(t *testing.T) {
	t.Run("returns empty when user_role not set", func(t *testing.T) {
		ctx := createTestFiberCtx()
		defer func() {
			app := ctx.App()
			app.ReleaseCtx(ctx)
		}()

		role, ok := GetUserRole(ctx)

		assert.False(t, ok)
		assert.Empty(t, role)
	})

	t.Run("returns user_role when set as string", func(t *testing.T) {
		ctx := createTestFiberCtx()
		defer func() {
			app := ctx.App()
			app.ReleaseCtx(ctx)
		}()

		ctx.Locals("user_role", "admin")

		role, ok := GetUserRole(ctx)

		assert.True(t, ok)
		assert.Equal(t, "admin", role)
	})

	t.Run("returns authenticated role", func(t *testing.T) {
		ctx := createTestFiberCtx()
		defer func() {
			app := ctx.App()
			app.ReleaseCtx(ctx)
		}()

		ctx.Locals("user_role", "authenticated")

		role, ok := GetUserRole(ctx)

		assert.True(t, ok)
		assert.Equal(t, "authenticated", role)
	})

	t.Run("returns instance_admin role", func(t *testing.T) {
		ctx := createTestFiberCtx()
		defer func() {
			app := ctx.App()
			app.ReleaseCtx(ctx)
		}()

		ctx.Locals("user_role", "instance_admin")

		role, ok := GetUserRole(ctx)

		assert.True(t, ok)
		assert.Equal(t, "instance_admin", role)
	})

	t.Run("returns empty when user_role is wrong type", func(t *testing.T) {
		ctx := createTestFiberCtx()
		defer func() {
			app := ctx.App()
			app.ReleaseCtx(ctx)
		}()

		ctx.Locals("user_role", true) // Wrong type (bool instead of string)

		role, ok := GetUserRole(ctx)

		assert.False(t, ok)
		assert.Empty(t, role)
	})
}

// =============================================================================
// RequireRole Tests
// =============================================================================

func TestRequireRole(t *testing.T) {
	t.Run("middleware creation with single role", func(t *testing.T) {
		middleware := RequireRole("admin")

		require.NotNil(t, middleware)
	})

	t.Run("middleware creation with multiple roles", func(t *testing.T) {
		middleware := RequireRole("admin", "service_role", "instance_admin")

		require.NotNil(t, middleware)
	})

	t.Run("middleware creation with no roles", func(t *testing.T) {
		middleware := RequireRole()

		require.NotNil(t, middleware)
	})
}

// =============================================================================
// AuthMiddleware Creation Tests
// =============================================================================

func TestAuthMiddleware_Creation(t *testing.T) {
	t.Run("creates middleware with nil service", func(t *testing.T) {
		middleware := AuthMiddleware(nil)

		require.NotNil(t, middleware)
	})
}

// =============================================================================
// OptionalAuthMiddleware Creation Tests
// =============================================================================

func TestOptionalAuthMiddleware_Creation(t *testing.T) {
	t.Run("creates middleware with nil service", func(t *testing.T) {
		middleware := OptionalAuthMiddleware(nil)

		require.NotNil(t, middleware)
	})
}

// =============================================================================
// UnifiedAuthMiddleware Creation Tests
// =============================================================================

func TestUnifiedAuthMiddleware_Creation(t *testing.T) {
	t.Run("creates middleware with nil parameters", func(t *testing.T) {
		middleware := UnifiedAuthMiddleware(nil, nil, nil)

		require.NotNil(t, middleware)
	})
}

// =============================================================================
// Context Local Keys Tests
// =============================================================================

func TestContextLocalKeys(t *testing.T) {
	t.Run("standard local keys are consistent", func(t *testing.T) {
		ctx := createTestFiberCtx()
		defer func() {
			app := ctx.App()
			app.ReleaseCtx(ctx)
		}()

		// Set all standard locals that auth middleware sets
		ctx.Locals("user_id", "test-user-id")
		ctx.Locals("user_email", "test@example.com")
		ctx.Locals("user_role", "authenticated")
		ctx.Locals("session_id", "session-123")

		// Verify they can all be retrieved
		userID := mw.GetUserID(ctx)
		assert.Equal(t, "test-user-id", userID)

		email, ok := GetUserEmail(ctx)
		assert.True(t, ok)
		assert.Equal(t, "test@example.com", email)

		role, ok := GetUserRole(ctx)
		assert.True(t, ok)
		assert.Equal(t, "authenticated", role)

		sessionID := ctx.Locals("session_id")
		assert.Equal(t, "session-123", sessionID)
	})

	t.Run("jwt_claims local key can store claims", func(t *testing.T) {
		ctx := createTestFiberCtx()
		defer func() {
			app := ctx.App()
			app.ReleaseCtx(ctx)
		}()

		// jwt_claims is used to store full TokenClaims for Supabase compatibility
		type mockClaims struct {
			UserID string
			Role   string
		}

		claims := &mockClaims{UserID: "user-123", Role: "admin"}
		ctx.Locals("jwt_claims", claims)

		retrieved := ctx.Locals("jwt_claims")
		assert.NotNil(t, retrieved)

		typedClaims, ok := retrieved.(*mockClaims)
		assert.True(t, ok)
		assert.Equal(t, "user-123", typedClaims.UserID)
		assert.Equal(t, "admin", typedClaims.Role)
	})
}

// =============================================================================
// Role Constants Tests
// =============================================================================

func TestRoleConstants(t *testing.T) {
	t.Run("common role strings", func(t *testing.T) {
		// Document expected role values used in the system
		roles := map[string]string{
			"authenticated":  "Standard authenticated user",
			"admin":          "Application administrator",
			"service_role":   "Service account with elevated privileges",
			"instance_admin": "Dashboard admin user",
			"anon":           "Anonymous/unauthenticated user",
		}

		for role, description := range roles {
			assert.NotEmpty(t, role, "role should not be empty: %s", description)
		}
	})
}

// =============================================================================
// RequireRole Runtime Behavior Tests
// =============================================================================

func TestRequireRole_ServiceKeyTypes(t *testing.T) {
	tests := []struct {
		name           string
		serviceKeyType string
		allowedRoles   []string
		expectedStatus int
	}{
		{
			name:           "global service key bypasses admin-only route",
			serviceKeyType: "service",
			allowedRoles:   []string{"admin"},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "global_service key bypasses admin-only route",
			serviceKeyType: "global_service",
			allowedRoles:   []string{"admin"},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "tenant_service key rejected on admin-only route",
			serviceKeyType: "tenant_service",
			allowedRoles:   []string{"admin"},
			expectedStatus: fiber.StatusForbidden,
		},
		{
			name:           "tenant_service key allowed on route accepting tenant_admin",
			serviceKeyType: "tenant_service",
			allowedRoles:   []string{"admin", "tenant_admin"},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "tenant_service key allowed on tenant_admin-only route",
			serviceKeyType: "tenant_service",
			allowedRoles:   []string{"tenant_admin"},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "unset service_key_type treated as global (migrations middleware compat)",
			serviceKeyType: "",
			allowedRoles:   []string{"admin"},
			expectedStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApp(t)

			// Simulate service key auth having already validated the key
			app.Use(func(c fiber.Ctx) error {
				c.Locals("auth_type", "service_key")
				if tt.serviceKeyType != "" {
					c.Locals("service_key_type", tt.serviceKeyType)
				}
				return c.Next()
			})
			app.Use(RequireRole(tt.allowedRoles...))
			app.Get("/test", func(c fiber.Ctx) error {
				return c.SendString("OK")
			})

			req := httptest.NewRequest(fiber.MethodGet, "/test", nil)
			resp, err := app.Test(req)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestRequireRole_JWTAuth(t *testing.T) {
	tests := []struct {
		name           string
		userRole       interface{} // string or nil
		allowedRoles   []string
		expectedStatus int
	}{
		{
			name:           "service_role JWT bypasses admin-only route",
			userRole:       "service_role",
			allowedRoles:   []string{"admin"},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "service_role JWT bypasses tenant route",
			userRole:       "service_role",
			allowedRoles:   []string{"admin", "tenant_admin"},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "tenant_admin rejected on admin-only route",
			userRole:       "tenant_admin",
			allowedRoles:   []string{"admin"},
			expectedStatus: fiber.StatusForbidden,
		},
		{
			name:           "admin allowed on admin route",
			userRole:       "admin",
			allowedRoles:   []string{"admin"},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "nil user_role returns unauthorized",
			userRole:       nil,
			allowedRoles:   []string{"admin"},
			expectedStatus: fiber.StatusUnauthorized,
		},
		{
			name:           "tenant_service JWT allowed on route accepting tenant_admin",
			userRole:       "tenant_service",
			allowedRoles:   []string{"admin", "tenant_admin"},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "tenant_service JWT allowed on tenant_admin-only route",
			userRole:       "tenant_service",
			allowedRoles:   []string{"tenant_admin"},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "tenant_service JWT rejected on admin-only route",
			userRole:       "tenant_service",
			allowedRoles:   []string{"admin"},
			expectedStatus: fiber.StatusForbidden,
		},
		{
			name:           "tenant_service JWT rejected on admin+instance_admin route",
			userRole:       "tenant_service",
			allowedRoles:   []string{"admin", "instance_admin"},
			expectedStatus: fiber.StatusForbidden,
		},
		{
			name:           "tenant_service JWT allowed on route with admin+instance_admin+tenant_admin",
			userRole:       "tenant_service",
			allowedRoles:   []string{"admin", "instance_admin", "tenant_admin"},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "tenant_service JWT allowed on service_role+tenant_service route (no tenant_admin)",
			userRole:       "tenant_service",
			allowedRoles:   []string{"service_role", "tenant_service"},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "tenant_service JWT rejected on service_role-only route",
			userRole:       "tenant_service",
			allowedRoles:   []string{"service_role"},
			expectedStatus: fiber.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApp(t)

			app.Use(func(c fiber.Ctx) error {
				c.Locals("auth_type", "jwt")
				if tt.userRole != nil {
					c.Locals("user_role", tt.userRole)
				}
				return c.Next()
			})
			app.Use(RequireRole(tt.allowedRoles...))
			app.Get("/test", func(c fiber.Ctx) error {
				return c.SendString("OK")
			})

			req := httptest.NewRequest(fiber.MethodGet, "/test", nil)
			resp, err := app.Test(req)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Verify error response bodies
			if tt.expectedStatus == fiber.StatusForbidden {
				body, _ := io.ReadAll(resp.Body)
				assert.Contains(t, string(body), "Insufficient permissions")
			} else if tt.expectedStatus == fiber.StatusUnauthorized {
				body, _ := io.ReadAll(resp.Body)
				assert.Contains(t, string(body), "Unauthorized")
			}
		})
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkGetUserID(b *testing.B) {
	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	ctx.Locals("user_id", "user-123-benchmark")
	defer app.ReleaseCtx(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mw.GetUserID(ctx)
	}
}

func BenchmarkGetUserEmail(b *testing.B) {
	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	ctx.Locals("user_email", "benchmark@example.com")
	defer app.ReleaseCtx(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GetUserEmail(ctx)
	}
}

func BenchmarkGetUserRole(b *testing.B) {
	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	ctx.Locals("user_role", "authenticated")
	defer app.ReleaseCtx(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GetUserRole(ctx)
	}
}

func BenchmarkRequireRole_Creation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = RequireRole("admin", "service_role")
	}
}
