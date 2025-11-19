-- Rollback App Schema RLS Policies

DROP POLICY IF EXISTS "Settings can be deleted by authorized roles" ON app.settings;
DROP POLICY IF EXISTS "Settings can be updated by authorized roles" ON app.settings;
DROP POLICY IF EXISTS "Settings can be created by authorized roles" ON app.settings;
DROP POLICY IF EXISTS "Authenticated users can read non-secret settings" ON app.settings;
DROP POLICY IF EXISTS "Public settings are readable by anyone" ON app.settings;
DROP POLICY IF EXISTS "Service role has full access to app settings" ON app.settings;

ALTER TABLE app.settings DISABLE ROW LEVEL SECURITY;
