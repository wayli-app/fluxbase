-- Migration: OAuth User Linking and Token Storage
-- This migration creates tables for OAuth authentication

-- OAuth user linking table
CREATE TABLE IF NOT EXISTS auth.oauth_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(provider, provider_user_id),
    CONSTRAINT fk_oauth_links_user FOREIGN KEY (user_id) REFERENCES auth.users(id) ON DELETE CASCADE
);

CREATE INDEX idx_oauth_links_user ON auth.oauth_links(user_id);
CREATE INDEX idx_oauth_links_provider ON auth.oauth_links(provider, provider_user_id);

-- OAuth tokens storage
CREATE TABLE IF NOT EXISTS auth.oauth_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    token_expiry TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(user_id, provider),
    CONSTRAINT fk_oauth_tokens_user FOREIGN KEY (user_id) REFERENCES auth.users(id) ON DELETE CASCADE
);

CREATE INDEX idx_oauth_tokens_user ON auth.oauth_tokens(user_id);
CREATE INDEX idx_oauth_tokens_provider ON auth.oauth_tokens(user_id, provider);

-- Add updated_at trigger for oauth_links
CREATE TRIGGER update_oauth_links_updated_at
    BEFORE UPDATE ON auth.oauth_links
    FOR EACH ROW
    EXECUTE FUNCTION public.update_updated_at();

-- Add updated_at trigger for oauth_tokens
CREATE TRIGGER update_oauth_tokens_updated_at
    BEFORE UPDATE ON auth.oauth_tokens
    FOR EACH ROW
    EXECUTE FUNCTION public.update_updated_at();

-- RLS policies for OAuth tables
ALTER TABLE auth.oauth_links ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.oauth_tokens ENABLE ROW LEVEL SECURITY;

-- Users can only see their own OAuth links
CREATE POLICY oauth_links_select ON auth.oauth_links
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Users can only see their own OAuth tokens
CREATE POLICY oauth_tokens_select ON auth.oauth_tokens
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Service role can access all OAuth data
CREATE POLICY oauth_links_service_all ON auth.oauth_links
    FOR ALL
    USING (auth.current_user_role() = 'service_role');

CREATE POLICY oauth_tokens_service_all ON auth.oauth_tokens
    FOR ALL
    USING (auth.current_user_role() = 'service_role');
