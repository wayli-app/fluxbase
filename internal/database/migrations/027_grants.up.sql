-- ============================================================================
-- GRANT TABLE PERMISSIONS TO RLS ROLES
-- ============================================================================
-- This migration grants table-level permissions to anon, authenticated, and
-- service_role for use with SET ROLE in RLS middleware.
-- Actual data access is still controlled by RLS policies.
-- This runs AFTER all tables are created (migrations 004-021).
--
-- Security principle: "closed by default"
-- - Anon has minimal access (only what's explicitly needed)
-- - Authenticated has CRUD on user-facing schemas (controlled by RLS)
-- - Service role has full access (BYPASSRLS)
-- - Internal schemas (migrations) are service_role only
-- - Public schema requires explicit RLS policies for access
-- ============================================================================

-- ==========================
-- AUTH SCHEMA PERMISSIONS
-- ==========================

-- Anon role: No direct access (signup/signin use service_role internally)
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
-- PUBLIC SCHEMA PERMISSIONS
-- ==========================

-- Public schema: "closed by default"
-- No default access for anon or authenticated users
-- Developers must create RLS policies to grant access to their tables
-- This prevents accidental data exposure if RLS is not enabled
GRANT ALL ON ALL TABLES IN SCHEMA public TO service_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO service_role;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO service_role;

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

-- Default privileges for public schema - service_role only
ALTER DEFAULT PRIVILEGES FOR ROLE CURRENT_USER IN SCHEMA public
    GRANT ALL ON TABLES TO service_role;

ALTER DEFAULT PRIVILEGES FOR ROLE CURRENT_USER IN SCHEMA public
    GRANT EXECUTE ON FUNCTIONS TO service_role;

-- ==========================
-- JOBS SCHEMA PERMISSIONS
-- ==========================

-- Grant schema usage to authenticated and service_role
GRANT USAGE ON SCHEMA jobs TO authenticated, service_role;

-- Authenticated role: Can view jobs data (controlled by RLS)
GRANT SELECT ON ALL TABLES IN SCHEMA jobs TO authenticated;

-- Service role: Full access for admin operations
GRANT ALL ON ALL TABLES IN SCHEMA jobs TO service_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA jobs TO service_role;

-- Default privileges for future tables
ALTER DEFAULT PRIVILEGES IN SCHEMA jobs
    GRANT SELECT ON TABLES TO authenticated;

ALTER DEFAULT PRIVILEGES IN SCHEMA jobs
    GRANT ALL ON TABLES TO service_role;

ALTER DEFAULT PRIVILEGES IN SCHEMA jobs
    GRANT ALL ON SEQUENCES TO service_role;

-- ==========================
-- AI SCHEMA PERMISSIONS
-- ==========================

-- Grant schema usage
GRANT USAGE ON SCHEMA ai TO anon, authenticated, service_role;

-- Authenticated role: Can interact with AI features (controlled by RLS)
GRANT SELECT ON ALL TABLES IN SCHEMA ai TO authenticated;
GRANT ALL ON ai.user_provider_preferences TO authenticated;
GRANT ALL ON ai.conversations TO authenticated;
GRANT ALL ON ai.messages TO authenticated;

-- Service role: Full access
GRANT ALL ON ALL TABLES IN SCHEMA ai TO service_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA ai TO service_role;

-- Default privileges
ALTER DEFAULT PRIVILEGES IN SCHEMA ai
    GRANT SELECT ON TABLES TO authenticated;

ALTER DEFAULT PRIVILEGES IN SCHEMA ai
    GRANT ALL ON TABLES TO service_role;

-- ==========================
-- RPC SCHEMA PERMISSIONS
-- ==========================

-- Grant schema usage
GRANT USAGE ON SCHEMA rpc TO authenticated, service_role;

-- Service role: Full access for RPC execution
GRANT ALL ON ALL TABLES IN SCHEMA rpc TO service_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA rpc TO service_role;

-- Default privileges
ALTER DEFAULT PRIVILEGES IN SCHEMA rpc
    GRANT ALL ON TABLES TO service_role;
