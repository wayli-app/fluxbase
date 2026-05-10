//go:build integration

package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/database/bootstrap"
	"github.com/nimbleflux/fluxbase/internal/tenantdb"
	"github.com/nimbleflux/fluxbase/test/dbhelpers"
)

// rlsTestEnv holds all resources needed for an RLS integration test.
type rlsTestEnv struct {
	mainPool  *pgxpool.Pool
	adminPool *pgxpool.Pool
	storage   *tenantdb.Repository
	router    *tenantdb.Router
	tenant    *tenantdb.Tenant
	tenantDB  string
	dbConn    *database.Connection
	testCtx   *dbhelpers.DBTestContext
	dbURL     string
	adminURL  string
}

// setupRLSIntegrationTest creates a tenant with a separate database for RLS tests.
func setupRLSIntegrationTest(t *testing.T) *rlsTestEnv {
	t.Helper()

	testCtx := dbhelpers.NewDBTestContext(t)
	mainPool := testCtx.Pool
	ctx := context.Background()

	cfg := dbhelpers.GetTestConfig()

	// Create admin pool for CREATE DATABASE and privileged operations.
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

	// Create Storage.
	storage := tenantdb.NewRepository(mainPool)

	// Create a tenant with a separate database.
	tenantDBName := fmt.Sprintf("tenant_rls_%s", uuid.New().String()[:8])
	tenant := &tenantdb.Tenant{
		Slug:     fmt.Sprintf("rls-test-%s", uuid.New().String()[:8]),
		Name:     "RLS Integration Test Tenant",
		DBName:   &tenantDBName,
		Status:   tenantdb.TenantStatusActive,
		Metadata: map[string]any{},
	}
	err = storage.CreateTenant(ctx, tenant)
	require.NoError(t, err, "Failed to create tenant")

	// Create the actual database.
	_, err = adminPool.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s ENCODING 'UTF8'", tenantDBName))
	require.NoError(t, err, "Failed to create tenant database")

	// Bootstrap the tenant database using admin credentials.
	tenantAdminURL := buildTenantRLSDBURL(adminDBURL, tenantDBName)
	err = bootstrap.RunBootstrapOnDB(ctx, tenantAdminURL, "fluxbase_app")
	require.NoError(t, err, "Failed to bootstrap tenant database")

	// Create Router.
	router := tenantdb.NewRouter(storage, tenantdb.DefaultConfig(), mainPool, adminPool, testCtx.DatabaseURL())

	// Update tenant DB name in storage.
	err = storage.UpdateTenantDBName(ctx, tenant.ID, tenantDBName)
	require.NoError(t, err, "Failed to update tenant DB name")

	// Refresh tenant from DB.
	tenant, err = storage.GetTenant(ctx, tenant.ID)
	require.NoError(t, err, "Failed to refresh tenant")

	env := &rlsTestEnv{
		mainPool:  mainPool,
		adminPool: adminPool,
		storage:   storage,
		router:    router,
		tenant:    tenant,
		tenantDB:  tenantDBName,
		dbConn:    database.NewConnectionWithPool(mainPool),
		testCtx:   testCtx,
		dbURL:     testCtx.DatabaseURL(),
		adminURL:  adminDBURL,
	}

	t.Cleanup(func() {
		router.Close()

		// Remove tenant records.
		_, _ = mainPool.Exec(ctx, "DELETE FROM platform.tenants WHERE id = $1", tenant.ID)

		// Terminate connections and drop the tenant database.
		_, _ = adminPool.Exec(ctx, fmt.Sprintf(
			"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s' AND pid <> pg_backend_pid()",
			tenantDBName,
		))
		_, _ = adminPool.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", tenantDBName))

		adminPool.Close()
		testCtx.Close()
	})

	return env
}

// buildTenantRLSDBURL constructs a database URL for a specific database name
// by replacing the database in a base connection URL.
func buildTenantRLSDBURL(baseURL, dbName string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL + dbName // fallback
	}
	u.Path = "/" + dbName
	return u.String()
}

