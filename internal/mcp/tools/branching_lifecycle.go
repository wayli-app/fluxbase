package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/branching"
	"github.com/nimbleflux/fluxbase/internal/mcp"
)

// ============================================================================
// CREATE BRANCH TOOL
// ============================================================================

// CreateBranchTool implements the create_branch MCP tool
type CreateBranchTool struct {
	manager *branching.Manager
}

// NewCreateBranchTool creates a new create_branch tool
func NewCreateBranchTool(manager *branching.Manager) *CreateBranchTool {
	return &CreateBranchTool{manager: manager}
}

func (t *CreateBranchTool) Name() string {
	return "create_branch"
}

func (t *CreateBranchTool) Description() string {
	return `Create a new isolated database branch for development or testing.

Parameters:
  - name: Branch name (required, will be used to generate slug)
  - parent_branch_id: ID of parent branch to clone from (default: main branch)
  - data_clone_mode: How to clone data: schema_only (default), full_clone, seed_data
  - type: Branch type: preview (default), persistent
  - expires_at: ISO 8601 datetime when branch should auto-delete

Returns the created branch details including connection information.`
}

func (t *CreateBranchTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Branch name (required)",
			},
			"parent_branch_id": map[string]any{
				"type":        "string",
				"description": "Parent branch UUID to clone from (default: main)",
			},
			"data_clone_mode": map[string]any{
				"type":        "string",
				"description": "How to clone data: schema_only (default), full_clone, seed_data",
				"enum":        []string{"schema_only", "full_clone", "seed_data"},
				"default":     "schema_only",
			},
			"type": map[string]any{
				"type":        "string",
				"description": "Branch type: preview (auto-expires), persistent (manual delete)",
				"enum":        []string{"preview", "persistent"},
				"default":     "preview",
			},
			"expires_at": map[string]any{
				"type":        "string",
				"description": "ISO 8601 datetime when branch should auto-delete",
			},
		},
		"required": []string{"name"},
	}
}

func (t *CreateBranchTool) RequiredScopes() []string {
	return []string{mcp.ScopeBranchWrite}
}

func (t *CreateBranchTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("branch name is required")
	}

	req := branching.CreateBranchRequest{
		Name: name,
	}

	if parentID, ok := args["parent_branch_id"].(string); ok && parentID != "" {
		id, err := uuid.Parse(parentID)
		if err != nil {
			return &mcp.ToolResult{
				Content: []mcp.Content{mcp.ErrorContent("Invalid parent_branch_id format")},
				IsError: true,
			}, nil
		}
		req.ParentBranchID = &id
	}

	if dataCloneMode, ok := args["data_clone_mode"].(string); ok && dataCloneMode != "" {
		req.DataCloneMode = branching.DataCloneMode(dataCloneMode)
	}

	if branchType, ok := args["type"].(string); ok && branchType != "" {
		req.Type = branching.BranchType(branchType)
	}

	if expiresAt, ok := args["expires_at"].(string); ok && expiresAt != "" {
		t, err := time.Parse(time.RFC3339, expiresAt)
		if err != nil {
			return &mcp.ToolResult{
				Content: []mcp.Content{mcp.ErrorContent("Invalid expires_at format. Use ISO 8601 (RFC3339)")},
				IsError: true,
			}, nil
		}
		req.ExpiresAt = &t
	}

	var createdBy *uuid.UUID
	if authCtx.UserID != nil {
		if id, err := uuid.Parse(*authCtx.UserID); err == nil {
			createdBy = &id
		}
	}

	log.Debug().
		Str("name", name).
		Str("data_clone_mode", string(req.DataCloneMode)).
		Str("type", string(req.Type)).
		Msg("MCP: create_branch - creating branch")

	branch, err := t.manager.CreateBranch(ctx, req, createdBy)
	if err != nil {
		log.Error().Err(err).Str("name", name).Msg("MCP: create_branch - failed")
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to create branch: %v", err))},
			IsError: true,
		}, nil
	}

	log.Info().
		Str("id", branch.ID.String()).
		Str("name", branch.Name).
		Str("slug", branch.Slug).
		Msg("MCP: create_branch - created")

	result := map[string]any{
		"id":              branch.ID.String(),
		"name":            branch.Name,
		"slug":            branch.Slug,
		"database_name":   branch.DatabaseName,
		"status":          string(branch.Status),
		"type":            string(branch.Type),
		"data_clone_mode": string(branch.DataCloneMode),
		"created_at":      branch.CreatedAt.Format(time.RFC3339),
	}

	if branch.ExpiresAt != nil {
		result["expires_at"] = branch.ExpiresAt.Format(time.RFC3339)
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.ToolResult{
		Content: []mcp.Content{mcp.TextContent(string(resultJSON))},
	}, nil
}

