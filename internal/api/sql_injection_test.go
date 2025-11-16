package api

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wayli-app/fluxbase/internal/config"
	"github.com/wayli-app/fluxbase/internal/database"
)

// testConfig creates a test config with default API settings for testing
func testConfigForInjectionTests() *config.Config {
	return &config.Config{
		API: config.APIConfig{
			MaxPageSize:     -1, // Unlimited for tests
			MaxTotalResults: -1, // Unlimited for tests
			DefaultPageSize: -1, // No default for tests
		},
	}
}

// TestSQLInjectionPrevention tests that the query parser properly prevents SQL injection attacks
func TestSQLInjectionPrevention(t *testing.T) {
	parser := NewQueryParser(testConfigForInjectionTests())

	tests := []struct {
		name           string
		queryString    string
		expectedError  bool
		description    string
		checkCondition func(*testing.T, *QueryParams, []interface{})
	}{
		{
			name:        "Classic OR injection in filter value",
			queryString: "email=eq.admin' OR '1'='1",
			description: "Attacker tries to bypass authentication with always-true condition",
			checkCondition: func(t *testing.T, params *QueryParams, args []interface{}) {
				// The malicious payload should be treated as a literal string parameter
				require.Len(t, params.Filters, 1)
				assert.Equal(t, "email", params.Filters[0].Column)
				assert.Equal(t, OpEqual, params.Filters[0].Operator)
				// The entire malicious string should be a single parameter value
				require.Len(t, args, 1)
				assert.Equal(t, "admin' OR '1'='1", args[0])
			},
		},
		{
			name:        "UNION-based injection",
			queryString: "id=eq.1 UNION SELECT * FROM passwords--",
			description: "Attacker tries to union with another table",
			checkCondition: func(t *testing.T, params *QueryParams, args []interface{}) {
				require.Len(t, params.Filters, 1)
				require.Len(t, args, 1)
				// The entire payload is treated as the value for id
				assert.Equal(t, "1 UNION SELECT * FROM passwords--", args[0])
			},
		},
		{
			name:        "Comment-based injection",
			queryString: "email=eq.admin@example.com-- AND password=x",
			description: "Attacker tries to comment out rest of query",
			checkCondition: func(t *testing.T, params *QueryParams, args []interface{}) {
				require.Len(t, params.Filters, 1)
				require.Len(t, args, 1)
				// The comment is part of the parameter value
				assert.Equal(t, "admin@example.com-- AND password=x", args[0])
			},
		},
		{
			name:        "Stacked query injection",
			queryString: "id=eq.1%3B%20DROP%20TABLE%20users--", // URL-encoded semicolon
			description: "Attacker tries to execute additional SQL statement",
			checkCondition: func(t *testing.T, params *QueryParams, args []interface{}) {
				require.Len(t, params.Filters, 1)
				require.Len(t, args, 1)
				// The DROP statement is part of the parameter value
				assert.Equal(t, "1; DROP TABLE users--", args[0])
			},
		},
		{
			name:        "Boolean blind injection",
			queryString: "id=eq.1 AND 1=1",
			description: "Attacker tries boolean-based blind injection",
			checkCondition: func(t *testing.T, params *QueryParams, args []interface{}) {
				require.Len(t, params.Filters, 1)
				require.Len(t, args, 1)
				assert.Equal(t, "1 AND 1=1", args[0])
			},
		},
		{
			name:        "Time-based blind injection",
			queryString: "id=eq.1' AND SLEEP(5)--",
			description: "Attacker tries time-based blind injection",
			checkCondition: func(t *testing.T, params *QueryParams, args []interface{}) {
				require.Len(t, params.Filters, 1)
				require.Len(t, args, 1)
				assert.Equal(t, "1' AND SLEEP(5)--", args[0])
			},
		},
		{
			name:        "Multiple filter injection",
			queryString: "email=eq.test@example.com&id=eq.1' OR '1'='1",
			description: "Attacker tries injection in second filter",
			checkCondition: func(t *testing.T, params *QueryParams, args []interface{}) {
				require.Len(t, params.Filters, 2)
				require.Len(t, args, 2)
				// Both should be parameterized (order may vary due to map iteration)
				assert.Contains(t, args, "test@example.com")
				assert.Contains(t, args, "1' OR '1'='1")
			},
		},
		{
			name:        "Hex encoding injection",
			queryString: "email=eq.0x61646d696e",
			description: "Attacker tries hex-encoded injection",
			checkCondition: func(t *testing.T, params *QueryParams, args []interface{}) {
				require.Len(t, params.Filters, 1)
				require.Len(t, args, 1)
				// Hex string is treated as literal
				assert.Equal(t, "0x61646d696e", args[0])
			},
		},
		{
			name:        "NULL byte injection",
			queryString: "email=eq.admin%00' OR '1'='1",
			description: "Attacker tries NULL byte to truncate query",
			checkCondition: func(t *testing.T, params *QueryParams, args []interface{}) {
				require.Len(t, params.Filters, 1)
				require.Len(t, args, 1)
				// Should decode URL encoding and treat as parameter
				assert.Contains(t, args[0].(string), "admin")
			},
		},
		{
			name:        "Subquery injection",
			queryString: "id=eq.(SELECT password FROM admin WHERE id=1)",
			description: "Attacker tries subquery injection",
			checkCondition: func(t *testing.T, params *QueryParams, args []interface{}) {
				require.Len(t, params.Filters, 1)
				require.Len(t, args, 1)
				// Subquery is treated as literal string
				assert.Equal(t, "(SELECT password FROM admin WHERE id=1)", args[0])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse query string
			values, err := url.ParseQuery(tt.queryString)
			require.NoError(t, err, "Failed to parse test query string")

			// Parse with our parser
			params, err := parser.Parse(values)

			if tt.expectedError {
				assert.Error(t, err, "Expected error for: %s", tt.description)
				return
			}

			require.NoError(t, err, "Unexpected error for: %s", tt.description)
			require.NotNil(t, params, "Params should not be nil")

			// Build SQL query to verify parameterization
			argCounter := 1
			whereClause, args := params.buildWhereClause(&argCounter)

			t.Logf("Test: %s", tt.description)
			t.Logf("WHERE clause: %s", whereClause)
			t.Logf("Args: %v", args)

			// Verify the WHERE clause uses parameters ($1, $2, etc.)
			if len(params.Filters) > 0 {
				assert.Contains(t, whereClause, "$", "WHERE clause should use parameterized queries")
			}

			// Run custom condition check
			if tt.checkCondition != nil {
				tt.checkCondition(t, params, args)
			}

			// Ensure no dangerous SQL keywords are directly in the WHERE clause
			// (they should only be in parameterized values)
			dangerousKeywords := []string{"DROP", "DELETE", "UPDATE", "INSERT", "UNION", "EXEC", "EXECUTE"}
			for _, keyword := range dangerousKeywords {
				assert.NotContains(t, whereClause, keyword,
					"WHERE clause should not contain '%s' - it should be parameterized", keyword)
			}
		})
	}
}

