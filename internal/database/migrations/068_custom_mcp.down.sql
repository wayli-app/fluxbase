-- Rollback custom MCP tools and resources

DROP TRIGGER IF EXISTS custom_resources_updated_at ON mcp.custom_resources;
DROP TRIGGER IF EXISTS custom_tools_updated_at ON mcp.custom_tools;
DROP FUNCTION IF EXISTS mcp.update_updated_at();

DROP TABLE IF EXISTS mcp.custom_resources;
DROP TABLE IF EXISTS mcp.custom_tools;

-- Note: We don't drop the mcp schema as it may contain other tables
