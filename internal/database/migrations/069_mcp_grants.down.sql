-- Revoke MCP schema permissions

-- Revoke default privileges
ALTER DEFAULT PRIVILEGES IN SCHEMA mcp
    REVOKE ALL ON TABLES FROM service_role;

ALTER DEFAULT PRIVILEGES IN SCHEMA mcp
    REVOKE ALL ON SEQUENCES FROM service_role;

ALTER DEFAULT PRIVILEGES IN SCHEMA mcp
    REVOKE ALL ON TABLES FROM authenticated;

-- Revoke table permissions
REVOKE ALL ON ALL TABLES IN SCHEMA mcp FROM service_role;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA mcp FROM service_role;
REVOKE ALL ON ALL TABLES IN SCHEMA mcp FROM authenticated;

-- Revoke schema usage
REVOKE USAGE ON SCHEMA mcp FROM anon, authenticated, service_role;
REVOKE USAGE, CREATE ON SCHEMA mcp FROM CURRENT_USER;
