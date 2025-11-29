-- Remove RLS policies
DROP POLICY IF EXISTS "Dashboard admins can read all jobs" ON jobs.job_queue;
DROP POLICY IF EXISTS "Dashboard admins can read all job functions" ON jobs.job_functions;
DROP POLICY IF EXISTS "Dashboard admins can read all workers" ON jobs.workers;
DROP POLICY IF EXISTS "Dashboard admins can read all job function files" ON jobs.job_function_files;

-- Remove from registry
DELETE FROM realtime.schema_registry
WHERE schema_name = 'jobs';

-- Drop triggers
DROP TRIGGER IF EXISTS job_queue_realtime_notify ON jobs.job_queue;
-- DROP TRIGGER IF EXISTS job_functions_realtime_notify ON jobs.job_functions; -- Never created
DROP TRIGGER IF EXISTS workers_realtime_notify ON jobs.workers;
DROP TRIGGER IF EXISTS job_function_files_realtime_notify ON jobs.job_function_files;

-- Drop function
DROP FUNCTION IF EXISTS jobs.notify_realtime_change();

-- Reset replica identity
ALTER TABLE jobs.job_queue REPLICA IDENTITY DEFAULT;
ALTER TABLE jobs.job_functions REPLICA IDENTITY DEFAULT;
ALTER TABLE jobs.workers REPLICA IDENTITY DEFAULT;
ALTER TABLE jobs.job_function_files REPLICA IDENTITY DEFAULT;
