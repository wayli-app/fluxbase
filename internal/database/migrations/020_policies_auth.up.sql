-- ============================================================================
-- AUTH SCHEMA RLS
-- ============================================================================
-- This file contains all Row Level Security (RLS) policies for the Auth schema.
-- These policies control access to authentication-related tables such as users,
-- sessions, client keys, OAuth, 2FA, webhooks, and impersonation.
-- ============================================================================

-- Auth users table
-- RLS is ENABLED with service_role policies (Supabase-aligned security model)
-- Auth operations (signup, signin) use WrapWithServiceRole() which executes SET LOCAL ROLE service_role
-- This is equivalent to how Supabase's GoTrue uses supabase_auth_admin for privileged access
ALTER TABLE auth.users ENABLE ROW LEVEL SECURITY;

-- Service role has full access (for auth operations: signup, signin, password reset, admin tasks)
DROP POLICY IF EXISTS auth_users_service_role_all ON auth.users;
CREATE POLICY auth_users_service_role_all ON auth.users
    FOR ALL
    TO service_role
    USING (true)
    WITH CHECK (true);

COMMENT ON POLICY auth_users_service_role_all ON auth.users IS 'Service role has full access for auth operations (signup, signin, admin). Equivalent to Supabase supabase_auth_admin.';

-- Authenticated users can see their own record, admins can see all
DROP POLICY IF EXISTS auth_users_select_own ON auth.users;
CREATE POLICY auth_users_select_own ON auth.users
    FOR SELECT
    TO authenticated
    USING (
        id = auth.current_user_id()
        OR auth.is_admin()
        OR auth.current_user_role() = 'dashboard_admin'
    );

COMMENT ON POLICY auth_users_select_own ON auth.users IS 'Users can only see their own record. Admins and dashboard admins can see all users.';

-- Authenticated users can update their own record, admins can update any
DROP POLICY IF EXISTS auth_users_update_own ON auth.users;
CREATE POLICY auth_users_update_own ON auth.users
    FOR UPDATE
    TO authenticated
    USING (
        auth.is_admin()
        OR auth.current_user_role() = 'dashboard_admin'
        OR auth.current_user_id()::TEXT = id::TEXT
    )
    WITH CHECK (
        auth.is_admin()
        OR auth.current_user_role() = 'dashboard_admin'
        OR auth.current_user_id()::TEXT = id::TEXT
    );

COMMENT ON POLICY auth_users_update_own ON auth.users IS 'Users can update their own record. Admins and dashboard admins can update any user.';

-- Only admins can delete users
DROP POLICY IF EXISTS auth_users_delete_admin ON auth.users;
CREATE POLICY auth_users_delete_admin ON auth.users
    FOR DELETE
    TO authenticated
    USING (
        auth.is_admin()
        OR auth.current_user_role() = 'dashboard_admin'
    );

COMMENT ON POLICY auth_users_delete_admin ON auth.users IS 'Only admins and dashboard admins can delete user records.';

-- Auth sessions table
-- RLS is ENABLED with service_role policies (Supabase-aligned security model)
-- Session creation during signup/signin uses WrapWithServiceRole() with SET LOCAL ROLE service_role
ALTER TABLE auth.sessions ENABLE ROW LEVEL SECURITY;

-- Service role has full access (for auth operations: session creation, cleanup, admin tasks)
DROP POLICY IF EXISTS auth_sessions_service_role_all ON auth.sessions;
CREATE POLICY auth_sessions_service_role_all ON auth.sessions
    FOR ALL
    TO service_role
    USING (true)
    WITH CHECK (true);

COMMENT ON POLICY auth_sessions_service_role_all ON auth.sessions IS 'Service role has full access for session management (creation during signin/signup, cleanup, admin tasks).';

-- Authenticated users can view their own sessions, admins can view all
DROP POLICY IF EXISTS auth_sessions_select_own ON auth.sessions;
CREATE POLICY auth_sessions_select_own ON auth.sessions
    FOR SELECT
    TO authenticated
    USING (
        user_id = auth.current_user_id()
        OR auth.current_user_role() = 'dashboard_admin'
    );

COMMENT ON POLICY auth_sessions_select_own ON auth.sessions IS 'Users can view their own sessions. Dashboard admins can view all sessions.';

