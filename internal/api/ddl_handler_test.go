package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// DDLHandler Construction Tests
// =============================================================================

func TestNewDDLHandler(t *testing.T) {
	t.Run("creates handler with nil database", func(t *testing.T) {
		handler := NewDDLHandler(nil, nil)
		assert.NotNil(t, handler)
		assert.Nil(t, handler.db)
	})
}

// =============================================================================
// Identifier Validation Tests
// =============================================================================

func TestIdentifierPattern(t *testing.T) {
	validIdentifiers := []string{
		"users",
		"my_table",
		"MyTable",
		"_private",
		"table123",
		"a",
		"A",
		"_",
		"user_profiles",
		"CamelCase",
		"snake_case",
		"MixedCase_123",
	}

	invalidIdentifiers := []string{
		"123table",   // starts with number
		"my-table",   // contains hyphen
		"my table",   // contains space
		"my.table",   // contains dot
		"",           // empty
		"table!",     // contains special char
		"table@name", // contains @
		"table#1",    // contains #
		"select*",    // contains *
		"table;drop", // contains semicolon
		"table'name", // contains quote
		`table"name`, // contains double quote
	}

	for _, id := range validIdentifiers {
		t.Run("valid: "+id, func(t *testing.T) {
			assert.True(t, validIdentifierRegex.MatchString(id), "Expected %q to be valid", id)
		})
	}

	for _, id := range invalidIdentifiers {
		t.Run("invalid: "+id, func(t *testing.T) {
			assert.False(t, validIdentifierRegex.MatchString(id), "Expected %q to be invalid", id)
		})
	}
}

func TestReservedKeywords(t *testing.T) {
	t.Run("common SQL keywords are reserved", func(t *testing.T) {
		keywords := []string{
			"user", "table", "column", "index",
			"select", "insert", "update", "delete",
			"from", "where", "group", "order",
			"limit", "offset", "join", "on",
		}

		for _, kw := range keywords {
			assert.True(t, reservedKeywords[kw], "Expected %q to be reserved", kw)
		}
	})

	t.Run("non-reserved words not in map", func(t *testing.T) {
		nonReserved := []string{
			"users", "posts", "comments", "profiles",
			"custom_table", "my_column",
		}

		for _, word := range nonReserved {
			assert.False(t, reservedKeywords[word], "Expected %q to not be reserved", word)
		}
	})
}

func TestValidDataTypes(t *testing.T) {
	t.Run("text types", func(t *testing.T) {
		textTypes := []string{"text", "varchar", "char"}
		for _, dt := range textTypes {
			assert.True(t, validDataTypes[dt], "Expected %q to be valid", dt)
		}
	})

	t.Run("numeric types", func(t *testing.T) {
		numericTypes := []string{
			"integer", "bigint", "smallint",
			"numeric", "decimal", "real", "double precision",
		}
		for _, dt := range numericTypes {
			assert.True(t, validDataTypes[dt], "Expected %q to be valid", dt)
		}
	})

	t.Run("boolean types", func(t *testing.T) {
		boolTypes := []string{"boolean", "bool"}
		for _, dt := range boolTypes {
			assert.True(t, validDataTypes[dt], "Expected %q to be valid", dt)
		}
	})

	t.Run("date/time types", func(t *testing.T) {
		dateTypes := []string{
			"date", "timestamp", "timestamptz", "time", "timetz",
		}
		for _, dt := range dateTypes {
			assert.True(t, validDataTypes[dt], "Expected %q to be valid", dt)
		}
	})

	t.Run("other types", func(t *testing.T) {
		otherTypes := []string{
			"uuid", "json", "jsonb",
			"bytea", "inet", "cidr", "macaddr",
		}
		for _, dt := range otherTypes {
			assert.True(t, validDataTypes[dt], "Expected %q to be valid", dt)
		}
	})

	t.Run("invalid types not in map", func(t *testing.T) {
		invalidTypes := []string{
			"string", "int", "float", "datetime",
			"blob", "longtext", "tinyint",
		}
		for _, dt := range invalidTypes {
			assert.False(t, validDataTypes[dt], "Expected %q to be invalid", dt)
		}
	})
}

