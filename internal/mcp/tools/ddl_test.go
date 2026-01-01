package tools

import (
	"testing"

	"github.com/fluxbase-eu/fluxbase/internal/mcp"
	"github.com/stretchr/testify/assert"
)

func TestValidateDDLIdentifier(t *testing.T) {
	t.Run("valid identifiers pass validation", func(t *testing.T) {
		validNames := []string{
			"users",
			"user_accounts",
			"_private",
			"Table1",
			"a",
			"_",
			"users_v2",
			"snake_case_name",
			"CamelCase",
			"mixedCase123",
		}

		for _, name := range validNames {
			t.Run(name, func(t *testing.T) {
				err := validateDDLIdentifier(name, "table")
				assert.NoError(t, err, "identifier '%s' should be valid", name)
			})
		}
	})

	t.Run("empty name rejected", func(t *testing.T) {
		err := validateDDLIdentifier("", "table")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("names exceeding 63 characters rejected", func(t *testing.T) {
		longName := "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz1234567890ab" // 64 characters
		err := validateDDLIdentifier(longName, "table")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot exceed 63 characters")
	})

	t.Run("name at 63 characters accepted", func(t *testing.T) {
		name := "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz12345678901"
		assert.Len(t, name, 63)
		err := validateDDLIdentifier(name, "table")
		assert.NoError(t, err)
	})

	t.Run("names starting with number rejected", func(t *testing.T) {
		invalidNames := []string{
			"1users",
			"123",
			"0_table",
		}

		for _, name := range invalidNames {
			t.Run(name, func(t *testing.T) {
				err := validateDDLIdentifier(name, "table")
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "must start with a letter or underscore")
			})
		}
	})

	t.Run("names with invalid characters rejected", func(t *testing.T) {
		invalidNames := []string{
			"user-name",
			"user.name",
			"user name",
			"user@email",
			"table$1",
			"drop;--",
			"user's",
			"table\"name",
		}

		for _, name := range invalidNames {
			t.Run(name, func(t *testing.T) {
				err := validateDDLIdentifier(name, "table")
				assert.Error(t, err)
			})
		}
	})

	t.Run("reserved keywords rejected", func(t *testing.T) {
		reservedNames := []string{
			"user",
			"table",
			"column",
			"index",
			"select",
			"insert",
			"update",
			"delete",
		}

		for _, name := range reservedNames {
			t.Run(name, func(t *testing.T) {
				err := validateDDLIdentifier(name, "table")
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "reserved keyword")
			})
		}
	})

	t.Run("reserved keywords case insensitive", func(t *testing.T) {
		testCases := []string{
			"USER",
			"User",
			"SELECT",
			"Select",
			"TABLE",
			"Table",
		}

		for _, name := range testCases {
			t.Run(name, func(t *testing.T) {
				err := validateDDLIdentifier(name, "table")
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "reserved keyword")
			})
		}
	})
}

func TestIsSystemSchema(t *testing.T) {
	t.Run("system schemas identified", func(t *testing.T) {
		systemSchemasList := []string{
			"auth",
			"storage",
			"jobs",
			"functions",
			"branching",
			"information_schema",
			"pg_catalog",
			"pg_toast",
		}

		for _, schema := range systemSchemasList {
			t.Run(schema, func(t *testing.T) {
				assert.True(t, isSystemSchema(schema), "%s should be a system schema", schema)
			})
		}
	})

	t.Run("user schemas not identified as system", func(t *testing.T) {
		userSchemas := []string{
			"public",
			"my_schema",
			"custom",
			"app",
		}

		for _, schema := range userSchemas {
			t.Run(schema, func(t *testing.T) {
				assert.False(t, isSystemSchema(schema), "%s should not be a system schema", schema)
			})
		}
	})
}

