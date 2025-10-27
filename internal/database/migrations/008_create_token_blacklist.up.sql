-- Create auth.token_blacklist table for revoked/invalidated tokens
CREATE TABLE IF NOT EXISTS auth.token_blacklist (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_jti TEXT UNIQUE NOT NULL,  -- JWT ID (jti claim from token)
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    reason TEXT,  -- logout, compromised, admin_revoke, etc.
    revoked_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL  -- When token would naturally expire (for cleanup)
);

-- Create indexes for fast lookups
CREATE INDEX IF NOT EXISTS idx_token_blacklist_jti ON auth.token_blacklist(token_jti);
CREATE INDEX IF NOT EXISTS idx_token_blacklist_user_id ON auth.token_blacklist(user_id);
CREATE INDEX IF NOT EXISTS idx_token_blacklist_expires_at ON auth.token_blacklist(expires_at);

-- Add comment
COMMENT ON TABLE auth.token_blacklist IS 'Stores revoked JWT tokens to prevent their reuse';
COMMENT ON COLUMN auth.token_blacklist.token_jti IS 'JWT ID from the jti claim - unique identifier for the token';
COMMENT ON COLUMN auth.token_blacklist.reason IS 'Reason for revocation: logout, compromised, admin_revoke, security_incident, etc.';
