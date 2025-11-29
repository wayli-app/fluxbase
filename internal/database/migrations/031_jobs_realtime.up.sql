-- ============================================
-- REPLICA IDENTITY for UPDATE/DELETE payloads
-- ============================================
ALTER TABLE jobs.job_queue REPLICA IDENTITY FULL;
ALTER TABLE jobs.job_functions REPLICA IDENTITY FULL;
ALTER TABLE jobs.workers REPLICA IDENTITY FULL;
ALTER TABLE jobs.job_function_files REPLICA IDENTITY FULL;

-- ============================================
-- Generic notify function for jobs schema
-- ============================================
CREATE OR REPLACE FUNCTION jobs.notify_realtime_change()
RETURNS TRIGGER AS $$
BEGIN
  PERFORM pg_notify(
    'fluxbase_changes',
    json_build_object(
      'schema', TG_TABLE_SCHEMA,
      'table', TG_TABLE_NAME,
      'type', TG_OP,
      'record', CASE WHEN TG_OP != 'DELETE' THEN row_to_json(NEW) END,
      'old_record', CASE WHEN TG_OP != 'INSERT' THEN row_to_json(OLD) END
    )::text
  );
  RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- Attach triggers to all jobs tables
-- ============================================
-- Note: job_queue is the only table that needs realtime notifications
-- (for tracking actual job execution and progress).
-- job_functions contains large code fields (20MB+) that would exceed
-- pg_notify's 8KB limit, so we skip the trigger for that table.
-- ============================================
CREATE TRIGGER job_queue_realtime_notify
AFTER INSERT OR UPDATE OR DELETE ON jobs.job_queue
FOR EACH ROW EXECUTE FUNCTION jobs.notify_realtime_change();

-- Skipping job_functions - code fields are too large for pg_notify (8KB limit)
-- CREATE TRIGGER job_functions_realtime_notify
-- AFTER INSERT OR UPDATE OR DELETE ON jobs.job_functions
-- FOR EACH ROW EXECUTE FUNCTION jobs.notify_realtime_change();

CREATE TRIGGER workers_realtime_notify
AFTER INSERT OR UPDATE OR DELETE ON jobs.workers
FOR EACH ROW EXECUTE FUNCTION jobs.notify_realtime_change();

CREATE TRIGGER job_function_files_realtime_notify
AFTER INSERT OR UPDATE OR DELETE ON jobs.job_function_files
FOR EACH ROW EXECUTE FUNCTION jobs.notify_realtime_change();

-- ============================================
-- Register tables for realtime in schema registry
-- ============================================
-- Note: job_functions is excluded because code fields exceed pg_notify's 8KB limit
INSERT INTO realtime.schema_registry (schema_name, table_name, realtime_enabled, events)
VALUES
    ('jobs', 'job_queue', true, ARRAY['INSERT', 'UPDATE', 'DELETE']),
    -- ('jobs', 'job_functions', true, ARRAY['INSERT', 'UPDATE', 'DELETE']), -- Excluded: large code fields
    ('jobs', 'workers', true, ARRAY['INSERT', 'UPDATE', 'DELETE']),
    ('jobs', 'job_function_files', true, ARRAY['INSERT', 'UPDATE', 'DELETE'])
ON CONFLICT (schema_name, table_name) DO UPDATE
SET realtime_enabled = true,
    events = ARRAY['INSERT', 'UPDATE', 'DELETE'];

-- ============================================
-- RLS Policy: dashboard_admin can read all jobs
-- ============================================
CREATE POLICY "Dashboard admins can read all jobs"
    ON jobs.job_queue FOR SELECT
    TO authenticated
    USING (auth.role() = 'dashboard_admin');

CREATE POLICY "Dashboard admins can read all job functions"
    ON jobs.job_functions FOR SELECT
    TO authenticated
    USING (auth.role() = 'dashboard_admin');

CREATE POLICY "Dashboard admins can read all workers"
    ON jobs.workers FOR SELECT
    TO authenticated
    USING (auth.role() = 'dashboard_admin');

CREATE POLICY "Dashboard admins can read all job function files"
    ON jobs.job_function_files FOR SELECT
    TO authenticated
    USING (auth.role() = 'dashboard_admin');
