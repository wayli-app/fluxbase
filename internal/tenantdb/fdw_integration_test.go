//go:build integration

package tenantdb

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/database/bootstrap"
	"github.com/nimbleflux/fluxbase/test/dbhelpers"
)

// buildTenantDBURL constructs a database URL for a specific database name
// by replacing the database in the main connection URL.
func buildTenantDBURL(baseURL, dbName string) string {
	// baseURL is like postgresql://user:pass@host:5432/fluxbase_dev?sslmode=disable
	// We need to replace fluxbase_dev with the new dbName
	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL + dbName // fallback
	}
	u.Path = "/" + dbName
	return u.String()
}

// setupFDWTest creates a temporary tenant database, bootstraps it, and returns
// both the main pool, tenant admin pool (superuser), and FDW config along with a cleanup function.
func setupFDWTest(t *testing.T) (mainPool, tenantPool *pgxpool.Pool, cfg FDWConfig, cleanup func()) {
	t.Helper()

	testCtx := dbhelpers.NewDBTestContext(t)
	mainPool = testCtx.Pool

	dbName := fmt.Sprintf("fdw_test_%s", uuid.New().String()[:8])
	ctx := context.Background()

	// Build admin URL for creating databases and bootstrapping (needs superuser)
	testCfg := dbhelpers.GetTestConfig()
	adminDBURL := fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s?sslmode=%s",
		testCfg.Database.AdminUser,
		testCfg.Database.AdminPassword,
		testCfg.Database.Host,
		testCfg.Database.Port,
		testCfg.Database.Database,
		testCfg.Database.SSLMode,
	)
	mainAdminPool, err := pgxpool.New(ctx, adminDBURL)
	require.NoError(t, err, "Failed to create main admin pool")

	// Create the tenant database using admin pool
	_, err = mainAdminPool.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s ENCODING 'UTF8'", dbName))
	require.NoError(t, err, "Failed to create test tenant database")

	// Build tenant DB URL using admin credentials and bootstrap it
	tenantAdminURL := buildTenantDBURL(adminDBURL, dbName)
	err = bootstrap.RunBootstrapOnDB(ctx, tenantAdminURL, "fluxbase_app")
	require.NoError(t, err, "Failed to bootstrap tenant database")

	// Connect to the tenant database as admin (needed for FDW setup)
	tenantPoolCfg, err := pgxpool.ParseConfig(tenantAdminURL)
	require.NoError(t, err)
	tenantPoolCfg.MaxConns = 5
	tenantPool, err = pgxpool.NewWithConfig(ctx, tenantPoolCfg)
	require.NoError(t, err, "Failed to connect to tenant database as admin")

	// Parse FDW config from the main DB URL (using admin credentials for FDW user mapping)
	cfg, err = ParseFDWConfig(adminDBURL)
	require.NoError(t, err, "Failed to parse FDW config from database URL")

	cleanup = func() {
		if tenantPool != nil {
			tenantPool.Close()
		}
		// Terminate connections and drop the test database using admin pool
		_, _ = mainAdminPool.Exec(ctx, fmt.Sprintf(
			"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s' AND pid <> pg_backend_pid()",
			dbName,
		))
		_, _ = mainAdminPool.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
		mainAdminPool.Close()
		testCtx.Close()
	}

	return mainPool, tenantPool, cfg, cleanup
}

