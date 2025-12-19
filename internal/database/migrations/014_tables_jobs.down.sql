-- Drop jobs tables and related objects
-- Drop realtime triggers first
DROP TRIGGER IF EXISTS function_files_realtime_notify ON jobs.function_files;
DROP TRIGGER IF EXISTS workers_realtime_notify ON jobs.workers;
DROP TRIGGER IF EXISTS queue_realtime_notify ON jobs.queue;
DROP FUNCTION IF EXISTS jobs.notify_realtime_change();

DROP TRIGGER IF EXISTS update_functions_updated_at ON jobs.functions;
DROP FUNCTION IF EXISTS jobs.update_updated_at_column();

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS jobs.function_files;
DROP TABLE IF EXISTS jobs.queue;
DROP TABLE IF EXISTS jobs.workers;
DROP TABLE IF EXISTS jobs.functions;
