-- Add require_role column to functions for role-based access control
ALTER TABLE jobs.functions
ADD COLUMN require_role TEXT;

COMMENT ON COLUMN jobs.functions.require_role IS 'Required role to submit this job (admin, authenticated, anon, or null for any)';

-- Add user_role to queue to track submitter's role at time of submission
ALTER TABLE jobs.queue
ADD COLUMN user_role TEXT;

COMMENT ON COLUMN jobs.queue.user_role IS 'Role of the user who submitted the job';

-- Add user_email to queue for context
ALTER TABLE jobs.queue
ADD COLUMN user_email TEXT;

COMMENT ON COLUMN jobs.queue.user_email IS 'Email of the user who submitted the job';

-- Add user_name to queue for display
ALTER TABLE jobs.queue
ADD COLUMN user_name TEXT;

COMMENT ON COLUMN jobs.queue.user_name IS 'Display name of the user who submitted the job';
