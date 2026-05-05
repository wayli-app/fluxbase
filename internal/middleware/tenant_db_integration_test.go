//go:build integration

package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/database/bootstrap"
	"github.com/nimbleflux/fluxbase/internal/tenantdb"
	"github.com/nimbleflux/fluxbase/test/dbhelpers"
)

// buildTenantDBURL constructs a database URL for a specific database name
// by replacing the database in a base connection URL.
func buildTenantDBURL(baseURL, dbName string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL + dbName // fallback
	}
	u.Path = "/" + dbName
	return u.String()
}

// middlewareTestEnv holds all resources needed for a middleware integration test.
type middlewareTestEnv struct {
	mainPool   *pgxpool.Pool
	storage    *tenantdb.Storage
	router     *tenantdb.Router
	adminPool  *pgxpool.Pool
	dbURL      string
	defaultT   *tenantdb.Tenant
	separateT  *tenantdb.Tenant
	separateDB string
	testCtx    *dbhelpers.DBTestContext
}

func setupMiddlewareIntegration(t *testing.T) *middlewareTestEnv {
	t.Helper()

	testCtx := dbhelpers.NewDBTestContext(t)
	mainPool := testCtx.Pool
	ctx := context.Background()

	// Create an admin pool (same DB, admin user for CREATE DATABASE)
	cfg := dbhelpers.GetTestConfig()
	adminDBURL := fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.Database.AdminUser,
		cfg.Database.AdminPassword,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Database,
	)
	adminPool, err := pgxpool.New(ctx, adminDBURL)
	require.NoError(t, err, "Failed to create admin pool")

	// Create Storage
	storage := tenantdb.NewStorage(mainPool)

	// Create a default tenant (no separate DB)
	defaultTenant := &tenantdb.Tenant{
		Slug:      fmt.Sprintf("mit-default-%s", uuid.New().String()[:8]),
		Name:      "Default Integration Test Tenant",
		IsDefault: true,
		Status:    tenantdb.TenantStatusActive,
		Metadata:  map[string]any{},
	}
	err = storage.CreateTenant(ctx, defaultTenant)
	require.NoError(t, err, "Failed to create default tenant")

	// Create a separate-DB tenant
	separateDBName := fmt.Sprintf("tenant_mit_%s", uuid.New().String()[:8])
	separateTenant := &tenantdb.Tenant{
		Slug:     fmt.Sprintf("mit-sep-%s", uuid.New().String()[:8]),
		Name:     "Separate DB Integration Test Tenant",
		DBName:   &separateDBName,
		Status:   tenantdb.TenantStatusActive,
		Metadata: map[string]any{},
	}
	err = storage.CreateTenant(ctx, separateTenant)
	require.NoError(t, err, "Failed to create separate-DB tenant")

	// Create the actual database
	_, err = adminPool.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s ENCODING 'UTF8'", separateDBName))
	require.NoError(t, err, "Failed to create separate database")

	// Bootstrap the separate DB using admin credentials (app user can't create extensions)
	tenantAdminURL := buildTenantDBURL(adminDBURL, separateDBName)
	err = bootstrap.RunBootstrapOnDB(ctx, tenantAdminURL, "fluxbase_app")
	require.NoError(t, err, "Failed to bootstrap separate database")

	// Create Router
	router := tenantdb.NewRouter(storage, tenantdb.DefaultConfig(), mainPool, adminPool, testCtx.DatabaseURL())

	// Update the separate tenant's DB name in the database
	err = storage.UpdateTenantDBName(ctx, separateTenant.ID, separateDBName)
	require.NoError(t, err, "Failed to update tenant DB name")

	// Refresh tenant from DB
	separateTenant, err = storage.GetTenant(ctx, separateTenant.ID)
	require.NoError(t, err)

	env := &middlewareTestEnv{
		mainPool:   mainPool,
		storage:    storage,
		router:     router,
		adminPool:  adminPool,
		dbURL:      testCtx.DatabaseURL(),
		defaultT:   defaultTenant,
		separateT:  separateTenant,
		separateDB: separateDBName,
		testCtx:    testCtx,
	}

	t.Cleanup(func() {
		router.Close()

		// Remove tenant records
		_, _ = mainPool.Exec(ctx, "DELETE FROM platform.tenants WHERE id = $1", defaultTenant.ID)
		_, _ = mainPool.Exec(ctx, "DELETE FROM platform.tenants WHERE id = $1", separateTenant.ID)

		// Drop the separate database
		_, _ = adminPool.Exec(ctx, fmt.Sprintf(
			"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s' AND pid <> pg_backend_pid()",
			separateDBName,
		))
		_, _ = adminPool.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", separateDBName))

		adminPool.Close()
		testCtx.Close()
	})

	return env
}

