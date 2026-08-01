package middleware

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/database"
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

// =============================================================================
// isExpiredToken Tests (pure unit — runs in CI under -short)
//
// isExpiredToken gates the optional-auth degrade-to-anon behavior. ErrExpiredToken
// is only ever produced after a JWT's signature and claims have validated against
// the platform secret, so it must NOT match tampering / wrong-secret / malformed.
// =============================================================================

func TestIsExpiredToken(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"exact ErrExpiredToken", auth.ErrExpiredToken, true},
		{"wrapped ErrExpiredToken", fmt.Errorf("validation: %w", auth.ErrExpiredToken), true},
		{"ErrInvalidToken", auth.ErrInvalidToken, false},
		{"ErrInvalidSignature", auth.ErrInvalidSignature, false},
		{"generic error", errors.New("something else"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isExpiredToken(tt.err))
		})
	}
}

// =============================================================================
// OptionalAuth expired-token degrade (integration — requires DB for auth.Service)
//
// These exercise the real OptionalAuthOrServiceKey / RequireAuthOrServiceKey
// middleware end to end. They require a database to construct auth.Service, so
// they skip under -short (CI's `make test` runs the unit suite only). The
// security-relevant predicate (isExpiredToken) is covered above by the pure
// unit test that always runs.
// =============================================================================

const testJWTSecret = "test-secret-key-for-testing-only"

// newTestAuthService builds a real auth.Service backed by a database connection.
// Skips under -short. Mirrors the setup in internal/api/auth_otp_test.go.
func newTestAuthService(t *testing.T) *auth.Service {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping DB-backed auth middleware test in short mode")
	}

	dbHost := os.Getenv("FLUXBASE_DATABASE_HOST")
	if dbHost == "" {
		dbHost = os.Getenv("DB_HOST")
	}
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbUser := os.Getenv("FLUXBASE_DATABASE_USER")
	if dbUser == "" {
		dbUser = "fluxbase_app"
	}
	dbPassword := os.Getenv("FLUXBASE_DATABASE_PASSWORD")
	dbDatabase := os.Getenv("FLUXBASE_TEST_DATABASE")
	if dbDatabase == "" {
		dbDatabase = "fluxbase"
	}

	dbConfig := config.DatabaseConfig{
		Host:            dbHost,
		Port:            5432,
		User:            dbUser,
		Password:        dbPassword,
		Database:        dbDatabase,
		SSLMode:         "disable",
		MaxConnections:  5,
		MinConnections:  1,
		MaxConnLifetime: 5 * time.Minute,
		MaxConnIdleTime: 5 * time.Minute,
		HealthCheck:     30 * time.Second,
	}

	db, err := database.NewConnection(dbConfig)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, db.Health(ctx))

	authConfig := &config.AuthConfig{
		JWTSecret:           testJWTSecret,
		JWTExpiry:           15 * time.Minute,
		RefreshExpiry:       7 * 24 * time.Hour,
		ServiceRoleTTL:      24 * time.Hour,
		AnonTTL:             24 * time.Hour,
		MagicLinkExpiry:     15 * time.Minute,
		PasswordResetExpiry: 1 * time.Hour,
		PasswordMinLen:      8,
		BcryptCost:          4,
		SignupEnabled:       true,
	}

	return auth.NewService(db, authConfig, &auth.NoOpOTPSender{}, "http://localhost:3000", nil)
}

