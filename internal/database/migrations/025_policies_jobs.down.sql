-- Drop RLS policies on jobs tables

-- Dashboard admin policies
-- Note: Execution logs are now stored in the central logging schema (logging.entries)
DROP POLICY IF EXISTS "Dashboard admins can read all function files" ON jobs.function_files;
DROP POLICY IF EXISTS "Dashboard admins can read all workers" ON jobs.workers;
DROP POLICY IF EXISTS "Dashboard admins can read all functions" ON jobs.functions;
DROP POLICY IF EXISTS "Dashboard admins can read all jobs" ON jobs.queue;

-- Other policies
DROP POLICY IF EXISTS "Service role can manage functions" ON jobs.functions;
DROP POLICY IF EXISTS "Users can read their own jobs" ON jobs.queue;
DROP POLICY IF EXISTS "Users can submit jobs" ON jobs.queue;
DROP POLICY IF EXISTS "Users can cancel their own pending/running jobs" ON jobs.queue;
DROP POLICY IF EXISTS "Service role can manage all jobs" ON jobs.queue;
DROP POLICY IF EXISTS "Service role can manage workers" ON jobs.workers;
DROP POLICY IF EXISTS "Service role can manage function files" ON jobs.function_files;

-- Disable RLS
ALTER TABLE jobs.functions DISABLE ROW LEVEL SECURITY;
ALTER TABLE jobs.queue DISABLE ROW LEVEL SECURITY;
ALTER TABLE jobs.workers DISABLE ROW LEVEL SECURITY;
ALTER TABLE jobs.function_files DISABLE ROW LEVEL SECURITY;
