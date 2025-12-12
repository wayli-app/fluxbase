-- Add failed login tracking to auth.users for brute-force protection
-- This mirrors the functionality already present in dashboard.users

ALTER TABLE auth.users
ADD COLUMN IF NOT EXISTS failed_login_attempts INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS is_locked BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS locked_until TIMESTAMP WITH TIME ZONE;

-- Add index for efficient lookup of locked accounts
CREATE INDEX IF NOT EXISTS idx_auth_users_is_locked ON auth.users(is_locked) WHERE is_locked = true;

COMMENT ON COLUMN auth.users.failed_login_attempts IS 'Number of consecutive failed login attempts';
COMMENT ON COLUMN auth.users.is_locked IS 'Whether the account is locked due to too many failed attempts';
COMMENT ON COLUMN auth.users.locked_until IS 'When the account lock expires (null = permanent until admin unlocks)';
