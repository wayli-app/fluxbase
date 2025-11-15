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
GRANT USAGE ON SCHEMA public TO anon, authenticated, service_role;

-- Note: Specific table permissions will be managed through RLS policies
-- The dashboard and app schemas contain admin settings, access controlled via RLS
