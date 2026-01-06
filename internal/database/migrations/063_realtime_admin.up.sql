-- ============================================================================
-- REALTIME ADMIN - Schema updates and shared notify function
-- ============================================================================
-- Adds excluded_columns to realtime.schema_registry and creates a shared
-- notify function for user tables to use.
-- ============================================================================

-- Add excluded_columns to track which columns to omit from notifications
ALTER TABLE realtime.schema_registry
ADD COLUMN IF NOT EXISTS excluded_columns TEXT[] DEFAULT '{}';

-- ============================================================================
-- Shared notify function for user tables
-- This function is used by dynamically created triggers on user tables.
-- It looks up excluded columns from the schema registry and omits them.
-- ============================================================================
CREATE OR REPLACE FUNCTION public.notify_realtime_change()
RETURNS TRIGGER AS $$
DECLARE
    notification_record JSONB;
    old_notification_record JSONB;
    excluded_cols TEXT[];
    col TEXT;
BEGIN
    -- Get excluded columns from registry
    SELECT excluded_columns INTO excluded_cols
    FROM realtime.schema_registry
    WHERE schema_name = TG_TABLE_SCHEMA AND table_name = TG_TABLE_NAME;

    -- Build notification records, excluding specified columns
    IF TG_OP != 'DELETE' THEN
        notification_record := to_jsonb(NEW);
        IF excluded_cols IS NOT NULL THEN
            FOREACH col IN ARRAY excluded_cols LOOP
                notification_record := notification_record - col;
            END LOOP;
        END IF;
    END IF;

    IF TG_OP != 'INSERT' THEN
        old_notification_record := to_jsonb(OLD);
        IF excluded_cols IS NOT NULL THEN
            FOREACH col IN ARRAY excluded_cols LOOP
                old_notification_record := old_notification_record - col;
            END LOOP;
        END IF;
    END IF;

    PERFORM pg_notify(
        'fluxbase_changes',
        json_build_object(
            'schema', TG_TABLE_SCHEMA,
            'table', TG_TABLE_NAME,
            'type', TG_OP,
            'record', notification_record,
            'old_record', old_notification_record
        )::text
    );

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION public.notify_realtime_change() IS 'Shared notify function for realtime-enabled user tables. Dynamically excludes columns based on registry configuration.';
