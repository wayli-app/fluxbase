-- Rollback SSO Dashboard Support

-- Drop dashboard SSO identities table
DROP TABLE IF EXISTS dashboard.sso_identities;

-- Drop the trigger function
DROP FUNCTION IF EXISTS dashboard.update_sso_identities_updated_at();

-- Remove columns from OAuth providers
ALTER TABLE dashboard.oauth_providers DROP COLUMN IF EXISTS allow_dashboard_login;
ALTER TABLE dashboard.oauth_providers DROP COLUMN IF EXISTS allow_app_login;

-- Remove columns from SAML providers
ALTER TABLE auth.saml_providers DROP COLUMN IF EXISTS allow_dashboard_login;
ALTER TABLE auth.saml_providers DROP COLUMN IF EXISTS allow_app_login;
ALTER TABLE auth.saml_providers DROP COLUMN IF EXISTS allow_idp_initiated;
ALTER TABLE auth.saml_providers DROP COLUMN IF EXISTS allowed_redirect_hosts;
ALTER TABLE auth.saml_providers DROP COLUMN IF EXISTS source;
ALTER TABLE auth.saml_providers DROP COLUMN IF EXISTS display_name;

-- Remove dashboard admin policy on SAML providers
DROP POLICY IF EXISTS "Dashboard admin can manage saml_providers" ON auth.saml_providers;
