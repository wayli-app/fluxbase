-- RPC Schema Migration
-- Creates tables for RPC procedures, executions, and execution logs

-- Create RPC schema
CREATE SCHEMA IF NOT EXISTS rpc;

-- Grant usage on rpc schema
GRANT USAGE ON SCHEMA rpc TO anon, authenticated, service_role;

-- ============================================================================
-- RPC PROCEDURES
-- SQL-based procedures with annotations for configuration
-- ============================================================================

CREATE TABLE IF NOT EXISTS rpc.procedures (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    namespace TEXT NOT NULL DEFAULT 'default',
    description TEXT,
    sql_query TEXT NOT NULL,
    original_code TEXT,

    -- Parsed from annotations
    input_schema JSONB,
    output_schema JSONB,
    allowed_tables TEXT[] DEFAULT ARRAY[]::TEXT[],
    allowed_schemas TEXT[] DEFAULT ARRAY['public']::TEXT[],
    max_execution_time_seconds INTEGER DEFAULT 30,
    require_role TEXT,
    is_public BOOLEAN DEFAULT false,

    -- Runtime config
    enabled BOOLEAN DEFAULT true,
    version INTEGER DEFAULT 1,
    source TEXT NOT NULL DEFAULT 'filesystem' CHECK (source IN ('filesystem', 'api', 'sdk')),
    created_by UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT unique_rpc_procedure_name_namespace UNIQUE (name, namespace)
);

CREATE INDEX IF NOT EXISTS idx_rpc_procedures_name ON rpc.procedures(name);
CREATE INDEX IF NOT EXISTS idx_rpc_procedures_namespace ON rpc.procedures(namespace);
CREATE INDEX IF NOT EXISTS idx_rpc_procedures_enabled ON rpc.procedures(enabled);
CREATE INDEX IF NOT EXISTS idx_rpc_procedures_source ON rpc.procedures(source);
CREATE INDEX IF NOT EXISTS idx_rpc_procedures_is_public ON rpc.procedures(is_public);

COMMENT ON TABLE rpc.procedures IS 'RPC procedure definitions with SQL queries and configuration';
COMMENT ON COLUMN rpc.procedures.sql_query IS 'The SQL query to execute (with $param_name placeholders)';
COMMENT ON COLUMN rpc.procedures.input_schema IS 'JSON Schema for input validation (null for schemaless)';
COMMENT ON COLUMN rpc.procedures.output_schema IS 'JSON Schema for output validation (null for schemaless)';
COMMENT ON COLUMN rpc.procedures.allowed_tables IS 'Tables the procedure can access (from @fluxbase:allowed-tables annotation)';
COMMENT ON COLUMN rpc.procedures.require_role IS 'Role required to invoke (authenticated, admin, anon, or null for any)';

-- ============================================================================
-- RPC EXECUTIONS
-- Execution history for auditing and debugging
-- ============================================================================

CREATE TABLE IF NOT EXISTS rpc.executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    procedure_id UUID REFERENCES rpc.procedures(id) ON DELETE SET NULL,
    procedure_name TEXT NOT NULL,
    namespace TEXT NOT NULL DEFAULT 'default',
    status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled', 'timeout')),

    -- Input/Output
    input_params JSONB,
    result JSONB,
    error_message TEXT,
    rows_returned INTEGER,

    -- Performance
    duration_ms INTEGER,

    -- User context
    user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    user_role TEXT,
    user_email TEXT,

    -- Execution mode
    is_async BOOLEAN DEFAULT false,

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_rpc_executions_procedure ON rpc.executions(procedure_id);
CREATE INDEX IF NOT EXISTS idx_rpc_executions_procedure_name ON rpc.executions(procedure_name);
CREATE INDEX IF NOT EXISTS idx_rpc_executions_namespace ON rpc.executions(namespace);
CREATE INDEX IF NOT EXISTS idx_rpc_executions_status ON rpc.executions(status);
CREATE INDEX IF NOT EXISTS idx_rpc_executions_user ON rpc.executions(user_id);
CREATE INDEX IF NOT EXISTS idx_rpc_executions_created ON rpc.executions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_rpc_executions_is_async ON rpc.executions(is_async) WHERE is_async = true;

COMMENT ON TABLE rpc.executions IS 'RPC execution history with input, output, and performance metrics';
COMMENT ON COLUMN rpc.executions.is_async IS 'Whether this was an async invocation (returns execution_id immediately)';

-- ============================================================================
-- RPC EXECUTION LOGS
-- Live streaming logs for execution progress via Realtime
-- ============================================================================

