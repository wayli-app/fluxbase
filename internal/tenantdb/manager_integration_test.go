//go:build integration

package tenantdb

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/test/dbhelpers"
)

type managerTestEnv struct {
	mainPool  *pgxpool.Pool
	adminPool *pgxpool.Pool
	storage   *Repository
	router    *Router
	manager   *Manager
	dbURL     string
	adminURL  string
	fdwCfg    FDWConfig
	testCtx   *dbhelpers.DBTestContext
}

func setupManagerTest(t *testing.T) *managerTestEnv {
	t.Helper()

	testCtx := dbhelpers.NewDBTestContext(t)
	mainPool := testCtx.Pool
	ctx := context.Background()

	// Build admin URL for creating databases (needs superuser)
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

	// Create Storage from main pool
	storage := NewRepository(mainPool)

	// Count existing tenants so we can account for them in MaxTenants checks
	existingCount, err := storage.CountTenants(ctx)
	require.NoError(t, err, "Failed to count existing tenants")

	// Tenant config — use unlimited max so the setup itself succeeds
	tenantCfg := Config{
		Enabled:        true,
		DatabasePrefix: "tenant_",
		MaxTenants:     0, // unlimited
		Pool: PoolConfig{
			MaxTotalConnections: 100,
			EvictionAge:         30 * 60 * 1e9, // 30 minutes in nanoseconds
		},
	}

	// Use the app user URL for the Manager and Router (matches production pattern)
	dbURL := testCtx.DatabaseURL()

	// Create Router
	router := NewRouter(storage, tenantCfg, mainPool, adminPool, dbURL)

	// Create Manager
	manager := NewManager(storage, tenantCfg, adminPool, dbURL)
	manager.SetRouter(router)
	manager.SetAdminDBURL(adminDBURL)

	// Parse FDW config from the admin DB URL
	fdwCfg, err := ParseFDWConfig(adminDBURL)
	require.NoError(t, err, "Failed to parse FDW config")
	manager.SetFDWConfig(fdwCfg)

	env := &managerTestEnv{
		mainPool:  mainPool,
		adminPool: adminPool,
		storage:   storage,
		router:    router,
		manager:   manager,
		dbURL:     dbURL,
		adminURL:  adminDBURL,
		fdwCfg:    fdwCfg,
		testCtx:   testCtx,
	}

	t.Cleanup(func() {
		router.Close()
		adminPool.Close()
		testCtx.Close()
	})

	// Store existing count for tests that need it
	t.Logf("Existing tenants: %d", existingCount)

	return env
}

// dropTenantDatabase terminates connections and drops the physical database,
// then hard-deletes the tenant record from storage.
func dropTenantDatabase(t *testing.T, env *managerTestEnv, tenant *Tenant) {
	t.Helper()
	ctx := context.Background()

	if tenant.DBName != nil && *tenant.DBName != "" {
		_, _ = env.adminPool.Exec(ctx, fmt.Sprintf(
			"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s' AND pid <> pg_backend_pid()",
			*tenant.DBName,
		))
		_, _ = env.adminPool.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", *tenant.DBName))
	}
	_ = env.storage.HardDeleteTenant(ctx, tenant.ID)
}

func TestManager_CreateTenantDatabase_FullPipeline(t *testing.T) {
	env := setupManagerTest(t)
	ctx := context.Background()

	slug := fmt.Sprintf("mit_full_%s", uuid.New().String()[:8])
	tenant, err := env.manager.CreateTenantDatabase(ctx, CreateTenantRequest{
		Slug: slug,
		Name: "Full Pipeline Test",
	})
	require.NoError(t, err, "CreateTenantDatabase should succeed")
	require.NotNil(t, tenant, "Tenant should not be nil")
	assert.NotEmpty(t, tenant.ID, "Tenant ID should be set")
	assert.Equal(t, TenantStatusActive, tenant.Status, "Tenant status should be active")

	t.Cleanup(func() {
		dropTenantDatabase(t, env, tenant)
	})

	// Verify tenant record exists in storage with DBName set
	stored, err := env.storage.GetTenant(ctx, tenant.ID)
	require.NoError(t, err, "GetTenant should succeed")
	require.NotNil(t, stored.DBName, "DBName should be set")
	assert.NotEmpty(t, *stored.DBName, "DBName should not be empty")

	// Verify the database actually exists
	var dbExists bool
	err = env.adminPool.QueryRow(
		ctx,
		"SELECT EXISTS(SELECT datname FROM pg_database WHERE datname = $1)",
		*stored.DBName,
	).Scan(&dbExists)
	require.NoError(t, err)
	assert.True(t, dbExists, "Tenant database should exist in pg_database")

	// Verify schemas are bootstrapped by connecting through the router
	tenantPool, err := env.router.GetPool(tenant.ID)
	require.NoError(t, err, "GetPool should succeed for active tenant")

	rows, err := tenantPool.Query(ctx, `
		SELECT schema_name FROM information_schema.schemata
		WHERE schema_name IN ('auth', 'public', 'storage')
	`)
	require.NoError(t, err)
	defer rows.Close()

	schemas := map[string]bool{}
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		schemas[name] = true
	}
	assert.True(t, schemas["auth"], "auth schema should exist in tenant DB")
	assert.True(t, schemas["public"], "public schema should exist in tenant DB")

	// Verify FDW is configured (foreign tables in auth schema)
	var foreignTableCount int
	err = tenantPool.QueryRow(ctx, `
		SELECT COUNT(*) FROM information_schema.foreign_tables
		WHERE foreign_table_schema = 'auth'
	`).Scan(&foreignTableCount)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, foreignTableCount, 1,
		"Should have at least one foreign table (auth.users) from FDW setup")
}