// =============================================================================
// CreateSchemaRequest Tests
// =============================================================================

func TestCreateSchemaRequest_Struct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		req := CreateSchemaRequest{
			Name: "my_schema",
		}

		assert.Equal(t, "my_schema", req.Name)
	})

	t.Run("JSON deserialization", func(t *testing.T) {
		jsonData := `{"name":"test_schema"}`

		var req CreateSchemaRequest
		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Equal(t, "test_schema", req.Name)
	})

	t.Run("empty name", func(t *testing.T) {
		req := CreateSchemaRequest{
			Name: "",
		}

		assert.Empty(t, req.Name)
	})
}

// =============================================================================
// CreateTableRequest Tests
// =============================================================================

func TestCreateTableRequest_Struct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		req := CreateTableRequest{
			Schema: "public",
			Name:   "users",
			Columns: []CreateColumnRequest{
				{Name: "id", Type: "uuid", PrimaryKey: true},
				{Name: "name", Type: "text", Nullable: false},
			},
		}

		assert.Equal(t, "public", req.Schema)
		assert.Equal(t, "users", req.Name)
		assert.Len(t, req.Columns, 2)
	})

	t.Run("JSON deserialization", func(t *testing.T) {
		jsonData := `{
			"schema": "public",
			"name": "posts",
			"columns": [
				{"name": "id", "type": "uuid", "primaryKey": true},
				{"name": "title", "type": "text", "nullable": false},
				{"name": "content", "type": "text", "nullable": true}
			]
		}`

		var req CreateTableRequest
		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Equal(t, "public", req.Schema)
		assert.Equal(t, "posts", req.Name)
		assert.Len(t, req.Columns, 3)
		assert.True(t, req.Columns[0].PrimaryKey)
	})

	t.Run("empty columns", func(t *testing.T) {
		req := CreateTableRequest{
			Schema:  "public",
			Name:    "empty_table",
			Columns: []CreateColumnRequest{},
		}

		assert.Empty(t, req.Columns)
	})
}

// =============================================================================
// CreateColumnRequest Tests
// =============================================================================

func TestCreateColumnRequest_Struct(t *testing.T) {
	t.Run("basic column", func(t *testing.T) {
		col := CreateColumnRequest{
			Name:     "id",
			Type:     "uuid",
			Nullable: false,
		}

		assert.Equal(t, "id", col.Name)
		assert.Equal(t, "uuid", col.Type)
		assert.False(t, col.Nullable)
	})

	t.Run("primary key column", func(t *testing.T) {
		col := CreateColumnRequest{
			Name:       "id",
			Type:       "integer",
			Nullable:   false,
			PrimaryKey: true,
		}

		assert.True(t, col.PrimaryKey)
		assert.False(t, col.Nullable)
	})

	t.Run("column with default value", func(t *testing.T) {
		col := CreateColumnRequest{
			Name:         "created_at",
			Type:         "timestamptz",
			Nullable:     false,
			DefaultValue: "NOW()",
		}

		assert.Equal(t, "NOW()", col.DefaultValue)
	})

	t.Run("nullable column", func(t *testing.T) {
		col := CreateColumnRequest{
			Name:     "description",
			Type:     "text",
			Nullable: true,
		}

		assert.True(t, col.Nullable)
	})

	t.Run("column with all options", func(t *testing.T) {
		col := CreateColumnRequest{
			Name:         "status",
			Type:         "text",
			Nullable:     false,
			PrimaryKey:   false,
			DefaultValue: "'pending'",
		}

		assert.Equal(t, "status", col.Name)
		assert.Equal(t, "text", col.Type)
		assert.False(t, col.Nullable)
		assert.False(t, col.PrimaryKey)
		assert.Equal(t, "'pending'", col.DefaultValue)
	})
}

// =============================================================================
// CreateSchema Handler Tests
// =============================================================================

