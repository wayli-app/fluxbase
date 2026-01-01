package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/mcp"
	"github.com/fluxbase-eu/fluxbase/internal/migrations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// SyncMigrationTool implements the sync_migration MCP tool for deploying database migrations
type SyncMigrationTool struct {
	storage  *migrations.Storage
	executor *migrations.Executor
}

// NewSyncMigrationTool creates a new sync_migration tool
func NewSyncMigrationTool(storage *migrations.Storage, executor *migrations.Executor) *SyncMigrationTool {
	return &SyncMigrationTool{
		storage:  storage,
		executor: executor,
	}
}

func (t *SyncMigrationTool) Name() string {
	return "sync_migration"
}

func (t *SyncMigrationTool) Description() string {
	return `Create a database migration. CAUTION: Migrations modify database schema.

By default, migrations are created in 'pending' status and require explicit application.
For safety, auto_apply defaults to false and dry_run defaults to true.

Parameters:
  - name: Migration name (will be prefixed with timestamp, e.g., "20241201_120000_add_users_table")
  - up_sql: SQL statements to apply the migration
  - down_sql: Optional SQL to rollback the migration
  - namespace: Namespace (default: 'default')
  - description: Optional description
  - auto_apply: Whether to apply immediately (default: false - creates in pending status)
  - dry_run: When true, validates SQL but doesn't save (default: true for safety)

Example:
sync_migration({
  name: "add_user_stats",
  up_sql: "CREATE TABLE user_stats (user_id UUID REFERENCES users(id), total_posts INT DEFAULT 0);",
  down_sql: "DROP TABLE user_stats;",
  dry_run: false
})`
}

func (t *SyncMigrationTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Migration name (will be prefixed with timestamp)",
			},
			"up_sql": map[string]any{
				"type":        "string",
				"description": "SQL statements to apply the migration",
			},
			"down_sql": map[string]any{
				"type":        "string",
				"description": "SQL statements to rollback the migration (recommended but optional)",
			},
			"namespace": map[string]any{
				"type":        "string",
				"description": "Namespace for isolating migrations (default: 'default')",
				"default":     "default",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Optional description of what the migration does",
			},
			"auto_apply": map[string]any{
				"type":        "boolean",
				"description": "Whether to apply the migration immediately (default: false)",
				"default":     false,
			},
			"dry_run": map[string]any{
				"type":        "boolean",
				"description": "Validate SQL without saving (default: true for safety)",
				"default":     true,
			},
		},
		"required": []string{"name", "up_sql"},
	}
}

func (t *SyncMigrationTool) RequiredScopes() []string {
	return []string{mcp.ScopeSyncMigrations}
}

