-- Rollback Realtime Schema RLS Policies

DROP POLICY IF EXISTS "Admins can manage realtime configuration" ON realtime.schema_registry;
DROP POLICY IF EXISTS "Authenticated users can view realtime configuration" ON realtime.schema_registry;

ALTER TABLE realtime.schema_registry DISABLE ROW LEVEL SECURITY;
