-- Add require_role column to job_functions for role-based access control
ALTER TABLE jobs.job_functions
ADD COLUMN require_role TEXT;

COMMENT ON COLUMN jobs.job_functions.require_role IS 'Required role to submit this job (admin, authenticated, anon, or null for any)';

-- Add user_role to job_queue to track submitter's role at time of submission
ALTER TABLE jobs.job_queue
ADD COLUMN user_role TEXT;

COMMENT ON COLUMN jobs.job_queue.user_role IS 'Role of the user who submitted the job';

-- Add user_email to job_queue for context
ALTER TABLE jobs.job_queue
ADD COLUMN user_email TEXT;

COMMENT ON COLUMN jobs.job_queue.user_email IS 'Email of the user who submitted the job';
