-- Grant permissions for jobs.execution_logs table
-- This table was created in migration 031, after the general grants in migration 030,
-- so it needs explicit grants for authenticated role access.

-- Grant SELECT to authenticated role (required for RLS policies to work)
GRANT SELECT ON jobs.execution_logs TO authenticated;

-- Grant ALL to service_role for admin operations
GRANT ALL ON jobs.execution_logs TO service_role;

-- Grant usage on the sequence for service_role
GRANT USAGE, SELECT ON SEQUENCE jobs.execution_logs_id_seq TO service_role;
