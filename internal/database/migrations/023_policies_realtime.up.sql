-- ============================================================================
-- REALTIME SCHEMA RLS
-- ============================================================================
-- This file contains Row Level Security (RLS) policies for the Realtime schema.
-- These policies restrict access to realtime configuration to authenticated users
-- and allow only admins to modify the configuration.
-- ============================================================================

-- Enable RLS on realtime.schema_registry
ALTER TABLE realtime.schema_registry ENABLE ROW LEVEL SECURITY;
ALTER TABLE realtime.schema_registry FORCE ROW LEVEL SECURITY;

-- Authenticated users can view realtime configuration
DROP POLICY IF EXISTS "Authenticated users can view realtime configuration" ON realtime.schema_registry;
CREATE POLICY "Authenticated users can view realtime configuration"
    ON realtime.schema_registry
    FOR SELECT
    TO authenticated
    USING (true);

-- Only admins and service_role can modify realtime configuration
DROP POLICY IF EXISTS "Admins can manage realtime configuration" ON realtime.schema_registry;
CREATE POLICY "Admins can manage realtime configuration"
    ON realtime.schema_registry
    FOR ALL
    TO authenticated
    USING (
        auth.current_user_role() = 'service_role'
        OR auth.current_user_role() = 'dashboard_admin'
        OR auth.is_admin()
    )
    WITH CHECK (
        auth.current_user_role() = 'service_role'
        OR auth.current_user_role() = 'dashboard_admin'
        OR auth.is_admin()
    );

COMMENT ON POLICY "Authenticated users can view realtime configuration" ON realtime.schema_registry
    IS 'Authenticated users can view which tables have realtime enabled';
COMMENT ON POLICY "Admins can manage realtime configuration" ON realtime.schema_registry
    IS 'Only admins, dashboard admins, and service role can modify realtime configuration';
