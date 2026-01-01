package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/mcp"
	"github.com/fluxbase-eu/fluxbase/internal/rpc"
	"github.com/rs/zerolog/log"
)

// SyncRPCTool implements the sync_rpc MCP tool for deploying RPC procedures
type SyncRPCTool struct {
	storage *rpc.Storage
}

// NewSyncRPCTool creates a new sync_rpc tool
func NewSyncRPCTool(storage *rpc.Storage) *SyncRPCTool {
	return &SyncRPCTool{
		storage: storage,
	}
}

func (t *SyncRPCTool) Name() string {
	return "sync_rpc"
}

func (t *SyncRPCTool) Description() string {
	return `Deploy or update an RPC procedure (stored SQL). Parses @fluxbase annotations from SQL comments.

Supported annotations:
  @fluxbase:description <text> - Procedure description
  @fluxbase:public - Make procedure publicly discoverable
  @fluxbase:timeout <seconds> - Max execution time (default: 30)
  @fluxbase:require-role <role> - Required role: admin, authenticated, anon
  @fluxbase:allowed-tables <tables> - Comma-separated list of allowed tables
  @fluxbase:allowed-schemas <schemas> - Comma-separated list of allowed schemas
  @fluxbase:schedule "<cron>" - Optional cron schedule for periodic execution
  @fluxbase:disable-logs - Disable execution logging

Example:
-- @fluxbase:description Get user profile with stats
-- @fluxbase:public
-- @fluxbase:allowed-tables users,user_stats
-- @fluxbase:timeout 10
SELECT u.*, s.total_posts, s.followers
FROM users u
LEFT JOIN user_stats s ON u.id = s.user_id
WHERE u.id = $1;`
}

func (t *SyncRPCTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Procedure name (alphanumeric, hyphens, underscores)",
			},
			"sql_code": map[string]any{
				"type":        "string",
				"description": "SQL code with optional @fluxbase annotations in comments",
			},
			"namespace": map[string]any{
				"type":        "string",
				"description": "Namespace for isolating procedures (default: 'default')",
				"default":     "default",
			},
		},
		"required": []string{"name", "sql_code"},
	}
}

func (t *SyncRPCTool) RequiredScopes() []string {
	return []string{mcp.ScopeSyncRPC}
}

