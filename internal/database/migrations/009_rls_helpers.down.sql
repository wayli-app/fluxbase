-- Rollback RLS Helper Functions

DROP FUNCTION IF EXISTS auth.disable_rls(TEXT, TEXT);
DROP FUNCTION IF EXISTS auth.enable_rls(TEXT, TEXT);
DROP FUNCTION IF EXISTS auth.is_authenticated();
DROP FUNCTION IF EXISTS auth.is_admin();
DROP FUNCTION IF EXISTS auth.current_user_role();
DROP FUNCTION IF EXISTS auth.current_user_id();
