package jobs

import (
	"testing"
)

func TestParseAnnotations_ProgressTimeout(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected int
	}{
		{
			name:     "default when no annotation",
			code:     "export function handler() {}",
			expected: 300, // Default after fix
		},
		{
			name:     "explicit 300",
			code:     "// @fluxbase:progress-timeout 300\nexport function handler() {}",
			expected: 300,
		},
		{
			name:     "explicit 600",
			code:     "// @fluxbase:progress-timeout 600\nexport function handler() {}",
			expected: 600,
		},
		{
			name:     "explicit 60",
			code:     "// @fluxbase:progress-timeout 60\nexport function handler() {}",
			expected: 60,
		},
		{
			name:     "with tab instead of space",
			code:     "// @fluxbase:progress-timeout\t120\nexport function handler() {}",
			expected: 120,
		},
		{
			name:     "with multiple spaces",
			code:     "// @fluxbase:progress-timeout   180\nexport function handler() {}",
			expected: 180,
		},
		{
			name:     "in multiline comment",
			code:     "/* @fluxbase:progress-timeout 240 */\nexport function handler() {}",
			expected: 240,
		},
		{
			name:     "annotation in middle of file",
			code:     "// Some comment\n// @fluxbase:progress-timeout 500\nexport function handler() {}",
			expected: 500,
		},
		{
			name:     "wrong format - no space",
			code:     "// @fluxbase:progress-timeout300\nexport function handler() {}",
			expected: 300, // Should fall back to default
		},
		{
			name:     "wrong format - with colon before number",
			code:     "// @fluxbase:progress-timeout:300\nexport function handler() {}",
			expected: 300, // Should fall back to default
		},
		{
			name:     "wrong format - with s suffix",
			code:     "// @fluxbase:progress-timeout 300s\nexport function handler() {}",
			expected: 300, // Should parse 300 (regex captures digits before 's')
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotations := parseAnnotations(tt.code)
			if annotations.ProgressTimeoutSeconds != tt.expected {
				t.Errorf("ProgressTimeoutSeconds = %d, want %d", annotations.ProgressTimeoutSeconds, tt.expected)
			}
		})
	}
}

func TestParseAnnotations_Timeout(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected int
	}{
		{
			name:     "default when no annotation",
			code:     "export function handler() {}",
			expected: 300,
		},
		{
			name:     "explicit 600",
			code:     "// @fluxbase:timeout 600\nexport function handler() {}",
			expected: 600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotations := parseAnnotations(tt.code)
			if annotations.TimeoutSeconds != tt.expected {
				t.Errorf("TimeoutSeconds = %d, want %d", annotations.TimeoutSeconds, tt.expected)
			}
		})
	}
}

func TestParseAnnotations_MaxRetries(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected int
	}{
		{
			name:     "default when no annotation",
			code:     "export function handler() {}",
			expected: 0,
		},
		{
			name:     "explicit 3",
			code:     "// @fluxbase:max-retries 3\nexport function handler() {}",
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotations := parseAnnotations(tt.code)
			if annotations.MaxRetries != tt.expected {
				t.Errorf("MaxRetries = %d, want %d", annotations.MaxRetries, tt.expected)
			}
		})
	}
}

func TestParseAnnotations_Permissions(t *testing.T) {
	tests := []struct {
		name       string
		code       string
		allowNet   bool
		allowEnv   bool
		allowRead  bool
		allowWrite bool
	}{
		{
			name:       "defaults",
			code:       "export function handler() {}",
			allowNet:   true,
			allowEnv:   true,
			allowRead:  false,
			allowWrite: false,
		},
		{
			name:       "allow-read true",
			code:       "// @fluxbase:allow-read true\nexport function handler() {}",
			allowNet:   true,
			allowEnv:   true,
			allowRead:  true,
			allowWrite: false,
		},
		{
			name:       "allow-net false",
			code:       "// @fluxbase:allow-net false\nexport function handler() {}",
			allowNet:   false,
			allowEnv:   true,
			allowRead:  false,
			allowWrite: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotations := parseAnnotations(tt.code)
			if annotations.AllowNet != tt.allowNet {
				t.Errorf("AllowNet = %v, want %v", annotations.AllowNet, tt.allowNet)
			}
			if annotations.AllowEnv != tt.allowEnv {
				t.Errorf("AllowEnv = %v, want %v", annotations.AllowEnv, tt.allowEnv)
			}
			if annotations.AllowRead != tt.allowRead {
				t.Errorf("AllowRead = %v, want %v", annotations.AllowRead, tt.allowRead)
			}
			if annotations.AllowWrite != tt.allowWrite {
				t.Errorf("AllowWrite = %v, want %v", annotations.AllowWrite, tt.allowWrite)
			}
		})
	}
}

func TestParseAnnotations_MultipleAnnotations(t *testing.T) {
	code := `// @fluxbase:timeout 600
// @fluxbase:progress-timeout 120
// @fluxbase:max-retries 3
// @fluxbase:memory 512
// @fluxbase:allow-read true
// @fluxbase:allow-net false

export async function handler(request: Request) {
  // job code
}`

	annotations := parseAnnotations(code)

	if annotations.TimeoutSeconds != 600 {
		t.Errorf("TimeoutSeconds = %d, want 600", annotations.TimeoutSeconds)
	}
	if annotations.ProgressTimeoutSeconds != 120 {
		t.Errorf("ProgressTimeoutSeconds = %d, want 120", annotations.ProgressTimeoutSeconds)
	}
	if annotations.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", annotations.MaxRetries)
	}
	if annotations.MemoryLimitMB != 512 {
		t.Errorf("MemoryLimitMB = %d, want 512", annotations.MemoryLimitMB)
	}
	if !annotations.AllowRead {
		t.Error("AllowRead should be true")
	}
	if annotations.AllowNet {
		t.Error("AllowNet should be false")
	}
}

