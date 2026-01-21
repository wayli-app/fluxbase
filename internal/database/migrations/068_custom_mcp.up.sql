-- Custom MCP Tools and Resources
-- Allows users to define custom MCP tools and resources using TypeScript

-- Create mcp schema if not exists
CREATE SCHEMA IF NOT EXISTS mcp;

-- Custom MCP Tools table
CREATE TABLE mcp.custom_tools (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(64) NOT NULL,
    namespace VARCHAR(64) NOT NULL DEFAULT 'default',
    description TEXT,

    -- Tool definition
    code TEXT NOT NULL,
    input_schema JSONB NOT NULL DEFAULT '{"type": "object", "properties": {}}',

    -- Execution settings
    required_scopes TEXT[] NOT NULL DEFAULT '{}',
    timeout_seconds INT NOT NULL DEFAULT 30,
    memory_limit_mb INT NOT NULL DEFAULT 128,

    -- Deno sandbox permissions
    allow_net BOOLEAN NOT NULL DEFAULT true,
    allow_env BOOLEAN NOT NULL DEFAULT false,
    allow_read BOOLEAN NOT NULL DEFAULT false,
    allow_write BOOLEAN NOT NULL DEFAULT false,

    -- Metadata
    enabled BOOLEAN NOT NULL DEFAULT true,
    version INT NOT NULL DEFAULT 1,
    created_by UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Unique constraint on name within namespace
    CONSTRAINT custom_tools_name_namespace_unique UNIQUE (name, namespace)
);

-- Custom MCP Resources table
CREATE TABLE mcp.custom_resources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    uri VARCHAR(255) NOT NULL,
    name VARCHAR(64) NOT NULL,
    namespace VARCHAR(64) NOT NULL DEFAULT 'default',
    description TEXT,
    mime_type VARCHAR(64) NOT NULL DEFAULT 'application/json',

    -- Resource definition
    code TEXT NOT NULL,
    is_template BOOLEAN NOT NULL DEFAULT false,

    -- Security
    required_scopes TEXT[] NOT NULL DEFAULT '{}',

    -- Execution settings
    timeout_seconds INT NOT NULL DEFAULT 10,
    cache_ttl_seconds INT NOT NULL DEFAULT 60,

    -- Metadata
    enabled BOOLEAN NOT NULL DEFAULT true,
    version INT NOT NULL DEFAULT 1,
    created_by UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Unique constraint on URI within namespace
    CONSTRAINT custom_resources_uri_namespace_unique UNIQUE (uri, namespace)
);

-- Indexes for efficient querying
CREATE INDEX idx_custom_tools_namespace ON mcp.custom_tools(namespace);
CREATE INDEX idx_custom_tools_enabled ON mcp.custom_tools(enabled) WHERE enabled = true;
CREATE INDEX idx_custom_tools_name ON mcp.custom_tools(name);

CREATE INDEX idx_custom_resources_namespace ON mcp.custom_resources(namespace);
CREATE INDEX idx_custom_resources_enabled ON mcp.custom_resources(enabled) WHERE enabled = true;
CREATE INDEX idx_custom_resources_uri ON mcp.custom_resources(uri);

-- Trigger for updated_at
CREATE OR REPLACE FUNCTION mcp.update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER custom_tools_updated_at
    BEFORE UPDATE ON mcp.custom_tools
    FOR EACH ROW
    EXECUTE FUNCTION mcp.update_updated_at();

CREATE TRIGGER custom_resources_updated_at
    BEFORE UPDATE ON mcp.custom_resources
    FOR EACH ROW
    EXECUTE FUNCTION mcp.update_updated_at();

-- Comments for documentation
COMMENT ON TABLE mcp.custom_tools IS 'User-defined MCP tools implemented in TypeScript';
COMMENT ON TABLE mcp.custom_resources IS 'User-defined MCP resources implemented in TypeScript';

COMMENT ON COLUMN mcp.custom_tools.code IS 'TypeScript code implementing the tool handler';
COMMENT ON COLUMN mcp.custom_tools.input_schema IS 'JSON Schema defining the tool input parameters';
COMMENT ON COLUMN mcp.custom_tools.required_scopes IS 'MCP scopes required to execute this tool';
COMMENT ON COLUMN mcp.custom_tools.allow_net IS 'Allow network access in Deno sandbox';
COMMENT ON COLUMN mcp.custom_tools.allow_env IS 'Allow environment variable access in Deno sandbox';

COMMENT ON COLUMN mcp.custom_resources.uri IS 'MCP resource URI (e.g., fluxbase://custom/myresource)';
COMMENT ON COLUMN mcp.custom_resources.is_template IS 'Whether URI contains parameters (e.g., {id})';
COMMENT ON COLUMN mcp.custom_resources.cache_ttl_seconds IS 'How long to cache resource responses';