// ============================================================================
// DELETE BRANCH TOOL
// ============================================================================

// DeleteBranchTool implements the delete_branch MCP tool
type DeleteBranchTool struct {
	manager *branching.Manager
	storage *branching.Storage
}

// NewDeleteBranchTool creates a new delete_branch tool
func NewDeleteBranchTool(manager *branching.Manager, storage *branching.Storage) *DeleteBranchTool {
	return &DeleteBranchTool{manager: manager, storage: storage}
}

func (t *DeleteBranchTool) Name() string {
	return "delete_branch"
}

func (t *DeleteBranchTool) Description() string {
	return `Delete a database branch. Cannot delete the main branch.

Parameters:
  - branch_id: Branch UUID (use this OR slug)
  - slug: Branch slug (use this OR branch_id)

Returns confirmation of deletion.`
}

func (t *DeleteBranchTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"branch_id": map[string]any{
				"type":        "string",
				"description": "Branch UUID",
			},
			"slug": map[string]any{
				"type":        "string",
				"description": "Branch slug",
			},
		},
	}
}

func (t *DeleteBranchTool) RequiredScopes() []string {
	return []string{mcp.ScopeBranchWrite}
}

func (t *DeleteBranchTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	var branchID uuid.UUID

	if id, ok := args["branch_id"].(string); ok && id != "" {
		parsed, err := uuid.Parse(id)
		if err != nil {
			return &mcp.ToolResult{
				Content: []mcp.Content{mcp.ErrorContent("Invalid branch_id format")},
				IsError: true,
			}, nil
		}
		branchID = parsed
	} else if slug, ok := args["slug"].(string); ok && slug != "" {
		branch, err := t.storage.GetBranchBySlug(ctx, slug, nil)
		if err != nil {
			if errors.Is(err, branching.ErrBranchNotFound) {
				return &mcp.ToolResult{
					Content: []mcp.Content{mcp.ErrorContent("Branch not found")},
					IsError: true,
				}, nil
			}
			return &mcp.ToolResult{
				Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to find branch: %v", err))},
				IsError: true,
			}, nil
		}
		branchID = branch.ID
	} else {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent("Either branch_id or slug is required")},
			IsError: true,
		}, nil
	}

	var deletedBy *uuid.UUID
	if authCtx.UserID != nil {
		if id, err := uuid.Parse(*authCtx.UserID); err == nil {
			deletedBy = &id
		}
	}

	log.Debug().Str("branch_id", branchID.String()).Msg("MCP: delete_branch - deleting")

	if err := t.manager.DeleteBranch(ctx, branchID, deletedBy); err != nil {
		log.Error().Err(err).Str("branch_id", branchID.String()).Msg("MCP: delete_branch - failed")
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to delete branch: %v", err))},
			IsError: true,
		}, nil
	}

	log.Info().Str("branch_id", branchID.String()).Msg("MCP: delete_branch - deleted")

	result := map[string]any{
		"action":    "deleted",
		"branch_id": branchID.String(),
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.ToolResult{
		Content: []mcp.Content{mcp.TextContent(string(resultJSON))},
	}, nil
}

// ============================================================================
// RESET BRANCH TOOL
// ============================================================================

// ResetBranchTool implements the reset_branch MCP tool
type ResetBranchTool struct {
	manager *branching.Manager
	storage *branching.Storage
}

// NewResetBranchTool creates a new reset_branch tool
func NewResetBranchTool(manager *branching.Manager, storage *branching.Storage) *ResetBranchTool {
	return &ResetBranchTool{manager: manager, storage: storage}
}

func (t *ResetBranchTool) Name() string {
	return "reset_branch"
}

func (t *ResetBranchTool) Description() string {
	return `Reset a database branch to its parent's current state.

This drops all data in the branch and re-clones from the parent branch.
Cannot reset the main branch.

Parameters:
  - branch_id: Branch UUID (use this OR slug)
  - slug: Branch slug (use this OR branch_id)

Returns confirmation of reset.`
}

func (t *ResetBranchTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"branch_id": map[string]any{
				"type":        "string",
				"description": "Branch UUID",
			},
			"slug": map[string]any{
				"type":        "string",
				"description": "Branch slug",
			},
		},
	}
}