// signTestServiceRoleToken mints a service-role/anon-style JWT signed with the
// given secret, for the given role, at the provided iat/exp times. Mirrors the
// token shapes used in internal/auth/jwt_test.go.
func signTestServiceRoleToken(t *testing.T, secret, role string, iat, exp time.Time) string {
	t.Helper()
	claims := jwt.MapClaims{
		"role": role,
		"iss":  "fluxbase",
		"iat":  iat.Unix(),
		"exp":  exp.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return s
}

// authTestApp wires a Fiber app with the chosen auth middleware, returning the
// auth service so tests can mint tokens signed with the same secret.
func authTestApp(t *testing.T, required bool) (*fiber.App, *auth.Service) {
	t.Helper()
	authService := newTestAuthService(t)

	// ClientKeyService is only reached on the success / client-key paths, which
	// the failure-path tests below never exercise. A nil DB is safe here.
	cks := auth.NewClientKeyService(nil, nil)
	var mw fiber.Handler
	if required {
		mw = RequireAuthOrServiceKey(authService, cks, nil, &config.SecurityConfig{})
	} else {
		mw = OptionalAuthOrServiceKey(authService, cks, nil, &config.SecurityConfig{})
	}

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			return c.SendStatus(http.StatusInternalServerError)
		},
	})
	app.Use(mw)
	app.Get("/settings/:key", func(c fiber.Ctx) error {
		// Mirror what the real settings handler observes: an anonymous request
		// has no rls_role/user_id set (rls.go later defaults nil -> "anon").
		role, _ := c.Locals("rls_role").(string)
		userID := c.Locals("rls_user_id")
		return c.JSON(fiber.Map{"rls_role": role, "has_user_id": userID != nil})
	})
	return app, authService
}

func TestOptionalAuth_ExpiredBearerDegradesToAnonymous(t *testing.T) {
	app, authService := authTestApp(t, false)

	now := time.Now()
	// Signature is valid (signed with the platform secret) but exp is in the past.
	expired := signTestServiceRoleToken(t, testJWTSecret, "anon",
		now.Add(-2*time.Hour), now.Add(-1*time.Hour))
	// Sanity: the token really is expired (signature valid, exp past).
	_, err := authService.JWTManager().ValidateServiceRoleToken(expired)
	require.ErrorIs(t, err, auth.ErrExpiredToken)

	req := httptest.NewRequest(http.MethodGet, "/settings/wayli.is_setup_complete", nil)
	req.Header.Set("Authorization", "Bearer "+expired)
	resp, err := app.Test(req)

	require.NoError(t, err)
	// Degrades to anonymous (200), not 401 INVALID_TOKEN.
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), `"rls_role":""`)
	assert.Contains(t, string(body), `"has_user_id":false`)
}

func TestOptionalAuth_TamperedBearerStillRejects(t *testing.T) {
	app, _ := authTestApp(t, false)

	// Garbage "eyJ"-prefixed token: signature won't validate -> ErrInvalidToken,
	// NOT ErrExpiredToken. Must still 401.
	req := httptest.NewRequest(http.MethodGet, "/settings/wayli.is_setup_complete", nil)
	req.Header.Set("Authorization", "Bearer "+strings.Repeat("a", 40))
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestOptionalAuth_ValidBearerAuthenticates(t *testing.T) {
	app, authService := authTestApp(t, false)

	// Fresh anon token (signature valid, not expired) -> authenticates as anon.
	valid, err := authService.JWTManager().GenerateAnonToken()
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/settings/wayli.is_setup_complete", nil)
	req.Header.Set("Authorization", "Bearer "+valid)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), `"rls_role":"anon"`)
}

func TestOptionalAuth_NoTokenIsAnonymous(t *testing.T) {
	app, _ := authTestApp(t, false)

	req := httptest.NewRequest(http.MethodGet, "/settings/wayli.is_setup_complete", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), `"rls_role":""`)
	assert.Contains(t, string(body), `"has_user_id":false`)
}

func TestRequireAuth_ExpiredBearerStillRejects(t *testing.T) {
	app, authService := authTestApp(t, true) // required = true

	now := time.Now()
	expired := signTestServiceRoleToken(t, testJWTSecret, "anon",
		now.Add(-2*time.Hour), now.Add(-1*time.Hour))
	_, err := authService.JWTManager().ValidateServiceRoleToken(expired)
	require.ErrorIs(t, err, auth.ErrExpiredToken)

	req := httptest.NewRequest(http.MethodGet, "/settings/wayli.is_setup_complete", nil)
	req.Header.Set("Authorization", "Bearer "+expired)
	resp, err := app.Test(req)

	require.NoError(t, err)
	// Required routes do NOT degrade — expiry still 401s.
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
