-- Fluxbase Initial Database Schema - Auth Functions
-- This file creates authentication and authorization helper functions

--
-- AUTHENTICATION HELPER FUNCTIONS
--

-- Get current user ID from session variable
CREATE OR REPLACE FUNCTION auth.current_user_id()
RETURNS UUID AS $$
DECLARE
    user_id_var TEXT;
BEGIN
    user_id_var := current_setting('app.user_id', true);
    IF user_id_var IS NULL OR user_id_var = '' THEN
        RETURN NULL;
    END IF;
    RETURN user_id_var::UUID;
EXCEPTION
    WHEN OTHERS THEN
        RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION auth.current_user_id() IS 'Returns the current authenticated user ID from PostgreSQL session variable app.user_id. Returns NULL if not set or invalid.';

-- Supabase compatibility: auth.uid() is an alias for auth.current_user_id()
CREATE OR REPLACE FUNCTION auth.uid()
RETURNS UUID AS $$
BEGIN
    RETURN auth.current_user_id();
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION auth.uid() IS 'Supabase-compatible alias for auth.current_user_id(). Returns the current authenticated user ID.';

-- Supabase compatibility: auth.jwt() returns JWT claims as JSONB
CREATE OR REPLACE FUNCTION auth.jwt()
RETURNS JSONB AS $$
DECLARE
    user_record RECORD;
    jwt_claims JSONB;
BEGIN
    -- Get the current user ID
    IF auth.current_user_id() IS NULL THEN
        RETURN NULL;
    END IF;

    -- Fetch user data including metadata
    SELECT
        id,
        email,
        role,
        user_metadata,
        app_metadata
    INTO user_record
    FROM auth.users
    WHERE id = auth.current_user_id();

    IF NOT FOUND THEN
        RETURN NULL;
    END IF;

    -- Build JWT claims object compatible with Supabase
    jwt_claims := jsonb_build_object(
        'sub', user_record.id,
        'email', user_record.email,
        'role', COALESCE(user_record.role, auth.current_user_role()),
        'user_metadata', COALESCE(user_record.user_metadata, '{}'::JSONB),
        'app_metadata', COALESCE(user_record.app_metadata, '{}'::JSONB)
    );

    RETURN jwt_claims;
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

COMMENT ON FUNCTION auth.jwt() IS 'Supabase-compatible function that returns JWT claims as JSONB, including user_metadata and app_metadata. Use ->> operator to extract text values or -> for JSONB.';

-- Supabase compatibility: auth.role() returns the current user's role
CREATE OR REPLACE FUNCTION auth.role()
RETURNS TEXT AS $$
BEGIN
    RETURN auth.current_user_role();
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION auth.role() IS 'Supabase-compatible alias for auth.current_user_role(). Returns the current user role.';

-- Get current user role from session variable
CREATE OR REPLACE FUNCTION auth.current_user_role()
RETURNS TEXT AS $$
DECLARE
    role_var TEXT;
BEGIN
    role_var := current_setting('app.role', true);
    IF role_var IS NULL OR role_var = '' THEN
        RETURN 'anon';
    END IF;
    RETURN role_var;
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION auth.current_user_role() IS 'Returns the current user role from PostgreSQL session variable app.role. Returns "anon" if not set.';

-- Check if user is authenticated
CREATE OR REPLACE FUNCTION auth.is_authenticated()
RETURNS BOOLEAN AS $$
BEGIN
    RETURN auth.current_user_id() IS NOT NULL;
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION auth.is_authenticated() IS 'Returns TRUE if a user is authenticated (user_id is set), FALSE for anonymous users.';

-- Check if user is admin
CREATE OR REPLACE FUNCTION auth.is_admin()
RETURNS BOOLEAN AS $$
BEGIN
    RETURN auth.current_user_role() = 'admin';
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION auth.is_admin() IS 'Returns TRUE if the current user role is "admin", FALSE otherwise.';

--
-- ROW LEVEL SECURITY HELPER FUNCTIONS
--

-- Enable RLS on a table
CREATE OR REPLACE FUNCTION auth.enable_rls(table_name TEXT, schema_name TEXT DEFAULT 'public')
RETURNS VOID AS $$
BEGIN
    EXECUTE format('ALTER TABLE %I.%I ENABLE ROW LEVEL SECURITY', schema_name, table_name);
    EXECUTE format('ALTER TABLE %I.%I FORCE ROW LEVEL SECURITY', schema_name, table_name);
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION auth.enable_rls(TEXT, TEXT) IS 'Enables Row Level Security on the specified table and forces it even for table owners.';

-- Disable RLS on a table
CREATE OR REPLACE FUNCTION auth.disable_rls(table_name TEXT, schema_name TEXT DEFAULT 'public')
RETURNS VOID AS $$
BEGIN
    EXECUTE format('ALTER TABLE %I.%I DISABLE ROW LEVEL SECURITY', schema_name, table_name);
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION auth.disable_rls(TEXT, TEXT) IS 'Disables Row Level Security on the specified table.';
