--
-- FUNCTIONS SCHEMA TABLES
-- Edge functions and their executions
--

-- Edge functions table (with allow_unauthenticated support and bundling)
CREATE TABLE IF NOT EXISTS functions.edge_functions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    namespace TEXT NOT NULL DEFAULT 'default',
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
    is_public BOOLEAN DEFAULT true,
    cron_schedule TEXT,
    version INTEGER DEFAULT 1,
    created_by UUID,
    source TEXT NOT NULL DEFAULT 'filesystem',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT unique_function_name_namespace UNIQUE (name, namespace)
);

CREATE INDEX IF NOT EXISTS idx_functions_edge_functions_name ON functions.edge_functions(name);
CREATE INDEX IF NOT EXISTS idx_functions_edge_functions_namespace ON functions.edge_functions(namespace);
CREATE INDEX IF NOT EXISTS idx_functions_edge_functions_enabled ON functions.edge_functions(enabled);
CREATE INDEX IF NOT EXISTS idx_functions_edge_functions_is_public ON functions.edge_functions(is_public);
CREATE INDEX IF NOT EXISTS idx_functions_edge_functions_cron_schedule ON functions.edge_functions(cron_schedule) WHERE cron_schedule IS NOT NULL;

COMMENT ON COLUMN functions.edge_functions.namespace IS 'Namespace for isolating functions across different apps/deployments. Functions with same name can exist in different namespaces.';
COMMENT ON COLUMN functions.edge_functions.allow_unauthenticated IS 'When true, allows this function to be invoked without authentication. Use with caution.';
COMMENT ON COLUMN functions.edge_functions.is_public IS 'Whether the function is publicly listed in the functions directory. Private functions can still be invoked if the name is known.';
COMMENT ON COLUMN functions.edge_functions.original_code IS 'Original source code before bundling (for editing in UI)';
COMMENT ON COLUMN functions.edge_functions.is_bundled IS 'Whether the code field contains bundled output with dependencies';
COMMENT ON COLUMN functions.edge_functions.bundle_error IS 'Error message if bundling failed (function still works with unbundled code)';
COMMENT ON COLUMN functions.edge_functions.source IS 'Source of function: filesystem or api';

-- Edge triggers table
CREATE TABLE IF NOT EXISTS functions.edge_triggers (
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

CREATE INDEX IF NOT EXISTS idx_functions_edge_triggers_function_id ON functions.edge_triggers(function_id);
CREATE INDEX IF NOT EXISTS idx_functions_edge_triggers_enabled ON functions.edge_triggers(enabled);
CREATE INDEX IF NOT EXISTS idx_functions_edge_triggers_table ON functions.edge_triggers(schema_name, table_name);

-- Edge executions table
CREATE TABLE IF NOT EXISTS functions.edge_executions (
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

CREATE INDEX IF NOT EXISTS idx_functions_edge_executions_function_id ON functions.edge_executions(function_id);
CREATE INDEX IF NOT EXISTS idx_functions_edge_executions_started_at ON functions.edge_executions(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_functions_edge_executions_status ON functions.edge_executions(status);

-- Edge files table (for multi-file functions)
CREATE TABLE IF NOT EXISTS functions.edge_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    function_id UUID NOT NULL REFERENCES functions.edge_functions(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_edge_file_path UNIQUE (function_id, file_path),
    CONSTRAINT valid_file_path CHECK (
        file_path ~ '^[a-zA-Z0-9_/-]+\.(ts|js|mts|mjs)$' AND
        file_path NOT LIKE '../%' AND
        file_path NOT LIKE '%/../%'
    )
);

CREATE INDEX IF NOT EXISTS idx_edge_files_function_id ON functions.edge_files(function_id);

COMMENT ON TABLE functions.edge_files IS 'Supporting files for edge functions (utils, helpers, types)';

-- Shared modules table (for _shared/* modules accessible by all functions)
CREATE TABLE IF NOT EXISTS functions.shared_modules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_path TEXT NOT NULL UNIQUE,
    content TEXT NOT NULL,
    description TEXT,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    CONSTRAINT valid_module_path CHECK (
        module_path ~ '^_shared/[a-zA-Z0-9_/-]+\.(ts|js|mts|mjs)$' AND
        module_path NOT LIKE '%/../%'
    )
);

CREATE INDEX IF NOT EXISTS idx_shared_modules_module_path ON functions.shared_modules(module_path);

COMMENT ON TABLE functions.shared_modules IS 'Shared modules accessible by all edge functions (_shared/*)';

-- Function dependencies table (tracks which functions use which shared modules)
CREATE TABLE IF NOT EXISTS functions.function_dependencies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    function_id UUID NOT NULL REFERENCES functions.edge_functions(id) ON DELETE CASCADE,
    shared_module_id UUID NOT NULL REFERENCES functions.shared_modules(id) ON DELETE CASCADE,
    shared_module_version INTEGER NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(function_id, shared_module_id)
);

CREATE INDEX IF NOT EXISTS idx_function_dependencies_function_id ON functions.function_dependencies(function_id);
CREATE INDEX IF NOT EXISTS idx_function_dependencies_shared_module_id ON functions.function_dependencies(shared_module_id);

COMMENT ON TABLE functions.function_dependencies IS 'Tracks which edge functions depend on which shared modules for automatic rebundling';

-- Add enhancements to edge_functions table
ALTER TABLE functions.edge_functions
    ADD COLUMN IF NOT EXISTS needs_rebundle BOOLEAN DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS cors_origins TEXT,
    ADD COLUMN IF NOT EXISTS cors_methods TEXT,
    ADD COLUMN IF NOT EXISTS cors_headers TEXT,
    ADD COLUMN IF NOT EXISTS cors_credentials BOOLEAN,
    ADD COLUMN IF NOT EXISTS cors_max_age INTEGER;

COMMENT ON COLUMN functions.edge_functions.needs_rebundle IS 'Flag indicating the function needs rebundling due to shared module updates';
COMMENT ON COLUMN functions.edge_functions.cors_origins IS 'Comma-separated list of allowed CORS origins (NULL means use global config)';
COMMENT ON COLUMN functions.edge_functions.cors_methods IS 'Comma-separated list of allowed CORS methods (NULL means use global config)';
COMMENT ON COLUMN functions.edge_functions.cors_headers IS 'Comma-separated list of allowed CORS headers (NULL means use global config)';
COMMENT ON COLUMN functions.edge_functions.cors_credentials IS 'Allow credentials in CORS requests (NULL means use global config)';
COMMENT ON COLUMN functions.edge_functions.cors_max_age IS 'Max age for CORS preflight cache in seconds (NULL means use global config)';

-- RPC function configuration table
-- Controls which database functions are exposed via the RPC API
CREATE TABLE IF NOT EXISTS functions.rpc_function_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schema_name TEXT NOT NULL,
    function_name TEXT NOT NULL,
    is_public BOOLEAN NOT NULL DEFAULT false,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT unique_rpc_function UNIQUE (schema_name, function_name)
);

CREATE INDEX IF NOT EXISTS idx_rpc_function_config_schema ON functions.rpc_function_config(schema_name);
CREATE INDEX IF NOT EXISTS idx_rpc_function_config_is_public ON functions.rpc_function_config(is_public);

COMMENT ON TABLE functions.rpc_function_config IS 'Configuration for database functions exposed via RPC API. Functions not in this table default to public.';
COMMENT ON COLUMN functions.rpc_function_config.is_public IS 'Whether the function can be called via the public RPC API (false = system function, not publicly callable)';
