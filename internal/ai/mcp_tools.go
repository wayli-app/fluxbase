package ai

import (
	"fmt"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/mcp"
)

// MCPToolMapping maps MCP tool names to their required scopes
var MCPToolMapping = map[string][]string{
	// Data tools
	"query_table":   {mcp.ScopeReadTables},
	"insert_record": {mcp.ScopeWriteTables},
	"update_record": {mcp.ScopeWriteTables},
	"delete_record": {mcp.ScopeWriteTables},
	"execute_sql":   {mcp.ScopeExecuteSQL},

	// Execution tools
	"invoke_function": {mcp.ScopeExecuteFunctions},
	"invoke_rpc":      {mcp.ScopeExecuteRPC},
	"submit_job":      {mcp.ScopeExecuteJobs},
	"get_job_status":  {mcp.ScopeExecuteJobs},

	// Storage tools
	"list_objects":    {mcp.ScopeReadStorage},
	"upload_object":   {mcp.ScopeWriteStorage},
	"download_object": {mcp.ScopeReadStorage},
	"delete_object":   {mcp.ScopeWriteStorage},

	// Vector search
	"search_vectors": {mcp.ScopeReadVectors},
	"vector_search":  {mcp.ScopeReadVectors}, // Alias for search_vectors (legacy chatbot configs)

	// HTTP requests
	"http_request": {mcp.ScopeExecuteHTTP},
}

// AllMCPTools returns all available MCP tool names
func AllMCPTools() []string {
	tools := make([]string, 0, len(MCPToolMapping))
	for tool := range MCPToolMapping {
		tools = append(tools, tool)
	}
	return tools
}

// ValidateMCPTools checks if all provided tool names are valid
func ValidateMCPTools(tools []string) error {
	invalid := []string{}
	for _, tool := range tools {
		if _, exists := MCPToolMapping[tool]; !exists {
			invalid = append(invalid, tool)
		}
	}
	if len(invalid) > 0 {
		return fmt.Errorf("invalid MCP tools: %s (valid tools: %s)",
			strings.Join(invalid, ", "),
			strings.Join(AllMCPTools(), ", "))
	}
	return nil
}

// DeriveScopes returns the unique scopes required for the given tools
func DeriveScopes(tools []string) []string {
	scopeSet := make(map[string]bool)
	for _, tool := range tools {
		if scopes, exists := MCPToolMapping[tool]; exists {
			for _, scope := range scopes {
				scopeSet[scope] = true
			}
		}
	}

	scopes := make([]string, 0, len(scopeSet))
	for scope := range scopeSet {
		scopes = append(scopes, scope)
	}
	return scopes
}

// GetToolScopes returns the scopes required for a specific tool
func GetToolScopes(tool string) ([]string, bool) {
	scopes, exists := MCPToolMapping[tool]
	return scopes, exists
}

// IsToolAllowed checks if a tool is in the allowed list
func IsToolAllowed(tool string, allowedTools []string) bool {
	if len(allowedTools) == 0 {
		return false // No tools allowed if list is empty
	}
	for _, allowed := range allowedTools {
		if allowed == tool {
			return true
		}
	}
	return false
}

// MCPToolCategory represents a category of MCP tools
type MCPToolCategory string

const (
	MCPToolCategoryData      MCPToolCategory = "data"
	MCPToolCategoryExecution MCPToolCategory = "execution"
	MCPToolCategoryStorage   MCPToolCategory = "storage"
	MCPToolCategoryVectors   MCPToolCategory = "vectors"
	MCPToolCategoryHTTP      MCPToolCategory = "http"
)

// MCPToolInfo contains information about an MCP tool
type MCPToolInfo struct {
	Name        string
	Description string
	Category    MCPToolCategory
	Scopes      []string
	ReadOnly    bool
}

