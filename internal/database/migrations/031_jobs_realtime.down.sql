-- Remove RLS policies
DROP POLICY IF EXISTS "Dashboard admins can read all jobs" ON jobs.queue;
DROP POLICY IF EXISTS "Dashboard admins can read all functions" ON jobs.functions;
DROP POLICY IF EXISTS "Dashboard admins can read all workers" ON jobs.workers;
DROP POLICY IF EXISTS "Dashboard admins can read all function files" ON jobs.function_files;
DROP POLICY IF EXISTS "Dashboard admins can read all execution logs" ON jobs.execution_logs;
DROP POLICY IF EXISTS "Users can read their own job logs" ON jobs.execution_logs;
DROP POLICY IF EXISTS "Service role can manage execution logs" ON jobs.execution_logs;

-- Remove from registry
DELETE FROM realtime.schema_registry
WHERE schema_name = 'jobs';

-- Drop triggers
DROP TRIGGER IF EXISTS queue_realtime_notify ON jobs.queue;
-- DROP TRIGGER IF EXISTS functions_realtime_notify ON jobs.functions; -- Never created
DROP TRIGGER IF EXISTS workers_realtime_notify ON jobs.workers;
DROP TRIGGER IF EXISTS function_files_realtime_notify ON jobs.function_files;
DROP TRIGGER IF EXISTS execution_logs_realtime_notify ON jobs.execution_logs;

-- Drop function
DROP FUNCTION IF EXISTS jobs.notify_realtime_change();

-- Drop execution_logs table
DROP TABLE IF EXISTS jobs.execution_logs;

-- Reset replica identity
ALTER TABLE jobs.queue REPLICA IDENTITY DEFAULT;
ALTER TABLE jobs.functions REPLICA IDENTITY DEFAULT;
ALTER TABLE jobs.workers REPLICA IDENTITY DEFAULT;
ALTER TABLE jobs.function_files REPLICA IDENTITY DEFAULT;
