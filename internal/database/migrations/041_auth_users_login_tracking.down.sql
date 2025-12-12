-- Remove failed login tracking from auth.users

DROP INDEX IF EXISTS idx_auth_users_is_locked;

ALTER TABLE auth.users
DROP COLUMN IF EXISTS failed_login_attempts,
DROP COLUMN IF EXISTS is_locked,
DROP COLUMN IF EXISTS locked_until;
