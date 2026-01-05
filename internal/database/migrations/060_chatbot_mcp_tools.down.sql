-- Remove MCP tools configuration from chatbots
DROP INDEX IF EXISTS idx_ai_chatbots_mcp_tools;
ALTER TABLE ai.chatbots DROP COLUMN IF EXISTS use_mcp_schema;
ALTER TABLE ai.chatbots DROP COLUMN IF EXISTS mcp_tools;
