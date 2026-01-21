-- ============================================================================
-- GRANT PERMISSIONS FOR MCP SCHEMA
-- ============================================================================
-- This migration adds the missing permissions for the mcp schema created in 068.
-- Migration 068 forgot to grant USAGE to CURRENT_USER (like 002_schemas.up.sql does).
-- ============================================================================

-- Grant schema usage to CURRENT_USER (the migration/runtime user)
-- This was missing from 068_custom_mcp.up.sql
GRANT USAGE, CREATE ON SCHEMA mcp TO CURRENT_USER;

-- Grant schema usage to RLS roles for SET ROLE operations
GRANT USAGE ON SCHEMA mcp TO anon, authenticated, service_role;

-- Service role: Full access for admin operations
GRANT ALL ON ALL TABLES IN SCHEMA mcp TO service_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA mcp TO service_role;

-- Authenticated role: Can manage MCP tools/resources (admin dashboard uses this via SET ROLE)
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA mcp TO authenticated;

-- Default privileges for future tables
ALTER DEFAULT PRIVILEGES IN SCHEMA mcp
    GRANT ALL ON TABLES TO service_role;

ALTER DEFAULT PRIVILEGES IN SCHEMA mcp
    GRANT ALL ON SEQUENCES TO service_role;

ALTER DEFAULT PRIVILEGES IN SCHEMA mcp
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO authenticated;
