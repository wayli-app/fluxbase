-- Fluxbase Initial Database Schema
-- This migration creates the complete Fluxbase database schema
-- Version: 0.1.0

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

--
-- _FLUXBASE SCHEMA
-- Internal Fluxbase system schema for migration tracking and system tables
--

CREATE SCHEMA IF NOT EXISTS _fluxbase;
GRANT USAGE, CREATE ON SCHEMA _fluxbase TO CURRENT_USER;

-- User migrations tracking table (for user-provided migrations)
CREATE TABLE IF NOT EXISTS _fluxbase.user_migrations (
    version BIGINT NOT NULL PRIMARY KEY,
    dirty BOOLEAN NOT NULL
);

COMMENT ON TABLE _fluxbase.user_migrations IS 'Tracks user-provided database migration versions (managed by golang-migrate)';

--
-- AUTH SCHEMA
-- Handles application user authentication, API keys, and sessions
--

CREATE SCHEMA IF NOT EXISTS auth;
GRANT USAGE, CREATE ON SCHEMA auth TO CURRENT_USER;

-- Auth helper functions
CREATE OR REPLACE FUNCTION auth.current_user_id()
RETURNS UUID AS $$
DECLARE
    user_id_var TEXT;
BEGIN
    user_id_var := current_setting('app.user_id', true);
    IF user_id_var IS NULL OR user_id_var = '' THEN
        RETURN NULL;
    END IF;
    RETURN user_id_var::UUID;
EXCEPTION
    WHEN OTHERS THEN
        RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION auth.current_user_id() IS 'Returns the current authenticated user ID from PostgreSQL session variable app.user_id. Returns NULL if not set or invalid.';

-- Supabase compatibility: auth.uid() is an alias for auth.current_user_id()
CREATE OR REPLACE FUNCTION auth.uid()
RETURNS UUID AS $$
BEGIN
    RETURN auth.current_user_id();
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION auth.uid() IS 'Supabase-compatible alias for auth.current_user_id(). Returns the current authenticated user ID.';

-- Supabase compatibility: auth.jwt() returns JWT claims as JSONB
CREATE OR REPLACE FUNCTION auth.jwt()
RETURNS JSONB AS $$
DECLARE
    user_record RECORD;
    jwt_claims JSONB;
BEGIN
    -- Get the current user ID
    IF auth.current_user_id() IS NULL THEN
        RETURN NULL;
    END IF;

    -- Fetch user data including metadata
    SELECT
        id,
        email,
        role,
        user_metadata,
        app_metadata
    INTO user_record
    FROM auth.users
    WHERE id = auth.current_user_id();

    IF NOT FOUND THEN
        RETURN NULL;
    END IF;

    -- Build JWT claims object compatible with Supabase
    jwt_claims := jsonb_build_object(
        'sub', user_record.id,
        'email', user_record.email,
        'role', COALESCE(user_record.role, auth.current_user_role()),
        'user_metadata', COALESCE(user_record.user_metadata, '{}'::JSONB),
        'app_metadata', COALESCE(user_record.app_metadata, '{}'::JSONB)
    );

    RETURN jwt_claims;
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

COMMENT ON FUNCTION auth.jwt() IS 'Supabase-compatible function that returns JWT claims as JSONB, including user_metadata and app_metadata. Use ->> operator to extract text values or -> for JSONB.';

-- Supabase compatibility: auth.role() returns the current user's role
CREATE OR REPLACE FUNCTION auth.role()
RETURNS TEXT AS $$
BEGIN
    RETURN auth.current_user_role();
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION auth.role() IS 'Supabase-compatible alias for auth.current_user_role(). Returns the current user role.';

CREATE OR REPLACE FUNCTION auth.current_user_role()
RETURNS TEXT AS $$
DECLARE
    role_var TEXT;
BEGIN
    role_var := current_setting('app.role', true);
    IF role_var IS NULL OR role_var = '' THEN
        RETURN 'anon';
    END IF;
    RETURN role_var;
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION auth.current_user_role() IS 'Returns the current user role from PostgreSQL session variable app.role. Returns "anon" if not set.';

CREATE OR REPLACE FUNCTION auth.is_authenticated()
RETURNS BOOLEAN AS $$
BEGIN
    RETURN auth.current_user_id() IS NOT NULL;
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION auth.is_authenticated() IS 'Returns TRUE if a user is authenticated (user_id is set), FALSE for anonymous users.';

CREATE OR REPLACE FUNCTION auth.is_admin()
RETURNS BOOLEAN AS $$
BEGIN
    RETURN auth.current_user_role() = 'admin';
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION auth.is_admin() IS 'Returns TRUE if the current user role is "admin", FALSE otherwise.';

-- RLS helper functions
CREATE OR REPLACE FUNCTION auth.enable_rls(table_name TEXT, schema_name TEXT DEFAULT 'public')
RETURNS VOID AS $$
BEGIN
    EXECUTE format('ALTER TABLE %I.%I ENABLE ROW LEVEL SECURITY', schema_name, table_name);
    EXECUTE format('ALTER TABLE %I.%I FORCE ROW LEVEL SECURITY', schema_name, table_name);
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION auth.enable_rls(TEXT, TEXT) IS 'Enables Row Level Security on the specified table and forces it even for table owners.';

CREATE OR REPLACE FUNCTION auth.disable_rls(table_name TEXT, schema_name TEXT DEFAULT 'public')
RETURNS VOID AS $$
BEGIN
    EXECUTE format('ALTER TABLE %I.%I DISABLE ROW LEVEL SECURITY', schema_name, table_name);
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION auth.disable_rls(TEXT, TEXT) IS 'Disables Row Level Security on the specified table.';

-- Users table (with 2FA and split metadata support)
CREATE TABLE IF NOT EXISTS auth.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT,
    email_verified BOOLEAN DEFAULT false,
    role TEXT,
    user_metadata JSONB DEFAULT '{}'::JSONB,
    app_metadata JSONB DEFAULT '{}'::JSONB,
    totp_secret VARCHAR(32),
    totp_enabled BOOLEAN DEFAULT FALSE,
    backup_codes TEXT[],
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auth_users_email ON auth.users(email);
CREATE INDEX IF NOT EXISTS idx_auth_users_totp_enabled ON auth.users(totp_enabled) WHERE totp_enabled = TRUE;
CREATE INDEX IF NOT EXISTS idx_auth_users_user_metadata ON auth.users USING GIN (user_metadata);
CREATE INDEX IF NOT EXISTS idx_auth_users_app_metadata ON auth.users USING GIN (app_metadata);

COMMENT ON COLUMN auth.users.user_metadata IS 'User-editable metadata. Users can update this field themselves. Included in JWT claims.';
COMMENT ON COLUMN auth.users.app_metadata IS 'Application/admin-only metadata. Can only be updated by admins or service role. Included in JWT claims.';

-- Sessions table
CREATE TABLE IF NOT EXISTS auth.sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE NOT NULL,
    access_token TEXT UNIQUE NOT NULL,
    refresh_token TEXT UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auth_sessions_user_id ON auth.sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_access_token ON auth.sessions(access_token);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_refresh_token ON auth.sessions(refresh_token);

-- Magic links table
CREATE TABLE IF NOT EXISTS auth.magic_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL,
    token TEXT UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN DEFAULT false,
    used_at TIMESTAMPTZ,
    ip_address TEXT,
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auth_magic_links_token ON auth.magic_links(token);
CREATE INDEX IF NOT EXISTS idx_auth_magic_links_email ON auth.magic_links(email);

-- Password reset tokens table
CREATE TABLE IF NOT EXISTS auth.password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE NOT NULL,
    token TEXT UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN DEFAULT false,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auth_password_reset_tokens_token ON auth.password_reset_tokens(token);
CREATE INDEX IF NOT EXISTS idx_auth_password_reset_tokens_user_id ON auth.password_reset_tokens(user_id);

-- Token blacklist table
CREATE TABLE IF NOT EXISTS auth.token_blacklist (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_jti TEXT UNIQUE NOT NULL,
    token_type TEXT NOT NULL DEFAULT 'access',
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_by UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auth_token_blacklist_token_jti ON auth.token_blacklist(token_jti);
CREATE INDEX IF NOT EXISTS idx_auth_token_blacklist_expires_at ON auth.token_blacklist(expires_at);

-- API keys table
CREATE TABLE IF NOT EXISTS auth.api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    key_hash TEXT UNIQUE NOT NULL,
    key_prefix TEXT NOT NULL,
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    scopes TEXT[] DEFAULT ARRAY[]::TEXT[],
    rate_limit_per_minute INTEGER DEFAULT 60,
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    revoked BOOLEAN DEFAULT false,
    revoked_at TIMESTAMPTZ,
    revoked_by UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auth_api_keys_key_hash ON auth.api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_auth_api_keys_user_id ON auth.api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_auth_api_keys_key_prefix ON auth.api_keys(key_prefix);

-- API key usage tracking
CREATE TABLE IF NOT EXISTS auth.api_key_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    api_key_id UUID REFERENCES auth.api_keys(id) ON DELETE CASCADE NOT NULL,
    endpoint TEXT NOT NULL,
    method TEXT NOT NULL,
    status_code INTEGER,
    response_time_ms INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auth_api_key_usage_api_key_id ON auth.api_key_usage(api_key_id);
CREATE INDEX IF NOT EXISTS idx_auth_api_key_usage_created_at ON auth.api_key_usage(created_at DESC);

-- Service keys table (for service role authentication)
CREATE TABLE IF NOT EXISTS auth.service_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    key_hash TEXT NOT NULL,
    key_prefix TEXT NOT NULL,
    scopes TEXT[] DEFAULT ARRAY[]::TEXT[],
    enabled BOOLEAN DEFAULT true,
    created_by UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    UNIQUE(key_prefix)
);

