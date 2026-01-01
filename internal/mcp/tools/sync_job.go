package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/jobs"
	"github.com/fluxbase-eu/fluxbase/internal/mcp"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// SyncJobTool implements the sync_job MCP tool for deploying background jobs
type SyncJobTool struct {
	storage *jobs.Storage
}

// NewSyncJobTool creates a new sync_job tool
func NewSyncJobTool(storage *jobs.Storage) *SyncJobTool {
	return &SyncJobTool{
		storage: storage,
	}
}

func (t *SyncJobTool) Name() string {
	return "sync_job"
}

func (t *SyncJobTool) Description() string {
	return `Deploy or update a background job. Parses @fluxbase annotations from code comments for configuration.

Required annotation:
  @fluxbase:schedule "<cron>" - Cron expression for job scheduling (e.g., "0 */5 * * *" for every 5 minutes)

Optional annotations:
  @fluxbase:description <text> - Job description
  @fluxbase:timeout <seconds> - Execution timeout (default: 300)
  @fluxbase:memory <mb> - Memory limit in MB (default: 256)
  @fluxbase:max-retries <n> - Maximum retry attempts (default: 3)
  @fluxbase:require-role <role> - Required role: admin, authenticated, anon
  @fluxbase:allow-net - Allow network access (default: true)
  @fluxbase:allow-env - Allow environment variable access (default: true)
  @fluxbase:disable-logs - Disable execution logging

Example:
// @fluxbase:schedule "0 0 * * *"
// @fluxbase:description Daily cleanup job
// @fluxbase:max-retries 5
export default async function cleanup() {
  // Runs daily at midnight
}`
}

func (t *SyncJobTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Job name (alphanumeric, hyphens, underscores)",
			},
			"code": map[string]any{
				"type":        "string",
				"description": "TypeScript/JavaScript code with @fluxbase:schedule annotation required",
			},
			"namespace": map[string]any{
				"type":        "string",
				"description": "Namespace for isolating jobs (default: 'default')",
				"default":     "default",
			},
		},
		"required": []string{"name", "code"},
	}
}

func (t *SyncJobTool) RequiredScopes() []string {
	return []string{mcp.ScopeSyncJobs}
}

func (t *SyncJobTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	// Parse arguments
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("job name is required")
	}

	code, ok := args["code"].(string)
	if !ok || code == "" {
		return nil, fmt.Errorf("job code is required")
	}

	namespace := "default"
	if ns, ok := args["namespace"].(string); ok && ns != "" {
		namespace = ns
	}

	// Validate name format
	if !isValidFunctionName(name) {
		return nil, fmt.Errorf("invalid job name: must be alphanumeric with hyphens/underscores, 1-63 characters")
	}

	// Check namespace access
	if !authCtx.HasNamespaceAccess(namespace) {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Access denied to namespace: %s", namespace))},
			IsError: true,
		}, nil
	}

	// Parse annotations from code
	config := parseJobAnnotations(code)

	// Schedule is required for jobs
	if config.Schedule == "" {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent("@fluxbase:schedule annotation is required for jobs. Example: // @fluxbase:schedule \"0 */5 * * *\"")},
			IsError: true,
		}, nil
	}

	// Validate cron expression (basic validation)
	if !isValidCronExpression(config.Schedule) {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Invalid cron expression: %s", config.Schedule))},
			IsError: true,
		}, nil
	}

	log.Debug().
		Str("name", name).
		Str("namespace", namespace).
		Str("schedule", config.Schedule).
		Interface("config", config).
		Msg("MCP: sync_job - parsed annotations")

	// Create job function struct
	fn := &jobs.JobFunction{
		ID:                     uuid.New(),
		Name:                   name,
		Namespace:              namespace,
		Code:                   &code,
		OriginalCode:           &code,
		IsBundled:              false,
		Enabled:                true,
		Schedule:               &config.Schedule,
		Source:                 "mcp",
		TimeoutSeconds:         config.Timeout,
		MemoryLimitMB:          config.Memory,
		MaxRetries:             config.MaxRetries,
		ProgressTimeoutSeconds: 60, // Default progress timeout
		AllowNet:               config.AllowNet,
		AllowEnv:               config.AllowEnv,
		AllowRead:              true,
		AllowWrite:             false,
		DisableExecutionLogs:   config.DisableLogs,
	}

	// Set optional fields
	if config.Description != "" {
		fn.Description = &config.Description
	}
	if config.RequireRole != "" {
		fn.RequireRole = &config.RequireRole
	}

	// Use upsert to create or update
	err := t.storage.UpsertJobFunction(ctx, fn)
	if err != nil {
		// Check if this was an update (conflict on name/namespace)
		if strings.Contains(err.Error(), "no rows") {
			return &mcp.ToolResult{
				Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to sync job: %v", err))},
				IsError: true,
			}, nil
		}
		// Try to determine if it was create or update based on version
		if fn.Version == 1 {
			log.Info().
				Str("name", name).
				Str("namespace", namespace).
				Str("id", fn.ID.String()).
				Msg("MCP: sync_job - created new job")
		} else {
			log.Info().
				Str("name", name).
				Str("namespace", namespace).
				Str("id", fn.ID.String()).
				Int("version", fn.Version).
				Msg("MCP: sync_job - updated existing job")
		}
	}

	// Check for pgx.ErrNoRows which indicates failure
	if err == pgx.ErrNoRows {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent("Failed to sync job: database error")},
			IsError: true,
		}, nil
	}

	action := "created"
	if fn.Version > 1 {
		action = "updated"
	}

	result := map[string]any{
		"action":    action,
		"id":        fn.ID.String(),
		"name":      fn.Name,
		"namespace": fn.Namespace,
		"version":   fn.Version,
		"schedule":  config.Schedule,
	}

	// Serialize result
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to serialize result: %v", err))},
			IsError: true,
		}, nil
	}

	return &mcp.ToolResult{
		Content: []mcp.Content{mcp.TextContent(string(resultJSON))},
	}, nil
}

