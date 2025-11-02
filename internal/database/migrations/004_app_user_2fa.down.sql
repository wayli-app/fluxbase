-- Rollback: Remove 2FA Support for App Users

-- Drop RLS policies first
DROP POLICY IF EXISTS two_factor_recovery_admin_select ON auth.two_factor_recovery_attempts;
DROP POLICY IF EXISTS two_factor_recovery_select ON auth.two_factor_recovery_attempts;
DROP POLICY IF EXISTS two_factor_setups_admin_select ON auth.two_factor_setups;
DROP POLICY IF EXISTS two_factor_setups_delete ON auth.two_factor_setups;
DROP POLICY IF EXISTS two_factor_setups_insert ON auth.two_factor_setups;
DROP POLICY IF EXISTS two_factor_setups_select ON auth.two_factor_setups;

-- Drop tables
DROP TABLE IF EXISTS auth.two_factor_recovery_attempts;
DROP TABLE IF EXISTS auth.two_factor_setups;

-- Drop index
DROP INDEX IF EXISTS auth.idx_auth_users_totp_enabled;

-- Remove 2FA columns from auth.users
ALTER TABLE auth.users DROP COLUMN IF EXISTS backup_codes;
ALTER TABLE auth.users DROP COLUMN IF EXISTS totp_enabled;
ALTER TABLE auth.users DROP COLUMN IF EXISTS totp_secret;
