// Package e2e contains end-to-end tests for Fluxbase.
//
// # Test Setup
//
// Tests run against a dedicated fluxbase_go_e2e database that is fully reset
// and re-bootstrapped on every test run. This makes the tests self-sufficient:
// no external setup step (make test-setup-db, CI schema steps) is needed.
//
// # Database Users
//
// Three database users are used for different purposes:
//
//  1. postgres (superuser) - Used for schema reset, bootstrap, and permission grants
//  2. fluxbase_app (has BYPASSRLS) - Used by NewTestContext for general testing
//  3. fluxbase_rls_test (no BYPASSRLS) - Used by NewRLSTestContext for RLS testing
//
// # Test Execution Flow
//
//  1. resetTestDatabase() - Drop all schemas, ensure users exist
//  2. bootstrapTestDatabase() - Run bootstrap SQL + declarative schema
//  3. setupTestTables() - Create products and tasks tables
//  4. Individual tests run
//  5. teardownTestTables() - Drop test tables
package e2e

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/database/bootstrap"
	dbschema "github.com/nimbleflux/fluxbase/internal/database/schema"
	"github.com/nimbleflux/fluxbase/internal/migrations"
	"github.com/nimbleflux/fluxbase/test"
)

// getEnvOrDefault returns the value of an environment variable or a default value.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getDatabase creates a database connection for setup/teardown operations.
func getDatabase(cfg *config.Config) (*database.Connection, error) {
	return database.NewConnection(cfg.Database)
}

// TestMain runs before all e2e tests to set up a fresh database and after all tests to clean up.
//
// Execution Flow:
//  1. resetTestDatabase() - Drop all non-system schemas, ensure test users exist
//  2. bootstrapTestDatabase() - Run bootstrap SQL + declarative schema application
//  3. CleanupE2ETestUsersGlobal() - Clean up any leftover test users
//  4. InitSharedTestContext() - Create server with connection pool
//  5. setupTestTables() - Create products and tasks tables with RLS policies
//  6. m.Run() - Run all test functions
//  7. teardownTestTables() - Drop test tables
func TestMain(m *testing.M) {
	ctx := context.Background()

	// Phase 1: Reset database to clean state
	if !resetTestDatabase() {
		log.Error().Msg("Failed to reset test database - cannot run tests")
		os.Exit(1)
	}

	// Phase 2: Bootstrap + declarative schema (same code path as server startup)
	if !bootstrapTestDatabase(ctx) {
		log.Error().Msg("Failed to bootstrap test database - cannot run tests")
		os.Exit(1)
	}

	// Phase 3: Clean up leftover users from previous runs
	cfg := test.GetTestConfig()
	test.CleanupE2ETestUsersGlobal(cfg)

	// Phase 4: Create shared server context
	test.InitSharedTestContext()

	// Phase 5: Create test tables (products, tasks, etc.)
	if !setupTestTables() {
		log.Error().Msg("Failed to setup test tables - cannot run tests")
		os.Exit(1)
	}

	// Refresh schema cache so the REST API knows about test tables
	refreshSharedTestContextSchemaCache()

	// Phase 6: Run all tests
	code := m.Run()

	// Cleanup: Remove test users created during this test run
	test.CleanupE2ETestUsersGlobal(cfg)

	// Teardown: Clean up test tables after all tests complete
	if os.Getenv("FLUXBASE_PARALLEL_TEST") != "true" {
		teardownTestTables()
	} else {
		log.Info().Msg("Parallel test mode detected - skipping test table teardown")
	}

	// Clean up shared test context before exit
	test.CleanupSharedTestContext()

	os.Exit(code)
}