-- Authenticated users can update their own sessions (e.g., last_accessed_at)
DROP POLICY IF EXISTS auth_sessions_update_own ON auth.sessions;
CREATE POLICY auth_sessions_update_own ON auth.sessions
    FOR UPDATE
    TO authenticated
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());

COMMENT ON POLICY auth_sessions_update_own ON auth.sessions IS 'Users can update their own sessions (e.g., refresh token rotation).';

-- Authenticated users can delete their own sessions (logout), admins can delete any
DROP POLICY IF EXISTS auth_sessions_delete_own ON auth.sessions;
CREATE POLICY auth_sessions_delete_own ON auth.sessions
    FOR DELETE
    TO authenticated
    USING (
        user_id = auth.current_user_id()
        OR auth.current_user_role() = 'dashboard_admin'
    );

COMMENT ON POLICY auth_sessions_delete_own ON auth.sessions IS 'Users can delete their own sessions (logout). Dashboard admins can delete any session.';

-- Auth client keys table
ALTER TABLE auth.api_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.api_keys FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS auth_api_keys_policy ON auth.api_keys;
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

DROP POLICY IF EXISTS api_key_usage_service_write ON auth.api_key_usage;
CREATE POLICY api_key_usage_service_write ON auth.api_key_usage
    FOR INSERT
    WITH CHECK (auth.current_user_role() = 'service_role');

COMMENT ON POLICY api_key_usage_service_write ON auth.api_key_usage IS 'Service role can record API key usage.';

DROP POLICY IF EXISTS api_key_usage_user_read ON auth.api_key_usage;
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

COMMENT ON POLICY api_key_usage_user_read ON auth.api_key_usage IS 'Users can view usage for their own client keys. Admins can view all usage.';

-- Auth magic links
ALTER TABLE auth.magic_links ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.magic_links FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS magic_links_service_only ON auth.magic_links;
CREATE POLICY magic_links_service_only ON auth.magic_links
    FOR ALL
    USING (auth.current_user_role() = 'service_role');

COMMENT ON POLICY magic_links_service_only ON auth.magic_links IS 'Only service role can access magic links (used internally for auth flow).';

-- Auth password reset tokens
ALTER TABLE auth.password_reset_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.password_reset_tokens FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS password_reset_tokens_service_only ON auth.password_reset_tokens;
CREATE POLICY password_reset_tokens_service_only ON auth.password_reset_tokens
    FOR ALL
    USING (auth.current_user_role() = 'service_role');

COMMENT ON POLICY password_reset_tokens_service_only ON auth.password_reset_tokens IS 'Only service role can access password reset tokens (used internally for password reset flow).';

-- Auth token blacklist
ALTER TABLE auth.token_blacklist ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.token_blacklist FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS token_blacklist_admin_only ON auth.token_blacklist;
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

DROP POLICY IF EXISTS oauth_links_select ON auth.oauth_links;
CREATE POLICY oauth_links_select ON auth.oauth_links
    FOR SELECT
    USING (user_id = auth.current_user_id());

DROP POLICY IF EXISTS oauth_links_service_all ON auth.oauth_links;
CREATE POLICY oauth_links_service_all ON auth.oauth_links
    FOR ALL
    USING (auth.current_user_role() = 'service_role');

-- OAuth tokens
ALTER TABLE auth.oauth_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.oauth_tokens FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS oauth_tokens_select ON auth.oauth_tokens;
CREATE POLICY oauth_tokens_select ON auth.oauth_tokens
    FOR SELECT
    USING (user_id = auth.current_user_id());

DROP POLICY IF EXISTS oauth_tokens_service_all ON auth.oauth_tokens;
CREATE POLICY oauth_tokens_service_all ON auth.oauth_tokens
    FOR ALL
    USING (auth.current_user_role() = 'service_role');

-- 2FA setups
ALTER TABLE auth.two_factor_setups ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.two_factor_setups FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS two_factor_setups_select ON auth.two_factor_setups;
CREATE POLICY two_factor_setups_select ON auth.two_factor_setups
    FOR SELECT
    USING (user_id = auth.current_user_id());

