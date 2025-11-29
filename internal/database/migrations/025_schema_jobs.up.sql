-- Create jobs schema
CREATE SCHEMA IF NOT EXISTS jobs;

COMMENT ON SCHEMA jobs IS 'Long-running background jobs system';

-- Grant schema access to service_role for admin operations
GRANT USAGE ON SCHEMA jobs TO service_role;
GRANT ALL ON ALL TABLES IN SCHEMA jobs TO service_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA jobs TO service_role;
