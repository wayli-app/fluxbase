// Package annotations provides parsing of @fluxbase: annotations from code comments.
// This is used by the CLI to extract configuration before bundling, since esbuild
// strips comments during the bundling process.
package annotations

import (
	"regexp"
	"strconv"
	"strings"
)

// FunctionConfig contains parsed @fluxbase: annotations for edge functions
type FunctionConfig struct {
	Namespace            *string // @fluxbase:namespace production
	AllowUnauthenticated bool
	IsPublic             bool
	DisableExecutionLogs bool
	CorsOrigins          *string
	CorsMethods          *string
	CorsHeaders          *string
	CorsCredentials      *bool
	CorsMaxAge           *int
	RateLimitPerMinute   *int
	RateLimitPerHour     *int
	RateLimitPerDay      *int
}

// JobConfig contains parsed @fluxbase: annotations for background jobs
type JobConfig struct {
	Namespace            *string // @fluxbase:namespace production
	Schedule             *string
	TimeoutSeconds       *int
	MemoryLimitMB        *int
	MaxRetries           *int
	ProgressTimeout      *int
	Enabled              *bool
	AllowRead            *bool
	AllowWrite           *bool
	AllowNet             *bool
	AllowEnv             *bool
	RequireRoles         []string
	DisableExecutionLogs bool
}

// ParseFunctionAnnotations parses @fluxbase: annotations from function code.
// Returns a FunctionConfig with the parsed values.
func ParseFunctionAnnotations(code string) FunctionConfig {
	config := FunctionConfig{
		AllowUnauthenticated: false, // Secure by default
		IsPublic:             true,  // Public by default
		DisableExecutionLogs: false, // Logging enabled by default
	}

	// Match @fluxbase:namespace with value
	namespacePattern := regexp.MustCompile(`(?m)^\s*(?://|/\*|\*)\s*@fluxbase:namespace\s+(\S+)`)
	if matches := namespacePattern.FindStringSubmatch(code); len(matches) > 1 {
		value := strings.TrimSpace(matches[1])
		config.Namespace = &value
	}

	// Match @fluxbase:allow-unauthenticated
	// Matches: // @fluxbase:allow-unauthenticated or /* @fluxbase:allow-unauthenticated */ or * @fluxbase:allow-unauthenticated
	allowUnauthPattern := regexp.MustCompile(`(?m)^\s*(?://|/\*|\*)\s*@fluxbase:allow-unauthenticated`)
	if allowUnauthPattern.MatchString(code) {
		config.AllowUnauthenticated = true
	}

	// Match @fluxbase:public with optional true/false value
	publicPattern := regexp.MustCompile(`(?m)^\s*(?://|/\*|\*)\s*@fluxbase:public(?:\s+(true|false))?`)
	if matches := publicPattern.FindStringSubmatch(code); matches != nil {
		if len(matches) > 1 && matches[1] == "false" {
			config.IsPublic = false
		}
	}

	// Match @fluxbase:disable-execution-logs with optional boolean value
	disableLogsPattern := regexp.MustCompile(`(?m)^\s*(?://|/\*|\*)\s*@fluxbase:disable-execution-logs(?:\s+(true|false))?`)
	if matches := disableLogsPattern.FindStringSubmatch(code); matches != nil {
		// If no value specified or value is "true", disable logs
		if len(matches) <= 1 || matches[1] == "" || matches[1] == "true" {
			config.DisableExecutionLogs = true
		}
	}

	// Match @fluxbase:cors-origins with value
	corsOriginsPattern := regexp.MustCompile(`(?m)^\s*(?://|/\*|\*)\s*@fluxbase:cors-origins\s+(.+?)\s*$`)
	if matches := corsOriginsPattern.FindStringSubmatch(code); len(matches) > 1 {
		value := strings.TrimSpace(matches[1])
		config.CorsOrigins = &value
	}

	// Match @fluxbase:cors-methods with value
	corsMethodsPattern := regexp.MustCompile(`(?m)^\s*(?://|/\*|\*)\s*@fluxbase:cors-methods\s+(.+?)\s*$`)
	if matches := corsMethodsPattern.FindStringSubmatch(code); len(matches) > 1 {
		value := strings.TrimSpace(matches[1])
		config.CorsMethods = &value
	}

	// Match @fluxbase:cors-headers with value
	corsHeadersPattern := regexp.MustCompile(`(?m)^\s*(?://|/\*|\*)\s*@fluxbase:cors-headers\s+(.+?)\s*$`)
	if matches := corsHeadersPattern.FindStringSubmatch(code); len(matches) > 1 {
		value := strings.TrimSpace(matches[1])
		config.CorsHeaders = &value
	}

	// Match @fluxbase:cors-credentials with boolean value
	corsCredentialsPattern := regexp.MustCompile(`(?m)^\s*(?://|/\*|\*)\s*@fluxbase:cors-credentials\s+(true|false)\s*$`)
	if matches := corsCredentialsPattern.FindStringSubmatch(code); len(matches) > 1 {
		value := matches[1] == "true"
		config.CorsCredentials = &value
	}

	// Match @fluxbase:cors-max-age with integer value
	corsMaxAgePattern := regexp.MustCompile(`(?m)^\s*(?://|/\*|\*)\s*@fluxbase:cors-max-age\s+(\d+)\s*$`)
	if matches := corsMaxAgePattern.FindStringSubmatch(code); len(matches) > 1 {
		if value, err := strconv.Atoi(matches[1]); err == nil {
			config.CorsMaxAge = &value
		}
	}

	// Match @fluxbase:rate-limit with value and unit (e.g., 100/min, 1000/hour, 10000/day)
	rateLimitPattern := regexp.MustCompile(`(?m)^\s*(?://|/\*|\*)\s*@fluxbase:rate-limit\s+(\d+)/(min|hour|day)\s*$`)
	if matches := rateLimitPattern.FindStringSubmatch(code); len(matches) > 2 {
		if value, err := strconv.Atoi(matches[1]); err == nil {
			unit := matches[2]
			switch unit {
			case "min":
				config.RateLimitPerMinute = &value
			case "hour":
				config.RateLimitPerHour = &value
			case "day":
				config.RateLimitPerDay = &value
			}
		}
	}

	return config
}

