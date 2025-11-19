-- ============================================================================
-- GRANT TABLE PERMISSIONS TO RLS ROLES
-- ============================================================================
-- This migration grants table-level permissions to anon, authenticated, and
-- service_role for use with SET ROLE in RLS middleware.
-- Actual data access is still controlled by RLS policies.
-- This runs AFTER all tables are created (migrations 004-021).
-- ============================================================================

-- ==========================
-- AUTH SCHEMA PERMISSIONS
-- ==========================

-- Anon role: Minimal permissions for signup flow only
-- INSERT on users (for signup) - RLS policies will restrict SELECT
GRANT INSERT ON auth.users TO anon;
GRANT INSERT ON auth.sessions TO anon;

-- Authenticated role: Full CRUD on auth tables (filtered by RLS)
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA auth TO authenticated;

-- Service role: Already has BYPASSRLS, grant all permissions
GRANT ALL ON ALL TABLES IN SCHEMA auth TO service_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA auth TO service_role;

-- ==========================
-- APP SCHEMA PERMISSIONS
-- ==========================

-- Anon role: Can read public settings only (controlled by RLS policies)
GRANT SELECT ON app.settings TO anon;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA app TO authenticated;
GRANT ALL ON ALL TABLES IN SCHEMA app TO service_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA app TO service_role;

-- ==========================
-- STORAGE SCHEMA PERMISSIONS
-- ==========================

-- Anon role: Can view public buckets and objects (controlled by RLS policies)
GRANT SELECT ON storage.buckets TO anon;
GRANT SELECT ON storage.objects TO anon;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA storage TO authenticated;
GRANT ALL ON ALL TABLES IN SCHEMA storage TO service_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA storage TO service_role;

-- ==========================
-- FUNCTIONS SCHEMA PERMISSIONS
-- ==========================

-- Anon role: No access to function configurations
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA functions TO authenticated;
GRANT ALL ON ALL TABLES IN SCHEMA functions TO service_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA functions TO service_role;

-- ==========================
-- REALTIME SCHEMA PERMISSIONS
-- ==========================

-- Anon role: No access to realtime configurations
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA realtime TO authenticated;
GRANT ALL ON ALL TABLES IN SCHEMA realtime TO service_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA realtime TO service_role;

-- ==========================
-- DASHBOARD SCHEMA PERMISSIONS
-- ==========================

-- Dashboard is admin-only, accessed via authenticated role with role checks
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA dashboard TO authenticated;
GRANT ALL ON ALL TABLES IN SCHEMA dashboard TO service_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA dashboard TO service_role;

-- ==========================
-- _FLUXBASE SCHEMA PERMISSIONS
-- ==========================

-- Anon role: No access to internal Fluxbase tables
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA _fluxbase TO authenticated;
GRANT ALL ON ALL TABLES IN SCHEMA _fluxbase TO service_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA _fluxbase TO service_role;

-- ==========================
-- PUBLIC SCHEMA PERMISSIONS
-- ==========================

-- Grant full CRUD to all roles on public schema tables
-- Note: RLS policies on individual tables control actual data access
-- For tables without RLS (like products used in REST tests), all roles have full access
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO anon, authenticated;
GRANT ALL ON ALL TABLES IN SCHEMA public TO service_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO service_role;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO anon, authenticated, service_role;

-- ==========================
-- DEFAULT PRIVILEGES
-- ==========================
-- Ensure future tables automatically get permissions
-- Note: Anon gets minimal default privileges; grant explicitly per table

ALTER DEFAULT PRIVILEGES IN SCHEMA auth
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO authenticated;

ALTER DEFAULT PRIVILEGES IN SCHEMA auth
    GRANT ALL ON TABLES TO service_role;

ALTER DEFAULT PRIVILEGES IN SCHEMA app
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO authenticated;

ALTER DEFAULT PRIVILEGES IN SCHEMA app
    GRANT ALL ON TABLES TO service_role;

ALTER DEFAULT PRIVILEGES IN SCHEMA storage
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO authenticated;

ALTER DEFAULT PRIVILEGES IN SCHEMA storage
    GRANT ALL ON TABLES TO service_role;

ALTER DEFAULT PRIVILEGES IN SCHEMA functions
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO authenticated;

ALTER DEFAULT PRIVILEGES IN SCHEMA functions
    GRANT ALL ON TABLES TO service_role;

ALTER DEFAULT PRIVILEGES IN SCHEMA realtime
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO authenticated;

ALTER DEFAULT PRIVILEGES IN SCHEMA realtime
    GRANT ALL ON TABLES TO service_role;

ALTER DEFAULT PRIVILEGES IN SCHEMA dashboard
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO authenticated;

ALTER DEFAULT PRIVILEGES IN SCHEMA dashboard
    GRANT ALL ON TABLES TO service_role;

ALTER DEFAULT PRIVILEGES IN SCHEMA _fluxbase
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO authenticated;

ALTER DEFAULT PRIVILEGES IN SCHEMA _fluxbase
    GRANT ALL ON TABLES TO service_role;

-- Default privileges for tables created by fluxbase_app
-- Note: Test tables (like products) are created by fluxbase_app in both local and CI
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO anon, authenticated;

ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA public
    GRANT ALL ON TABLES TO service_role;