func TestSetupFDW_CreatesForeignServerAndTables(t *testing.T) {
	mainPool, tenantPool, cfg, cleanup := setupFDWTest(t)
	defer cleanup()

	ctx := context.Background()

	err := SetupFDW(ctx, tenantPool, cfg, nil)
	require.NoError(t, err, "SetupFDW should succeed")

	// Verify foreign server exists
	var serverExists bool
	err = tenantPool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_foreign_server WHERE srvname = 'main_server'
		)
	`).Scan(&serverExists)
	require.NoError(t, err)
	assert.True(t, serverExists, "Foreign server 'main_server' should exist")

	// Verify user mapping exists
	var mappingExists bool
	err = tenantPool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_user_mapping WHERE umserver = (
				SELECT oid FROM pg_foreign_server WHERE srvname = 'main_server'
			)
		)
	`).Scan(&mappingExists)
	require.NoError(t, err)
	assert.True(t, mappingExists, "User mapping should exist")

	// Verify foreign tables were imported (default list includes users + identities,
	// but IMPORT FOREIGN SCHEMA only imports tables that exist in the source schema)
	var foreignTableCount int
	err = tenantPool.QueryRow(ctx, `
		SELECT COUNT(*) FROM information_schema.foreign_tables
		WHERE foreign_table_schema = 'auth'
	`).Scan(&foreignTableCount)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, foreignTableCount, 1, "Should import at least auth.users")

	// Verify auth.users foreign table exists (it always exists in the main DB)
	var usersExists bool
	err = tenantPool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.foreign_tables
			WHERE foreign_table_schema = 'auth' AND foreign_table_name = 'users'
		)
	`).Scan(&usersExists)
	require.NoError(t, err)
	assert.True(t, usersExists, "Foreign table auth.users should exist")

	// Cross-reference: the main database should have the auth.users table
	var mainUsersExists bool
	err = mainPool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'auth' AND table_name = 'users'
		)
	`).Scan(&mainUsersExists)
	require.NoError(t, err)
	assert.True(t, mainUsersExists, "Main DB should have auth.users table")
}

func TestSetupFDW_CustomTableList(t *testing.T) {
	_, tenantPool, cfg, cleanup := setupFDWTest(t)
	defer cleanup()

	ctx := context.Background()

	err := SetupFDW(ctx, tenantPool, cfg, []string{"users"})
	require.NoError(t, err, "SetupFDW with custom table list should succeed")

	var foreignTableCount int
	err = tenantPool.QueryRow(ctx, `
		SELECT COUNT(*) FROM information_schema.foreign_tables
		WHERE foreign_table_schema = 'auth'
	`).Scan(&foreignTableCount)
	require.NoError(t, err)
	assert.Equal(t, 1, foreignTableCount, "Should import only the specified table")
}

func TestSetupFDW_IsIdempotent(t *testing.T) {
	_, tenantPool, cfg, cleanup := setupFDWTest(t)
	defer cleanup()

	ctx := context.Background()

	// First call
	err := SetupFDW(ctx, tenantPool, cfg, nil)
	require.NoError(t, err, "First SetupFDW should succeed")

	// Second call (should succeed due to IF NOT EXISTS)
	err = SetupFDW(ctx, tenantPool, cfg, nil)
	require.NoError(t, err, "Second SetupFDW should succeed (idempotent)")

	// Verify tables still exist and aren't duplicated
	var foreignTableCount int
	err = tenantPool.QueryRow(ctx, `
		SELECT COUNT(*) FROM information_schema.foreign_tables
		WHERE foreign_table_schema = 'auth'
	`).Scan(&foreignTableCount)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, foreignTableCount, 1, "Should still have at least auth.users foreign table")
}