// ParseJobAnnotations parses @fluxbase: annotations from job code.
// Returns a JobConfig with the parsed values.
func ParseJobAnnotations(code string) JobConfig {
	config := JobConfig{}

	// Parse namespace
	if match := regexp.MustCompile(`@fluxbase:namespace\s+(\S+)`).FindStringSubmatch(code); match != nil {
		namespace := strings.TrimSpace(match[1])
		config.Namespace = &namespace
	}

	// Parse schedule (cron expression)
	// Supports: @fluxbase:schedule 0 2 * * *
	// Use [^\n]+ to match until end of line, then extract the cron part
	if match := regexp.MustCompile(`@fluxbase:schedule\s+([0-9*,/\- ]+)`).FindStringSubmatch(code); match != nil {
		schedule := strings.TrimSpace(match[1])
		config.Schedule = &schedule
	}

	// Parse timeout
	if match := regexp.MustCompile(`@fluxbase:timeout\s+(\d+)`).FindStringSubmatch(code); match != nil {
		if timeout, err := strconv.Atoi(match[1]); err == nil {
			config.TimeoutSeconds = &timeout
		}
	}

	// Parse memory limit
	if match := regexp.MustCompile(`@fluxbase:memory\s+(\d+)`).FindStringSubmatch(code); match != nil {
		if memory, err := strconv.Atoi(match[1]); err == nil {
			config.MemoryLimitMB = &memory
		}
	}

	// Parse max retries
	if match := regexp.MustCompile(`@fluxbase:max-retries\s+(\d+)`).FindStringSubmatch(code); match != nil {
		if retries, err := strconv.Atoi(match[1]); err == nil {
			config.MaxRetries = &retries
		}
	}

	// Parse progress timeout
	if match := regexp.MustCompile(`@fluxbase:progress-timeout\s+(\d+)`).FindStringSubmatch(code); match != nil {
		if timeout, err := strconv.Atoi(match[1]); err == nil {
			config.ProgressTimeout = &timeout
		}
	}

	// Parse enabled
	if regexp.MustCompile(`@fluxbase:enabled\s+false`).MatchString(code) {
		enabled := false
		config.Enabled = &enabled
	}

	// Parse permissions
	if regexp.MustCompile(`@fluxbase:allow-read\s+true`).MatchString(code) {
		allowRead := true
		config.AllowRead = &allowRead
	}
	if regexp.MustCompile(`@fluxbase:allow-write\s+true`).MatchString(code) {
		allowWrite := true
		config.AllowWrite = &allowWrite
	}
	if regexp.MustCompile(`@fluxbase:allow-net\s+false`).MatchString(code) {
		allowNet := false
		config.AllowNet = &allowNet
	}
	if regexp.MustCompile(`@fluxbase:allow-env\s+false`).MatchString(code) {
		allowEnv := false
		config.AllowEnv = &allowEnv
	}

	// Parse require-role (supports comma-separated list of roles)
	if match := regexp.MustCompile(`@fluxbase:require-role\s+(.+)`).FindStringSubmatch(code); match != nil {
		rolesStr := strings.TrimSpace(match[1])
		var roles []string
		for _, role := range strings.Split(rolesStr, ",") {
			role = strings.TrimSpace(role)
			if role != "" {
				roles = append(roles, role)
			}
		}
		if len(roles) > 0 {
			config.RequireRoles = roles
		}
	}

	// Parse disable-execution-logs
	if regexp.MustCompile(`@fluxbase:disable-execution-logs(?:\s+true)?`).MatchString(code) {
		config.DisableExecutionLogs = true
	}

	return config
}

