-- Table to store OAuth state tokens for CSRF protection
-- Enables OAuth flows in multi-instance deployments where callback may hit different instance
CREATE TABLE IF NOT EXISTS auth.oauth_states (
    -- The state token (random string)
    state TEXT PRIMARY KEY,

    -- OAuth provider name (e.g., "google", "github", "apple")
    provider TEXT NOT NULL,

    -- Optional custom redirect URI for this OAuth flow
    redirect_uri TEXT,

    -- PKCE code verifier (for providers that support PKCE)
    code_verifier TEXT,

    -- Optional nonce for OpenID Connect
    nonce TEXT,

    -- When the state was created
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- When the state expires (default: 10 minutes)
    expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '10 minutes'
);

-- Index for cleanup of expired states
CREATE INDEX IF NOT EXISTS idx_oauth_states_expires_at
    ON auth.oauth_states(expires_at);

-- Index for provider-specific queries
CREATE INDEX IF NOT EXISTS idx_oauth_states_provider
    ON auth.oauth_states(provider);

-- Add comments
COMMENT ON TABLE auth.oauth_states IS 'OAuth state tokens for CSRF protection in multi-instance deployments';
COMMENT ON COLUMN auth.oauth_states.state IS 'Random state token for CSRF protection';
COMMENT ON COLUMN auth.oauth_states.provider IS 'OAuth provider name';
COMMENT ON COLUMN auth.oauth_states.redirect_uri IS 'Custom redirect URI for this flow';
COMMENT ON COLUMN auth.oauth_states.code_verifier IS 'PKCE code verifier for enhanced security';
