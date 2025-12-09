-- Add execution_logs table for edge functions (like RPC has)
-- This allows for individual log entries with levels and timestamps,
-- enabling live streaming via Realtime and better log filtering

CREATE TABLE IF NOT EXISTS functions.execution_logs (
    id BIGSERIAL PRIMARY KEY,
    execution_id UUID NOT NULL REFERENCES functions.edge_executions(id) ON DELETE CASCADE,
    line_number INTEGER NOT NULL,
    level TEXT NOT NULL DEFAULT 'info' CHECK (level IN ('debug', 'info', 'warn', 'error')),
    message TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_functions_execution_logs_execution ON functions.execution_logs(execution_id);
CREATE INDEX IF NOT EXISTS idx_functions_execution_logs_execution_line ON functions.execution_logs(execution_id, line_number);

COMMENT ON TABLE functions.execution_logs IS 'Individual log lines for edge function execution (streamed via Realtime)';

-- Enable RLS
ALTER TABLE functions.execution_logs ENABLE ROW LEVEL SECURITY;

-- Service role can do everything (bypasses RLS)
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'functions' AND tablename = 'execution_logs' AND policyname = 'functions_execution_logs_service_all') THEN
        CREATE POLICY "functions_execution_logs_service_all" ON functions.execution_logs
            FOR ALL TO service_role USING (true);
    END IF;
END $$;

-- Grant permissions
GRANT SELECT ON functions.execution_logs TO authenticated;
GRANT ALL ON functions.execution_logs TO service_role;

-- Grant sequence usage for execution_logs
GRANT USAGE, SELECT ON SEQUENCE functions.execution_logs_id_seq TO service_role;

-- Set replica identity for Realtime UPDATE/DELETE payloads
ALTER TABLE functions.execution_logs REPLICA IDENTITY FULL;

-- Create notify function for functions schema (if not exists)
CREATE OR REPLACE FUNCTION functions.notify_realtime_change()
RETURNS TRIGGER AS $$
BEGIN
  PERFORM pg_notify(
    'fluxbase_changes',
    json_build_object(
      'schema', TG_TABLE_SCHEMA,
      'table', TG_TABLE_NAME,
      'type', TG_OP,
      'record', CASE WHEN TG_OP != 'DELETE' THEN to_jsonb(NEW) ELSE NULL END,
      'old_record', CASE WHEN TG_OP != 'INSERT' THEN to_jsonb(OLD) ELSE NULL END
    )::text
  );
  RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Attach trigger for Realtime notifications (logs are append-only, so only INSERT)
DROP TRIGGER IF EXISTS execution_logs_realtime_notify ON functions.execution_logs;
CREATE TRIGGER execution_logs_realtime_notify
AFTER INSERT ON functions.execution_logs
FOR EACH ROW EXECUTE FUNCTION functions.notify_realtime_change();

-- Register table for realtime in schema registry
INSERT INTO realtime.schema_registry (schema_name, table_name, realtime_enabled, events)
VALUES ('functions', 'execution_logs', true, ARRAY['INSERT'])
ON CONFLICT (schema_name, table_name) DO UPDATE
SET realtime_enabled = true,
    events = EXCLUDED.events;
