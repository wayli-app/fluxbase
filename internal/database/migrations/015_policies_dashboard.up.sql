-- ============================================================================
-- DASHBOARD SCHEMA RLS
-- ============================================================================
-- This file contains all Row Level Security (RLS) policies for the Dashboard schema.
-- These policies control access to dashboard-related tables such as users, sessions,
-- settings, and templates.
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

-- Note: dashboard.system_settings has been migrated to app.settings
-- RLS policies for app.settings are in migration 014_policies_app

-- Note: dashboard.custom_settings has been migrated to app.settings
-- RLS policies for app.settings are in migration 014_policies_app

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

-- Note: dashboard.auth_settings has been migrated to app.settings
-- RLS policies for app.settings are in migration 014_policies_app
