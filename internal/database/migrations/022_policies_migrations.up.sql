--
-- POLICIES FOR MIGRATIONS SCHEMA
-- Row-level security and permissions for API-managed migrations
--

-- Grant permissions to service_role (admin operations)
GRANT USAGE ON SCHEMA migrations TO service_role;
GRANT ALL ON ALL TABLES IN SCHEMA migrations TO service_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA migrations TO service_role;

-- Allow authenticated users to view their namespace's migrations (read-only)
GRANT USAGE ON SCHEMA migrations TO authenticated;
GRANT SELECT ON migrations.migrations TO authenticated;
GRANT SELECT ON migrations.execution_logs TO authenticated;