// TestColumnNameValidation tests that column names are validated against schema
func TestColumnNameValidation(t *testing.T) {
	// Create a mock REST handler with test table
	testTable := database.TableInfo{
		Schema: "public",
		Name:   "users",
		Columns: []database.ColumnInfo{
			{Name: "id"},
			{Name: "email"},
			{Name: "name"},
		},
		PrimaryKey: []string{"id"},
	}

	// Create handler (db can be nil for this test)
	handler := &RESTHandler{
		db:     nil,
		parser: NewQueryParser(testConfigForInjectionTests()),
	}

	tests := []struct {
		name        string
		columnName  string
		shouldExist bool
		description string
	}{
		{
			name:        "Valid column",
			columnName:  "email",
			shouldExist: true,
			description: "Normal column name should be allowed",
		},
		{
			name:        "Invalid column with SQL injection attempt",
			columnName:  "email' OR '1'='1",
			shouldExist: false,
			description: "Column name with SQL injection should not be valid",
		},
		{
			name:        "Non-existent column",
			columnName:  "password",
			shouldExist: false,
			description: "Column not in schema should be rejected",
		},
		{
			name:        "Column with SQL comment",
			columnName:  "id-- DROP TABLE users",
			shouldExist: false,
			description: "Column name with SQL comment should be rejected",
		},
		{
			name:        "Column with semicolon",
			columnName:  "id; DROP TABLE users",
			shouldExist: false,
			description: "Column name with semicolon should be rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists := handler.columnExists(testTable, tt.columnName)
			assert.Equal(t, tt.shouldExist, exists, tt.description)
		})
	}
}

// TestOrderByInjection tests that ORDER BY clauses are safe
func TestOrderByInjection(t *testing.T) {
	parser := NewQueryParser(testConfigForInjectionTests())

	tests := []struct {
		name          string
		queryString   string
		expectedError bool
		description   string
	}{
		{
			name:          "Normal order by",
			queryString:   "order=created_at.desc",
			expectedError: false,
			description:   "Normal ORDER BY should work",
		},
		{
			name:          "Order by with injection attempt",
			queryString:   "order=id%3BDROP%20TABLE%20users--.asc", // URL-encoded semicolon with valid suffix
			expectedError: false,                                   // Parsed but would fail column validation
			description:   "Injection in ORDER BY should be treated as column name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, err := url.ParseQuery(tt.queryString)
			require.NoError(t, err)

			params, err := parser.Parse(values)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				t.Logf("ORDER BY params: %+v", params.Order)
			}
		})
	}
}

// BenchmarkQueryParsing benchmarks the query parser with injection payloads
func BenchmarkQueryParsing(b *testing.B) {
	parser := NewQueryParser(testConfigForInjectionTests())
	queryString := "email=eq.admin' OR '1'='1&id=gt.100&name=like.%test%"

	values, err := url.ParseQuery(queryString)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(values)
	}
}

// TestOWASPInjectionPayloads tests against OWASP injection payloads
func TestOWASPInjectionPayloads(t *testing.T) {
	parser := NewQueryParser(testConfigForInjectionTests())

	// OWASP common injection payloads
	payloads := []string{
		"' OR '1'='1",
		"' OR '1'='1' --",
		"' OR '1'='1' /*",
		"admin'--",
		"admin' #",
		"admin'/*",
		"' or 1=1--",
		"' or 1=1#",
		"' or 1=1/*",
		"') or '1'='1--",
		"') or ('1'='1--",
		"1' ORDER BY 1--",
		"1' UNION SELECT NULL--",
		"' WAITFOR DELAY '00:00:10'--",
		"1'; DROP TABLE users--",
	}

	for _, payload := range payloads {
		t.Run("OWASP_"+payload, func(t *testing.T) {
			queryString := "email=eq." + url.QueryEscape(payload)
			values, err := url.ParseQuery(queryString)
			require.NoError(t, err)

			params, err := parser.Parse(values)
			require.NoError(t, err, "Should parse without error")

			// Build WHERE clause
			argCounter := 1
			whereClause, args := params.buildWhereClause(&argCounter)

			// Verify it's parameterized
			assert.Contains(t, whereClause, "$1", "Should use parameterized query")
			assert.Len(t, args, 1, "Should have one argument")
			assert.Equal(t, payload, args[0], "Payload should be the parameter value")

			t.Logf("Payload: %s", payload)
			t.Logf("WHERE: %s", whereClause)
			t.Logf("Args: %v", args)
		})
	}
}
