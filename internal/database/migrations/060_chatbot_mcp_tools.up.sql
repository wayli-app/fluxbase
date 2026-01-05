-- Add MCP tools configuration to chatbots
ALTER TABLE ai.chatbots ADD COLUMN IF NOT EXISTS mcp_tools TEXT[] DEFAULT ARRAY[]::TEXT[];
ALTER TABLE ai.chatbots ADD COLUMN IF NOT EXISTS use_mcp_schema BOOLEAN DEFAULT false;

-- Index for efficient filtering by MCP tools
CREATE INDEX IF NOT EXISTS idx_ai_chatbots_mcp_tools ON ai.chatbots USING GIN (mcp_tools);

COMMENT ON COLUMN ai.chatbots.mcp_tools IS 'List of MCP tools this chatbot can use (e.g., query_table, insert_record, invoke_function)';
COMMENT ON COLUMN ai.chatbots.use_mcp_schema IS 'If true, fetch schema from MCP resources instead of direct DB introspection';
