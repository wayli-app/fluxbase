package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/branching"
	"github.com/nimbleflux/fluxbase/internal/mcp"
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
		branch, err = t.storage.GetBranch(ctx, id, nil)
	} else if slug, ok := args["slug"].(string); ok && slug != "" {
		branch, err = t.storage.GetBranchBySlug(ctx, slug, nil)
	} else {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent("Either branch_id or slug is required")},
			IsError: true,
		}, nil
	}

	if err != nil {
		if errors.Is(err, branching.ErrBranchNotFound) {
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
// GET ACTIVE BRANCH TOOL
// ============================================================================

// GetActiveBranchTool implements the get_active_branch MCP tool
type GetActiveBranchTool struct {
	router *branching.Router
}

// NewGetActiveBranchTool creates a new get_active_branch tool
func NewGetActiveBranchTool(router *branching.Router) *GetActiveBranchTool {
	return &GetActiveBranchTool{router: router}
}

func (t *GetActiveBranchTool) Name() string {
	return "get_active_branch"
}

func (t *GetActiveBranchTool) Description() string {
	return `Get the current server-wide active/default branch.

Returns the active branch and its source (api, config, or default).
The active branch is used when no per-request branch is specified via header or query param.`
}

func (t *GetActiveBranchTool) InputSchema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (t *GetActiveBranchTool) RequiredScopes() []string {
	return []string{mcp.ScopeBranchRead}
}

func (t *GetActiveBranchTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	branch := t.router.GetDefaultBranch()
	source := t.router.GetActiveBranchSource()

	result := map[string]any{
		"branch": branch,
		"source": source,
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

	accessLevel := branching.BranchAccessLevel(strings.ToLower(accessLevelStr))
	if accessLevel != branching.BranchAccessRead &&
		accessLevel != branching.BranchAccessWrite &&
		accessLevel != branching.BranchAccessAdmin {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent("Invalid access_level. Must be: read, write, or admin")},
			IsError: true,
		}, nil
	}

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