CREATE TABLE IF NOT EXISTS rpc.execution_logs (
    id BIGSERIAL PRIMARY KEY,
    execution_id UUID NOT NULL REFERENCES rpc.executions(id) ON DELETE CASCADE,
    line_number INTEGER NOT NULL,
    level TEXT NOT NULL DEFAULT 'info' CHECK (level IN ('debug', 'info', 'warn', 'error')),
    message TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rpc_execution_logs_execution ON rpc.execution_logs(execution_id);
CREATE INDEX IF NOT EXISTS idx_rpc_execution_logs_execution_line ON rpc.execution_logs(execution_id, line_number);

COMMENT ON TABLE rpc.execution_logs IS 'Individual log lines for RPC execution (streamed via Realtime)';

-- ============================================================================
-- ROW LEVEL SECURITY
-- ============================================================================

ALTER TABLE rpc.procedures ENABLE ROW LEVEL SECURITY;
ALTER TABLE rpc.executions ENABLE ROW LEVEL SECURITY;
ALTER TABLE rpc.execution_logs ENABLE ROW LEVEL SECURITY;

-- Service role can do everything (bypasses RLS)
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'rpc' AND tablename = 'procedures' AND policyname = 'rpc_procedures_service_all') THEN
        CREATE POLICY "rpc_procedures_service_all" ON rpc.procedures FOR ALL TO service_role USING (true);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'rpc' AND tablename = 'executions' AND policyname = 'rpc_executions_service_all') THEN
        CREATE POLICY "rpc_executions_service_all" ON rpc.executions FOR ALL TO service_role USING (true);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'rpc' AND tablename = 'execution_logs' AND policyname = 'rpc_execution_logs_service_all') THEN
        CREATE POLICY "rpc_execution_logs_service_all" ON rpc.execution_logs FOR ALL TO service_role USING (true);
    END IF;
END $$;

-- Authenticated users can read public, enabled procedures
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'rpc' AND tablename = 'procedures' AND policyname = 'rpc_procedures_read_public') THEN
        CREATE POLICY "rpc_procedures_read_public" ON rpc.procedures
            FOR SELECT TO authenticated
            USING (enabled = true AND is_public = true);
    END IF;
END $$;

-- Anonymous users can read public, enabled procedures
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'rpc' AND tablename = 'procedures' AND policyname = 'rpc_procedures_read_anon') THEN
        CREATE POLICY "rpc_procedures_read_anon" ON rpc.procedures
            FOR SELECT TO anon
            USING (enabled = true AND is_public = true);
    END IF;
END $$;

-- Users can read their own executions
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'rpc' AND tablename = 'executions' AND policyname = 'rpc_executions_read_own') THEN
        CREATE POLICY "rpc_executions_read_own" ON rpc.executions
            FOR SELECT TO authenticated
            USING (user_id = auth.current_user_id());
    END IF;
END $$;

-- Users can read logs for their own executions
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'rpc' AND tablename = 'execution_logs' AND policyname = 'rpc_execution_logs_read_own') THEN
        CREATE POLICY "rpc_execution_logs_read_own" ON rpc.execution_logs
            FOR SELECT TO authenticated
            USING (
                EXISTS (
                    SELECT 1 FROM rpc.executions
                    WHERE executions.id = execution_logs.execution_id
                    AND executions.user_id = auth.current_user_id()
                )
            );
    END IF;
END $$;

-- Dashboard admins can read all procedures
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'rpc' AND tablename = 'procedures' AND policyname = 'rpc_procedures_dashboard_admin_read') THEN
        CREATE POLICY "rpc_procedures_dashboard_admin_read" ON rpc.procedures
            FOR SELECT TO authenticated
            USING (auth.role() = 'dashboard_admin');
    END IF;
END $$;

-- Dashboard admins can read all executions
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'rpc' AND tablename = 'executions' AND policyname = 'rpc_executions_dashboard_admin_read') THEN
        CREATE POLICY "rpc_executions_dashboard_admin_read" ON rpc.executions
            FOR SELECT TO authenticated
            USING (auth.role() = 'dashboard_admin');
    END IF;
END $$;

-- Dashboard admins can read all execution logs
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'rpc' AND tablename = 'execution_logs' AND policyname = 'rpc_execution_logs_dashboard_admin_read') THEN
        CREATE POLICY "rpc_execution_logs_dashboard_admin_read" ON rpc.execution_logs
            FOR SELECT TO authenticated
            USING (auth.role() = 'dashboard_admin');
    END IF;
END $$;

-- ============================================================================
-- PERMISSIONS
-- ============================================================================

-- Grant permissions on rpc schema tables
GRANT SELECT ON rpc.procedures TO authenticated, anon;
GRANT ALL ON rpc.procedures TO service_role;