DROP POLICY IF EXISTS two_factor_setups_insert ON auth.two_factor_setups;
CREATE POLICY two_factor_setups_insert ON auth.two_factor_setups
    FOR INSERT
    WITH CHECK (user_id = auth.current_user_id());

DROP POLICY IF EXISTS two_factor_setups_delete ON auth.two_factor_setups;
CREATE POLICY two_factor_setups_delete ON auth.two_factor_setups
    FOR DELETE
    USING (user_id = auth.current_user_id());

DROP POLICY IF EXISTS two_factor_setups_update ON auth.two_factor_setups;
CREATE POLICY two_factor_setups_update ON auth.two_factor_setups
    FOR UPDATE
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());

DROP POLICY IF EXISTS two_factor_setups_admin_select ON auth.two_factor_setups;
CREATE POLICY two_factor_setups_admin_select ON auth.two_factor_setups
    FOR SELECT
    USING (auth.is_admin() OR auth.current_user_role() = 'dashboard_admin');

-- 2FA recovery attempts
ALTER TABLE auth.two_factor_recovery_attempts ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.two_factor_recovery_attempts FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS two_factor_recovery_select ON auth.two_factor_recovery_attempts;
CREATE POLICY two_factor_recovery_select ON auth.two_factor_recovery_attempts
    FOR SELECT
    USING (user_id = auth.current_user_id());

DROP POLICY IF EXISTS two_factor_recovery_insert ON auth.two_factor_recovery_attempts;
CREATE POLICY two_factor_recovery_insert ON auth.two_factor_recovery_attempts
    FOR INSERT
    WITH CHECK (
        -- Allow service role to log all attempts (for backend logging)
        auth.current_user_role() = 'service_role'
        -- Allow users to log their own attempts (for client-side logging)
        OR user_id = auth.current_user_id()
    );

DROP POLICY IF EXISTS two_factor_recovery_admin_select ON auth.two_factor_recovery_attempts;
CREATE POLICY two_factor_recovery_admin_select ON auth.two_factor_recovery_attempts
    FOR SELECT
    USING (auth.is_admin() OR auth.current_user_role() = 'dashboard_admin');

-- Webhooks
ALTER TABLE auth.webhooks ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.webhooks FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS webhooks_admin_only ON auth.webhooks;
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

DROP POLICY IF EXISTS webhook_deliveries_service_write ON auth.webhook_deliveries;
CREATE POLICY webhook_deliveries_service_write ON auth.webhook_deliveries
    FOR INSERT
    WITH CHECK (auth.current_user_role() = 'service_role');

COMMENT ON POLICY webhook_deliveries_service_write ON auth.webhook_deliveries IS 'Service role can create webhook delivery records.';

DROP POLICY IF EXISTS webhook_deliveries_admin_read ON auth.webhook_deliveries;
CREATE POLICY webhook_deliveries_admin_read ON auth.webhook_deliveries
    FOR SELECT
    USING (
        auth.current_user_role() = 'service_role'
        OR auth.current_user_role() = 'dashboard_admin'
        OR auth.is_admin()
    );

COMMENT ON POLICY webhook_deliveries_admin_read ON auth.webhook_deliveries IS 'Admins, dashboard admins, and service role can view webhook delivery logs.';

DROP POLICY IF EXISTS webhook_deliveries_service_update ON auth.webhook_deliveries;
CREATE POLICY webhook_deliveries_service_update ON auth.webhook_deliveries
    FOR UPDATE
    USING (auth.current_user_role() = 'service_role')
    WITH CHECK (auth.current_user_role() = 'service_role');

COMMENT ON POLICY webhook_deliveries_service_update ON auth.webhook_deliveries IS 'Service role can update webhook delivery status.';

-- Webhook events
ALTER TABLE auth.webhook_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.webhook_events FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS webhook_events_admin_select ON auth.webhook_events;
CREATE POLICY webhook_events_admin_select ON auth.webhook_events
    FOR SELECT
    USING (auth.is_admin() OR auth.current_user_role() = 'dashboard_admin');

DROP POLICY IF EXISTS webhook_events_service ON auth.webhook_events;
CREATE POLICY webhook_events_service ON auth.webhook_events
    FOR ALL
    USING (auth.current_user_role() = 'service_role');

