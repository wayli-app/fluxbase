-- Rollback: Dashboard Authentication Schema

-- Drop triggers
DROP TRIGGER IF EXISTS update_dashboard_users_updated_at ON dashboard.users;

-- Drop functions
DROP FUNCTION IF EXISTS dashboard.cleanup_expired_tokens();
DROP FUNCTION IF EXISTS dashboard.update_updated_at_column();

-- Drop tables (in reverse order of dependencies)
DROP TABLE IF EXISTS dashboard.activity_log;
DROP TABLE IF EXISTS dashboard.sessions;
DROP TABLE IF EXISTS dashboard.password_reset_tokens;
DROP TABLE IF EXISTS dashboard.email_verification_tokens;
DROP TABLE IF EXISTS dashboard.users;

-- Drop schema
DROP SCHEMA IF EXISTS dashboard CASCADE;
