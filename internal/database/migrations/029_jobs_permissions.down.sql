-- Remove permissions columns from jobs tables
ALTER TABLE jobs.job_queue
DROP COLUMN IF EXISTS user_email,
DROP COLUMN IF EXISTS user_role;

ALTER TABLE jobs.job_functions
DROP COLUMN IF EXISTS require_role;
