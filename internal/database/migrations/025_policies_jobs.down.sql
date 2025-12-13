-- Drop RLS policies on jobs tables

-- Dashboard admin policies
DROP POLICY IF EXISTS "Dashboard admins can read all execution logs" ON jobs.execution_logs;
DROP POLICY IF EXISTS "Dashboard admins can read all function files" ON jobs.function_files;
DROP POLICY IF EXISTS "Dashboard admins can read all workers" ON jobs.workers;
DROP POLICY IF EXISTS "Dashboard admins can read all functions" ON jobs.functions;
DROP POLICY IF EXISTS "Dashboard admins can read all jobs" ON jobs.queue;

-- Execution logs policies
DROP POLICY IF EXISTS "Service role can manage execution logs" ON jobs.execution_logs;
DROP POLICY IF EXISTS "Users can read their own job logs" ON jobs.execution_logs;

-- Other policies
DROP POLICY IF EXISTS "Service role can manage functions" ON jobs.functions;
DROP POLICY IF EXISTS "Users can read their own jobs" ON jobs.queue;
DROP POLICY IF EXISTS "Users can submit jobs" ON jobs.queue;
DROP POLICY IF EXISTS "Users can cancel their own pending/running jobs" ON jobs.queue;
DROP POLICY IF EXISTS "Service role can manage all jobs" ON jobs.queue;
DROP POLICY IF EXISTS "Service role can manage workers" ON jobs.workers;
DROP POLICY IF EXISTS "Service role can manage function files" ON jobs.function_files;

-- Disable RLS
ALTER TABLE jobs.execution_logs DISABLE ROW LEVEL SECURITY;
ALTER TABLE jobs.functions DISABLE ROW LEVEL SECURITY;
ALTER TABLE jobs.queue DISABLE ROW LEVEL SECURITY;
ALTER TABLE jobs.workers DISABLE ROW LEVEL SECURITY;
ALTER TABLE jobs.function_files DISABLE ROW LEVEL SECURITY;
