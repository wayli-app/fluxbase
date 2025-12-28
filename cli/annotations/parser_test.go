package annotations

import (
	"testing"
)

func TestParseFunctionAnnotations_AllowUnauthenticated(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name: "JSDoc style comment",
			code: `/**
 * OwnTracks Points Edge Function
 * @fluxbase:allow-unauthenticated
 */
export function handler() {}`,
			expected: true,
		},
		{
			name: "Single line comment",
			code: `// @fluxbase:allow-unauthenticated
export function handler() {}`,
			expected: true,
		},
		{
			name: "Block comment",
			code: `/* @fluxbase:allow-unauthenticated */
export function handler() {}`,
			expected: true,
		},
		{
			name:     "No annotation",
			code:     `export function handler() {}`,
			expected: false,
		},
		{
			name: "Annotation in string (should not match)",
			code: `const x = "@fluxbase:allow-unauthenticated"
export function handler() {}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ParseFunctionAnnotations(tt.code)
			if config.AllowUnauthenticated != tt.expected {
				t.Errorf("AllowUnauthenticated = %v, want %v", config.AllowUnauthenticated, tt.expected)
			}
		})
	}
}

func TestParseFunctionAnnotations_Public(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "Default (no annotation)",
			code:     `export function handler() {}`,
			expected: true,
		},
		{
			name:     "Public false",
			code:     `// @fluxbase:public false`,
			expected: false,
		},
		{
			name:     "Public true",
			code:     `// @fluxbase:public true`,
			expected: true,
		},
		{
			name:     "Public no value (defaults to true)",
			code:     `// @fluxbase:public`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ParseFunctionAnnotations(tt.code)
			if config.IsPublic != tt.expected {
				t.Errorf("IsPublic = %v, want %v", config.IsPublic, tt.expected)
			}
		})
	}
}

func TestParseFunctionAnnotations_CORS(t *testing.T) {
	code := `// @fluxbase:cors-origins https://example.com,https://other.com
// @fluxbase:cors-methods GET,POST
// @fluxbase:cors-headers X-Custom-Header
// @fluxbase:cors-credentials true
// @fluxbase:cors-max-age 3600
export function handler() {}`

	config := ParseFunctionAnnotations(code)

	if config.CorsOrigins == nil || *config.CorsOrigins != "https://example.com,https://other.com" {
		t.Errorf("CorsOrigins = %v, want https://example.com,https://other.com", config.CorsOrigins)
	}
	if config.CorsMethods == nil || *config.CorsMethods != "GET,POST" {
		t.Errorf("CorsMethods = %v, want GET,POST", config.CorsMethods)
	}
	if config.CorsHeaders == nil || *config.CorsHeaders != "X-Custom-Header" {
		t.Errorf("CorsHeaders = %v, want X-Custom-Header", config.CorsHeaders)
	}
	if config.CorsCredentials == nil || *config.CorsCredentials != true {
		t.Errorf("CorsCredentials = %v, want true", config.CorsCredentials)
	}
	if config.CorsMaxAge == nil || *config.CorsMaxAge != 3600 {
		t.Errorf("CorsMaxAge = %v, want 3600", config.CorsMaxAge)
	}
}

func TestParseFunctionAnnotations_RateLimit(t *testing.T) {
	tests := []struct {
		name               string
		code               string
		expectedPerMinute  *int
		expectedPerHour    *int
		expectedPerDay     *int
	}{
		{
			name:              "Per minute",
			code:              `// @fluxbase:rate-limit 100/min`,
			expectedPerMinute: intPtr(100),
		},
		{
			name:            "Per hour",
			code:            `// @fluxbase:rate-limit 1000/hour`,
			expectedPerHour: intPtr(1000),
		},
		{
			name:           "Per day",
			code:           `// @fluxbase:rate-limit 10000/day`,
			expectedPerDay: intPtr(10000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ParseFunctionAnnotations(tt.code)
			if !intPtrEqual(config.RateLimitPerMinute, tt.expectedPerMinute) {
				t.Errorf("RateLimitPerMinute = %v, want %v", config.RateLimitPerMinute, tt.expectedPerMinute)
			}
			if !intPtrEqual(config.RateLimitPerHour, tt.expectedPerHour) {
				t.Errorf("RateLimitPerHour = %v, want %v", config.RateLimitPerHour, tt.expectedPerHour)
			}
			if !intPtrEqual(config.RateLimitPerDay, tt.expectedPerDay) {
				t.Errorf("RateLimitPerDay = %v, want %v", config.RateLimitPerDay, tt.expectedPerDay)
			}
		})
	}
}

