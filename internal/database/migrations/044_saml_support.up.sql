-- SAML SSO Support
-- This migration adds tables for SAML 2.0 Service Provider (SP) support

-- SAML providers table stores IdP configuration
-- Note: Most SAML config comes from fluxbase.yaml, this stores runtime/database-managed providers
CREATE TABLE IF NOT EXISTS auth.saml_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT UNIQUE NOT NULL,
    enabled BOOLEAN DEFAULT true,
    idp_metadata_url TEXT,
    idp_metadata_xml TEXT,
    idp_metadata_cached TEXT,  -- Cached parsed metadata
    idp_metadata_cached_at TIMESTAMPTZ,
    entity_id TEXT NOT NULL,
    acs_url TEXT NOT NULL,
    certificate TEXT,  -- SP signing certificate (PEM format)
    private_key TEXT,  -- SP private key (encrypted with encryption_key)
    attribute_mapping JSONB DEFAULT '{"email": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress", "name": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name"}',
    auto_create_users BOOLEAN DEFAULT true,
    default_role TEXT DEFAULT 'authenticated',
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- SAML sessions track active SAML authentication sessions
-- Used for Single Logout (SLO) and session binding
CREATE TABLE IF NOT EXISTS auth.saml_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    provider_id UUID REFERENCES auth.saml_providers(id) ON DELETE SET NULL,
    provider_name TEXT NOT NULL,  -- Keep provider name even if provider is deleted
    name_id TEXT NOT NULL,  -- SAML NameID from IdP
    name_id_format TEXT,  -- NameID format
    session_index TEXT,  -- IdP session reference for SLO
    attributes JSONB,  -- Raw SAML attributes
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Index for user lookups
CREATE INDEX IF NOT EXISTS idx_saml_sessions_user_id ON auth.saml_sessions(user_id);

-- Index for session lookups by name_id (used during SLO)
CREATE INDEX IF NOT EXISTS idx_saml_sessions_name_id ON auth.saml_sessions(name_id);

-- Index for provider lookups
CREATE INDEX IF NOT EXISTS idx_saml_sessions_provider_name ON auth.saml_sessions(provider_name);

-- Extend auth.identities to support SAML identity linking
-- Add saml-specific columns if they don't exist
DO $$
BEGIN
    -- Add saml_name_id column if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'auth'
        AND table_name = 'identities'
        AND column_name = 'saml_name_id'
    ) THEN
        ALTER TABLE auth.identities ADD COLUMN saml_name_id TEXT;
    END IF;

    -- Add saml_attributes column if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'auth'
        AND table_name = 'identities'
        AND column_name = 'saml_attributes'
    ) THEN
        ALTER TABLE auth.identities ADD COLUMN saml_attributes JSONB;
    END IF;
END $$;

-- Create index for SAML name_id lookups
CREATE INDEX IF NOT EXISTS idx_identities_saml_name_id ON auth.identities(saml_name_id) WHERE saml_name_id IS NOT NULL;

-- SAML assertion replay prevention table
-- Stores assertion IDs to prevent replay attacks
CREATE TABLE IF NOT EXISTS auth.saml_assertion_ids (
    assertion_id TEXT PRIMARY KEY,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Index for cleanup of expired assertions
CREATE INDEX IF NOT EXISTS idx_saml_assertion_ids_expires ON auth.saml_assertion_ids(expires_at);

-- Updated_at trigger for saml_providers
CREATE OR REPLACE FUNCTION auth.update_saml_providers_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_update_saml_providers_updated_at ON auth.saml_providers;
CREATE TRIGGER trigger_update_saml_providers_updated_at
    BEFORE UPDATE ON auth.saml_providers
    FOR EACH ROW
    EXECUTE FUNCTION auth.update_saml_providers_updated_at();

-- RLS policies for SAML tables (admin access only)
ALTER TABLE auth.saml_providers ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.saml_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.saml_assertion_ids ENABLE ROW LEVEL SECURITY;

-- Service role can manage all SAML data
CREATE POLICY "Service role can manage saml_providers" ON auth.saml_providers
    FOR ALL
    TO service_role
    USING (true)
    WITH CHECK (true);

CREATE POLICY "Service role can manage saml_sessions" ON auth.saml_sessions
    FOR ALL
    TO service_role
    USING (true)
    WITH CHECK (true);

CREATE POLICY "Service role can manage saml_assertion_ids" ON auth.saml_assertion_ids
    FOR ALL
    TO service_role
    USING (true)
    WITH CHECK (true);

-- Grant permissions
GRANT SELECT, INSERT, UPDATE, DELETE ON auth.saml_providers TO service_role;
GRANT SELECT, INSERT, UPDATE, DELETE ON auth.saml_sessions TO service_role;
GRANT SELECT, INSERT, DELETE ON auth.saml_assertion_ids TO service_role;

COMMENT ON TABLE auth.saml_providers IS 'SAML 2.0 Identity Provider configurations for enterprise SSO';
COMMENT ON TABLE auth.saml_sessions IS 'Active SAML authentication sessions for Single Logout support';
COMMENT ON TABLE auth.saml_assertion_ids IS 'SAML assertion IDs for replay attack prevention';