func TestManager_CreateTenantDatabase_FDW_CrossDBQuery(t *testing.T) {
	env := setupManagerTest(t)
	ctx := context.Background()

	slug := fmt.Sprintf("mit_fdw_%s", uuid.New().String()[:8])
	tenant, err := env.manager.CreateTenantDatabase(ctx, CreateTenantRequest{
		Slug: slug,
		Name: "FDW Cross-DB Query Test",
	})
	require.NoError(t, err, "CreateTenantDatabase should succeed")

	t.Cleanup(func() {
		dropTenantDatabase(t, env, tenant)
	})

	// Insert a user in the main database's auth.users table
	testUserID := uuid.New().String()
	testEmail := fmt.Sprintf("mit_fdw_%s@example.com", uuid.New().String()[:8])

	_, err = env.mainPool.Exec(ctx, `
		INSERT INTO auth.users (id, email, password_hash, created_at, updated_at)
		VALUES ($1::uuid, $2, '', NOW(), NOW())
	`, testUserID, testEmail)
	require.NoError(t, err, "Failed to insert test user in main DB")

	// Clean up user after test
	t.Cleanup(func() {
		_, _ = env.mainPool.Exec(ctx, "DELETE FROM auth.users WHERE id = $1::uuid", testUserID)
	})

	// Get tenant pool via router
	tenantPool, err := env.router.GetPool(tenant.ID)
	require.NoError(t, err, "GetPool should succeed")

	// Query via FDW: the tenant DB should see the main DB's auth.users
	var email string
	err = tenantPool.QueryRow(ctx, `
		SELECT email FROM auth.users WHERE id = $1::uuid
	`, testUserID).Scan(&email)
	require.NoError(t, err, "Cross-DB query via FDW should succeed")
	assert.Equal(t, testEmail, email,
		"Cross-DB query should return the correct email from the main database")
}

func TestManager_DeleteTenantDatabase_CleansUp(t *testing.T) {
	env := setupManagerTest(t)
	ctx := context.Background()

	// Create a tenant first
	slug := fmt.Sprintf("mit_del_%s", uuid.New().String()[:8])
	tenant, err := env.manager.CreateTenantDatabase(ctx, CreateTenantRequest{
		Slug: slug,
		Name: "Delete Test Tenant",
	})
	require.NoError(t, err, "CreateTenantDatabase should succeed")

	dbName := *tenant.DBName

	// Verify tenant exists in storage
	_, err = env.storage.GetTenant(ctx, tenant.ID)
	require.NoError(t, err, "Tenant should exist in storage before deletion")

	// Delete the tenant
	err = env.manager.DeleteTenantDatabase(ctx, tenant.ID)
	require.NoError(t, err, "DeleteTenantDatabase should succeed")

	// Verify storage returns ErrTenantNotFound
	_, err = env.storage.GetTenant(ctx, tenant.ID)
	assert.Equal(t, ErrTenantNotFound, err,
		"GetTenant should return ErrTenantNotFound after deletion")

	// Verify database is dropped
	var dbExists bool
	err = env.adminPool.QueryRow(
		ctx,
		"SELECT EXISTS(SELECT datname FROM pg_database WHERE datname = $1)",
		dbName,
	).Scan(&dbExists)
	require.NoError(t, err)
	assert.False(t, dbExists, "Tenant database should be dropped after deletion")
}

func TestManager_CreateTenantDatabase_MaxTenantsExceeded(t *testing.T) {
	env := setupManagerTest(t)
	ctx := context.Background()

	// Count existing tenants so set set MaxTenants appropriately
	existingCount, err := env.storage.CountTenants(ctx)
	require.NoError(t, err, "Failed to count existing tenants")

	// Create a separate manager with MaxTenants = existingCount + 1
	limitedCfg := Config{
		Enabled:        true,
		DatabasePrefix: "tenant_",
		MaxTenants:     existingCount + 1,
		Pool: PoolConfig{
			MaxTotalConnections: 100,
			EvictionAge:         30 * 60 * 1e9,
		},
	}
	limitedManager := NewManager(env.storage, limitedCfg, env.adminPool, env.dbURL)
	limitedManager.SetRouter(env.router)
	limitedManager.SetFDWConfig(env.fdwCfg)
	limitedManager.SetAdminDBURL(env.adminURL)

	// Create first tenant -- should succeed (brings count to existingCount + 1)
	slug1 := fmt.Sprintf("mit_max1_%s", uuid.New().String()[:8])
	tenant1, err := limitedManager.CreateTenantDatabase(ctx, CreateTenantRequest{
		Slug: slug1,
		Name: "First Tenant",
	})
	require.NoError(t, err, "First tenant creation should succeed")

	// Create second tenant -- should fail with ErrMaxTenantsReached
	slug2 := fmt.Sprintf("mit_max2_%s", uuid.New().String()[:8])
	_, err = limitedManager.CreateTenantDatabase(ctx, CreateTenantRequest{
		Slug: slug2,
		Name: "Second Tenant",
	})
	assert.Equal(t, ErrMaxTenantsReached, err,
		"Second tenant creation should fail with ErrMaxTenantsReached")

	// Cleanup: drop the first tenant's database and record
	dropTenantDatabase(t, env, tenant1)
}
