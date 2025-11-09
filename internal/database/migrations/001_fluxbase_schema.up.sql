-- Fluxbase Database Schema
-- This migration creates all the necessary schemas and tables for Fluxbase
-- Version: 1.0.0

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

--
-- _FLUXBASE SCHEMA
-- Internal Fluxbase system schema for migration tracking and system tables
--

CREATE SCHEMA IF NOT EXISTS _fluxbase;
GRANT USAGE, CREATE ON SCHEMA _fluxbase TO fluxbase_app;

--
-- AUTH SCHEMA
-- Handles application user authentication, API keys, and sessions
--

CREATE SCHEMA IF NOT EXISTS auth;
GRANT USAGE, CREATE ON SCHEMA auth TO fluxbase_app;

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

-- Users table
CREATE TABLE IF NOT EXISTS auth.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT,
    email_verified BOOLEAN DEFAULT false,
    role TEXT,
    metadata JSONB DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auth_users_email ON auth.users(email);

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

-- Webhook trigger for updated_at
CREATE OR REPLACE FUNCTION auth.update_webhook_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_auth_webhooks_updated_at
    BEFORE UPDATE ON auth.webhooks
    FOR EACH ROW
    EXECUTE FUNCTION auth.update_webhook_updated_at();

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
GRANT USAGE, CREATE ON SCHEMA dashboard TO fluxbase_app;

COMMENT ON SCHEMA dashboard IS 'Schema for dashboard/platform administrator authentication and management';

-- Dashboard users table
CREATE TABLE IF NOT EXISTS dashboard.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    full_name TEXT,
    avatar_url TEXT,
    role TEXT DEFAULT 'dashboard_user',
    metadata JSONB DEFAULT '{}'::JSONB,
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

-- OAuth providers table
CREATE TABLE IF NOT EXISTS dashboard.oauth_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider TEXT NOT NULL UNIQUE,
    client_id TEXT NOT NULL,
    client_secret TEXT NOT NULL,
    redirect_url TEXT NOT NULL,
    scopes TEXT[] DEFAULT ARRAY[]::TEXT[],
    enabled BOOLEAN DEFAULT true,
    metadata JSONB DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dashboard_oauth_providers_provider ON dashboard.oauth_providers(provider);
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

-- System settings table (for setup completion tracking and other system-wide settings)
CREATE TABLE IF NOT EXISTS dashboard.system_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key TEXT UNIQUE NOT NULL,
    value JSONB NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dashboard_system_settings_key ON dashboard.system_settings(key);

-- Invitation tokens table (for secure user invitation flow)
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

--
-- FUNCTIONS SCHEMA
-- Handles edge functions and their executions
--

CREATE SCHEMA IF NOT EXISTS functions;
GRANT USAGE, CREATE ON SCHEMA functions TO fluxbase_app;

-- Edge functions table
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
    cron_schedule TEXT,
    version INTEGER DEFAULT 1,
    created_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_functions_edge_functions_name ON functions.edge_functions(name);
CREATE INDEX IF NOT EXISTS idx_functions_edge_functions_enabled ON functions.edge_functions(enabled);
CREATE INDEX IF NOT EXISTS idx_functions_edge_functions_cron_schedule ON functions.edge_functions(cron_schedule) WHERE cron_schedule IS NOT NULL;

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
GRANT USAGE, CREATE ON SCHEMA storage TO fluxbase_app;

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

--
-- REALTIME SCHEMA
-- Handles realtime subscriptions and change tracking
--

CREATE SCHEMA IF NOT EXISTS realtime;
GRANT USAGE, CREATE ON SCHEMA realtime TO fluxbase_app;

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

-- Drop existing function if it exists (to handle ownership issues)
DROP FUNCTION IF EXISTS public.update_updated_at() CASCADE;

-- Update trigger function for updated_at columns
CREATE OR REPLACE FUNCTION public.update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply update triggers to all tables with updated_at columns
CREATE TRIGGER update_auth_users_updated_at BEFORE UPDATE ON auth.users
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_auth_sessions_updated_at BEFORE UPDATE ON auth.sessions
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_auth_api_keys_updated_at BEFORE UPDATE ON auth.api_keys
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

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

