-- Increase totp_secret column size to accommodate encrypted secrets
-- Encrypted secrets using AES-256-GCM with base64 encoding are ~60+ characters
ALTER TABLE auth.users ALTER COLUMN totp_secret TYPE VARCHAR(255);
