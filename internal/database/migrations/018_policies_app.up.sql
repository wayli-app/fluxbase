--
-- APP SCHEMA RLS POLICIES
-- Row Level Security policies for application settings
--

-- Enable RLS on app.settings
ALTER TABLE app.settings ENABLE ROW LEVEL SECURITY;

-- Allow service_role to do everything (bypasses RLS anyway, but explicit is good)
DROP POLICY IF EXISTS "Service role has full access to app settings" ON app.settings;
CREATE POLICY "Service role has full access to app settings"
    ON app.settings
    FOR ALL
    TO service_role
    USING (true)
    WITH CHECK (true);

-- Allow public/anon users to read public settings
DROP POLICY IF EXISTS "Public settings are readable by anyone" ON app.settings;
CREATE POLICY "Public settings are readable by anyone"
    ON app.settings
    FOR SELECT
    TO anon, authenticated
    USING (is_public = true AND is_secret = false);

-- Allow authenticated users to read non-secret settings
DROP POLICY IF EXISTS "Authenticated users can read non-secret settings" ON app.settings;
CREATE POLICY "Authenticated users can read non-secret settings"
    ON app.settings
    FOR SELECT
    TO authenticated
    USING (is_secret = false);

-- Settings write policies: Only roles in editable_by array can modify settings
DROP POLICY IF EXISTS "Settings can be created by authorized roles" ON app.settings;
CREATE POLICY "Settings can be created by authorized roles"
    ON app.settings
    FOR INSERT
    TO authenticated
    WITH CHECK (
        auth.current_user_role() = 'service_role'
        OR auth.current_user_role() = ANY(editable_by)
    );

DROP POLICY IF EXISTS "Settings can be updated by authorized roles" ON app.settings;
CREATE POLICY "Settings can be updated by authorized roles"
    ON app.settings
    FOR UPDATE
    TO authenticated
    USING (
        auth.current_user_role() = 'service_role'
        OR auth.current_user_role() = ANY(editable_by)
    )
    WITH CHECK (
        auth.current_user_role() = 'service_role'
        OR auth.current_user_role() = ANY(editable_by)
    );

DROP POLICY IF EXISTS "Settings can be deleted by authorized roles" ON app.settings;
CREATE POLICY "Settings can be deleted by authorized roles"
    ON app.settings
    FOR DELETE
    TO authenticated
    USING (
        auth.current_user_role() = 'service_role'
        OR auth.current_user_role() = ANY(editable_by)
    );

COMMENT ON POLICY "Service role has full access to app settings" ON app.settings
    IS 'Service role can manage all settings';
COMMENT ON POLICY "Public settings are readable by anyone" ON app.settings
    IS 'Public, non-secret settings can be read by anonymous and authenticated users';
COMMENT ON POLICY "Authenticated users can read non-secret settings" ON app.settings
    IS 'Authenticated users can read all non-secret settings regardless of public flag';
COMMENT ON POLICY "Settings can be created by authorized roles" ON app.settings
    IS 'Only users with roles listed in the editable_by array can create settings';
COMMENT ON POLICY "Settings can be updated by authorized roles" ON app.settings
    IS 'Only users with roles listed in the editable_by array can update settings';
COMMENT ON POLICY "Settings can be deleted by authorized roles" ON app.settings
    IS 'Only users with roles listed in the editable_by array can delete settings';
