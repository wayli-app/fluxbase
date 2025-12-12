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
