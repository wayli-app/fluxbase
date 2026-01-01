package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/branching"
	"github.com/fluxbase-eu/fluxbase/internal/mcp"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// ============================================================================
// LIST BRANCHES TOOL
// ============================================================================

// ListBranchesTool implements the list_branches MCP tool
type ListBranchesTool struct {
	storage *branching.Storage
}

// NewListBranchesTool creates a new list_branches tool
func NewListBranchesTool(storage *branching.Storage) *ListBranchesTool {
	return &ListBranchesTool{storage: storage}
}

func (t *ListBranchesTool) Name() string {
	return "list_branches"
}

func (t *ListBranchesTool) Description() string {
	return `List database branches with optional filtering.

Parameters:
  - status: Filter by status (creating, ready, migrating, error, deleting)
  - type: Filter by type (main, preview, persistent)
  - limit: Maximum number of results (default: 50, max: 100)
  - offset: Number of results to skip for pagination

Returns list of branches with id, name, slug, status, type, and timestamps.`
}

func (t *ListBranchesTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"status": map[string]any{
				"type":        "string",
				"description": "Filter by branch status: creating, ready, migrating, error, deleting",
				"enum":        []string{"creating", "ready", "migrating", "error", "deleting"},
			},
			"type": map[string]any{
				"type":        "string",
				"description": "Filter by branch type: main, preview, persistent",
				"enum":        []string{"main", "preview", "persistent"},
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of results (default: 50, max: 100)",
				"default":     50,
			},
			"offset": map[string]any{
				"type":        "integer",
				"description": "Number of results to skip for pagination",
				"default":     0,
			},
		},
	}
}

func (t *ListBranchesTool) RequiredScopes() []string {
	return []string{mcp.ScopeBranchRead}
}

func (t *ListBranchesTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	filter := branching.ListBranchesFilter{
		Limit:  50,
		Offset: 0,
	}

	if status, ok := args["status"].(string); ok && status != "" {
		s := branching.BranchStatus(status)
		filter.Status = &s
	}

	if branchType, ok := args["type"].(string); ok && branchType != "" {
		t := branching.BranchType(branchType)
		filter.Type = &t
	}

	if limit, ok := args["limit"].(float64); ok {
		filter.Limit = int(limit)
		if filter.Limit > 100 {
			filter.Limit = 100
		}
	}

	if offset, ok := args["offset"].(float64); ok {
		filter.Offset = int(offset)
	}

	branches, err := t.storage.ListBranches(ctx, filter)
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to list branches: %v", err))},
			IsError: true,
		}, nil
	}

	// Convert to simplified response
	result := make([]map[string]any, 0, len(branches))
	for _, b := range branches {
		item := map[string]any{
			"id":         b.ID.String(),
			"name":       b.Name,
			"slug":       b.Slug,
			"status":     string(b.Status),
			"type":       string(b.Type),
			"created_at": b.CreatedAt.Format(time.RFC3339),
		}
		if b.ParentBranchID != nil {
			item["parent_branch_id"] = b.ParentBranchID.String()
		}
		if b.ExpiresAt != nil {
			item["expires_at"] = b.ExpiresAt.Format(time.RFC3339)
		}
		result = append(result, item)
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.ToolResult{
		Content: []mcp.Content{mcp.TextContent(string(resultJSON))},
	}, nil
}

// ============================================================================
// GET BRANCH TOOL
// ============================================================================

// GetBranchTool implements the get_branch MCP tool
type GetBranchTool struct {
	storage *branching.Storage
}

// NewGetBranchTool creates a new get_branch tool
func NewGetBranchTool(storage *branching.Storage) *GetBranchTool {
	return &GetBranchTool{storage: storage}
}

func (t *GetBranchTool) Name() string {
	return "get_branch"
}

func (t *GetBranchTool) Description() string {
	return `Get details of a specific database branch by ID or slug.

Parameters:
  - branch_id: Branch UUID (use this OR slug)
  - slug: Branch slug (use this OR branch_id)

Returns complete branch details including database name, status, and configuration.`
}

