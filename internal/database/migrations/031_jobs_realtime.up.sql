-- ============================================
-- REPLICA IDENTITY for UPDATE/DELETE payloads
-- ============================================
ALTER TABLE jobs.queue REPLICA IDENTITY FULL;
ALTER TABLE jobs.functions REPLICA IDENTITY FULL;
ALTER TABLE jobs.workers REPLICA IDENTITY FULL;
ALTER TABLE jobs.function_files REPLICA IDENTITY FULL;

-- ============================================
-- Execution logs table (separate from queue for efficient Realtime streaming)
-- ============================================
CREATE TABLE jobs.execution_logs (
    id BIGSERIAL PRIMARY KEY,
    job_id UUID NOT NULL REFERENCES jobs.queue(id) ON DELETE CASCADE,
    line_number INTEGER NOT NULL,
    level TEXT NOT NULL DEFAULT 'info',
    message TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_execution_logs_job_id ON jobs.execution_logs(job_id);
CREATE INDEX IF NOT EXISTS idx_execution_logs_job_id_line ON jobs.execution_logs(job_id, line_number);

-- Enable Realtime for execution_logs
ALTER TABLE jobs.execution_logs REPLICA IDENTITY FULL;

COMMENT ON TABLE jobs.execution_logs IS 'Individual log lines for job execution (streamed via Realtime)';

-- Enable RLS on execution_logs
ALTER TABLE jobs.execution_logs ENABLE ROW LEVEL SECURITY;

-- Execution Logs: Users can read logs for their own jobs
CREATE POLICY "Users can read their own job logs"
    ON jobs.execution_logs FOR SELECT
    TO authenticated
    USING (
        EXISTS (
            SELECT 1 FROM jobs.queue
            WHERE queue.id = execution_logs.job_id
            AND queue.created_by = auth.uid()
        )
    );

CREATE POLICY "Service role can manage execution logs"
    ON jobs.execution_logs FOR ALL
    TO service_role
    USING (true)
    WITH CHECK (true);

-- ============================================
-- Generic notify function for jobs schema
-- Excludes large fields (result, payload) from notifications
-- ============================================
CREATE OR REPLACE FUNCTION jobs.notify_realtime_change()
RETURNS TRIGGER AS $$
DECLARE
  notification_record JSONB;
  old_notification_record JSONB;
BEGIN
  -- Build record without large fields for notification efficiency
  IF TG_OP != 'DELETE' THEN
    notification_record := to_jsonb(NEW) - 'result' - 'payload';
  END IF;
  IF TG_OP != 'INSERT' THEN
    old_notification_record := to_jsonb(OLD) - 'result' - 'payload';
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

-- ============================================
-- Attach triggers to all jobs tables
-- ============================================
-- Note: queue is the only table that needs realtime notifications
-- (for tracking actual job execution and progress).
-- functions contains large code fields (20MB+) that would exceed
-- pg_notify's 8KB limit, so we skip the trigger for that table.
-- ============================================
CREATE TRIGGER queue_realtime_notify
AFTER INSERT OR UPDATE OR DELETE ON jobs.queue
FOR EACH ROW EXECUTE FUNCTION jobs.notify_realtime_change();

-- Skipping functions - code fields are too large for pg_notify (8KB limit)
-- CREATE TRIGGER functions_realtime_notify
-- AFTER INSERT OR UPDATE OR DELETE ON jobs.functions
-- FOR EACH ROW EXECUTE FUNCTION jobs.notify_realtime_change();

CREATE TRIGGER workers_realtime_notify
AFTER INSERT OR UPDATE OR DELETE ON jobs.workers
FOR EACH ROW EXECUTE FUNCTION jobs.notify_realtime_change();

CREATE TRIGGER function_files_realtime_notify
AFTER INSERT OR UPDATE OR DELETE ON jobs.function_files
FOR EACH ROW EXECUTE FUNCTION jobs.notify_realtime_change();

-- execution_logs only needs INSERT notifications (logs are append-only)
CREATE TRIGGER execution_logs_realtime_notify
AFTER INSERT ON jobs.execution_logs
FOR EACH ROW EXECUTE FUNCTION jobs.notify_realtime_change();

-- ============================================
-- Register tables for realtime in schema registry
-- ============================================
-- Note: functions is excluded because code fields exceed pg_notify's 8KB limit
-- Note: execution_logs only sends INSERT events (logs are append-only)
INSERT INTO realtime.schema_registry (schema_name, table_name, realtime_enabled, events)
VALUES
    ('jobs', 'queue', true, ARRAY['INSERT', 'UPDATE', 'DELETE']),
    -- ('jobs', 'functions', true, ARRAY['INSERT', 'UPDATE', 'DELETE']), -- Excluded: large code fields
    ('jobs', 'workers', true, ARRAY['INSERT', 'UPDATE', 'DELETE']),
    ('jobs', 'function_files', true, ARRAY['INSERT', 'UPDATE', 'DELETE']),
    ('jobs', 'execution_logs', true, ARRAY['INSERT'])
ON CONFLICT (schema_name, table_name) DO UPDATE
SET realtime_enabled = true,
    events = EXCLUDED.events;

-- ============================================
-- RLS Policy: dashboard_admin can read all jobs
-- ============================================
CREATE POLICY "Dashboard admins can read all jobs"
    ON jobs.queue FOR SELECT
    TO authenticated
    USING (auth.role() = 'dashboard_admin');

CREATE POLICY "Dashboard admins can read all functions"
    ON jobs.functions FOR SELECT
    TO authenticated
    USING (auth.role() = 'dashboard_admin');

CREATE POLICY "Dashboard admins can read all workers"
    ON jobs.workers FOR SELECT
    TO authenticated
    USING (auth.role() = 'dashboard_admin');

CREATE POLICY "Dashboard admins can read all function files"
    ON jobs.function_files FOR SELECT
    TO authenticated
    USING (auth.role() = 'dashboard_admin');

CREATE POLICY "Dashboard admins can read all execution logs"
    ON jobs.execution_logs FOR SELECT
    TO authenticated
    USING (auth.role() = 'dashboard_admin');
