--
-- GLOBAL HELPER FUNCTIONS
-- This file contains globally used helper functions across all schemas
--

-- Update trigger function for updated_at columns
CREATE OR REPLACE FUNCTION public.update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Webhook trigger function to queue webhook events
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

-- Function to validate app_metadata updates (only admins can modify)
CREATE OR REPLACE FUNCTION auth.validate_app_metadata_update()
RETURNS TRIGGER AS $$
DECLARE
    user_role TEXT;
BEGIN
    -- Get the current user's role
    user_role := auth.current_user_role();

    -- Check if app_metadata is being modified
    IF OLD.app_metadata IS DISTINCT FROM NEW.app_metadata THEN
        -- Only allow admins and dashboard admins to modify app_metadata
        IF user_role != 'admin' AND user_role != 'dashboard_admin' THEN
            -- Also check if user has admin privileges via is_admin() function
            IF NOT auth.is_admin() THEN
                RAISE EXCEPTION 'Only admins can modify app_metadata'
                    USING ERRCODE = 'insufficient_privilege';
            END IF;
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

COMMENT ON FUNCTION auth.validate_app_metadata_update() IS 'Validates that only admins and dashboard admins can modify the app_metadata field on auth.users';

-- Webhook updated_at trigger function
CREATE OR REPLACE FUNCTION auth.update_webhook_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Storage helper functions (moved here after storage schema is created)
CREATE OR REPLACE FUNCTION storage.user_can_access_object(p_object_id UUID, p_required_permission TEXT DEFAULT 'read')
RETURNS BOOLEAN AS $$
DECLARE
    v_owner_id UUID;
    v_bucket_public BOOLEAN;
    v_has_permission BOOLEAN;
    v_user_role TEXT;
BEGIN
    v_user_role := auth.current_user_role();

    -- Admins and service roles can access everything
    IF v_user_role IN ('dashboard_admin', 'service_role') THEN
        RETURN TRUE;
    END IF;

    -- Get object owner and bucket public status
    SELECT o.owner_id, b.public INTO v_owner_id, v_bucket_public
    FROM storage.objects o
    JOIN storage.buckets b ON b.id = o.bucket_id
    WHERE o.id = p_object_id;

    -- If object not found, deny access
    IF NOT FOUND THEN
        RETURN FALSE;
    END IF;

    -- Check if user is the owner
    IF v_owner_id = auth.current_user_id() THEN
        RETURN TRUE;
    END IF;

    -- Check if bucket is public (read-only for non-owners)
    IF v_bucket_public AND p_required_permission = 'read' THEN
        RETURN TRUE;
    END IF;

    -- Check object_permissions table for explicit shares
    IF auth.current_user_id() IS NOT NULL THEN
        SELECT EXISTS(
            SELECT 1 FROM storage.object_permissions
            WHERE object_id = p_object_id
            AND user_id = auth.current_user_id()
            AND (permission = 'write' OR (permission = 'read' AND p_required_permission = 'read'))
        ) INTO v_has_permission;

        IF v_has_permission THEN
            RETURN TRUE;
        END IF;
    END IF;

    RETURN FALSE;
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

COMMENT ON FUNCTION storage.user_can_access_object(UUID, TEXT) IS 'Checks if the current user can access a storage object with the required permission (read or write). Returns TRUE if: user is admin/service role, user owns the object, object is in public bucket (read only), or user has been granted permission via object_permissions table.';

-- SECURITY DEFINER function to check if bucket is public
-- This prevents infinite recursion when RLS policies on storage.objects
-- reference storage.buckets (which also has RLS enabled)
CREATE OR REPLACE FUNCTION storage.is_bucket_public(bucket_name TEXT)
RETURNS BOOLEAN
LANGUAGE plpgsql
SECURITY DEFINER
STABLE
AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM storage.buckets
        WHERE id = bucket_name AND public = true
    );
END;
$$;

COMMENT ON FUNCTION storage.is_bucket_public IS
    'Check if a bucket is public, bypassing RLS to prevent infinite recursion';

-- SECURITY DEFINER function to check object permissions
-- This prevents infinite recursion when RLS policies on storage.objects
-- reference storage.object_permissions (which also has RLS enabled)
CREATE OR REPLACE FUNCTION storage.has_object_permission(
    p_object_id UUID,
    p_user_id UUID,
    p_permission TEXT
)
RETURNS BOOLEAN
LANGUAGE plpgsql
SECURITY DEFINER
STABLE
AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM storage.object_permissions
        WHERE object_id = p_object_id
        AND user_id = p_user_id
        AND (permission = p_permission OR (p_permission = 'read' AND permission = 'write'))
    );
END;
$$;

COMMENT ON FUNCTION storage.has_object_permission IS
    'Check if user has permission on object, bypassing RLS to prevent infinite recursion';

-- Supabase compatibility: storage.foldername() extracts folder path from object name
CREATE OR REPLACE FUNCTION storage.foldername(name TEXT)
RETURNS TEXT[] AS $$
DECLARE
    path_parts TEXT[];
    folder_parts TEXT[];
BEGIN
    IF name IS NULL OR name = '' THEN
        RETURN ARRAY[]::TEXT[];
    END IF;

    -- Split the path by '/' to get folder structure
    path_parts := string_to_array(name, '/');

    -- Remove the last element (filename) to get just folders
    IF array_length(path_parts, 1) > 1 THEN
        folder_parts := path_parts[1:array_length(path_parts, 1) - 1];
    ELSE
        -- No folders, just a filename at root
        folder_parts := ARRAY[]::TEXT[];
    END IF;

    RETURN folder_parts;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

COMMENT ON FUNCTION storage.foldername(TEXT) IS 'Supabase-compatible function that extracts folder path components from an object name/path. Returns array of folder names. Use [1] to get first folder, [2] for second, etc.';