func (t *GetBranchTool) InputSchema() map[string]any {
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

func (t *GetBranchTool) RequiredScopes() []string {
	return []string{mcp.ScopeBranchRead}
}

func (t *GetBranchTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	var branch *branching.Branch
	var err error

	if branchID, ok := args["branch_id"].(string); ok && branchID != "" {
		id, parseErr := uuid.Parse(branchID)
		if parseErr != nil {
			return &mcp.ToolResult{
				Content: []mcp.Content{mcp.ErrorContent("Invalid branch_id format")},
				IsError: true,
			}, nil
		}
		branch, err = t.storage.GetBranch(ctx, id)
	} else if slug, ok := args["slug"].(string); ok && slug != "" {
		branch, err = t.storage.GetBranchBySlug(ctx, slug)
	} else {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent("Either branch_id or slug is required")},
			IsError: true,
		}, nil
	}

	if err != nil {
		if err == branching.ErrBranchNotFound {
			return &mcp.ToolResult{
				Content: []mcp.Content{mcp.ErrorContent("Branch not found")},
				IsError: true,
			}, nil
		}
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to get branch: %v", err))},
			IsError: true,
		}, nil
	}

	result := map[string]any{
		"id":              branch.ID.String(),
		"name":            branch.Name,
		"slug":            branch.Slug,
		"database_name":   branch.DatabaseName,
		"status":          string(branch.Status),
		"type":            string(branch.Type),
		"data_clone_mode": string(branch.DataCloneMode),
		"created_at":      branch.CreatedAt.Format(time.RFC3339),
		"updated_at":      branch.UpdatedAt.Format(time.RFC3339),
	}

	if branch.ParentBranchID != nil {
		result["parent_branch_id"] = branch.ParentBranchID.String()
	}
	if branch.ExpiresAt != nil {
		result["expires_at"] = branch.ExpiresAt.Format(time.RFC3339)
	}
	if branch.ErrorMessage != nil {
		result["error_message"] = *branch.ErrorMessage
	}
	if branch.GitHubPRNumber != nil {
		result["github_pr_number"] = *branch.GitHubPRNumber
	}
	if branch.GitHubPRURL != nil {
		result["github_pr_url"] = *branch.GitHubPRURL
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.ToolResult{
		Content: []mcp.Content{mcp.TextContent(string(resultJSON))},
	}, nil
}

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

	// Get user ID for created_by
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
		branch, err := t.storage.GetBranchBySlug(ctx, slug)
		if err != nil {
			if err == branching.ErrBranchNotFound {
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

	// Get user ID for audit
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
		branch, err := t.storage.GetBranchBySlug(ctx, slug)
		if err != nil {
			if err == branching.ErrBranchNotFound {
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

	// Get user ID for audit
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
// GRANT BRANCH ACCESS TOOL
// ============================================================================

// GrantBranchAccessTool implements the grant_branch_access MCP tool
type GrantBranchAccessTool struct {
	storage *branching.Storage
}

// NewGrantBranchAccessTool creates a new grant_branch_access tool
func NewGrantBranchAccessTool(storage *branching.Storage) *GrantBranchAccessTool {
	return &GrantBranchAccessTool{storage: storage}
}

func (t *GrantBranchAccessTool) Name() string {
	return "grant_branch_access"
}

func (t *GrantBranchAccessTool) Description() string {
	return `Grant a user access to a database branch.

Parameters:
  - branch_id: Branch UUID
  - user_id: User UUID to grant access to
  - access_level: Access level: read, write, admin

Returns confirmation of access grant.`
}

func (t *GrantBranchAccessTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"branch_id": map[string]any{
				"type":        "string",
				"description": "Branch UUID",
			},
			"user_id": map[string]any{
				"type":        "string",
				"description": "User UUID to grant access to",
			},
			"access_level": map[string]any{
				"type":        "string",
				"description": "Access level: read, write, admin",
				"enum":        []string{"read", "write", "admin"},
			},
		},
		"required": []string{"branch_id", "user_id", "access_level"},
	}
}

func (t *GrantBranchAccessTool) RequiredScopes() []string {
	return []string{mcp.ScopeBranchAccess}
}

func (t *GrantBranchAccessTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	branchIDStr, ok := args["branch_id"].(string)
	if !ok || branchIDStr == "" {
		return nil, fmt.Errorf("branch_id is required")
	}

	branchID, err := uuid.Parse(branchIDStr)
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent("Invalid branch_id format")},
			IsError: true,
		}, nil
	}

	userIDStr, ok := args["user_id"].(string)
	if !ok || userIDStr == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent("Invalid user_id format")},
			IsError: true,
		}, nil
	}

	accessLevelStr, ok := args["access_level"].(string)
	if !ok || accessLevelStr == "" {
		return nil, fmt.Errorf("access_level is required")
	}

	// Validate access level
	accessLevel := branching.BranchAccessLevel(strings.ToLower(accessLevelStr))
	if accessLevel != branching.BranchAccessRead &&
		accessLevel != branching.BranchAccessWrite &&
		accessLevel != branching.BranchAccessAdmin {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent("Invalid access_level. Must be: read, write, or admin")},
			IsError: true,
		}, nil
	}

	// Get granter ID
	var grantedBy *uuid.UUID
	if authCtx.UserID != nil {
		if id, err := uuid.Parse(*authCtx.UserID); err == nil {
			grantedBy = &id
		}
	}

	access := &branching.BranchAccess{
		BranchID:    branchID,
		UserID:      userID,
		AccessLevel: accessLevel,
		GrantedBy:   grantedBy,
	}

	if err := t.storage.GrantAccess(ctx, access); err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to grant access: %v", err))},
			IsError: true,
		}, nil
	}

	log.Info().
		Str("branch_id", branchID.String()).
		Str("user_id", userID.String()).
		Str("access_level", string(accessLevel)).
		Msg("MCP: grant_branch_access - granted")

	result := map[string]any{
		"action":       "granted",
		"branch_id":    branchID.String(),
		"user_id":      userID.String(),
		"access_level": string(accessLevel),
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.ToolResult{
		Content: []mcp.Content{mcp.TextContent(string(resultJSON))},
	}, nil
}

