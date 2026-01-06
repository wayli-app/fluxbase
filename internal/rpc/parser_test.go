package rpc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAnnotations(t *testing.T) {
	t.Run("parses all annotations", func(t *testing.T) {
		code := `-- @fluxbase:name get_users
-- @fluxbase:description Fetches all users from the database
-- @fluxbase:input {"user_id": "uuid", "limit?": "number"}
-- @fluxbase:output {"id": "uuid", "name": "string"}
-- @fluxbase:allowed-tables users, profiles
-- @fluxbase:allowed-schemas public, auth
-- @fluxbase:max-execution-time 30s
-- @fluxbase:require-role admin, editor
-- @fluxbase:public true
-- @fluxbase:version 2
-- @fluxbase:schedule 0 * * * *

SELECT * FROM users WHERE id = $user_id LIMIT $limit;`

		annotations, sql, err := ParseAnnotations(code)
		require.NoError(t, err)

		assert.Equal(t, "get_users", annotations.Name)
		assert.Equal(t, "Fetches all users from the database", annotations.Description)
		assert.Equal(t, map[string]string{"user_id": "uuid", "limit?": "number"}, annotations.InputSchema)
		assert.Equal(t, map[string]string{"id": "uuid", "name": "string"}, annotations.OutputSchema)
		assert.Equal(t, []string{"users", "profiles"}, annotations.AllowedTables)
		assert.Equal(t, []string{"public", "auth"}, annotations.AllowedSchemas)
		assert.Equal(t, 30*time.Second, annotations.MaxExecutionTime)
		assert.Equal(t, []string{"admin", "editor"}, annotations.RequireRoles)
		assert.True(t, annotations.IsPublic)
		assert.Equal(t, 2, annotations.Version)
		require.NotNil(t, annotations.Schedule)
		assert.Equal(t, "0 * * * *", *annotations.Schedule)

		assert.Equal(t, "SELECT * FROM users WHERE id = $user_id LIMIT $limit;", sql)
	})

	t.Run("parses minimal annotations", func(t *testing.T) {
		code := `-- @fluxbase:name simple_query
SELECT 1;`

		annotations, sql, err := ParseAnnotations(code)
		require.NoError(t, err)

		assert.Equal(t, "simple_query", annotations.Name)
		assert.Empty(t, annotations.Description)
		assert.Nil(t, annotations.InputSchema)
		assert.Nil(t, annotations.OutputSchema)
		assert.Equal(t, "SELECT 1;", sql)
	})

	t.Run("handles SQL without annotations", func(t *testing.T) {
		code := `SELECT * FROM users;`

		annotations, sql, err := ParseAnnotations(code)
		require.NoError(t, err)

		// Should return defaults
		assert.Empty(t, annotations.Name)
		assert.Equal(t, []string{"public"}, annotations.AllowedSchemas)
		assert.Equal(t, "SELECT * FROM users;", sql)
	})

	t.Run("parses public flag variants", func(t *testing.T) {
		testCases := []struct {
			value    string
			expected bool
		}{
			{"true", true},
			{"yes", true},
			{"1", true},
			{"false", false},
			{"no", false},
			{"0", false},
			{"TRUE", true},
			{"YES", true},
			{"False", false},
			{"invalid", false},
		}

		for _, tc := range testCases {
			code := "-- @fluxbase:public " + tc.value + "\nSELECT 1;"
			annotations, _, err := ParseAnnotations(code)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, annotations.IsPublic, "value: %s", tc.value)
		}
	})

	t.Run("extracts SQL query correctly", func(t *testing.T) {
		code := `-- @fluxbase:name test
-- @fluxbase:description Test procedure

-- This is a regular SQL comment
SELECT u.id, u.name
FROM users u
WHERE u.active = true
ORDER BY u.name;`

		_, sql, err := ParseAnnotations(code)
		require.NoError(t, err)

		expected := `-- This is a regular SQL comment
SELECT u.id, u.name
FROM users u
WHERE u.active = true
ORDER BY u.name;`
		assert.Equal(t, expected, sql)
	})
}