func (t *ResetBranchTool) RequiredScopes() []string {
	return []string{mcp.ScopeBranchWrite}
}

func (t *ResetBranchTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	var branchID uuid.UUID

	if id, ok := args["branch_id"].(string); ok && id != "" {
		parsed, err := uuid.Parse(id)
		if err != nil {
			return &mcp.ToolResult{
				Content: []mcp.Content{mcp.ErrorContent("Invalid branch_id format")},
				IsError: true,
			}, nil
		}
		branchID = parsed
	} else if slug, ok := args["slug"].(string); ok && slug != "" {
		branch, err := t.storage.GetBranchBySlug(ctx, slug, nil)
		if err != nil {
			if errors.Is(err, branching.ErrBranchNotFound) {
				return &mcp.ToolResult{
					Content: []mcp.Content{mcp.ErrorContent("Branch not found")},
					IsError: true,
				}, nil
			}
			return &mcp.ToolResult{
				Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to find branch: %v", err))},
				IsError: true,
			}, nil
		}
		branchID = branch.ID
	} else {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent("Either branch_id or slug is required")},
			IsError: true,
		}, nil
	}

	var resetBy *uuid.UUID
	if authCtx.UserID != nil {
		if id, err := uuid.Parse(*authCtx.UserID); err == nil {
			resetBy = &id
		}
	}

	log.Debug().Str("branch_id", branchID.String()).Msg("MCP: reset_branch - resetting")

	if err := t.manager.ResetBranch(ctx, branchID, resetBy); err != nil {
		log.Error().Err(err).Str("branch_id", branchID.String()).Msg("MCP: reset_branch - failed")
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to reset branch: %v", err))},
			IsError: true,
		}, nil
	}

	log.Info().Str("branch_id", branchID.String()).Msg("MCP: reset_branch - reset complete")

	result := map[string]any{
		"action":    "reset",
		"branch_id": branchID.String(),
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.ToolResult{
		Content: []mcp.Content{mcp.TextContent(string(resultJSON))},
	}, nil
}

// ============================================================================
// SET ACTIVE BRANCH TOOL
// ============================================================================

// SetActiveBranchTool implements the set_active_branch MCP tool
type SetActiveBranchTool struct {
	router  *branching.Router
	storage *branching.Storage
}

// NewSetActiveBranchTool creates a new set_active_branch tool
func NewSetActiveBranchTool(router *branching.Router, storage *branching.Storage) *SetActiveBranchTool {
	return &SetActiveBranchTool{router: router, storage: storage}
}

func (t *SetActiveBranchTool) Name() string {
	return "set_active_branch"
}

func (t *SetActiveBranchTool) Description() string {
	return `Set the server-wide active/default branch.

This sets the branch that will be used for all requests that don't specify a branch
via the X-Fluxbase-Branch header or ?branch= query parameter.

Parameters:
  - branch: Branch slug to set as active (use "main" for the main branch, or empty string to reset to default)

Returns the new active branch and previous branch.`
}

func (t *SetActiveBranchTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"branch": map[string]any{
				"type":        "string",
				"description": "Branch slug to set as active (empty string to reset to default)",
			},
		},
		"required": []string{"branch"},
	}
}

func (t *SetActiveBranchTool) RequiredScopes() []string {
	return []string{mcp.ScopeBranchWrite}
}

func (t *SetActiveBranchTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	branch, _ := args["branch"].(string)

	previous := t.router.GetDefaultBranch()

	if branch != "" && branch != "main" {
		_, err := t.storage.GetBranchBySlug(ctx, branch, nil)
		if err != nil {
			if errors.Is(err, branching.ErrBranchNotFound) {
				return &mcp.ToolResult{
					Content: []mcp.Content{mcp.ErrorContent("Branch not found: " + branch)},
					IsError: true,
				}, nil
			}
			return &mcp.ToolResult{
				Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to verify branch: %v", err))},
				IsError: true,
			}, nil
		}
	}

	t.router.SetActiveBranch(branch)

	newBranch := t.router.GetDefaultBranch()

	log.Info().
		Str("previous", previous).
		Str("new", newBranch).
		Msg("MCP: set_active_branch - changed")

	result := map[string]any{
		"branch":   newBranch,
		"previous": previous,
	}

	if branch == "" {
		result["message"] = "Active branch reset to default"
	} else {
		result["message"] = "Active branch set successfully"
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.ToolResult{
		Content: []mcp.Content{mcp.TextContent(string(resultJSON))},
	}, nil
}