// ============================================================================
// REVOKE BRANCH ACCESS TOOL
// ============================================================================

// RevokeBranchAccessTool implements the revoke_branch_access MCP tool
type RevokeBranchAccessTool struct {
	storage *branching.Storage
}

// NewRevokeBranchAccessTool creates a new revoke_branch_access tool
func NewRevokeBranchAccessTool(storage *branching.Storage) *RevokeBranchAccessTool {
	return &RevokeBranchAccessTool{storage: storage}
}

func (t *RevokeBranchAccessTool) Name() string {
	return "revoke_branch_access"
}

func (t *RevokeBranchAccessTool) Description() string {
	return `Revoke a user's access to a database branch.

Parameters:
  - branch_id: Branch UUID
  - user_id: User UUID to revoke access from

Returns confirmation of access revocation.`
}

func (t *RevokeBranchAccessTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"branch_id": map[string]any{
				"type":        "string",
				"description": "Branch UUID",
			},
			"user_id": map[string]any{
				"type":        "string",
				"description": "User UUID to revoke access from",
			},
		},
		"required": []string{"branch_id", "user_id"},
	}
}

func (t *RevokeBranchAccessTool) RequiredScopes() []string {
	return []string{mcp.ScopeBranchAccess}
}

func (t *RevokeBranchAccessTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	branchIDStr, ok := args["branch_id"].(string)
	if !ok || branchIDStr == "" {
		return nil, fmt.Errorf("branch_id is required")
	}

	branchID, err := uuid.Parse(branchIDStr)
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent("Invalid branch_id format")},
			IsError: true,
		}, nil
	}

	userIDStr, ok := args["user_id"].(string)
	if !ok || userIDStr == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent("Invalid user_id format")},
			IsError: true,
		}, nil
	}

	if err := t.storage.RevokeAccess(ctx, branchID, userID); err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to revoke access: %v", err))},
			IsError: true,
		}, nil
	}

	log.Info().
		Str("branch_id", branchID.String()).
		Str("user_id", userID.String()).
		Msg("MCP: revoke_branch_access - revoked")

	result := map[string]any{
		"action":    "revoked",
		"branch_id": branchID.String(),
		"user_id":   userID.String(),
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.ToolResult{
		Content: []mcp.Content{mcp.TextContent(string(resultJSON))},
	}, nil
}
