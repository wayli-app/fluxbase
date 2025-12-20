-- Rollback: Remove locked_until column from dashboard.users

-- Drop the index first
DROP INDEX IF EXISTS dashboard.idx_dashboard_users_locked_until;

-- Remove the locked_until column
ALTER TABLE dashboard.users
DROP COLUMN IF EXISTS locked_until;