// ApplyFunctionConfig applies the parsed function configuration to a map.
// Only non-default values are added to avoid overriding server defaults.
func ApplyFunctionConfig(fn map[string]interface{}, config FunctionConfig) {
	// Namespace (if specified in annotation, it overrides CLI flag)
	if config.Namespace != nil {
		fn["namespace"] = *config.Namespace
	}

	// Only add non-default values
	if config.AllowUnauthenticated {
		fn["allow_unauthenticated"] = true
	}
	if !config.IsPublic {
		fn["is_public"] = false
	}
	if config.DisableExecutionLogs {
		fn["disable_execution_logs"] = true
	}

	// CORS settings
	if config.CorsOrigins != nil {
		fn["cors_origins"] = *config.CorsOrigins
	}
	if config.CorsMethods != nil {
		fn["cors_methods"] = *config.CorsMethods
	}
	if config.CorsHeaders != nil {
		fn["cors_headers"] = *config.CorsHeaders
	}
	if config.CorsCredentials != nil {
		fn["cors_credentials"] = *config.CorsCredentials
	}
	if config.CorsMaxAge != nil {
		fn["cors_max_age"] = *config.CorsMaxAge
	}

	// Rate limiting
	if config.RateLimitPerMinute != nil {
		fn["rate_limit_per_minute"] = *config.RateLimitPerMinute
	}
	if config.RateLimitPerHour != nil {
		fn["rate_limit_per_hour"] = *config.RateLimitPerHour
	}
	if config.RateLimitPerDay != nil {
		fn["rate_limit_per_day"] = *config.RateLimitPerDay
	}
}

// ApplyJobConfig applies the parsed job configuration to a map.
// Only non-nil values are added to avoid overriding server defaults.
func ApplyJobConfig(job map[string]interface{}, config JobConfig) {
	// Namespace (if specified in annotation, it overrides CLI flag)
	if config.Namespace != nil {
		job["namespace"] = *config.Namespace
	}
	if config.Schedule != nil {
		job["schedule"] = *config.Schedule
	}
	if config.TimeoutSeconds != nil {
		job["timeout_seconds"] = *config.TimeoutSeconds
	}
	if config.MemoryLimitMB != nil {
		job["memory_limit_mb"] = *config.MemoryLimitMB
	}
	if config.MaxRetries != nil {
		job["max_retries"] = *config.MaxRetries
	}
	if config.ProgressTimeout != nil {
		job["progress_timeout_seconds"] = *config.ProgressTimeout
	}
	if config.Enabled != nil {
		job["enabled"] = *config.Enabled
	}
	if config.AllowRead != nil {
		job["allow_read"] = *config.AllowRead
	}
	if config.AllowWrite != nil {
		job["allow_write"] = *config.AllowWrite
	}
	if config.AllowNet != nil {
		job["allow_net"] = *config.AllowNet
	}
	if config.AllowEnv != nil {
		job["allow_env"] = *config.AllowEnv
	}
	if len(config.RequireRoles) > 0 {
		job["require_roles"] = config.RequireRoles
	}
	if config.DisableExecutionLogs {
		job["disable_execution_logs"] = true
	}
}
