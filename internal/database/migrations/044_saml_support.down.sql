-- Rollback SAML SSO Support

-- Drop RLS policies
DROP POLICY IF EXISTS "Service role can manage saml_providers" ON auth.saml_providers;
DROP POLICY IF EXISTS "Service role can manage saml_sessions" ON auth.saml_sessions;
DROP POLICY IF EXISTS "Service role can manage saml_assertion_ids" ON auth.saml_assertion_ids;

-- Drop trigger
DROP TRIGGER IF EXISTS trigger_update_saml_providers_updated_at ON auth.saml_providers;
DROP FUNCTION IF EXISTS auth.update_saml_providers_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS auth.idx_saml_assertion_ids_expires;
DROP INDEX IF EXISTS auth.idx_identities_saml_name_id;
DROP INDEX IF EXISTS auth.idx_saml_sessions_provider_name;
DROP INDEX IF EXISTS auth.idx_saml_sessions_name_id;
DROP INDEX IF EXISTS auth.idx_saml_sessions_user_id;

-- Drop tables
DROP TABLE IF EXISTS auth.saml_assertion_ids;
DROP TABLE IF EXISTS auth.saml_sessions;
DROP TABLE IF EXISTS auth.saml_providers;

-- Remove SAML columns from identities (optional - uncomment if needed)
-- ALTER TABLE auth.identities DROP COLUMN IF EXISTS saml_name_id;
-- ALTER TABLE auth.identities DROP COLUMN IF EXISTS saml_attributes;