GRANT SELECT ON rpc.executions TO authenticated;
GRANT ALL ON rpc.executions TO service_role;

GRANT SELECT ON rpc.execution_logs TO authenticated;
GRANT ALL ON rpc.execution_logs TO service_role;

-- Grant sequence usage for execution_logs
GRANT USAGE, SELECT ON SEQUENCE rpc.execution_logs_id_seq TO service_role;

-- ============================================================================
-- REALTIME SUPPORT
-- ============================================================================

-- Set replica identity for UPDATE/DELETE payloads
ALTER TABLE rpc.executions REPLICA IDENTITY FULL;
ALTER TABLE rpc.execution_logs REPLICA IDENTITY FULL;

-- Note: procedures table is not included in realtime because SQL queries can be large

-- Create notify function for rpc schema
CREATE OR REPLACE FUNCTION rpc.notify_realtime_change()
RETURNS TRIGGER AS $$
DECLARE
  notification_record JSONB;
  old_notification_record JSONB;
BEGIN
  -- Build record without large fields for notification efficiency
  IF TG_OP != 'DELETE' THEN
    IF TG_TABLE_NAME = 'executions' THEN
      -- Exclude result and input_params (can be large)
      notification_record := to_jsonb(NEW) - 'result' - 'input_params';
    ELSE
      notification_record := to_jsonb(NEW);
    END IF;
  END IF;
  IF TG_OP != 'INSERT' THEN
    IF TG_TABLE_NAME = 'executions' THEN
      old_notification_record := to_jsonb(OLD) - 'result' - 'input_params';
    ELSE
      old_notification_record := to_jsonb(OLD);
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

-- Attach triggers for Realtime notifications
DROP TRIGGER IF EXISTS executions_realtime_notify ON rpc.executions;
CREATE TRIGGER executions_realtime_notify
AFTER INSERT OR UPDATE OR DELETE ON rpc.executions
FOR EACH ROW EXECUTE FUNCTION rpc.notify_realtime_change();

-- execution_logs only needs INSERT notifications (logs are append-only)
DROP TRIGGER IF EXISTS execution_logs_realtime_notify ON rpc.execution_logs;
CREATE TRIGGER execution_logs_realtime_notify
AFTER INSERT ON rpc.execution_logs
FOR EACH ROW EXECUTE FUNCTION rpc.notify_realtime_change();

-- Register tables for realtime in schema registry
INSERT INTO realtime.schema_registry (schema_name, table_name, realtime_enabled, events)
VALUES
    ('rpc', 'executions', true, ARRAY['INSERT', 'UPDATE', 'DELETE']),
    ('rpc', 'execution_logs', true, ARRAY['INSERT'])
ON CONFLICT (schema_name, table_name) DO UPDATE
SET realtime_enabled = true,
    events = EXCLUDED.events;

-- ============================================================================
-- AUTO-UPDATE TIMESTAMPS
-- ============================================================================

-- Create trigger function for updating timestamps
CREATE OR REPLACE FUNCTION rpc.update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply to procedures table
DROP TRIGGER IF EXISTS procedures_update_updated_at ON rpc.procedures;
CREATE TRIGGER procedures_update_updated_at
BEFORE UPDATE ON rpc.procedures
FOR EACH ROW EXECUTE FUNCTION rpc.update_updated_at();

-- ============================================================================
-- DEFAULT SETTINGS
-- Add RPC feature flags to app.settings
-- ============================================================================

INSERT INTO app.settings (key, value, value_type, category, description, is_public, is_secret, editable_by)
VALUES
    (
        'app.features.enable_rpc',
        '{"value": true}'::JSONB,
        'boolean',
        'system',
        'Enable or disable RPC procedure functionality',
        false,
        false,
        ARRAY['dashboard_admin']::TEXT[]
    ),
    (
        'app.rpc.default_max_execution_time_seconds',
        '{"value": 30}'::JSONB,
        'number',
        'system',
        'Default maximum execution time for RPC procedures in seconds',
        false,
        false,
        ARRAY['dashboard_admin']::TEXT[]
    ),
    (
        'app.rpc.max_max_execution_time_seconds',
        '{"value": 300}'::JSONB,
        'number',
        'system',
        'Maximum allowed execution time for RPC procedures in seconds',
        false,
        false,
        ARRAY['dashboard_admin']::TEXT[]
    ),
    (
        'app.rpc.default_max_rows',
        '{"value": 1000}'::JSONB,
        'number',
        'system',
        'Default maximum rows returned by RPC procedures',
        false,
        false,
        ARRAY['dashboard_admin']::TEXT[]
    )
ON CONFLICT (key) DO NOTHING;
