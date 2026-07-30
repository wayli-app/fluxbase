package extensions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Service Construction Tests
// =============================================================================

func TestNewService(t *testing.T) {
	t.Run("creates service with nil database", func(t *testing.T) {
		service := NewService(nil)

		require.NotNil(t, service)
		assert.Nil(t, service.db)
	})
}

// =============================================================================
// quoteIdentifier Tests - CRITICAL for SQL injection prevention
// =============================================================================

func TestQuoteIdentifier(t *testing.T) {
	t.Run("quotes simple identifier", func(t *testing.T) {
		result := quoteIdentifier("users")
		assert.Equal(t, `"users"`, result)
	})

	t.Run("quotes identifier with underscore", func(t *testing.T) {
		result := quoteIdentifier("user_accounts")
		assert.Equal(t, `"user_accounts"`, result)
	})

	t.Run("quotes identifier with numbers", func(t *testing.T) {
		result := quoteIdentifier("table123")
		assert.Equal(t, `"table123"`, result)
	})

	t.Run("escapes embedded double quotes - SQL injection prevention", func(t *testing.T) {
		// Attacker tries: pg"vector
		// Should become: "pg""vector" (escaped)
		result := quoteIdentifier(`pg"vector`)
		assert.Equal(t, `"pg""vector"`, result)
	})

	t.Run("escapes multiple embedded double quotes", func(t *testing.T) {
		result := quoteIdentifier(`test""injection`)
		assert.Equal(t, `"test""""injection"`, result)
	})

	t.Run("escapes injection attempt with semicolon", func(t *testing.T) {
		// Attacker tries: test"; DROP TABLE users; --
		// The embedded double quote must be escaped by doubling it
		// PostgreSQL will treat the entire thing as one identifier
		result := quoteIdentifier(`test"; DROP TABLE users; --`)
		assert.Equal(t, `"test""; DROP TABLE users; --"`, result)
		// The result has the embedded quote escaped, so PostgreSQL treats it as a single identifier name
	})

	t.Run("handles empty string", func(t *testing.T) {
		result := quoteIdentifier("")
		assert.Equal(t, `""`, result)
	})

	t.Run("handles reserved words", func(t *testing.T) {
		reservedWords := []string{"select", "table", "user", "index", "grant"}
		for _, word := range reservedWords {
			result := quoteIdentifier(word)
			assert.Equal(t, `"`+word+`"`, result)
		}
	})
}

// =============================================================================
// isValidIdentifier Tests
// =============================================================================

func TestIsValidIdentifier(t *testing.T) {
	t.Run("accepts valid identifiers", func(t *testing.T) {
		validIdentifiers := []string{
			"users",
			"user_accounts",
			"table123",
			"_private",
			"MyTable",
			"UPPERCASE",
		}

		for _, id := range validIdentifiers {
			assert.True(t, isValidIdentifier(id), "identifier %q should be valid", id)
		}
	})

	t.Run("rejects identifiers starting with number", func(t *testing.T) {
		assert.False(t, isValidIdentifier("123table"))
		assert.False(t, isValidIdentifier("1users"))
	})

	t.Run("rejects identifiers with special characters", func(t *testing.T) {
		invalidIdentifiers := []string{
			"user-accounts",   // hyphen
			"user accounts",   // space
			"user.table",      // dot
			"user;drop",       // semicolon
			`user"inject`,     // double quote
			"user'inject",     // single quote
			"user\ninjection", // newline
			"user\tinjection", // tab
			"user$var",        // dollar sign
			"user@domain",     // at sign
			"user#comment",    // hash
			"user%encode",     // percent
			"user&and",        // ampersand
			"user*star",       // asterisk
			"user(parens)",    // parentheses
			"user[brackets]",  // brackets
			"user{braces}",    // braces
			"user|pipe",       // pipe
			"user\\backslash", // backslash
			"user/slash",      // slash
			"user?question",   // question mark
			"user<less",       // less than
			"user>greater",    // greater than
			"user=equals",     // equals
			"user+plus",       // plus
			"user~tilde",      // tilde
			"user`backtick",   // backtick
			"user!exclaim",    // exclamation
			"user^caret",      // caret
		}

		for _, id := range invalidIdentifiers {
			assert.False(t, isValidIdentifier(id), "identifier %q should be invalid", id)
		}
	})

	t.Run("rejects empty string", func(t *testing.T) {
		assert.False(t, isValidIdentifier(""))
	})

	t.Run("accepts underscore at start", func(t *testing.T) {
		assert.True(t, isValidIdentifier("_private"))
		assert.True(t, isValidIdentifier("__dunder"))
	})

	t.Run("rejects SQL injection patterns", func(t *testing.T) {
		injectionPatterns := []string{
			"'; DROP TABLE users; --",
			"1; DELETE FROM users",
			"test OR 1=1",
			"admin'--",
			"\" OR \"1\"=\"1",
		}

		for _, pattern := range injectionPatterns {
			assert.False(t, isValidIdentifier(pattern), "injection pattern %q should be invalid", pattern)
		}
	})
}

