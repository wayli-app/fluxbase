package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateIdentifier(t *testing.T) {
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
				err := validateIdentifier(name, "table")
				assert.NoError(t, err, "identifier '%s' should be valid", name)
			})
		}
	})

	t.Run("empty name rejected", func(t *testing.T) {
		err := validateIdentifier("", "table")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("names exceeding 63 characters rejected", func(t *testing.T) {
		longName := "a" + string(make([]byte, 63)) // 64 characters
		err := validateIdentifier(longName, "table")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot exceed 63 characters")
	})

	t.Run("name at 63 characters accepted", func(t *testing.T) {
		// Create exactly 63 character name starting with letter
		name := "a" + string(make([]byte, 62))
		for i := range name {
			if name[i] == 0 {
				name = name[:i] + "b" + name[i+1:]
			}
		}
		// Rebuild with valid chars
		name = "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz12345678901"
		assert.Len(t, name, 63)
		err := validateIdentifier(name, "table")
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
				err := validateIdentifier(name, "table")
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
			"table\ttab",
			"table\nnewline",
		}

		for _, name := range invalidNames {
			t.Run(name, func(t *testing.T) {
				err := validateIdentifier(name, "table")
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
			"from",
			"where",
			"group",
			"order",
			"limit",
			"offset",
			"join",
			"on",
		}

		for _, name := range reservedNames {
			t.Run(name, func(t *testing.T) {
				err := validateIdentifier(name, "table")
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
				err := validateIdentifier(name, "table")
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "reserved keyword")
			})
		}
	})

	t.Run("entity type appears in error message", func(t *testing.T) {
		err := validateIdentifier("", "schema")
		assert.Contains(t, err.Error(), "schema")

		err = validateIdentifier("", "column")
		assert.Contains(t, err.Error(), "column")

		err = validateIdentifier("", "table")
		assert.Contains(t, err.Error(), "table")
	})
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
		}

		for _, dtype := range validTypes {
			t.Run(dtype, func(t *testing.T) {
				assert.True(t, validDataTypes[dtype], "type '%s' should be valid", dtype)
			})
		}
	})

	t.Run("invalid data types are rejected", func(t *testing.T) {
		invalidTypes := []string{
			"string",      // Not PostgreSQL
			"int",         // Abbreviated
			"datetime",    // Not PostgreSQL
			"blob",        // Not PostgreSQL
			"invalid",     // Nonsense
			"drop table;", // SQL injection attempt
		}

		for _, dtype := range invalidTypes {
			t.Run(dtype, func(t *testing.T) {
				assert.False(t, validDataTypes[dtype], "type '%s' should not be valid", dtype)
			})
		}
	})
}

