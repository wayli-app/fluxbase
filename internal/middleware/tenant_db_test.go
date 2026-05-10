package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/tenantdb"
)

// ---------------------------------------------------------------------------
// GetPoolForSchema tests
// ---------------------------------------------------------------------------

func TestGetPoolForSchema_TenantPoolForAllSchemas(t *testing.T) {
	// Verify that ALL schemas route to tenant pool (not just public).
	// This is critical for FDW cross-database joins.
	schemas := []string{"public", "auth", "storage", "app", "jobs", "ai", "rpc", "mcp"}

	for _, schema := range schemas {
		t.Run("schema_"+schema, func(t *testing.T) {
			app := fiber.New()
			tenantPool := &pgxpool.Pool{}
			var capturedPool *pgxpool.Pool

			app.Get("/test", func(c fiber.Ctx) error {
				c.Locals("tenant_db", tenantPool)
				capturedPool = GetPoolForSchema(c, schema, nil)
				return c.SendString("OK")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			_, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tenantPool, capturedPool,
				"tenant pool should be used for schema %q", schema)
		})
	}
}

func TestGetPoolForSchema_NoTenantPool_ReturnsMainPool(t *testing.T) {
	app := fiber.New()
	mainPool := &pgxpool.Pool{}

	app.Get("/test", func(c fiber.Ctx) error {
		pool := GetPoolForSchema(c, "public", mainPool)
		assert.Equal(t, mainPool, pool, "should return main pool when no tenant pool")
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGetPoolForSchema_BranchPoolOverridesTenantPool(t *testing.T) {
	app := fiber.New()
	branchPool := &pgxpool.Pool{}
	tenantPool := &pgxpool.Pool{}
	mainPool := &pgxpool.Pool{}
	var capturedPool *pgxpool.Pool

	app.Get("/test", func(c fiber.Ctx) error {
		c.Locals("branch_pool", branchPool)
		c.Locals("tenant_db", tenantPool)
		capturedPool = GetPoolForSchema(c, "public", mainPool)
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	_, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, branchPool, capturedPool, "branch pool should override tenant pool")
}

// ---------------------------------------------------------------------------
// GetTenantPool / SetTargetSchema / GetTargetSchema
// ---------------------------------------------------------------------------

func TestGetTenantPool(t *testing.T) {
	t.Run("returns pool when set", func(t *testing.T) {
		app := fiber.New()
		tenantPool := &pgxpool.Pool{}

		app.Get("/test", func(c fiber.Ctx) error {
			c.Locals("tenant_db", tenantPool)
			pool := GetTenantPool(c)
			assert.Equal(t, tenantPool, pool)
			return c.SendString("OK")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		_, err := app.Test(req)
		require.NoError(t, err)
	})

	t.Run("returns nil when not set", func(t *testing.T) {
		app := fiber.New()

		app.Get("/test", func(c fiber.Ctx) error {
			pool := GetTenantPool(c)
			assert.Nil(t, pool)
			return c.SendString("OK")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		_, err := app.Test(req)
		require.NoError(t, err)
	})
}

func TestSetTargetSchema_GetTargetSchema(t *testing.T) {
	app := fiber.New()

	app.Get("/test", func(c fiber.Ctx) error {
		SetTargetSchema(c, "public")
		schema := GetTargetSchema(c)
		assert.Equal(t, "public", schema)
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGetTargetSchema_DefaultsEmpty(t *testing.T) {
	app := fiber.New()

	app.Get("/test", func(c fiber.Ctx) error {
		schema := GetTargetSchema(c)
		assert.Equal(t, "", schema)
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	_, err := app.Test(req)
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// UsesMainDatabase
// ---------------------------------------------------------------------------

func TestUsesMainDatabase(t *testing.T) {
	t.Run("nil DBName uses main database", func(t *testing.T) {
		tenant := &tenantdb.Tenant{DBName: nil}
		assert.True(t, tenant.UsesMainDatabase())
	})

	t.Run("empty DBName uses main database", func(t *testing.T) {
		dbName := ""
		tenant := &tenantdb.Tenant{DBName: &dbName}
		assert.True(t, tenant.UsesMainDatabase())
	})

	t.Run("non-empty DBName uses separate database", func(t *testing.T) {
		dbName := "tenant_test"
		tenant := &tenantdb.Tenant{DBName: &dbName}
		assert.False(t, tenant.UsesMainDatabase())
	})
}

// ---------------------------------------------------------------------------
// Phase 2: RequireTenantRole("tenant_admin") unit tests
// ---------------------------------------------------------------------------

func TestRequireTenantRole_TenantAdmin_InstanceAdminWithoutTenantContext(t *testing.T) {
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
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestRequireTenantRole_TenantAdmin_InstanceAdminWithHeaderSource_NeedsRole(t *testing.T) {
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
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestRequireTenantRole_TenantAdmin_TenantAdminRole_Passes(t *testing.T) {
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
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestRequireTenantRole_TenantAdmin_NoTenantRole_Forbidden(t *testing.T) {
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals("is_instance_admin", false)
		c.Locals("tenant_id", "t-123")
		c.Locals("tenant_source", "header")
		// tenant_role is empty
		return c.Next()
	})
	app.Use(RequireTenantRole("tenant_admin"))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)

	body := make([]byte, resp.ContentLength)
	resp.Body.Read(body)
	assert.Contains(t, string(body), "tenant membership required")
}

func TestRequireTenantRole_TenantAdmin_NonAdminRole_Forbidden(t *testing.T) {
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals("is_instance_admin", false)
		c.Locals("tenant_id", "t-123")
		c.Locals("tenant_source", "jwt")
		c.Locals("tenant_role", "tenant_viewer")
		return c.Next()
	})
	app.Use(RequireTenantRole("tenant_admin"))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)

	body := make([]byte, resp.ContentLength)
	resp.Body.Read(body)
	assert.Contains(t, string(body), "tenant_admin role required")
}

func TestRequireTenantRole_TenantAdmin_JWTSource_ActingAsTenantAdmin(t *testing.T) {
	// JWT source counts as "acting as tenant admin" for instance admins,
	// but non-admin with jwt source still needs tenant_role
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals("is_instance_admin", false)
		c.Locals("tenant_id", "t-123")
		c.Locals("tenant_source", "jwt")
		// no tenant_role
		return c.Next()
	})
	app.Use(RequireTenantRole("tenant_admin"))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestRequireTenantRole_TenantAdmin_InstanceAdminJWTSource_WithRole(t *testing.T) {
	// Instance admin acting as tenant admin via JWT with role
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals("is_instance_admin", true)
		c.Locals("tenant_id", "t-123")
		c.Locals("tenant_source", "jwt")
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
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// Phase 3: Mock-based TenantDBMiddleware tests
// ---------------------------------------------------------------------------

// mockTenantRepository implements TenantStore for testing.
type mockTenantRepository struct {
	getTenantFunc        func(ctx context.Context, id string) (*tenantdb.Tenant, error)
	getTenantBySlugFunc  func(ctx context.Context, slug string) (*tenantdb.Tenant, error)
	getDefaultTenantFunc func(ctx context.Context) (*tenantdb.Tenant, error)
	isUserAssignedFunc   func(ctx context.Context, userID, tenantID string) (bool, error)
}

func (m *mockTenantRepository) GetTenant(ctx context.Context, id string) (*tenantdb.Tenant, error) {
	if m.getTenantFunc != nil {
		return m.getTenantFunc(ctx, id)
	}
	return nil, tenantdb.ErrTenantNotFound
}

func (m *mockTenantRepository) GetTenantBySlug(ctx context.Context, slug string) (*tenantdb.Tenant, error) {
	if m.getTenantBySlugFunc != nil {
		return m.getTenantBySlugFunc(ctx, slug)
	}
	return nil, tenantdb.ErrTenantNotFound
}

func (m *mockTenantRepository) GetDefaultTenant(ctx context.Context) (*tenantdb.Tenant, error) {
	if m.getDefaultTenantFunc != nil {
		return m.getDefaultTenantFunc(ctx)
	}
	return nil, tenantdb.ErrNoDefaultTenant
}

func (m *mockTenantRepository) IsUserAssignedToTenant(ctx context.Context, userID, tenantID string) (bool, error) {
	if m.isUserAssignedFunc != nil {
		return m.isUserAssignedFunc(ctx, userID, tenantID)
	}
	return true, nil
}

// mockUserTenantLister implements UserTenantLister for testing.
type mockUserTenantLister struct {
	getTenantsForUserFunc func(ctx context.Context, userID string) ([]tenantdb.Tenant, error)
}

func (m *mockUserTenantLister) GetTenantsForUser(ctx context.Context, userID string) ([]tenantdb.Tenant, error) {
	if m.getTenantsForUserFunc != nil {
		return m.getTenantsForUserFunc(ctx, userID)
	}
	return nil, nil
}

func TestTenantDBMiddleware_DefaultTenant_SetsNoPool(t *testing.T) {
	// Default tenant (DBName=nil) should NOT set tenant_db
	store := &mockTenantRepository{
		getDefaultTenantFunc: func(ctx context.Context) (*tenantdb.Tenant, error) {
			return &tenantdb.Tenant{
				ID:        "default-id",
				Slug:      "default",
				IsDefault: true,
				DBName:    nil, // UsesMainDatabase() == true
			}, nil
		},
	}

	app := fiber.New()
	app.Use(TenantDBMiddleware(TenantDBConfig{
		Repository: store,
	}))
	app.Get("/test", func(c fiber.Ctx) error {
		tenantID, _ := c.Locals("tenant_id").(string)
		assert.Equal(t, "default-id", tenantID)
		assert.Nil(t, c.Locals("tenant_db"), "tenant_db should be nil for default tenant")
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestTenantDBMiddleware_SeparateDBTenant_SetsPool(t *testing.T) {
	// Tenant with separate DB should set tenant_db and tenant_db_name
	tenantPool := &pgxpool.Pool{}
	dbName := "tenant_acme"

	store := &mockTenantRepository{
		getTenantBySlugFunc: func(ctx context.Context, slug string) (*tenantdb.Tenant, error) {
			return &tenantdb.Tenant{
				ID:     "acme-id",
				Slug:   "acme",
				DBName: &dbName, // UsesMainDatabase() == false
			}, nil
		},
	}

	// We need a Router to provide the pool. Since Router needs *Storage,
	// and our mock is a TenantStore not *Storage, we skip Router
	// and test the storage/tenant resolution path only.
	// The pool setup via Router is tested in integration tests.
	_ = tenantPool

	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		// Simulate auth middleware
		c.Locals("user_id", "user-123")
		return c.Next()
	})
	app.Use(TenantDBMiddleware(TenantDBConfig{
		Repository: store,
		// Router is nil — we can't test pool creation here,
		// that's covered by integration tests
	}))
	app.Get("/test", func(c fiber.Ctx) error {
		tenantID, _ := c.Locals("tenant_id").(string)
		tenantSource, _ := c.Locals("tenant_source").(string)
		assert.Equal(t, "acme-id", tenantID)
		assert.Equal(t, "header", tenantSource)
		// tenant_db is nil because Router is nil (no pool created)
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-FB-Tenant", "acme")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestTenantDBMiddleware_UserNotAssigned_ReturnsForbidden(t *testing.T) {
	store := &mockTenantRepository{
		getTenantBySlugFunc: func(ctx context.Context, slug string) (*tenantdb.Tenant, error) {
			return &tenantdb.Tenant{
				ID:   "t-123",
				Slug: slug,
			}, nil
		},
		isUserAssignedFunc: func(ctx context.Context, userID, tenantID string) (bool, error) {
			return false, nil
		},
	}

	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user_id", "user-123")
		c.Locals("is_instance_admin", false)
		return c.Next()
	})
	app.Use(TenantDBMiddleware(TenantDBConfig{
		Repository: store,
	}))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-FB-Tenant", "acme")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestTenantDBMiddleware_InstanceAdmin_SkipsAccessCheck(t *testing.T) {
	accessChecked := false
	store := &mockTenantRepository{
		getTenantBySlugFunc: func(ctx context.Context, slug string) (*tenantdb.Tenant, error) {
			return &tenantdb.Tenant{ID: "t-123", Slug: slug}, nil
		},
		isUserAssignedFunc: func(ctx context.Context, userID, tenantID string) (bool, error) {
			accessChecked = true
			return true, nil
		},
	}

	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user_id", "admin-123")
		c.Locals("is_instance_admin", true)
		return c.Next()
	})
	app.Use(TenantDBMiddleware(TenantDBConfig{
		Repository: store,
	}))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-FB-Tenant", "acme")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.False(t, accessChecked, "instance admin should skip access check")
}

func TestTenantDBMiddleware_ResolveFromHeader(t *testing.T) {
	store := &mockTenantRepository{
		getTenantBySlugFunc: func(ctx context.Context, slug string) (*tenantdb.Tenant, error) {
			return &tenantdb.Tenant{ID: "h-id", Slug: slug}, nil
		},
	}

	app := fiber.New()
	app.Use(TenantDBMiddleware(TenantDBConfig{Repository: store}))
	app.Get("/test", func(c fiber.Ctx) error {
		source, _ := c.Locals("tenant_source").(string)
		assert.Equal(t, "header", source)
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-FB-Tenant", "mytenant")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestTenantDBMiddleware_ResolveFromDefault(t *testing.T) {
	store := &mockTenantRepository{
		getDefaultTenantFunc: func(ctx context.Context) (*tenantdb.Tenant, error) {
			return &tenantdb.Tenant{ID: "def-id", Slug: "default", IsDefault: true}, nil
		},
	}

	app := fiber.New()
	app.Use(TenantDBMiddleware(TenantDBConfig{Repository: store}))
	app.Get("/test", func(c fiber.Ctx) error {
		tenantID, _ := c.Locals("tenant_id").(string)
		source, _ := c.Locals("tenant_source").(string)
		assert.Equal(t, "def-id", tenantID)
		assert.Equal(t, "default", source)
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestTenantDBMiddleware_NoTenantAtAll(t *testing.T) {
	store := &mockTenantRepository{
		getDefaultTenantFunc: func(ctx context.Context) (*tenantdb.Tenant, error) {
			return nil, tenantdb.ErrNoDefaultTenant
		},
	}

	app := fiber.New()
	app.Use(TenantDBMiddleware(TenantDBConfig{Repository: store}))
	app.Get("/test", func(c fiber.Ctx) error {
		tenantID, _ := c.Locals("tenant_id").(string)
		assert.Empty(t, tenantID)
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestTenantDBMiddleware_ResolveFromJWT(t *testing.T) {
	store := &mockTenantRepository{}

	jwtTenantID := "jwt-tenant-id"
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals("claims", &auth.TokenClaims{TenantID: &jwtTenantID})
		return c.Next()
	})
	app.Use(TenantDBMiddleware(TenantDBConfig{Repository: store}))
	app.Get("/test", func(c fiber.Ctx) error {
		tenantID, _ := c.Locals("tenant_id").(string)
		source, _ := c.Locals("tenant_source").(string)
		assert.Equal(t, "jwt-tenant-id", tenantID)
		assert.Equal(t, "jwt", source)
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestTenantDBMiddleware_NilStorage_NoPanic(t *testing.T) {
	app := fiber.New()
	app.Use(TenantDBMiddleware(TenantDBConfig{Repository: nil}))
	app.Get("/test", func(c fiber.Ctx) error {
		tenantID, _ := c.Locals("tenant_id").(string)
		assert.Empty(t, tenantID)
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

// JWT claims set tenant_role when matching
func TestTenantDBMiddleware_JWTClaimsSetTenantRole(t *testing.T) {
	store := &mockTenantRepository{}
	jwtTenantID := "jwt-tenant-id"

	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user_id", "user-123") // Required for tenant_role to be set
		c.Locals("claims", &auth.TokenClaims{
			TenantID:   &jwtTenantID,
			TenantRole: "tenant_admin",
		})
		return c.Next()
	})
	app.Use(TenantDBMiddleware(TenantDBConfig{Repository: store}))
	app.Get("/test", func(c fiber.Ctx) error {
		role, _ := c.Locals("tenant_role").(string)
		assert.Equal(t, "tenant_admin", role)
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

// GetUserManagedTenantIDs tests
func TestGetUserManagedTenantIDs_Success(t *testing.T) {
	lister := &mockUserTenantLister{
		getTenantsForUserFunc: func(ctx context.Context, userID string) ([]tenantdb.Tenant, error) {
			return []tenantdb.Tenant{
				{ID: "t-1"},
				{ID: "t-2"},
			}, nil
		},
	}

	ids, err := GetUserManagedTenantIDs(context.Background(), lister, "user-123")
	require.NoError(t, err)
	assert.Equal(t, []string{"t-1", "t-2"}, ids)
}

func TestGetUserManagedTenantIDs_NilStorage(t *testing.T) {
	ids, err := GetUserManagedTenantIDs(context.Background(), nil, "user-123")
	assert.Nil(t, ids)
	assert.EqualError(t, err, "storage not initialized")
}

func TestGetUserManagedTenantIDs_StorageError(t *testing.T) {
	lister := &mockUserTenantLister{
		getTenantsForUserFunc: func(ctx context.Context, userID string) ([]tenantdb.Tenant, error) {
			return nil, errors.New("db error")
		},
	}

	ids, err := GetUserManagedTenantIDs(context.Background(), lister, "user-123")
	assert.Nil(t, ids)
	assert.Contains(t, err.Error(), "failed to get user tenants")
}

// ---------------------------------------------------------------------------
// resolveTenantID tests — validates the fix for the X-FB-Tenant header
// being accepted without storage validation.
// ---------------------------------------------------------------------------

func TestResolveTenantID_ValidSlug(t *testing.T) {
	tenantID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	store := &mockTenantRepository{
		getTenantBySlugFunc: func(_ context.Context, slug string) (*tenantdb.Tenant, error) {
			if slug == "acme" {
				return &tenantdb.Tenant{ID: tenantID, Slug: "acme"}, nil
			}
			return nil, tenantdb.ErrTenantNotFound
		},
	}

	app := fiber.New()
	var gotID, gotSource string
	app.Get("/test", func(c fiber.Ctx) error {
		gotID, gotSource = resolveTenantID(c, "", false, nil, store)
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-FB-Tenant", "acme")
	_, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, tenantID, gotID)
	assert.Equal(t, "header", gotSource)
}

func TestResolveTenantID_InvalidSlugReturnsEmpty(t *testing.T) {
	store := &mockTenantRepository{
		getTenantBySlugFunc: func(_ context.Context, _ string) (*tenantdb.Tenant, error) {
			return nil, tenantdb.ErrTenantNotFound
		},
		getTenantFunc: func(_ context.Context, _ string) (*tenantdb.Tenant, error) {
			return nil, tenantdb.ErrTenantNotFound
		},
		getDefaultTenantFunc: func(_ context.Context) (*tenantdb.Tenant, error) {
			return &tenantdb.Tenant{ID: "default-id", Slug: "default"}, nil
		},
	}

	app := fiber.New()
	var gotID, gotSource string
	app.Get("/test", func(c fiber.Ctx) error {
		gotID, gotSource = resolveTenantID(c, "", false, nil, store)
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-FB-Tenant", "nonexistent")
	_, err := app.Test(req)
	require.NoError(t, err)
	assert.Empty(t, gotID, "Should return empty when header value matches no tenant")
	assert.Empty(t, gotSource, "Source should be empty when tenant not found")
}

func TestResolveTenantID_FallbackToDefault(t *testing.T) {
	store := &mockTenantRepository{
		getDefaultTenantFunc: func(_ context.Context) (*tenantdb.Tenant, error) {
			return &tenantdb.Tenant{ID: "default-id", Slug: "default"}, nil
		},
	}

	app := fiber.New()
	var gotID, gotSource string
	app.Get("/test", func(c fiber.Ctx) error {
		gotID, gotSource = resolveTenantID(c, "", false, nil, store)
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	_, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, "default-id", gotID)
	assert.Equal(t, "default", gotSource)
}

func TestResolveTenantID_JWTClaims(t *testing.T) {
	tenantID := "jwt-tenant-id"
	store := &mockTenantRepository{
		getDefaultTenantFunc: func(_ context.Context) (*tenantdb.Tenant, error) {
			return &tenantdb.Tenant{ID: "default-id", Slug: "default"}, nil
		},
	}

	claims := &auth.TokenClaims{}
	tenantIDStr := tenantID
	claims.TenantID = &tenantIDStr

	app := fiber.New()
	var gotID, gotSource string
	app.Get("/test", func(c fiber.Ctx) error {
		gotID, gotSource = resolveTenantID(c, "", false, claims, store)
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	_, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, tenantID, gotID)
	assert.Equal(t, "jwt", gotSource)
}
