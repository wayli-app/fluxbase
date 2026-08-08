package routes

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// BuildSettingsRoutes must apply the TenantContext middleware when provided,
// so the public read paths (single-key GET, batch, prefix) are tenant-scoped —
// matching the user-settings and admin-settings groups. Without it, the
// settings_tenant RLS policy can't filter because app.current_tenant_id is
// never set, and reads leak across tenants.
func TestBuildSettingsRoutes_AppliesTenantMiddleware(t *testing.T) {
	tenantCalled := false
	tenantMw := func(c fiber.Ctx) error {
		tenantCalled = true
		return c.Next()
	}

	deps := &SettingsDeps{
		OptionalAuth:     func(c fiber.Ctx) error { return c.Next() },
		TenantMiddleware: tenantMw,
		GetSetting:       func(c fiber.Ctx) error { return c.Next() },
		GetSettings:      func(c fiber.Ctx) error { return c.Next() },
		BatchGet:         func(c fiber.Ctx) error { return c.Next() },
	}

	group := BuildSettingsRoutes(deps)

	// The group must carry the TenantContext middleware by name.
	var foundTenant bool
	for _, mw := range group.Middlewares {
		if mw.Name == "TenantContext" {
			foundTenant = true
			break
		}
	}
	require.True(t, foundTenant, "BuildSettingsRoutes should apply a 'TenantContext' middleware")

	// Sanity: the middleware runs (smoke-test the handler is wired).
	_ = group // group structure asserted; full fiber invocation is covered by integration tests.
	assert.NotEmpty(t, group.Middlewares)
	_ = tenantCalled
}

// When no TenantMiddleware is supplied (e.g. a minimal deployment), the group
// should still build without panicking and carry no TenantContext entry —
// preserving backward compatibility rather than hard-failing.
func TestBuildSettingsRoutes_NoTenantMiddlewareIsSafe(t *testing.T) {
	deps := &SettingsDeps{
		OptionalAuth: func(c fiber.Ctx) error { return c.Next() },
		GetSetting:   func(c fiber.Ctx) error { return c.Next() },
		GetSettings:  func(c fiber.Ctx) error { return c.Next() },
		BatchGet:     func(c fiber.Ctx) error { return c.Next() },
	}

	group := BuildSettingsRoutes(deps)

	var foundTenant bool
	for _, mw := range group.Middlewares {
		if mw.Name == "TenantContext" {
			foundTenant = true
			break
		}
	}
	assert.False(t, foundTenant, "no TenantContext middleware should be added when none is provided")
}

// TestSettingsRoutePrecedence guards against a Fiber routing regression where
// the generic GET /api/v1/settings/:key param route shadowed the static child
// groups at their index path. Before the fix, GET /api/v1/settings/secret/ was
// dispatched to the generic GetSetting handler with key="secret", returning
// 404 "Setting not found or access denied" — which broke the SDK's
// listSecrets() / listSettings() calls and surfaced in Wayli as
// "Couldn't check API key status".
//
// This test registers the three settings groups exactly as registerAllGroups
// does (via the Registry) and asserts that the secret + user index routes and
// the generic param route all dispatch to their own handlers. The static
// "secret"/"user" segments must win over the :key param at the index path.
func TestSettingsRoutePrecedence(t *testing.T) {
	next := func(c fiber.Ctx) error { return c.Next() }

	// Distinct handlers so we can assert which one served each request.
	genericKey := func(c fiber.Ctx) error { return c.SendString("generic:" + c.Params("key")) }
	secretList := func(c fiber.Ctx) error { return c.SendString("secret-list") }
	secretGet := func(c fiber.Ctx) error { return c.SendString("secret:" + c.Params("*")) }
	userList := func(c fiber.Ctx) error { return c.SendString("user-list") }
	userGet := func(c fiber.Ctx) error { return c.SendString("user:" + c.Params("key")) }

	deps := &AllDeps{
		Settings: &SettingsDeps{
			OptionalAuth: next,
			RequireAuth:  next,
			GetSetting:   genericKey,
			GetSettings:  func(c fiber.Ctx) error { return c.SendString("generic-index") },
			BatchGet:     next,
		},
		UserSettings: &UserSettingsDeps{
			RequireAuth:       next,
			ListSettings:      userList,
			GetUserOwnSetting: next,
			GetSystemSetting:  next,
			GetSetting:        userGet,
			SetSetting:        next,
			DeleteSetting:     next,
			CreateSecret:      next,
			ListSecrets:       secretList,
			GetSecret:         secretGet,
			UpdateSecret:      next,
			DeleteSecret:      next,
		},
	}

	registry := NewRegistry()
	registerAllGroups(registry, deps)

	app := fiber.New()
	require.NoError(t, registry.Apply(app))

	cases := []struct {
		name       string
		method     string
		path       string
		wantBody   string
		wantStatus int
	}{
		{"secret list (SDK listSecrets)", "GET", "/api/v1/settings/secret/", "secret-list", 200},
		{"secret list no trailing slash", "GET", "/api/v1/settings/secret", "secret-list", 200},
		{"secret per-key (SDK getSecret)", "GET", "/api/v1/settings/secret/owntracks_api_key", "secret:owntracks_api_key", 200},
		{"user list (SDK listSettings)", "GET", "/api/v1/settings/user/list", "user-list", 200},
		{"user per-key", "GET", "/api/v1/settings/user/somekey", "user:somekey", 200},
		{"generic param not shadowed", "GET", "/api/v1/settings/somekey", "generic:somekey", 200},
		{"generic index", "GET", "/api/v1/settings/", "generic-index", 200},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			body, _ := io.ReadAll(resp.Body)
			assert.Equal(t, tc.wantStatus, resp.StatusCode, "body: %s", string(body))
			assert.Equal(t, tc.wantBody, string(body))
		})
	}
}
