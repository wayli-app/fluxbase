-- Create function to notify on table changes
CREATE OR REPLACE FUNCTION notify_table_change()
RETURNS TRIGGER AS $$
DECLARE
  payload JSON;
  notification JSON;
BEGIN
  -- Build notification payload
  IF (TG_OP = 'DELETE') THEN
    notification = json_build_object(
      'type', TG_OP,
      'table', TG_TABLE_NAME,
      'schema', TG_TABLE_SCHEMA,
      'old_record', row_to_json(OLD)
    );
  ELSIF (TG_OP = 'UPDATE') THEN
    notification = json_build_object(
      'type', TG_OP,
      'table', TG_TABLE_NAME,
      'schema', TG_TABLE_SCHEMA,
      'record', row_to_json(NEW),
      'old_record', row_to_json(OLD)
    );
  ELSIF (TG_OP = 'INSERT') THEN
    notification = json_build_object(
      'type', TG_OP,
      'table', TG_TABLE_NAME,
      'schema', TG_TABLE_SCHEMA,
      'record', row_to_json(NEW)
    );
  END IF;

  -- Send notification
  PERFORM pg_notify('fluxbase_changes', notification::text);

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for products table (example)
DROP TRIGGER IF EXISTS products_notify_change ON public.products;
CREATE TRIGGER products_notify_change
  AFTER INSERT OR UPDATE OR DELETE ON public.products
  FOR EACH ROW
  EXECUTE FUNCTION notify_table_change();

-- Create helper function to enable realtime on any table
CREATE OR REPLACE FUNCTION enable_realtime(schema_name TEXT, table_name TEXT)
RETURNS VOID AS $$
BEGIN
  EXECUTE format(
    'DROP TRIGGER IF EXISTS %I ON %I.%I',
    table_name || '_notify_change',
    schema_name,
    table_name
  );

  EXECUTE format(
    'CREATE TRIGGER %I AFTER INSERT OR UPDATE OR DELETE ON %I.%I FOR EACH ROW EXECUTE FUNCTION notify_table_change()',
    table_name || '_notify_change',
    schema_name,
    table_name
  );
END;
$$ LANGUAGE plpgsql;

-- Create helper function to disable realtime on any table
CREATE OR REPLACE FUNCTION disable_realtime(schema_name TEXT, table_name TEXT)
RETURNS VOID AS $$
BEGIN
  EXECUTE format(
    'DROP TRIGGER IF EXISTS %I ON %I.%I',
    table_name || '_notify_change',
    schema_name,
    table_name
  );
END;
$$ LANGUAGE plpgsql;
