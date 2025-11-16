--
-- APP SCHEMA RLS POLICIES
-- Row Level Security policies for application settings
--

-- Enable RLS on app.settings
ALTER TABLE app.settings ENABLE ROW LEVEL SECURITY;

-- Allow service_role to do everything (bypasses RLS anyway, but explicit is good)
CREATE POLICY "Service role has full access to app settings"
    ON app.settings
    FOR ALL
    TO service_role
    USING (true)
    WITH CHECK (true);

-- Allow public/anon users to read public settings
CREATE POLICY "Public settings are readable by anyone"
    ON app.settings
    FOR SELECT
    TO anon, authenticated
    USING (is_public = true AND is_secret = false);

-- Allow authenticated users to read non-secret settings
CREATE POLICY "Authenticated users can read non-secret settings"
    ON app.settings
    FOR SELECT
    TO authenticated
    USING (is_secret = false);

-- Settings are only editable by roles specified in editable_by array
-- This policy would require a function to check current user's role
-- For now, we'll rely on application-level authorization for writes

COMMENT ON POLICY "Service role has full access to app settings" ON app.settings
    IS 'Service role can manage all settings';
COMMENT ON POLICY "Public settings are readable by anyone" ON app.settings
    IS 'Public, non-secret settings can be read by anonymous and authenticated users';
COMMENT ON POLICY "Authenticated users can read non-secret settings" ON app.settings
    IS 'Authenticated users can read all non-secret settings regardless of public flag';
