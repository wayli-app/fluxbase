-- Fluxbase Bootstrap SQL
-- This file creates the minimal infrastructure needed before declarative schema management.
-- It handles operations that pgschema cannot manage:
-- - CREATE EXTENSION (database-level)
-- - CREATE SCHEMA (database-level)
-- - CREATE ROLE (cluster-level)
-- - ALTER DEFAULT PRIVILEGES (database-level)
--
-- This file is idempotent and safe to run multiple times.

-- ============================================================================
-- EXTENSIONS
-- ============================================================================

-- UUID generation functions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";    -- Provides uuid_generate_v4()
CREATE EXTENSION IF NOT EXISTS "pgcrypto";     -- Provides gen_random_uuid() and crypto functions

-- Text search and indexing
CREATE EXTENSION IF NOT EXISTS "pg_trgm";      -- Trigram text search
CREATE EXTENSION IF NOT EXISTS "btree_gin";    -- GIN indexes for btree-indexable data types

-- Vector similarity search (pgvector)
CREATE EXTENSION IF NOT EXISTS "vector";       -- Vector embeddings for AI/ML

-- Foreign data wrapper for cross-database joins (tenant → main DB)
CREATE EXTENSION IF NOT EXISTS "postgres_fdw";

-- ============================================================================
-- SCHEMAS
-- ============================================================================

-- Auth schema: Application user authentication, client keys, and sessions
CREATE SCHEMA IF NOT EXISTS auth;
GRANT USAGE, CREATE ON SCHEMA auth TO CURRENT_USER;
COMMENT ON SCHEMA auth IS 'Application user authentication, client keys, and sessions';

-- App schema: Application-level configuration and settings
CREATE SCHEMA IF NOT EXISTS app;
GRANT USAGE, CREATE ON SCHEMA app TO CURRENT_USER;
COMMENT ON SCHEMA app IS 'Application-level configuration, settings, and metadata';

-- Functions schema: Edge functions and their executions
CREATE SCHEMA IF NOT EXISTS functions;
GRANT USAGE, CREATE ON SCHEMA functions TO CURRENT_USER;
COMMENT ON SCHEMA functions IS 'Edge functions and their executions';

-- Storage schema: File storage buckets and objects
CREATE SCHEMA IF NOT EXISTS storage;
GRANT USAGE, CREATE ON SCHEMA storage TO CURRENT_USER;
COMMENT ON SCHEMA storage IS 'File storage buckets and objects';

-- Realtime schema: Realtime subscriptions and change tracking
CREATE SCHEMA IF NOT EXISTS realtime;
GRANT USAGE, CREATE ON SCHEMA realtime TO CURRENT_USER;
COMMENT ON SCHEMA realtime IS 'Realtime subscriptions and change tracking';

-- Jobs schema: Long-running background jobs
CREATE SCHEMA IF NOT EXISTS jobs;
GRANT USAGE, CREATE ON SCHEMA jobs TO CURRENT_USER;
COMMENT ON SCHEMA jobs IS 'Long-running background jobs system';

-- AI schema: AI chatbots, conversations, and query auditing
CREATE SCHEMA IF NOT EXISTS ai;
GRANT USAGE, CREATE ON SCHEMA ai TO CURRENT_USER;
COMMENT ON SCHEMA ai IS 'AI chatbots, conversations, and query auditing';

-- RPC schema: Stored procedure definitions and executions
CREATE SCHEMA IF NOT EXISTS rpc;
GRANT USAGE, CREATE ON SCHEMA rpc TO CURRENT_USER;
COMMENT ON SCHEMA rpc IS 'Stored procedure definitions and executions';

-- Logging schema: Centralized logging entries
CREATE SCHEMA IF NOT EXISTS logging;
GRANT USAGE, CREATE ON SCHEMA logging TO CURRENT_USER;
COMMENT ON SCHEMA logging IS 'Centralized logging entries';

-- Branching schema: Database branching for dev/test environments
CREATE SCHEMA IF NOT EXISTS branching;
GRANT USAGE, CREATE ON SCHEMA branching TO CURRENT_USER;
COMMENT ON SCHEMA branching IS 'Database branching for dev/test environments';

