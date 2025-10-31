-- Edge Functions Schema
-- Create tables for storing and managing Deno-based edge functions

-- Edge functions table
CREATE TABLE IF NOT EXISTS edge_functions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT UNIQUE NOT NULL CHECK (name ~ '^[a-z0-9_-]+$'),
    description TEXT,
    code TEXT NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,

    -- Scheduling
    cron_schedule TEXT, -- Cron expression for scheduled execution (e.g., "*/5 * * * *")

    -- Configuration
    enabled BOOLEAN NOT NULL DEFAULT true,
    timeout_seconds INTEGER NOT NULL DEFAULT 30 CHECK (timeout_seconds > 0 AND timeout_seconds <= 300),
    memory_limit_mb INTEGER NOT NULL DEFAULT 128 CHECK (memory_limit_mb > 0 AND memory_limit_mb <= 1024),

    -- Permissions (Deno security model)
    allow_net BOOLEAN NOT NULL DEFAULT true,
    allow_env BOOLEAN NOT NULL DEFAULT true,
    allow_read BOOLEAN NOT NULL DEFAULT false,
    allow_write BOOLEAN NOT NULL DEFAULT false,

    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES auth.users(id) ON DELETE SET NULL,

    CONSTRAINT valid_cron_schedule CHECK (cron_schedule IS NULL OR length(cron_schedule) > 0)
);

-- Function execution logs
CREATE TABLE IF NOT EXISTS edge_function_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    function_id UUID NOT NULL REFERENCES edge_functions(id) ON DELETE CASCADE,

    -- Execution details
    trigger_type TEXT NOT NULL CHECK (trigger_type IN ('http', 'cron', 'database', 'manual')),
    trigger_payload JSONB, -- Context about what triggered the execution

    -- Results
    status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'success', 'error', 'timeout')),
    status_code INTEGER, -- HTTP status code returned by function
    duration_ms INTEGER,

    -- Output
    result JSONB, -- Function return value
    logs TEXT, -- Combined stdout/stderr
    error_message TEXT,
    error_stack TEXT,

    -- Metadata
    executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,

    -- Indexes for performance
    CONSTRAINT execution_duration_positive CHECK (duration_ms IS NULL OR duration_ms >= 0)
);

-- Database triggers for edge functions
CREATE TABLE IF NOT EXISTS edge_function_triggers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    function_id UUID NOT NULL REFERENCES edge_functions(id) ON DELETE CASCADE,

    -- Trigger configuration
    table_schema TEXT NOT NULL DEFAULT 'public',
    table_name TEXT NOT NULL,
    events TEXT[] NOT NULL CHECK (array_length(events, 1) > 0), -- ['INSERT', 'UPDATE', 'DELETE']

    -- Filtering
    condition TEXT, -- SQL condition for when to trigger (e.g., "NEW.status = 'published'")

    -- Metadata
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Ensure unique triggers per table/events combination
    CONSTRAINT unique_function_table_events UNIQUE (function_id, table_schema, table_name)
);

-- Indexes for performance
CREATE INDEX idx_edge_functions_enabled ON edge_functions(enabled) WHERE enabled = true;
CREATE INDEX idx_edge_functions_cron ON edge_functions(cron_schedule) WHERE cron_schedule IS NOT NULL AND enabled = true;
CREATE INDEX idx_edge_function_executions_function_id ON edge_function_executions(function_id);
CREATE INDEX idx_edge_function_executions_executed_at ON edge_function_executions(executed_at DESC);
CREATE INDEX idx_edge_function_executions_status ON edge_function_executions(status);
CREATE INDEX idx_edge_function_triggers_table ON edge_function_triggers(table_schema, table_name) WHERE enabled = true;

-- Update trigger for updated_at
CREATE OR REPLACE FUNCTION update_edge_function_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    NEW.version = OLD.version + 1;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER edge_functions_updated_at
    BEFORE UPDATE ON edge_functions
    FOR EACH ROW
    EXECUTE FUNCTION update_edge_function_updated_at();

-- Function to clean up old execution logs (keep last 30 days)
CREATE OR REPLACE FUNCTION cleanup_old_edge_function_executions()
RETURNS void AS $$
BEGIN
    DELETE FROM edge_function_executions
    WHERE executed_at < NOW() - INTERVAL '30 days';
END;
$$ LANGUAGE plpgsql;

-- Grant permissions
GRANT SELECT, INSERT, UPDATE, DELETE ON edge_functions TO authenticated;
GRANT SELECT ON edge_function_executions TO authenticated;
GRANT SELECT ON edge_function_triggers TO authenticated;

-- Comments for documentation
COMMENT ON TABLE edge_functions IS 'Stores Deno-based serverless functions';
COMMENT ON TABLE edge_function_executions IS 'Logs all function execution attempts with results';
COMMENT ON TABLE edge_function_triggers IS 'Database triggers that invoke edge functions on table changes';
COMMENT ON COLUMN edge_functions.code IS 'TypeScript/JavaScript code to execute in Deno runtime';
COMMENT ON COLUMN edge_functions.cron_schedule IS 'Cron expression for scheduled execution (e.g., "0 */1 * * *" for hourly)';
COMMENT ON COLUMN edge_function_executions.trigger_type IS 'How the function was invoked: http (API call), cron (scheduled), database (trigger), manual (admin)';
COMMENT ON COLUMN edge_function_executions.logs IS 'Combined stdout and stderr output from the function';