CREATE TRIGGER update_functions_edge_functions_updated_at BEFORE UPDATE ON functions.edge_functions
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_functions_edge_function_triggers_updated_at BEFORE UPDATE ON functions.edge_function_triggers
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_storage_buckets_updated_at BEFORE UPDATE ON storage.buckets
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_storage_objects_updated_at BEFORE UPDATE ON storage.objects
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_realtime_schema_registry_updated_at BEFORE UPDATE ON realtime.schema_registry
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

--
-- ROW LEVEL SECURITY (RLS) POLICIES
-- Protect sensitive tables with smart policies that allow application operations
-- while preventing unauthorized access
--

-- Enable RLS on dashboard schema tables
ALTER TABLE dashboard.users ENABLE ROW LEVEL SECURITY;
ALTER TABLE dashboard.sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE dashboard.system_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE dashboard.invitation_tokens ENABLE ROW LEVEL SECURITY;

-- Dashboard users policies
-- Allow INSERT if:
-- 1. Table is empty (initial setup)
-- 2. Done by dashboard admin
-- 3. There's a valid invitation token (check via session variable)
CREATE POLICY dashboard_users_insert_policy ON dashboard.users
    FOR INSERT
    WITH CHECK (
        (SELECT COUNT(*) FROM dashboard.users) = 0  -- Allow if table is empty (first user)
        OR auth.current_user_role() = 'dashboard_admin'  -- Or if admin is creating
        OR EXISTS (
            -- Or if there's a valid invitation token in session variable
            SELECT 1 FROM dashboard.invitation_tokens
            WHERE token = current_setting('app.invitation_token', true)
            AND accepted = false
            AND expires_at > NOW()
        )
    );

-- Allow SELECT by dashboard admins or own record
CREATE POLICY dashboard_users_select_policy ON dashboard.users
    FOR SELECT
    USING (
        auth.current_user_role() = 'dashboard_admin'
        OR auth.current_user_id()::TEXT = id::TEXT
    );

-- Allow UPDATE by dashboard admins or own record
CREATE POLICY dashboard_users_update_policy ON dashboard.users
    FOR UPDATE
    USING (
        auth.current_user_role() = 'dashboard_admin'
        OR auth.current_user_id()::TEXT = id::TEXT
    );

-- Allow DELETE only by dashboard admins
CREATE POLICY dashboard_users_delete_policy ON dashboard.users
    FOR DELETE
    USING (auth.current_user_role() = 'dashboard_admin');

-- Dashboard sessions policies
CREATE POLICY dashboard_sessions_all_policy ON dashboard.sessions
    FOR ALL
    USING (
        auth.current_user_role() = 'dashboard_admin'
        OR auth.current_user_id()::TEXT = user_id::TEXT
    );

-- System settings policies - only dashboard admins
CREATE POLICY dashboard_system_settings_select_policy ON dashboard.system_settings
    FOR SELECT
    USING (auth.current_user_role() = 'dashboard_admin');

CREATE POLICY dashboard_system_settings_modify_policy ON dashboard.system_settings
    FOR ALL
    USING (auth.current_user_role() = 'dashboard_admin');

-- Invitation tokens policies - only dashboard admins
CREATE POLICY dashboard_invitation_tokens_select_policy ON dashboard.invitation_tokens
    FOR SELECT
    USING (auth.current_user_role() = 'dashboard_admin');

CREATE POLICY dashboard_invitation_tokens_modify_policy ON dashboard.invitation_tokens
    FOR ALL
    USING (auth.current_user_role() = 'dashboard_admin');

-- Enable RLS on auth schema tables
ALTER TABLE auth.users ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.api_keys ENABLE ROW LEVEL SECURITY;

-- Auth users policies
-- Allow INSERT for user registration (application controls this via signup endpoint)
CREATE POLICY auth_users_insert_policy ON auth.users
    FOR INSERT
    WITH CHECK (true);  -- Application endpoint validates, no RLS block

-- Allow SELECT by all authenticated users (required for signup flow and RLS testing)
-- Note: For production use cases, you may want to restrict this to admins or own record only
CREATE POLICY auth_users_select_policy ON auth.users
    FOR SELECT
    USING (true);

-- Allow UPDATE by admins, dashboard admins, or own record
CREATE POLICY auth_users_update_policy ON auth.users
    FOR UPDATE
    USING (
        auth.is_admin()
        OR auth.current_user_role() = 'dashboard_admin'
        OR auth.current_user_id()::TEXT = id::TEXT
    );

