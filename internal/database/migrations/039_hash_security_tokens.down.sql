-- Rollback: Revert token hashing for magic links and password reset tokens
-- WARNING: This does not restore any deleted tokens. All tokens will remain deleted.

-- ============================================================================
-- MAGIC LINKS TABLE
-- ============================================================================

-- Drop the new index
DROP INDEX IF EXISTS auth.idx_auth_magic_links_token_hash;

-- Rename the column back to token
ALTER TABLE auth.magic_links RENAME COLUMN token_hash TO token;

-- Recreate the original index
CREATE INDEX idx_auth_magic_links_token ON auth.magic_links(token);

-- ============================================================================
-- PASSWORD RESET TOKENS TABLE
-- ============================================================================

-- Drop the new index
DROP INDEX IF EXISTS auth.idx_auth_password_reset_tokens_token_hash;

-- Rename the column back to token
ALTER TABLE auth.password_reset_tokens RENAME COLUMN token_hash TO token;

-- Recreate the original index
CREATE INDEX idx_auth_password_reset_tokens_token ON auth.password_reset_tokens(token);

-- Remove comments
COMMENT ON COLUMN auth.magic_links.token IS NULL;
COMMENT ON COLUMN auth.password_reset_tokens.token IS NULL;