CREATE INDEX IF NOT EXISTS idx_service_keys_prefix ON auth.service_keys(key_prefix);
CREATE INDEX IF NOT EXISTS idx_service_keys_enabled ON auth.service_keys(enabled);

COMMENT ON TABLE auth.service_keys IS 'Service role keys with elevated privileges that bypass RLS. Use for backend services only.';
COMMENT ON COLUMN auth.service_keys.key_hash IS 'Bcrypt hash of the full service key. Never store keys in plaintext.';
COMMENT ON COLUMN auth.service_keys.key_prefix IS 'First 16 characters of the key for identification in logs (e.g., "sk_test_Ab3xY...").';
COMMENT ON COLUMN auth.service_keys.scopes IS 'Optional array of scope restrictions. Empty array means full service role access.';

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

-- 2FA setup tracking table
CREATE TABLE IF NOT EXISTS auth.two_factor_setups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    secret VARCHAR(32) NOT NULL,
    qr_code_url TEXT NOT NULL,
    verified BOOLEAN DEFAULT FALSE,
    expires_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP + INTERVAL '10 minutes',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_2fa_setup_user FOREIGN KEY (user_id) REFERENCES auth.users(id) ON DELETE CASCADE,
    CONSTRAINT two_factor_setups_user_id_key UNIQUE (user_id)
);

CREATE INDEX IF NOT EXISTS idx_2fa_setup_user ON auth.two_factor_setups(user_id);
CREATE INDEX IF NOT EXISTS idx_2fa_setup_expires ON auth.two_factor_setups(expires_at);

COMMENT ON TABLE auth.two_factor_setups IS 'Temporary storage for 2FA setup process. Entries expire after 10 minutes and should be cleaned up periodically.';

-- 2FA recovery/backup code usage tracking table
CREATE TABLE IF NOT EXISTS auth.two_factor_recovery_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    code_used VARCHAR(255),
    success BOOLEAN NOT NULL,
    ip_address INET,
    user_agent TEXT,
    attempted_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_2fa_recovery_user FOREIGN KEY (user_id) REFERENCES auth.users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_2fa_recovery_user ON auth.two_factor_recovery_attempts(user_id);
CREATE INDEX IF NOT EXISTS idx_2fa_recovery_time ON auth.two_factor_recovery_attempts(attempted_at);

COMMENT ON TABLE auth.two_factor_recovery_attempts IS 'Audit log for 2FA recovery/backup code usage attempts for security monitoring.';

-- Webhooks table
CREATE TABLE IF NOT EXISTS auth.webhooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    url TEXT NOT NULL,
    events JSONB DEFAULT '[]'::JSONB,
    secret TEXT,
    enabled BOOLEAN DEFAULT true,
    headers JSONB DEFAULT '{}'::JSONB,
    timeout_seconds INTEGER DEFAULT 30,
    max_retries INTEGER DEFAULT 3,
    retry_backoff_seconds INTEGER DEFAULT 5,
    created_by UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auth_webhooks_enabled ON auth.webhooks(enabled);

-- Webhook deliveries table
CREATE TABLE IF NOT EXISTS auth.webhook_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id UUID REFERENCES auth.webhooks(id) ON DELETE CASCADE NOT NULL,
    event TEXT NOT NULL,
    payload JSONB NOT NULL,
    status TEXT NOT NULL,
    status_code INTEGER,
    response_body TEXT,
    error TEXT,
    attempt INTEGER DEFAULT 1,
    delivered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auth_webhook_deliveries_webhook_id ON auth.webhook_deliveries(webhook_id);