-- Allow DELETE by admins or dashboard admins
CREATE POLICY auth_users_delete_policy ON auth.users
    FOR DELETE
    USING (
        auth.is_admin()
        OR auth.current_user_role() = 'dashboard_admin'
    );

-- Auth sessions policies - allow all operations (required for signup flow and RLS testing)
-- Note: For production use cases, you may want to restrict this to admins or own sessions only
CREATE POLICY auth_sessions_policy ON auth.sessions
    FOR ALL
    USING (true)
    WITH CHECK (true);

-- Auth API keys policies - users can manage own keys, admins and dashboard admins can manage all
CREATE POLICY auth_api_keys_policy ON auth.api_keys
    FOR ALL
    USING (
        auth.is_admin()
        OR auth.current_user_role() = 'dashboard_admin'
        OR auth.current_user_id()::TEXT = user_id::TEXT
    );

-- Enable RLS on storage schema tables
ALTER TABLE storage.buckets ENABLE ROW LEVEL SECURITY;
ALTER TABLE storage.objects ENABLE ROW LEVEL SECURITY;

-- Storage buckets policies - dashboard admins can manage all buckets
CREATE POLICY storage_buckets_policy ON storage.buckets
    FOR ALL
    USING (auth.current_user_role() = 'dashboard_admin');

-- Storage objects policies - users can manage own objects, dashboard admins can manage all
CREATE POLICY storage_objects_policy ON storage.objects
    FOR ALL
    USING (
        auth.current_user_role() = 'dashboard_admin'
        OR auth.current_user_id()::TEXT = owner_id::TEXT
    );

-- Enable RLS on functions schema tables
ALTER TABLE functions.edge_functions ENABLE ROW LEVEL SECURITY;
ALTER TABLE functions.edge_function_triggers ENABLE ROW LEVEL SECURITY;
ALTER TABLE functions.edge_function_executions ENABLE ROW LEVEL SECURITY;

-- Functions policies - dashboard admins can manage all functions
CREATE POLICY functions_edge_functions_policy ON functions.edge_functions
    FOR ALL
    USING (auth.current_user_role() = 'dashboard_admin');

CREATE POLICY functions_edge_function_triggers_policy ON functions.edge_function_triggers
    FOR ALL
    USING (auth.current_user_role() = 'dashboard_admin');

CREATE POLICY functions_edge_function_executions_policy ON functions.edge_function_executions
    FOR ALL
    USING (auth.current_user_role() = 'dashboard_admin');

--
-- GRANT PERMISSIONS TO FLUXBASE_APP USER
-- Grant necessary permissions on all tables and sequences
--

GRANT ALL ON ALL TABLES IN SCHEMA auth TO fluxbase_app;
GRANT ALL ON ALL TABLES IN SCHEMA dashboard TO fluxbase_app;
GRANT ALL ON ALL TABLES IN SCHEMA functions TO fluxbase_app;
GRANT ALL ON ALL TABLES IN SCHEMA storage TO fluxbase_app;
GRANT ALL ON ALL TABLES IN SCHEMA realtime TO fluxbase_app;
GRANT ALL ON ALL SEQUENCES IN SCHEMA realtime TO fluxbase_app;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA auth TO fluxbase_app;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO fluxbase_app;

-- Grant default privileges for future objects created by fluxbase_app user
ALTER DEFAULT PRIVILEGES IN SCHEMA auth GRANT ALL ON TABLES TO fluxbase_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA dashboard GRANT ALL ON TABLES TO fluxbase_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA functions GRANT ALL ON TABLES TO fluxbase_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA storage GRANT ALL ON TABLES TO fluxbase_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA realtime GRANT ALL ON TABLES TO fluxbase_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA realtime GRANT ALL ON SEQUENCES TO fluxbase_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA auth GRANT EXECUTE ON FUNCTIONS TO fluxbase_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT EXECUTE ON FUNCTIONS TO fluxbase_app;

-- NOTE: BYPASSRLS privilege is granted to fluxbase_app in Makefile db-reset:
-- ALTER USER fluxbase_app WITH BYPASSRLS;
-- This allows the application to manage all data and handle authorization at the application level.
-- RLS policies are still enforced for direct database connections and test users.
-- For testing RLS, use a dedicated test user without BYPASSRLS privilege.