func TestSetRLSContext_SetsSessionVariables(t *testing.T) {
	env := setupRLSIntegrationTest(t)
	ctx := context.Background()

	// Get tenant pool from router.
	tenantPool, err := env.router.GetPool(env.tenant.ID)
	require.NoError(t, err, "Failed to get tenant pool")

	// Begin a transaction on the tenant pool.
	tx, err := tenantPool.Begin(ctx)
	require.NoError(t, err, "Failed to begin transaction")
	defer func() { _ = tx.Rollback(ctx) }()

	// Create claims with tenant, user, and email.
	tenantID := env.tenant.ID
	userID := uuid.New().String()
	claims := &auth.TokenClaims{
		UserID:   userID,
		Email:    "rls-test@example.com",
		TenantID: &tenantID,
	}

	// Call SetRLSContext.
	err = SetRLSContext(ctx, tx, userID, "authenticated", claims)
	require.NoError(t, err, "SetRLSContext failed")

	// Query the JWT claims session variable.
	var claimsJSON string
	err = tx.QueryRow(ctx, "SELECT current_setting('request.jwt.claims', true)").Scan(&claimsJSON)
	require.NoError(t, err, "Failed to query request.jwt.claims")

	// Parse and assert claims.
	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(claimsJSON), &parsed)
	require.NoError(t, err, "Failed to parse JWT claims JSON")

	assert.Equal(t, userID, parsed["sub"], "JWT claims 'sub' should match userID")
	assert.Equal(t, "authenticated", parsed["role"], "JWT claims 'role' should be 'authenticated'")
	assert.Equal(t, tenantID, parsed["tenant_id"], "JWT claims 'tenant_id' should match tenant ID")
}

func TestSetRLSContext_SetsTenantID(t *testing.T) {
	env := setupRLSIntegrationTest(t)
	ctx := context.Background()

	tenantPool, err := env.router.GetPool(env.tenant.ID)
	require.NoError(t, err, "Failed to get tenant pool")

	tx, err := tenantPool.Begin(ctx)
	require.NoError(t, err, "Failed to begin transaction")
	defer func() { _ = tx.Rollback(ctx) }()

	tenantID := env.tenant.ID
	userID := uuid.New().String()
	claims := &auth.TokenClaims{
		UserID:   userID,
		TenantID: &tenantID,
	}

	err = SetRLSContext(ctx, tx, userID, "authenticated", claims)
	require.NoError(t, err, "SetRLSContext failed")

	// Query app.current_tenant_id.
	var currentTenantID string
	err = tx.QueryRow(ctx, "SELECT current_setting('app.current_tenant_id', true)").Scan(&currentTenantID)
	require.NoError(t, err, "Failed to query app.current_tenant_id")

	assert.Equal(t, tenantID, currentTenantID, "app.current_tenant_id should match claims.TenantID")
}

func TestSetRLSContext_SetsLocalRole(t *testing.T) {
	env := setupRLSIntegrationTest(t)
	ctx := context.Background()

	tenantPool, err := env.router.GetPool(env.tenant.ID)
	require.NoError(t, err, "Failed to get tenant pool")

	tx, err := tenantPool.Begin(ctx)
	require.NoError(t, err, "Failed to begin transaction")
	defer func() { _ = tx.Rollback(ctx) }()

	// Call SetRLSContext with nil claims.
	err = SetRLSContext(ctx, tx, "user-1", "authenticated", nil)
	require.NoError(t, err, "SetRLSContext failed")

	// Query current_user within the same transaction.
	var currentUser string
	err = tx.QueryRow(ctx, "SELECT current_user").Scan(&currentUser)
	require.NoError(t, err, "Failed to query current_user")

	assert.Equal(t, "authenticated", currentUser, "current_user should be set to 'authenticated'")
}