CREATE INDEX IF NOT EXISTS idx_auth_webhook_deliveries_created_at ON auth.webhook_deliveries(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_auth_webhook_deliveries_status ON auth.webhook_deliveries(status);

-- Webhook event queue table
CREATE TABLE IF NOT EXISTS auth.webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id UUID REFERENCES auth.webhooks(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    table_schema VARCHAR(255) NOT NULL,
    table_name VARCHAR(255) NOT NULL,
    record_id TEXT,
    old_data JSONB,
    new_data JSONB,
    processed BOOLEAN DEFAULT FALSE,
    attempts INT DEFAULT 0,
    last_attempt_at TIMESTAMPTZ,
    next_retry_at TIMESTAMPTZ,
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_webhook_event_webhook FOREIGN KEY (webhook_id) REFERENCES auth.webhooks(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_webhook_events_unprocessed ON auth.webhook_events(processed, next_retry_at) WHERE processed = FALSE;
CREATE INDEX IF NOT EXISTS idx_webhook_events_webhook ON auth.webhook_events(webhook_id);
CREATE INDEX IF NOT EXISTS idx_webhook_events_created ON auth.webhook_events(created_at);

COMMENT ON TABLE auth.webhook_events IS 'Queue for webhook events to be delivered. Processed events are kept for history.';

-- Impersonation sessions table
CREATE TABLE IF NOT EXISTS auth.impersonation_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE NOT NULL,
    impersonated_user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE NOT NULL,
    impersonation_type TEXT NOT NULL DEFAULT 'full',
    reason TEXT,
    started_at TIMESTAMPTZ DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT true
);

CREATE INDEX IF NOT EXISTS idx_auth_impersonation_admin_user_id ON auth.impersonation_sessions(admin_user_id);
CREATE INDEX IF NOT EXISTS idx_auth_impersonation_impersonated_user_id ON auth.impersonation_sessions(impersonated_user_id);
CREATE INDEX IF NOT EXISTS idx_auth_impersonation_is_active ON auth.impersonation_sessions(is_active);

--
-- DASHBOARD SCHEMA
-- Handles Fluxbase platform administrator authentication and management
--

CREATE SCHEMA IF NOT EXISTS dashboard;
GRANT USAGE, CREATE ON SCHEMA dashboard TO CURRENT_USER;

COMMENT ON SCHEMA dashboard IS 'Schema for dashboard/platform administrator authentication and management';

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

--
-- FUNCTIONS SCHEMA
-- Handles edge functions and their executions
--

CREATE SCHEMA IF NOT EXISTS functions;
GRANT USAGE, CREATE ON SCHEMA functions TO CURRENT_USER;

-- Edge functions table (with allow_unauthenticated support)
CREATE TABLE IF NOT EXISTS functions.edge_functions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT UNIQUE NOT NULL,
    description TEXT,
    code TEXT NOT NULL,
    enabled BOOLEAN DEFAULT true,
    timeout_seconds INTEGER DEFAULT 30,
    memory_limit_mb INTEGER DEFAULT 128,
    allow_net BOOLEAN DEFAULT true,
    allow_env BOOLEAN DEFAULT true,
    allow_read BOOLEAN DEFAULT false,
    allow_write BOOLEAN DEFAULT false,
    allow_unauthenticated BOOLEAN DEFAULT false,
    cron_schedule TEXT,
    version INTEGER DEFAULT 1,
    created_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_functions_edge_functions_name ON functions.edge_functions(name);
CREATE INDEX IF NOT EXISTS idx_functions_edge_functions_enabled ON functions.edge_functions(enabled);
CREATE INDEX IF NOT EXISTS idx_functions_edge_functions_cron_schedule ON functions.edge_functions(cron_schedule) WHERE cron_schedule IS NOT NULL;

COMMENT ON COLUMN functions.edge_functions.allow_unauthenticated IS 'When true, allows this function to be invoked without authentication. Use with caution.';

-- Edge function triggers table
CREATE TABLE IF NOT EXISTS functions.edge_function_triggers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    function_id UUID REFERENCES functions.edge_functions(id) ON DELETE CASCADE NOT NULL,
    trigger_type TEXT NOT NULL,
    schema_name TEXT,
    table_name TEXT,
    events TEXT[] DEFAULT ARRAY[]::TEXT[],
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_functions_edge_function_triggers_function_id ON functions.edge_function_triggers(function_id);
CREATE INDEX IF NOT EXISTS idx_functions_edge_function_triggers_enabled ON functions.edge_function_triggers(enabled);
CREATE INDEX IF NOT EXISTS idx_functions_edge_function_triggers_table ON functions.edge_function_triggers(schema_name, table_name);

-- Edge function executions table
CREATE TABLE IF NOT EXISTS functions.edge_function_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    function_id UUID REFERENCES functions.edge_functions(id) ON DELETE CASCADE NOT NULL,
    trigger_type TEXT NOT NULL,
    status TEXT NOT NULL,
    status_code INTEGER,
    error_message TEXT,
    logs TEXT,
    result TEXT,
    duration_ms INTEGER,
    started_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_functions_edge_function_executions_function_id ON functions.edge_function_executions(function_id);
CREATE INDEX IF NOT EXISTS idx_functions_edge_function_executions_started_at ON functions.edge_function_executions(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_functions_edge_function_executions_status ON functions.edge_function_executions(status);

--
-- STORAGE SCHEMA
-- Handles file storage buckets and objects
--

CREATE SCHEMA IF NOT EXISTS storage;
GRANT USAGE, CREATE ON SCHEMA storage TO CURRENT_USER;

-- Storage buckets table
CREATE TABLE IF NOT EXISTS storage.buckets (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    public BOOLEAN DEFAULT false,
    allowed_mime_types TEXT[],
    max_file_size BIGINT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Storage objects table
CREATE TABLE IF NOT EXISTS storage.objects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bucket_id TEXT REFERENCES storage.buckets(id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    mime_type TEXT,
    size BIGINT,
    metadata JSONB,
    owner_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(bucket_id, path)
);

CREATE INDEX IF NOT EXISTS idx_storage_objects_bucket_id ON storage.objects(bucket_id);
CREATE INDEX IF NOT EXISTS idx_storage_objects_owner_id ON storage.objects(owner_id);

-- Storage object permissions table (for file sharing)
CREATE TABLE IF NOT EXISTS storage.object_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    object_id UUID REFERENCES storage.objects(id) ON DELETE CASCADE,
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    permission TEXT NOT NULL CHECK (permission IN ('read', 'write')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(object_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_storage_object_permissions_object_id ON storage.object_permissions(object_id);
CREATE INDEX IF NOT EXISTS idx_storage_object_permissions_user_id ON storage.object_permissions(user_id);

COMMENT ON TABLE storage.object_permissions IS 'Tracks file sharing permissions between users';
COMMENT ON COLUMN storage.object_permissions.permission IS 'Permission level: read (download only) or write (download, update, delete)';

-- Insert default buckets
INSERT INTO storage.buckets (id, name, public) VALUES
    ('public', 'public', true),
    ('temp-files', 'temp-files', false),
    ('user-uploads', 'user-uploads', false)
ON CONFLICT (id) DO NOTHING;

COMMENT ON TABLE storage.buckets IS 'Storage buckets configuration. Public buckets allow unauthenticated read access.';
COMMENT ON TABLE storage.objects IS 'Storage objects metadata. All file operations are tracked here for RLS enforcement.';

-- Storage helper functions (moved here after storage schema is created)
CREATE OR REPLACE FUNCTION storage.user_can_access_object(p_object_id UUID, p_required_permission TEXT DEFAULT 'read')
RETURNS BOOLEAN AS $$
DECLARE
    v_owner_id UUID;
    v_bucket_public BOOLEAN;
    v_has_permission BOOLEAN;
    v_user_role TEXT;
BEGIN
    v_user_role := auth.current_user_role();

    -- Admins and service roles can access everything
    IF v_user_role IN ('dashboard_admin', 'service_role') THEN
        RETURN TRUE;
    END IF;

    -- Get object owner and bucket public status
    SELECT o.owner_id, b.public INTO v_owner_id, v_bucket_public
    FROM storage.objects o
    JOIN storage.buckets b ON b.id = o.bucket_id
    WHERE o.id = p_object_id;

    -- If object not found, deny access
    IF NOT FOUND THEN
        RETURN FALSE;
    END IF;

    -- Check if user is the owner
    IF v_owner_id = auth.current_user_id() THEN
        RETURN TRUE;
    END IF;

    -- Check if bucket is public (read-only for non-owners)
    IF v_bucket_public AND p_required_permission = 'read' THEN
        RETURN TRUE;
    END IF;

    -- Check object_permissions table for explicit shares
    IF auth.current_user_id() IS NOT NULL THEN
        SELECT EXISTS(
            SELECT 1 FROM storage.object_permissions
            WHERE object_id = p_object_id
            AND user_id = auth.current_user_id()
            AND (permission = 'write' OR (permission = 'read' AND p_required_permission = 'read'))
        ) INTO v_has_permission;

        IF v_has_permission THEN
            RETURN TRUE;
        END IF;
    END IF;

    RETURN FALSE;
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

COMMENT ON FUNCTION storage.user_can_access_object(UUID, TEXT) IS 'Checks if the current user can access a storage object with the required permission (read or write). Returns TRUE if: user is admin/service role, user owns the object, object is in public bucket (read only), or user has been granted permission via object_permissions table.';

--
-- REALTIME SCHEMA
-- Handles realtime subscriptions and change tracking
--

CREATE SCHEMA IF NOT EXISTS realtime;
GRANT USAGE, CREATE ON SCHEMA realtime TO CURRENT_USER;

-- Realtime schema registry table
CREATE TABLE IF NOT EXISTS realtime.schema_registry (
    id SERIAL PRIMARY KEY,
    schema_name TEXT NOT NULL,
    table_name TEXT NOT NULL,
    realtime_enabled BOOLEAN DEFAULT true,
    events TEXT[] DEFAULT ARRAY['INSERT', 'UPDATE', 'DELETE'],
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(schema_name, table_name)
);

--
-- GLOBAL HELPER FUNCTIONS
--

-- Update trigger function for updated_at columns
CREATE OR REPLACE FUNCTION public.update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Webhook trigger function to queue webhook events
CREATE OR REPLACE FUNCTION auth.queue_webhook_event()
RETURNS TRIGGER AS $$
DECLARE
    webhook_record RECORD;
    event_type TEXT;
    old_data JSONB;
    new_data JSONB;
    record_id_value TEXT;
    should_trigger BOOLEAN;
BEGIN
    -- Determine event type
    IF TG_OP = 'INSERT' THEN
        event_type := 'INSERT';
        old_data := NULL;
        new_data := to_jsonb(NEW);
        record_id_value := COALESCE((NEW.id)::TEXT, '');
    ELSIF TG_OP = 'UPDATE' THEN
        event_type := 'UPDATE';
        old_data := to_jsonb(OLD);
        new_data := to_jsonb(NEW);
        record_id_value := COALESCE((NEW.id)::TEXT, (OLD.id)::TEXT, '');
    ELSIF TG_OP = 'DELETE' THEN
        event_type := 'DELETE';
        old_data := to_jsonb(OLD);
        new_data := NULL;
        record_id_value := COALESCE((OLD.id)::TEXT, '');
    ELSE
        RETURN NULL;
    END IF;

    -- Find matching webhooks
    FOR webhook_record IN
        SELECT id, events
        FROM auth.webhooks
        WHERE enabled = TRUE
    LOOP
        -- Check if this webhook is interested in this event
        should_trigger := FALSE;

        -- Parse the events JSONB array to check if it matches
        IF jsonb_typeof(webhook_record.events) = 'array' THEN
            should_trigger := EXISTS (
                SELECT 1
                FROM jsonb_array_elements(webhook_record.events) AS event
                WHERE
                    (event->>'table' = TG_TABLE_NAME OR event->>'table' = '*')
                    AND (
                        event->'operations' @> to_jsonb(ARRAY[event_type])
                        OR event->'operations' @> to_jsonb(ARRAY['*'])
                    )
            );
        END IF;

        -- Queue event if webhook is interested
        IF should_trigger THEN
            INSERT INTO auth.webhook_events (
                webhook_id,
                event_type,
                table_schema,
                table_name,
                record_id,
                old_data,
                new_data,
                next_retry_at
            ) VALUES (
                webhook_record.id,
                event_type,
                TG_TABLE_SCHEMA,
                TG_TABLE_NAME,
                record_id_value,
                old_data,
                new_data,
                CURRENT_TIMESTAMP
            );

            -- Send notification to application via pg_notify
            PERFORM pg_notify('webhook_event', webhook_record.id::TEXT);
        END IF;
    END LOOP;

    -- Return appropriate value based on operation
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    ELSE
        RETURN NEW;
    END IF;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION auth.queue_webhook_event() IS 'Trigger function that queues webhook events when data changes occur';

-- Function to create webhook trigger on a table
CREATE OR REPLACE FUNCTION auth.create_webhook_trigger(
    schema_name TEXT,
    table_name TEXT
) RETURNS VOID AS $$
DECLARE
    trigger_name TEXT;
    full_table_name TEXT;
BEGIN
    trigger_name := format('webhook_trigger_%s_%s', schema_name, table_name);
    full_table_name := format('%I.%I', schema_name, table_name);

    -- Drop existing trigger if exists
    EXECUTE format('DROP TRIGGER IF EXISTS %I ON %s', trigger_name, full_table_name);

    -- Create new trigger
    EXECUTE format('
        CREATE TRIGGER %I
        AFTER INSERT OR UPDATE OR DELETE ON %s
        FOR EACH ROW EXECUTE FUNCTION auth.queue_webhook_event()
    ', trigger_name, full_table_name);

    RAISE NOTICE 'Created webhook trigger % on %', trigger_name, full_table_name;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

COMMENT ON FUNCTION auth.create_webhook_trigger IS 'Creates a webhook trigger on a specified table';

-- Function to remove webhook trigger from a table
CREATE OR REPLACE FUNCTION auth.remove_webhook_trigger(
    schema_name TEXT,
    table_name TEXT
) RETURNS VOID AS $$
DECLARE
    trigger_name TEXT;
    full_table_name TEXT;
BEGIN
    trigger_name := format('webhook_trigger_%s_%s', schema_name, table_name);
    full_table_name := format('%I.%I', schema_name, table_name);

    EXECUTE format('DROP TRIGGER IF EXISTS %I ON %s', trigger_name, full_table_name);

    RAISE NOTICE 'Removed webhook trigger % from %', trigger_name, full_table_name;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

COMMENT ON FUNCTION auth.remove_webhook_trigger IS 'Removes a webhook trigger from a specified table';

-- Function to validate app_metadata updates (only admins can modify)
CREATE OR REPLACE FUNCTION auth.validate_app_metadata_update()
RETURNS TRIGGER AS $$
DECLARE
    user_role TEXT;
BEGIN
    -- Get the current user's role
    user_role := auth.current_user_role();

    -- Check if app_metadata is being modified
    IF OLD.app_metadata IS DISTINCT FROM NEW.app_metadata THEN
        -- Only allow admins and dashboard admins to modify app_metadata
        IF user_role != 'admin' AND user_role != 'dashboard_admin' THEN
            -- Also check if user has admin privileges via is_admin() function
            IF NOT auth.is_admin() THEN
                RAISE EXCEPTION 'Only admins can modify app_metadata'
                    USING ERRCODE = 'insufficient_privilege';
            END IF;
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

COMMENT ON FUNCTION auth.validate_app_metadata_update() IS 'Validates that only admins and dashboard admins can modify the app_metadata field on auth.users';

-- Webhook updated_at trigger function
CREATE OR REPLACE FUNCTION auth.update_webhook_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

--
-- APPLY UPDATE TRIGGERS
--

-- Auth schema triggers
CREATE TRIGGER update_auth_users_updated_at BEFORE UPDATE ON auth.users
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_auth_sessions_updated_at BEFORE UPDATE ON auth.sessions
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_auth_api_keys_updated_at BEFORE UPDATE ON auth.api_keys
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_oauth_links_updated_at BEFORE UPDATE ON auth.oauth_links
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_oauth_tokens_updated_at BEFORE UPDATE ON auth.oauth_tokens
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_auth_webhooks_updated_at BEFORE UPDATE ON auth.webhooks
    FOR EACH ROW EXECUTE FUNCTION auth.update_webhook_updated_at();

-- App metadata protection trigger
CREATE TRIGGER validate_app_metadata_trigger BEFORE UPDATE ON auth.users
    FOR EACH ROW EXECUTE FUNCTION auth.validate_app_metadata_update();

-- Dashboard schema triggers
CREATE TRIGGER update_dashboard_users_updated_at BEFORE UPDATE ON dashboard.users
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_dashboard_sessions_updated_at BEFORE UPDATE ON dashboard.sessions
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_dashboard_oauth_providers_updated_at BEFORE UPDATE ON dashboard.oauth_providers
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_dashboard_auth_settings_updated_at BEFORE UPDATE ON dashboard.auth_settings
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_dashboard_system_settings_updated_at BEFORE UPDATE ON dashboard.system_settings
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_dashboard_custom_settings_updated_at BEFORE UPDATE ON dashboard.custom_settings
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_dashboard_email_templates_updated_at BEFORE UPDATE ON dashboard.email_templates
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

-- App metadata protection trigger for dashboard users
CREATE TRIGGER validate_app_metadata_trigger BEFORE UPDATE ON dashboard.users
    FOR EACH ROW EXECUTE FUNCTION auth.validate_app_metadata_update();

-- Functions schema triggers
CREATE TRIGGER update_functions_edge_functions_updated_at BEFORE UPDATE ON functions.edge_functions
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_functions_edge_function_triggers_updated_at BEFORE UPDATE ON functions.edge_function_triggers
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

-- Storage schema triggers
CREATE TRIGGER update_storage_buckets_updated_at BEFORE UPDATE ON storage.buckets
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_storage_objects_updated_at BEFORE UPDATE ON storage.objects
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

-- Realtime schema triggers
CREATE TRIGGER update_realtime_schema_registry_updated_at BEFORE UPDATE ON realtime.schema_registry
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

--
-- ROW LEVEL SECURITY (RLS) POLICIES
-- Comprehensive security policies with FORCE RLS enabled
--

-- ============================================================================
-- DASHBOARD SCHEMA RLS
-- ============================================================================

-- Dashboard users table
ALTER TABLE dashboard.users ENABLE ROW LEVEL SECURITY;
ALTER TABLE dashboard.users FORCE ROW LEVEL SECURITY;

CREATE POLICY dashboard_users_insert_policy ON dashboard.users
    FOR INSERT
    WITH CHECK (
        (SELECT COUNT(*) FROM dashboard.users) = 0
        OR auth.current_user_role() = 'dashboard_admin'
        OR EXISTS (
            SELECT 1 FROM dashboard.invitation_tokens
            WHERE token = current_setting('app.invitation_token', true)
            AND accepted = false
            AND expires_at > NOW()
        )
    );

CREATE POLICY dashboard_users_select_policy ON dashboard.users
    FOR SELECT
    USING (
        auth.current_user_role() = 'dashboard_admin'
        OR auth.current_user_id()::TEXT = id::TEXT
    );

CREATE POLICY dashboard_users_update_policy ON dashboard.users
    FOR UPDATE
    USING (
        auth.current_user_role() = 'dashboard_admin'
        OR auth.current_user_id()::TEXT = id::TEXT
    );

CREATE POLICY dashboard_users_delete_policy ON dashboard.users
    FOR DELETE
    USING (auth.current_user_role() = 'dashboard_admin');

-- Dashboard sessions table
ALTER TABLE dashboard.sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE dashboard.sessions FORCE ROW LEVEL SECURITY;

CREATE POLICY dashboard_sessions_all_policy ON dashboard.sessions
    FOR ALL
    USING (
        auth.current_user_role() = 'dashboard_admin'
        OR auth.current_user_id()::TEXT = user_id::TEXT
    );

-- Dashboard system settings table
ALTER TABLE dashboard.system_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE dashboard.system_settings FORCE ROW LEVEL SECURITY;

CREATE POLICY dashboard_system_settings_select_policy ON dashboard.system_settings
    FOR SELECT
    USING (auth.current_user_role() = 'dashboard_admin');

CREATE POLICY dashboard_system_settings_modify_policy ON dashboard.system_settings
    FOR ALL
    USING (auth.current_user_role() = 'dashboard_admin');

-- Dashboard custom settings table
ALTER TABLE dashboard.custom_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE dashboard.custom_settings FORCE ROW LEVEL SECURITY;

-- dashboard_admin can do everything with custom settings
CREATE POLICY dashboard_custom_settings_dashboard_admin_all ON dashboard.custom_settings
    FOR ALL
    USING (auth.current_user_role() = 'dashboard_admin')
    WITH CHECK (auth.current_user_role() = 'dashboard_admin');

-- admin can read all custom settings
CREATE POLICY dashboard_custom_settings_admin_select ON dashboard.custom_settings
    FOR SELECT
    USING (auth.current_user_role() = 'admin');

-- admin can update/delete only if 'admin' is in editable_by array
CREATE POLICY dashboard_custom_settings_admin_update ON dashboard.custom_settings
    FOR UPDATE
    USING (auth.current_user_role() = 'admin' AND 'admin' = ANY(editable_by))
    WITH CHECK (auth.current_user_role() = 'admin' AND 'admin' = ANY(editable_by));

CREATE POLICY dashboard_custom_settings_admin_delete ON dashboard.custom_settings
    FOR DELETE
    USING (auth.current_user_role() = 'admin' AND 'admin' = ANY(editable_by));

-- service_role can do everything (for internal operations)
CREATE POLICY dashboard_custom_settings_service_role_all ON dashboard.custom_settings
    FOR ALL
    USING (auth.current_user_role() = 'service_role')
    WITH CHECK (auth.current_user_role() = 'service_role');

COMMENT ON POLICY dashboard_custom_settings_dashboard_admin_all ON dashboard.custom_settings IS 'Dashboard admins have full access to all custom settings';
COMMENT ON POLICY dashboard_custom_settings_admin_select ON dashboard.custom_settings IS 'Admins can read all custom settings';
COMMENT ON POLICY dashboard_custom_settings_admin_update ON dashboard.custom_settings IS 'Admins can update custom settings only if admin is in editable_by array';
COMMENT ON POLICY dashboard_custom_settings_admin_delete ON dashboard.custom_settings IS 'Admins can delete custom settings only if admin is in editable_by array';
COMMENT ON POLICY dashboard_custom_settings_service_role_all ON dashboard.custom_settings IS 'Service role has full access for internal operations';

-- Dashboard invitation tokens table
ALTER TABLE dashboard.invitation_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE dashboard.invitation_tokens FORCE ROW LEVEL SECURITY;

CREATE POLICY dashboard_invitation_tokens_select_policy ON dashboard.invitation_tokens
    FOR SELECT
    USING (auth.current_user_role() = 'dashboard_admin');

CREATE POLICY dashboard_invitation_tokens_modify_policy ON dashboard.invitation_tokens
    FOR ALL
    USING (auth.current_user_role() = 'dashboard_admin');

-- Dashboard email templates
ALTER TABLE dashboard.email_templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE dashboard.email_templates FORCE ROW LEVEL SECURITY;

CREATE POLICY dashboard_email_templates_select_policy ON dashboard.email_templates
    FOR SELECT
    USING (auth.current_user_role() = 'dashboard_admin' OR auth.current_user_role() = 'service_role');

CREATE POLICY dashboard_email_templates_modify_policy ON dashboard.email_templates
    FOR ALL
    USING (auth.current_user_role() = 'dashboard_admin');

COMMENT ON POLICY dashboard_email_templates_select_policy ON dashboard.email_templates IS 'Dashboard admins and service role can read email templates';
COMMENT ON POLICY dashboard_email_templates_modify_policy ON dashboard.email_templates IS 'Only dashboard admins can modify email templates';

-- Dashboard password reset tokens
ALTER TABLE dashboard.password_reset_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE dashboard.password_reset_tokens FORCE ROW LEVEL SECURITY;

CREATE POLICY dashboard_password_reset_service_only ON dashboard.password_reset_tokens
    FOR ALL
    USING (
        auth.current_user_role() = 'service_role'
        OR auth.current_user_role() = 'dashboard_admin'
    );

COMMENT ON POLICY dashboard_password_reset_service_only ON dashboard.password_reset_tokens IS 'Only service role and dashboard admins can access dashboard password reset tokens.';

-- Dashboard email verification tokens
ALTER TABLE dashboard.email_verification_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE dashboard.email_verification_tokens FORCE ROW LEVEL SECURITY;

CREATE POLICY dashboard_email_verification_service_only ON dashboard.email_verification_tokens
    FOR ALL
    USING (
        auth.current_user_role() = 'service_role'
        OR auth.current_user_role() = 'dashboard_admin'
    );

COMMENT ON POLICY dashboard_email_verification_service_only ON dashboard.email_verification_tokens IS 'Only service role and dashboard admins can access email verification tokens.';

-- Dashboard activity log
ALTER TABLE dashboard.activity_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE dashboard.activity_log FORCE ROW LEVEL SECURITY;

CREATE POLICY activity_log_service_write ON dashboard.activity_log
    FOR INSERT
    WITH CHECK (auth.current_user_role() = 'service_role');

COMMENT ON POLICY activity_log_service_write ON dashboard.activity_log IS 'Service role can create activity log entries.';

CREATE POLICY activity_log_admin_read ON dashboard.activity_log
    FOR SELECT
    USING (auth.current_user_role() = 'dashboard_admin');

COMMENT ON POLICY activity_log_admin_read ON dashboard.activity_log IS 'Dashboard admins can view activity log entries.';

-- Dashboard OAuth providers
ALTER TABLE dashboard.oauth_providers ENABLE ROW LEVEL SECURITY;
ALTER TABLE dashboard.oauth_providers FORCE ROW LEVEL SECURITY;

CREATE POLICY oauth_providers_dashboard_admin_only ON dashboard.oauth_providers
    FOR ALL
    USING (
        auth.current_user_role() = 'service_role'
        OR auth.current_user_role() = 'dashboard_admin'
    );

COMMENT ON POLICY oauth_providers_dashboard_admin_only ON dashboard.oauth_providers IS 'Only dashboard admins and service role can manage OAuth providers.';

-- Dashboard auth settings
ALTER TABLE dashboard.auth_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE dashboard.auth_settings FORCE ROW LEVEL SECURITY;

CREATE POLICY auth_settings_dashboard_admin_only ON dashboard.auth_settings
    FOR ALL
    USING (
        auth.current_user_role() = 'service_role'
        OR auth.current_user_role() = 'dashboard_admin'
    );

COMMENT ON POLICY auth_settings_dashboard_admin_only ON dashboard.auth_settings IS 'Only dashboard admins and service role can manage auth settings.';

-- ============================================================================
-- AUTH SCHEMA RLS
-- ============================================================================

-- Auth users table
ALTER TABLE auth.users ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.users FORCE ROW LEVEL SECURITY;

COMMENT ON TABLE auth.users IS 'Application users with FORCE RLS - even table owners must follow policies';

CREATE POLICY auth_users_insert_policy ON auth.users
    FOR INSERT
    WITH CHECK (true);

CREATE POLICY auth_users_select_own ON auth.users
    FOR SELECT
    USING (
        id = auth.current_user_id()
        OR auth.is_admin()
        OR auth.current_user_role() = 'dashboard_admin'
        OR auth.current_user_role() = 'service_role'
        OR auth.current_user_role() = 'anon'
    );

COMMENT ON POLICY auth_users_select_own ON auth.users IS 'Users can only see their own record. Admins, dashboard admins, service role, and anon role (for signup RETURNING) can see all users.';

CREATE POLICY auth_users_update_policy ON auth.users
    FOR UPDATE
    USING (
        auth.is_admin()
        OR auth.current_user_role() = 'dashboard_admin'
        OR auth.current_user_id()::TEXT = id::TEXT
    );

CREATE POLICY auth_users_delete_policy ON auth.users
    FOR DELETE
    USING (
        auth.is_admin()
        OR auth.current_user_role() = 'dashboard_admin'
    );

-- Auth sessions table
ALTER TABLE auth.sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.sessions FORCE ROW LEVEL SECURITY;

COMMENT ON TABLE auth.sessions IS 'User sessions with FORCE RLS for security';

CREATE POLICY auth_sessions_select ON auth.sessions
    FOR SELECT
    USING (
        user_id = auth.current_user_id()
        OR auth.current_user_role() = 'service_role'
        OR auth.current_user_role() = 'dashboard_admin'
        OR auth.current_user_role() = 'anon'
    );

COMMENT ON POLICY auth_sessions_select ON auth.sessions IS 'Users can view their own sessions. Service role, dashboard admins, and anon role (for signup RETURNING) can view all sessions.';

CREATE POLICY auth_sessions_insert ON auth.sessions
    FOR INSERT
    WITH CHECK (
        user_id = auth.current_user_id()
        OR auth.current_user_role() = 'service_role'
        OR auth.current_user_role() = 'anon'
    );

COMMENT ON POLICY auth_sessions_insert ON auth.sessions IS 'Users can create sessions for themselves. Service role can create sessions for any user. Anon users can create sessions (signup flow).';

CREATE POLICY auth_sessions_update ON auth.sessions
    FOR UPDATE
    USING (
        user_id = auth.current_user_id()
        OR auth.current_user_role() = 'service_role'
    )
    WITH CHECK (
        user_id = auth.current_user_id()
        OR auth.current_user_role() = 'service_role'
    );

COMMENT ON POLICY auth_sessions_update ON auth.sessions IS 'Users can update their own sessions. Service role can update any session.';

CREATE POLICY auth_sessions_delete ON auth.sessions
    FOR DELETE
    USING (
        user_id = auth.current_user_id()
        OR auth.current_user_role() = 'service_role'
        OR auth.current_user_role() = 'dashboard_admin'
    );

COMMENT ON POLICY auth_sessions_delete ON auth.sessions IS 'Users can delete their own sessions. Service role and dashboard admins can delete any session.';

-- Auth API keys table
ALTER TABLE auth.api_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.api_keys FORCE ROW LEVEL SECURITY;

CREATE POLICY auth_api_keys_policy ON auth.api_keys
    FOR ALL
    USING (
        auth.is_admin()
        OR auth.current_user_role() = 'dashboard_admin'
        OR auth.current_user_id()::TEXT = user_id::TEXT
    );

-- Auth API key usage
ALTER TABLE auth.api_key_usage ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.api_key_usage FORCE ROW LEVEL SECURITY;

CREATE POLICY api_key_usage_service_write ON auth.api_key_usage
    FOR INSERT
    WITH CHECK (auth.current_user_role() = 'service_role');

COMMENT ON POLICY api_key_usage_service_write ON auth.api_key_usage IS 'Service role can record API key usage.';

CREATE POLICY api_key_usage_user_read ON auth.api_key_usage
    FOR SELECT
    USING (
        api_key_id IN (
            SELECT id FROM auth.api_keys WHERE user_id = auth.current_user_id()
        )
        OR auth.is_admin()
        OR auth.current_user_role() = 'dashboard_admin'
        OR auth.current_user_role() = 'service_role'
    );

COMMENT ON POLICY api_key_usage_user_read ON auth.api_key_usage IS 'Users can view usage for their own API keys. Admins can view all usage.';

-- Auth magic links
ALTER TABLE auth.magic_links ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.magic_links FORCE ROW LEVEL SECURITY;

CREATE POLICY magic_links_service_only ON auth.magic_links
    FOR ALL
    USING (auth.current_user_role() = 'service_role');

COMMENT ON POLICY magic_links_service_only ON auth.magic_links IS 'Only service role can access magic links (used internally for auth flow).';

-- Auth password reset tokens
ALTER TABLE auth.password_reset_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.password_reset_tokens FORCE ROW LEVEL SECURITY;

CREATE POLICY password_reset_tokens_service_only ON auth.password_reset_tokens
    FOR ALL
    USING (auth.current_user_role() = 'service_role');

COMMENT ON POLICY password_reset_tokens_service_only ON auth.password_reset_tokens IS 'Only service role can access password reset tokens (used internally for password reset flow).';

-- Auth token blacklist
ALTER TABLE auth.token_blacklist ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.token_blacklist FORCE ROW LEVEL SECURITY;

CREATE POLICY token_blacklist_admin_only ON auth.token_blacklist
    FOR ALL
    USING (
        auth.current_user_role() = 'service_role'
        OR auth.current_user_role() = 'dashboard_admin'
    );

COMMENT ON POLICY token_blacklist_admin_only ON auth.token_blacklist IS 'Only service role and dashboard admins can access token blacklist.';

-- OAuth links
ALTER TABLE auth.oauth_links ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.oauth_links FORCE ROW LEVEL SECURITY;

CREATE POLICY oauth_links_select ON auth.oauth_links
    FOR SELECT
    USING (user_id = auth.current_user_id());

CREATE POLICY oauth_links_service_all ON auth.oauth_links
    FOR ALL
    USING (auth.current_user_role() = 'service_role');

-- OAuth tokens
ALTER TABLE auth.oauth_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.oauth_tokens FORCE ROW LEVEL SECURITY;

CREATE POLICY oauth_tokens_select ON auth.oauth_tokens
    FOR SELECT
    USING (user_id = auth.current_user_id());

CREATE POLICY oauth_tokens_service_all ON auth.oauth_tokens
    FOR ALL
    USING (auth.current_user_role() = 'service_role');

-- 2FA setups
ALTER TABLE auth.two_factor_setups ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.two_factor_setups FORCE ROW LEVEL SECURITY;

CREATE POLICY two_factor_setups_select ON auth.two_factor_setups
    FOR SELECT
    USING (user_id = auth.current_user_id());

CREATE POLICY two_factor_setups_insert ON auth.two_factor_setups
    FOR INSERT
    WITH CHECK (user_id = auth.current_user_id());

CREATE POLICY two_factor_setups_delete ON auth.two_factor_setups
    FOR DELETE
    USING (user_id = auth.current_user_id());

CREATE POLICY two_factor_setups_admin_select ON auth.two_factor_setups
    FOR SELECT
    USING (auth.is_admin() OR auth.current_user_role() = 'dashboard_admin');

-- 2FA recovery attempts
ALTER TABLE auth.two_factor_recovery_attempts ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.two_factor_recovery_attempts FORCE ROW LEVEL SECURITY;

CREATE POLICY two_factor_recovery_select ON auth.two_factor_recovery_attempts
    FOR SELECT
    USING (user_id = auth.current_user_id());

CREATE POLICY two_factor_recovery_admin_select ON auth.two_factor_recovery_attempts
    FOR SELECT
    USING (auth.is_admin() OR auth.current_user_role() = 'dashboard_admin');

-- Webhooks
ALTER TABLE auth.webhooks ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.webhooks FORCE ROW LEVEL SECURITY;

CREATE POLICY webhooks_admin_only ON auth.webhooks
    FOR ALL
    USING (
        auth.current_user_role() = 'service_role'
        OR auth.current_user_role() = 'dashboard_admin'
        OR auth.is_admin()
    );

COMMENT ON POLICY webhooks_admin_only ON auth.webhooks IS 'Only admins, dashboard admins, and service role can manage webhooks.';

-- Webhook deliveries
ALTER TABLE auth.webhook_deliveries ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.webhook_deliveries FORCE ROW LEVEL SECURITY;

CREATE POLICY webhook_deliveries_service_write ON auth.webhook_deliveries
    FOR INSERT
    WITH CHECK (auth.current_user_role() = 'service_role');

COMMENT ON POLICY webhook_deliveries_service_write ON auth.webhook_deliveries IS 'Service role can create webhook delivery records.';

CREATE POLICY webhook_deliveries_admin_read ON auth.webhook_deliveries
    FOR SELECT
    USING (
        auth.current_user_role() = 'service_role'
        OR auth.current_user_role() = 'dashboard_admin'
        OR auth.is_admin()
    );

COMMENT ON POLICY webhook_deliveries_admin_read ON auth.webhook_deliveries IS 'Admins, dashboard admins, and service role can view webhook delivery logs.';

CREATE POLICY webhook_deliveries_service_update ON auth.webhook_deliveries
    FOR UPDATE
    USING (auth.current_user_role() = 'service_role')
    WITH CHECK (auth.current_user_role() = 'service_role');

COMMENT ON POLICY webhook_deliveries_service_update ON auth.webhook_deliveries IS 'Service role can update webhook delivery status.';

-- Webhook events
ALTER TABLE auth.webhook_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.webhook_events FORCE ROW LEVEL SECURITY;

CREATE POLICY webhook_events_admin_select ON auth.webhook_events
    FOR SELECT
    USING (auth.is_admin() OR auth.current_user_role() = 'dashboard_admin');

CREATE POLICY webhook_events_service ON auth.webhook_events
    FOR ALL
    USING (auth.current_user_role() = 'service_role');

-- Impersonation sessions
ALTER TABLE auth.impersonation_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.impersonation_sessions FORCE ROW LEVEL SECURITY;

CREATE POLICY impersonation_sessions_dashboard_admin_only ON auth.impersonation_sessions
    FOR ALL
    USING (
        auth.current_user_role() = 'service_role'
        OR auth.current_user_role() = 'dashboard_admin'
    );

COMMENT ON POLICY impersonation_sessions_dashboard_admin_only ON auth.impersonation_sessions IS 'Only dashboard admins and service role can access impersonation session records.';

-- ============================================================================
-- STORAGE SCHEMA RLS
-- ============================================================================

-- Storage buckets
ALTER TABLE storage.buckets ENABLE ROW LEVEL SECURITY;
ALTER TABLE storage.buckets FORCE ROW LEVEL SECURITY;

-- Admins and service roles can do everything with buckets
CREATE POLICY storage_buckets_admin ON storage.buckets
    FOR ALL
    USING (auth.current_user_role() IN ('dashboard_admin', 'service_role'))
    WITH CHECK (auth.current_user_role() IN ('dashboard_admin', 'service_role'));

COMMENT ON POLICY storage_buckets_admin ON storage.buckets IS 'Dashboard admins and service role have full access to all buckets';

-- Anyone can view public buckets (read-only)
CREATE POLICY storage_buckets_public_view ON storage.buckets
    FOR SELECT
    USING (public = true);

COMMENT ON POLICY storage_buckets_public_view ON storage.buckets IS 'Public buckets are visible to everyone (including unauthenticated users)';

-- Storage objects
ALTER TABLE storage.objects ENABLE ROW LEVEL SECURITY;
ALTER TABLE storage.objects FORCE ROW LEVEL SECURITY;

-- Admins and service roles can do everything with objects
CREATE POLICY storage_objects_admin ON storage.objects
    FOR ALL
    USING (auth.current_user_role() IN ('dashboard_admin', 'service_role'))
    WITH CHECK (auth.current_user_role() IN ('dashboard_admin', 'service_role'));

COMMENT ON POLICY storage_objects_admin ON storage.objects IS 'Dashboard admins and service role have full access to all objects';

-- Owners can do everything with their files
CREATE POLICY storage_objects_owner ON storage.objects
    FOR ALL
    USING (auth.current_user_id() = owner_id)
    WITH CHECK (auth.current_user_id() = owner_id);

COMMENT ON POLICY storage_objects_owner ON storage.objects IS 'Users can fully manage their own files';

-- Anyone can read files in public buckets (unauthenticated access allowed)
CREATE POLICY storage_objects_public_read ON storage.objects
    FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM storage.buckets
            WHERE buckets.id = objects.bucket_id
            AND buckets.public = true
        )
    );

COMMENT ON POLICY storage_objects_public_read ON storage.objects IS 'Files in public buckets are readable by everyone (including unauthenticated users)';

-- Users can read files shared with them (via object_permissions)
CREATE POLICY storage_objects_shared_read ON storage.objects
    FOR SELECT
    USING (
        auth.current_user_id() IS NOT NULL
        AND EXISTS (
            SELECT 1 FROM storage.object_permissions
            WHERE object_permissions.object_id = objects.id
            AND object_permissions.user_id = auth.current_user_id()
            AND object_permissions.permission IN ('read', 'write')
        )
    );

COMMENT ON POLICY storage_objects_shared_read ON storage.objects IS 'Users can read files that have been shared with them';

-- Users can update/delete files shared with them with write permission
CREATE POLICY storage_objects_shared_write ON storage.objects
    FOR UPDATE
    USING (
        auth.current_user_id() IS NOT NULL
        AND EXISTS (
            SELECT 1 FROM storage.object_permissions
            WHERE object_permissions.object_id = objects.id
            AND object_permissions.user_id = auth.current_user_id()
            AND object_permissions.permission = 'write'
        )
    );

COMMENT ON POLICY storage_objects_shared_write ON storage.objects IS 'Users can update files that have been shared with them with write permission';

CREATE POLICY storage_objects_shared_delete ON storage.objects
    FOR DELETE
    USING (
        auth.current_user_id() IS NOT NULL
        AND EXISTS (
            SELECT 1 FROM storage.object_permissions
            WHERE object_permissions.object_id = objects.id
            AND object_permissions.user_id = auth.current_user_id()
            AND object_permissions.permission = 'write'
        )
    );

COMMENT ON POLICY storage_objects_shared_delete ON storage.objects IS 'Users can delete files that have been shared with them with write permission';

-- Authenticated users can insert objects (owner_id will be set by application)
CREATE POLICY storage_objects_insert ON storage.objects
    FOR INSERT
    WITH CHECK (
        auth.current_user_role() IN ('dashboard_admin', 'service_role')
        OR (auth.current_user_id() IS NOT NULL AND auth.current_user_id() = owner_id)
    );

COMMENT ON POLICY storage_objects_insert ON storage.objects IS 'Users can upload files (owner_id must match their user ID). Public buckets allow READ but not WRITE for unauthenticated users.';

-- Storage object permissions
ALTER TABLE storage.object_permissions ENABLE ROW LEVEL SECURITY;
ALTER TABLE storage.object_permissions FORCE ROW LEVEL SECURITY;

-- Admins can manage all permissions
CREATE POLICY storage_object_permissions_admin ON storage.object_permissions
    FOR ALL
    USING (auth.current_user_role() IN ('dashboard_admin', 'service_role'))
    WITH CHECK (auth.current_user_role() IN ('dashboard_admin', 'service_role'));

COMMENT ON POLICY storage_object_permissions_admin ON storage.object_permissions IS 'Dashboard admins and service role can manage all file sharing permissions';

-- Owners can share their own files
CREATE POLICY storage_object_permissions_owner_manage ON storage.object_permissions
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM storage.objects
            WHERE objects.id = object_permissions.object_id
            AND objects.owner_id = auth.current_user_id()
        )
    )
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM storage.objects
            WHERE objects.id = object_permissions.object_id
            AND objects.owner_id = auth.current_user_id()
        )
    );

COMMENT ON POLICY storage_object_permissions_owner_manage ON storage.object_permissions IS 'File owners can manage sharing permissions for their files';

-- Users can view permissions for files shared with them
CREATE POLICY storage_object_permissions_view_shared ON storage.object_permissions
    FOR SELECT
    USING (
        user_id = auth.current_user_id()
    );

COMMENT ON POLICY storage_object_permissions_view_shared ON storage.object_permissions IS 'Users can view sharing permissions for files shared with them';

-- ============================================================================
-- CUSTOM RLS POLICY EXAMPLES
-- ============================================================================
--
-- Below are examples of custom RLS policies you can add to storage.objects
-- to implement specific access patterns for your application.
--
-- EXAMPLE 1: User folder restriction
-- Restrict users to only upload files to paths matching their user ID
-- Useful for user-uploads bucket where each user has their own folder
--
-- CREATE POLICY user_uploads_path_restriction ON storage.objects
--     FOR INSERT
--     WITH CHECK (
--         bucket_id = 'user-uploads'
--         AND path LIKE (auth.current_user_id()::TEXT || '/%')
--         AND owner_id = auth.current_user_id()
--     );
--
-- EXAMPLE 2: Role-based bucket access
-- Restrict certain buckets to users with specific roles
--
-- CREATE POLICY premium_content_access ON storage.objects
--     FOR SELECT
--     USING (
--         bucket_id = 'premium-content'
--         AND (
--             auth.current_user_role() IN ('dashboard_admin', 'service_role')
--             OR EXISTS (
--                 SELECT 1 FROM auth.users
--                 WHERE id = auth.current_user_id()
--                 AND user_metadata->>'subscription' = 'premium'
--             )
--         )
--     );
--
-- EXAMPLE 3: Read-only public bucket with admin-only writes
-- Allow everyone to read but only admins to write
--
-- CREATE POLICY public_assets_read_only ON storage.objects
--     FOR SELECT
--     USING (bucket_id = 'public-assets');
--
-- CREATE POLICY public_assets_admin_write ON storage.objects
--     FOR INSERT
--     WITH CHECK (
--         bucket_id = 'public-assets'
--         AND auth.current_user_role() = 'dashboard_admin'
--     );
--
-- EXAMPLE 4: Team/organization-based access
-- Allow users in the same organization to access each other's files
--
-- CREATE POLICY organization_files_access ON storage.objects
--     FOR SELECT
--     USING (
--         bucket_id = 'team-files'
--         AND EXISTS (
--             SELECT 1 FROM auth.users owner
--             JOIN auth.users viewer ON owner.user_metadata->>'org_id' = viewer.user_metadata->>'org_id'
--             WHERE owner.id = objects.owner_id
--             AND viewer.id = auth.current_user_id()
--         )
--     );
--
-- EXAMPLE 5: Time-based access restrictions
-- Only allow access to files during certain hours or after certain dates
--
-- CREATE POLICY scheduled_content_access ON storage.objects
--     FOR SELECT
--     USING (
--         bucket_id = 'scheduled-releases'
--         AND (
--             auth.current_user_role() = 'dashboard_admin'
--             OR (objects.metadata->>'release_date')::TIMESTAMPTZ <= NOW()
--         )
--     );
--

-- ============================================================================
-- STORAGE RLS FIXES - Prevent Infinite Recursion
-- ============================================================================

-- SECURITY DEFINER function to check if bucket is public
-- This prevents infinite recursion when RLS policies on storage.objects
-- reference storage.buckets (which also has RLS enabled)
CREATE OR REPLACE FUNCTION storage.is_bucket_public(bucket_name TEXT)
RETURNS BOOLEAN
LANGUAGE plpgsql
SECURITY DEFINER
STABLE
AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM storage.buckets
        WHERE id = bucket_name AND public = true
    );