// JobConfig holds parsed @fluxbase annotations for jobs
type JobConfig struct {
	Description string
	Schedule    string
	Timeout     int
	Memory      int
	MaxRetries  int
	RequireRole string
	AllowNet    bool
	AllowEnv    bool
	DisableLogs bool
}

// parseJobAnnotations extracts configuration from @fluxbase: comments in job code
func parseJobAnnotations(code string) JobConfig {
	config := JobConfig{
		Timeout:    300, // Default timeout for jobs (5 minutes)
		Memory:     256, // Default memory limit for jobs
		MaxRetries: 3,   // Default retries
		AllowNet:   true,
		AllowEnv:   true,
	}

	// Match @fluxbase:annotation patterns in comments
	// Process line by line to avoid multiline regex matching issues
	lineAnnotationPattern := regexp.MustCompile(`^//\s*@fluxbase:(\S+)(?:\s+(.*))?$`)
	blockAnnotationPattern := regexp.MustCompile(`^\s*\*\s*@fluxbase:(\S+)(?:\s+(.*))?$`)

	lines := strings.Split(code, "\n")
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		var matches []string
		if matches = lineAnnotationPattern.FindStringSubmatch(trimmedLine); matches == nil {
			matches = blockAnnotationPattern.FindStringSubmatch(trimmedLine)
		}

		if len(matches) < 2 {
			continue
		}

		annotation := strings.ToLower(strings.TrimSpace(matches[1]))
		value := ""
		if len(matches) > 2 {
			value = strings.TrimSpace(matches[2])
		}

		switch annotation {
		case "schedule":
			// Remove surrounding quotes if present
			config.Schedule = strings.Trim(value, `"'`)
		case "description":
			config.Description = value
		case "timeout":
			if t, err := strconv.Atoi(value); err == nil && t > 0 {
				config.Timeout = t
			}
		case "memory":
			if m, err := strconv.Atoi(value); err == nil && m > 0 {
				config.Memory = m
			}
		case "max-retries":
			if r, err := strconv.Atoi(value); err == nil && r >= 0 {
				config.MaxRetries = r
			}
		case "require-role":
			// Validate role
			role := strings.ToLower(value)
			if role == "admin" || role == "authenticated" || role == "anon" {
				config.RequireRole = role
			}
		case "allow-net":
			config.AllowNet = true
		case "deny-net":
			config.AllowNet = false
		case "allow-env":
			config.AllowEnv = true
		case "deny-env":
			config.AllowEnv = false
		case "disable-logs":
			config.DisableLogs = true
		}
	}

	return config
}

// isValidCronExpression performs basic validation of cron expressions
func isValidCronExpression(expr string) bool {
	// Basic validation: should have 5 or 6 space-separated fields
	fields := strings.Fields(expr)
	if len(fields) < 5 || len(fields) > 6 {
		return false
	}

	// Each field should contain valid cron characters
	validChars := regexp.MustCompile(`^[\d\*\/\-\,\?LW#]+$`)
	for _, field := range fields {
		if !validChars.MatchString(field) {
			return false
		}
	}

	return true
}
