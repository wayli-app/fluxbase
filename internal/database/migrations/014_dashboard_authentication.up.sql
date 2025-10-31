-- Migration: Dashboard Authentication Schema
-- Purpose: Separate dashboard admin users from application end-users
-- This enables proper access control where dashboard users manage the platform
-- but cannot be confused with application end-users

-- Create dashboard schema for admin/platform users
CREATE SCHEMA IF NOT EXISTS dashboard;

-- Dashboard users table (platform administrators)
CREATE TABLE dashboard.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT UNIQUE NOT NULL,
    email_verified BOOLEAN DEFAULT false,
    password_hash TEXT NOT NULL,

    -- Profile information (no bio - removed per requirements)
    full_name TEXT,
    avatar_url TEXT,

    -- Two-factor authentication
    totp_secret TEXT,
    totp_enabled BOOLEAN DEFAULT false,
    backup_codes TEXT[], -- Array of hashed backup codes

    -- Account status
    is_active BOOLEAN DEFAULT true,
    is_locked BOOLEAN DEFAULT false,
    failed_login_attempts INTEGER DEFAULT 0,
    last_failed_login_at TIMESTAMPTZ,

    -- Session management
    last_login_at TIMESTAMPTZ,
    last_activity_at TIMESTAMPTZ,

    -- Audit fields
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    deleted_at TIMESTAMPTZ -- Soft delete support
);

-- Email verification tokens
CREATE TABLE dashboard.email_verification_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES dashboard.users(id) ON DELETE CASCADE,
    token TEXT UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Password reset tokens
CREATE TABLE dashboard.password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES dashboard.users(id) ON DELETE CASCADE,
    token TEXT UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Dashboard sessions (separate from application sessions)
CREATE TABLE dashboard.sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES dashboard.users(id) ON DELETE CASCADE,
    token_hash TEXT UNIQUE NOT NULL, -- Hashed JWT jti claim
    ip_address INET,
    user_agent TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    last_activity_at TIMESTAMPTZ DEFAULT now()
);

-- Activity log for security auditing
CREATE TABLE dashboard.activity_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES dashboard.users(id) ON DELETE SET NULL,
    action TEXT NOT NULL, -- e.g., 'login', 'logout', 'password_change', '2fa_enable'
    resource_type TEXT, -- e.g., 'user', 'settings'
    resource_id TEXT,
    ip_address INET,
    user_agent TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Indexes for performance
CREATE INDEX idx_dashboard_users_email ON dashboard.users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_dashboard_users_active ON dashboard.users(is_active) WHERE deleted_at IS NULL;
CREATE INDEX idx_dashboard_sessions_user_id ON dashboard.sessions(user_id);
CREATE INDEX idx_dashboard_sessions_expires_at ON dashboard.sessions(expires_at);
CREATE INDEX idx_dashboard_activity_log_user_id ON dashboard.activity_log(user_id);
CREATE INDEX idx_dashboard_activity_log_created_at ON dashboard.activity_log(created_at DESC);
CREATE INDEX idx_email_verification_tokens_user_id ON dashboard.email_verification_tokens(user_id);
CREATE INDEX idx_password_reset_tokens_user_id ON dashboard.password_reset_tokens(user_id);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION dashboard.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-update updated_at
CREATE TRIGGER update_dashboard_users_updated_at
    BEFORE UPDATE ON dashboard.users
    FOR EACH ROW
    EXECUTE FUNCTION dashboard.update_updated_at_column();

-- Function to clean up expired tokens and sessions
CREATE OR REPLACE FUNCTION dashboard.cleanup_expired_tokens()
RETURNS void AS $$
BEGIN
    -- Delete expired email verification tokens
    DELETE FROM dashboard.email_verification_tokens
    WHERE expires_at < now() AND used_at IS NULL;

    -- Delete expired password reset tokens
    DELETE FROM dashboard.password_reset_tokens
    WHERE expires_at < now() AND used_at IS NULL;

    -- Delete expired sessions
    DELETE FROM dashboard.sessions
    WHERE expires_at < now();

    -- Delete old activity logs (keep 90 days)
    DELETE FROM dashboard.activity_log
    WHERE created_at < now() - INTERVAL '90 days';
END;
$$ LANGUAGE plpgsql;

-- Grant permissions (only if role exists)
-- The fluxbase_app role may not exist in all environments (e.g., CI tests)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'fluxbase_app') THEN
        GRANT USAGE ON SCHEMA dashboard TO fluxbase_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA dashboard TO fluxbase_app;
        GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA dashboard TO fluxbase_app;
    END IF;
END$$;

-- Comments for documentation
COMMENT ON SCHEMA dashboard IS 'Schema for dashboard/platform administrator authentication and management';
COMMENT ON TABLE dashboard.users IS 'Dashboard administrator users (separate from application end-users in auth.users)';
COMMENT ON TABLE dashboard.sessions IS 'Active dashboard sessions for security tracking';
COMMENT ON TABLE dashboard.activity_log IS 'Audit log for dashboard user actions';
COMMENT ON COLUMN dashboard.users.totp_secret IS 'Base32-encoded TOTP secret for 2FA';
COMMENT ON COLUMN dashboard.users.backup_codes IS 'Hashed backup codes for 2FA recovery';