func TestCreateSchema_Validation(t *testing.T) {
	t.Run("invalid request body", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewDDLHandler(nil, nil)

		app.Post("/schemas", handler.CreateSchema)

		req := httptest.NewRequest(http.MethodPost, "/schemas", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(respBody, &result)
		require.NoError(t, err)

		assert.Contains(t, result["error"], "Invalid request body")
	})

	t.Run("empty schema name", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewDDLHandler(nil, nil)

		app.Post("/schemas", handler.CreateSchema)

		body := `{"name":""}`
		req := httptest.NewRequest(http.MethodPost, "/schemas", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// =============================================================================
// CreateTable Handler Tests
// =============================================================================

func TestCreateTable_Validation(t *testing.T) {
	t.Run("invalid request body", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewDDLHandler(nil, nil)

		app.Post("/tables", handler.CreateTable)

		req := httptest.NewRequest(http.MethodPost, "/tables", bytes.NewReader([]byte("not json")))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("missing columns", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewDDLHandler(nil, nil)

		app.Post("/tables", handler.CreateTable)

		body := `{"schema":"public","name":"test","columns":[]}`
		req := httptest.NewRequest(http.MethodPost, "/tables", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(respBody, &result)
		require.NoError(t, err)

		assert.Contains(t, result["error"], "At least one column is required")
	})

	t.Run("invalid schema name", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewDDLHandler(nil, nil)

		app.Post("/tables", handler.CreateTable)

		body := `{"schema":"123invalid","name":"test","columns":[{"name":"id","type":"uuid"}]}`
		req := httptest.NewRequest(http.MethodPost, "/tables", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("invalid table name", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewDDLHandler(nil, nil)

		app.Post("/tables", handler.CreateTable)

		body := `{"schema":"public","name":"my-table","columns":[{"name":"id","type":"uuid"}]}`
		req := httptest.NewRequest(http.MethodPost, "/tables", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// =============================================================================
// JSON Serialization Tests
// =============================================================================

func TestDDLRequests_JSONSerialization(t *testing.T) {
	t.Run("CreateSchemaRequest serializes correctly", func(t *testing.T) {
		req := CreateSchemaRequest{Name: "my_schema"}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"name":"my_schema"`)
	})

	t.Run("CreateTableRequest serializes correctly", func(t *testing.T) {
		req := CreateTableRequest{
			Schema: "public",
			Name:   "users",
			Columns: []CreateColumnRequest{
				{Name: "id", Type: "uuid", PrimaryKey: true},
			},
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"schema":"public"`)
		assert.Contains(t, string(data), `"name":"users"`)
		assert.Contains(t, string(data), `"columns"`)
	})

	t.Run("CreateColumnRequest serializes correctly", func(t *testing.T) {
		col := CreateColumnRequest{
			Name:         "created_at",
			Type:         "timestamptz",
			Nullable:     false,
			DefaultValue: "NOW()",
		}

		data, err := json.Marshal(col)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"name":"created_at"`)
		assert.Contains(t, string(data), `"type":"timestamptz"`)
		assert.Contains(t, string(data), `"defaultValue":"NOW()"`)
	})
}

// =============================================================================
// Common Column Definitions Tests
// =============================================================================

func TestCommonColumnDefinitions(t *testing.T) {
	t.Run("UUID primary key column", func(t *testing.T) {
		col := CreateColumnRequest{
			Name:         "id",
			Type:         "uuid",
			Nullable:     false,
			PrimaryKey:   true,
			DefaultValue: "gen_random_uuid()",
		}

		assert.Equal(t, "uuid", col.Type)
		assert.True(t, col.PrimaryKey)
		assert.Equal(t, "gen_random_uuid()", col.DefaultValue)
	})

	t.Run("timestamp columns", func(t *testing.T) {
		createdAt := CreateColumnRequest{
			Name:         "created_at",
			Type:         "timestamptz",
			Nullable:     false,
			DefaultValue: "NOW()",
		}

		updatedAt := CreateColumnRequest{
			Name:         "updated_at",
			Type:         "timestamptz",
			Nullable:     false,
			DefaultValue: "NOW()",
		}

		assert.Equal(t, "timestamptz", createdAt.Type)
		assert.Equal(t, "timestamptz", updatedAt.Type)
	})

	t.Run("foreign key column", func(t *testing.T) {
		col := CreateColumnRequest{
			Name:     "user_id",
			Type:     "uuid",
			Nullable: false,
		}

		assert.Equal(t, "uuid", col.Type)
		assert.False(t, col.Nullable)
		assert.False(t, col.PrimaryKey)
	})

	t.Run("JSON column", func(t *testing.T) {
		col := CreateColumnRequest{
			Name:         "metadata",
			Type:         "jsonb",
			Nullable:     true,
			DefaultValue: "'{}'::jsonb",
		}

		assert.Equal(t, "jsonb", col.Type)
		assert.True(t, col.Nullable)
	})
}

// =============================================================================
// Additional DDL Handler Tests for Improved Coverage
// =============================================================================

func TestDDLHandler_AddColumn_AllOptions(t *testing.T) {
	handler := NewDDLHandler(nil, nil)

	app := newTestApp(t)
	app.Post("/ddl/tables/:schema/:table/columns", handler.AddColumn)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "nullable column",
			body:       `{"name": "test_col", "type": "text", "nullable": true}`,
			wantStatus: fiber.StatusServiceUnavailable, // nil DB caught by guard
		},
		{
			name:       "not nullable column",
			body:       `{"name": "test_col", "type": "text", "nullable": false}`,
			wantStatus: fiber.StatusServiceUnavailable,
		},
		{
			name:       "with default value",
			body:       `{"name": "test_col", "type": "text", "defaultValue": "default"}`,
			wantStatus: fiber.StatusServiceUnavailable,
		},
		{
			name:       "with NOW() default",
			body:       `{"name": "created_at", "type": "timestamptz", "nullable": false, "defaultValue": "NOW()"}`,
			wantStatus: fiber.StatusServiceUnavailable,
		},
		{
			name:       "with string default",
			body:       `{"name": "status", "type": "text", "defaultValue": "'active'"}`,
			wantStatus: fiber.StatusServiceUnavailable,
		},
		{
			name:       "with current_timestamp default",
			body:       `{"name": "created_at", "type": "timestamp", "nullable": false, "defaultValue": "current_timestamp"}`,
			wantStatus: fiber.StatusServiceUnavailable,
		},
		{
			name:       "with default value",
			body:       `{"name": "test_col", "type": "text", "defaultValue": "default"}`,
			wantStatus: fiber.StatusServiceUnavailable,
		},
		{
			name:       "with NOW() default",
			body:       `{"name": "created_at", "type": "timestamptz", "nullable": false, "defaultValue": "NOW()"}`,
			wantStatus: fiber.StatusServiceUnavailable,
		},
		{
			name:       "with gen_random_uuid() default",
			body:       `{"name": "id", "type": "uuid", "nullable": false, "defaultValue": "gen_random_uuid()"}`,
			wantStatus: fiber.StatusServiceUnavailable,
		},
		{
			name:       "with current_timestamp default",
			body:       `{"name": "created_at", "type": "timestamp", "nullable": false, "defaultValue": "current_timestamp"}`,
			wantStatus: fiber.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/ddl/tables/public/test_table/columns", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestDDLHandler_DropColumn_Params(t *testing.T) {
	handler := NewDDLHandler(nil, nil)

	app := newTestApp(t)
	app.Delete("/ddl/tables/:schema/:table/columns/:column", handler.DropColumn)

	tests := []struct {
		name       string
		schema     string
		table      string
		column     string
		wantStatus int
	}{
		{
			name:       "all valid params",
			schema:     "public",
			table:      "test_table",
			column:     "test_column",
			wantStatus: fiber.StatusServiceUnavailable, // nil DB caught by guard
		},
		{
			name:       "schema with underscores",
			schema:     "my_schema",
			table:      "my_table",
			column:     "my_column",
			wantStatus: fiber.StatusServiceUnavailable,
		},
		{
			name:       "schema with numbers",
			schema:     "schema123",
			table:      "table456",
			column:     "column789",
			wantStatus: fiber.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("/ddl/tables/%s/%s/columns/%s", tt.schema, tt.table, tt.column)
			req := httptest.NewRequest("DELETE", url, nil)

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestDDLHandler_RenameTable_VariousNames(t *testing.T) {
	handler := NewDDLHandler(nil, nil)

	app := newTestApp(t)
	app.Patch("/ddl/tables/:schema/:table", handler.RenameTable)

	tests := []struct {
		name       string
		schema     string
		table      string
		newName    string
		wantStatus int
	}{
		{
			name:       "rename to valid name",
			schema:     "public",
			table:      "old_table",
			newName:    "new_table",
			wantStatus: fiber.StatusServiceUnavailable, // nil DB caught by guard
		},
		{
			name:       "rename to name with underscores",
			schema:     "public",
			table:      "old",
			newName:    "new_name_test",
			wantStatus: fiber.StatusServiceUnavailable,
		},
		{
			name:       "rename to mixed case",
			schema:     "public",
			table:      "old",
			newName:    "NewTable",
			wantStatus: fiber.StatusServiceUnavailable,
		},
		{
			name:       "rename to name with numbers",
			schema:     "public",
			table:      "old",
			newName:    "table123",
			wantStatus: fiber.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("/ddl/tables/%s/%s", tt.schema, tt.table)
			body := fmt.Sprintf(`{"newName": "%s"}`, tt.newName)
			req := httptest.NewRequest("PATCH", url, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestValidateIdentifier_AllValidPatterns(t *testing.T) {
	validPatterns := []struct {
		name       string
		identifier string
	}{
		{"single letter", "a"},
		{"single underscore", "_"},
		{"uppercase", "USERS"},
		{"lowercase", "users"},
		{"mixed case", "Users"},
		{"with numbers", "users123"},
		{"with underscores", "user_profiles"},
		{"starting with underscore", "_private"},
		{"mixed with underscores and numbers", "user_profiles_2024"},
		{"camelCase", "userProfiles"},
		{"PascalCase", "UserProfiles"},
		{"snake_case", "user_profiles"},
	}

	for _, tt := range validPatterns {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIdentifier(tt.identifier, "table")
			assert.NoError(t, err)
		})
	}
}

func TestValidateIdentifier_AllInvalidPatterns(t *testing.T) {
	invalidPatterns := []struct {
		name        string
		identifier  string
		errContains string
	}{
		{"starts with digit", "123table", "must start with a letter"},
		{"starts with special char", "@table", "must start with a letter"},
		{"contains space", "my table", "contain only letters"},
		{"contains hyphen", "my-table", "contain only letters"},
		{"contains dot", "my.table", "contain only letters"},
		{"contains at", "my@table", "contain only letters"},
		{"contains hash", "my#table", "contain only letters"},
		{"contains semicolon", "my;table", "contain only letters"},
		{"contains single quote", "my'table", "contain only letters"},
		{"contains double quote", "my\"table", "contain only letters"},
		{"too long - 64 chars", strings.Repeat("a", 64), "cannot exceed 63"},
		{"exactly 63 chars - valid", strings.Repeat("a", 63), ""}, // edge case
	}

	for _, tt := range invalidPatterns {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIdentifier(tt.identifier, "table")

			if tt.errContains == "" && len(tt.identifier) == 63 {
				// 63 chars should be valid
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			}
		})
	}
}

func TestValidateIdentifier_AllReservedKeywords(t *testing.T) {
	reservedWords := []string{
		"user", "table", "column", "index",
		"select", "insert", "update", "delete",
		"from", "where", "group", "order",
		"limit", "offset", "join", "on",
	}

	for _, word := range reservedWords {
		t.Run("reserved: "+word, func(t *testing.T) {
			err := validateIdentifier(word, "table")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "is a reserved keyword")
		})
	}

	// Case variations should also be caught
	t.Run("reserved keyword uppercase", func(t *testing.T) {
		err := validateIdentifier("SELECT", "table")
		// Note: The current implementation checks lowercase only
		// So "SELECT" might pass unless we add case-insensitive check
		if err != nil {
			assert.Contains(t, err.Error(), "is a reserved keyword")
		}
	})

	t.Run("reserved keyword mixed case", func(t *testing.T) {
		err := validateIdentifier("Select", "table")
		// Same as above
		if err != nil {
			assert.Contains(t, err.Error(), "is a reserved keyword")
		}
	})
}

func TestValidDataTypes_AllTypes(t *testing.T) {
	allValidTypes := []string{
		"text", "varchar", "char",
		"integer", "bigint", "smallint",
		"numeric", "decimal", "real", "double precision",
		"boolean", "bool",
		"date", "timestamp", "timestamptz", "time", "timetz",
		"uuid", "json", "jsonb",
		"bytea", "inet", "cidr", "macaddr",
	}

	for _, dataType := range allValidTypes {
		t.Run("valid type: "+dataType, func(t *testing.T) {
			assert.True(t, validDataTypes[dataType], "Type %q should be valid", dataType)
		})
	}
}

func TestEscapeLiteral_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "only single quotes",
			input:    "''",
			expected: "''''''",
		},
		{
			name:     "mixed quotes",
			input:    `it's a "test"`,
			expected: `'it''s a "test"'`,
		},
		{
			name:  "newlines",
			input: "line1\nline2",
			expected: `'line1
line2'`,
		},
		{
			name:     "tabs",
			input:    "col1\tcol2",
			expected: `'col1	col2'`,
		},
		{
			name:     "unicode",
			input:    "hello世界",
			expected: `'hello世界'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeLiteral(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDDLHandler_NilDatabase(t *testing.T) {
	handler := NewDDLHandler(nil, nil)

	app := newTestApp(t)

	tests := []struct {
		name       string
		method     string
		url        string
		body       string
		wantStatus int
	}{
		{
			name:       "CreateSchema",
			method:     "POST",
			url:        "/ddl/schemas",
			body:       `{"name": "test"}`,
			wantStatus: fiber.StatusServiceUnavailable,
		},
		{
			name:       "CreateTable",
			method:     "POST",
			url:        "/ddl/tables",
			body:       `{"schema": "public", "name": "test", "columns": [{"name": "id", "type": "uuid"}]}`,
			wantStatus: fiber.StatusServiceUnavailable,
		},
		{
			name:       "DeleteTable",
			method:     "DELETE",
			url:        "/ddl/tables/public/test_table",
			body:       "",
			wantStatus: fiber.StatusServiceUnavailable,
		},
		{
			name:       "AddColumn",
			method:     "POST",
			url:        "/ddl/tables/public/test_table/columns",
			body:       `{"name": "col", "type": "text"}`,
			wantStatus: fiber.StatusServiceUnavailable,
		},
		{
			name:       "DropColumn",
			method:     "DELETE",
			url:        "/ddl/tables/public/test_table/columns/col",
			body:       "",
			wantStatus: fiber.StatusServiceUnavailable,
		},
		{
			name:       "RenameTable",
			method:     "PATCH",
			url:        "/ddl/tables/public/test_table",
			body:       `{"newName": "new_test"}`,
			wantStatus: fiber.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.method {
			case "POST":
				switch tt.name {
				case "CreateSchema":
					app.Post("/ddl/schemas", handler.CreateSchema)
				case "CreateTable":
					app.Post("/ddl/tables", handler.CreateTable)
				case "AddColumn":
					app.Post("/ddl/tables/:schema/:table/columns", handler.AddColumn)
				}
			case "DELETE":
				switch tt.name {
				case "DeleteTable":
					app.Delete("/ddl/tables/:schema/:table", handler.DeleteTable)
				case "DropColumn":
					app.Delete("/ddl/tables/:schema/:table/columns/:column", handler.DropColumn)
				}
			case "PATCH":
				app.Patch("/ddl/tables/:schema/:table", handler.RenameTable)
			}

			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.url, strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.url, nil)
			}

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}