-- MCP schema: Model Context Protocol server
CREATE SCHEMA IF NOT EXISTS mcp;
GRANT USAGE, CREATE ON SCHEMA mcp TO CURRENT_USER;
COMMENT ON SCHEMA mcp IS 'Model Context Protocol server for AI assistant integration';

-- Platform schema: Multi-tenancy control plane
CREATE SCHEMA IF NOT EXISTS platform;
GRANT USAGE, CREATE ON SCHEMA platform TO CURRENT_USER;
COMMENT ON SCHEMA platform IS 'Multi-tenancy control plane (tenants, service keys, admin assignments)';

-- ============================================================================
-- ROLES
-- ============================================================================

-- Anonymous role: For unauthenticated requests with public access only
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'anon') THEN
        CREATE ROLE anon NOLOGIN NOINHERIT;
    END IF;
END
$$;
COMMENT ON ROLE anon IS 'Anonymous role for unauthenticated requests with public access only';

-- Authenticated role: For users with valid JWT tokens
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'authenticated') THEN
        CREATE ROLE authenticated NOLOGIN NOINHERIT;
    END IF;
END
$$;
COMMENT ON ROLE authenticated IS 'Authenticated role for users with valid JWT tokens';

-- Service role: For backend services with elevated permissions and RLS bypass
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'service_role') THEN
        CREATE ROLE service_role NOLOGIN NOINHERIT BYPASSRLS;
    END IF;
END
$$;
COMMENT ON ROLE service_role IS 'Service role for backend services with elevated permissions and RLS bypass';

-- Tenant service role: Tenant-scoped service role for multi-tenant isolation
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'tenant_service') THEN
        CREATE ROLE tenant_service NOLOGIN NOINHERIT NOBYPASSRLS;
    END IF;
END
$$;
COMMENT ON ROLE tenant_service IS 'Tenant-scoped service role for multi-tenant isolation - respects RLS with tenant context';

-- Tenant migration role: For tenant-specific schema migrations
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'tenant_migration_role') THEN
        CREATE ROLE tenant_migration_role NOLOGIN NOINHERIT NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
    END IF;
END
$$;
COMMENT ON ROLE tenant_migration_role IS 'Role for tenant-specific schema migrations with limited permissions';

-- ============================================================================
-- ROLE GRANTS TO CURRENT USER
-- These allow the RLS middleware to execute SET LOCAL ROLE
-- ============================================================================

GRANT anon TO CURRENT_USER;
GRANT authenticated TO CURRENT_USER;
GRANT service_role TO CURRENT_USER;
GRANT tenant_service TO CURRENT_USER;
GRANT tenant_migration_role TO CURRENT_USER;

-- ============================================================================
-- SCHEMA USAGE GRANTS FOR RLS ROLES
-- ============================================================================

-- Core schemas accessible to all RLS roles
GRANT USAGE ON SCHEMA auth TO anon, authenticated, service_role;
GRANT USAGE ON SCHEMA app TO anon, authenticated, service_role;
GRANT USAGE ON SCHEMA storage TO anon, authenticated, service_role;
GRANT USAGE ON SCHEMA functions TO anon, authenticated, service_role;
GRANT USAGE ON SCHEMA realtime TO anon, authenticated, service_role;
GRANT USAGE ON SCHEMA public TO anon, authenticated, service_role;

-- AI schema accessible to all
GRANT USAGE ON SCHEMA ai TO anon, authenticated, service_role;

-- Jobs schema - authenticated and service only
GRANT USAGE ON SCHEMA jobs TO authenticated, service_role;

-- RPC schema - authenticated and service only
GRANT USAGE ON SCHEMA rpc TO authenticated, service_role;

-- MCP schema - all roles
GRANT USAGE ON SCHEMA mcp TO anon, authenticated, service_role;

-- Branching schema - service role and authenticated (SQL editor tenant admin access, RLS enforced)
GRANT USAGE ON SCHEMA branching TO authenticated, service_role;

-- Logging schema - service role, {{APP_USER}}, and authenticated (SQL editor tenant admin access, RLS enforced)
GRANT USAGE ON SCHEMA logging TO authenticated, service_role, {{APP_USER}};

