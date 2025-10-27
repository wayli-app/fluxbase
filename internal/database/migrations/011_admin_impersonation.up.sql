-- Migration: Admin Impersonation
-- Allows admins to impersonate users for support and debugging

-- Create impersonation audit log table
CREATE TABLE IF NOT EXISTS auth.impersonation_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    target_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    reason TEXT,
    started_at TIMESTAMPTZ DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    ip_address TEXT,
    user_agent TEXT,
    is_active BOOLEAN DEFAULT TRUE
);

-- Create indexes
CREATE INDEX idx_impersonation_admin ON auth.impersonation_sessions(admin_user_id);
CREATE INDEX idx_impersonation_target ON auth.impersonation_sessions(target_user_id);
CREATE INDEX idx_impersonation_active ON auth.impersonation_sessions(is_active) WHERE is_active = TRUE;
CREATE INDEX idx_impersonation_started ON auth.impersonation_sessions(started_at DESC);

-- Add constraint: can't impersonate yourself
ALTER TABLE auth.impersonation_sessions
    ADD CONSTRAINT check_no_self_impersonation
    CHECK (admin_user_id != target_user_id);

COMMENT ON TABLE auth.impersonation_sessions IS
'Audit log of admin impersonation sessions for compliance and security tracking';

COMMENT ON COLUMN auth.impersonation_sessions.admin_user_id IS
'The admin user who is performing the impersonation';

COMMENT ON COLUMN auth.impersonation_sessions.target_user_id IS
'The user being impersonated';

COMMENT ON COLUMN auth.impersonation_sessions.reason IS
'Business reason for impersonation (e.g., "Customer support ticket #1234")';

-- Grant permissions
