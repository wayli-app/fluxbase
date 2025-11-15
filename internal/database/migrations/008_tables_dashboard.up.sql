--
-- DASHBOARD SCHEMA TABLES
-- Platform administrator authentication and management
--

-- Dashboard users table (with split metadata support)
CREATE TABLE IF NOT EXISTS dashboard.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    full_name TEXT,
    avatar_url TEXT,
    role TEXT DEFAULT 'dashboard_user',
    user_metadata JSONB DEFAULT '{}'::JSONB,
    app_metadata JSONB DEFAULT '{}'::JSONB,
    email_verified BOOLEAN DEFAULT false,
    email_verified_at TIMESTAMPTZ,
    totp_enabled BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    is_locked BOOLEAN DEFAULT false,
    failed_login_attempts INTEGER DEFAULT 0,
    last_login_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dashboard_users_email ON dashboard.users(email);
CREATE INDEX IF NOT EXISTS idx_dashboard_users_role ON dashboard.users(role);
CREATE INDEX IF NOT EXISTS idx_dashboard_users_user_metadata ON dashboard.users USING GIN (user_metadata);
CREATE INDEX IF NOT EXISTS idx_dashboard_users_app_metadata ON dashboard.users USING GIN (app_metadata);

COMMENT ON COLUMN dashboard.users.user_metadata IS 'User-editable metadata for dashboard users.';
COMMENT ON COLUMN dashboard.users.app_metadata IS 'Application/admin-only metadata for dashboard users.';

-- Dashboard sessions table
CREATE TABLE IF NOT EXISTS dashboard.sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES dashboard.users(id) ON DELETE CASCADE NOT NULL,
    token TEXT UNIQUE NOT NULL,
    refresh_token TEXT UNIQUE,
    ip_address TEXT,
    user_agent TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dashboard_sessions_user_id ON dashboard.sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_dashboard_sessions_token ON dashboard.sessions(token);
CREATE INDEX IF NOT EXISTS idx_dashboard_sessions_refresh_token ON dashboard.sessions(refresh_token);

-- Dashboard password reset tokens
CREATE TABLE IF NOT EXISTS dashboard.password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES dashboard.users(id) ON DELETE CASCADE NOT NULL,
    token TEXT UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN DEFAULT false,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dashboard_password_reset_tokens_token ON dashboard.password_reset_tokens(token);

-- Dashboard email verification tokens
CREATE TABLE IF NOT EXISTS dashboard.email_verification_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES dashboard.users(id) ON DELETE CASCADE NOT NULL,
    token TEXT UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN DEFAULT false,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dashboard_email_verification_tokens_token ON dashboard.email_verification_tokens(token);

