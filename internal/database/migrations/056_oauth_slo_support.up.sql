--
-- OAuth Single Logout (SLO) Support
-- Adds support for OAuth token revocation and OIDC RP-Initiated Logout
--

-- Add logout endpoints to OAuth providers
ALTER TABLE dashboard.oauth_providers
ADD COLUMN IF NOT EXISTS revocation_endpoint TEXT;

ALTER TABLE dashboard.oauth_providers
ADD COLUMN IF NOT EXISTS end_session_endpoint TEXT;

COMMENT ON COLUMN dashboard.oauth_providers.revocation_endpoint IS 'OAuth 2.0 Token Revocation endpoint (RFC 7009) for revoking access/refresh tokens';
COMMENT ON COLUMN dashboard.oauth_providers.end_session_endpoint IS 'OIDC RP-Initiated Logout endpoint for redirecting user to IdP logout page';

-- Store ID token for OIDC logout (needed for id_token_hint parameter)
ALTER TABLE auth.oauth_tokens
ADD COLUMN IF NOT EXISTS id_token TEXT;

COMMENT ON COLUMN auth.oauth_tokens.id_token IS 'OIDC ID token stored for use with end_session_endpoint id_token_hint parameter';

-- OAuth logout state tracking (CSRF protection for logout callback)
CREATE TABLE IF NOT EXISTS auth.oauth_logout_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    state TEXT UNIQUE NOT NULL,
    post_logout_redirect_uri TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ NOT NULL DEFAULT (CURRENT_TIMESTAMP + INTERVAL '10 minutes')
);

CREATE INDEX IF NOT EXISTS idx_oauth_logout_states_state ON auth.oauth_logout_states(state);
CREATE INDEX IF NOT EXISTS idx_oauth_logout_states_expires_at ON auth.oauth_logout_states(expires_at);

COMMENT ON TABLE auth.oauth_logout_states IS 'Temporary storage for OAuth logout states to track SP-initiated logout flow and provide CSRF protection';

-- Enable RLS on oauth_logout_states
ALTER TABLE auth.oauth_logout_states ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.oauth_logout_states FORCE ROW LEVEL SECURITY;

-- Service role can manage logout states
CREATE POLICY oauth_logout_states_service_access ON auth.oauth_logout_states
    FOR ALL
    USING (auth.current_user_role() = 'service_role')
    WITH CHECK (auth.current_user_role() = 'service_role');

-- Grant permissions
GRANT SELECT, INSERT, DELETE ON auth.oauth_logout_states TO service_role;