func TestEscapeLiteral(t *testing.T) {
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
		{"special chars: @#$%", "'special chars: @#$%'"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := escapeLiteral(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsValidIdentifier(t *testing.T) {
	t.Run("valid identifiers", func(t *testing.T) {
		valid := []string{
			"column_name",
			"TableName",
			"_private",
			"id",
			"user123",
			"UPPERCASE",
			"mixedCase",
			"a",
			"_",
		}

		for _, name := range valid {
			t.Run(name, func(t *testing.T) {
				assert.True(t, isValidIdentifier(name))
			})
		}
	})

	t.Run("invalid identifiers", func(t *testing.T) {
		invalid := []string{
			"1starts_with_number",
			"has space",
			"has-dash",
			"has.dot",
			"",
			"special@char",
			"semicolon;",
		}

		for _, name := range invalid {
			t.Run(name, func(t *testing.T) {
				assert.False(t, isValidIdentifier(name))
			})
		}
	})
}

func TestQuoteIdentifier(t *testing.T) {
	t.Run("valid identifiers are quoted", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{"users", `"users"`},
			{"column_name", `"column_name"`},
			{"TableName", `"TableName"`},
			{"_private", `"_private"`},
		}

		for _, tc := range testCases {
			t.Run(tc.input, func(t *testing.T) {
				result := quoteIdentifier(tc.input)
				assert.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("invalid identifiers return empty string", func(t *testing.T) {
		invalid := []string{
			"1invalid",
			"has space",
			"has-dash",
			"",
			"special;char",
		}

		for _, name := range invalid {
			t.Run(name, func(t *testing.T) {
				result := quoteIdentifier(name)
				assert.Empty(t, result)
			})
		}
	})
}

func TestReservedKeywordsMap(t *testing.T) {
	t.Run("expected keywords are reserved", func(t *testing.T) {
		expectedReserved := []string{
			"user", "table", "column", "index",
			"select", "insert", "update", "delete",
			"from", "where", "group", "order",
			"limit", "offset", "join", "on",
		}

		for _, keyword := range expectedReserved {
			t.Run(keyword, func(t *testing.T) {
				assert.True(t, reservedKeywords[keyword], "'%s' should be reserved", keyword)
			})
		}
	})

	t.Run("non-reserved words are not in map", func(t *testing.T) {
		nonReserved := []string{
			"users",
			"products",
			"orders",
			"custom_table",
			"my_column",
		}

		for _, word := range nonReserved {
			t.Run(word, func(t *testing.T) {
				assert.False(t, reservedKeywords[word], "'%s' should not be reserved", word)
			})
		}
	})
}

func TestIdentifierPatternRegex(t *testing.T) {
	t.Run("matches valid patterns", func(t *testing.T) {
		valid := []string{
			"a",
			"_",
			"abc",
			"_abc",
			"a1",
			"a_b_c",
			"UPPER",
			"MixedCase",
			"name123",
		}

		for _, s := range valid {
			t.Run(s, func(t *testing.T) {
				assert.True(t, identifierPattern.MatchString(s))
			})
		}
	})

	t.Run("does not match invalid patterns", func(t *testing.T) {
		invalid := []string{
			"1abc",     // starts with number
			"a b",      // space
			"a-b",      // hyphen
			"a.b",      // dot
			"",         // empty
			"a;b",      // semicolon
			"a'b",      // quote
			"SELECT *", // SQL
		}

		for _, s := range invalid {
			t.Run(s, func(t *testing.T) {
				assert.False(t, identifierPattern.MatchString(s))
			})
		}
	})
}

func TestNewDDLHandler(t *testing.T) {
	t.Run("creates handler with nil db", func(t *testing.T) {
		handler := NewDDLHandler(nil)
		assert.NotNil(t, handler)
		assert.Nil(t, handler.db)
	})
}

func TestSplitOrderParams(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple order",
			input:    "name.asc,created_at.desc",
			expected: []string{"name.asc", "created_at.desc"},
		},
		{
			name:     "vector order with brackets",
			input:    "embedding.vec_cos.[0.1,0.2,0.3].asc,name.desc",
			expected: []string{"embedding.vec_cos.[0.1,0.2,0.3].asc", "name.desc"},
		},
		{
			name:     "multiple vector orders",
			input:    "emb1.vec_l2.[1,2,3].asc,emb2.vec_cos.[4,5,6].desc",
			expected: []string{"emb1.vec_l2.[1,2,3].asc", "emb2.vec_cos.[4,5,6].desc"},
		},
		{
			name:     "single order",
			input:    "name.asc",
			expected: []string{"name.asc"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "with spaces",
			input:    " name.asc , created_at.desc ",
			expected: []string{"name.asc", "created_at.desc"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := splitOrderParams(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestValidIdentifierRegex(t *testing.T) {
	t.Run("validates SQL identifier patterns", func(t *testing.T) {
		// Valid patterns
		valid := []string{
			"users", "user_id", "_private", "Table123", "a", "_",
		}
		for _, id := range valid {
			assert.True(t, validIdentifierRegex.MatchString(id), "%s should be valid", id)
		}

		// Invalid patterns
		invalid := []string{
			"123abc", "user-id", "table.name", "", "has space",
		}
		for _, id := range invalid {
			assert.False(t, validIdentifierRegex.MatchString(id), "%s should be invalid", id)
		}
	})
}
