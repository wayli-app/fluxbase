-- Add RBAC (Role-Based Access Control) columns to SSO provider tables
-- This migration adds support for role/group-based filtering during authentication

-- SAML Providers: Add group-based access control
ALTER TABLE auth.saml_providers
    ADD COLUMN IF NOT EXISTS required_groups TEXT[] DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS required_groups_all TEXT[] DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS denied_groups TEXT[] DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS group_attribute TEXT DEFAULT 'groups';

COMMENT ON COLUMN auth.saml_providers.required_groups IS 'User must be member of at least ONE of these groups (OR logic)';
COMMENT ON COLUMN auth.saml_providers.required_groups_all IS 'User must be member of ALL of these groups (AND logic)';
COMMENT ON COLUMN auth.saml_providers.denied_groups IS 'Reject users who are members of any of these groups';
COMMENT ON COLUMN auth.saml_providers.group_attribute IS 'SAML attribute name containing group memberships (default: groups)';

-- OAuth Providers: Add claims-based access control
ALTER TABLE dashboard.oauth_providers
    ADD COLUMN IF NOT EXISTS required_claims JSONB DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS denied_claims JSONB DEFAULT NULL;

COMMENT ON COLUMN dashboard.oauth_providers.required_claims IS 'JSON object of claims that must be present in ID token. Format: {"claim_name": ["value1", "value2"]}';
COMMENT ON COLUMN dashboard.oauth_providers.denied_claims IS 'JSON object of claims that, if present, will deny access. Format: {"claim_name": ["value1", "value2"]}';

-- Create index for JSONB columns for better query performance
CREATE INDEX IF NOT EXISTS idx_oauth_providers_required_claims ON dashboard.oauth_providers USING GIN (required_claims);
CREATE INDEX IF NOT EXISTS idx_oauth_providers_denied_claims ON dashboard.oauth_providers USING GIN (denied_claims);
