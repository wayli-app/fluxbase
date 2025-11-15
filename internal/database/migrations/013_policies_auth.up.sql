-- ============================================================================
-- AUTH SCHEMA RLS
-- ============================================================================
-- This file contains all Row Level Security (RLS) policies for the Auth schema.
-- These policies control access to authentication-related tables such as users,
-- sessions, API keys, OAuth, 2FA, webhooks, and impersonation.
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
