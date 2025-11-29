package jobs

import "testing"

func TestRoleSatisfiesRequirement(t *testing.T) {
	tests := []struct {
		name         string
		userRole     string
		requiredRole string
		expected     bool
	}{
		// Admin can access everything
		{"admin satisfies admin", "admin", "admin", true},
		{"admin satisfies authenticated", "admin", "authenticated", true},
		{"admin satisfies anon", "admin", "anon", true},

		// Authenticated can access authenticated and anon
		{"authenticated satisfies authenticated", "authenticated", "authenticated", true},
		{"authenticated satisfies anon", "authenticated", "anon", true},
		{"authenticated does not satisfy admin", "authenticated", "admin", false},

		// Anon can only access anon
		{"anon satisfies anon", "anon", "anon", true},
		{"anon does not satisfy authenticated", "anon", "authenticated", false},
		{"anon does not satisfy admin", "anon", "admin", false},

		// Custom roles are treated as authenticated level
		{"custom role satisfies authenticated", "moderator", "authenticated", true},
		{"custom role satisfies anon", "editor", "anon", true},
		{"custom role does not satisfy admin", "moderator", "admin", false},

		// Custom required roles require exact match
		{"exact match for custom required role", "moderator", "moderator", true},
		{"no match for different custom role", "editor", "moderator", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := roleSatisfiesRequirement(tt.userRole, tt.requiredRole)
			if result != tt.expected {
				t.Errorf("roleSatisfiesRequirement(%q, %q) = %v, want %v",
					tt.userRole, tt.requiredRole, result, tt.expected)
			}
		})
	}
}
