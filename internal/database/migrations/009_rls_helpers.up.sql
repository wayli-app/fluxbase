-- Migration: RLS Helper Functions
-- This migration provides helper functions for Row Level Security (RLS)

-- Create PostgreSQL roles for RLS (if they don't exist)
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'anon') THEN
        CREATE ROLE anon NOLOGIN;
    END IF;
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'authenticated') THEN
        CREATE ROLE authenticated NOLOGIN;
    END IF;
END
$$;

-- Grant necessary permissions to roles
GRANT USAGE ON SCHEMA auth TO anon;
GRANT USAGE ON SCHEMA auth TO authenticated;
GRANT USAGE ON SCHEMA public TO anon;
GRANT USAGE ON SCHEMA public TO authenticated;

-- Function to get current user ID from session variable
CREATE OR REPLACE FUNCTION auth.current_user_id()
RETURNS UUID AS $$
DECLARE
    user_id_text TEXT;
BEGIN
    user_id_text := current_setting('app.user_id', TRUE);
    IF user_id_text IS NULL OR user_id_text = '' THEN
        RETURN NULL;
    END IF;
    RETURN user_id_text::UUID;
EXCEPTION
    WHEN OTHERS THEN
        RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

COMMENT ON FUNCTION auth.current_user_id() IS
'Returns the current authenticated user ID from PostgreSQL session variable app.user_id. Returns NULL if not set or invalid.';

-- Function to get current user role from session variable
CREATE OR REPLACE FUNCTION auth.current_user_role()
RETURNS TEXT AS $$
DECLARE
    role_text TEXT;
BEGIN
    role_text := current_setting('app.role', TRUE);
    IF role_text IS NULL OR role_text = '' THEN
        RETURN 'anon';
    END IF;
    RETURN role_text;
EXCEPTION
    WHEN OTHERS THEN
        RETURN 'anon';
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

COMMENT ON FUNCTION auth.current_user_role() IS
'Returns the current user role from PostgreSQL session variable app.role. Returns "anon" if not set.';

-- Function to check if current user is admin
CREATE OR REPLACE FUNCTION auth.is_admin()
RETURNS BOOLEAN AS $$
BEGIN
    RETURN auth.current_user_role() = 'admin';
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

COMMENT ON FUNCTION auth.is_admin() IS
'Returns TRUE if the current user role is "admin", FALSE otherwise.';

-- Function to check if current user is authenticated
CREATE OR REPLACE FUNCTION auth.is_authenticated()
RETURNS BOOLEAN AS $$
BEGIN
    RETURN auth.current_user_id() IS NOT NULL;
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

COMMENT ON FUNCTION auth.is_authenticated() IS
'Returns TRUE if a user is authenticated (user_id is set), FALSE for anonymous users.';

-- Function to enable RLS on a table
CREATE OR REPLACE FUNCTION auth.enable_rls(table_name TEXT, schema_name TEXT DEFAULT 'public')
RETURNS VOID AS $$
BEGIN
    EXECUTE format('ALTER TABLE %I.%I ENABLE ROW LEVEL SECURITY', schema_name, table_name);
    EXECUTE format('ALTER TABLE %I.%I FORCE ROW LEVEL SECURITY', schema_name, table_name);
    RAISE NOTICE 'RLS enabled on %.%', schema_name, table_name;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION auth.enable_rls(TEXT, TEXT) IS
'Enables Row Level Security on the specified table and forces it even for table owners.';

-- Function to disable RLS on a table
CREATE OR REPLACE FUNCTION auth.disable_rls(table_name TEXT, schema_name TEXT DEFAULT 'public')
RETURNS VOID AS $$
BEGIN
    EXECUTE format('ALTER TABLE %I.%I DISABLE ROW LEVEL SECURITY', schema_name, table_name);
    RAISE NOTICE 'RLS disabled on %.%', schema_name, table_name;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION auth.disable_rls(TEXT, TEXT) IS
'Disables Row Level Security on the specified table.';

-- Example RLS policies for a multi-tenant table
-- This is commented out but serves as a template for users

/*
-- Example: Enable RLS on a table
SELECT auth.enable_rls('tasks');

-- Policy: Users can only see their own tasks
CREATE POLICY tasks_select_policy ON public.tasks
    FOR SELECT
    USING (user_id = auth.current_user_id() OR auth.is_admin());

-- Policy: Users can only insert tasks for themselves
CREATE POLICY tasks_insert_policy ON public.tasks
    FOR INSERT
    WITH CHECK (user_id = auth.current_user_id());

-- Policy: Users can only update their own tasks
CREATE POLICY tasks_update_policy ON public.tasks
    FOR UPDATE
    USING (user_id = auth.current_user_id() OR auth.is_admin())
    WITH CHECK (user_id = auth.current_user_id() OR auth.is_admin());

-- Policy: Users can only delete their own tasks
CREATE POLICY tasks_delete_policy ON public.tasks
    FOR DELETE
    USING (user_id = auth.current_user_id() OR auth.is_admin());

-- Policy: Allow anonymous read access to public tasks
CREATE POLICY tasks_anon_select_policy ON public.tasks
    FOR SELECT
    USING (is_public = TRUE);
*/

-- Grant execute permissions to authenticated users
GRANT EXECUTE ON FUNCTION auth.current_user_id() TO authenticated;
GRANT EXECUTE ON FUNCTION auth.current_user_role() TO authenticated;
GRANT EXECUTE ON FUNCTION auth.is_admin() TO authenticated;
GRANT EXECUTE ON FUNCTION auth.is_authenticated() TO authenticated;

-- Grant execute permissions to anonymous users for read-only functions
GRANT EXECUTE ON FUNCTION auth.current_user_id() TO anon;
GRANT EXECUTE ON FUNCTION auth.current_user_role() TO anon;
GRANT EXECUTE ON FUNCTION auth.is_authenticated() TO anon;
