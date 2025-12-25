-- Migration: Hash magic link and password reset tokens
-- SECURITY: Tokens are now stored as SHA-256 hashes instead of plaintext.
-- This prevents token exposure if the database is breached.

-- ============================================================================
-- MAGIC LINKS TABLE
-- ============================================================================

-- Rename the token column to token_hash
ALTER TABLE auth.magic_links RENAME COLUMN token TO token_hash;

-- Delete all existing magic links (they contain plaintext tokens which cannot be hashed)
-- Users will need to request new magic links after this migration
DELETE FROM auth.magic_links;

-- Update the index to use the new column name
DROP INDEX IF EXISTS auth.idx_auth_magic_links_token;
CREATE INDEX idx_auth_magic_links_token_hash ON auth.magic_links(token_hash);

-- ============================================================================
-- PASSWORD RESET TOKENS TABLE
-- ============================================================================

-- Rename the token column to token_hash
ALTER TABLE auth.password_reset_tokens RENAME COLUMN token TO token_hash;

-- Delete all existing password reset tokens (they contain plaintext tokens which cannot be hashed)
-- Users will need to request new password reset tokens after this migration
DELETE FROM auth.password_reset_tokens;

-- Update the index to use the new column name
DROP INDEX IF EXISTS auth.idx_auth_password_reset_tokens_token;
CREATE INDEX idx_auth_password_reset_tokens_token_hash ON auth.password_reset_tokens(token_hash);

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON COLUMN auth.magic_links.token_hash IS 'SHA-256 hash of the magic link token (base64 encoded). Plaintext token is never stored.';
COMMENT ON COLUMN auth.password_reset_tokens.token_hash IS 'SHA-256 hash of the password reset token (base64 encoded). Plaintext token is never stored.';
