-- Revert users table changes
DROP INDEX IF EXISTS idx_auth_users_role;
ALTER TABLE auth.users DROP COLUMN IF EXISTS metadata;
ALTER TABLE auth.users DROP COLUMN IF EXISTS role;
ALTER TABLE auth.users RENAME COLUMN email_verified TO email_confirmed;
ALTER TABLE auth.users RENAME COLUMN password_hash TO encrypted_password;

-- Revert sessions table indexes
DROP INDEX IF EXISTS idx_auth_sessions_expires_at;
DROP INDEX IF EXISTS idx_auth_sessions_refresh_token;
DROP INDEX IF EXISTS idx_auth_sessions_access_token;
CREATE INDEX IF NOT EXISTS idx_auth_sessions_token ON auth.sessions(access_token);

-- Revert sessions table column name
ALTER TABLE auth.sessions RENAME COLUMN access_token TO token;

-- Revert magic_links table changes
DROP INDEX IF EXISTS idx_auth_magic_links_expires_at;
ALTER TABLE auth.magic_links DROP COLUMN IF EXISTS used_at;
ALTER TABLE auth.magic_links ADD COLUMN IF NOT EXISTS used BOOLEAN DEFAULT false;
