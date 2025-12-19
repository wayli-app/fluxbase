-- Rollback: Auth Security Migration
-- Reverses security enhancements to authentication system

-- ============================================================================
-- ROLLBACK SESSION TOKEN HASHING
-- WARNING: This will delete all sessions again since we can't recover plaintext from hashes
-- ============================================================================

-- Delete all sessions (hashes can't be converted back to plaintext)
DELETE FROM auth.sessions;

-- Drop hash constraints and indexes
ALTER TABLE auth.sessions DROP CONSTRAINT IF EXISTS auth_sessions_access_token_hash_unique;
DROP INDEX IF EXISTS idx_auth_sessions_refresh_token_hash_unique;
DROP INDEX IF EXISTS idx_auth_sessions_access_token_hash;
DROP INDEX IF EXISTS idx_auth_sessions_refresh_token_hash;

-- Re-add plaintext columns
ALTER TABLE auth.sessions ADD COLUMN IF NOT EXISTS access_token TEXT;
ALTER TABLE auth.sessions ADD COLUMN IF NOT EXISTS refresh_token TEXT;

-- Make access_token required
ALTER TABLE auth.sessions ALTER COLUMN access_token SET NOT NULL;

-- Add unique constraints
ALTER TABLE auth.sessions ADD CONSTRAINT auth_sessions_access_token_unique UNIQUE (access_token);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_auth_sessions_access_token ON auth.sessions(access_token);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_refresh_token ON auth.sessions(refresh_token);

-- Drop hash columns
ALTER TABLE auth.sessions DROP COLUMN IF EXISTS access_token_hash;
ALTER TABLE auth.sessions DROP COLUMN IF EXISTS refresh_token_hash;

-- ============================================================================
-- ROLLBACK NONCES
-- ============================================================================

DROP TABLE IF EXISTS auth.nonces;
