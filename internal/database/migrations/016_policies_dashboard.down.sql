-- Drop all dashboard schema RLS policies

-- Note: dashboard.auth_settings has been migrated to app.settings
DROP POLICY IF EXISTS oauth_providers_dashboard_admin_only ON dashboard.oauth_providers;
DROP POLICY IF EXISTS activity_log_admin_read ON dashboard.activity_log;
DROP POLICY IF EXISTS activity_log_service_write ON dashboard.activity_log;
DROP POLICY IF EXISTS dashboard_email_verification_service_only ON dashboard.email_verification_tokens;
DROP POLICY IF EXISTS dashboard_password_reset_service_only ON dashboard.password_reset_tokens;
DROP POLICY IF EXISTS dashboard_email_templates_modify_policy ON dashboard.email_templates;
DROP POLICY IF EXISTS dashboard_email_templates_select_policy ON dashboard.email_templates;
DROP POLICY IF EXISTS dashboard_invitation_tokens_modify_policy ON dashboard.invitation_tokens;
DROP POLICY IF EXISTS dashboard_invitation_tokens_select_policy ON dashboard.invitation_tokens;
-- Note: dashboard.custom_settings has been migrated to app.settings
-- Note: dashboard.system_settings has been migrated to app.settings
DROP POLICY IF EXISTS dashboard_sessions_all_policy ON dashboard.sessions;
DROP POLICY IF EXISTS dashboard_users_delete_policy ON dashboard.users;
DROP POLICY IF EXISTS dashboard_users_update_policy ON dashboard.users;
DROP POLICY IF EXISTS dashboard_users_select_policy ON dashboard.users;
DROP POLICY IF EXISTS dashboard_users_insert_policy ON dashboard.users;