END;
$$;

COMMENT ON FUNCTION storage.is_bucket_public IS
    'Check if a bucket is public, bypassing RLS to prevent infinite recursion';

-- SECURITY DEFINER function to check object permissions
-- This prevents infinite recursion when RLS policies on storage.objects
-- reference storage.object_permissions (which also has RLS enabled)
CREATE OR REPLACE FUNCTION storage.has_object_permission(
    p_object_id UUID,
    p_user_id UUID,
    p_permission TEXT
)
RETURNS BOOLEAN
LANGUAGE plpgsql
SECURITY DEFINER
STABLE
AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM storage.object_permissions
        WHERE object_id = p_object_id
        AND user_id = p_user_id
        AND (permission = p_permission OR (p_permission = 'read' AND permission = 'write'))
    );
END;
$$;

COMMENT ON FUNCTION storage.has_object_permission IS
    'Check if user has permission on object, bypassing RLS to prevent infinite recursion';

-- Supabase compatibility: storage.foldername() extracts folder path from object name
CREATE OR REPLACE FUNCTION storage.foldername(name TEXT)
RETURNS TEXT[] AS $$
DECLARE
    path_parts TEXT[];
    folder_parts TEXT[];
BEGIN
    IF name IS NULL OR name = '' THEN
        RETURN ARRAY[]::TEXT[];
    END IF;

    -- Split the path by '/' to get folder structure
    path_parts := string_to_array(name, '/');

    -- Remove the last element (filename) to get just folders
    IF array_length(path_parts, 1) > 1 THEN
        folder_parts := path_parts[1:array_length(path_parts, 1) - 1];
    ELSE
        -- No folders, just a filename at root
        folder_parts := ARRAY[]::TEXT[];
    END IF;

    RETURN folder_parts;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

