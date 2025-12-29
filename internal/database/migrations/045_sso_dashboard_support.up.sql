-- SSO Dashboard Support
-- Adds support for SAML/OAuth authentication for dashboard admins
-- and configurable provider targeting (app users vs dashboard admins)

-- Add allow_dashboard_login and allow_app_login columns to OAuth providers
ALTER TABLE dashboard.oauth_providers
ADD COLUMN IF NOT EXISTS allow_dashboard_login BOOLEAN DEFAULT false;

ALTER TABLE dashboard.oauth_providers
ADD COLUMN IF NOT EXISTS allow_app_login BOOLEAN DEFAULT true;

COMMENT ON COLUMN dashboard.oauth_providers.allow_dashboard_login IS 'Allow this provider for dashboard admin SSO login';
COMMENT ON COLUMN dashboard.oauth_providers.allow_app_login IS 'Allow this provider for application user authentication';

-- Add columns to SAML providers for dashboard login and additional config
ALTER TABLE auth.saml_providers
ADD COLUMN IF NOT EXISTS allow_dashboard_login BOOLEAN DEFAULT false;

ALTER TABLE auth.saml_providers
ADD COLUMN IF NOT EXISTS allow_app_login BOOLEAN DEFAULT true;

ALTER TABLE auth.saml_providers
ADD COLUMN IF NOT EXISTS allow_idp_initiated BOOLEAN DEFAULT false;

ALTER TABLE auth.saml_providers
ADD COLUMN IF NOT EXISTS allowed_redirect_hosts TEXT[] DEFAULT ARRAY[]::TEXT[];

ALTER TABLE auth.saml_providers
ADD COLUMN IF NOT EXISTS source TEXT DEFAULT 'database';

ALTER TABLE auth.saml_providers
ADD COLUMN IF NOT EXISTS display_name TEXT;

COMMENT ON COLUMN auth.saml_providers.allow_dashboard_login IS 'Allow this provider for dashboard admin SSO login';
COMMENT ON COLUMN auth.saml_providers.allow_app_login IS 'Allow this provider for application user authentication';
COMMENT ON COLUMN auth.saml_providers.allow_idp_initiated IS 'Allow IdP-initiated SSO (less secure)';
COMMENT ON COLUMN auth.saml_providers.allowed_redirect_hosts IS 'Whitelist of allowed hosts for RelayState redirects';
COMMENT ON COLUMN auth.saml_providers.source IS 'Provider source: database (UI-managed) or config (YAML file)';
COMMENT ON COLUMN auth.saml_providers.display_name IS 'Human-friendly display name for the provider';

-- Dashboard SSO identity linking table
-- Links dashboard users to their SSO identities (OAuth/SAML)
CREATE TABLE IF NOT EXISTS dashboard.sso_identities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES dashboard.users(id) ON DELETE CASCADE,
    provider_type TEXT NOT NULL CHECK (provider_type IN ('oauth', 'saml')),
    provider_name TEXT NOT NULL,
    provider_user_id TEXT NOT NULL,
    email TEXT,
    name TEXT,
    raw_attributes JSONB DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE(provider_type, provider_name, provider_user_id)
);

CREATE INDEX IF NOT EXISTS idx_dashboard_sso_identities_user_id
ON dashboard.sso_identities(user_id);

CREATE INDEX IF NOT EXISTS idx_dashboard_sso_identities_provider
ON dashboard.sso_identities(provider_type, provider_name);

COMMENT ON TABLE dashboard.sso_identities IS 'Links dashboard admin users to their SSO identities';

-- Updated_at trigger for sso_identities
CREATE OR REPLACE FUNCTION dashboard.update_sso_identities_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_update_sso_identities_updated_at ON dashboard.sso_identities;
CREATE TRIGGER trigger_update_sso_identities_updated_at
    BEFORE UPDATE ON dashboard.sso_identities
    FOR EACH ROW
    EXECUTE FUNCTION dashboard.update_sso_identities_updated_at();

-- RLS policies for dashboard.sso_identities
ALTER TABLE dashboard.sso_identities ENABLE ROW LEVEL SECURITY;
ALTER TABLE dashboard.sso_identities FORCE ROW LEVEL SECURITY;

-- Service role and dashboard admins can manage SSO identities
CREATE POLICY "SSO identities admin access" ON dashboard.sso_identities
    FOR ALL
    USING (
        auth.current_user_role() = 'service_role'
        OR auth.current_user_role() = 'dashboard_admin'
    );

-- Grant permissions
GRANT SELECT, INSERT, UPDATE, DELETE ON dashboard.sso_identities TO service_role;

-- Add RLS policy for dashboard_admin role on SAML providers
DROP POLICY IF EXISTS "Dashboard admin can manage saml_providers" ON auth.saml_providers;
CREATE POLICY "Dashboard admin can manage saml_providers" ON auth.saml_providers
    FOR ALL
    TO authenticated
    USING (auth.current_user_role() = 'dashboard_admin')
    WITH CHECK (auth.current_user_role() = 'dashboard_admin');
