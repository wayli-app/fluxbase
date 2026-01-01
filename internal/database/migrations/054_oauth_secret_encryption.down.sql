-- Remove is_encrypted column from oauth_providers
ALTER TABLE dashboard.oauth_providers
DROP COLUMN IF EXISTS is_encrypted;
