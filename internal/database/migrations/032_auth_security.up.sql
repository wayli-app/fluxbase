-- Auth Security Migration
-- Adds security enhancements to authentication system
-- Note: auth schema is created in 002_schemas

-- ============================================================================
-- NONCES
-- Single-use nonces for reauthentication flows
-- Enables stateless multi-instance deployments without sticky sessions
-- ============================================================================

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

COMMENT ON TABLE auth.nonces IS 'Single-use nonces for reauthentication flows. Enables stateless multi-instance deployments.';

-- ============================================================================
-- SESSION TOKEN HASHING
-- SECURITY: Hash session tokens instead of storing plaintext
-- A database breach would expose all active session tokens if stored as plaintext
-- ============================================================================

-- Add hash columns
ALTER TABLE auth.sessions ADD COLUMN IF NOT EXISTS access_token_hash TEXT;
ALTER TABLE auth.sessions ADD COLUMN IF NOT EXISTS refresh_token_hash TEXT;

-- Create indexes on hash columns (for lookup performance)
CREATE INDEX IF NOT EXISTS idx_auth_sessions_access_token_hash ON auth.sessions(access_token_hash);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_refresh_token_hash ON auth.sessions(refresh_token_hash);

-- Delete all existing sessions (they have plaintext tokens, not hashes)
-- This forces all users to re-authenticate, which is the secure approach
-- after implementing token hashing
DELETE FROM auth.sessions;

-- Make hash columns required for new sessions
ALTER TABLE auth.sessions ALTER COLUMN access_token_hash SET NOT NULL;

-- Drop the old plaintext token columns and indexes
DROP INDEX IF EXISTS idx_auth_sessions_access_token;
DROP INDEX IF EXISTS idx_auth_sessions_refresh_token;
ALTER TABLE auth.sessions DROP COLUMN IF EXISTS access_token;
ALTER TABLE auth.sessions DROP COLUMN IF EXISTS refresh_token;

-- Add comments explaining the hashing
COMMENT ON COLUMN auth.sessions.access_token_hash IS 'SHA-256 hash of access token (base64 encoded). Plaintext token is never stored.';
COMMENT ON COLUMN auth.sessions.refresh_token_hash IS 'SHA-256 hash of refresh token (base64 encoded). Plaintext token is never stored.';

-- Add unique constraints on hashes
ALTER TABLE auth.sessions ADD CONSTRAINT auth_sessions_access_token_hash_unique UNIQUE (access_token_hash);
CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_sessions_refresh_token_hash_unique
    ON auth.sessions(refresh_token_hash) WHERE refresh_token_hash IS NOT NULL;
