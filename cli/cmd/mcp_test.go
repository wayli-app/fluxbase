package cmd

import (
	"testing"
)

func TestParseMCPAnnotations(t *testing.T) {
	tests := []struct {
		name             string
		code             string
		filename         string
		expectedName     string
		expectedAnnCount int
		checkAnnotations map[string]interface{}
	}{
		{
			name:             "no annotations - name from filename",
			code:             `export function handler() {}`,
			filename:         "weather_forecast.ts",
			expectedName:     "weather_forecast",
			expectedAnnCount: 0,
		},
		{
			name: "name annotation overrides filename",
			code: `// @fluxbase:name custom_name
export function handler() {}`,
			filename:         "original_name.ts",
			expectedName:     "custom_name",
			expectedAnnCount: 1,
		},
		{
			name: "namespace annotation",
			code: `// @fluxbase:namespace production
export function handler() {}`,
			filename:         "my_tool.ts",
			expectedName:     "my_tool",
			expectedAnnCount: 1,
			checkAnnotations: map[string]interface{}{
				"namespace": "production",
			},
		},
		{
			name: "multiple annotations",
			code: `// @fluxbase:name get_orders
// @fluxbase:namespace ecommerce
// @fluxbase:description Get all orders for a user
// @fluxbase:timeout 60
// @fluxbase:allow-net
export function handler() {}`,
			filename:         "orders.ts",
			expectedName:     "get_orders",
			expectedAnnCount: 5,
			checkAnnotations: map[string]interface{}{
				"namespace":   "ecommerce",
				"description": "Get all orders for a user",
				"timeout":     "60",
				"allow-net":   true,
			},
		},
		{
			name:             "hyphenated filename converts to underscore",
			code:             `export function handler() {}`,
			filename:         "get-user-data.ts",
			expectedName:     "get_user_data",
			expectedAnnCount: 0,
		},
		{
			name: "uri annotation for resources",
			code: `// @fluxbase:uri fluxbase://custom/users/{id}/profile
// @fluxbase:namespace api
export function handler() {}`,
			filename:         "user_profile.ts",
			expectedName:     "user_profile",
			expectedAnnCount: 2,
			checkAnnotations: map[string]interface{}{
				"uri":       "fluxbase://custom/users/{id}/profile",
				"namespace": "api",
			},
		},
		{
			name: "boolean annotation without value",
			code: `// @fluxbase:allow-net
// @fluxbase:allow-env
export function handler() {}`,
			filename:         "tool.ts",
			expectedName:     "tool",
			expectedAnnCount: 2,
			checkAnnotations: map[string]interface{}{
				"allow-net": true,
				"allow-env": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, annotations := parseMCPAnnotations(tt.code, tt.filename)

			if name != tt.expectedName {
				t.Errorf("name = %q, want %q", name, tt.expectedName)
			}

			if len(annotations) != tt.expectedAnnCount {
				t.Errorf("annotation count = %d, want %d", len(annotations), tt.expectedAnnCount)
			}

			for key, expected := range tt.checkAnnotations {
				actual, exists := annotations[key]
				if !exists {
					t.Errorf("annotation %q not found", key)
					continue
				}
				if actual != expected {
					t.Errorf("annotation %q = %v, want %v", key, actual, expected)
				}
			}
		})
	}
}

func TestParseMCPAnnotations_NamespaceIntegration(t *testing.T) {
	// Test that namespace annotation is correctly parsed and would override CLI flag
	code := `// @fluxbase:namespace production
// @fluxbase:description Production tool for processing orders
export function handler(args, fluxbase, fluxbaseService, utils) {
  return { content: [{ type: "text", text: "OK" }] };
}`

	name, annotations := parseMCPAnnotations(code, "process_orders.ts")

	if name != "process_orders" {
		t.Errorf("name = %q, want %q", name, "process_orders")
	}

	ns, ok := annotations["namespace"]
	if !ok {
		t.Error("namespace annotation not found")
	} else if ns != "production" {
		t.Errorf("namespace = %v, want %v", ns, "production")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10c", 10, "exactly10c"},
		{"this is a longer string", 10, "this is..."},
		{"", 5, ""},
		{"ab", 5, "ab"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}