func TestTeardownFDW_RemovesAllArtifacts(t *testing.T) {
	_, tenantPool, cfg, cleanup := setupFDWTest(t)
	defer cleanup()

	ctx := context.Background()

	// Set up FDW first
	err := SetupFDW(ctx, tenantPool, cfg, nil)
	require.NoError(t, err)

	// Teardown
	err = TeardownFDW(ctx, tenantPool)
	require.NoError(t, err, "TeardownFDW should succeed")

	// Verify foreign server is gone
	var serverExists bool
	err = tenantPool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_foreign_server WHERE srvname = 'main_server'
		)
	`).Scan(&serverExists)
	require.NoError(t, err)
	assert.False(t, serverExists, "Foreign server should be removed")

	// Verify foreign tables are gone (cascade)
	var foreignTableCount int
	err = tenantPool.QueryRow(ctx, `
		SELECT COUNT(*) FROM information_schema.foreign_tables
		WHERE foreign_table_schema = 'auth'
	`).Scan(&foreignTableCount)
	require.NoError(t, err)
	assert.Equal(t, 0, foreignTableCount, "All foreign tables should be removed")
}

func TestSetupFDW_DropsLocalTablesBeforeImport(t *testing.T) {
	_, tenantPool, cfg, cleanup := setupFDWTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a local auth.users table in the tenant DB (simulating pre-bootstrap state)
	_, err := tenantPool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS auth.users (
			id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
			email text NOT NULL
		)
	`)
	require.NoError(t, err, "Failed to create local auth.users")

	// Verify local table exists before FDW
	var localExists bool
	err = tenantPool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'auth' AND table_name = 'users' AND table_type = 'BASE TABLE'
		)
	`).Scan(&localExists)
	require.NoError(t, err)
	assert.True(t, localExists, "Local auth.users should exist before FDW setup")

	// SetupFDW should drop the local table and replace with foreign table
	err = SetupFDW(ctx, tenantPool, cfg, []string{"users"})
	require.NoError(t, err, "SetupFDW should succeed even with pre-existing local tables")

	// Verify it's now a foreign table (not a base table)
	var isBaseTable bool
	err = tenantPool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'auth' AND table_name = 'users' AND table_type = 'BASE TABLE'
		)
	`).Scan(&isBaseTable)
	require.NoError(t, err)
	assert.False(t, isBaseTable, "Local auth.users should be dropped")

	var isForeignTable bool
	err = tenantPool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.foreign_tables
			WHERE foreign_table_schema = 'auth' AND foreign_table_name = 'users'
		)
	`).Scan(&isForeignTable)
	require.NoError(t, err)
	assert.True(t, isForeignTable, "auth.users should be a foreign table")
}

func TestFDW_CrossDBQueryWorks(t *testing.T) {
	mainPool, tenantPool, cfg, cleanup := setupFDWTest(t)
	defer cleanup()

	ctx := context.Background()

	// Setup FDW
	err := SetupFDW(ctx, tenantPool, cfg, []string{"users"})
	require.NoError(t, err, "SetupFDW should succeed")

	// Insert a user in the main database's auth.users table
	testEmail := fmt.Sprintf("fdw-test-%s@example.com", uuid.New().String()[:8])
	testUserID := uuid.New().String()

	// Insert user using the app user (fluxbase_app has BYPASSRLS)
	_, err = mainPool.Exec(ctx, `
		INSERT INTO auth.users (id, email, password_hash, created_at, updated_at)
		VALUES ($1::uuid, $2, '', NOW(), NOW())
	`, testUserID, testEmail)
	require.NoError(t, err, "Failed to insert test user in main DB")
	defer func() {
		_, _ = mainPool.Exec(ctx, "DELETE FROM auth.users WHERE id = $1::uuid", testUserID)
	}()

	// Query the user via the tenant pool's foreign table — this proves FDW works
	var email string
	err = tenantPool.QueryRow(ctx, `
		SELECT email FROM auth.users WHERE id = $1::uuid
	`, testUserID).Scan(&email)
	require.NoError(t, err, "Cross-DB query via FDW should succeed")
	assert.Equal(t, testEmail, email, "Cross-DB query should return the correct data")

	// Also test a JOIN: query the foreign table from within a more complex statement
	var count int
	err = tenantPool.QueryRow(ctx, `
		SELECT COUNT(*) FROM auth.users WHERE email LIKE 'fdw-test-%'
	`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "Should find the inserted user via foreign table")
}

func TestFDW_CrossDBQueryWithTimeout(t *testing.T) {
	_, tenantPool, cfg, cleanup := setupFDWTest(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := SetupFDW(ctx, tenantPool, cfg, []string{"users"})
	require.NoError(t, err, "SetupFDW should succeed with timeout context")

	// Query should work within the timeout
	var count int
	err = tenantPool.QueryRow(ctx, `SELECT COUNT(*) FROM auth.users`).Scan(&count)
	require.NoError(t, err, "Query via FDW should succeed within timeout")
}
