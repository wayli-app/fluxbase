package auth

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// ParseNamespaceScope Tests
// =============================================================================

func TestParseNamespaceScope(t *testing.T) {
	tests := []struct {
		name              string
		scope             string
		expectedAction    string
		expectedResource  string
		expectedNamespace string
	}{
		// Wildcard
		{
			name:              "wildcard scope",
			scope:             "*",
			expectedAction:    "*",
			expectedResource:  "*",
			expectedNamespace: "*",
		},
		// Two-part scopes (action:resource)
		{
			name:              "execute functions",
			scope:             "execute:functions",
			expectedAction:    "execute",
			expectedResource:  "functions",
			expectedNamespace: "*",
		},
		{
			name:              "read tables",
			scope:             "read:tables",
			expectedAction:    "read",
			expectedResource:  "tables",
			expectedNamespace: "*",
		},
		{
			name:              "write storage",
			scope:             "write:storage",
			expectedAction:    "write",
			expectedResource:  "storage",
			expectedNamespace: "*",
		},
		// Three-part scopes (action:resource:namespace)
		{
			name:              "execute functions in prod namespace",
			scope:             "execute:functions:prod",
			expectedAction:    "execute",
			expectedResource:  "functions",
			expectedNamespace: "prod",
		},
		{
			name:              "execute functions with prefix pattern",
			scope:             "execute:functions:prod-*",
			expectedAction:    "execute",
			expectedResource:  "functions",
			expectedNamespace: "prod-*",
		},
		{
			name:              "read rpc in specific namespace",
			scope:             "read:rpc:api-v1",
			expectedAction:    "read",
			expectedResource:  "rpc",
			expectedNamespace: "api-v1",
		},
		// Edge cases
		{
			name:              "namespace with colon",
			scope:             "execute:functions:prod:v1",
			expectedAction:    "execute",
			expectedResource:  "functions",
			expectedNamespace: "prod:v1",
		},
		{
			name:              "namespace with multiple colons",
			scope:             "read:tables:schema:public:v2",
			expectedAction:    "read",
			expectedResource:  "tables",
			expectedNamespace: "schema:public:v2",
		},
		{
			name:              "single part (malformed)",
			scope:             "execute",
			expectedAction:    "execute",
			expectedResource:  "",
			expectedNamespace: "*",
		},
		{
			name:              "empty scope",
			scope:             "",
			expectedAction:    "",
			expectedResource:  "",
			expectedNamespace: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			action, resource, namespace := ParseNamespaceScope(tc.scope)

			assert.Equal(t, tc.expectedAction, action, "action mismatch")
			assert.Equal(t, tc.expectedResource, resource, "resource mismatch")
			assert.Equal(t, tc.expectedNamespace, namespace, "namespace mismatch")
		})
	}
}

// =============================================================================
// MatchNamespacePattern Tests
// =============================================================================

