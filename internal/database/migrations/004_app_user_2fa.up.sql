-- Migration: Add 2FA Support for App Users
-- This migration adds TOTP/2FA capabilities to regular application users (auth.users)

-- Add 2FA fields to auth.users table
ALTER TABLE auth.users
    ADD COLUMN IF NOT EXISTS totp_secret VARCHAR(32),
    ADD COLUMN IF NOT EXISTS totp_enabled BOOLEAN DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS backup_codes TEXT[];

-- Create index for 2FA enabled users (for faster lookups during login)
CREATE INDEX IF NOT EXISTS idx_auth_users_totp_enabled ON auth.users(totp_enabled) WHERE totp_enabled = TRUE;

-- Create 2FA setup tracking table
CREATE TABLE IF NOT EXISTS auth.two_factor_setups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    secret VARCHAR(32) NOT NULL,
    qr_code_url TEXT NOT NULL,
    verified BOOLEAN DEFAULT FALSE,
    expires_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP + INTERVAL '10 minutes',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_2fa_setup_user FOREIGN KEY (user_id) REFERENCES auth.users(id) ON DELETE CASCADE,
    CONSTRAINT two_factor_setups_user_id_key UNIQUE (user_id)
);

CREATE INDEX IF NOT EXISTS idx_2fa_setup_user ON auth.two_factor_setups(user_id);
CREATE INDEX IF NOT EXISTS idx_2fa_setup_expires ON auth.two_factor_setups(expires_at);

COMMENT ON TABLE auth.two_factor_setups IS 'Temporary storage for 2FA setup process. Entries expire after 10 minutes and should be cleaned up periodically.';

-- Create 2FA recovery/backup code usage tracking table
CREATE TABLE IF NOT EXISTS auth.two_factor_recovery_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    code_used VARCHAR(255),
    success BOOLEAN NOT NULL,
    ip_address INET,
    user_agent TEXT,
    attempted_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_2fa_recovery_user FOREIGN KEY (user_id) REFERENCES auth.users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_2fa_recovery_user ON auth.two_factor_recovery_attempts(user_id);
CREATE INDEX IF NOT EXISTS idx_2fa_recovery_time ON auth.two_factor_recovery_attempts(attempted_at);

COMMENT ON TABLE auth.two_factor_recovery_attempts IS 'Audit log for 2FA recovery/backup code usage attempts for security monitoring.';

-- Enable RLS on new tables
ALTER TABLE auth.two_factor_setups ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.two_factor_recovery_attempts ENABLE ROW LEVEL SECURITY;

-- RLS Policies for two_factor_setups
-- Users can only see their own 2FA setup records
CREATE POLICY two_factor_setups_select ON auth.two_factor_setups
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Users can insert their own 2FA setup records
CREATE POLICY two_factor_setups_insert ON auth.two_factor_setups
    FOR INSERT
    WITH CHECK (user_id = auth.current_user_id());

-- Users can delete their own 2FA setup records
CREATE POLICY two_factor_setups_delete ON auth.two_factor_setups
    FOR DELETE
    USING (user_id = auth.current_user_id());

-- Admins can see all 2FA setups
CREATE POLICY two_factor_setups_admin_select ON auth.two_factor_setups
    FOR SELECT
    USING (auth.is_admin() OR auth.current_user_role() = 'dashboard_admin');

-- RLS Policies for two_factor_recovery_attempts
-- Users can only see their own recovery attempts
CREATE POLICY two_factor_recovery_select ON auth.two_factor_recovery_attempts
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Admins can see all recovery attempts
CREATE POLICY two_factor_recovery_admin_select ON auth.two_factor_recovery_attempts
    FOR SELECT
    USING (auth.is_admin() OR auth.current_user_role() = 'dashboard_admin');

-- Grant necessary permissions
GRANT SELECT, INSERT, UPDATE, DELETE ON auth.two_factor_setups TO fluxbase_app;
GRANT SELECT, INSERT ON auth.two_factor_recovery_attempts TO fluxbase_app;
GRANT USAGE ON SCHEMA auth TO fluxbase_app;
