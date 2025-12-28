package jobs

import (
	"os"
	"strings"
	"testing"
)

func TestRoleSatisfiesRequirement(t *testing.T) {
	tests := []struct {
		name         string
		userRole     string
		requiredRole string
		expected     bool
	}{
		// service_role can access everything (highest privilege)
		{"service_role satisfies admin", "service_role", "admin", true},
		{"service_role satisfies authenticated", "service_role", "authenticated", true},
		{"service_role satisfies anon", "service_role", "anon", true},
		{"service_role satisfies custom role", "service_role", "moderator", true},

		// dashboard_admin can access everything (highest privilege)
		{"dashboard_admin satisfies admin", "dashboard_admin", "admin", true},
		{"dashboard_admin satisfies authenticated", "dashboard_admin", "authenticated", true},
		{"dashboard_admin satisfies anon", "dashboard_admin", "anon", true},
		{"dashboard_admin satisfies custom role", "dashboard_admin", "editor", true},

		// Admin can access everything except service_role/dashboard_admin level
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

// TestEmbeddedSDKEndpoint verifies that the embedded SDK uses the correct API endpoint
// for database operations. This prevents regressions where the endpoint path might
// be accidentally changed back to an incorrect value.
func TestEmbeddedSDKEndpoint(t *testing.T) {
	// Read the embedded SDK file (now in internal/runtime/)
	embeddedSDKCode, err := os.ReadFile("../runtime/embedded_sdk.js")
	if err != nil {
		t.Fatalf("Failed to read embedded_sdk.js: %v", err)
	}

	code := string(embeddedSDKCode)

	// Verify the QueryBuilder uses the correct endpoint path
	correctEndpoint := "/api/v1/tables/"
	incorrectEndpoint := "/api/v1/rest/"

	// The embedded SDK should contain the correct endpoint
	if !strings.Contains(code, correctEndpoint) {
		t.Errorf("Embedded SDK does not contain the correct endpoint %q", correctEndpoint)
	}

	// The embedded SDK should NOT contain the old incorrect endpoint
	if strings.Contains(code, incorrectEndpoint) {
		t.Errorf("Embedded SDK contains the incorrect endpoint %q. "+
			"Database operations in job handlers must use %q for proper routing.",
			incorrectEndpoint, correctEndpoint)
	}

	// Additional validation: ensure QueryBuilder uses buildTablePath which returns the correct path
	// Look for the buildTablePath method that constructs the /api/v1/tables/ path
	buildTablePathIndex := strings.Index(code, "buildTablePath()")
	if buildTablePathIndex == -1 {
		t.Fatal("Could not find buildTablePath() method in embedded SDK")
	}

	// Extract a reasonable section of code after buildTablePath() to check the path construction
	endIndex := buildTablePathIndex + 200
	if endIndex > len(code) {
		endIndex = len(code)
	}
	codeSection := code[buildTablePathIndex:endIndex]

	// Check that buildTablePath returns the correct API endpoint
	if !strings.Contains(codeSection, "/api/v1/tables/") {
		t.Error("buildTablePath() does not construct path with '/api/v1/tables/'")
	}
}