// resetTestDatabase drops all non-system schemas and ensures test users exist.
// This mirrors the Playwright reset_playwright_db() in scripts/start-e2e-ui.sh.
func resetTestDatabase() bool {
	ctx := context.Background()

	cfg := test.GetTestConfig()
	cfg.Database.User = getEnvOrDefault("FLUXBASE_DATABASE_ADMIN_USER", "postgres")
	cfg.Database.AdminUser = cfg.Database.User
	cfg.Database.Password = getEnvOrDefault("FLUXBASE_DATABASE_ADMIN_PASSWORD", "postgres")

	db, err := getDatabase(cfg)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect as admin for database reset")
		return false
	}
	defer db.Close()

	dbName := cfg.Database.Database
	log.Info().Str("database", dbName).Msg("Resetting test database...")

	// Drop tenant databases from previous runs
	rows, err := db.Query(ctx, "SELECT datname FROM pg_database WHERE datname LIKE 'tenant_%'")
	if err != nil {
		log.Warn().Err(err).Msg("Failed to query tenant databases")
	} else {
		var tenantDBs []string
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err == nil {
				tenantDBs = append(tenantDBs, name)
			}
		}
		rows.Close()
		for _, name := range tenantDBs {
			_, _ = db.Exec(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS %q`, name))
		}
	}

	// Drop all non-system schemas (preserves extensions and roles).
	// Use a blocklist approach: system schemas + extension-managed schemas are kept.
	// TimescaleDB creates internal schemas (_timescaledb_*) that cannot be dropped.
	_, err = db.Exec(ctx, `DO $$ DECLARE r RECORD; BEGIN FOR r IN SELECT nspname FROM pg_namespace WHERE nspname NOT IN (
		'pg_catalog', 'information_schema', 'pg_toast',
		'_timescaledb_cache', '_timescaledb_catalog', '_timescaledb_config',
		'_timescaledb_functions', '_timescaledb_internal',
		'timescaledb_experimental', 'timescaledb_information'
	) LOOP EXECUTE 'DROP SCHEMA IF EXISTS ' || quote_ident(r.nspname) || ' CASCADE'; END LOOP; END $$`)
	if err != nil {
		log.Error().Err(err).Msg("Failed to drop schemas")
		return false
	}

	// Recreate public schema with basic grants
	_, err = db.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS public; GRANT ALL ON SCHEMA public TO public`)
	if err != nil {
		log.Error().Err(err).Msg("Failed to recreate public schema")
		return false
	}

	// Ensure test users exist
	_, _ = db.Exec(ctx, `CREATE USER fluxbase_app WITH PASSWORD 'fluxbase_app_password' LOGIN BYPASSRLS CREATEDB CREATEROLE`)
	_, _ = db.Exec(ctx, `CREATE USER fluxbase_rls_test WITH PASSWORD 'fluxbase_rls_test_password' LOGIN NOBYPASSRLS CREATEDB`)

	// Ensure BYPASSRLS is set (idempotent)
	_, _ = db.Exec(ctx, `ALTER USER fluxbase_app WITH BYPASSRLS`)

	// Grant database privileges
	_, _ = db.Exec(ctx, fmt.Sprintf(`GRANT ALL ON DATABASE %s TO fluxbase_app, fluxbase_rls_test`, dbName))
	_, _ = db.Exec(ctx, fmt.Sprintf(`GRANT CREATE ON DATABASE %s TO fluxbase_rls_test, fluxbase_app`, dbName))

	log.Info().Msg("Test database reset complete")
	return true
}

// bootstrapTestDatabase runs bootstrap SQL and declarative schema application.
// This uses the exact same Go code path as cmd/fluxbase/main.go startup.
func bootstrapTestDatabase(ctx context.Context) bool {
	cfg := test.GetTestConfig()

	log.Info().Msg("Running database bootstrap...")

	// Connect as admin for bootstrap
	adminCfg := *cfg
	adminCfg.Database.User = getEnvOrDefault("FLUXBASE_DATABASE_ADMIN_USER", "postgres")
	adminCfg.Database.AdminUser = adminCfg.Database.User
	adminCfg.Database.Password = getEnvOrDefault("FLUXBASE_DATABASE_ADMIN_PASSWORD", "postgres")

	adminDB, err := getDatabase(&adminCfg)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect as admin for bootstrap")
		return false
	}
	defer adminDB.Close()

	// Also connect as app user — bootstrap grants roles to CURRENT_USER
	appDB, err := getDatabase(cfg)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect as app user for bootstrap")
		return false
	}
	defer appDB.Close()

	// Run bootstrap as admin (creates schemas, extensions, roles)
	bootstrapSvc := bootstrap.NewServiceWithConfig(adminDB.Pool(), bootstrap.Config{
		Host:          cfg.Database.Host,
		Port:          cfg.Database.Port,
		Database:      cfg.Database.Database,
		User:          cfg.Database.User,
		Password:      cfg.Database.Password,
		AdminUser:     adminCfg.Database.User,
		AdminPassword: adminCfg.Database.Password,
	})
	if err := bootstrapSvc.EnsureBootstrap(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to run bootstrap")
		return false
	}
	bootstrapSvc.Close()
	log.Info().Msg("Database bootstrap completed successfully")

	// Extract embedded schema files to temp directory
	schemaDir, err := dbschema.ExtractSchemas()
	if err != nil {
		log.Error().Err(err).Msg("Failed to extract embedded schemas")
		return false
	}
	defer func() { _ = os.RemoveAll(schemaDir) }()

	// Apply declarative schema (tables, indexes, functions, policies)
	adminUser := adminCfg.Database.User
	adminPassword := adminCfg.Database.Password

	declarativeSvc := migrations.NewDeclarativeService(
		"pgschema",
		cfg.Database.Host,
		cfg.Database.Port,
		adminUser,
		adminPassword,
		cfg.Database.Database,
		migrations.DeclarativeConfig{
			SchemaDir:        schemaDir,
			Schemas:          migrations.DefaultFluxbaseSchemas,
			AllowDestructive: false,
			LockTimeout:      30,
		},
	)
	declarativeSvc.SetPool(adminDB.Pool())
	declarativeSvc.SetAppUser(cfg.Database.User)

	if err := declarativeSvc.ApplyDeclarativeWithSource(ctx, "fresh_install"); err != nil {
		log.Error().Err(err).Msg("Failed to apply declarative schema")
		return false
	}
	log.Info().Msg("Declarative schema applied successfully")

	// Grant role memberships to test users
	// Bootstrap creates roles and grants them to CURRENT_USER (postgres admin),
	// but fluxbase_app and fluxbase_rls_test also need them for SET ROLE support
	_, err = adminDB.Exec(ctx, `
		GRANT anon, authenticated, service_role, tenant_service, tenant_migration_role
		TO fluxbase_app, fluxbase_rls_test`)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to grant role memberships (may already exist)")
	}

	log.Info().Msg("Test database bootstrap complete")
	return true
}

