package ai

import (
	"fmt"
	"strings"

	"github.com/nimbleflux/fluxbase/internal/mcp"
)

// MCPToolMapping maps MCP tool names to their required scopes
var MCPToolMapping = map[string][]string{
	// Reasoning tools (no scopes required - always available)
	"think": {},

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
		// Custom tools (prefix "custom:") require the execute:custom scope,
		// which the custom tool handler enforces via RequiredScopes. Without
		// emitting it here, the derived AuthContext.Scopes omits it and a
		// correctly-exposed custom tool is rejected at execution time.
		if strings.HasPrefix(tool, "custom:") {
			scopeSet["execute:custom"] = true
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
	MCPToolCategoryReasoning MCPToolCategory = "reasoning"
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
	// Reasoning tools
	"think": {
		Name: "think",
		Description: `Plan your approach before executing queries. Use this to analyze the question and decide which tool(s) to use.

WHEN TO USE:
- Before executing any data query
- When the question is complex or ambiguous
- To decide between SQL (query_table/execute_sql) vs KB search (search_vectors)
- To plan multi-step query strategies

WHEN NOT TO USE:
- For simple conversational responses that don't need data

EXAMPLE: Before querying for "Italian restaurants visited last month", use think to plan:
1. Use search_vectors for Italian cuisine context
2. Use query_table for visits with date filter`,
		Category: MCPToolCategoryReasoning,
		Scopes:   []string{}, // No scopes required - always available
		ReadOnly: true,
	},

	// Data tools
	"query_table": {
		Name: "query_table",
		Description: `Query a database table with filters, ordering, and pagination.

WHEN TO USE:
- You need specific records matching criteria
- Filtering by dates, numbers, or exact values
- Ordering or paginating through results
- Looking for user-owned data (filtered by user_id)

WHEN NOT TO USE:
- Conceptual information (use search_vectors instead)
- Full-text search on descriptions (use search_vectors instead)
- Complex multi-table joins (use execute_sql instead)

EXAMPLE: "restaurants visited last week" → query_table with date filter`,
		Category: MCPToolCategoryData,
		Scopes:   []string{mcp.ScopeReadTables},
		ReadOnly: true,
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
		Name: "execute_sql",
		Description: `Execute a custom SQL SELECT query against the database.

WHEN TO USE:
- Complex queries spanning multiple tables (JOINs)
- Aggregations (COUNT, SUM, AVG, GROUP BY)
- Queries that can't be expressed with simple filters
- Need precise control over the query structure

WHEN NOT TO USE:
- Simple single-table queries (use query_table instead)
- When you don't know the table structure well
- For conceptual information (use search_vectors instead)

EXAMPLE: "Count visits by city" → SELECT city, COUNT(*) FROM visits GROUP BY city`,
		Category: MCPToolCategoryData,
		Scopes:   []string{mcp.ScopeExecuteSQL},
		ReadOnly: true,
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
		Name: "search_vectors",
		Description: `Semantic search over knowledge base documents using vector embeddings.

WHEN TO USE:
- Conceptual or descriptive questions
- Questions starting with "what is", "tell me about", "explain"
- When the answer depends on document context
- Topic-based searches for information

WHEN NOT TO USE:
- Exact date/number filtering (use query_table instead)
- Counting records (use execute_sql instead)
- Listing specific user records (use query_table instead)
- When you need precise structured data

EXAMPLE: "Tell me about Italian cuisine" → search_vectors for cuisine concepts`,
		Category: MCPToolCategoryVectors,
		Scopes:   []string{mcp.ScopeReadVectors},
		ReadOnly: true,
	},
	"vector_search": {
		Name: "vector_search", // Alias for search_vectors (legacy chatbot configs)
		Description: `Semantic search over knowledge base documents using vector embeddings.

WHEN TO USE:
- Conceptual or descriptive questions
- Questions starting with "what is", "tell me about", "explain"
- When the answer depends on document context

WHEN NOT TO USE:
- Exact date/number filtering (use query_table instead)
- Counting or listing specific records (use query_table instead)

EXAMPLE: "Tell me about Italian cuisine" → vector_search for cuisine concepts`,
		Category: MCPToolCategoryVectors,
		Scopes:   []string{mcp.ScopeReadVectors},
		ReadOnly: true,
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
