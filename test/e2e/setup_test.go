// Package e2e contains end-to-end tests for Fluxbase.
//
// # Test Setup
//
// TestMain creates two test tables before running all tests:
//
//  1. products - Simple table for REST API testing (no RLS)
//  2. tasks - Complex table for RLS policy testing (RLS enabled)
//
// # Database Users
//
// Three database users are used for different purposes:
//
//  1. postgres (superuser) - Used only for granting permissions
//  2. fluxbase_app (has BYPASSRLS) - Used by NewTestContext for general testing
//  3. fluxbase_rls_test (no BYPASSRLS) - Used by NewRLSTestContext for RLS testing
//
// # Test Execution Flow
//
//  1. TestMain runs setupTestTables() - Creates products and tasks tables
//  2. Individual tests run (each should truncate tables for isolation)
//  3. TestMain runs teardownTestTables() - Drops all test tables
//
// See test/README.md for detailed testing guide.
package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/fluxbase-eu/fluxbase/test"
	"github.com/rs/zerolog/log"
)

// getDatabase creates a database connection for setup/teardown operations.
func getDatabase(cfg *config.Config) (*database.Connection, error) {
	return database.NewConnection(cfg.Database)
}

// TestMain runs before all e2e tests to set up test tables and after all tests to clean up.
//
// Execution Flow:
//  1. CleanupE2ETestUsersGlobal() - Clean up any leftover test users from previous runs
//  2. setupTestTables() - Creates products and tasks tables with RLS policies
//  3. m.Run() - Runs all test functions in the e2e package
//  4. teardownTestTables() - Drops all test tables
//
// Note: Individual tests should truncate tables for test isolation.
func TestMain(m *testing.M) {
	// Cleanup: Remove any leftover e2e test users from previous runs
	// This ensures a clean state even if previous test runs failed
	cfg := test.GetTestConfig()
	test.CleanupE2ETestUsersGlobal(cfg)

	// Setup: Create test tables before running tests
	if !setupTestTables() {
		log.Fatal().Msg("Failed to setup test tables - cannot run tests")
		os.Exit(1)
	}

	// Run all tests
	code := m.Run()

	// Cleanup: Remove test users created during this test run
	test.CleanupE2ETestUsersGlobal(cfg)

	// Teardown: Clean up test tables after all tests complete
	teardownTestTables()

	// Exit with the test result code
	os.Exit(code)
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
	cfg.Database.User = "postgres"
	cfg.Database.AdminUser = "postgres"
	cfg.Database.Password = "postgres"

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

	// Get database connection using fluxbase_app (has BYPASSRLS for setup)
	cfg := test.GetTestConfig()
	db, err := getDatabase(cfg)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to database for test setup")
		return false
	}
	defer db.Close()

	log.Info().Msg("Setting up e2e test tables...")

	// Create products table for REST tests
	_, err = db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS public.products (
			id INTEGER PRIMARY KEY GENERATED BY DEFAULT AS IDENTITY,
			name TEXT NOT NULL,
			price NUMERIC(10, 2) NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create products table")
		return false
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
	db.Exec(ctx, `DROP POLICY IF EXISTS tasks_select_own ON public.tasks`)
	db.Exec(ctx, `DROP POLICY IF EXISTS tasks_insert_own ON public.tasks`)
	db.Exec(ctx, `DROP POLICY IF EXISTS tasks_update_own ON public.tasks`)
	db.Exec(ctx, `DROP POLICY IF EXISTS tasks_delete_own ON public.tasks`)

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

// grantRLSTestPermissions grants necessary permissions to the fluxbase_rls_test and fluxbase_app database users.
//
// This function connects as the postgres superuser to grant permissions because
// fluxbase_app does not own the schemas and cannot grant permissions on them.
//
// Permissions Granted:
//   - Schema USAGE and CREATE on: auth, dashboard, functions, storage, realtime
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

	// Connect as postgres superuser to grant permissions
	// fluxbase_app cannot grant permissions on schemas it doesn't own
	cfg := test.GetTestConfig()
	cfg.Database.User = "postgres"
	cfg.Database.AdminUser = "postgres"
	cfg.Database.Password = "postgres"

	db, err := getDatabase(cfg)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect as postgres for granting permissions")
		return
	}
	defer db.Close()

	// Grant database and schema permissions to both test users
	// Note: Database name must match the actual database being used
	dbName := cfg.Database.Database
	_, err = db.Exec(ctx, fmt.Sprintf(`
		GRANT CREATE ON DATABASE %s TO fluxbase_rls_test, fluxbase_app;
		GRANT USAGE, CREATE ON SCHEMA app TO fluxbase_rls_test, fluxbase_app;
		GRANT USAGE, CREATE ON SCHEMA auth TO fluxbase_rls_test, fluxbase_app;
		GRANT USAGE, CREATE ON SCHEMA dashboard TO fluxbase_rls_test, fluxbase_app;
		GRANT USAGE, CREATE ON SCHEMA functions TO fluxbase_rls_test, fluxbase_app;
		GRANT USAGE, CREATE ON SCHEMA jobs TO fluxbase_rls_test, fluxbase_app;
		GRANT USAGE, CREATE ON SCHEMA storage TO fluxbase_rls_test, fluxbase_app;
		GRANT USAGE, CREATE ON SCHEMA realtime TO fluxbase_rls_test, fluxbase_app;
	`, dbName))
	if err != nil {
		log.Error().Err(err).Msg("Failed to grant schema permissions to test users")
		return
	}

	// Grant table and sequence permissions to both test users
	_, err = db.Exec(ctx, `
		GRANT ALL ON ALL TABLES IN SCHEMA public TO fluxbase_rls_test, fluxbase_app;
		GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO fluxbase_rls_test, fluxbase_app;
		GRANT ALL ON ALL TABLES IN SCHEMA app TO fluxbase_rls_test, fluxbase_app;
		GRANT ALL ON ALL SEQUENCES IN SCHEMA app TO fluxbase_rls_test, fluxbase_app;
		GRANT ALL ON ALL TABLES IN SCHEMA auth TO fluxbase_rls_test, fluxbase_app;
		GRANT ALL ON ALL SEQUENCES IN SCHEMA auth TO fluxbase_rls_test, fluxbase_app;
		GRANT ALL ON ALL TABLES IN SCHEMA dashboard TO fluxbase_rls_test, fluxbase_app;
		GRANT ALL ON ALL SEQUENCES IN SCHEMA dashboard TO fluxbase_rls_test, fluxbase_app;
		GRANT ALL ON ALL TABLES IN SCHEMA functions TO fluxbase_rls_test, fluxbase_app;
		GRANT ALL ON ALL SEQUENCES IN SCHEMA functions TO fluxbase_rls_test, fluxbase_app;
		GRANT ALL ON ALL TABLES IN SCHEMA jobs TO fluxbase_rls_test, fluxbase_app;
		GRANT ALL ON ALL SEQUENCES IN SCHEMA jobs TO fluxbase_rls_test, fluxbase_app;
		GRANT ALL ON ALL TABLES IN SCHEMA storage TO fluxbase_rls_test, fluxbase_app;
		GRANT ALL ON ALL SEQUENCES IN SCHEMA storage TO fluxbase_rls_test, fluxbase_app;
		GRANT ALL ON ALL TABLES IN SCHEMA realtime TO fluxbase_rls_test, fluxbase_app;
		GRANT ALL ON ALL SEQUENCES IN SCHEMA realtime TO fluxbase_rls_test, fluxbase_app;

		-- Grant permissions on future tables/sequences (in case migrations add new ones)
		ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO fluxbase_rls_test, fluxbase_app;
		ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO fluxbase_rls_test, fluxbase_app;
		ALTER DEFAULT PRIVILEGES IN SCHEMA app GRANT ALL ON TABLES TO fluxbase_rls_test, fluxbase_app;
		ALTER DEFAULT PRIVILEGES IN SCHEMA app GRANT ALL ON SEQUENCES TO fluxbase_rls_test, fluxbase_app;
		ALTER DEFAULT PRIVILEGES IN SCHEMA auth GRANT ALL ON TABLES TO fluxbase_rls_test, fluxbase_app;
		ALTER DEFAULT PRIVILEGES IN SCHEMA auth GRANT ALL ON SEQUENCES TO fluxbase_rls_test, fluxbase_app;
		ALTER DEFAULT PRIVILEGES IN SCHEMA dashboard GRANT ALL ON TABLES TO fluxbase_rls_test, fluxbase_app;
		ALTER DEFAULT PRIVILEGES IN SCHEMA dashboard GRANT ALL ON SEQUENCES TO fluxbase_rls_test, fluxbase_app;
		ALTER DEFAULT PRIVILEGES IN SCHEMA functions GRANT ALL ON TABLES TO fluxbase_rls_test, fluxbase_app;
		ALTER DEFAULT PRIVILEGES IN SCHEMA functions GRANT ALL ON SEQUENCES TO fluxbase_rls_test, fluxbase_app;
		ALTER DEFAULT PRIVILEGES IN SCHEMA jobs GRANT ALL ON TABLES TO fluxbase_rls_test, fluxbase_app;
		ALTER DEFAULT PRIVILEGES IN SCHEMA jobs GRANT ALL ON SEQUENCES TO fluxbase_rls_test, fluxbase_app;
		ALTER DEFAULT PRIVILEGES IN SCHEMA storage GRANT ALL ON TABLES TO fluxbase_rls_test, fluxbase_app;
		ALTER DEFAULT PRIVILEGES IN SCHEMA storage GRANT ALL ON SEQUENCES TO fluxbase_rls_test, fluxbase_app;
		ALTER DEFAULT PRIVILEGES IN SCHEMA realtime GRANT ALL ON TABLES TO fluxbase_rls_test, fluxbase_app;
		ALTER DEFAULT PRIVILEGES IN SCHEMA realtime GRANT ALL ON SEQUENCES TO fluxbase_rls_test, fluxbase_app;
	`)
	if err != nil {
		log.Error().Err(err).Msg("Failed to grant table/sequence permissions to test users")
		return
	}

	// Grant function execution permissions to both test users
	_, err = db.Exec(ctx, `
		GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA functions TO fluxbase_rls_test, fluxbase_app;
		GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA auth TO fluxbase_rls_test, fluxbase_app;
		GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA storage TO fluxbase_rls_test, fluxbase_app;
	`)
	if err != nil {
		log.Error().Err(err).Msg("Failed to grant function execution permissions to test users")
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
	cfg := test.GetTestConfig()
	db, err := getDatabase(cfg)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to database for test teardown")
		return
	}
	defer db.Close()

	log.Info().Msg("Tearing down e2e test tables...")

	// Drop test tables
	_, err = db.Exec(ctx, `DROP TABLE IF EXISTS public.products CASCADE`)
	if err != nil {
		log.Error().Err(err).Msg("Failed to drop products table")
	}

	_, err = db.Exec(ctx, `DROP TABLE IF EXISTS public.tasks CASCADE`)
	if err != nil {
		log.Error().Err(err).Msg("Failed to drop tasks table")
	}

	// Drop PostGIS test tables (if they exist)
	_, err = db.Exec(ctx, `DROP TABLE IF EXISTS public.locations CASCADE`)
	if err != nil {
		log.Error().Err(err).Msg("Failed to drop locations table")
	}

	_, err = db.Exec(ctx, `DROP TABLE IF EXISTS public.regions CASCADE`)
	if err != nil {
		log.Error().Err(err).Msg("Failed to drop regions table")
	}

	log.Info().Msg("E2E test tables teardown complete")
}