-- Dashboard activity log
CREATE TABLE IF NOT EXISTS dashboard.activity_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES dashboard.users(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    resource_type TEXT,
    resource_id TEXT,
    details JSONB DEFAULT '{}'::JSONB,
    ip_address TEXT,
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dashboard_activity_log_user_id ON dashboard.activity_log(user_id);
CREATE INDEX IF NOT EXISTS idx_dashboard_activity_log_created_at ON dashboard.activity_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_dashboard_activity_log_action ON dashboard.activity_log(action);

-- OAuth providers table (with updated schema)
CREATE TABLE IF NOT EXISTS dashboard.oauth_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_name TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL DEFAULT '',
    client_id TEXT NOT NULL,
    client_secret TEXT NOT NULL,
    redirect_url TEXT NOT NULL,
    scopes TEXT[] DEFAULT ARRAY[]::TEXT[],
    enabled BOOLEAN DEFAULT true,
    is_custom BOOLEAN DEFAULT FALSE,
    authorization_url TEXT,
    token_url TEXT,
    user_info_url TEXT,
    metadata JSONB DEFAULT '{}'::JSONB,
    created_by UUID,
    updated_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dashboard_oauth_providers_provider_name ON dashboard.oauth_providers(provider_name);
CREATE INDEX IF NOT EXISTS idx_dashboard_oauth_providers_enabled ON dashboard.oauth_providers(enabled);

-- Auth settings table
CREATE TABLE IF NOT EXISTS dashboard.auth_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key TEXT UNIQUE NOT NULL,
    value JSONB NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dashboard_auth_settings_key ON dashboard.auth_settings(key);

-- System settings table
CREATE TABLE IF NOT EXISTS dashboard.system_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key TEXT UNIQUE NOT NULL,
    value JSONB NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dashboard_system_settings_key ON dashboard.system_settings(key);

-- Custom settings table (flexible admin-managed key-value configuration)
CREATE TABLE IF NOT EXISTS dashboard.custom_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key TEXT UNIQUE NOT NULL,
    value JSONB NOT NULL,
    value_type TEXT NOT NULL DEFAULT 'string' CHECK (value_type IN ('string', 'number', 'boolean', 'json')),
    description TEXT,
    editable_by TEXT[] NOT NULL DEFAULT ARRAY['dashboard_admin']::TEXT[],
    metadata JSONB DEFAULT '{}'::JSONB,
    created_by UUID REFERENCES dashboard.users(id) ON DELETE SET NULL,
    updated_by UUID REFERENCES dashboard.users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dashboard_custom_settings_key ON dashboard.custom_settings(key);
CREATE INDEX IF NOT EXISTS idx_dashboard_custom_settings_editable_by ON dashboard.custom_settings USING GIN(editable_by);
CREATE INDEX IF NOT EXISTS idx_dashboard_custom_settings_created_at ON dashboard.custom_settings(created_at);

COMMENT ON TABLE dashboard.custom_settings IS 'Flexible key-value settings that can be created and managed by admins and dashboard_admins';

-- Invitation tokens table
CREATE TABLE IF NOT EXISTS dashboard.invitation_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL,
    token TEXT UNIQUE NOT NULL,
    role TEXT NOT NULL DEFAULT 'dashboard_user',
    invited_by UUID REFERENCES dashboard.users(id) ON DELETE SET NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    accepted BOOLEAN DEFAULT false,
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dashboard_invitation_tokens_token ON dashboard.invitation_tokens(token);
CREATE INDEX IF NOT EXISTS idx_dashboard_invitation_tokens_email ON dashboard.invitation_tokens(email);
CREATE INDEX IF NOT EXISTS idx_dashboard_invitation_tokens_expires_at ON dashboard.invitation_tokens(expires_at);
CREATE INDEX IF NOT EXISTS idx_dashboard_invitation_tokens_accepted ON dashboard.invitation_tokens(accepted);

-- Schema migrations tracking
CREATE TABLE IF NOT EXISTS dashboard.schema_migrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schema_name TEXT NOT NULL,
    migration_type TEXT NOT NULL,
    migration_sql TEXT NOT NULL,
    applied_by UUID REFERENCES dashboard.users(id) ON DELETE SET NULL,
    applied_at TIMESTAMPTZ DEFAULT NOW(),
    rolled_back BOOLEAN DEFAULT false,
    rolled_back_at TIMESTAMPTZ,
    rolled_back_by UUID REFERENCES dashboard.users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_dashboard_schema_migrations_schema_name ON dashboard.schema_migrations(schema_name);
CREATE INDEX IF NOT EXISTS idx_dashboard_schema_migrations_applied_at ON dashboard.schema_migrations(applied_at DESC);

-- Email templates table
CREATE TABLE IF NOT EXISTS dashboard.email_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_type TEXT UNIQUE NOT NULL, -- 'magic_link', 'email_verification', 'password_reset'
    subject TEXT NOT NULL,
    html_body TEXT NOT NULL,
    text_body TEXT,
    is_custom BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dashboard_email_templates_type ON dashboard.email_templates(template_type);

COMMENT ON TABLE dashboard.email_templates IS 'Customizable email templates for authentication flows';
COMMENT ON COLUMN dashboard.email_templates.template_type IS 'Type of template: magic_link, email_verification, password_reset';
COMMENT ON COLUMN dashboard.email_templates.is_custom IS 'Whether this template has been customized from defaults';
