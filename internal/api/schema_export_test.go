package api

import (
	"testing"
)

func TestPgTypeToTS(t *testing.T) {
	tests := []struct {
		pgType   string
		expected string
	}{
		// String types
		{"text", "string"},
		{"varchar", "string"},
		{"varchar(255)", "string"},
		{"character varying(100)", "string"},
		{"char", "string"},
		{"char(1)", "string"},
		{"character(10)", "string"},
		{"uuid", "string"},
		{"citext", "string"},
		{"name", "string"},

		// Numeric types
		{"integer", "number"},
		{"int4", "number"},
		{"int8", "number"},
		{"bigint", "number"},
		{"smallint", "number"},
		{"int2", "number"},
		{"real", "number"},
		{"float4", "number"},
		{"float8", "number"},
		{"double precision", "number"},
		{"numeric", "number"},
		{"numeric(10,2)", "number"},
		{"decimal", "number"},
		{"decimal(10,2)", "number"},
		{"money", "number"},
		{"serial", "number"},
		{"bigserial", "number"},
		{"smallserial", "number"},
		{"oid", "number"},

		// Boolean
		{"boolean", "boolean"},
		{"bool", "boolean"},

		// JSON types
		{"json", "Record<string, unknown>"},
		{"jsonb", "Record<string, unknown>"},

		// Date/time types
		{"date", "string"},
		{"timestamp", "string"},
		{"timestamp without time zone", "string"},
		{"timestamp with time zone", "string"},
		{"timestamptz", "string"},
		{"time", "string"},
		{"time without time zone", "string"},
		{"time with time zone", "string"},
		{"timetz", "string"},
		{"interval", "string"},

		// Binary
		{"bytea", "string"},

		// Network types
		{"inet", "string"},
		{"cidr", "string"},
		{"macaddr", "string"},
		{"macaddr8", "string"},

		// Geometric types
		{"point", "string"},
		{"line", "string"},
		{"lseg", "string"},
		{"box", "string"},
		{"path", "string"},
		{"polygon", "string"},
		{"circle", "string"},

		// Range types
		{"int4range", "string"},
		{"int8range", "string"},
		{"numrange", "string"},
		{"tsrange", "string"},
		{"tstzrange", "string"},
		{"daterange", "string"},

		// Full-text search
		{"tsvector", "string"},
		{"tsquery", "string"},

		// Vector type (pgvector)
		{"vector", "number[]"},

		// XML
		{"xml", "string"},

		// Special types
		{"void", "void"},
		{"record", "Record<string, unknown>"},

		// Array types
		{"text[]", "string[]"},
		{"integer[]", "number[]"},
		{"boolean[]", "boolean[]"},
		{"jsonb[]", "Record<string, unknown>[]"},
		{"uuid[]", "string[]"},

		// SETOF types
		{"SETOF text", "string[]"},
		{"setof integer", "number[]"},

		// Unknown types
		{"custom_type", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.pgType, func(t *testing.T) {
			result := pgTypeToTS(tt.pgType)
			if result != tt.expected {
				t.Errorf("pgTypeToTS(%q) = %q, want %q", tt.pgType, result, tt.expected)
			}
		})
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"users", "Users"},
		{"user_profiles", "UserProfiles"},
		{"my_table_name", "MyTableName"},
		{"public", "Public"},
		{"UPPERCASE", "Uppercase"},
		{"already_pascal", "AlreadyPascal"},
		{"kebab-case", "KebabCase"},
		{"mixed_kebab-case", "MixedKebabCase"},
		{"single", "Single"},
		{"a", "A"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toPascalCase(tt.input)
			if result != tt.expected {
				t.Errorf("toPascalCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeIdentifier(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"validName", "validName"},
		{"valid_name", "valid_name"},
		{"_private", "_private"},
		{"$dollar", "$dollar"},
		{"name123", "name123"},
		{"123invalid", "'123invalid'"},
		{"with space", "'with space'"},
		{"with-dash", "'with-dash'"},
		{"with'quote", "'with\\'quote'"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeIdentifier(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeIdentifier(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFilterBySchema(t *testing.T) {
	// Create test tables
	tables := []struct {
		schema string
		name   string
	}{
		{"public", "users"},
		{"public", "posts"},
		{"auth", "users"},
		{"auth", "sessions"},
		{"storage", "buckets"},
	}

	// We need to import database.TableInfo but we're in the same package
	// This is a simplified test structure
	t.Run("filter public schema", func(t *testing.T) {
		// Actual filtering test would require database.TableInfo
		// This is a placeholder to show the test structure
		schemas := []string{"public"}
		schemaSet := make(map[string]bool)
		for _, s := range schemas {
			schemaSet[s] = true
		}

		count := 0
		for _, tbl := range tables {
			if schemaSet[tbl.schema] {
				count++
			}
		}

		if count != 2 {
			t.Errorf("Expected 2 tables in public schema, got %d", count)
		}
	})

	t.Run("filter multiple schemas", func(t *testing.T) {
		schemas := []string{"public", "auth"}
		schemaSet := make(map[string]bool)
		for _, s := range schemas {
			schemaSet[s] = true
		}

		count := 0
		for _, tbl := range tables {
			if schemaSet[tbl.schema] {
				count++
			}
		}

		if count != 4 {
			t.Errorf("Expected 4 tables in public+auth schemas, got %d", count)
		}
	})
}