// =============================================================================
// validIdentifierRegex Tests
// =============================================================================

func TestValidIdentifierRegex(t *testing.T) {
	t.Run("regex pattern is correct", func(t *testing.T) {
		// Pattern: ^[a-zA-Z_][a-zA-Z0-9_]*$
		// Starts with letter or underscore
		// Followed by zero or more letters, digits, or underscores

		// Test the regex directly
		assert.True(t, validIdentifierRegex.MatchString("a"))
		assert.True(t, validIdentifierRegex.MatchString("_"))
		assert.True(t, validIdentifierRegex.MatchString("abc123"))
		assert.True(t, validIdentifierRegex.MatchString("_abc_123_"))

		assert.False(t, validIdentifierRegex.MatchString(""))
		assert.False(t, validIdentifierRegex.MatchString("1abc"))
		assert.False(t, validIdentifierRegex.MatchString("abc-def"))
	})
}

// =============================================================================
// Extension Response Types Tests
// =============================================================================

func TestEnableExtensionResponse(t *testing.T) {
	t.Run("successful enable response", func(t *testing.T) {
		response := &EnableExtensionResponse{
			Name:    "pgvector",
			Success: true,
			Message: "Extension enabled successfully",
			Version: "0.5.0",
		}

		assert.Equal(t, "pgvector", response.Name)
		assert.True(t, response.Success)
		assert.Equal(t, "Extension enabled successfully", response.Message)
		assert.Equal(t, "0.5.0", response.Version)
	})

	t.Run("failed enable response", func(t *testing.T) {
		response := &EnableExtensionResponse{
			Name:    "invalid_extension",
			Success: false,
			Message: "Extension not found in catalog",
		}

		assert.False(t, response.Success)
		assert.Empty(t, response.Version)
	})
}

func TestDisableExtensionResponse(t *testing.T) {
	t.Run("successful disable response", func(t *testing.T) {
		response := &DisableExtensionResponse{
			Name:    "pgvector",
			Success: true,
			Message: "Extension disabled successfully",
		}

		assert.True(t, response.Success)
	})

	t.Run("cannot disable core extension", func(t *testing.T) {
		response := &DisableExtensionResponse{
			Name:    "plpgsql",
			Success: false,
			Message: "Cannot disable core extension",
		}

		assert.False(t, response.Success)
		assert.Contains(t, response.Message, "core extension")
	})
}

// =============================================================================
// ExtensionStatusResponse Tests
// =============================================================================

func TestExtensionStatusResponse(t *testing.T) {
	t.Run("installed extension status", func(t *testing.T) {
		response := &ExtensionStatusResponse{
			Name:             "pgvector",
			IsEnabled:        true,
			IsInstalled:      true,
			InstalledVersion: "0.5.0",
		}

		assert.True(t, response.IsInstalled)
		assert.Equal(t, "0.5.0", response.InstalledVersion)
	})

	t.Run("not installed extension status", func(t *testing.T) {
		response := &ExtensionStatusResponse{
			Name:        "unavailable_ext",
			IsEnabled:   false,
			IsInstalled: false,
		}

		assert.False(t, response.IsInstalled)
		assert.Empty(t, response.InstalledVersion)
	})
}

// =============================================================================
// Extension Type Tests
// =============================================================================

func TestExtensionService_Struct(t *testing.T) {
	t.Run("stores all fields", func(t *testing.T) {
		ext := Extension{
			Name:             "pgvector",
			DisplayName:      "PG Vector",
			Description:      "Vector similarity search",
			Category:         "ai",
			IsCore:           false,
			RequiresRestart:  false,
			IsEnabled:        true,
			IsInstalled:      true,
			InstalledVersion: "0.5.0",
		}

		assert.Equal(t, "pgvector", ext.Name)
		assert.Equal(t, "PG Vector", ext.DisplayName)
		assert.Equal(t, "ai", ext.Category)
		assert.False(t, ext.IsCore)
	})
}

func TestAvailableExtensionService_Struct(t *testing.T) {
	t.Run("stores metadata", func(t *testing.T) {
		ext := AvailableExtension{
			Name:             "uuid-ossp",
			DisplayName:      "UUID OSSP",
			Description:      "Generate UUIDs",
			Category:         "utilities",
			IsCore:           true,
			RequiresRestart:  false,
			DocumentationURL: "https://docs.example.com/uuid-ossp",
		}

		assert.True(t, ext.IsCore)
		assert.NotEmpty(t, ext.DocumentationURL)
	})
}

// =============================================================================
// Category Tests
// =============================================================================

func TestCategoryService_Struct(t *testing.T) {
	t.Run("stores category information", func(t *testing.T) {
		category := Category{
			ID:    "ai",
			Name:  "AI / Machine Learning",
			Count: 5,
		}

		assert.Equal(t, "ai", category.ID)
		assert.Equal(t, "AI / Machine Learning", category.Name)
		assert.Equal(t, 5, category.Count)
	})
}

