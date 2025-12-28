-- Drop email verification tokens table
DROP POLICY IF EXISTS email_verification_tokens_service_only ON auth.email_verification_tokens;
DROP INDEX IF EXISTS idx_auth_email_verification_tokens_hash;
DROP INDEX IF EXISTS idx_auth_email_verification_tokens_user_id;
DROP TABLE IF EXISTS auth.email_verification_tokens;
