-- Restore foreign key constraint on admin_user_id
-- This will fail if there are rows with admin_user_id not in auth.users

ALTER TABLE auth.impersonation_sessions
    ADD CONSTRAINT impersonation_sessions_admin_user_id_fkey
    FOREIGN KEY (admin_user_id) REFERENCES auth.users(id) ON DELETE CASCADE;
