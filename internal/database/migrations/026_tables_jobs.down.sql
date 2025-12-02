-- Drop jobs tables
DROP TRIGGER IF EXISTS update_functions_updated_at ON jobs.functions;
DROP FUNCTION IF EXISTS jobs.update_updated_at_column();

DROP TABLE IF EXISTS jobs.function_files;
DROP TABLE IF EXISTS jobs.queue;
DROP TABLE IF EXISTS jobs.workers;
DROP TABLE IF EXISTS jobs.functions;
