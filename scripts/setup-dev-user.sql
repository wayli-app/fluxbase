-- Setup script for fluxbase_app user in local development
-- This script grants all necessary permissions to the fluxbase_app user
-- for local development and testing with RLS enabled.
--
-- NOTE: This script is for backward compatibility only.
-- Fluxbase now uses configurable database users:
-- - FLUXBASE_DATABASE_ADMIN_USER: Runs migrations (default: postgres)
-- - FLUXBASE_DATABASE_RUNTIME_USER: Runtime operations (default: same as admin)
-- Migrations automatically grant permissions to CURRENT_USER (whoever runs them).

-- Grant CREATE privilege on the database
GRANT CREATE ON DATABASE fluxbase_dev TO fluxbase_app;

-- Grant ALL privileges on all schemas
GRANT ALL ON SCHEMA public TO fluxbase_app;
GRANT ALL ON SCHEMA auth TO fluxbase_app;
GRANT ALL ON SCHEMA storage TO fluxbase_app;
GRANT ALL ON SCHEMA functions TO fluxbase_app;
GRANT ALL ON SCHEMA realtime TO fluxbase_app;
GRANT ALL ON SCHEMA dashboard TO fluxbase_app;

-- Grant ALL PRIVILEGES on all existing tables
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO fluxbase_app;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA auth TO fluxbase_app;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA storage TO fluxbase_app;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA functions TO fluxbase_app;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA realtime TO fluxbase_app;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA dashboard TO fluxbase_app;

-- Grant ALL PRIVILEGES on all existing sequences
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO fluxbase_app;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA auth TO fluxbase_app;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA storage TO fluxbase_app;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA functions TO fluxbase_app;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA realtime TO fluxbase_app;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA dashboard TO fluxbase_app;

-- Grant EXECUTE on all functions
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO fluxbase_app;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA auth TO fluxbase_app;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA storage TO fluxbase_app;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA functions TO fluxbase_app;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA realtime TO fluxbase_app;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA dashboard TO fluxbase_app;

-- Change owner of all tables to fluxbase_app
-- This is needed for operations like upsert that require ownership
DO $$
DECLARE
    r RECORD;
BEGIN
    -- Change owner of all tables to fluxbase_app
    FOR r IN SELECT schemaname, tablename FROM pg_tables WHERE schemaname IN ('public', 'auth', 'storage', 'functions', 'realtime', 'dashboard')
    LOOP
        EXECUTE 'ALTER TABLE ' || quote_ident(r.schemaname) || '.' || quote_ident(r.tablename) || ' OWNER TO fluxbase_app';
    END LOOP;

    -- Change owner of all sequences to fluxbase_app
    FOR r IN SELECT schemaname, sequencename FROM pg_sequences WHERE schemaname IN ('public', 'auth', 'storage', 'functions', 'realtime', 'dashboard')
    LOOP
        EXECUTE 'ALTER SEQUENCE ' || quote_ident(r.schemaname) || '.' || quote_ident(r.sequencename) || ' OWNER TO fluxbase_app';
    END LOOP;

    -- Change owner of all schemas to fluxbase_app (optional but recommended)
    FOR r IN SELECT nspname FROM pg_namespace WHERE nspname IN ('auth', 'storage', 'functions', 'realtime', 'dashboard')
    LOOP
        EXECUTE 'ALTER SCHEMA ' || quote_ident(r.nspname) || ' OWNER TO fluxbase_app';
    END LOOP;
END$$;
