package ai

import (
	"testing"

	"github.com/fluxbase-eu/fluxbase/internal/mcp"
	"github.com/stretchr/testify/assert"
)

func TestValidateMCPTools(t *testing.T) {
	t.Run("valid tools", func(t *testing.T) {
		err := ValidateMCPTools([]string{"query_table", "insert_record", "invoke_function"})
		assert.NoError(t, err)
	})

	t.Run("empty list is valid", func(t *testing.T) {
		err := ValidateMCPTools([]string{})
		assert.NoError(t, err)
	})

	t.Run("invalid tool returns error", func(t *testing.T) {
		err := ValidateMCPTools([]string{"query_table", "invalid_tool"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid_tool")
	})

	t.Run("multiple invalid tools", func(t *testing.T) {
		err := ValidateMCPTools([]string{"foo", "bar"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "foo")
		assert.Contains(t, err.Error(), "bar")
	})
}

func TestDeriveScopes(t *testing.T) {
	t.Run("empty tools returns empty scopes", func(t *testing.T) {
		scopes := DeriveScopes([]string{})
		assert.Empty(t, scopes)
	})

	t.Run("single tool returns its scope", func(t *testing.T) {
		scopes := DeriveScopes([]string{"query_table"})
		assert.Contains(t, scopes, mcp.ScopeReadTables)
	})

	t.Run("multiple tools with same scope deduplicates", func(t *testing.T) {
		scopes := DeriveScopes([]string{"insert_record", "update_record", "delete_record"})
		// All should require ScopeWriteTables
		assert.Len(t, scopes, 1)
		assert.Contains(t, scopes, mcp.ScopeWriteTables)
	})

	t.Run("multiple tools with different scopes", func(t *testing.T) {
		scopes := DeriveScopes([]string{"query_table", "insert_record", "invoke_function", "list_objects"})
		assert.Contains(t, scopes, mcp.ScopeReadTables)
		assert.Contains(t, scopes, mcp.ScopeWriteTables)
		assert.Contains(t, scopes, mcp.ScopeExecuteFunctions)
		assert.Contains(t, scopes, mcp.ScopeReadStorage)
	})

	t.Run("ignores invalid tools", func(t *testing.T) {
		scopes := DeriveScopes([]string{"query_table", "invalid_tool"})
		assert.Len(t, scopes, 1)
		assert.Contains(t, scopes, mcp.ScopeReadTables)
	})
}

func TestGetToolScopes(t *testing.T) {
	t.Run("valid tool returns scopes", func(t *testing.T) {
		scopes, exists := GetToolScopes("query_table")
		assert.True(t, exists)
		assert.Equal(t, []string{mcp.ScopeReadTables}, scopes)
	})

	t.Run("invalid tool returns false", func(t *testing.T) {
		scopes, exists := GetToolScopes("nonexistent")
		assert.False(t, exists)
		assert.Nil(t, scopes)
	})
}

func TestIsToolAllowed(t *testing.T) {
	t.Run("empty allowed list returns false", func(t *testing.T) {
		assert.False(t, IsToolAllowed("query_table", []string{}))
	})

	t.Run("tool in allowed list", func(t *testing.T) {
		assert.True(t, IsToolAllowed("query_table", []string{"query_table", "insert_record"}))
	})

	t.Run("tool not in allowed list", func(t *testing.T) {
		assert.False(t, IsToolAllowed("delete_record", []string{"query_table", "insert_record"}))
	})
}

func TestAllMCPTools(t *testing.T) {
	tools := AllMCPTools()

	// Check that all expected tools are present
	expectedTools := []string{
		"query_table", "insert_record", "update_record", "delete_record", "execute_sql",
		"invoke_function", "invoke_rpc", "submit_job", "get_job_status",
		"list_objects", "upload_object", "download_object", "delete_object",
		"search_vectors", "vector_search", "http_request",
	}

	assert.Len(t, tools, len(expectedTools))
	for _, expected := range expectedTools {
		assert.Contains(t, tools, expected)
	}
}

func TestMCPToolInfoMap(t *testing.T) {
	t.Run("all tools have info", func(t *testing.T) {
		for tool := range MCPToolMapping {
			info, exists := MCPToolInfoMap[tool]
			assert.True(t, exists, "tool %s should have info", tool)
			assert.Equal(t, tool, info.Name)
			assert.NotEmpty(t, info.Description)
			assert.NotEmpty(t, info.Category)
			assert.NotEmpty(t, info.Scopes)
		}
	})

	t.Run("query_table is read-only", func(t *testing.T) {
		info := MCPToolInfoMap["query_table"]
		assert.True(t, info.ReadOnly)
	})

	t.Run("insert_record is not read-only", func(t *testing.T) {
		info := MCPToolInfoMap["insert_record"]
		assert.False(t, info.ReadOnly)
	})
}

func TestGetToolsByCategory(t *testing.T) {
	t.Run("data tools", func(t *testing.T) {
		tools := GetToolsByCategory(MCPToolCategoryData)
		assert.Len(t, tools, 5) // query, insert, update, delete, execute_sql
	})

	t.Run("execution tools", func(t *testing.T) {
		tools := GetToolsByCategory(MCPToolCategoryExecution)
		assert.Len(t, tools, 4) // invoke_function, invoke_rpc, submit_job, get_job_status
	})

	t.Run("storage tools", func(t *testing.T) {
		tools := GetToolsByCategory(MCPToolCategoryStorage)
		assert.Len(t, tools, 4) // list, upload, download, delete
	})

	t.Run("vector tools", func(t *testing.T) {
		tools := GetToolsByCategory(MCPToolCategoryVectors)
		assert.Len(t, tools, 2) // search_vectors, vector_search (alias)
	})
}

func TestGetReadOnlyTools(t *testing.T) {
	tools := GetReadOnlyTools()

	// Check expected read-only tools
	assert.Contains(t, tools, "query_table")
	assert.Contains(t, tools, "get_job_status")
	assert.Contains(t, tools, "list_objects")
	assert.Contains(t, tools, "download_object")
	assert.Contains(t, tools, "search_vectors")

	// Check that write tools are not included
	assert.NotContains(t, tools, "insert_record")
	assert.NotContains(t, tools, "update_record")
	assert.NotContains(t, tools, "delete_record")
	assert.NotContains(t, tools, "upload_object")
	assert.NotContains(t, tools, "delete_object")
}

func TestFilterAllowedTools(t *testing.T) {
	t.Run("empty allowed list returns empty", func(t *testing.T) {
		result := FilterAllowedTools([]string{"query_table", "insert_record"}, []string{})
		assert.Empty(t, result)
	})

	t.Run("filters to allowed tools only", func(t *testing.T) {
		result := FilterAllowedTools(
			[]string{"query_table", "insert_record", "delete_record"},
			[]string{"query_table", "insert_record"},
		)
		assert.Len(t, result, 2)
		assert.Contains(t, result, "query_table")
		assert.Contains(t, result, "insert_record")
		assert.NotContains(t, result, "delete_record")
	})

	t.Run("empty tools returns empty", func(t *testing.T) {
		result := FilterAllowedTools([]string{}, []string{"query_table"})
		assert.Empty(t, result)
	})
}
