-- Rollback RBAC enhancements for SSO providers

-- Drop indexes
DROP INDEX IF EXISTS idx_oauth_providers_denied_claims;
DROP INDEX IF EXISTS idx_oauth_providers_required_claims;

-- Remove OAuth provider RBAC columns
ALTER TABLE dashboard.oauth_providers
    DROP COLUMN IF EXISTS denied_claims,
    DROP COLUMN IF EXISTS required_claims;

-- Remove SAML provider RBAC columns
ALTER TABLE auth.saml_providers
    DROP COLUMN IF EXISTS group_attribute,
    DROP COLUMN IF EXISTS denied_groups,
    DROP COLUMN IF EXISTS required_groups_all,
    DROP COLUMN IF EXISTS required_groups;
