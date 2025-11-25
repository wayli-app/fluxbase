-- Drop jobs tables
DROP TRIGGER IF EXISTS update_job_functions_updated_at ON jobs.job_functions;
DROP FUNCTION IF EXISTS jobs.update_updated_at_column();

DROP TABLE IF EXISTS jobs.job_function_files;
DROP TABLE IF EXISTS jobs.job_queue;
DROP TABLE IF EXISTS jobs.workers;
DROP TABLE IF EXISTS jobs.job_functions;
