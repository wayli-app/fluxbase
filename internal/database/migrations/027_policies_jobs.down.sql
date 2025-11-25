-- Drop RLS policies on jobs tables
DROP POLICY IF EXISTS "Service role can manage job functions" ON jobs.job_functions;
DROP POLICY IF EXISTS "Users can read their own jobs" ON jobs.job_queue;
DROP POLICY IF EXISTS "Users can submit jobs" ON jobs.job_queue;
DROP POLICY IF EXISTS "Users can cancel their own pending/running jobs" ON jobs.job_queue;
DROP POLICY IF EXISTS "Service role can manage all jobs" ON jobs.job_queue;
DROP POLICY IF EXISTS "Service role can manage workers" ON jobs.workers;
DROP POLICY IF EXISTS "Service role can manage job function files" ON jobs.job_function_files;

-- Disable RLS
ALTER TABLE jobs.job_functions DISABLE ROW LEVEL SECURITY;
ALTER TABLE jobs.job_queue DISABLE ROW LEVEL SECURITY;
ALTER TABLE jobs.workers DISABLE ROW LEVEL SECURITY;
ALTER TABLE jobs.job_function_files DISABLE ROW LEVEL SECURITY;