func TestTenantDBMiddleware_Integration_SeparateDBTenant_SetsPool(t *testing.T) {
	env := setupMiddlewareIntegration(t)
	ctx := context.Background()

	// Create a user in platform.users (required for FK constraint in tenant_admin_assignments)
	userID := uuid.New().String()
	_, err := env.mainPool.Exec(ctx, `
		INSERT INTO platform.users (id, email, password_hash, role, created_at, updated_at)
		VALUES ($1::uuid, $2, '', 'tenant_admin', NOW(), NOW())
	`, userID, fmt.Sprintf("mit-test-%s@example.com", userID[:8]))
	require.NoError(t, err, "Failed to create test user")
	defer func() {
		_, _ = env.mainPool.Exec(ctx, "DELETE FROM platform.users WHERE id = $1::uuid", userID)
	}()

	// Assign the user to the tenant
	err = env.storage.AssignUserToTenant(ctx, userID, env.separateT.ID)
	require.NoError(t, err)
	defer func() {
		_ = env.storage.RemoveUserFromTenant(ctx, userID, env.separateT.ID)
	}()

	var capturedPool *pgxpool.Pool
	var capturedDBName string
	var capturedTenantID string

	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user_id", userID)
		c.Locals("is_instance_admin", false)
		return c.Next()
	})
	app.Use(TenantDBMiddleware(TenantDBConfig{
		Storage: env.storage,
		Router:  env.router,
	}))
	app.Get("/test", func(c fiber.Ctx) error {
		capturedPool = GetTenantPool(c)
		capturedDBName, _ = c.Locals("tenant_db_name").(string)
		capturedTenantID, _ = c.Locals("tenant_id").(string)
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-FB-Tenant", env.separateT.Slug)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	assert.NotNil(t, capturedPool, "tenant_db should be set for separate-DB tenant")
	assert.Equal(t, env.separateDB, capturedDBName, "tenant_db_name should match")
	assert.Equal(t, env.separateT.ID, capturedTenantID, "tenant_id should match")

	// Verify the pool connects to the correct database
	var currentDB string
	err = capturedPool.QueryRow(ctx, "SELECT current_database()").Scan(&currentDB)
	require.NoError(t, err)
	assert.Equal(t, env.separateDB, currentDB,
		"Tenant pool should connect to the tenant's database, not the main database")
}

func TestTenantDBMiddleware_DefaultTenant_NoPool(t *testing.T) {
	env := setupMiddlewareIntegration(t)

	var capturedPool *pgxpool.Pool
	var capturedTenantID string

	// Use instance admin to bypass tenant access check for default tenant
	// (non-admin users need explicit tenant assignment even for the default tenant)
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user_id", uuid.New().String())
		c.Locals("is_instance_admin", true)
		return c.Next()
	})
	app.Use(TenantDBMiddleware(TenantDBConfig{
		Storage: env.storage,
		Router:  env.router,
	}))
	app.Get("/test", func(c fiber.Ctx) error {
		capturedPool = GetTenantPool(c)
		capturedTenantID, _ = c.Locals("tenant_id").(string)
		return c.SendString("OK")
	})

	// No X-FB-Tenant header → resolves to default tenant
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	assert.Nil(t, capturedPool, "tenant_db should be nil for default tenant (backward compat)")
	assert.NotEmpty(t, capturedTenantID, "tenant_id should be set to some tenant (default resolution)")
}