func TestSetTenantDBSessionContext_SetsTenantID(t *testing.T) {
	env := setupRLSIntegrationTest(t)
	ctx := context.Background()

	tenantPool, err := env.router.GetPool(env.tenant.ID)
	require.NoError(t, err, "Failed to get tenant pool")

	tx, err := tenantPool.Begin(ctx)
	require.NoError(t, err, "Failed to begin transaction")
	defer func() { _ = tx.Rollback(ctx) }()

	tenantID := env.tenant.ID

	err = SetTenantSessionContext(ctx, tx, tenantID)
	require.NoError(t, err, "SetTenantSessionContext failed")

	var currentTenantID string
	err = tx.QueryRow(ctx, "SELECT current_setting('app.current_tenant_id', true)").Scan(&currentTenantID)
	require.NoError(t, err, "Failed to query app.current_tenant_id")

	assert.Equal(t, tenantID, currentTenantID, "app.current_tenant_id should match tenantID")
}

func TestSetTenantDBSessionContext_EmptyTenantID_Noop(t *testing.T) {
	env := setupRLSIntegrationTest(t)
	ctx := context.Background()

	tenantPool, err := env.router.GetPool(env.tenant.ID)
	require.NoError(t, err, "Failed to get tenant pool")

	tx, err := tenantPool.Begin(ctx)
	require.NoError(t, err, "Failed to begin transaction")
	defer func() { _ = tx.Rollback(ctx) }()

	err = SetTenantSessionContext(ctx, tx, "")
	require.NoError(t, err, "SetTenantSessionContext with empty tenantID should not error")

	var currentTenantID *string
	err = tx.QueryRow(ctx, "SELECT current_setting('app.current_tenant_id', true)").Scan(&currentTenantID)
	require.NoError(t, err, "Failed to query app.current_tenant_id")

	assert.Nil(t, currentTenantID, "app.current_tenant_id should be NULL when not set")
}

func TestWrapWithRLS_TenantPool_RoutesCorrectly(t *testing.T) {
	env := setupRLSIntegrationTest(t)
	ctx := context.Background()

	tenantPool, err := env.router.GetPool(env.tenant.ID)
	require.NoError(t, err, "Failed to get tenant pool")

	var currentDB string

	app := fiber.New()
	app.Get("/test", func(c fiber.Ctx) error {
		// Set tenant pool and RLS context in Fiber locals.
		c.Locals("tenant_db", tenantPool)
		c.Locals("rls_user_id", "test-user")
		c.Locals("rls_role", "authenticated")

		return WrapWithRLS(ctx, env.dbConn, c, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, "SELECT current_database()").Scan(&currentDB)
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	assert.Equal(t, env.tenantDB, currentDB,
		"WrapWithRLS should route to the tenant database when tenant_db is set")
}

func TestWrapWithRLS_TenantPool_SetsRLSContext(t *testing.T) {
	env := setupRLSIntegrationTest(t)
	ctx := context.Background()

	tenantPool, err := env.router.GetPool(env.tenant.ID)
	require.NoError(t, err, "Failed to get tenant pool")

	tenantID := env.tenant.ID
	var capturedTenantID string

	app := fiber.New()
	app.Get("/test", func(c fiber.Ctx) error {
		c.Locals("tenant_db", tenantPool)
		c.Locals("rls_user_id", "test-user")
		c.Locals("rls_role", "authenticated")
		c.Locals("jwt_claims", &auth.TokenClaims{
			UserID:   "test-user",
			TenantID: &tenantID,
		})

		return WrapWithRLS(ctx, env.dbConn, c, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, "SELECT current_setting('app.current_tenant_id', true)").Scan(&capturedTenantID)
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	assert.Equal(t, tenantID, capturedTenantID,
		"WrapWithRLS should set app.current_tenant_id from JWT claims")
}

func TestWrapWithRLS_MainPool_NoTenant(t *testing.T) {
	env := setupRLSIntegrationTest(t)
	ctx := context.Background()

	var currentDB string

	app := fiber.New()
	app.Get("/test", func(c fiber.Ctx) error {
		// Do NOT set tenant_db — should use main pool.
		c.Locals("rls_user_id", "test-user")
		c.Locals("rls_role", "authenticated")

		return WrapWithRLS(ctx, env.dbConn, c, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, "SELECT current_database()").Scan(&currentDB)
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	// The main database name comes from the test context config.
	assert.Equal(t, env.testCtx.Config.Database.Database, currentDB,
		"WrapWithRLS should use main pool when no tenant_db is set")
}
