-- Migration: Add Webhook Triggering System
-- This migration creates the automatic webhook event queue and trigger system

-- Create webhook event queue table
CREATE TABLE IF NOT EXISTS auth.webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id UUID REFERENCES auth.webhooks(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL, -- INSERT, UPDATE, DELETE
    table_schema VARCHAR(255) NOT NULL,
    table_name VARCHAR(255) NOT NULL,
    record_id TEXT,
    old_data JSONB,
    new_data JSONB,
    processed BOOLEAN DEFAULT FALSE,
    attempts INT DEFAULT 0,
    last_attempt_at TIMESTAMPTZ,
    next_retry_at TIMESTAMPTZ,
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_webhook_event_webhook FOREIGN KEY (webhook_id) REFERENCES auth.webhooks(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_webhook_events_unprocessed ON auth.webhook_events(processed, next_retry_at) WHERE processed = FALSE;
CREATE INDEX IF NOT EXISTS idx_webhook_events_webhook ON auth.webhook_events(webhook_id);
CREATE INDEX IF NOT EXISTS idx_webhook_events_created ON auth.webhook_events(created_at);

COMMENT ON TABLE auth.webhook_events IS 'Queue for webhook events to be delivered. Processed events are kept for history.';

-- Function to queue webhook events
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
        -- Events structure: [{"table": "users", "operations": ["INSERT", "UPDATE"]}]
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
                CURRENT_TIMESTAMP -- Ready to be processed immediately
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

-- Function to create webhook trigger on a table
CREATE OR REPLACE FUNCTION auth.create_webhook_trigger(
    schema_name TEXT,
    table_name TEXT
) RETURNS VOID AS $$
DECLARE
    trigger_name TEXT;
    full_table_name TEXT;
BEGIN
    trigger_name := format('webhook_trigger_%s_%s', schema_name, table_name);
    full_table_name := format('%I.%I', schema_name, table_name);

    -- Drop existing trigger if exists
    EXECUTE format('DROP TRIGGER IF EXISTS %I ON %s', trigger_name, full_table_name);

    -- Create new trigger
    EXECUTE format('
        CREATE TRIGGER %I
        AFTER INSERT OR UPDATE OR DELETE ON %s
        FOR EACH ROW EXECUTE FUNCTION auth.queue_webhook_event()
    ', trigger_name, full_table_name);

    RAISE NOTICE 'Created webhook trigger % on %', trigger_name, full_table_name;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

COMMENT ON FUNCTION auth.create_webhook_trigger IS 'Creates a webhook trigger on a specified table';

-- Function to remove webhook trigger from a table
CREATE OR REPLACE FUNCTION auth.remove_webhook_trigger(
    schema_name TEXT,
    table_name TEXT
) RETURNS VOID AS $$
DECLARE
    trigger_name TEXT;
    full_table_name TEXT;
BEGIN
    trigger_name := format('webhook_trigger_%s_%s', schema_name, table_name);
    full_table_name := format('%I.%I', schema_name, table_name);

    EXECUTE format('DROP TRIGGER IF EXISTS %I ON %s', trigger_name, full_table_name);

    RAISE NOTICE 'Removed webhook trigger % from %', trigger_name, full_table_name;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

COMMENT ON FUNCTION auth.remove_webhook_trigger IS 'Removes a webhook trigger from a specified table';

-- Enable RLS on webhook_events table
ALTER TABLE auth.webhook_events ENABLE ROW LEVEL SECURITY;

-- RLS Policies for webhook_events
-- Admins can see all webhook events
CREATE POLICY webhook_events_admin_select ON auth.webhook_events
    FOR SELECT
    USING (auth.is_admin() OR auth.current_user_role() = 'dashboard_admin');

-- Service role can manage webhook events
CREATE POLICY webhook_events_service ON auth.webhook_events
    FOR ALL
    USING (auth.current_user_role() = 'service_role');

-- Grant necessary permissions
GRANT SELECT, INSERT, UPDATE, DELETE ON auth.webhook_events TO fluxbase_app;
GRANT USAGE ON SCHEMA auth TO fluxbase_app;

-- Note: Webhook triggers should be created manually using:
-- SELECT auth.create_webhook_trigger('schema_name', 'table_name');
-- This allows you to control which tables have webhook triggers enabled.

-- Add helpful comment
COMMENT ON TABLE auth.webhook_events IS 'Webhook event queue - events are automatically queued when data changes occur in tables with webhook triggers enabled';
