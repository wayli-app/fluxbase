--
-- WEBHOOK TRIGGERS AND SCOPING
-- This migration adds automatic trigger management and user-based scoping to webhooks
--

-- 1. Create table for tracking monitored tables (reference counting)
CREATE TABLE IF NOT EXISTS auth.webhook_monitored_tables (
    schema_name TEXT NOT NULL,
    table_name TEXT NOT NULL,
    webhook_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (schema_name, table_name)
);

COMMENT ON TABLE auth.webhook_monitored_tables IS 'Tracks which tables have webhook triggers installed and how many webhooks monitor each table';

-- 2. Add scope column to webhooks
ALTER TABLE auth.webhooks
ADD COLUMN IF NOT EXISTS scope TEXT DEFAULT 'user'
CHECK (scope IN ('user', 'global'));

COMMENT ON COLUMN auth.webhooks.scope IS 'Scope determines which events trigger the webhook: user = only events on records owned by created_by, global = all events (admin only)';

-- 3. Helper function: Increment webhook count and create trigger if first
CREATE OR REPLACE FUNCTION auth.increment_webhook_table_count(p_schema TEXT, p_table TEXT)
RETURNS VOID AS $$
DECLARE
    v_count INTEGER;
BEGIN
    INSERT INTO auth.webhook_monitored_tables (schema_name, table_name, webhook_count)
    VALUES (p_schema, p_table, 1)
    ON CONFLICT (schema_name, table_name)
    DO UPDATE SET webhook_count = auth.webhook_monitored_tables.webhook_count + 1;

    -- Get the current count
    SELECT webhook_count INTO v_count
    FROM auth.webhook_monitored_tables
    WHERE schema_name = p_schema AND table_name = p_table;

    -- Create trigger if this is the first webhook monitoring this table
    IF v_count = 1 THEN
        PERFORM auth.create_webhook_trigger(p_schema, p_table);
    END IF;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

COMMENT ON FUNCTION auth.increment_webhook_table_count IS 'Increments the webhook count for a table and creates the trigger if this is the first webhook';

-- 4. Helper function: Decrement webhook count and remove trigger if zero
CREATE OR REPLACE FUNCTION auth.decrement_webhook_table_count(p_schema TEXT, p_table TEXT)
RETURNS VOID AS $$
DECLARE
    v_count INTEGER;
BEGIN
    UPDATE auth.webhook_monitored_tables
    SET webhook_count = GREATEST(0, webhook_count - 1)
    WHERE schema_name = p_schema AND table_name = p_table
    RETURNING webhook_count INTO v_count;

    -- Remove trigger and tracking row if no webhooks left
    IF v_count IS NOT NULL AND v_count = 0 THEN
        PERFORM auth.remove_webhook_trigger(p_schema, p_table);
        DELETE FROM auth.webhook_monitored_tables
        WHERE schema_name = p_schema AND table_name = p_table;
    END IF;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

COMMENT ON FUNCTION auth.decrement_webhook_table_count IS 'Decrements the webhook count for a table and removes the trigger if no webhooks remain';

-- 5. Update queue_webhook_event function with scoping logic
CREATE OR REPLACE FUNCTION auth.queue_webhook_event()
RETURNS TRIGGER AS $$
DECLARE
    webhook_record RECORD;
    event_type TEXT;
    old_data JSONB;
    new_data JSONB;
    record_id_value TEXT;
    record_owner_id UUID;
    should_trigger BOOLEAN;
BEGIN
    -- Determine event type and prepare data
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

    -- Extract record owner for scoping
    -- Check common ownership columns in order of precedence
    BEGIN
        record_owner_id := COALESCE(
            ((CASE WHEN TG_OP = 'DELETE' THEN old_data ELSE new_data END)->>'user_id')::UUID,
            ((CASE WHEN TG_OP = 'DELETE' THEN old_data ELSE new_data END)->>'owner_id')::UUID,
            ((CASE WHEN TG_OP = 'DELETE' THEN old_data ELSE new_data END)->>'created_by')::UUID,
            -- For auth.users table, use the record's own id as the owner
            CASE WHEN TG_TABLE_SCHEMA = 'auth' AND TG_TABLE_NAME = 'users'
                 THEN ((CASE WHEN TG_OP = 'DELETE' THEN old_data ELSE new_data END)->>'id')::UUID
                 ELSE NULL END
        );
    EXCEPTION WHEN OTHERS THEN
        -- If UUID parsing fails, set to NULL (unowned record)
        record_owner_id := NULL;
    END;

    -- Find matching webhooks WITH SCOPING
    FOR webhook_record IN
        SELECT id, events, created_by, scope
        FROM auth.webhooks
        WHERE enabled = TRUE
          AND (
              scope = 'global'                    -- Global webhooks see everything
              OR created_by IS NULL              -- Legacy webhooks (no owner) see everything
              OR record_owner_id IS NULL         -- Unowned records are visible to all
              OR created_by = record_owner_id    -- User-scoped: owner matches
          )
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

COMMENT ON FUNCTION auth.queue_webhook_event() IS 'Trigger function that queues webhook events when data changes occur, with user-based scoping support';

-- 6. Enable RLS on the new table
ALTER TABLE auth.webhook_monitored_tables ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.webhook_monitored_tables FORCE ROW LEVEL SECURITY;

-- Only service role can manage monitored tables (internal use only)
CREATE POLICY webhook_monitored_tables_service_only ON auth.webhook_monitored_tables
    FOR ALL
    USING (auth.current_user_role() = 'service_role')
    WITH CHECK (auth.current_user_role() = 'service_role');

-- 7. Bootstrap: Create triggers for existing enabled webhooks
DO $$
DECLARE
    webhook_rec RECORD;
    event_rec JSONB;
    v_table_name TEXT;
    v_schema_name TEXT;
BEGIN
    FOR webhook_rec IN SELECT id, events FROM auth.webhooks WHERE enabled = TRUE
    LOOP
        IF jsonb_typeof(webhook_rec.events) = 'array' THEN
            FOR event_rec IN SELECT * FROM jsonb_array_elements(webhook_rec.events)
            LOOP
                v_table_name := event_rec->>'table';
                IF v_table_name IS NOT NULL AND v_table_name != '*' THEN
                    -- Parse schema.table or default to auth schema
                    IF position('.' IN v_table_name) > 0 THEN
                        v_schema_name := split_part(v_table_name, '.', 1);
                        v_table_name := split_part(v_table_name, '.', 2);
                    ELSE
                        -- Default to auth schema since most webhook targets are auth tables
                        v_schema_name := 'auth';
                    END IF;

                    -- Increment count (will create trigger if first)
                    PERFORM auth.increment_webhook_table_count(v_schema_name, v_table_name);
                END IF;
            END LOOP;
        END IF;
    END LOOP;
END $$;