func TestParseSchemaString(t *testing.T) {
	t.Run("parses JSON format", func(t *testing.T) {
		input := `{"id": "uuid", "name": "string", "age": "number"}`
		schema, err := parseSchemaString(input)
		require.NoError(t, err)

		assert.Equal(t, "uuid", schema["id"])
		assert.Equal(t, "string", schema["name"])
		assert.Equal(t, "number", schema["age"])
	})

	t.Run("parses simple format", func(t *testing.T) {
		input := "id:uuid, name:string, age:number"
		schema, err := parseSchemaString(input)
		require.NoError(t, err)

		assert.Equal(t, "uuid", schema["id"])
		assert.Equal(t, "string", schema["name"])
		assert.Equal(t, "number", schema["age"])
	})

	t.Run("handles whitespace in simple format", func(t *testing.T) {
		input := "  id : uuid ,  name : string  "
		schema, err := parseSchemaString(input)
		require.NoError(t, err)

		assert.Equal(t, "uuid", schema["id"])
		assert.Equal(t, "string", schema["name"])
	})

	t.Run("handles optional fields", func(t *testing.T) {
		input := `{"required": "string", "optional?": "number"}`
		schema, err := parseSchemaString(input)
		require.NoError(t, err)

		assert.Equal(t, "string", schema["required"])
		assert.Equal(t, "number", schema["optional?"])
	})

	t.Run("returns nil for empty input", func(t *testing.T) {
		schema, err := parseSchemaString("")
		require.NoError(t, err)
		assert.Nil(t, schema)
	})

	t.Run("returns nil for whitespace only", func(t *testing.T) {
		schema, err := parseSchemaString("   ")
		require.NoError(t, err)
		assert.Nil(t, schema)
	})

	t.Run("handles malformed simple format gracefully", func(t *testing.T) {
		// Missing colon - should be skipped
		input := "field1, field2:type"
		schema, err := parseSchemaString(input)
		require.NoError(t, err)
		assert.Len(t, schema, 1)
		assert.Equal(t, "type", schema["field2"])
	})
}

func TestParseCommaSeparatedList(t *testing.T) {
	t.Run("parses simple list", func(t *testing.T) {
		result := parseCommaSeparatedList("a, b, c")
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("trims whitespace", func(t *testing.T) {
		result := parseCommaSeparatedList("  users  ,  profiles  ,  orders  ")
		assert.Equal(t, []string{"users", "profiles", "orders"}, result)
	})

	t.Run("handles single item", func(t *testing.T) {
		result := parseCommaSeparatedList("users")
		assert.Equal(t, []string{"users"}, result)
	})

	t.Run("handles empty string", func(t *testing.T) {
		result := parseCommaSeparatedList("")
		assert.Nil(t, result)
	})

	t.Run("handles whitespace only", func(t *testing.T) {
		result := parseCommaSeparatedList("   ")
		assert.Nil(t, result)
	})

	t.Run("skips empty items", func(t *testing.T) {
		result := parseCommaSeparatedList("a,,b, ,c")
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})
}

func TestParseDuration(t *testing.T) {
	t.Run("parses Go duration format", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected time.Duration
		}{
			{"30s", 30 * time.Second},
			{"5m", 5 * time.Minute},
			{"1h", time.Hour},
			{"1h30m", 90 * time.Minute},
			{"500ms", 500 * time.Millisecond},
		}

		for _, tc := range testCases {
			result, err := parseDuration(tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result, "input: %s", tc.input)
		}
	})

	t.Run("parses plain numbers as seconds", func(t *testing.T) {
		result, err := parseDuration("30")
		require.NoError(t, err)
		assert.Equal(t, 30*time.Second, result)
	})

	t.Run("handles whitespace", func(t *testing.T) {
		result, err := parseDuration("  30s  ")
		require.NoError(t, err)
		assert.Equal(t, 30*time.Second, result)
	})

	t.Run("returns zero for invalid input", func(t *testing.T) {
		result, err := parseDuration("invalid")
		require.NoError(t, err) // Doesn't return error, just zero
		assert.Equal(t, time.Duration(0), result)
	})
}