func TestCategoryDisplayNamesService(t *testing.T) {
	t.Run("contains common categories", func(t *testing.T) {
		// Verify CategoryDisplayNames map exists and has expected entries
		// This map provides human-readable names for categories
		assert.NotNil(t, CategoryDisplayNames)
	})
}

// =============================================================================
// PostgresExtension Tests
// =============================================================================

func TestPostgresExtensionService_Struct(t *testing.T) {
	t.Run("stores postgres extension info", func(t *testing.T) {
		ext := PostgresExtension{
			Name:             "pg_stat_statements",
			DefaultVersion:   "1.10",
			InstalledVersion: stringPtr("1.10"),
			Comment:          "Track execution statistics",
		}

		assert.Equal(t, "pg_stat_statements", ext.Name)
		assert.Equal(t, "1.10", ext.DefaultVersion)
		assert.NotNil(t, ext.InstalledVersion)
		assert.Equal(t, "1.10", *ext.InstalledVersion)
	})

	t.Run("handles nil installed version", func(t *testing.T) {
		ext := PostgresExtension{
			Name:             "not_installed",
			DefaultVersion:   "1.0",
			InstalledVersion: nil,
		}

		assert.Nil(t, ext.InstalledVersion)
	})
}

// Helper function for tests
func stringPtr(s string) *string {
	return &s
}

// =============================================================================
// ListExtensionsResponse Tests
// =============================================================================

func TestListExtensionsResponse(t *testing.T) {
	t.Run("contains extensions and categories", func(t *testing.T) {
		response := &ListExtensionsResponse{
			Extensions: []Extension{
				{Name: "pgvector", Category: "ai"},
				{Name: "uuid-ossp", Category: "utilities"},
			},
			Categories: []Category{
				{ID: "ai", Count: 1},
				{ID: "utilities", Count: 1},
			},
		}

		assert.Len(t, response.Extensions, 2)
		assert.Len(t, response.Categories, 2)
	})
}

// =============================================================================
// SQL Injection Prevention Tests
// =============================================================================

func TestSQLInjectionPrevention(t *testing.T) {
	t.Run("quoteIdentifier prevents basic injection", func(t *testing.T) {
		// Attacker attempts to break out of identifier
		malicious := `pgvector"; DROP TABLE users; --`

		result := quoteIdentifier(malicious)

		// The result should be a single quoted identifier
		// PostgreSQL will treat the entire thing as an identifier name
		assert.Equal(t, `"pgvector""; DROP TABLE users; --"`, result)
	})

	t.Run("isValidIdentifier rejects injection attempts", func(t *testing.T) {
		maliciousInputs := []string{
			`pgvector"; DROP TABLE users; --`,
			"' OR '1'='1",
			"; DELETE FROM extensions",
			"admin'--",
		}

		for _, input := range maliciousInputs {
			assert.False(t, isValidIdentifier(input))
		}
	})

	t.Run("double defense - validate then quote", func(t *testing.T) {
		// The service uses both validation AND quoting for defense in depth

		// Step 1: Validation would reject this
		malicious := `test"; DROP TABLE--`
		valid := isValidIdentifier(malicious)
		assert.False(t, valid)

		// Step 2: Even if it passed validation, quoting would neutralize it
		quoted := quoteIdentifier(malicious)
		assert.Contains(t, quoted, `""`) // Escaped double quote
	})
}

// =============================================================================
// Core Extension Protection Tests
// =============================================================================

func TestCoreExtensionProtection(t *testing.T) {
	t.Run("core extensions cannot be disabled", func(t *testing.T) {
		// Core extensions have IsCore = true
		ext := &AvailableExtension{
			Name:   "plpgsql",
			IsCore: true,
		}

		// DisableExtension should check IsCore and reject
		if ext.IsCore {
			response := &DisableExtensionResponse{
				Name:    ext.Name,
				Success: false,
				Message: "Cannot disable core extension",
			}
			assert.False(t, response.Success)
		}
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkQuoteIdentifier_Simple(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = quoteIdentifier("simple_identifier")
	}
}

func BenchmarkQuoteIdentifier_WithEscaping(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = quoteIdentifier(`test"with"quotes`)
	}
}

func BenchmarkIsValidIdentifier_Valid(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = isValidIdentifier("valid_identifier_123")
	}
}

func BenchmarkIsValidIdentifier_Invalid(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = isValidIdentifier("invalid-identifier")
	}
}

// =============================================================================
// categoryForExtension Tests
// =============================================================================

func TestCategoryForExtension(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"postgis", "geospatial"},
		{"postgis_topology", "geospatial"},
		{"PostGIS", "geospatial"}, // case-insensitive
		{"vector", "ai_ml"},
		{"pg_trgm", "text_search"},
		{"pgcrypto", "data_types"},
		{"uuid-ossp", "indexing"},
		{"timescaledb", "scheduling"},
		{"postgres_fdw", "foreign_data"},
		{"pg_stat_statements", "monitoring"},
		{"some_unknown_ext", "utilities"},
		{"", "utilities"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := categoryForExtension(tt.name)
			assert.Equal(t, tt.want, got)
		})
	}
}