COMMENT ON FUNCTION storage.foldername(TEXT) IS 'Supabase-compatible function that extracts folder path components from an object name/path. Returns array of folder names. Use [1] to get first folder, [2] for second, etc.';

-- Update storage.objects public read policy to use SECURITY DEFINER function
DROP POLICY IF EXISTS storage_objects_public_read ON storage.objects;
CREATE POLICY storage_objects_public_read ON storage.objects
    FOR SELECT
    USING (storage.is_bucket_public(bucket_id));

-- Update storage.objects shared read policy to use SECURITY DEFINER function
DROP POLICY IF EXISTS storage_objects_shared_read ON storage.objects;
CREATE POLICY storage_objects_shared_read ON storage.objects
    FOR SELECT
    USING (
        auth.current_user_id() IS NOT NULL
        AND storage.has_object_permission(id, auth.current_user_id(), 'read')
    );

-- Update storage.objects shared write policy to use SECURITY DEFINER function
DROP POLICY IF EXISTS storage_objects_shared_write ON storage.objects;
CREATE POLICY storage_objects_shared_write ON storage.objects
    FOR UPDATE
    USING (
        auth.current_user_id() IS NOT NULL
        AND storage.has_object_permission(id, auth.current_user_id(), 'write')
    );

-- Update storage.objects shared delete policy to use SECURITY DEFINER function
DROP POLICY IF EXISTS storage_objects_shared_delete ON storage.objects;
CREATE POLICY storage_objects_shared_delete ON storage.objects
    FOR DELETE
    USING (
        auth.current_user_id() IS NOT NULL
        AND storage.has_object_permission(id, auth.current_user_id(), 'write')
    );

