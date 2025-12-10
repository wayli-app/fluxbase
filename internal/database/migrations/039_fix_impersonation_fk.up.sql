-- Remove foreign key constraint on admin_user_id
-- This allows dashboard.users (not just auth.users) to create impersonation sessions

ALTER TABLE auth.impersonation_sessions
    DROP CONSTRAINT IF EXISTS impersonation_sessions_admin_user_id_fkey;

-- Note: We intentionally don't add a new FK because admin_user_id can reference
-- either dashboard.users OR auth.users (for users with dashboard_admin role)
