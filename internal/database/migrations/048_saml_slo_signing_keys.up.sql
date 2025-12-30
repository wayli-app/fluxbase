-- Add SP signing key columns to saml_providers for SAML SLO support
-- These are used to sign LogoutRequest and LogoutResponse messages

ALTER TABLE auth.saml_providers
ADD COLUMN IF NOT EXISTS sp_certificate TEXT,
ADD COLUMN IF NOT EXISTS sp_private_key_encrypted BYTEA;

-- Add comment explaining the columns
COMMENT ON COLUMN auth.saml_providers.sp_certificate IS 'PEM-encoded X.509 certificate for signing SAML messages (SLO)';
COMMENT ON COLUMN auth.saml_providers.sp_private_key_encrypted IS 'Encrypted PEM-encoded private key for signing SAML messages (SLO)';

-- Add IdP SLO URL column if not exists (for clarity, may already be extracted from metadata)
ALTER TABLE auth.saml_providers
ADD COLUMN IF NOT EXISTS idp_slo_url TEXT;

COMMENT ON COLUMN auth.saml_providers.idp_slo_url IS 'IdP Single Logout URL extracted from metadata';