-- Fix INSERT policy to prevent unauthenticated uploads to public buckets
-- Public buckets should allow READ but not WRITE for unauthenticated users
DROP POLICY IF EXISTS storage_objects_insert ON storage.objects;
CREATE POLICY storage_objects_insert ON storage.objects
    FOR INSERT
    WITH CHECK (
        auth.current_user_role() IN ('dashboard_admin', 'service_role')
        OR (auth.current_user_id() IS NOT NULL AND auth.current_user_id() = owner_id)
    );

-- ============================================================================
-- FUNCTIONS SCHEMA RLS
-- ============================================================================

-- Edge functions
ALTER TABLE functions.edge_functions ENABLE ROW LEVEL SECURITY;
ALTER TABLE functions.edge_functions FORCE ROW LEVEL SECURITY;

CREATE POLICY functions_edge_functions_policy ON functions.edge_functions
    FOR ALL
    USING (auth.current_user_role() = 'dashboard_admin');

-- Edge function triggers
ALTER TABLE functions.edge_function_triggers ENABLE ROW LEVEL SECURITY;
ALTER TABLE functions.edge_function_triggers FORCE ROW LEVEL SECURITY;

CREATE POLICY functions_edge_function_triggers_policy ON functions.edge_function_triggers
    FOR ALL
    USING (auth.current_user_role() = 'dashboard_admin');