func TestParseAnnotations_Schedule(t *testing.T) {
	tests := []struct {
		name           string
		code           string
		expectSchedule *string
	}{
		{
			name:           "no schedule",
			code:           "export function handler() {}",
			expectSchedule: nil,
		},
		{
			name:           "every 5 minutes",
			code:           "// @fluxbase:schedule */5 * * * *\nexport function handler() {}",
			expectSchedule: strPtr("*/5 * * * *"),
		},
		{
			name:           "daily at midnight",
			code:           "// @fluxbase:schedule 0 0 * * *\nexport function handler() {}",
			expectSchedule: strPtr("0 0 * * *"),
		},
		{
			name:           "every hour",
			code:           "// @fluxbase:schedule 0 * * * *\nexport function handler() {}",
			expectSchedule: strPtr("0 * * * *"),
		},
		{
			name:           "weekly on sunday",
			code:           "// @fluxbase:schedule 0 0 * * 0\nexport function handler() {}",
			expectSchedule: strPtr("0 0 * * 0"),
		},
		{
			name:           "every minute",
			code:           "// @fluxbase:schedule * * * * *\nexport function handler() {}",
			expectSchedule: strPtr("* * * * *"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotations := parseAnnotations(tt.code)
			if tt.expectSchedule == nil {
				if annotations.Schedule != nil {
					t.Errorf("Schedule = %v, want nil", *annotations.Schedule)
				}
			} else {
				if annotations.Schedule == nil {
					t.Errorf("Schedule = nil, want %v", *tt.expectSchedule)
				} else if *annotations.Schedule != *tt.expectSchedule {
					t.Errorf("Schedule = %v, want %v", *annotations.Schedule, *tt.expectSchedule)
				}
			}
		})
	}
}

func TestParseAnnotations_RequireRole(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		expectRoles []string
	}{
		{
			name:        "no require-role",
			code:        "export function handler() {}",
			expectRoles: nil,
		},
		{
			name:        "require admin",
			code:        "// @fluxbase:require-role admin\nexport function handler() {}",
			expectRoles: []string{"admin"},
		},
		{
			name:        "require authenticated",
			code:        "// @fluxbase:require-role authenticated\nexport function handler() {}",
			expectRoles: []string{"authenticated"},
		},
		{
			name:        "require anon",
			code:        "// @fluxbase:require-role anon\nexport function handler() {}",
			expectRoles: []string{"anon"},
		},
		{
			name:        "require multiple roles",
			code:        "// @fluxbase:require-role admin, editor, moderator\nexport function handler() {}",
			expectRoles: []string{"admin", "editor", "moderator"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotations := parseAnnotations(tt.code)
			if tt.expectRoles == nil {
				if len(annotations.RequireRoles) != 0 {
					t.Errorf("RequireRoles = %v, want nil/empty", annotations.RequireRoles)
				}
			} else {
				if len(annotations.RequireRoles) != len(tt.expectRoles) {
					t.Errorf("RequireRoles = %v, want %v", annotations.RequireRoles, tt.expectRoles)
				} else {
					for i, role := range annotations.RequireRoles {
						if role != tt.expectRoles[i] {
							t.Errorf("RequireRoles[%d] = %v, want %v", i, role, tt.expectRoles[i])
						}
					}
				}
			}
		})
	}
}

func TestParseAnnotations_Enabled(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		enabled bool
	}{
		{
			name:    "default enabled",
			code:    "export function handler() {}",
			enabled: true,
		},
		{
			name:    "explicitly disabled",
			code:    "// @fluxbase:enabled false\nexport function handler() {}",
			enabled: false,
		},
		{
			name:    "explicitly enabled (redundant but valid)",
			code:    "// @fluxbase:enabled true\nexport function handler() {}",
			enabled: true, // Should remain true (default)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotations := parseAnnotations(tt.code)
			if annotations.Enabled != tt.enabled {
				t.Errorf("Enabled = %v, want %v", annotations.Enabled, tt.enabled)
			}
		})
	}
}

func TestParseAnnotations_Memory(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected int
	}{
		{
			name:     "default memory",
			code:     "export function handler() {}",
			expected: 256, // Default
		},
		{
			name:     "explicit 128MB",
			code:     "// @fluxbase:memory 128\nexport function handler() {}",
			expected: 128,
		},
		{
			name:     "explicit 512MB",
			code:     "// @fluxbase:memory 512\nexport function handler() {}",
			expected: 512,
		},
		{
			name:     "explicit 1024MB (1GB)",
			code:     "// @fluxbase:memory 1024\nexport function handler() {}",
			expected: 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotations := parseAnnotations(tt.code)
			if annotations.MemoryLimitMB != tt.expected {
				t.Errorf("MemoryLimitMB = %d, want %d", annotations.MemoryLimitMB, tt.expected)
			}
		})
	}
}

func TestParseAnnotations_ScheduleWithParams(t *testing.T) {
	code := `// @fluxbase:schedule 0 2 * * *
// @fluxbase:schedule-params {"type": "daily", "notify": true}
export function handler() {}`

	annotations := parseAnnotations(code)

	if annotations.Schedule == nil {
		t.Fatal("Schedule should not be nil")
	}

	// Schedule should contain the combined format: cron|json
	// Note: JSON marshal may reorder keys, so we check for presence of both parts
	if annotations.Schedule == nil || !contains(*annotations.Schedule, "0 2 * * *") || !contains(*annotations.Schedule, "|") {
		t.Errorf("Schedule = %v, expected to contain cron and pipe separator", *annotations.Schedule)
	}
}

// Helper functions for tests
func strPtr(s string) *string {
	return &s
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 1; i < len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
