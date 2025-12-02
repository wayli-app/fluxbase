-- ============================================================================
-- JOBS SCHEMA PERMISSIONS FOR SERVICE_ROLE
-- ============================================================================
-- The jobs schema was created after migration 024_grant_role_permissions.
-- This migration grants service_role (used by dashboard_admin) full access
-- to the jobs schema, matching the pattern in 024 for other schemas.
-- ============================================================================

-- Grant schema usage
GRANT USAGE ON SCHEMA jobs TO service_role;

-- Grant ALL on all existing tables and sequences
GRANT ALL ON ALL TABLES IN SCHEMA jobs TO service_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA jobs TO service_role;

-- Set default privileges for future tables
ALTER DEFAULT PRIVILEGES IN SCHEMA jobs
    GRANT ALL ON TABLES TO service_role;

ALTER DEFAULT PRIVILEGES IN SCHEMA jobs
    GRANT ALL ON SEQUENCES TO service_role;
