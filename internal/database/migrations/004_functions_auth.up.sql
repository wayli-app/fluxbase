-- Fluxbase Initial Database Schema - Auth Functions
-- This file creates authentication and authorization helper functions

--
-- AUTHENTICATION HELPER FUNCTIONS
--

-- Get current user ID from session variable
-- Uses Supabase-compatible request.jwt.claims format
CREATE OR REPLACE FUNCTION auth.current_user_id()
RETURNS UUID AS $$
DECLARE
    jwt_claims_var TEXT;
    user_id_var TEXT;
BEGIN
    -- Get user ID from request.jwt.claims (Supabase format)
    jwt_claims_var := current_setting('request.jwt.claims', true);
    IF jwt_claims_var IS NOT NULL AND jwt_claims_var <> '' THEN
        user_id_var := jwt_claims_var::json->>'sub';
        IF user_id_var IS NOT NULL AND user_id_var <> '' THEN
            RETURN user_id_var::UUID;
        END IF;
    END IF;

    RETURN NULL;
EXCEPTION
    WHEN OTHERS THEN
        RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION auth.current_user_id() IS 'Returns the current authenticated user ID from PostgreSQL session variable request.jwt.claims (Supabase format). Returns NULL if not set or invalid.';

-- Supabase compatibility: auth.uid() is an alias for auth.current_user_id()
CREATE OR REPLACE FUNCTION auth.uid()
RETURNS UUID AS $$
BEGIN
    RETURN auth.current_user_id();
EXCEPTION
    WHEN OTHERS THEN
        RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION auth.uid() IS 'Supabase-compatible alias for auth.current_user_id(). Returns the current authenticated user ID.';

-- Supabase compatibility: auth.jwt() returns JWT claims as JSONB
CREATE OR REPLACE FUNCTION auth.jwt()
RETURNS JSONB AS $$
DECLARE
    jwt_claims_var TEXT;
BEGIN
    -- Return request.jwt.claims (Supabase format)
    jwt_claims_var := current_setting('request.jwt.claims', true);
    IF jwt_claims_var IS NOT NULL AND jwt_claims_var <> '' THEN
        BEGIN
            RETURN jwt_claims_var::JSONB;
        EXCEPTION
            WHEN OTHERS THEN
                RETURN NULL;
        END;
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION auth.jwt() IS 'Supabase-compatible function that returns JWT claims as JSONB from request.jwt.claims session variable. Use ->> operator to extract text values or -> for JSONB.';

-- Supabase compatibility: auth.role() returns the current user's role
CREATE OR REPLACE FUNCTION auth.role()
RETURNS TEXT AS $$
BEGIN
    RETURN auth.current_user_role();
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION auth.role() IS 'Supabase-compatible alias for auth.current_user_role(). Returns the current user role.';

-- Get current user role from session variable
-- Uses Supabase-compatible request.jwt.claims format
CREATE OR REPLACE FUNCTION auth.current_user_role()
RETURNS TEXT AS $$
DECLARE
    jwt_claims_var TEXT;
    role_var TEXT;
BEGIN
    -- Get role from request.jwt.claims (Supabase format)
    jwt_claims_var := current_setting('request.jwt.claims', true);
    IF jwt_claims_var IS NOT NULL AND jwt_claims_var <> '' THEN
        role_var := jwt_claims_var::json->>'role';
        IF role_var IS NOT NULL AND role_var <> '' THEN
            RETURN role_var;
        END IF;
    END IF;

    -- Default to 'anon' if not set
    RETURN 'anon';
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION auth.current_user_role() IS 'Returns the current user role from PostgreSQL session variable request.jwt.claims (Supabase format). Returns "anon" if not set.';

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