-- Edge function executions
ALTER TABLE functions.edge_function_executions ENABLE ROW LEVEL SECURITY;
ALTER TABLE functions.edge_function_executions FORCE ROW LEVEL SECURITY;

CREATE POLICY functions_edge_function_executions_policy ON functions.edge_function_executions
    FOR ALL
    USING (auth.current_user_role() = 'dashboard_admin');

--
-- PERFORMANCE INDEXES FOR RLS POLICIES
--

-- Index for auth.api_keys RLS policy (filtering by user_id)
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON auth.api_keys(user_id);

-- Index for auth.api_key_usage RLS policy (filtering by api_key_id)
CREATE INDEX IF NOT EXISTS idx_api_key_usage_api_key_id ON auth.api_key_usage(api_key_id);

-- Index for auth.sessions RLS policy (filtering by user_id)
CREATE INDEX IF NOT EXISTS idx_auth_sessions_user_id ON auth.sessions(user_id);

-- Index for auth.webhook_deliveries RLS policy (filtering by webhook_id)
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_webhook_id ON auth.webhook_deliveries(webhook_id);

-- Indexes for auth.impersonation_sessions RLS policy (filtering by admin_user_id and impersonated_user_id)
CREATE INDEX IF NOT EXISTS idx_impersonation_sessions_admin_user_id ON auth.impersonation_sessions(admin_user_id);
CREATE INDEX IF NOT EXISTS idx_impersonation_sessions_impersonated_user_id ON auth.impersonation_sessions(impersonated_user_id);

