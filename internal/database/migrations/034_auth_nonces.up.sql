-- Migration: 034_auth_nonces
-- Description: Add nonces table for distributed reauthentication flows
-- This enables stateless multi-instance deployments without sticky sessions

CREATE TABLE IF NOT EXISTS auth.nonces (
    nonce TEXT PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for cleanup queries (deleting expired nonces)
CREATE INDEX IF NOT EXISTS idx_auth_nonces_expires_at ON auth.nonces(expires_at);

-- Index for user lookups (when user is deleted, CASCADE handles cleanup)
CREATE INDEX IF NOT EXISTS idx_auth_nonces_user_id ON auth.nonces(user_id);

-- Comment for documentation
COMMENT ON TABLE auth.nonces IS 'Single-use nonces for reauthentication flows. Enables stateless multi-instance deployments.';
