-- Revert totp_secret column size (may lose data if encrypted values exist)
-- Note: This will fail if any values exceed 32 characters
ALTER TABLE auth.users ALTER COLUMN totp_secret TYPE VARCHAR(32);