func TestParseJobAnnotations(t *testing.T) {
	code := `// @fluxbase:schedule 0 2 * * *
// @fluxbase:timeout 600
// @fluxbase:memory 512
// @fluxbase:max-retries 3
// @fluxbase:progress-timeout 120
// @fluxbase:enabled false
// @fluxbase:allow-read true
// @fluxbase:allow-write true
// @fluxbase:allow-net false
// @fluxbase:allow-env false
// @fluxbase:require-role admin
// @fluxbase:disable-execution-logs
export function handler() {}`

	config := ParseJobAnnotations(code)

	if config.Schedule == nil || *config.Schedule != "0 2 * * *" {
		t.Errorf("Schedule = %v, want '0 2 * * *'", config.Schedule)
	}
	if config.TimeoutSeconds == nil || *config.TimeoutSeconds != 600 {
		t.Errorf("TimeoutSeconds = %v, want 600", config.TimeoutSeconds)
	}
	if config.MemoryLimitMB == nil || *config.MemoryLimitMB != 512 {
		t.Errorf("MemoryLimitMB = %v, want 512", config.MemoryLimitMB)
	}
	if config.MaxRetries == nil || *config.MaxRetries != 3 {
		t.Errorf("MaxRetries = %v, want 3", config.MaxRetries)
	}
	if config.ProgressTimeout == nil || *config.ProgressTimeout != 120 {
		t.Errorf("ProgressTimeout = %v, want 120", config.ProgressTimeout)
	}
	if config.Enabled == nil || *config.Enabled != false {
		t.Errorf("Enabled = %v, want false", config.Enabled)
	}
	if config.AllowRead == nil || *config.AllowRead != true {
		t.Errorf("AllowRead = %v, want true", config.AllowRead)
	}
	if config.AllowWrite == nil || *config.AllowWrite != true {
		t.Errorf("AllowWrite = %v, want true", config.AllowWrite)
	}
	if config.AllowNet == nil || *config.AllowNet != false {
		t.Errorf("AllowNet = %v, want false", config.AllowNet)
	}
	if config.AllowEnv == nil || *config.AllowEnv != false {
		t.Errorf("AllowEnv = %v, want false", config.AllowEnv)
	}
	if config.RequireRole == nil || *config.RequireRole != "admin" {
		t.Errorf("RequireRole = %v, want 'admin'", config.RequireRole)
	}
	if !config.DisableExecutionLogs {
		t.Error("DisableExecutionLogs = false, want true")
	}
}

func TestApplyFunctionConfig(t *testing.T) {
	fn := map[string]interface{}{
		"name": "test-fn",
		"code": "export function handler() {}",
	}

	config := FunctionConfig{
		AllowUnauthenticated: true,
		IsPublic:             false,
		DisableExecutionLogs: true,
		CorsOrigins:          stringPtr("https://example.com"),
		RateLimitPerMinute:   intPtr(100),
	}

	ApplyFunctionConfig(fn, config)

	if fn["allow_unauthenticated"] != true {
		t.Error("allow_unauthenticated not set")
	}
	if fn["is_public"] != false {
		t.Error("is_public not set to false")
	}
	if fn["disable_execution_logs"] != true {
		t.Error("disable_execution_logs not set")
	}
	if fn["cors_origins"] != "https://example.com" {
		t.Error("cors_origins not set")
	}
	if fn["rate_limit_per_minute"] != 100 {
		t.Error("rate_limit_per_minute not set")
	}
}

func TestApplyJobConfig(t *testing.T) {
	job := map[string]interface{}{
		"name": "test-job",
		"code": "export function handler() {}",
	}

	timeout := 600
	config := JobConfig{
		Schedule:       stringPtr("0 2 * * *"),
		TimeoutSeconds: &timeout,
	}

	ApplyJobConfig(job, config)

	if job["schedule"] != "0 2 * * *" {
		t.Error("schedule not set")
	}
	if job["timeout_seconds"] != 600 {
		t.Error("timeout_seconds not set")
	}
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}

func intPtrEqual(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