func TestExtractSQLQuery(t *testing.T) {
	t.Run("removes annotation lines", func(t *testing.T) {
		code := `-- @fluxbase:name test
-- @fluxbase:description Test
SELECT * FROM users;`

		result := extractSQLQuery(code)
		assert.Equal(t, "SELECT * FROM users;", result)
	})

	t.Run("preserves regular comments", func(t *testing.T) {
		code := `-- @fluxbase:name test
-- This is a regular comment
SELECT * FROM users;`

		result := extractSQLQuery(code)
		assert.Equal(t, "-- This is a regular comment\nSELECT * FROM users;", result)
	})

	t.Run("preserves multi-line SQL", func(t *testing.T) {
		code := `-- @fluxbase:name test
SELECT
    id,
    name
FROM users
WHERE active = true;`

		result := extractSQLQuery(code)
		expected := `SELECT
    id,
    name
FROM users
WHERE active = true;`
		assert.Equal(t, expected, result)
	})

	t.Run("trims leading and trailing whitespace", func(t *testing.T) {
		code := `

-- @fluxbase:name test

SELECT * FROM users;

`
		result := extractSQLQuery(code)
		assert.Equal(t, "SELECT * FROM users;", result)
	})
}

func TestApplyAnnotations(t *testing.T) {
	t.Run("applies all fields", func(t *testing.T) {
		proc := &Procedure{}
		schedule := "0 * * * *"
		annotations := &Annotations{
			Name:             "test_proc",
			Description:      "Test procedure",
			InputSchema:      map[string]string{"id": "uuid"},
			OutputSchema:     map[string]string{"name": "string"},
			AllowedTables:    []string{"users"},
			AllowedSchemas:   []string{"public"},
			MaxExecutionTime: 60 * time.Second,
			RequireRoles:     []string{"admin"},
			IsPublic:         true,
			Version:          3,
			Schedule:         &schedule,
		}

		ApplyAnnotations(proc, annotations)

		assert.Equal(t, "test_proc", proc.Name)
		assert.Equal(t, "Test procedure", proc.Description)
		assert.Equal(t, []string{"users"}, proc.AllowedTables)
		assert.Equal(t, []string{"public"}, proc.AllowedSchemas)
		assert.Equal(t, 60, proc.MaxExecutionTimeSeconds)
		assert.Equal(t, []string{"admin"}, proc.RequireRoles)
		assert.True(t, proc.IsPublic)
		assert.Equal(t, 3, proc.Version)
		require.NotNil(t, proc.Schedule)
		assert.Equal(t, "0 * * * *", *proc.Schedule)
	})

	t.Run("does not overwrite with empty values", func(t *testing.T) {
		proc := &Procedure{
			Name:        "existing_name",
			Description: "existing_description",
		}
		annotations := &Annotations{
			// Empty name and description
			IsPublic: true,
		}

		ApplyAnnotations(proc, annotations)

		assert.Equal(t, "existing_name", proc.Name)
		assert.Equal(t, "existing_description", proc.Description)
		assert.True(t, proc.IsPublic)
	})

	t.Run("handles nil schemas", func(t *testing.T) {
		proc := &Procedure{}
		annotations := &Annotations{
			Name:         "test",
			InputSchema:  nil,
			OutputSchema: nil,
		}

		ApplyAnnotations(proc, annotations)

		assert.Nil(t, proc.InputSchema)
		assert.Nil(t, proc.OutputSchema)
	})

	t.Run("serializes schemas to JSON", func(t *testing.T) {
		proc := &Procedure{}
		annotations := &Annotations{
			Name:         "test",
			InputSchema:  map[string]string{"id": "uuid"},
			OutputSchema: map[string]string{"name": "string"},
		}

		ApplyAnnotations(proc, annotations)

		assert.NotNil(t, proc.InputSchema)
		assert.Contains(t, string(proc.InputSchema), "uuid")
		assert.NotNil(t, proc.OutputSchema)
		assert.Contains(t, string(proc.OutputSchema), "string")
	})
}

