--
-- FUNCTIONS SCHEMA TABLES
-- Edge functions and their executions
--

-- Edge functions table (with allow_unauthenticated support and bundling)
CREATE TABLE IF NOT EXISTS functions.edge_functions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT UNIQUE NOT NULL,
    description TEXT,
    code TEXT NOT NULL,
    original_code TEXT,
    is_bundled BOOLEAN DEFAULT false NOT NULL,
    bundle_error TEXT,
    enabled BOOLEAN DEFAULT true,
    timeout_seconds INTEGER DEFAULT 30,
    memory_limit_mb INTEGER DEFAULT 128,
    allow_net BOOLEAN DEFAULT true,
    allow_env BOOLEAN DEFAULT true,
    allow_read BOOLEAN DEFAULT false,
    allow_write BOOLEAN DEFAULT false,
    allow_unauthenticated BOOLEAN DEFAULT false,
    cron_schedule TEXT,
    version INTEGER DEFAULT 1,
    created_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_functions_edge_functions_name ON functions.edge_functions(name);
CREATE INDEX IF NOT EXISTS idx_functions_edge_functions_enabled ON functions.edge_functions(enabled);
CREATE INDEX IF NOT EXISTS idx_functions_edge_functions_cron_schedule ON functions.edge_functions(cron_schedule) WHERE cron_schedule IS NOT NULL;

COMMENT ON COLUMN functions.edge_functions.allow_unauthenticated IS 'When true, allows this function to be invoked without authentication. Use with caution.';
COMMENT ON COLUMN functions.edge_functions.original_code IS 'Original source code before bundling (for editing in UI)';
COMMENT ON COLUMN functions.edge_functions.is_bundled IS 'Whether the code field contains bundled output with dependencies';
COMMENT ON COLUMN functions.edge_functions.bundle_error IS 'Error message if bundling failed (function still works with unbundled code)';

-- Edge function triggers table
CREATE TABLE IF NOT EXISTS functions.edge_function_triggers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    function_id UUID REFERENCES functions.edge_functions(id) ON DELETE CASCADE NOT NULL,
    trigger_type TEXT NOT NULL,
    schema_name TEXT,
    table_name TEXT,
    events TEXT[] DEFAULT ARRAY[]::TEXT[],
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_functions_edge_function_triggers_function_id ON functions.edge_function_triggers(function_id);
CREATE INDEX IF NOT EXISTS idx_functions_edge_function_triggers_enabled ON functions.edge_function_triggers(enabled);
CREATE INDEX IF NOT EXISTS idx_functions_edge_function_triggers_table ON functions.edge_function_triggers(schema_name, table_name);

-- Edge function executions table
CREATE TABLE IF NOT EXISTS functions.edge_function_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    function_id UUID REFERENCES functions.edge_functions(id) ON DELETE CASCADE NOT NULL,
    trigger_type TEXT NOT NULL,
    status TEXT NOT NULL,
    status_code INTEGER,
    error_message TEXT,
    logs TEXT,
    result TEXT,
    duration_ms INTEGER,
    started_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_functions_edge_function_executions_function_id ON functions.edge_function_executions(function_id);
CREATE INDEX IF NOT EXISTS idx_functions_edge_function_executions_started_at ON functions.edge_function_executions(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_functions_edge_function_executions_status ON functions.edge_function_executions(status);
