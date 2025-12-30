--
-- APP SCHEMA TABLES
-- Application-level configuration and settings
--

-- Application settings table
-- Stores all application-level configuration in a flexible key-value format
CREATE TABLE IF NOT EXISTS app.settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key TEXT UNIQUE NOT NULL,
    value JSONB NOT NULL,
    value_type TEXT NOT NULL DEFAULT 'string' CHECK (value_type IN ('string', 'number', 'boolean', 'json', 'array')),
    category TEXT NOT NULL DEFAULT 'custom' CHECK (category IN ('auth', 'system', 'storage', 'functions', 'realtime', 'custom')),
    description TEXT,
    is_public BOOLEAN DEFAULT false,
    is_secret BOOLEAN DEFAULT false,
    editable_by TEXT[] NOT NULL DEFAULT ARRAY['dashboard_admin']::TEXT[],
    metadata JSONB DEFAULT '{}'::JSONB,
    created_by UUID,
    updated_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_app_settings_key ON app.settings(key);
CREATE INDEX IF NOT EXISTS idx_app_settings_category ON app.settings(category);
CREATE INDEX IF NOT EXISTS idx_app_settings_is_public ON app.settings(is_public);
CREATE INDEX IF NOT EXISTS idx_app_settings_editable_by ON app.settings USING GIN(editable_by);

COMMENT ON TABLE app.settings IS 'Application-level configuration and settings with flexible key-value storage';
COMMENT ON COLUMN app.settings.key IS 'Unique setting key (e.g., "jwt_secret", "max_upload_size")';
COMMENT ON COLUMN app.settings.value IS 'Setting value stored as JSONB for flexibility';
COMMENT ON COLUMN app.settings.value_type IS 'Type hint for the value: string, number, boolean, json, or array';
COMMENT ON COLUMN app.settings.category IS 'Category of setting: auth, system, storage, functions, realtime, or custom';
COMMENT ON COLUMN app.settings.is_public IS 'Whether this setting can be read by public/anon users';
COMMENT ON COLUMN app.settings.is_secret IS 'Whether this setting contains sensitive data (e.g., client keys, secrets)';
COMMENT ON COLUMN app.settings.editable_by IS 'Array of roles that can edit this setting';
COMMENT ON COLUMN app.settings.metadata IS 'Additional metadata about the setting (validation rules, UI hints, etc.)';

-- Insert default feature flag settings
-- All features are enabled by default
INSERT INTO app.settings (key, value, value_type, category, description, is_public, is_secret, editable_by)
VALUES
    (
        'app.realtime.enabled',
        '{"value": true}'::JSONB,
        'boolean',
        'system',
        'Enable or disable realtime functionality (WebSocket connections, subscriptions)',
        false,
        false,
        ARRAY['admin', 'dashboard_admin']::TEXT[]
    ),
    (
        'app.storage.enabled',
        '{"value": true}'::JSONB,
        'boolean',
        'system',
        'Enable or disable storage functionality (file uploads, downloads, management)',
        false,
        false,
        ARRAY['admin', 'dashboard_admin']::TEXT[]
    ),
    (
        'app.functions.enabled',
        '{"value": true}'::JSONB,
        'boolean',
        'system',
        'Enable or disable edge functions (serverless function execution)',
        false,
        false,
        ARRAY['admin', 'dashboard_admin']::TEXT[]
    ),
    (
        'app.ai.enabled',
        '{"value": true}'::JSONB,
        'boolean',
        'system',
        'Enable AI service functionality',
        false,
        false,
        ARRAY['admin', 'dashboard_admin']::TEXT[]
    ),
    (
        'app.rpc.enabled',
        '{"value": true}'::JSONB,
        'boolean',
        'system',
        'Enable RPC procedure execution',
        false,
        false,
        ARRAY['admin', 'dashboard_admin']::TEXT[]
    ),
    (
        'app.jobs.enabled',
        '{"value": true}'::JSONB,
        'boolean',
        'system',
        'Enable background job processing',
        false,
        false,
        ARRAY['admin', 'dashboard_admin']::TEXT[]
    ),
    (
        'app.email.enabled',
        '{"value": true}'::JSONB,
        'boolean',
        'system',
        'Enable email service functionality',
        false,
        false,
        ARRAY['admin', 'dashboard_admin']::TEXT[]
    )
ON CONFLICT (key) DO NOTHING;
