--
-- ROLLBACK: WEBHOOK TRIGGERS AND SCOPING
--

-- 1. Remove all webhook triggers from monitored tables
DO $$
DECLARE
    rec RECORD;
BEGIN
    FOR rec IN SELECT schema_name, table_name FROM auth.webhook_monitored_tables
    LOOP
        PERFORM auth.remove_webhook_trigger(rec.schema_name, rec.table_name);
    END LOOP;
END $$;

-- 2. Drop the RLS policy
DROP POLICY IF EXISTS webhook_monitored_tables_service_only ON auth.webhook_monitored_tables;

-- 3. Drop the monitored tables tracking table
DROP TABLE IF EXISTS auth.webhook_monitored_tables;

-- 4. Remove scope column from webhooks
ALTER TABLE auth.webhooks DROP COLUMN IF EXISTS scope;

-- 5. Drop helper functions
DROP FUNCTION IF EXISTS auth.increment_webhook_table_count(TEXT, TEXT);
DROP FUNCTION IF EXISTS auth.decrement_webhook_table_count(TEXT, TEXT);

-- 6. Restore original queue_webhook_event function (without scoping)
CREATE OR REPLACE FUNCTION auth.queue_webhook_event()
RETURNS TRIGGER AS $$
DECLARE
    webhook_record RECORD;
    event_type TEXT;
    old_data JSONB;
    new_data JSONB;
    record_id_value TEXT;
    should_trigger BOOLEAN;
BEGIN
    -- Determine event type
    IF TG_OP = 'INSERT' THEN
        event_type := 'INSERT';
        old_data := NULL;
        new_data := to_jsonb(NEW);
        record_id_value := COALESCE((NEW.id)::TEXT, '');
    ELSIF TG_OP = 'UPDATE' THEN
        event_type := 'UPDATE';
        old_data := to_jsonb(OLD);
        new_data := to_jsonb(NEW);
        record_id_value := COALESCE((NEW.id)::TEXT, (OLD.id)::TEXT, '');
    ELSIF TG_OP = 'DELETE' THEN
        event_type := 'DELETE';
        old_data := to_jsonb(OLD);
        new_data := NULL;
        record_id_value := COALESCE((OLD.id)::TEXT, '');
    ELSE
        RETURN NULL;
    END IF;

    -- Find matching webhooks
    FOR webhook_record IN
        SELECT id, events
        FROM auth.webhooks
        WHERE enabled = TRUE
    LOOP
        -- Check if this webhook is interested in this event
        should_trigger := FALSE;

        -- Parse the events JSONB array to check if it matches
        IF jsonb_typeof(webhook_record.events) = 'array' THEN
            should_trigger := EXISTS (
                SELECT 1
                FROM jsonb_array_elements(webhook_record.events) AS event
                WHERE
                    (event->>'table' = TG_TABLE_NAME OR event->>'table' = '*')
                    AND (
                        event->'operations' @> to_jsonb(ARRAY[event_type])
                        OR event->'operations' @> to_jsonb(ARRAY['*'])
                    )
            );
        END IF;

        -- Queue event if webhook is interested
        IF should_trigger THEN
            INSERT INTO auth.webhook_events (
                webhook_id,
                event_type,
                table_schema,
                table_name,
                record_id,
                old_data,
                new_data,
                next_retry_at
            ) VALUES (
                webhook_record.id,
                event_type,
                TG_TABLE_SCHEMA,
                TG_TABLE_NAME,
                record_id_value,
                old_data,
                new_data,
                CURRENT_TIMESTAMP
            );

            -- Send notification to application via pg_notify
            PERFORM pg_notify('webhook_event', webhook_record.id::TEXT);
        END IF;
    END LOOP;

    -- Return appropriate value based on operation
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    ELSE
        RETURN NEW;
    END IF;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION auth.queue_webhook_event() IS 'Trigger function that queues webhook events when data changes occur';
