package ai

import (
	"github.com/fluxbase-eu/fluxbase/internal/mcp"
)

// ChatbotAuthContext creates an MCP AuthContext from a ChatContext and Chatbot configuration.
// This bridges the chatbot's authentication context to MCP's authorization system.
func ChatbotAuthContext(chatCtx *ChatContext, chatbot *Chatbot) *mcp.AuthContext {
	// Derive scopes from chatbot's allowed MCP tools
	scopes := DeriveScopes(chatbot.MCPTools)

	var userID *string
	if chatCtx.UserID != nil {
		userID = chatCtx.UserID
	}

	// Build metadata with chatbot-specific configuration
	metadata := make(map[string]any)
	if len(chatbot.HTTPAllowedDomains) > 0 {
		metadata[mcp.MetadataKeyHTTPAllowedDomains] = chatbot.HTTPAllowedDomains
	}

	return &mcp.AuthContext{
		UserID:   userID,
		UserRole: chatCtx.Role,
		AuthType: "chatbot",
		Scopes:   scopes,
		// Chatbots never bypass RLS - they always operate as the authenticated user
		IsServiceRole: false,
		// AllowedNamespaces is nil - chatbots don't have namespace restrictions
		// They operate on the tables configured in allowed-tables/allowed-schemas
		AllowedNamespaces: nil,
		// Pass chatbot-specific configuration in metadata
		Metadata: metadata,
	}
}

// ChatbotAuthContextWithMetadata creates an MCP AuthContext with additional metadata
// for table filtering. This is useful when MCP tools need to enforce table restrictions.
type ChatbotAuthContextOptions struct {
	// AllowedTables restricts which tables the chatbot can access
	AllowedTables []string
	// AllowedSchemas restricts which schemas the chatbot can access
	AllowedSchemas []string
	// DefaultSchema is used when table names don't include a schema prefix
	DefaultSchema string
}

// NewChatbotAuthContext creates an MCP AuthContext with chatbot-specific options.
func NewChatbotAuthContext(chatCtx *ChatContext, chatbot *Chatbot, opts *ChatbotAuthContextOptions) *mcp.AuthContext {
	authCtx := ChatbotAuthContext(chatCtx, chatbot)

	// If options are provided, we could extend AuthContext with metadata
	// For now, the table filtering is handled by the MCP executor
	// which validates table access before executing tools

	return authCtx
}

// ValidateChatbotToolAccess checks if a chatbot is allowed to use a specific tool.
// Returns an error if the tool is not in the chatbot's allowed tools list.
func ValidateChatbotToolAccess(chatbot *Chatbot, toolName string) error {
	if !chatbot.HasMCPTools() {
		return nil // No MCP tools configured, use legacy execute_sql
	}

	if !IsToolAllowed(toolName, chatbot.MCPTools) {
		return &ToolNotAllowedError{
			Tool:         toolName,
			AllowedTools: chatbot.MCPTools,
		}
	}

	return nil
}

// ToolNotAllowedError is returned when a chatbot tries to use a tool it's not configured for
type ToolNotAllowedError struct {
	Tool         string
	AllowedTools []string
}

func (e *ToolNotAllowedError) Error() string {
	return "tool '" + e.Tool + "' is not allowed for this chatbot"
}

// IsTableAllowed checks if a table name is in the chatbot's allowed tables.
// It handles both simple table names and qualified schema.table names.
func IsTableAllowed(tableName string, chatbot *Chatbot) bool {
	// Parse the table name to handle qualified names
	qualifiedTables := ParseQualifiedTables(chatbot.AllowedTables, "public")

	// Group by schema for efficient lookup
	schemaTableMap := GroupTablesBySchema(qualifiedTables)

	// Parse the requested table name
	requestedTables := ParseQualifiedTables([]string{tableName}, "public")
	if len(requestedTables) == 0 {
		return false
	}

	requested := requestedTables[0]

	// Check if the schema is allowed
	if len(chatbot.AllowedSchemas) > 0 {
		schemaAllowed := false
		for _, s := range chatbot.AllowedSchemas {
			if s == requested.Schema {
				schemaAllowed = true
				break
			}
		}
		// If schemas are specified and this schema isn't in the list,
		// check if there's a specific table allowance
		if !schemaAllowed {
			// Check if there's an explicit table allowance for this schema
			if _, hasExplicit := schemaTableMap[requested.Schema]; !hasExplicit {
				return false
			}
		}
	}

	// Check if the specific table is allowed
	if tables, exists := schemaTableMap[requested.Schema]; exists {
		for _, t := range tables {
			if t == requested.Table {
				return true
			}
		}
	}

	// If no specific tables are configured for this schema,
	// check if the entire schema is allowed
	if len(chatbot.AllowedSchemas) > 0 && len(schemaTableMap[requested.Schema]) == 0 {
		for _, s := range chatbot.AllowedSchemas {
			if s == requested.Schema {
				return true
			}
		}
	}

	return false
}

// TableNotAllowedError is returned when a chatbot tries to access a table it's not configured for
type TableNotAllowedError struct {
	Table         string
	AllowedTables []string
}

func (e *TableNotAllowedError) Error() string {
	return "table '" + e.Table + "' is not allowed for this chatbot"
}