// MCPToolInfoMap provides detailed information about each MCP tool
var MCPToolInfoMap = map[string]MCPToolInfo{
	// Data tools
	"query_table": {
		Name:        "query_table",
		Description: "Query a table with filters, ordering, and pagination",
		Category:    MCPToolCategoryData,
		Scopes:      []string{mcp.ScopeReadTables},
		ReadOnly:    true,
	},
	"insert_record": {
		Name:        "insert_record",
		Description: "Insert a new record into a table",
		Category:    MCPToolCategoryData,
		Scopes:      []string{mcp.ScopeWriteTables},
		ReadOnly:    false,
	},
	"update_record": {
		Name:        "update_record",
		Description: "Update records matching filters",
		Category:    MCPToolCategoryData,
		Scopes:      []string{mcp.ScopeWriteTables},
		ReadOnly:    false,
	},
	"delete_record": {
		Name:        "delete_record",
		Description: "Delete records matching filters",
		Category:    MCPToolCategoryData,
		Scopes:      []string{mcp.ScopeWriteTables},
		ReadOnly:    false,
	},
	"execute_sql": {
		Name:        "execute_sql",
		Description: "Execute a read-only SQL query against the database",
		Category:    MCPToolCategoryData,
		Scopes:      []string{mcp.ScopeExecuteSQL},
		ReadOnly:    true,
	},

	// Execution tools
	"invoke_function": {
		Name:        "invoke_function",
		Description: "Call an edge function with body and headers",
		Category:    MCPToolCategoryExecution,
		Scopes:      []string{mcp.ScopeExecuteFunctions},
		ReadOnly:    false,
	},
	"invoke_rpc": {
		Name:        "invoke_rpc",
		Description: "Execute an RPC procedure with parameters",
		Category:    MCPToolCategoryExecution,
		Scopes:      []string{mcp.ScopeExecuteRPC},
		ReadOnly:    false,
	},
	"submit_job": {
		Name:        "submit_job",
		Description: "Queue a background job for async execution",
		Category:    MCPToolCategoryExecution,
		Scopes:      []string{mcp.ScopeExecuteJobs},
		ReadOnly:    false,
	},
	"get_job_status": {
		Name:        "get_job_status",
		Description: "Check the status of a submitted job",
		Category:    MCPToolCategoryExecution,
		Scopes:      []string{mcp.ScopeExecuteJobs},
		ReadOnly:    true,
	},

	// Storage tools
	"list_objects": {
		Name:        "list_objects",
		Description: "List objects in a storage bucket",
		Category:    MCPToolCategoryStorage,
		Scopes:      []string{mcp.ScopeReadStorage},
		ReadOnly:    true,
	},
	"upload_object": {
		Name:        "upload_object",
		Description: "Upload a file to a storage bucket",
		Category:    MCPToolCategoryStorage,
		Scopes:      []string{mcp.ScopeWriteStorage},
		ReadOnly:    false,
	},
	"download_object": {
		Name:        "download_object",
		Description: "Download a file from a storage bucket",
		Category:    MCPToolCategoryStorage,
		Scopes:      []string{mcp.ScopeReadStorage},
		ReadOnly:    true,
	},
	"delete_object": {
		Name:        "delete_object",
		Description: "Delete a file from a storage bucket",
		Category:    MCPToolCategoryStorage,
		Scopes:      []string{mcp.ScopeWriteStorage},
		ReadOnly:    false,
	},

	// Vector search
	"search_vectors": {
		Name:        "search_vectors",
		Description: "Semantic search over vector embeddings",
		Category:    MCPToolCategoryVectors,
		Scopes:      []string{mcp.ScopeReadVectors},
		ReadOnly:    true,
	},
	"vector_search": {
		Name:        "vector_search", // Alias for search_vectors (legacy chatbot configs)
		Description: "Semantic search over vector embeddings",
		Category:    MCPToolCategoryVectors,
		Scopes:      []string{mcp.ScopeReadVectors},
		ReadOnly:    true,
	},

	// HTTP requests
	"http_request": {
		Name:        "http_request",
		Description: "Make HTTP GET requests to allowed external APIs",
		Category:    MCPToolCategoryHTTP,
		Scopes:      []string{mcp.ScopeExecuteHTTP},
		ReadOnly:    true, // GET requests don't modify data
	},
}

// GetToolsByCategory returns all tools in a given category
func GetToolsByCategory(category MCPToolCategory) []MCPToolInfo {
	tools := []MCPToolInfo{}
	for _, info := range MCPToolInfoMap {
		if info.Category == category {
			tools = append(tools, info)
		}
	}
	return tools
}

// GetReadOnlyTools returns all tools that don't modify data
func GetReadOnlyTools() []string {
	tools := []string{}
	for name, info := range MCPToolInfoMap {
		if info.ReadOnly {
			tools = append(tools, name)
		}
	}
	return tools
}

// FilterAllowedTools filters a list of tools to only include allowed ones
func FilterAllowedTools(tools []string, allowedTools []string) []string {
	if len(allowedTools) == 0 {
		return []string{}
	}

	allowedSet := make(map[string]bool, len(allowedTools))
	for _, t := range allowedTools {
		allowedSet[t] = true
	}

	filtered := []string{}
	for _, tool := range tools {
		if allowedSet[tool] {
			filtered = append(filtered, tool)
		}
	}
	return filtered
}