func (t *SyncRPCTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	// Parse arguments
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("procedure name is required")
	}

	sqlCode, ok := args["sql_code"].(string)
	if !ok || sqlCode == "" {
		return nil, fmt.Errorf("sql_code is required")
	}

	namespace := "default"
	if ns, ok := args["namespace"].(string); ok && ns != "" {
		namespace = ns
	}

	// Validate name format
	if !isValidFunctionName(name) {
		return nil, fmt.Errorf("invalid procedure name: must be alphanumeric with hyphens/underscores, 1-63 characters")
	}

	// Check namespace access
	if !authCtx.HasNamespaceAccess(namespace) {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Access denied to namespace: %s", namespace))},
			IsError: true,
		}, nil
	}

	// Parse annotations from SQL comments
	config := parseRPCAnnotations(sqlCode)

	log.Debug().
		Str("name", name).
		Str("namespace", namespace).
		Interface("config", config).
		Msg("MCP: sync_rpc - parsed annotations")

	// Check if procedure already exists
	existing, err := t.storage.GetProcedureByName(ctx, namespace, name)
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to check existing procedure: %v", err))},
			IsError: true,
		}, nil
	}

	var result map[string]any

	if existing == nil {
		// Create new procedure
		proc := &rpc.Procedure{
			Name:                    name,
			Namespace:               namespace,
			Description:             config.Description,
			SQLQuery:                sqlCode,
			OriginalCode:            sqlCode,
			AllowedTables:           config.AllowedTables,
			AllowedSchemas:          config.AllowedSchemas,
			MaxExecutionTimeSeconds: config.Timeout,
			RequireRole:             config.RequireRole,
			IsPublic:                config.IsPublic,
			DisableExecutionLogs:    config.DisableLogs,
			Schedule:                config.Schedule,
			Enabled:                 true,
			Version:                 1,
			Source:                  "mcp",
		}

		if err := t.storage.CreateProcedure(ctx, proc); err != nil {
			return &mcp.ToolResult{
				Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to create procedure: %v", err))},
				IsError: true,
			}, nil
		}

		result = map[string]any{
			"action":    "created",
			"id":        proc.ID,
			"name":      proc.Name,
			"namespace": proc.Namespace,
			"version":   proc.Version,
		}

		log.Info().
			Str("name", name).
			Str("namespace", namespace).
			Str("id", proc.ID).
			Msg("MCP: sync_rpc - created new procedure")

	} else {
		// Update existing procedure
		existing.Description = config.Description
		existing.SQLQuery = sqlCode
		existing.OriginalCode = sqlCode
		existing.AllowedTables = config.AllowedTables
		existing.AllowedSchemas = config.AllowedSchemas
		existing.MaxExecutionTimeSeconds = config.Timeout
		existing.RequireRole = config.RequireRole
		existing.IsPublic = config.IsPublic
		existing.DisableExecutionLogs = config.DisableLogs
		existing.Schedule = config.Schedule
		existing.Source = "mcp"

		if err := t.storage.UpdateProcedure(ctx, existing); err != nil {
			return &mcp.ToolResult{
				Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to update procedure: %v", err))},
				IsError: true,
			}, nil
		}

		result = map[string]any{
			"action":           "updated",
			"id":               existing.ID,
			"name":             name,
			"namespace":        namespace,
			"previous_version": existing.Version,
		}

		log.Info().
			Str("name", name).
			Str("namespace", namespace).
			Str("id", existing.ID).
			Int("previous_version", existing.Version).
			Msg("MCP: sync_rpc - updated existing procedure")
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

// RPCConfig holds parsed @fluxbase annotations for RPC procedures
type RPCConfig struct {
	Description    string
	IsPublic       bool
	Timeout        int
	RequireRole    *string
	AllowedTables  []string
	AllowedSchemas []string
	Schedule       *string
	DisableLogs    bool
}

// parseRPCAnnotations extracts configuration from @fluxbase: comments in SQL code
func parseRPCAnnotations(sqlCode string) RPCConfig {
	config := RPCConfig{
		Timeout:        30, // Default timeout
		AllowedSchemas: []string{"public"},
		AllowedTables:  []string{},
	}

	// Match @fluxbase:annotation patterns in SQL comments
	// Process line by line to avoid multiline regex matching issues
	lineAnnotationPattern := regexp.MustCompile(`^--\s*@fluxbase:(\S+)(?:\s+(.*))?$`)
	blockAnnotationPattern := regexp.MustCompile(`^\s*\*\s*@fluxbase:(\S+)(?:\s+(.*))?$`)

	lines := strings.Split(sqlCode, "\n")
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
		case "description":
			config.Description = value
		case "public":
			config.IsPublic = true
		case "timeout":
			if t, err := strconv.Atoi(value); err == nil && t > 0 {
				config.Timeout = t
			}
		case "require-role":
			role := strings.ToLower(value)
			if role == "admin" || role == "authenticated" || role == "anon" {
				config.RequireRole = &role
			}
		case "allowed-tables":
			tables := parseCommaSeparatedList(value)
			if len(tables) > 0 {
				config.AllowedTables = tables
			}
		case "allowed-schemas":
			schemas := parseCommaSeparatedList(value)
			if len(schemas) > 0 {
				config.AllowedSchemas = schemas
			}
		case "schedule":
			schedule := strings.Trim(value, `"'`)
			if schedule != "" {
				config.Schedule = &schedule
			}
		case "disable-logs":
			config.DisableLogs = true
		}
	}

	return config
}

// parseCommaSeparatedList parses a comma-separated string into a slice
func parseCommaSeparatedList(s string) []string {
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