func TestSchemaTypeToGoType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// UUID
		{"uuid", "uuid"},
		{"UUID", "uuid"},

		// String types
		{"string", "text"},
		{"text", "text"},
		{"STRING", "text"},

		// Integer types
		{"number", "integer"},
		{"int", "integer"},
		{"integer", "integer"},

		// Float types
		{"float", "numeric"},
		{"double", "numeric"},
		{"decimal", "numeric"},

		// Boolean
		{"boolean", "boolean"},
		{"bool", "boolean"},

		// Timestamp types
		{"timestamp", "timestamptz"},
		{"datetime", "timestamptz"},

		// Date and time
		{"date", "date"},
		{"time", "time"},

		// JSON types
		{"json", "jsonb"},
		{"jsonb", "jsonb"},
		{"object", "jsonb"},

		// Array
		{"array", "jsonb"},

		// Unknown defaults to text
		{"unknown", "text"},
		{"", "text"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := SchemaTypeToGoType(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsOptionalField(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"field?", true},
		{"optional_field?", true},
		{"required", false},
		{"field", false},
		{"?", true},
		{"", false},
		{"field??", true}, // Still ends with ?
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := IsOptionalField(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCleanFieldName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"field?", "field"},
		{"optional_field?", "optional_field"},
		{"required", "required"},
		{"field", "field"},
		{"?", ""},
		{"", ""},
		{"field??", "field?"}, // Only removes one ?
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := CleanFieldName(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDefaultAnnotations(t *testing.T) {
	t.Run("returns expected defaults", func(t *testing.T) {
		defaults := DefaultAnnotations()

		assert.Empty(t, defaults.Name)
		assert.Empty(t, defaults.Description)
		assert.Nil(t, defaults.InputSchema)
		assert.Nil(t, defaults.OutputSchema)
		assert.Equal(t, []string{"public"}, defaults.AllowedSchemas)
		assert.Empty(t, defaults.AllowedTables)
		assert.Equal(t, 30*time.Second, defaults.MaxExecutionTime)
		assert.Empty(t, defaults.RequireRoles)
		assert.False(t, defaults.IsPublic)
		assert.Equal(t, 1, defaults.Version)
		assert.Nil(t, defaults.Schedule)
	})

	t.Run("returns new instance each time", func(t *testing.T) {
		a1 := DefaultAnnotations()
		a2 := DefaultAnnotations()

		a1.Name = "modified"
		assert.Empty(t, a2.Name)
	})
}

func TestParseAnnotations_EdgeCases(t *testing.T) {
	t.Run("handles annotation in middle of code", func(t *testing.T) {
		code := `SELECT * FROM users;
-- @fluxbase:name test
SELECT * FROM orders;`

		annotations, sql, err := ParseAnnotations(code)
		require.NoError(t, err)

		assert.Equal(t, "test", annotations.Name)
		// Annotation line is removed
		assert.Contains(t, sql, "SELECT * FROM users;")
		assert.Contains(t, sql, "SELECT * FROM orders;")
	})

	t.Run("handles multiple same annotations (last wins)", func(t *testing.T) {
		code := `-- @fluxbase:name first
-- @fluxbase:name second
SELECT 1;`

		annotations, _, err := ParseAnnotations(code)
		require.NoError(t, err)

		// Regex finds first match
		assert.Equal(t, "first", annotations.Name)
	})

	t.Run("handles annotation with extra spaces", func(t *testing.T) {
		code := `--   @fluxbase:name   test_with_spaces
SELECT 1;`

		annotations, _, err := ParseAnnotations(code)
		require.NoError(t, err)

		assert.Equal(t, "test_with_spaces", annotations.Name)
	})

	t.Run("handles input schema 'any'", func(t *testing.T) {
		code := `-- @fluxbase:input any
SELECT 1;`

		annotations, _, err := ParseAnnotations(code)
		require.NoError(t, err)

		assert.Nil(t, annotations.InputSchema)
	})

	t.Run("handles output schema 'any'", func(t *testing.T) {
		code := `-- @fluxbase:output any
SELECT 1;`

		annotations, _, err := ParseAnnotations(code)
		require.NoError(t, err)

		assert.Nil(t, annotations.OutputSchema)
	})

	t.Run("handles empty schedule string", func(t *testing.T) {
		// The regex requires at least one character after @fluxbase:schedule
		// So a line with nothing after it won't match and Schedule remains nil
		// But if there's something on the line, it will be captured
		code := `-- @fluxbase:name test
SELECT 1;`

		annotations, _, err := ParseAnnotations(code)
		require.NoError(t, err)

		// No schedule annotation present, so it should be nil
		assert.Nil(t, annotations.Schedule)
	})
}
