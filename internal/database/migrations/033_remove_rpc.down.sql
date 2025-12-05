-- Restore RPC function configuration table
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

COMMENT ON TABLE functions.rpc_function_config IS 'Configuration for database functions exposed via RPC API. Functions not in this table default to private.';
COMMENT ON COLUMN functions.rpc_function_config.is_public IS 'Whether the function can be called via the public RPC API';
