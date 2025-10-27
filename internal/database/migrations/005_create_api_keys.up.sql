-- Create auth.api_keys table
CREATE TABLE IF NOT EXISTS auth.api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    key_hash TEXT UNIQUE NOT NULL,
    key_prefix TEXT NOT NULL, -- First 8 chars for identification (e.g., "fbk_12345...")
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    scopes TEXT[] DEFAULT ARRAY['read:tables', 'write:tables', 'read:storage', 'write:storage', 'read:functions', 'execute:functions'],
    rate_limit_per_minute INTEGER DEFAULT 100,
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for API keys
CREATE INDEX IF NOT EXISTS idx_auth_api_keys_key_hash ON auth.api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_auth_api_keys_user_id ON auth.api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_auth_api_keys_key_prefix ON auth.api_keys(key_prefix);
CREATE INDEX IF NOT EXISTS idx_auth_api_keys_expires_at ON auth.api_keys(expires_at);

-- Apply update trigger to API keys table
CREATE TRIGGER update_auth_api_keys_updated_at BEFORE UPDATE ON auth.api_keys
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create API key usage logs table
CREATE TABLE IF NOT EXISTS auth.api_key_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    api_key_id UUID REFERENCES auth.api_keys(id) ON DELETE CASCADE,
    endpoint TEXT NOT NULL,
    method TEXT NOT NULL,
    status_code INTEGER,
    request_duration_ms INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for API key usage
CREATE INDEX IF NOT EXISTS idx_auth_api_key_usage_api_key_id ON auth.api_key_usage(api_key_id);
CREATE INDEX IF NOT EXISTS idx_auth_api_key_usage_created_at ON auth.api_key_usage(created_at DESC);