-- Impersonation sessions
ALTER TABLE auth.impersonation_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.impersonation_sessions FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS impersonation_sessions_dashboard_admin_only ON auth.impersonation_sessions;
CREATE POLICY impersonation_sessions_dashboard_admin_only ON auth.impersonation_sessions
    FOR ALL
    USING (
        auth.current_user_role() = 'service_role'
        OR auth.current_user_role() = 'dashboard_admin'
    );

COMMENT ON POLICY impersonation_sessions_dashboard_admin_only ON auth.impersonation_sessions IS 'Only dashboard admins and service role can access impersonation session records.';

-- RLS audit log policies
ALTER TABLE auth.rls_audit_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.rls_audit_log FORCE ROW LEVEL SECURITY;

-- Policy: Service role can insert audit logs (for system logging)
DROP POLICY IF EXISTS rls_audit_log_service_insert ON auth.rls_audit_log;
CREATE POLICY rls_audit_log_service_insert ON auth.rls_audit_log
    FOR INSERT
    TO authenticated
    WITH CHECK (auth.current_user_role() = 'service_role');

-- Policy: Admins can view all audit logs (for security monitoring)
DROP POLICY IF EXISTS rls_audit_log_admin_select ON auth.rls_audit_log;
CREATE POLICY rls_audit_log_admin_select ON auth.rls_audit_log
    FOR SELECT
    TO authenticated
    USING (auth.current_user_role() IN ('admin', 'dashboard_admin', 'service_role'));

-- Policy: Users can view their own audit logs (for transparency)
DROP POLICY IF EXISTS rls_audit_log_user_select ON auth.rls_audit_log;
CREATE POLICY rls_audit_log_user_select ON auth.rls_audit_log
    FOR SELECT
    TO authenticated
    USING (auth.current_user_id() = user_id);

COMMENT ON POLICY rls_audit_log_service_insert ON auth.rls_audit_log IS 'Only service role can insert audit log entries to prevent users from tampering with logs.';
COMMENT ON POLICY rls_audit_log_admin_select ON auth.rls_audit_log IS 'Admins can view all audit logs for security monitoring and compliance.';
COMMENT ON POLICY rls_audit_log_user_select ON auth.rls_audit_log IS 'Users can view their own audit log entries for transparency.';

-- MFA factors
ALTER TABLE auth.mfa_factors ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.mfa_factors FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS mfa_factors_select_own ON auth.mfa_factors;
CREATE POLICY mfa_factors_select_own ON auth.mfa_factors
    FOR SELECT
    USING (user_id = auth.current_user_id());

DROP POLICY IF EXISTS mfa_factors_insert_own ON auth.mfa_factors;
CREATE POLICY mfa_factors_insert_own ON auth.mfa_factors
    FOR INSERT
    WITH CHECK (user_id = auth.current_user_id());

DROP POLICY IF EXISTS mfa_factors_update_own ON auth.mfa_factors;
CREATE POLICY mfa_factors_update_own ON auth.mfa_factors
    FOR UPDATE
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());

DROP POLICY IF EXISTS mfa_factors_delete_own ON auth.mfa_factors;
CREATE POLICY mfa_factors_delete_own ON auth.mfa_factors
    FOR DELETE
    USING (user_id = auth.current_user_id());

DROP POLICY IF EXISTS mfa_factors_admin_all ON auth.mfa_factors;
CREATE POLICY mfa_factors_admin_all ON auth.mfa_factors
    FOR ALL
    USING (
        auth.is_admin()
        OR auth.current_user_role() = 'dashboard_admin'
        OR auth.current_user_role() = 'service_role'
    );

COMMENT ON POLICY mfa_factors_select_own ON auth.mfa_factors IS 'Users can view their own MFA factors.';
COMMENT ON POLICY mfa_factors_insert_own ON auth.mfa_factors IS 'Users can create their own MFA factors.';
COMMENT ON POLICY mfa_factors_update_own ON auth.mfa_factors IS 'Users can update their own MFA factors.';
COMMENT ON POLICY mfa_factors_delete_own ON auth.mfa_factors IS 'Users can delete their own MFA factors.';
COMMENT ON POLICY mfa_factors_admin_all ON auth.mfa_factors IS 'Admins, dashboard admins, and service role can manage all MFA factors.';