func TestTenantDBMiddleware_InstanceAdmin_SeparateDBTenant(t *testing.T) {
	env := setupMiddlewareIntegration(t)
	ctx := context.Background()

	var capturedPool *pgxpool.Pool
	var capturedDBName string

	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user_id", "admin-"+uuid.New().String())
		c.Locals("is_instance_admin", true)
		return c.Next()
	})
	app.Use(TenantDBMiddleware(TenantDBConfig{
		Storage: env.storage,
		Router:  env.router,
	}))
	app.Get("/test", func(c fiber.Ctx) error {
		capturedPool = GetTenantPool(c)
		capturedDBName, _ = c.Locals("tenant_db_name").(string)
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-FB-Tenant", env.separateT.Slug)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	assert.NotNil(t, capturedPool, "Instance admin should get tenant pool for separate-DB tenant")
	assert.Equal(t, env.separateDB, capturedDBName, "tenant_db_name should match")

	// Verify pool connects to the right database
	var currentDB string
	err = capturedPool.QueryRow(ctx, "SELECT current_database()").Scan(&currentDB)
	require.NoError(t, err)
	assert.Equal(t, env.separateDB, currentDB, "Pool should connect to tenant database")
}

func TestTenantDBMiddleware_InstanceAdmin_DefaultTenant(t *testing.T) {
	env := setupMiddlewareIntegration(t)

	var capturedPool *pgxpool.Pool

	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user_id", "admin-"+uuid.New().String())
		c.Locals("is_instance_admin", true)
		return c.Next()
	})
	app.Use(TenantDBMiddleware(TenantDBConfig{
		Storage: env.storage,
		Router:  env.router,
	}))
	app.Get("/test", func(c fiber.Ctx) error {
		capturedPool = GetTenantPool(c)
		return c.SendString("OK")
	})

	// No tenant header → resolves to default tenant
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	assert.Nil(t, capturedPool, "tenant_db should be nil for default tenant even as instance admin")
}

func TestTenantDBMiddleware_JWTSource_SeparateDBTenant(t *testing.T) {
	env := setupMiddlewareIntegration(t)
	ctx := context.Background()

	// Create a user in platform.users (required for FK constraint in tenant_admin_assignments)
	userID := uuid.New().String()
	_, err := env.mainPool.Exec(ctx, `
		INSERT INTO platform.users (id, email, password_hash, role, created_at, updated_at)
		VALUES ($1::uuid, $2, '', 'tenant_admin', NOW(), NOW())
	`, userID, fmt.Sprintf("mit-jwt-%s@example.com", userID[:8]))
	require.NoError(t, err, "Failed to create test user")
	defer func() {
		_, _ = env.mainPool.Exec(ctx, "DELETE FROM platform.users WHERE id = $1::uuid", userID)
	}()

	err = env.storage.AssignUserToTenant(ctx, userID, env.separateT.ID)
	require.NoError(t, err)
	defer func() {
		_ = env.storage.RemoveUserFromTenant(ctx, userID, env.separateT.ID)
	}()

	var capturedPool *pgxpool.Pool
	var capturedTenantID string
	var capturedSource string
	var capturedRole string

	tenantID := env.separateT.ID
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user_id", userID)
		c.Locals("is_instance_admin", false)
		c.Locals("claims", &auth.TokenClaims{
			UserID:     userID,
			TenantID:   &tenantID,
			TenantRole: "tenant_admin",
		})
		return c.Next()
	})
	app.Use(TenantDBMiddleware(TenantDBConfig{
		Storage: env.storage,
		Router:  env.router,
	}))
	app.Get("/test", func(c fiber.Ctx) error {
		capturedPool = GetTenantPool(c)
		capturedTenantID, _ = c.Locals("tenant_id").(string)
		capturedSource, _ = c.Locals("tenant_source").(string)
		capturedRole, _ = c.Locals("tenant_role").(string)
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	assert.NotNil(t, capturedPool, "tenant_db should be set for JWT-sourced separate-DB tenant")
	assert.Equal(t, env.separateT.ID, capturedTenantID)
	assert.Equal(t, "jwt", capturedSource)
	assert.Equal(t, "tenant_admin", capturedRole)
}

func TestGetPoolForSchema_Integration_TenantPoolForAllSchemas(t *testing.T) {
	env := setupMiddlewareIntegration(t)
	ctx := context.Background()

	// Get the tenant pool via the router
	tenantPool, err := env.router.GetPool(env.separateT.ID)
	require.NoError(t, err)

	var currentDB string
	err = tenantPool.QueryRow(ctx, "SELECT current_database()").Scan(&currentDB)
	require.NoError(t, err)
	assert.Equal(t, env.separateDB, currentDB)

	schemas := []string{"public", "auth", "storage", "app", "jobs", "ai", "rpc", "mcp"}
	for _, schema := range schemas {
		t.Run("schema_"+schema, func(t *testing.T) {
			app := fiber.New()
			var capturedPool *pgxpool.Pool

			app.Get("/test", func(c fiber.Ctx) error {
				c.Locals("tenant_db", tenantPool)
				capturedPool = GetPoolForSchema(c, schema, env.mainPool)
				return c.SendString("OK")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			_, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tenantPool, capturedPool,
				"GetPoolForSchema should return tenant pool for schema %q", schema)
		})
	}
}