func TestEscapeDDLLiteral(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"hello", "'hello'"},
		{"", "''"},
		{"O'Brien", "'O''Brien'"},
		{"it's", "'it''s'"},
		{"quote'test'value", "'quote''test''value'"},
		{"no quotes", "'no quotes'"},
		{"123", "'123'"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := escapeDDLLiteral(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestValidDataTypes(t *testing.T) {
	t.Run("all valid data types are accepted", func(t *testing.T) {
		validTypes := []string{
			"text", "varchar", "char",
			"integer", "bigint", "smallint",
			"numeric", "decimal", "real", "double precision",
			"boolean", "bool",
			"date", "timestamp", "timestamptz", "time", "timetz",
			"uuid", "json", "jsonb",
			"bytea", "inet", "cidr", "macaddr",
			"serial", "bigserial", "smallserial",
		}

		for _, dtype := range validTypes {
			t.Run(dtype, func(t *testing.T) {
				assert.True(t, validDataTypes[dtype], "type '%s' should be valid", dtype)
			})
		}
	})

	t.Run("invalid data types are rejected", func(t *testing.T) {
		invalidTypes := []string{
			"string",
			"int",
			"datetime",
			"blob",
			"invalid",
		}

		for _, dtype := range invalidTypes {
			t.Run(dtype, func(t *testing.T) {
				assert.False(t, validDataTypes[dtype], "type '%s' should not be valid", dtype)
			})
		}
	})
}

func TestListSchemasTool(t *testing.T) {
	t.Run("tool metadata", func(t *testing.T) {
		tool := NewListSchemasTool(nil)
		assert.Equal(t, "list_schemas", tool.Name())
		assert.Contains(t, tool.Description(), "schema")
		assert.Equal(t, []string{mcp.ScopeReadTables}, tool.RequiredScopes())
	})

	t.Run("input schema has include_system", func(t *testing.T) {
		tool := NewListSchemasTool(nil)
		schema := tool.InputSchema()
		props := schema["properties"].(map[string]any)
		assert.Contains(t, props, "include_system")
	})
}

func TestCreateSchemaTool(t *testing.T) {
	t.Run("tool metadata", func(t *testing.T) {
		tool := NewCreateSchemaTool(nil)
		assert.Equal(t, "create_schema", tool.Name())
		assert.Contains(t, tool.Description(), "admin:ddl")
		assert.Equal(t, []string{mcp.ScopeAdminDDL}, tool.RequiredScopes())
	})

	t.Run("requires name parameter", func(t *testing.T) {
		tool := NewCreateSchemaTool(nil)
		schema := tool.InputSchema()
		required := schema["required"].([]string)
		assert.Contains(t, required, "name")
	})
}

func TestCreateTableTool(t *testing.T) {
	t.Run("tool metadata", func(t *testing.T) {
		tool := NewCreateTableTool(nil)
		assert.Equal(t, "create_table", tool.Name())
		assert.Contains(t, tool.Description(), "admin:ddl")
		assert.Equal(t, []string{mcp.ScopeAdminDDL}, tool.RequiredScopes())
	})

	t.Run("requires name and columns parameters", func(t *testing.T) {
		tool := NewCreateTableTool(nil)
		schema := tool.InputSchema()
		required := schema["required"].([]string)
		assert.Contains(t, required, "name")
		assert.Contains(t, required, "columns")
	})

	t.Run("schema defaults to public", func(t *testing.T) {
		tool := NewCreateTableTool(nil)
		schema := tool.InputSchema()
		props := schema["properties"].(map[string]any)
		schemaProp := props["schema"].(map[string]any)
		assert.Equal(t, "public", schemaProp["default"])
	})
}

func TestDropTableTool(t *testing.T) {
	t.Run("tool metadata", func(t *testing.T) {
		tool := NewDropTableTool(nil)
		assert.Equal(t, "drop_table", tool.Name())
		assert.Contains(t, tool.Description(), "caution")
		assert.Equal(t, []string{mcp.ScopeAdminDDL}, tool.RequiredScopes())
	})

	t.Run("requires table parameter", func(t *testing.T) {
		tool := NewDropTableTool(nil)
		schema := tool.InputSchema()
		required := schema["required"].([]string)
		assert.Contains(t, required, "table")
	})

	t.Run("has cascade option", func(t *testing.T) {
		tool := NewDropTableTool(nil)
		schema := tool.InputSchema()
		props := schema["properties"].(map[string]any)
		assert.Contains(t, props, "cascade")
	})
}

func TestAddColumnTool(t *testing.T) {
	t.Run("tool metadata", func(t *testing.T) {
		tool := NewAddColumnTool(nil)
		assert.Equal(t, "add_column", tool.Name())
		assert.Contains(t, tool.Description(), "admin:ddl")
		assert.Equal(t, []string{mcp.ScopeAdminDDL}, tool.RequiredScopes())
	})

	t.Run("requires table, name, and type parameters", func(t *testing.T) {
		tool := NewAddColumnTool(nil)
		schema := tool.InputSchema()
		required := schema["required"].([]string)
		assert.Contains(t, required, "table")
		assert.Contains(t, required, "name")
		assert.Contains(t, required, "type")
	})
}

func TestDropColumnTool(t *testing.T) {
	t.Run("tool metadata", func(t *testing.T) {
		tool := NewDropColumnTool(nil)
		assert.Equal(t, "drop_column", tool.Name())
		assert.Contains(t, tool.Description(), "caution")
		assert.Equal(t, []string{mcp.ScopeAdminDDL}, tool.RequiredScopes())
	})

	t.Run("requires table and column parameters", func(t *testing.T) {
		tool := NewDropColumnTool(nil)
		schema := tool.InputSchema()
		required := schema["required"].([]string)
		assert.Contains(t, required, "table")
		assert.Contains(t, required, "column")
	})
}

func TestRenameTableTool(t *testing.T) {
	t.Run("tool metadata", func(t *testing.T) {
		tool := NewRenameTableTool(nil)
		assert.Equal(t, "rename_table", tool.Name())
		assert.Contains(t, tool.Description(), "admin:ddl")
		assert.Equal(t, []string{mcp.ScopeAdminDDL}, tool.RequiredScopes())
	})

	t.Run("requires table and new_name parameters", func(t *testing.T) {
		tool := NewRenameTableTool(nil)
		schema := tool.InputSchema()
		required := schema["required"].([]string)
		assert.Contains(t, required, "table")
		assert.Contains(t, required, "new_name")
	})
}

func TestDDLToolScopeEnforcement(t *testing.T) {
	// Test that all DDL modifying tools require admin:ddl scope
	t.Run("modifying tools require admin:ddl", func(t *testing.T) {
		modifyingTools := []struct {
			name string
			tool interface{ RequiredScopes() []string }
		}{
			{"create_schema", NewCreateSchemaTool(nil)},
			{"create_table", NewCreateTableTool(nil)},
			{"drop_table", NewDropTableTool(nil)},
			{"add_column", NewAddColumnTool(nil)},
			{"drop_column", NewDropColumnTool(nil)},
			{"rename_table", NewRenameTableTool(nil)},
		}

		for _, tc := range modifyingTools {
			t.Run(tc.name, func(t *testing.T) {
				scopes := tc.tool.RequiredScopes()
				assert.Contains(t, scopes, mcp.ScopeAdminDDL)
			})
		}
	})

	t.Run("list_schemas requires read:tables not admin:ddl", func(t *testing.T) {
		tool := NewListSchemasTool(nil)
		scopes := tool.RequiredScopes()
		assert.Contains(t, scopes, mcp.ScopeReadTables)
		assert.NotContains(t, scopes, mcp.ScopeAdminDDL)
	})
}
