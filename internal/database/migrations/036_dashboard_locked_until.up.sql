-- Migration: Add locked_until column to dashboard.users for automatic unlock after lockout period
-- This enables temporary lockouts that expire after a specified duration (similar to auth.users)

-- Add locked_until column to dashboard.users table
ALTER TABLE dashboard.users
ADD COLUMN IF NOT EXISTS locked_until TIMESTAMPTZ;

-- Create index for efficient lookups of users whose lock has expired
CREATE INDEX IF NOT EXISTS idx_dashboard_users_locked_until ON dashboard.users(locked_until)
WHERE locked_until IS NOT NULL;

-- Add comment explaining the column
COMMENT ON COLUMN dashboard.users.locked_until IS 'Timestamp when the account lock expires. NULL means no lock or lock is permanent (based on is_locked). When locked_until has passed, the account should be automatically unlocked on next login attempt.';