--
-- GRANT PERMISSIONS TO FLUXBASE USER
--

GRANT ALL ON ALL TABLES IN SCHEMA auth TO CURRENT_USER;
GRANT ALL ON ALL TABLES IN SCHEMA dashboard TO CURRENT_USER;
GRANT ALL ON ALL TABLES IN SCHEMA functions TO CURRENT_USER;
GRANT ALL ON ALL TABLES IN SCHEMA storage TO CURRENT_USER;
GRANT ALL ON ALL TABLES IN SCHEMA realtime TO CURRENT_USER;
GRANT ALL ON ALL TABLES IN SCHEMA _fluxbase TO CURRENT_USER;
GRANT ALL ON ALL SEQUENCES IN SCHEMA realtime TO CURRENT_USER;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA auth TO CURRENT_USER;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO CURRENT_USER;

-- Grant default privileges for future objects
ALTER DEFAULT PRIVILEGES IN SCHEMA auth GRANT ALL ON TABLES TO CURRENT_USER;
ALTER DEFAULT PRIVILEGES IN SCHEMA dashboard GRANT ALL ON TABLES TO CURRENT_USER;
ALTER DEFAULT PRIVILEGES IN SCHEMA functions GRANT ALL ON TABLES TO CURRENT_USER;
ALTER DEFAULT PRIVILEGES IN SCHEMA storage GRANT ALL ON TABLES TO CURRENT_USER;
ALTER DEFAULT PRIVILEGES IN SCHEMA realtime GRANT ALL ON TABLES TO CURRENT_USER;
ALTER DEFAULT PRIVILEGES IN SCHEMA _fluxbase GRANT ALL ON TABLES TO CURRENT_USER;
ALTER DEFAULT PRIVILEGES IN SCHEMA realtime GRANT ALL ON SEQUENCES TO CURRENT_USER;
ALTER DEFAULT PRIVILEGES IN SCHEMA auth GRANT EXECUTE ON FUNCTIONS TO CURRENT_USER;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT EXECUTE ON FUNCTIONS TO CURRENT_USER;

-- NOTE: BYPASSRLS privilege is granted to CURRENT_USER in Makefile db-reset:
-- ALTER USER CURRENT_USER WITH BYPASSRLS;
-- This allows the application to manage all data and handle authorization at the application level.
-- RLS policies are still enforced for direct database connections and test users.

--
-- TABLES INTENTIONALLY WITHOUT RLS
--

COMMENT ON TABLE auth.service_keys IS 'Service keys for backend services. No RLS - only accessible via direct DB connection.';
COMMENT ON TABLE dashboard.schema_migrations IS 'Migration tracking table. No RLS needed - contains no sensitive data.';
COMMENT ON TABLE realtime.schema_registry IS 'Realtime schema registry. No RLS - managed by realtime service.';

--
-- INITIALIZE DEFAULT SETTINGS
--

-- Initialize authentication settings
INSERT INTO dashboard.system_settings (key, value, description) VALUES
    ('app.auth.enable_signup', '{"value": true}', 'Enable user signup'),
    ('app.auth.enable_magic_link', '{"value": true}', 'Enable magic link authentication'),
    ('app.auth.password_min_length', '{"value": 8}', 'Minimum password length'),
    ('app.auth.require_email_verification', '{"value": false}', 'Require email verification')
ON CONFLICT (key) DO NOTHING;

-- Initialize feature settings
INSERT INTO dashboard.system_settings (key, value, description) VALUES
    ('app.features.enable_realtime', '{"value": true}', 'Enable realtime features'),
    ('app.features.enable_storage', '{"value": true}', 'Enable storage features'),
    ('app.features.enable_functions', '{"value": true}', 'Enable edge functions')
ON CONFLICT (key) DO NOTHING;

-- Initialize email settings
INSERT INTO dashboard.system_settings (key, value, description) VALUES
    ('app.email.enabled', '{"value": false}', 'Enable email service'),
    ('app.email.provider', '{"value": "smtp"}', 'Email provider')
ON CONFLICT (key) DO NOTHING;

-- Initialize security settings
INSERT INTO dashboard.system_settings (key, value, description) VALUES
    ('app.security.enable_global_rate_limit', '{"value": false}', 'Enable global rate limiting')
ON CONFLICT (key) DO NOTHING;

-- Initialize default email templates
INSERT INTO dashboard.email_templates (template_type, subject, html_body, text_body, is_custom) VALUES
    (
        'magic_link',
        'Your Magic Link - Sign in to {{.AppName}}',
        '<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background-color: #f4f4f4; padding: 20px; border-radius: 5px;">
        <h1 style="color: #2c3e50; margin-bottom: 20px;">Sign in to {{.AppName}}</h1>
        <p>Click the button below to sign in to your account. This link will expire in 15 minutes.</p>
        <div style="text-align: center; margin: 30px 0;">
            <a href="{{.MagicLink}}" style="background-color: #3498db; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">Sign In</a>
        </div>
        <p style="color: #7f8c8d; font-size: 14px;">If you didn''t request this email, you can safely ignore it.</p>
        <p style="color: #7f8c8d; font-size: 14px;">If the button doesn''t work, copy and paste this link into your browser:</p>
        <p style="word-break: break-all; color: #3498db; font-size: 12px;">{{.MagicLink}}</p>
    </div>
</body>
</html>',
        'Sign in to {{.AppName}}

Click the link below to sign in to your account. This link will expire in 15 minutes.

{{.MagicLink}}

If you didn''t request this email, you can safely ignore it.',
        false
    ),
    (
        'email_verification',
        'Verify Your Email - {{.AppName}}',
        '<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background-color: #f4f4f4; padding: 20px; border-radius: 5px;">
        <h1 style="color: #2c3e50; margin-bottom: 20px;">Verify Your Email</h1>
        <p>Thank you for signing up for {{.AppName}}! Please verify your email address by clicking the button below.</p>
        <div style="text-align: center; margin: 30px 0;">
            <a href="{{.VerificationLink}}" style="background-color: #27ae60; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">Verify Email</a>
        </div>
        <p style="color: #7f8c8d; font-size: 14px;">This link will expire in 24 hours.</p>
        <p style="color: #7f8c8d; font-size: 14px;">If you didn''t create an account, you can safely ignore this email.</p>
        <p style="color: #7f8c8d; font-size: 14px;">If the button doesn''t work, copy and paste this link into your browser:</p>
        <p style="word-break: break-all; color: #3498db; font-size: 12px;">{{.VerificationLink}}</p>
    </div>
</body>
</html>',
        'Verify Your Email

Thank you for signing up for {{.AppName}}! Please verify your email address by clicking the link below.

{{.VerificationLink}}

This link will expire in 24 hours.

If you didn''t create an account, you can safely ignore this email.',
        false
    ),
    (
        'password_reset',
        'Reset Your Password - {{.AppName}}',
        '<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background-color: #f4f4f4; padding: 20px; border-radius: 5px;">
        <h1 style="color: #2c3e50; margin-bottom: 20px;">Reset Your Password</h1>
        <p>We received a request to reset your password for {{.AppName}}. Click the button below to create a new password.</p>
        <div style="text-align: center; margin: 30px 0;">
            <a href="{{.ResetLink}}" style="background-color: #e74c3c; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">Reset Password</a>
        </div>
        <p style="color: #7f8c8d; font-size: 14px;">This link will expire in 1 hour.</p>
        <p style="color: #7f8c8d; font-size: 14px;">If you didn''t request a password reset, you can safely ignore this email. Your password will not be changed.</p>
        <p style="color: #7f8c8d; font-size: 14px;">If the button doesn''t work, copy and paste this link into your browser:</p>
        <p style="word-break: break-all; color: #3498db; font-size: 12px;">{{.ResetLink}}</p>
    </div>
</body>
</html>',
        'Reset Your Password

We received a request to reset your password for {{.AppName}}. Click the link below to create a new password.

{{.ResetLink}}

This link will expire in 1 hour.

If you didn''t request a password reset, you can safely ignore this email. Your password will not be changed.',
        false
    )
ON CONFLICT (template_type) DO NOTHING;