-- Platform schema - authenticated and service
GRANT USAGE ON SCHEMA platform TO authenticated, service_role;

-- Tenant migration role - public schema only
GRANT USAGE, CREATE ON SCHEMA public TO tenant_migration_role;

-- ============================================================================
-- ALTER DEFAULT PRIVILEGES
-- These ensure future tables automatically get correct permissions.
--
-- Note: Default privileges are only set for `authenticated` and `service_role`.
-- Roles `{{APP_USER}}` and `fluxbase_rls_test` receive per-table GRANTs in the
-- declarative schema SQL files (internal/database/schema/schemas/*.sql) rather
-- than default privileges, so pgschema can track them correctly.
-- ============================================================================

-- Auth schema
ALTER DEFAULT PRIVILEGES IN SCHEMA auth
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO authenticated;
ALTER DEFAULT PRIVILEGES IN SCHEMA auth
    GRANT ALL ON TABLES TO service_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA auth
    GRANT ALL ON SEQUENCES TO service_role;

-- App schema
ALTER DEFAULT PRIVILEGES IN SCHEMA app
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO authenticated;
ALTER DEFAULT PRIVILEGES IN SCHEMA app
    GRANT ALL ON TABLES TO service_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA app
    GRANT ALL ON SEQUENCES TO service_role;

-- Storage schema
ALTER DEFAULT PRIVILEGES IN SCHEMA storage
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO authenticated;
ALTER DEFAULT PRIVILEGES IN SCHEMA storage
    GRANT ALL ON TABLES TO service_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA storage
    GRANT ALL ON SEQUENCES TO service_role;

-- Functions schema
ALTER DEFAULT PRIVILEGES IN SCHEMA functions
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO authenticated;
ALTER DEFAULT PRIVILEGES IN SCHEMA functions
    GRANT ALL ON TABLES TO service_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA functions
    GRANT ALL ON SEQUENCES TO service_role;

-- Realtime schema
ALTER DEFAULT PRIVILEGES IN SCHEMA realtime
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO authenticated;
ALTER DEFAULT PRIVILEGES IN SCHEMA realtime
    GRANT ALL ON TABLES TO service_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA realtime
    GRANT ALL ON SEQUENCES TO service_role;

-- Jobs schema
ALTER DEFAULT PRIVILEGES IN SCHEMA jobs
    GRANT SELECT ON TABLES TO authenticated;
ALTER DEFAULT PRIVILEGES IN SCHEMA jobs
    GRANT ALL ON TABLES TO service_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA jobs
    GRANT ALL ON SEQUENCES TO service_role;

-- AI schema
ALTER DEFAULT PRIVILEGES IN SCHEMA ai
    GRANT SELECT ON TABLES TO authenticated;
ALTER DEFAULT PRIVILEGES IN SCHEMA ai
    GRANT ALL ON TABLES TO service_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA ai
    GRANT ALL ON SEQUENCES TO service_role;

-- RPC schema
ALTER DEFAULT PRIVILEGES IN SCHEMA rpc
    GRANT ALL ON TABLES TO service_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA rpc
    GRANT ALL ON SEQUENCES TO service_role;

-- Logging schema (needed because pgschema skips PRIVILEGE entries;
-- the per-table GRANTs in logging.sql may not be applied by the declarative engine)
-- Default privileges for tables created by the bootstrap user (CURRENT_USER)
ALTER DEFAULT PRIVILEGES IN SCHEMA logging
    GRANT ALL ON TABLES TO service_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA logging
    GRANT ALL ON SEQUENCES TO service_role;
-- Default privileges for tables created by {{APP_USER}} (pgschema runtime).
-- pgschema operates as {{APP_USER}}, so this is the critical path for logging tables.
ALTER DEFAULT PRIVILEGES FOR ROLE {{APP_USER}} IN SCHEMA logging
    GRANT ALL ON TABLES TO service_role;
ALTER DEFAULT PRIVILEGES FOR ROLE {{APP_USER}} IN SCHEMA logging
    GRANT ALL ON SEQUENCES TO service_role;
-- tenant_service needs access for the logs dashboard (subject to RLS).
ALTER DEFAULT PRIVILEGES IN SCHEMA logging
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO tenant_service;
ALTER DEFAULT PRIVILEGES FOR ROLE {{APP_USER}} IN SCHEMA logging
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO tenant_service;
ALTER DEFAULT PRIVILEGES IN SCHEMA logging
    GRANT SELECT ON TABLES TO authenticated;
ALTER DEFAULT PRIVILEGES FOR ROLE {{APP_USER}} IN SCHEMA logging
    GRANT SELECT ON TABLES TO authenticated;

-- MCP schema
ALTER DEFAULT PRIVILEGES IN SCHEMA mcp
    GRANT ALL ON TABLES TO service_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA mcp
    GRANT ALL ON SEQUENCES TO service_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA mcp
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO authenticated;

-- Branching schema
ALTER DEFAULT PRIVILEGES IN SCHEMA branching
    GRANT ALL ON TABLES TO service_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA branching
    GRANT ALL ON SEQUENCES TO service_role;

-- Public schema - service_role and tenant_migration_role
-- Default privileges cover future tables created by any role in the public schema.
-- We also set per-role defaults to cover tables created by specific roles (bootstrap user, etc.).
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT ALL ON TABLES TO service_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT ALL ON SEQUENCES TO service_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT EXECUTE ON FUNCTIONS TO service_role;

-- Also set defaults for the bootstrap role explicitly (covers tables created during bootstrap)
ALTER DEFAULT PRIVILEGES FOR ROLE CURRENT_USER IN SCHEMA public
    GRANT ALL ON TABLES TO service_role;
ALTER DEFAULT PRIVILEGES FOR ROLE CURRENT_USER IN SCHEMA public
    GRANT EXECUTE ON FUNCTIONS TO service_role;

-- Grant permissions on all existing public schema tables to service_role
-- This covers tables created by any role before these defaults were in place.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = 'public') THEN
        -- Grant ALL on all existing tables in public schema
        PERFORM 1;
        BEGIN
            EXECUTE format('GRANT ALL ON ALL TABLES IN SCHEMA public TO service_role');
        EXCEPTION WHEN others THEN
            -- Ignore errors if no tables exist or other issues
            RAISE NOTICE 'Could not grant on public tables: %', SQLERRM;
        END;

        -- Grant ALL on all existing sequences in public schema
        BEGIN
            EXECUTE format('GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO service_role');
        EXCEPTION WHEN others THEN
            RAISE NOTICE 'Could not grant on public sequences: %', SQLERRM;
        END;

        -- Grant ALL on all existing functions in public schema
        BEGIN
            EXECUTE format('GRANT ALL ON ALL FUNCTIONS IN SCHEMA public TO service_role');
        EXCEPTION WHEN others THEN
            RAISE NOTICE 'Could not grant on public functions: %', SQLERRM;
        END;
    END IF;
END
$$;

-- Grant permissions on all existing logging tables to service_role, tenant_service, and authenticated.
-- pgschema skips PRIVILEGE entries during plan/apply, so the per-table
-- GRANTs in logging.sql may never be applied. This DO block ensures both
-- service_role and tenant_service can query logging.entries (used by the admin
-- stats API and the logs dashboard). tenant_service and authenticated are subject
-- to RLS policies (auth.has_tenant_access) so tenants only see their own logs.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = 'logging') THEN
        BEGIN
            EXECUTE 'GRANT ALL ON ALL TABLES IN SCHEMA logging TO service_role';
        EXCEPTION WHEN others THEN
            RAISE NOTICE 'Could not grant on logging tables to service_role: %', SQLERRM;
        END;

        BEGIN
            EXECUTE 'GRANT ALL ON ALL SEQUENCES IN SCHEMA logging TO service_role';
        EXCEPTION WHEN others THEN
            RAISE NOTICE 'Could not grant on logging sequences to service_role: %', SQLERRM;
        END;

        BEGIN
            EXECUTE 'GRANT USAGE ON SCHEMA logging TO tenant_service';
        EXCEPTION WHEN others THEN
            RAISE NOTICE 'Could not grant usage on logging schema to tenant_service: %', SQLERRM;
        END;

        BEGIN
            EXECUTE 'GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA logging TO tenant_service';
        EXCEPTION WHEN others THEN
            RAISE NOTICE 'Could not grant on logging tables to tenant_service: %', SQLERRM;
        END;

        BEGIN
            EXECUTE 'GRANT ALL ON ALL SEQUENCES IN SCHEMA logging TO tenant_service';
        EXCEPTION WHEN others THEN
            RAISE NOTICE 'Could not grant on logging sequences to tenant_service: %', SQLERRM;
        END;

        BEGIN
            EXECUTE 'GRANT USAGE ON SCHEMA logging TO authenticated';
        EXCEPTION WHEN others THEN
            RAISE NOTICE 'Could not grant usage on logging schema to authenticated: %', SQLERRM;
        END;

        BEGIN
            EXECUTE 'GRANT SELECT ON ALL TABLES IN SCHEMA logging TO authenticated';
        EXCEPTION WHEN others THEN
            RAISE NOTICE 'Could not grant on logging tables to authenticated: %', SQLERRM;
        END;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = 'branching') THEN
        BEGIN
            EXECUTE 'GRANT USAGE ON SCHEMA branching TO authenticated';
        EXCEPTION WHEN others THEN
            RAISE NOTICE 'Could not grant usage on branching schema to authenticated: %', SQLERRM;
        END;

        BEGIN
            EXECUTE 'GRANT SELECT ON ALL TABLES IN SCHEMA branching TO authenticated';
        EXCEPTION WHEN others THEN
            RAISE NOTICE 'Could not grant on branching tables to authenticated: %', SQLERRM;
        END;
    END IF;
END
$$;

-- Tenant migration role - public schema
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT ALL ON TABLES TO tenant_migration_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT ALL ON SEQUENCES TO tenant_migration_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT EXECUTE ON FUNCTIONS TO tenant_migration_role;

-- ============================================================================
-- MIGRATIONS STATE TABLES
-- ============================================================================

-- Declarative schema state tracking
CREATE TABLE IF NOT EXISTS platform.declarative_state (
    id SERIAL PRIMARY KEY,
    schema_fingerprint TEXT NOT NULL,
    applied_at TIMESTAMPTZ DEFAULT NOW(),
    applied_by TEXT,
    source TEXT CHECK (source IN ('fresh_install', 'transitioned', 'schema_apply'))
);

-- Bootstrap state tracking
CREATE TABLE IF NOT EXISTS platform.bootstrap_state (
    id SERIAL PRIMARY KEY,
    bootstrapped_at TIMESTAMPTZ DEFAULT NOW(),
    version TEXT NOT NULL,
    checksum TEXT
);

-- Grant permissions on platform migration tables to service_role
GRANT SELECT, INSERT, UPDATE ON TABLE platform.declarative_state TO service_role;
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM pg_sequences WHERE schemaname = 'platform' AND sequencename = 'declarative_state_id_seq') THEN
        GRANT USAGE, SELECT ON SEQUENCE platform.declarative_state_id_seq TO service_role;
    END IF;
END $$;
GRANT SELECT, INSERT ON TABLE platform.bootstrap_state TO service_role;

-- Record bootstrap completion (idempotent)
INSERT INTO platform.bootstrap_state (version, checksum)
VALUES ('2.0.0', '')
ON CONFLICT DO NOTHING;

-- ============================================================================
-- MIGRATION: Move legacy dashboard.* tables to platform schema
-- Idempotent — safe to run multiple times, no-ops if already migrated.
-- ============================================================================
DO $$
DECLARE
    tbl record;
BEGIN
    -- Skip if dashboard schema doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = 'dashboard') THEN
        RETURN;
    END IF;

    FOR tbl IN
        SELECT table_name FROM information_schema.tables
        WHERE table_schema = 'dashboard' AND table_type = 'BASE TABLE'
    LOOP
        -- Skip if table already exists in platform schema (platform is authoritative)
        IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'platform' AND table_name = tbl.table_name AND table_type = 'BASE TABLE') THEN
            EXECUTE format('ALTER TABLE dashboard.%I SET SCHEMA platform', tbl.table_name);
            RAISE NOTICE 'Migrated dashboard.% to platform.%', tbl.table_name, tbl.table_name;
        ELSE
            EXECUTE format('DROP TABLE dashboard.%I CASCADE', tbl.table_name);
            RAISE NOTICE 'Dropped duplicate dashboard.% (platform.% already exists)', tbl.table_name, tbl.table_name;
        END IF;
    END LOOP;

    -- Drop the now-empty dashboard schema
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'dashboard' AND table_type = 'BASE TABLE') THEN
        EXECUTE 'DROP SCHEMA IF EXISTS dashboard CASCADE';
        RAISE NOTICE 'Dropped empty dashboard schema';
    END IF;
END
$$;

-- ============================================================================
-- MIGRATION: Backfill NULL tenant_id in storage tables to default tenant
-- Idempotent — safe to run multiple times.
-- ============================================================================
DO $$
DECLARE
    v_default_tenant_id UUID;
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_schema = 'platform' AND table_name = 'tenants'
              AND table_type = 'BASE TABLE'
    ) THEN
        RAISE NOTICE 'platform.tenants does not exist yet, skipping storage tenant_id backfill';
        RETURN;
    END IF;

    SELECT id INTO v_default_tenant_id
    FROM platform.tenants
    WHERE is_default = true AND deleted_at IS NULL
    LIMIT 1;

    IF v_default_tenant_id IS NULL THEN
        RAISE NOTICE 'No default tenant found, skipping storage tenant_id backfill';
        RETURN;
    END IF;

    UPDATE storage.buckets SET tenant_id = v_default_tenant_id WHERE tenant_id IS NULL;
    UPDATE storage.objects SET tenant_id = v_default_tenant_id WHERE tenant_id IS NULL;
    UPDATE storage.chunked_upload_sessions SET tenant_id = v_default_tenant_id WHERE tenant_id IS NULL;
    UPDATE storage.object_permissions SET tenant_id = v_default_tenant_id WHERE tenant_id IS NULL;

    RAISE NOTICE 'Backfilled storage tenant_id to default tenant %', v_default_tenant_id;
END
$$;

-- ============================================================================
-- MIGRATION: Update legacy dashboard_admin role to instance_admin
-- Idempotent — safe to run multiple times.
-- ============================================================================
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_schema = 'platform' AND table_name = 'users'
              AND table_type = 'BASE TABLE'
    ) THEN
        RAISE NOTICE 'platform.users does not exist yet, skipping dashboard_admin migration';
        RETURN;
    END IF;

    UPDATE platform.users
    SET role = 'instance_admin', updated_at = NOW()
    WHERE role = 'dashboard_admin';

    IF FOUND THEN
        RAISE NOTICE 'Migrated dashboard_admin users to instance_admin';
    END IF;
END
$$;

-- ============================================================================
-- MIGRATION: Move legacy system/api/migrations schema tables to platform
-- Idempotent — safe to run multiple times, no-ops if already migrated.
-- ============================================================================
DO $$
DECLARE
    rec record;
    v_new_name text;
BEGIN
    -- Map of (old_schema, old_table_name) -> new_table_name
    -- Tables that keep the same name: rate_limits, idempotency_keys,
    -- declarative_state, bootstrap_state
    -- Tables that are renamed: app -> migrations, execution_logs -> migration_execution_logs,
    -- fluxbase -> fluxbase_migrations

    FOR rec IN
        SELECT s.schema_name AS old_schema, t.table_name
        FROM information_schema.schemata s
        JOIN information_schema.tables t ON t.table_schema = s.schema_name
        WHERE s.schema_name IN ('system', 'api', 'migrations')
          AND t.table_type = 'BASE TABLE'
    LOOP
        -- Determine the new table name
        CASE
            WHEN rec.old_schema = 'migrations' AND rec.table_name = 'app' THEN
                v_new_name := 'migrations';
            WHEN rec.old_schema = 'migrations' AND rec.table_name = 'execution_logs' THEN
                v_new_name := 'migration_execution_logs';
            WHEN rec.old_schema = 'migrations' AND rec.table_name = 'fluxbase' THEN
                v_new_name := 'fluxbase_migrations';
            ELSE
                v_new_name := rec.table_name;
        END CASE;

        -- Skip if target table already exists in platform schema (platform is authoritative)
        IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'platform' AND table_name = v_new_name AND table_type = 'BASE TABLE') THEN
            EXECUTE format('ALTER TABLE %I.%I SET SCHEMA platform', rec.old_schema, rec.table_name);
            IF v_new_name != rec.table_name THEN
                EXECUTE format('ALTER TABLE platform.%I RENAME TO %I', rec.table_name, v_new_name);
            END IF;
            RAISE NOTICE 'Migrated %.% to platform.%', rec.old_schema, rec.table_name, v_new_name;
        ELSE
            EXECUTE format('DROP TABLE %I.%I CASCADE', rec.old_schema, rec.table_name);
            RAISE NOTICE 'Dropped duplicate %.% (platform.% already exists)', rec.old_schema, rec.table_name, v_new_name;
        END IF;
    END LOOP;

    -- Rename constraints on migrated tables to match declarative schema expectations.
    -- Without this, pgschema sees old constraint names (e.g., app_pkey) and tries to
    -- drop/recreate them, which fails when other objects depend on those constraints.
    --
    -- Uses pg_class joins instead of ::regclass to avoid errors when tables
    -- don't exist (e.g., on fresh tenant databases where migration is a no-op).

    -- migrations.app -> platform.migrations
    IF EXISTS (
        SELECT 1 FROM pg_constraint c
        JOIN pg_class t ON t.oid = c.conrelid
        JOIN pg_namespace n ON n.oid = t.relnamespace
        WHERE c.conname = 'app_pkey' AND n.nspname = 'platform' AND t.relname = 'migrations'
    ) THEN
        ALTER TABLE platform.migrations RENAME CONSTRAINT app_pkey TO migrations_pkey;
    END IF;

    -- migrations.execution_logs -> platform.migration_execution_logs
    IF EXISTS (
        SELECT 1 FROM pg_constraint c
        JOIN pg_class t ON t.oid = c.conrelid
        JOIN pg_namespace n ON n.oid = t.relnamespace
        WHERE c.conname = 'execution_logs_pkey' AND n.nspname = 'platform' AND t.relname = 'migration_execution_logs'
    ) THEN
        ALTER TABLE platform.migration_execution_logs RENAME CONSTRAINT execution_logs_pkey TO migration_execution_logs_pkey;
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_constraint c
        JOIN pg_class t ON t.oid = c.conrelid
        JOIN pg_namespace n ON n.oid = t.relnamespace
        WHERE c.conname = 'execution_logs_migration_id_fkey' AND n.nspname = 'platform' AND t.relname = 'migration_execution_logs'
    ) THEN
        ALTER TABLE platform.migration_execution_logs RENAME CONSTRAINT execution_logs_migration_id_fkey TO migration_execution_logs_migration_id_fkey;
    END IF;

    -- migrations.fluxbase -> platform.fluxbase_migrations
    IF EXISTS (
        SELECT 1 FROM pg_constraint c
        JOIN pg_class t ON t.oid = c.conrelid
        JOIN pg_namespace n ON n.oid = t.relnamespace
        WHERE c.conname = 'fluxbase_pkey' AND n.nspname = 'platform' AND t.relname = 'fluxbase_migrations'
    ) THEN
        ALTER TABLE platform.fluxbase_migrations RENAME CONSTRAINT fluxbase_pkey TO fluxbase_migrations_pkey;
    END IF;

    -- Drop the now-empty old schemas
    IF EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = 'system')
       AND NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'system' AND table_type = 'BASE TABLE') THEN
        EXECUTE 'DROP SCHEMA IF EXISTS system CASCADE';
        RAISE NOTICE 'Dropped empty system schema';
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = 'api')
       AND NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'api' AND table_type = 'BASE TABLE') THEN
        EXECUTE 'DROP SCHEMA IF EXISTS api CASCADE';
        RAISE NOTICE 'Dropped empty api schema';
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = 'migrations')
       AND NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'migrations' AND table_type = 'BASE TABLE') THEN
        EXECUTE 'DROP SCHEMA IF EXISTS migrations CASCADE';
        RAISE NOTICE 'Dropped empty migrations schema';
    END IF;
END
$$;
