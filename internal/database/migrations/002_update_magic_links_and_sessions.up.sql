-- Update magic_links table to use used_at instead of used boolean
ALTER TABLE auth.magic_links DROP COLUMN IF EXISTS used;
ALTER TABLE auth.magic_links ADD COLUMN IF NOT EXISTS used_at TIMESTAMPTZ;

-- Update sessions table columns to match the Session struct
-- The current schema has token/refresh_token, but code expects access_token/refresh_token
ALTER TABLE auth.sessions RENAME COLUMN token TO access_token;

-- Add index for used_at for efficient cleanup queries
CREATE INDEX IF NOT EXISTS idx_auth_magic_links_expires_at ON auth.magic_links(expires_at);

-- Add index for access_token on sessions
DROP INDEX IF EXISTS idx_auth_sessions_token;
CREATE INDEX IF NOT EXISTS idx_auth_sessions_access_token ON auth.sessions(access_token);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_refresh_token ON auth.sessions(refresh_token);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_expires_at ON auth.sessions(expires_at);

-- Update users table to match User struct (password_hash instead of encrypted_password)
ALTER TABLE auth.users RENAME COLUMN encrypted_password TO password_hash;
ALTER TABLE auth.users RENAME COLUMN email_confirmed TO email_verified;

-- Add role and metadata columns to users table
ALTER TABLE auth.users ADD COLUMN IF NOT EXISTS role TEXT DEFAULT 'user';
ALTER TABLE auth.users ADD COLUMN IF NOT EXISTS metadata JSONB;

-- Add index for user role
CREATE INDEX IF NOT EXISTS idx_auth_users_role ON auth.users(role);
