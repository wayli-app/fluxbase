--
-- POLICIES FOR MIGRATIONS SCHEMA
-- Permissions for migration tracking tables
-- Service role only - no user access to migration internals
--

-- Grant permissions to service_role only (admin operations)
GRANT USAGE ON SCHEMA migrations TO service_role;
GRANT ALL ON ALL TABLES IN SCHEMA migrations TO service_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA migrations TO service_role;

-- Default privileges for future tables
ALTER DEFAULT PRIVILEGES IN SCHEMA migrations
    GRANT ALL ON TABLES TO service_role;

ALTER DEFAULT PRIVILEGES IN SCHEMA migrations
    GRANT ALL ON SEQUENCES TO service_role;
