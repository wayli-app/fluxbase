-- ============================================================================
-- REALTIME ADMIN - Rollback
-- ============================================================================

-- Drop the shared notify function
DROP FUNCTION IF EXISTS public.notify_realtime_change();

-- Remove excluded_columns from schema registry
ALTER TABLE realtime.schema_registry
DROP COLUMN IF EXISTS excluded_columns;
