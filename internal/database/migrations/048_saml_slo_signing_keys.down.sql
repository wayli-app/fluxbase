-- Remove SP signing key columns from saml_providers

ALTER TABLE auth.saml_providers
DROP COLUMN IF EXISTS sp_certificate,
DROP COLUMN IF EXISTS sp_private_key_encrypted,
DROP COLUMN IF EXISTS idp_slo_url;
