-- Remove permissions columns from jobs tables
ALTER TABLE jobs.queue
DROP COLUMN IF EXISTS user_name,
DROP COLUMN IF EXISTS user_email,
DROP COLUMN IF EXISTS user_role;

ALTER TABLE jobs.functions
DROP COLUMN IF EXISTS require_role;
