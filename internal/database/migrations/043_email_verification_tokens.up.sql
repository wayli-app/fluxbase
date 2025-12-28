-- Email verification tokens for auth users
-- Follows the same security pattern as magic_links (token hashing)
CREATE TABLE IF NOT EXISTS auth.email_verification_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE NOT NULL,
    token_hash TEXT UNIQUE NOT NULL,  -- SHA-256 hash of token (security: plaintext never stored)
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN DEFAULT false,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for token lookup (used when verifying)
CREATE INDEX IF NOT EXISTS idx_auth_email_verification_tokens_hash ON auth.email_verification_tokens(token_hash);

-- Index for user lookup (used when deleting old tokens for a user)
CREATE INDEX IF NOT EXISTS idx_auth_email_verification_tokens_user_id ON auth.email_verification_tokens(user_id);

-- Enable RLS - only service role can access this table
ALTER TABLE auth.email_verification_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.email_verification_tokens FORCE ROW LEVEL SECURITY;

-- Only service role can access email verification tokens
CREATE POLICY email_verification_tokens_service_only ON auth.email_verification_tokens
    FOR ALL
    USING (auth.current_user_role() = 'service_role')
    WITH CHECK (auth.current_user_role() = 'service_role');
