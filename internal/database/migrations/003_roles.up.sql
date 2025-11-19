-- Fluxbase Database Roles
-- These roles are used for authentication and authorization
-- Similar to Supabase's role system

-- Anonymous role: For unauthenticated requests
-- This role has minimal permissions and is used for public access
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'anon') THEN
        CREATE ROLE anon NOLOGIN NOINHERIT;
    END IF;
END
$$;

COMMENT ON ROLE anon IS 'Anonymous role for unauthenticated requests with public access only';

-- Authenticated role: For authenticated users
-- This role is used when a valid JWT token is provided
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'authenticated') THEN
        CREATE ROLE authenticated NOLOGIN NOINHERIT;
    END IF;
END
$$;

COMMENT ON ROLE authenticated IS 'Authenticated role for users with valid JWT tokens';

-- Service role: For backend services and admin operations
-- This role bypasses RLS and has elevated permissions
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'service_role') THEN
        CREATE ROLE service_role NOLOGIN NOINHERIT BYPASSRLS;
    END IF;
END
$$;

COMMENT ON ROLE service_role IS 'Service role for backend services with elevated permissions and RLS bypass';

-- Grant schema usage permissions to roles
GRANT USAGE ON SCHEMA auth TO anon, authenticated, service_role;
GRANT USAGE ON SCHEMA app TO anon, authenticated, service_role;
GRANT USAGE ON SCHEMA storage TO anon, authenticated, service_role;
GRANT USAGE ON SCHEMA functions TO anon, authenticated, service_role;
GRANT USAGE ON SCHEMA realtime TO anon, authenticated, service_role;
GRANT USAGE ON SCHEMA dashboard TO anon, authenticated, service_role;
GRANT USAGE ON SCHEMA public TO anon, authenticated, service_role;

-- Grant application users permission to SET ROLE to anon, authenticated, service_role
-- This allows the RLS middleware to execute SET LOCAL ROLE for defense-in-depth security
-- The actual application users are: fluxbase_app (production) and fluxbase_rls_test (testing)
GRANT anon TO fluxbase_app;
GRANT authenticated TO fluxbase_app;
GRANT service_role TO fluxbase_app;

-- Also grant to test user (may not exist in production, hence DO block)
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'fluxbase_rls_test') THEN
        GRANT anon TO fluxbase_rls_test;
        GRANT authenticated TO fluxbase_rls_test;
        GRANT service_role TO fluxbase_rls_test;
    END IF;
END
$$;

-- Note: Table-level permissions are granted in migration 022 (after all tables are created)
-- This migration only creates roles and grants schema access