// setupTestTables creates the test tables needed for e2e tests.
//
// # Tables Created
//
// 1. products table:
//   - Schema: id, name, price, created_at, updated_at
//   - RLS: Disabled (for general REST API testing)
//   - Purpose: Test basic CRUD operations without RLS complexity
//
// 2. tasks table:
//   - Schema: id, user_id, title, description, completed, is_public, created_at, updated_at
//   - RLS: Enabled and enforced
//   - Purpose: Test Row-Level Security policies
//   - Policies:
//   - tasks_select_own: Users can SELECT their own tasks OR public tasks
//   - tasks_insert_own: Users can INSERT tasks where user_id matches their ID
//   - tasks_update_own: Users can UPDATE only their own tasks
//   - tasks_delete_own: Users can DELETE only their own tasks
//
// enablePostGISExtension creates the PostGIS extension using the postgres superuser.
// This must be done before test tables are created since fluxbase_app lacks permission.
func enablePostGISExtension() {
	ctx := context.Background()

	cfg := test.GetTestConfig()
	// Use postgres superuser from environment variables (with defaults for local dev)
	cfg.Database.User = getEnvOrDefault("FLUXBASE_DATABASE_ADMIN_USER", "postgres")
	cfg.Database.AdminUser = cfg.Database.User
	cfg.Database.Password = getEnvOrDefault("FLUXBASE_DATABASE_ADMIN_PASSWORD", "postgres")

	db, err := getDatabase(cfg)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to connect as postgres for PostGIS setup")
		return
	}
	defer db.Close()

	// Check if PostGIS is available in the system
	var postgisAvailable bool
	err = db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_available_extensions WHERE name = 'postgis')").Scan(&postgisAvailable)
	if err != nil || !postgisAvailable {
		log.Debug().Msg("PostGIS extension not available in this database image")
		return
	}

	// Create PostGIS extension
	_, err = db.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS postgis`)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to create PostGIS extension")
	} else {
		log.Info().Msg("PostGIS extension enabled for e2e tests")
	}
}

// # Database Users
//
// This function uses fluxbase_app (with BYPASSRLS) to create tables,
// then calls grantRLSTestPermissions() to grant permissions to fluxbase_rls_test user.
// Returns true if setup succeeded, false otherwise.
func setupTestTables() bool {
	ctx := context.Background()

	// First, enable PostGIS extension using postgres superuser (if available)
	enablePostGISExtension()

	// Small delay to ensure PostGIS extension changes are fully committed
	// This helps avoid "conn closed" errors from stale connection state
	time.Sleep(100 * time.Millisecond)

	// Get database connection using fluxbase_app (has BYPASSRLS for setup)
	// Retry connection to handle transient failures after extension creation
	cfg := test.GetTestConfig()
	var db *database.Connection
	var err error
	for attempt := 1; attempt <= 3; attempt++ {
		db, err = getDatabase(cfg)
		if err == nil {
			break
		}
		log.Warn().Err(err).Int("attempt", attempt).Msg("Failed to connect to database, retrying...")
		time.Sleep(time.Duration(attempt*100) * time.Millisecond)
	}
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to database for test setup after retries")
		return false
	}
	defer db.Close()

	log.Info().Msg("Setting up e2e test tables...")

	// Drop products table first to ensure clean state (fixes permission denied errors)
	_, err = db.Exec(ctx, `DROP TABLE IF EXISTS public.products CASCADE`)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to drop products table (may not exist)")
	}

	// Create products table for REST tests (with retry for transient connection issues)
	for attempt := 1; attempt <= 3; attempt++ {
		_, err = db.Exec(ctx, `
			CREATE TABLE public.products (
				id INTEGER PRIMARY KEY GENERATED BY DEFAULT AS IDENTITY,
				name TEXT NOT NULL,
				price NUMERIC(10, 2) NOT NULL,
				created_at TIMESTAMPTZ DEFAULT NOW(),
				updated_at TIMESTAMPTZ DEFAULT NOW()
			)
		`)
		if err == nil {
			break
		}
		log.Warn().Err(err).Int("attempt", attempt).Msg("Failed to create products table, retrying...")
		time.Sleep(time.Duration(attempt*100) * time.Millisecond)
	}
	if err != nil {
		log.Error().Err(err).Msg("Failed to create products table")
		return false
	}

	// Disable RLS on products table (allow all access for testing)
	_, err = db.Exec(ctx, `ALTER TABLE public.products DISABLE ROW LEVEL SECURITY`)
	if err != nil {
		log.Debug().Err(err).Msg("Products table RLS may already be disabled")
	}

	// Grant all privileges on products table to fluxbase_app user
	_, err = db.Exec(ctx, `GRANT ALL ON TABLE public.products TO fluxbase_app`)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to grant privileges on products table")
	}

	// Grant usage on products sequence for INSERT operations
	_, err = db.Exec(ctx, `GRANT USAGE, SELECT ON SEQUENCE public.products_id_seq TO fluxbase_app`)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to grant sequence privileges")
	}

	// Create trigger for products updated_at
	_, err = db.Exec(ctx, `
		CREATE TRIGGER update_products_updated_at
		BEFORE UPDATE ON public.products
		FOR EACH ROW
		EXECUTE FUNCTION public.update_updated_at()
	`)
	if err != nil {
		// Trigger might already exist, log but continue
		log.Debug().Err(err).Msg("Products trigger may already exist")
	}

	// Check if PostGIS extension is installed (created by enablePostGISExtension)
	var postgisInstalled bool
	err = db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'postgis')").Scan(&postgisInstalled)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to check PostGIS availability")
		postgisInstalled = false
	}

	// Create locations table for PostGIS tests (only if PostGIS is available)
	if postgisInstalled {
		_, err = db.Exec(ctx, `
			CREATE TABLE IF NOT EXISTS public.locations (
				id INTEGER PRIMARY KEY GENERATED BY DEFAULT AS IDENTITY,
				name TEXT NOT NULL,
				location GEOMETRY(Point, 4326),
				created_at TIMESTAMPTZ DEFAULT NOW(),
				updated_at TIMESTAMPTZ DEFAULT NOW()
			)
		`)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create locations table")
		} else {
			// Create trigger for locations updated_at
			_, err = db.Exec(ctx, `
				CREATE TRIGGER update_locations_updated_at
				BEFORE UPDATE ON public.locations
				FOR EACH ROW
				EXECUTE FUNCTION public.update_updated_at()
			`)
			if err != nil {
				log.Debug().Err(err).Msg("Locations trigger may already exist")
			}
			log.Info().Msg("Created locations table for PostGIS tests")
		}

		// Create regions table for PostGIS tests (only if PostGIS is available)
		_, err = db.Exec(ctx, `
			CREATE TABLE IF NOT EXISTS public.regions (
				id INTEGER PRIMARY KEY GENERATED BY DEFAULT AS IDENTITY,
				name TEXT NOT NULL,
				boundary GEOMETRY(Polygon, 4326),
				created_at TIMESTAMPTZ DEFAULT NOW(),
				updated_at TIMESTAMPTZ DEFAULT NOW()
			)
		`)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create regions table")
		} else {
			// Create trigger for regions updated_at
			_, err = db.Exec(ctx, `
				CREATE TRIGGER update_regions_updated_at
				BEFORE UPDATE ON public.regions
				FOR EACH ROW
				EXECUTE FUNCTION public.update_updated_at()
			`)
			if err != nil {
				log.Debug().Err(err).Msg("Regions trigger may already exist")
			}
			log.Info().Msg("Created regions table for PostGIS tests")
		}
	} else {
		log.Info().Msg("PostGIS not available, skipping PostGIS test table creation")
	}

	// Ensure uuid-ossp extension is available for uuid_generate_v4()
	_, err = db.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create uuid-ossp extension")
		return false
	}

	// Create tasks table for RLS tests
	_, err = db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS public.tasks (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			description TEXT,
			completed BOOLEAN DEFAULT FALSE,
			is_public BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create tasks table")
		return false
	}

	// Create trigger for tasks updated_at
	_, err = db.Exec(ctx, `
		CREATE TRIGGER update_tasks_updated_at
		BEFORE UPDATE ON public.tasks
		FOR EACH ROW
		EXECUTE FUNCTION public.update_updated_at()
	`)
	if err != nil {
		log.Debug().Err(err).Msg("Tasks trigger may already exist")
	}

	// Enable RLS on tasks table
	_, err = db.Exec(ctx, `ALTER TABLE public.tasks ENABLE ROW LEVEL SECURITY`)
	if err != nil {
		log.Debug().Err(err).Msg("RLS may already be enabled on tasks")
	}

	// Force RLS even for table owner (required for testing)
	_, err = db.Exec(ctx, `ALTER TABLE public.tasks FORCE ROW LEVEL SECURITY`)
	if err != nil {
		log.Debug().Err(err).Msg("FORCE RLS may already be enabled on tasks")
	}

	// Drop existing RLS policies if they exist (to avoid conflicts)
	_, _ = db.Exec(ctx, `DROP POLICY IF EXISTS tasks_select_own ON public.tasks`)
	_, _ = db.Exec(ctx, `DROP POLICY IF EXISTS tasks_insert_own ON public.tasks`)
	_, _ = db.Exec(ctx, `DROP POLICY IF EXISTS tasks_update_own ON public.tasks`)
	_, _ = db.Exec(ctx, `DROP POLICY IF EXISTS tasks_delete_own ON public.tasks`)

	// Create RLS policies for tasks
	_, err = db.Exec(ctx, `
		CREATE POLICY tasks_select_own ON public.tasks
		FOR SELECT
		USING (user_id = auth.current_user_id() OR is_public = true)
	`)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create tasks_select_own policy")
	}

	_, err = db.Exec(ctx, `
		CREATE POLICY tasks_insert_own ON public.tasks
		FOR INSERT
		WITH CHECK (user_id = auth.current_user_id())
	`)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create tasks_insert_own policy")
	}

	_, err = db.Exec(ctx, `
		CREATE POLICY tasks_update_own ON public.tasks
		FOR UPDATE
		USING (user_id = auth.current_user_id())
	`)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create tasks_update_own policy")
	}

	_, err = db.Exec(ctx, `
		CREATE POLICY tasks_delete_own ON public.tasks
		FOR DELETE
		USING (user_id = auth.current_user_id())
	`)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create tasks_delete_own policy")
	}

	// Grant permissions to fluxbase_rls_test user for testing RLS
	// This must be done as postgres superuser since fluxbase_app doesn't own the schemas
	grantRLSTestPermissions()

	// Enable signup for all tests (settings are checked from database first, then config)
	// Note: is_public must be true so anon role can read this setting during signup
	_, err = db.Exec(ctx, `DELETE FROM app.settings WHERE key = 'app.auth.signup_enabled'`)
	if err != nil {
		log.Error().Err(err).Msg("Failed to delete existing signup setting")
	}
	_, err = db.Exec(ctx, `
		INSERT INTO app.settings (key, value, category, is_public)
		VALUES ('app.auth.signup_enabled', '{"value": true}'::jsonb, 'system', true)
	`)
	if err != nil {
		log.Error().Err(err).Msg("Failed to enable signup in database settings")
	}

	log.Info().Msg("E2E test tables setup complete")
	return true
}

// refreshSharedTestContextSchemaCache refreshes the schema cache for the shared test context.
// This must be called after creating test tables so the server knows about them.
func refreshSharedTestContextSchemaCache() {
	ctx := context.Background()

	// Get the shared test context
	tc := test.GetSharedTestContextUnsafe()
	if tc == nil || tc.Server == nil {
		log.Warn().Msg("Shared test context not available for schema refresh")
		return
	}

	// Get the schema cache from the REST handler
	schemaCache := tc.Server.SchemaCache()
	if schemaCache == nil {
		log.Warn().Msg("Schema cache not initialized")
		return
	}

	// Force refresh the schema cache
	if err := schemaCache.Refresh(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to refresh schema cache after test table creation")
		return
	}

	log.Info().
		Int("tables", schemaCache.TableCount()).
		Int("views", schemaCache.ViewCount()).
		Msg("Schema cache refreshed successfully after test table creation")
}

// grantRLSTestPermissions grants necessary permissions to the fluxbase_rls_test and fluxbase_app database users.
//
// This function connects as the postgres superuser to grant permissions because
// fluxbase_app does not own the schemas and cannot grant permissions on them.
//
// Permissions Granted:
//   - Schema USAGE and CREATE on: auth, platform, functions, storage, realtime
//   - ALL privileges on tables and sequences in those schemas
//   - EXECUTE on all functions in functions schema
//
// The fluxbase_rls_test user needs these permissions to:
//   - Create test users in auth.users
//   - Query and insert test data
//   - Test RLS policies without BYPASSRLS privilege
//
// The fluxbase_app user needs these permissions to:
//   - Run tests with BYPASSRLS
//   - Access all schemas for testing and migration tracking
func grantRLSTestPermissions() {
	ctx := context.Background()

	cfg := test.GetTestConfig()
	cfg.Database.User = getEnvOrDefault("FLUXBASE_DATABASE_ADMIN_USER", "postgres")
	cfg.Database.AdminUser = cfg.Database.User
	cfg.Database.Password = getEnvOrDefault("FLUXBASE_DATABASE_ADMIN_PASSWORD", "postgres")

	db, err := getDatabase(cfg)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect as postgres for granting permissions")
		return
	}
	defer db.Close()

	const users = "fluxbase_rls_test, fluxbase_app"
	dbName := cfg.Database.Database

	schemas := []string{
		"public", "app", "auth", "platform", "functions", "jobs",
		"storage", "realtime", "mcp", "ai", "rpc", "logging", "branching",
	}

	funcSchemas := []string{
		"public", "auth", "functions", "storage", "ai", "rpc", "mcp",
	}

	// Database + schema permissions
	var sql strings.Builder
	fmt.Fprintf(&sql, "GRANT CREATE ON DATABASE %s TO %s;\n", dbName, users)
	for _, s := range schemas {
		fmt.Fprintf(&sql, "GRANT USAGE, CREATE ON SCHEMA %s TO %s;\n", s, users)
	}
	_, err = db.Exec(ctx, sql.String())
	if err != nil {
		log.Error().Err(err).Msg("Failed to grant schema permissions to test users")
		return
	}

	// Table, sequence, and default privileges (current user + postgres role)
	var privSQL strings.Builder
	for _, s := range schemas {
		fmt.Fprintf(&privSQL, "GRANT ALL ON ALL TABLES IN SCHEMA %s TO %s;\n", s, users)
		fmt.Fprintf(&privSQL, "GRANT ALL ON ALL SEQUENCES IN SCHEMA %s TO %s;\n", s, users)
		fmt.Fprintf(&privSQL, "ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT ALL ON TABLES TO %s;\n", s, users)
		fmt.Fprintf(&privSQL, "ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT ALL ON SEQUENCES TO %s;\n", s, users)
		fmt.Fprintf(&privSQL, "ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA %s GRANT ALL ON TABLES TO %s;\n", s, users)
		fmt.Fprintf(&privSQL, "ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA %s GRANT ALL ON SEQUENCES TO %s;\n", s, users)
	}
	_, err = db.Exec(ctx, privSQL.String())
	if err != nil {
		log.Error().Err(err).Msg("Failed to grant table/sequence permissions to test users")
		return
	}

	// Function execution permissions
	var funcSQL strings.Builder
	for _, s := range funcSchemas {
		fmt.Fprintf(&funcSQL, "GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA %s TO %s;\n", s, users)
	}
	_, err = db.Exec(ctx, funcSQL.String())
	if err != nil {
		log.Error().Err(err).Msg("Failed to grant function execution permissions to test users")
	}

	// Role grants for SET ROLE support (required by RLS middleware)
	_, err = db.Exec(ctx, `
		GRANT anon, authenticated, service_role, tenant_service, tenant_migration_role
		TO fluxbase_app, fluxbase_rls_test
	`)
	if err != nil {
		log.Error().Err(err).Msg("Failed to grant SET ROLE permissions to test users")
	}

	// Grant permissions to anon, authenticated, and service_role for test tables in public schema
	// This is needed because our security changes restricted anon's broad schema access
	_, err = db.Exec(ctx, `
		-- Grant SELECT, INSERT, UPDATE, DELETE on public.products to all roles
		GRANT SELECT, INSERT, UPDATE, DELETE ON public.products TO anon, authenticated, service_role;
		GRANT USAGE ON SEQUENCE products_id_seq TO anon, authenticated, service_role;

		-- Grant SELECT, INSERT, UPDATE, DELETE on public.tasks to all roles
		GRANT SELECT, INSERT, UPDATE, DELETE ON public.tasks TO anon, authenticated, service_role;
	`)
	if err != nil {
		log.Error().Err(err).Msg("Failed to grant permissions to anon/authenticated/service_role roles on test tables")
	}

	// Grant permissions for PostGIS test tables (if they exist)
	_, err = db.Exec(ctx, `
		-- Grant SELECT, INSERT, UPDATE, DELETE on public.locations to all roles
		GRANT SELECT, INSERT, UPDATE, DELETE ON public.locations TO anon, authenticated, service_role;
		GRANT USAGE ON SEQUENCE locations_id_seq TO anon, authenticated, service_role;

		-- Grant SELECT, INSERT, UPDATE, DELETE on public.regions to all roles
		GRANT SELECT, INSERT, UPDATE, DELETE ON public.regions TO anon, authenticated, service_role;
		GRANT USAGE ON SEQUENCE regions_id_seq TO anon, authenticated, service_role;
	`)
	if err != nil {
		// Tables might not exist if PostGIS is not available, log but continue
		log.Debug().Err(err).Msg("Failed to grant permissions on PostGIS test tables (tables may not exist)")
	}
}

// teardownTestTables drops the test tables after all tests complete
func teardownTestTables() {
	ctx := context.Background()

	// Get database connection
	// Connect as postgres superuser to drop tables that may be owned by postgres
	// (e.g., tasks table created by EnsureRLSTestTables())
	cfg := test.GetTestConfig()
	// Use postgres superuser from environment variables (with defaults for local dev)
	cfg.Database.User = getEnvOrDefault("FLUXBASE_DATABASE_ADMIN_USER", "postgres")
	cfg.Database.AdminUser = cfg.Database.User
	cfg.Database.Password = getEnvOrDefault("FLUXBASE_DATABASE_ADMIN_PASSWORD", "postgres")
	db, err := getDatabase(cfg)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to database for test teardown")
		return
	}
	defer db.Close()

	log.Info().Msg("Tearing down e2e test tables...")

	// Define test table patterns to clean up
	// 1. Hardcoded e2e test tables
	hardcodedTables := []string{
		"public.products",
		"public.tasks",
		"public.locations",
		"public.regions",
		"public.role_check",
		"public.sensitive_data",
	}

	// 2. Integration test table patterns (test_table_XXXXXXXX, etc.)
	// Note: These patterns are embedded in the SQL query below

	// Drop hardcoded tables
	for _, table := range hardcodedTables {
		_, err = db.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
		if err != nil {
			log.Error().Err(err).Str("table", table).Msg("Failed to drop table")
		} else {
			log.Debug().Str("table", table).Msg("Dropped table")
		}
	}

	// Drop tables matching test patterns using dynamic SQL
	// Query pg_tables to find all tables matching the patterns
	rows, err := db.Query(ctx, `
		SELECT tablename
		FROM pg_tables
		WHERE schemaname = 'public'
		AND (
			tablename LIKE 'test_table_%'
			OR tablename LIKE 'test_single_%'
			OR tablename LIKE 'test_already_%'
			OR tablename LIKE 'test_rollback_%'
			OR tablename LIKE 'test_nodown_%'
			OR tablename LIKE 'test_stop_%'
			OR tablename LIKE 'test_history_%'
			OR tablename LIKE 'test_retry_%'
			OR tablename LIKE 'test_delete_%'
			OR tablename LIKE 'ns_test_%'
		)
	`)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query test tables")
	} else {
		var tableNames []string
		for rows.Next() {
			var tableName string
			if err := rows.Scan(&tableName); err != nil {
				log.Error().Err(err).Msg("Failed to scan table name")
				continue
			}
			tableNames = append(tableNames, tableName)
		}
		rows.Close()

		// Drop each test table
		for _, tableName := range tableNames {
			// Quote the table name to handle any special characters
			quotedTableName := fmt.Sprintf("\"%s\"", tableName)
			_, err = db.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS public.%s CASCADE", quotedTableName))
			if err != nil {
				log.Error().Err(err).Str("table", tableName).Msg("Failed to drop test table")
			} else {
				log.Debug().Str("table", tableName).Msg("Dropped test table")
			}
		}

		if len(tableNames) > 0 {
			log.Info().Int("count", len(tableNames)).Msg("Dropped dynamic test tables")
		}
	}

	// Clean up test data from other tables
	// Delete test secrets (name starts with 'test')
	result, err := db.Exec(ctx, "DELETE FROM functions.secrets WHERE name LIKE 'test_%'")
	if err != nil {
		log.Error().Err(err).Msg("Failed to delete test secrets")
	} else if result.RowsAffected() > 0 {
		log.Info().Int64("count", result.RowsAffected()).Msg("Deleted test secrets")
	}

	// Delete test client keys (name starts with 'test')
	// Note: api_keys was renamed to client_keys in migration 047
	result, err = db.Exec(ctx, "DELETE FROM auth.client_keys WHERE name LIKE 'test_%'")
	if err != nil {
		log.Error().Err(err).Msg("Failed to delete test client keys")
	} else if result.RowsAffected() > 0 {
		log.Info().Int64("count", result.RowsAffected()).Msg("Deleted test client keys")
	}

	// Delete test storage buckets (id/name starts with 'test')
	// Note: Cascade will delete associated objects and permissions
	result, err = db.Exec(ctx, "DELETE FROM storage.buckets WHERE id LIKE 'test_%' OR name LIKE 'test_%' OR name IN ('bucket1', 'bucket2', 's3-bucket1', 's3-bucket2') OR name LIKE '%test-bucket%'")
	if err != nil {
		log.Error().Err(err).Msg("Failed to delete test storage buckets")
	} else if result.RowsAffected() > 0 {
		log.Info().Int64("count", result.RowsAffected()).Msg("Deleted test storage buckets")
	}

	// Delete test service keys (children of tenants, delete before tenants)
	result, err = db.Exec(ctx, "DELETE FROM auth.service_keys WHERE name LIKE 'test_%' OR name LIKE '%Test%' OR name LIKE '%Storage Admin%' OR name LIKE '%S3 Storage%'")
	if err != nil {
		log.Error().Err(err).Msg("Failed to delete test service keys")
	} else if result.RowsAffected() > 0 {
		log.Info().Int64("count", result.RowsAffected()).Msg("Deleted test service keys")
	}

	// Delete test webhooks
	result, err = db.Exec(ctx, "DELETE FROM auth.webhooks WHERE name LIKE 'test_%' OR name LIKE '%Test%'")
	if err != nil {
		log.Error().Err(err).Msg("Failed to delete test webhooks")
	} else if result.RowsAffected() > 0 {
		log.Info().Int64("count", result.RowsAffected()).Msg("Deleted test webhooks")
	}

	// Delete test edge functions
	result, err = db.Exec(ctx, "DELETE FROM functions.edge_functions WHERE name LIKE 'test_%'")
	if err != nil {
		log.Error().Err(err).Msg("Failed to delete test edge functions")
	} else if result.RowsAffected() > 0 {
		log.Info().Int64("count", result.RowsAffected()).Msg("Deleted test edge functions")
	}

	// Delete test background jobs
	result, err = db.Exec(ctx, "DELETE FROM jobs.queue WHERE job_name LIKE 'test_%'")
	if err != nil {
		log.Error().Err(err).Msg("Failed to delete test background jobs")
	} else if result.RowsAffected() > 0 {
		log.Info().Int64("count", result.RowsAffected()).Msg("Deleted test background jobs")
	}

	// Delete test tenants (must come after service_keys and other children)
	result, err = db.Exec(ctx, "DELETE FROM platform.tenants WHERE slug LIKE 'test-%' OR slug LIKE 'tenant-test-%'")
	if err != nil {
		log.Error().Err(err).Msg("Failed to delete test tenants")
	} else if result.RowsAffected() > 0 {
		log.Info().Int64("count", result.RowsAffected()).Msg("Deleted test tenants")
	}

	log.Info().Msg("E2E test tables teardown complete")
}
