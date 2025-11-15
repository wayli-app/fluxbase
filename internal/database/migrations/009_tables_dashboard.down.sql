-- Drop dashboard tables in reverse dependency order
DROP TABLE IF EXISTS dashboard.email_templates;
DROP TABLE IF EXISTS dashboard.schema_migrations;
DROP TABLE IF EXISTS dashboard.invitation_tokens;
DROP TABLE IF EXISTS dashboard.oauth_providers;
DROP TABLE IF EXISTS dashboard.activity_log;
DROP TABLE IF EXISTS dashboard.email_verification_tokens;
DROP TABLE IF EXISTS dashboard.password_reset_tokens;
DROP TABLE IF EXISTS dashboard.sessions;
DROP TABLE IF EXISTS dashboard.users;