func (t *SyncMigrationTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	// Parse arguments
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("migration name is required")
	}

	upSQL, ok := args["up_sql"].(string)
	if !ok || upSQL == "" {
		return nil, fmt.Errorf("up_sql is required")
	}

	downSQL := ""
	if ds, ok := args["down_sql"].(string); ok {
		downSQL = ds
	}

	namespace := "default"
	if ns, ok := args["namespace"].(string); ok && ns != "" {
		namespace = ns
	}

	description := ""
	if desc, ok := args["description"].(string); ok {
		description = desc
	}

	// IMPORTANT: Default dry_run to true for safety
	dryRun := true
	if dr, ok := args["dry_run"].(bool); ok {
		dryRun = dr
	}

	autoApply := false
	if aa, ok := args["auto_apply"].(bool); ok {
		autoApply = aa
	}

	// Validate name format (allow alphanumeric, underscores, hyphens)
	if !isValidMigrationName(name) {
		return nil, fmt.Errorf("invalid migration name: must be alphanumeric with underscores/hyphens, 1-100 characters")
	}

	// Check namespace access
	if !authCtx.HasNamespaceAccess(namespace) {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Access denied to namespace: %s", namespace))},
			IsError: true,
		}, nil
	}

	// Validate SQL (basic security checks)
	if err := validateMigrationSQL(upSQL); err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Invalid up_sql: %v", err))},
			IsError: true,
		}, nil
	}

	if downSQL != "" {
		if err := validateMigrationSQL(downSQL); err != nil {
			return &mcp.ToolResult{
				Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Invalid down_sql: %v", err))},
				IsError: true,
			}, nil
		}
	}

	// Generate timestamped name
	timestamp := time.Now().Format("20060102_150405")
	fullName := fmt.Sprintf("%s_%s", timestamp, name)

	log.Debug().
		Str("name", fullName).
		Str("namespace", namespace).
		Bool("dry_run", dryRun).
		Bool("auto_apply", autoApply).
		Msg("MCP: sync_migration - processing migration")

	// If dry run, return validation result without saving
	if dryRun {
		result := map[string]any{
			"action":     "dry_run",
			"name":       fullName,
			"namespace":  namespace,
			"validated":  true,
			"message":    "Migration SQL validated successfully. Set dry_run=false to create the migration.",
			"has_down":   downSQL != "",
			"auto_apply": autoApply,
		}

		resultJSON, _ := json.MarshalIndent(result, "", "  ")
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.TextContent(string(resultJSON))},
		}, nil
	}

	// Check if migration already exists
	existing, err := t.storage.GetMigration(ctx, namespace, fullName)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) && !strings.Contains(err.Error(), "no rows") {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to check existing migration: %v", err))},
			IsError: true,
		}, nil
	}

	if existing != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Migration already exists: %s", fullName))},
			IsError: true,
		}, nil
	}

	// Create migration
	migration := &migrations.Migration{
		Namespace: namespace,
		Name:      fullName,
		UpSQL:     upSQL,
	}

	if description != "" {
		migration.Description = &description
	}
	if downSQL != "" {
		migration.DownSQL = &downSQL
	}

	// Set created_by if we have a user ID
	if authCtx.UserID != nil {
		if userID, err := uuid.Parse(*authCtx.UserID); err == nil {
			migration.CreatedBy = &userID
		}
	}

	if err := t.storage.CreateMigration(ctx, migration); err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to create migration: %v", err))},
			IsError: true,
		}, nil
	}

	log.Info().
		Str("name", fullName).
		Str("namespace", namespace).
		Str("id", migration.ID.String()).
		Msg("MCP: sync_migration - created migration")

	result := map[string]any{
		"action":    "created",
		"id":        migration.ID.String(),
		"name":      fullName,
		"namespace": namespace,
		"status":    migration.Status,
		"has_down":  downSQL != "",
	}

	// Auto-apply if requested
	if autoApply {
		var executedBy *uuid.UUID
		if authCtx.UserID != nil {
			if userID, err := uuid.Parse(*authCtx.UserID); err == nil {
				executedBy = &userID
			}
		}

		if err := t.executor.ApplyMigration(ctx, namespace, fullName, executedBy); err != nil {
			result["apply_error"] = err.Error()
			result["status"] = "failed"

			log.Error().
				Err(err).
				Str("name", fullName).
				Str("namespace", namespace).
				Msg("MCP: sync_migration - failed to apply migration")
		} else {
			result["status"] = "applied"
			log.Info().
				Str("name", fullName).
				Str("namespace", namespace).
				Msg("MCP: sync_migration - applied migration")
		}
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

// isValidMigrationName validates migration name format
func isValidMigrationName(name string) bool {
	if len(name) == 0 || len(name) > 100 {
		return false
	}
	// Allow alphanumeric, underscores, hyphens; must start with letter or underscore
	match, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_-]*$`, name)
	return match
}

// validateMigrationSQL performs basic security validation on migration SQL
func validateMigrationSQL(sql string) error {
	normalizedSQL := strings.ToLower(strings.TrimSpace(sql))

	// Block dangerous operations on system schemas
	dangerousPatterns := []struct {
		pattern string
		message string
	}{
		{`drop\s+schema\s+(auth|storage|jobs|functions|branching|migrations)`, "Cannot drop system schemas"},
		{`alter\s+schema\s+(auth|storage|jobs|functions|branching|migrations)`, "Cannot alter system schemas"},
		{`drop\s+database`, "Cannot drop database via migrations"},
		{`create\s+database`, "Cannot create database via migrations"},
		{`truncate\s+(auth|storage|jobs|functions|branching|migrations)\.`, "Cannot truncate system tables"},
		{`drop\s+table\s+(auth|storage|jobs|functions|branching|migrations)\.`, "Cannot drop system tables"},
	}

	for _, dp := range dangerousPatterns {
		matched, _ := regexp.MatchString(dp.pattern, normalizedSQL)
		if matched {
			return fmt.Errorf("%s", dp.message)
		}
	}

	// Ensure SQL is not empty after trimming
	if normalizedSQL == "" {
		return fmt.Errorf("SQL cannot be empty")
	}

	return nil
}