func TestMatchNamespacePattern(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		pattern   string
		expected  bool
	}{
		// Wildcard matches everything
		{
			name:      "wildcard matches any namespace",
			namespace: "production",
			pattern:   "*",
			expected:  true,
		},
		{
			name:      "wildcard matches empty namespace",
			namespace: "",
			pattern:   "*",
			expected:  true,
		},
		{
			name:      "wildcard matches default",
			namespace: "default",
			pattern:   "*",
			expected:  true,
		},
		// Exact match
		{
			name:      "exact match succeeds",
			namespace: "production",
			pattern:   "production",
			expected:  true,
		},
		{
			name:      "exact match fails for different namespace",
			namespace: "production",
			pattern:   "staging",
			expected:  false,
		},
		{
			name:      "exact match is case sensitive",
			namespace: "Production",
			pattern:   "production",
			expected:  false,
		},
		// Prefix match
		{
			name:      "prefix match with star",
			namespace: "prod-us-east",
			pattern:   "prod-*",
			expected:  true,
		},
		{
			name:      "prefix match with exact prefix",
			namespace: "prod-",
			pattern:   "prod-*",
			expected:  true,
		},
		{
			name:      "prefix match fails for non-matching prefix",
			namespace: "staging-us-east",
			pattern:   "prod-*",
			expected:  false,
		},
		{
			name:      "prefix match with longer pattern",
			namespace: "api-v1-production",
			pattern:   "api-v1-*",
			expected:  true,
		},
		// Edge cases
		{
			name:      "empty namespace with exact pattern",
			namespace: "",
			pattern:   "production",
			expected:  false,
		},
		{
			name:      "empty namespace with empty pattern",
			namespace: "",
			pattern:   "",
			expected:  true,
		},
		{
			name:      "pattern is substring but not prefix",
			namespace: "my-prod-server",
			pattern:   "prod-*",
			expected:  false,
		},
		{
			name:      "namespace equals prefix exactly",
			namespace: "prod",
			pattern:   "prod-*",
			expected:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := MatchNamespacePattern(tc.namespace, tc.pattern)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// =============================================================================
// HasScopeForNamespace Tests
// =============================================================================

func TestHasScopeForNamespace(t *testing.T) {
	tests := []struct {
		name      string
		scopes    []string
		action    string
		resource  string
		namespace string
		expected  bool
	}{
		// Wildcard scope
		{
			name:      "wildcard scope grants all access",
			scopes:    []string{"*"},
			action:    "execute",
			resource:  "functions",
			namespace: "production",
			expected:  true,
		},
		// Exact matches
		{
			name:      "exact scope match",
			scopes:    []string{"execute:functions:production"},
			action:    "execute",
			resource:  "functions",
			namespace: "production",
			expected:  true,
		},
		{
			name:      "scope without namespace matches all namespaces",
			scopes:    []string{"execute:functions"},
			action:    "execute",
			resource:  "functions",
			namespace: "production",
			expected:  true,
		},
		// Pattern matches
		{
			name:      "prefix pattern matches namespace",
			scopes:    []string{"execute:functions:prod-*"},
			action:    "execute",
			resource:  "functions",
			namespace: "prod-us-east",
			expected:  true,
		},
		{
			name:      "prefix pattern does not match different namespace",
			scopes:    []string{"execute:functions:prod-*"},
			action:    "execute",
			resource:  "functions",
			namespace: "staging-us-east",
			expected:  false,
		},
		// Multiple scopes
		{
			name:      "one of multiple scopes matches",
			scopes:    []string{"read:tables:staging", "execute:functions:production"},
			action:    "execute",
			resource:  "functions",
			namespace: "production",
			expected:  true,
		},
		{
			name:      "none of multiple scopes match",
			scopes:    []string{"read:tables:staging", "execute:functions:development"},
			action:    "execute",
			resource:  "functions",
			namespace: "production",
			expected:  false,
		},
		// Action mismatch
		{
			name:      "wrong action does not match",
			scopes:    []string{"read:functions:production"},
			action:    "execute",
			resource:  "functions",
			namespace: "production",
			expected:  false,
		},
		// Resource mismatch
		{
			name:      "wrong resource does not match",
			scopes:    []string{"execute:rpc:production"},
			action:    "execute",
			resource:  "functions",
			namespace: "production",
			expected:  false,
		},
		// Wildcard action
		{
			name:      "wildcard action matches any action",
			scopes:    []string{"*:functions:production"},
			action:    "execute",
			resource:  "functions",
			namespace: "production",
			expected:  true,
		},
		// Wildcard resource
		{
			name:      "wildcard resource matches any resource",
			scopes:    []string{"execute:*:production"},
			action:    "execute",
			resource:  "functions",
			namespace: "production",
			expected:  true,
		},
		// Empty scopes
		{
			name:      "empty scopes grants no access",
			scopes:    []string{},
			action:    "execute",
			resource:  "functions",
			namespace: "production",
			expected:  false,
		},
		// Nil scopes
		{
			name:      "nil scopes grants no access",
			scopes:    nil,
			action:    "execute",
			resource:  "functions",
			namespace: "production",
			expected:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := HasScopeForNamespace(tc.scopes, tc.action, tc.resource, tc.namespace)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// =============================================================================
// ExtractAllowedNamespaces Tests
// =============================================================================

func TestExtractAllowedNamespaces(t *testing.T) {
	tests := []struct {
		name     string
		scopes   []string
		resource string
		expected []string // nil = all allowed, empty = default only
		isNil    bool
	}{
		// Wildcard scope
		{
			name:     "global wildcard allows all",
			scopes:   []string{"*"},
			resource: "functions",
			expected: nil,
			isNil:    true,
		},
		// Namespace wildcard in scope
		{
			name:     "namespace wildcard allows all",
			scopes:   []string{"execute:functions:*"},
			resource: "functions",
			expected: nil,
			isNil:    true,
		},
		{
			name:     "prefix pattern allows all",
			scopes:   []string{"execute:functions:prod-*"},
			resource: "functions",
			expected: nil,
			isNil:    true,
		},
		// Specific namespaces
		{
			name:     "single specific namespace",
			scopes:   []string{"execute:functions:production"},
			resource: "functions",
			expected: []string{"production"},
			isNil:    false,
		},
		{
			name:     "multiple specific namespaces",
			scopes:   []string{"execute:functions:production", "execute:functions:staging"},
			resource: "functions",
			expected: []string{"production", "staging"},
			isNil:    false,
		},
		// No namespace in scope = all allowed
		{
			name:     "scope without namespace allows all",
			scopes:   []string{"execute:functions"},
			resource: "functions",
			expected: nil,
			isNil:    true,
		},
		// Resource mismatch
		{
			name:     "scopes for different resource returns empty",
			scopes:   []string{"execute:rpc:production"},
			resource: "functions",
			expected: []string{},
			isNil:    false,
		},
		// Empty scopes (backward compatibility)
		{
			name:     "empty scopes allows all for backward compatibility",
			scopes:   []string{},
			resource: "functions",
			expected: nil,
			isNil:    true,
		},
		// Nil scopes (backward compatibility)
		{
			name:     "nil scopes allows all for backward compatibility",
			scopes:   nil,
			resource: "functions",
			expected: nil,
			isNil:    true,
		},
		// Mixed scopes
		{
			name:     "mixed specific and wildcard returns all",
			scopes:   []string{"execute:functions:production", "execute:functions:*"},
			resource: "functions",
			expected: nil,
			isNil:    true,
		},
		// Different actions
		{
			name:     "read action extracts namespaces",
			scopes:   []string{"read:functions:production"},
			resource: "functions",
			expected: []string{"production"},
			isNil:    false,
		},
		{
			name:     "write action extracts namespaces",
			scopes:   []string{"write:functions:staging"},
			resource: "functions",
			expected: []string{"staging"},
			isNil:    false,
		},
		// Wildcard action
		{
			name:     "wildcard action extracts namespaces",
			scopes:   []string{"*:functions:production"},
			resource: "functions",
			expected: []string{"production"},
			isNil:    false,
		},
		// Wildcard resource
		{
			name:     "wildcard resource extracts namespaces",
			scopes:   []string{"execute:*:production"},
			resource: "functions",
			expected: []string{"production"},
			isNil:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ExtractAllowedNamespaces(tc.scopes, tc.resource)

			if tc.isNil {
				assert.Nil(t, result)
			} else {
				// Sort both for comparison since map iteration is non-deterministic
				if result != nil {
					sort.Strings(result)
				}
				expected := tc.expected
				if expected != nil {
					sort.Strings(expected)
				}
				assert.Equal(t, expected, result)
			}
		})
	}
}

// =============================================================================
// IsNamespaceAllowed Tests
// =============================================================================

func TestIsNamespaceAllowed(t *testing.T) {
	tests := []struct {
		name              string
		namespace         string
		allowedNamespaces []string
		expected          bool
	}{
		// Nil = all allowed
		{
			name:              "nil allows any namespace",
			namespace:         "production",
			allowedNamespaces: nil,
			expected:          true,
		},
		{
			name:              "nil allows default namespace",
			namespace:         "default",
			allowedNamespaces: nil,
			expected:          true,
		},
		// Empty = default only
		{
			name:              "empty allows default namespace",
			namespace:         "default",
			allowedNamespaces: []string{},
			expected:          true,
		},
		{
			name:              "empty does not allow other namespaces",
			namespace:         "production",
			allowedNamespaces: []string{},
			expected:          false,
		},
		// Specific list
		{
			name:              "specific list allows matching namespace",
			namespace:         "production",
			allowedNamespaces: []string{"production", "staging"},
			expected:          true,
		},
		{
			name:              "specific list does not allow non-matching namespace",
			namespace:         "development",
			allowedNamespaces: []string{"production", "staging"},
			expected:          false,
		},
		// Pattern in allowed list
		{
			name:              "pattern in allowed list matches namespace",
			namespace:         "prod-us-east",
			allowedNamespaces: []string{"prod-*"},
			expected:          true,
		},
		{
			name:              "pattern in allowed list does not match different namespace",
			namespace:         "staging-us-east",
			allowedNamespaces: []string{"prod-*"},
			expected:          false,
		},
		// Mixed exact and patterns
		{
			name:              "mixed exact and pattern - exact match",
			namespace:         "staging",
			allowedNamespaces: []string{"prod-*", "staging"},
			expected:          true,
		},
		{
			name:              "mixed exact and pattern - pattern match",
			namespace:         "prod-eu-west",
			allowedNamespaces: []string{"prod-*", "staging"},
			expected:          true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsNamespaceAllowed(tc.namespace, tc.allowedNamespaces)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// =============================================================================
// Security-Focused Tests
// =============================================================================

func TestNamespaceScopes_Security(t *testing.T) {
	t.Run("prevents ReDoS with complex patterns", func(t *testing.T) {
		// The implementation only supports prefix wildcards, not regex
		// This test verifies that complex patterns are not interpreted as regex
		result := MatchNamespacePattern("aaaaaaaaaaaaa", "a{10,}")
		assert.False(t, result) // Pattern is treated literally, not as regex
	})

	t.Run("prevents access without explicit scope", func(t *testing.T) {
		scopes := []string{"read:tables:production"}

		// Should not have access to functions
		assert.False(t, HasScopeForNamespace(scopes, "execute", "functions", "production"))

		// Should not have write access
		assert.False(t, HasScopeForNamespace(scopes, "write", "tables", "production"))

		// Should not have access to staging
		assert.False(t, HasScopeForNamespace(scopes, "read", "tables", "staging"))
	})

	t.Run("case sensitivity prevents privilege escalation", func(t *testing.T) {
		scopes := []string{"execute:functions:production"}

		// Capital letters should not match
		assert.False(t, HasScopeForNamespace(scopes, "execute", "functions", "Production"))
		assert.False(t, HasScopeForNamespace(scopes, "execute", "functions", "PRODUCTION"))
	})

	t.Run("partial matches do not grant access", func(t *testing.T) {
		scopes := []string{"execute:functions:prod"}

		// "production" does not match "prod" (not a prefix pattern)
		assert.False(t, HasScopeForNamespace(scopes, "execute", "functions", "production"))
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkParseNamespaceScope(b *testing.B) {
	scopes := []string{
		"*",
		"execute:functions",
		"execute:functions:production",
		"execute:functions:prod-us-east-*",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, scope := range scopes {
			ParseNamespaceScope(scope)
		}
	}
}

func BenchmarkMatchNamespacePattern(b *testing.B) {
	patterns := []struct {
		namespace string
		pattern   string
	}{
		{"production", "*"},
		{"production", "production"},
		{"prod-us-east", "prod-*"},
		{"staging", "prod-*"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, p := range patterns {
			MatchNamespacePattern(p.namespace, p.pattern)
		}
	}
}

func BenchmarkHasScopeForNamespace(b *testing.B) {
	scopes := []string{
		"read:tables:production",
		"execute:functions:prod-*",
		"write:storage:staging",
		"*:rpc:*",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		HasScopeForNamespace(scopes, "execute", "functions", "prod-us-east")
	}
}

func BenchmarkExtractAllowedNamespaces(b *testing.B) {
	scopes := []string{
		"execute:functions:production",
		"execute:functions:staging",
		"execute:functions:development",
		"read:tables:*",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExtractAllowedNamespaces(scopes, "functions")
	}
}

func BenchmarkIsNamespaceAllowed(b *testing.B) {
	allowedNamespaces := []string{"production", "staging", "prod-*", "dev-*"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsNamespaceAllowed("prod-us-east", allowedNamespaces)
	}
}
