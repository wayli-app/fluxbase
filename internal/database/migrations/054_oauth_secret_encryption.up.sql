-- Add is_encrypted column to track whether client_secret is encrypted
-- This allows migration of existing plaintext secrets to encrypted format
ALTER TABLE dashboard.oauth_providers
ADD COLUMN IF NOT EXISTS is_encrypted BOOLEAN DEFAULT false;

-- Comment explaining the column
COMMENT ON COLUMN dashboard.oauth_providers.is_encrypted IS 'Indicates whether client_secret is encrypted at rest using AES-256-GCM';
