package api

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/branching"
	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// BranchHandler handles branch management API endpoints
type BranchHandler struct {
	manager *branching.Manager
	router  *branching.Router
	config  config.BranchingConfig
}

// NewBranchHandler creates a new branch handler
func NewBranchHandler(manager *branching.Manager, router *branching.Router, cfg config.BranchingConfig) *BranchHandler {
	return &BranchHandler{
		manager: manager,
		router:  router,
		config:  cfg,
	}
}

func (h *BranchHandler) requireManager(c fiber.Ctx) error {
	if h.manager == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "Branch manager not initialized")
	}
	return nil
}

func (h *BranchHandler) requireRouter(c fiber.Ctx) error {
	if h.router == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "Branch router not initialized")
	}
	return nil
}

// getTenantFilter returns a tenant ID filter for the current request.
// Returns nil for instance admins (no filter) or when no tenant context is available.
func getTenantFilter(c fiber.Ctx) *uuid.UUID {
	isInstanceAdmin, _ := c.Locals("is_instance_admin").(bool)
	if isInstanceAdmin {
		return nil
	}
	authType, _ := c.Locals("auth_type").(string)
	if authType == "service_key" {
		return nil
	}
	if tid := middleware.GetTenantID(c); tid != "" {
		if id, err := uuid.Parse(tid); err == nil {
			return &id
		}
	}
	return nil
}

// CreateBranchRequest represents the request body for creating a branch
type CreateBranchRequest struct {
	Name           string                  `json:"name"`
	TenantID       *uuid.UUID              `json:"tenant_id,omitempty"`
	ParentBranchID *uuid.UUID              `json:"parent_branch_id,omitempty"`
	DataCloneMode  branching.DataCloneMode `json:"data_clone_mode,omitempty"`
	Type           branching.BranchType    `json:"type,omitempty"`
	GitHubPRNumber *int                    `json:"github_pr_number,omitempty"`
	GitHubPRURL    *string                 `json:"github_pr_url,omitempty"`
	GitHubRepo     *string                 `json:"github_repo,omitempty"`
	ExpiresIn      *string                 `json:"expires_in,omitempty"` // Duration string like "24h", "7d"
}

// CreateBranch handles POST /admin/branches
func (h *BranchHandler) CreateBranch(c fiber.Ctx) error {
	if !h.config.Enabled {
		return SendErrorWithCode(c, 503, "Database branching is not enabled", "SERVICE_UNAVAILABLE")
	}

	var req CreateBranchRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Name == "" {
		return SendBadRequest(c, "Branch name is required", ErrCodeMissingField)
	}

	// Get user ID from context
	var userID *uuid.UUID
	if uid := middleware.GetUserID(c); uid != "" {
		if id, err := uuid.Parse(uid); err == nil {
			userID = &id
		}
	}

	// Get tenant ID from context (set by tenant middleware)
	var tenantID *uuid.UUID
	if tid := middleware.GetTenantID(c); tid != "" {
		if id, err := uuid.Parse(tid); err == nil {
			tenantID = &id
		}
	}

	// If request doesn't specify tenant_id, use context tenant
	if req.TenantID == nil && tenantID != nil {
		req.TenantID = tenantID
	}

	// Parse expires_in to ExpiresAt
	var expiresAt *time.Time
	if req.ExpiresIn != nil && *req.ExpiresIn != "" {
		duration, err := time.ParseDuration(*req.ExpiresIn)
		if err != nil {
			return SendBadRequest(c, "Invalid expires_in duration format", ErrCodeInvalidFormat)
		}
		t := time.Now().Add(duration)
		expiresAt = &t
	}

	// Normalize DataCloneModeFull alias ("full" -> "full_clone")
	if req.DataCloneMode == branching.DataCloneModeFull {
		req.DataCloneMode = branching.DataCloneModeFullClone
	}

	if err := h.requireManager(c); err != nil {
		return err
	}
	if err := h.requireRouter(c); err != nil {
		return err
	}

	// Create branch request
	branchReq := branching.CreateBranchRequest{
		Name:           req.Name,
		TenantID:       req.TenantID,
		ParentBranchID: req.ParentBranchID,
		DataCloneMode:  req.DataCloneMode,
		Type:           req.Type,
		GitHubPRNumber: req.GitHubPRNumber,
		GitHubPRURL:    req.GitHubPRURL,
		GitHubRepo:     req.GitHubRepo,
		ExpiresAt:      expiresAt,
	}

	branch, err := h.manager.CreateBranch(c.RequestCtx(), branchReq, userID)
	if err != nil {
		log.Error().Err(err).Str("name", req.Name).Msg("Failed to create branch")

		if errors.Is(err, branching.ErrBranchExists) {
			return SendConflict(c, "A branch with this name already exists", ErrCodeAlreadyExists)
		}
		if errors.Is(err, branching.ErrMaxBranchesReached) {
			return SendForbidden(c, "Maximum number of branches has been reached", ErrCodeAccessDenied)
		}
		if errors.Is(err, branching.ErrInvalidSlug) {
			return SendBadRequest(c, "Branch name contains invalid characters", ErrCodeInvalidInput)
		}
		return SendInternalError(c, "Failed to create branch")
	}

	go func() {
		if err := h.router.WarmupPool(c.RequestCtx(), branch.Slug); err != nil {
			log.Warn().Err(err).Str("slug", branch.Slug).Msg("Failed to warmup branch pool")
		}
	}()

	return c.Status(fiber.StatusCreated).JSON(branch)
}

// ListBranches handles GET /admin/branches
func (h *BranchHandler) ListBranches(c fiber.Ctx) error {
	filter := branching.ListBranchesFilter{
		Limit:  100,
		Offset: 0,
	}

	// Parse query parameters
	if limit := fiber.Query[int](c, "limit", 100); limit > 0 && limit <= 1000 {
		filter.Limit = limit
	}
	if offset := fiber.Query[int](c, "offset", 0); offset >= 0 {
		filter.Offset = offset
	}
	if status := c.Query("status"); status != "" {
		s := branching.BranchStatus(status)
		filter.Status = &s
	}
	if branchType := c.Query("type"); branchType != "" {
		t := branching.BranchType(branchType)
		filter.Type = &t
	}
	if repo := c.Query("github_repo"); repo != "" {
		filter.GitHubRepo = &repo
	}

	// Get user ID for filtering their branches
	if c.Query("mine") == "true" {
		if uid := middleware.GetUserID(c); uid != "" {
			if id, err := uuid.Parse(uid); err == nil {
				filter.CreatedBy = &id
			}
		}
	}

	// Auto-filter by tenant for non-instance-admins
	userRole, _ := c.Locals("user_role").(string)
	if userRole != "instance_admin" && userRole != "admin" {
		if tid := middleware.GetTenantID(c); tid != "" {
			if id, err := uuid.Parse(tid); err == nil {
				filter.TenantID = &id
			}
		}
	}

	if err := h.requireManager(c); err != nil {
		return err
	}

	branches, err := h.manager.GetStorage().ListBranches(c.RequestCtx(), filter)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list branches")
		return SendInternalError(c, "Failed to list branches")
	}

	// Get total count
	total, err := h.manager.GetStorage().CountBranches(c.RequestCtx(), filter)
	if err != nil {
		log.Error().Err(err).Msg("Failed to count branches")
		total = len(branches)
	}

	return c.JSON(fiber.Map{
		"branches": branches,
		"total":    total,
		"limit":    filter.Limit,
		"offset":   filter.Offset,
	})
}

// GetBranch handles GET /admin/branches/:id
func (h *BranchHandler) GetBranch(c fiber.Ctx) error {
	idParam := c.Params("id")

	// Try to parse as UUID first
	var branch *branching.Branch
	var err error
	tenantFilter := getTenantFilter(c)

	if err := h.requireManager(c); err != nil {
		return err
	}

	if id, parseErr := uuid.Parse(idParam); parseErr == nil {
		branch, err = h.manager.GetStorage().GetBranch(c.RequestCtx(), id, tenantFilter)
	} else {
		// Try as slug
		branch, err = h.manager.GetStorage().GetBranchBySlug(c.RequestCtx(), idParam, tenantFilter)
	}

	if err != nil {
		if errors.Is(err, branching.ErrBranchNotFound) {
			return SendNotFound(c, "Branch not found")
		}
		log.Error().Err(err).Str("id", idParam).Msg("Failed to get branch")
		return SendInternalError(c, "Failed to get branch")
	}

	return c.JSON(branch)
}

// DeleteBranch handles DELETE /admin/branches/:id
func (h *BranchHandler) DeleteBranch(c fiber.Ctx) error {
	idParam := c.Params("id")

	// Try to parse as UUID first
	var branchID uuid.UUID
	var branch *branching.Branch
	var err error
	tenantFilter := getTenantFilter(c)

	if err := h.requireManager(c); err != nil {
		return err
	}
	if err := h.requireRouter(c); err != nil {
		return err
	}

	if id, parseErr := uuid.Parse(idParam); parseErr == nil {
		branchID = id
		branch, err = h.manager.GetStorage().GetBranch(c.RequestCtx(), id, tenantFilter)
	} else {
		// Try as slug
		branch, err = h.manager.GetStorage().GetBranchBySlug(c.RequestCtx(), idParam, tenantFilter)
		if err == nil {
			branchID = branch.ID
		}
	}

	if err != nil {
		if errors.Is(err, branching.ErrBranchNotFound) {
			return SendNotFound(c, "Branch not found")
		}
		return SendInternalError(c, "Failed to get branch")
	}

	// Get user ID from context
	var userID *uuid.UUID
	if uid := middleware.GetUserID(c); uid != "" {
		if id, err := uuid.Parse(uid); err == nil {
			userID = &id
		}
	}

	// Check authorization - service keys and dashboard admins bypass this check
	authType, _ := c.Locals("auth_type").(string)
	userRole, _ := c.Locals("user_role").(string)
	isAdmin := authType == "service_key" || userRole == "instance_admin" || userRole == "admin" || userRole == "tenant_service"

	if !isAdmin && userID != nil {
		// Check if user has admin access to the branch
		hasAccess, err := h.manager.GetStorage().HasAccess(c.RequestCtx(), branch.ID, *userID, branching.BranchAccessAdmin)
		if err != nil {
			log.Error().Err(err).Str("branch_id", branch.ID.String()).Msg("Failed to check branch access")
			return SendInternalError(c, "Failed to verify branch access")
		}
		if !hasAccess {
			return SendForbidden(c, "You do not have permission to delete this branch", ErrCodeAccessDenied)
		}
	}

	h.router.ClosePool(branch.Slug)

	// Delete the branch
	if err := h.manager.DeleteBranch(c.RequestCtx(), branchID, userID); err != nil {
		log.Error().Err(err).Str("id", idParam).Msg("Failed to delete branch")

		if errors.Is(err, branching.ErrCannotDeleteMainBranch) {
			return SendForbidden(c, "Cannot delete the main branch", ErrCodeAccessDenied)
		}

		return SendInternalError(c, "Failed to delete branch")
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// ResetBranch handles POST /admin/branches/:id/reset
func (h *BranchHandler) ResetBranch(c fiber.Ctx) error {
	idParam := c.Params("id")

	// Try to parse as UUID first
	var branchID uuid.UUID
	var branch *branching.Branch
	var err error
	tenantFilter := getTenantFilter(c)

	if err := h.requireManager(c); err != nil {
		return err
	}
	if err := h.requireRouter(c); err != nil {
		return err
	}

	if id, parseErr := uuid.Parse(idParam); parseErr == nil {
		branchID = id
		branch, err = h.manager.GetStorage().GetBranch(c.RequestCtx(), id, tenantFilter)
	} else {
		// Try as slug
		branch, err = h.manager.GetStorage().GetBranchBySlug(c.RequestCtx(), idParam, tenantFilter)
		if err == nil {
			branchID = branch.ID
		}
	}

	if err != nil {
		if errors.Is(err, branching.ErrBranchNotFound) {
			return SendNotFound(c, "Branch not found")
		}
		return SendInternalError(c, "Failed to get branch")
	}

	// Get user ID from context
	var userID *uuid.UUID
	if uid := middleware.GetUserID(c); uid != "" {
		if id, err := uuid.Parse(uid); err == nil {
			userID = &id
		}
	}

	// Check authorization - service keys and dashboard admins bypass this check
	authType, _ := c.Locals("auth_type").(string)
	userRole, _ := c.Locals("user_role").(string)
	isAdmin := authType == "service_key" || userRole == "instance_admin" || userRole == "admin" || userRole == "tenant_service"

	if !isAdmin && userID != nil {
		// Check if user has admin access to the branch (reset is a destructive operation)
		hasAccess, err := h.manager.GetStorage().HasAccess(c.RequestCtx(), branch.ID, *userID, branching.BranchAccessAdmin)
		if err != nil {
			log.Error().Err(err).Str("branch_id", branch.ID.String()).Msg("Failed to check branch access")
			return SendInternalError(c, "Failed to verify branch access")
		}
		if !hasAccess {
			return SendForbidden(c, "You do not have permission to reset this branch", ErrCodeAccessDenied)
		}
	}

	h.router.ClosePool(branch.Slug)

	// Reset the branch
	if err := h.manager.ResetBranch(c.RequestCtx(), branchID, userID); err != nil {
		log.Error().Err(err).Str("id", idParam).Msg("Failed to reset branch")

		if errors.Is(err, branching.ErrCannotDeleteMainBranch) {
			return SendForbidden(c, "Cannot reset the main branch", ErrCodeAccessDenied)
		}

		return SendInternalError(c, "Failed to reset branch")
	}

	if err := h.router.RefreshPool(c.RequestCtx(), branch.Slug); err != nil {
		log.Warn().Err(err).Str("slug", branch.Slug).Msg("Failed to refresh branch pool after reset")
	}

	// Get updated branch
	updatedBranch, _ := h.manager.GetStorage().GetBranch(c.RequestCtx(), branchID, nil)
	if updatedBranch != nil {
		return c.JSON(updatedBranch)
	}

	return c.JSON(fiber.Map{"status": "reset_complete"})
}

// GetBranchActivity handles GET /admin/branches/:id/activity
func (h *BranchHandler) GetBranchActivity(c fiber.Ctx) error {
	idParam := c.Params("id")

	// Try to parse as UUID first
	var branchID uuid.UUID

	if err := h.requireManager(c); err != nil {
		return err
	}

	if id, parseErr := uuid.Parse(idParam); parseErr == nil {
		branchID = id
	} else {
		// Try as slug
		branch, err := h.manager.GetStorage().GetBranchBySlug(c.RequestCtx(), idParam, getTenantFilter(c))
		if err != nil {
			if errors.Is(err, branching.ErrBranchNotFound) {
				return SendNotFound(c, "Branch not found")
			}
			return SendInternalError(c, "Failed to get branch")
		}
		branchID = branch.ID
	}

	limit := fiber.Query[int](c, "limit", 50)
	if limit > 100 {
		limit = 100
	}

	activity, err := h.manager.GetStorage().GetActivityLog(c.RequestCtx(), branchID, limit)
	if err != nil {
		log.Error().Err(err).Str("id", idParam).Msg("Failed to get branch activity")
		return SendInternalError(c, "Failed to get branch activity")
	}

	return c.JSON(fiber.Map{
		"activity": activity,
	})
}

// GetPoolStats handles GET /admin/branches/stats/pools
func (h *BranchHandler) GetPoolStats(c fiber.Ctx) error {
	if err := h.requireRouter(c); err != nil {
		return err
	}

	stats := h.router.GetPoolStats()
	return c.JSON(fiber.Map{
		"pools": stats,
	})
}

// GetActiveBranch handles GET /admin/branches/active
func (h *BranchHandler) GetActiveBranch(c fiber.Ctx) error {
	if err := h.requireRouter(c); err != nil {
		return err
	}

	branch := h.router.GetDefaultBranch()
	source := h.router.GetActiveBranchSource()

	return c.JSON(fiber.Map{
		"branch": branch,
		"source": source,
	})
}

// SetActiveBranchRequest represents the request body for setting the active branch
type SetActiveBranchRequest struct {
	Branch string `json:"branch"`
}

// SetActiveBranch handles POST /admin/branches/active
func (h *BranchHandler) SetActiveBranch(c fiber.Ctx) error {
	if !h.config.Enabled {
		return SendErrorWithCode(c, 503, "Database branching is not enabled", "SERVICE_UNAVAILABLE")
	}

	var req SetActiveBranchRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Branch == "" {
		return SendBadRequest(c, "Branch slug is required", ErrCodeMissingField)
	}

	if err := h.requireManager(c); err != nil {
		return err
	}
	if err := h.requireRouter(c); err != nil {
		return err
	}

	// Verify the branch exists (unless it's "main")
	if req.Branch != "main" {
		_, err := h.manager.GetStorage().GetBranchBySlug(c.RequestCtx(), req.Branch, nil)
		if err != nil {
			if errors.Is(err, branching.ErrBranchNotFound) {
				return SendNotFound(c, "Branch not found: "+req.Branch)
			}
			log.Error().Err(err).Str("branch", req.Branch).Msg("Failed to verify branch")
			return SendInternalError(c, "Failed to verify branch exists")
		}
	}

	// Get previous branch for response
	previous := h.router.GetDefaultBranch()

	// Set the active branch
	h.router.SetActiveBranch(req.Branch)

	return c.JSON(fiber.Map{
		"branch":   req.Branch,
		"previous": previous,
		"message":  "Active branch set successfully",
	})
}

// ResetActiveBranch handles DELETE /admin/branches/active
func (h *BranchHandler) ResetActiveBranch(c fiber.Ctx) error {
	if err := h.requireRouter(c); err != nil {
		return err
	}

	// Get current branch for response
	previous := h.router.GetDefaultBranch()

	// Reset to default (empty string clears API-set value)
	h.router.SetActiveBranch("")

	// Get new default branch
	newBranch := h.router.GetDefaultBranch()

	return c.JSON(fiber.Map{
		"branch":   newBranch,
		"previous": previous,
		"message":  "Active branch reset to default",
	})
}

// GitHub Config handlers

// ListGitHubConfigs handles GET /admin/branches/github/configs
func (h *BranchHandler) ListGitHubConfigs(c fiber.Ctx) error {
	// Get tenant ID from context (set by tenant middleware)
	var tenantID *uuid.UUID
	if tid := middleware.GetTenantID(c); tid != "" {
		if id, err := uuid.Parse(tid); err == nil {
			tenantID = &id
		}
	}

	// Auto-filter by tenant for non-instance-admins
	userRole, _ := c.Locals("user_role").(string)
	if userRole == "instance_admin" || userRole == "admin" {
		// Instance admins can see all configs (pass nil)
		tenantID = nil
	}

	if err := h.requireManager(c); err != nil {
		return err
	}

	configs, err := h.manager.GetStorage().ListGitHubConfigs(c.RequestCtx(), tenantID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list GitHub configs")
		return SendInternalError(c, "Failed to list GitHub configurations")
	}

	return c.JSON(fiber.Map{
		"configs": configs,
	})
}

// UpsertGitHubConfigRequest represents the request for creating/updating GitHub config
type UpsertGitHubConfigRequest struct {
	Repository           string                  `json:"repository"`
	AutoCreateOnPR       *bool                   `json:"auto_create_on_pr,omitempty"`
	AutoDeleteOnMerge    *bool                   `json:"auto_delete_on_merge,omitempty"`
	DefaultDataCloneMode branching.DataCloneMode `json:"default_data_clone_mode,omitempty"`
	WebhookSecret        *string                 `json:"webhook_secret,omitempty"`
}

// UpsertGitHubConfig handles POST /admin/branches/github/configs
func (h *BranchHandler) UpsertGitHubConfig(c fiber.Ctx) error {
	var req UpsertGitHubConfigRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Repository == "" {
		return SendBadRequest(c, "Repository is required", ErrCodeMissingField)
	}

	if err := h.requireManager(c); err != nil {
		return err
	}

	config := &branching.GitHubConfig{
		Repository:           req.Repository,
		AutoCreateOnPR:       true, // Default
		AutoDeleteOnMerge:    true, // Default
		DefaultDataCloneMode: branching.DataCloneModeSchemaOnly,
	}

	if req.AutoCreateOnPR != nil {
		config.AutoCreateOnPR = *req.AutoCreateOnPR
	}
	if req.AutoDeleteOnMerge != nil {
		config.AutoDeleteOnMerge = *req.AutoDeleteOnMerge
	}
	if req.DefaultDataCloneMode != "" {
		config.DefaultDataCloneMode = req.DefaultDataCloneMode
	}
	if req.WebhookSecret != nil {
		config.WebhookSecret = req.WebhookSecret
	}

	if err := h.manager.GetStorage().UpsertGitHubConfig(c.RequestCtx(), config); err != nil {
		log.Error().Err(err).Str("repository", req.Repository).Msg("Failed to upsert GitHub config")
		return SendInternalError(c, "Failed to save GitHub configuration")
	}

	return c.Status(fiber.StatusOK).JSON(config)
}

// DeleteGitHubConfig handles DELETE /admin/branches/github/configs/:repository
func (h *BranchHandler) DeleteGitHubConfig(c fiber.Ctx) error {
	repository := c.Params("repository")

	if err := h.requireManager(c); err != nil {
		return err
	}

	if err := h.manager.GetStorage().DeleteGitHubConfig(c.RequestCtx(), repository); err != nil {
		if errors.Is(err, branching.ErrGitHubConfigNotFound) {
			return SendNotFound(c, "GitHub configuration not found")
		}
		log.Error().Err(err).Str("repository", repository).Msg("Failed to delete GitHub config")
		return SendInternalError(c, "Failed to delete GitHub configuration")
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// Access Management Handlers

// ListBranchAccess handles GET /admin/branches/:id/access
func (h *BranchHandler) ListBranchAccess(c fiber.Ctx) error {
	idParam := c.Params("id")

	// Try to parse as UUID first, then as slug
	var branch *branching.Branch
	var err error

	if err := h.requireManager(c); err != nil {
		return err
	}

	if id, parseErr := uuid.Parse(idParam); parseErr == nil {
		branch, err = h.manager.GetStorage().GetBranch(c.RequestCtx(), id, getTenantFilter(c))
	} else {
		branch, err = h.manager.GetStorage().GetBranchBySlug(c.RequestCtx(), idParam, getTenantFilter(c))
	}

	if err != nil {
		if errors.Is(err, branching.ErrBranchNotFound) {
			return SendNotFound(c, "Branch not found")
		}
		return SendInternalError(c, "Failed to get branch")
	}

	// Check authorization
	var userID *uuid.UUID
	if uid := middleware.GetUserID(c); uid != "" {
		if id, err := uuid.Parse(uid); err == nil {
			userID = &id
		}
	}

	authType, _ := c.Locals("auth_type").(string)
	userRole, _ := c.Locals("user_role").(string)
	isAdmin := authType == "service_key" || userRole == "instance_admin" || userRole == "admin" || userRole == "tenant_service"

	if !isAdmin && userID != nil {
		hasAccess, err := h.manager.GetStorage().HasAccess(c.RequestCtx(), branch.ID, *userID, branching.BranchAccessAdmin)
		if err != nil {
			return SendInternalError(c, "Failed to verify branch access")
		}
		if !hasAccess {
			return SendForbidden(c, "You do not have permission to view access grants for this branch", ErrCodeAccessDenied)
		}
	}

	accessList, err := h.manager.GetStorage().GetBranchAccessList(c.RequestCtx(), branch.ID)
	if err != nil {
		log.Error().Err(err).Str("branch_id", branch.ID.String()).Msg("Failed to list branch access")
		return SendInternalError(c, "Failed to list branch access")
	}

	return c.JSON(fiber.Map{
		"access": accessList,
	})
}

// GrantBranchAccessRequest represents the request body for granting access
type GrantBranchAccessRequest struct {
	UserID      string `json:"user_id"`
	AccessLevel string `json:"access_level"`
}

// GrantBranchAccess handles POST /admin/branches/:id/access
func (h *BranchHandler) GrantBranchAccess(c fiber.Ctx) error {
	idParam := c.Params("id")

	// Try to parse as UUID first, then as slug
	var branch *branching.Branch
	var err error

	if err := h.requireManager(c); err != nil {
		return err
	}

	if id, parseErr := uuid.Parse(idParam); parseErr == nil {
		branch, err = h.manager.GetStorage().GetBranch(c.RequestCtx(), id, getTenantFilter(c))
	} else {
		branch, err = h.manager.GetStorage().GetBranchBySlug(c.RequestCtx(), idParam, getTenantFilter(c))
	}

	if err != nil {
		if errors.Is(err, branching.ErrBranchNotFound) {
			return SendNotFound(c, "Branch not found")
		}
		return SendInternalError(c, "Failed to get branch")
	}

	// Parse request body
	var req GrantBranchAccessRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.UserID == "" {
		return SendBadRequest(c, "user_id is required", ErrCodeMissingField)
	}

	targetUserID, err := uuid.Parse(req.UserID)
	if err != nil {
		return SendBadRequest(c, "Invalid user_id format", ErrCodeInvalidID)
	}

	accessLevel := branching.BranchAccessLevel(req.AccessLevel)
	if accessLevel != branching.BranchAccessRead &&
		accessLevel != branching.BranchAccessWrite &&
		accessLevel != branching.BranchAccessAdmin {
		return SendBadRequest(c, "access_level must be one of: read, write, admin", ErrCodeInvalidInput)
	}

	// Check authorization
	var grantedBy *uuid.UUID
	if uid := middleware.GetUserID(c); uid != "" {
		if id, err := uuid.Parse(uid); err == nil {
			grantedBy = &id
		}
	}

	authType, _ := c.Locals("auth_type").(string)
	userRole, _ := c.Locals("user_role").(string)
	isAdmin := authType == "service_key" || userRole == "instance_admin" || userRole == "admin" || userRole == "tenant_service"

	if !isAdmin && grantedBy != nil {
		hasAccess, err := h.manager.GetStorage().HasAccess(c.RequestCtx(), branch.ID, *grantedBy, branching.BranchAccessAdmin)
		if err != nil {
			return SendInternalError(c, "Failed to verify branch access")
		}
		if !hasAccess {
			return SendForbidden(c, "You do not have permission to grant access to this branch", ErrCodeAccessDenied)
		}
	}

	// Grant access
	access := &branching.BranchAccess{
		ID:          uuid.New(),
		BranchID:    branch.ID,
		UserID:      targetUserID,
		AccessLevel: accessLevel,
		GrantedBy:   grantedBy,
	}

	if err := h.manager.GetStorage().GrantAccess(c.RequestCtx(), access); err != nil {
		log.Error().Err(err).
			Str("branch_id", branch.ID.String()).
			Str("user_id", targetUserID.String()).
			Msg("Failed to grant branch access")
		return SendInternalError(c, "Failed to grant access")
	}

	// Log activity
	_ = h.manager.GetStorage().LogActivity(c.RequestCtx(), &branching.ActivityLog{
		BranchID:   branch.ID,
		Action:     branching.ActivityActionAccessGranted,
		Status:     branching.ActivityStatusSuccess,
		ExecutedBy: grantedBy,
		Details: map[string]any{
			"user_id":      targetUserID.String(),
			"access_level": string(accessLevel),
		},
	})

	return c.Status(fiber.StatusCreated).JSON(access)
}

// RevokeBranchAccess handles DELETE /admin/branches/:id/access/:user_id
func (h *BranchHandler) RevokeBranchAccess(c fiber.Ctx) error {
	idParam := c.Params("id")
	userIDParam := c.Params("user_id")

	// Try to parse as UUID first, then as slug
	var branch *branching.Branch
	var err error

	if err := h.requireManager(c); err != nil {
		return err
	}

	if id, parseErr := uuid.Parse(idParam); parseErr == nil {
		branch, err = h.manager.GetStorage().GetBranch(c.RequestCtx(), id, getTenantFilter(c))
	} else {
		branch, err = h.manager.GetStorage().GetBranchBySlug(c.RequestCtx(), idParam, getTenantFilter(c))
	}

	if err != nil {
		if errors.Is(err, branching.ErrBranchNotFound) {
			return SendNotFound(c, "Branch not found")
		}
		return SendInternalError(c, "Failed to get branch")
	}

	targetUserID, err := uuid.Parse(userIDParam)
	if err != nil {
		return SendBadRequest(c, "Invalid user_id format", ErrCodeInvalidID)
	}

	// Check authorization
	var currentUserID *uuid.UUID
	if uid := middleware.GetUserID(c); uid != "" {
		if id, err := uuid.Parse(uid); err == nil {
			currentUserID = &id
		}
	}

	authType, _ := c.Locals("auth_type").(string)
	userRole, _ := c.Locals("user_role").(string)
	isAdmin := authType == "service_key" || userRole == "instance_admin" || userRole == "admin" || userRole == "tenant_service"

	if !isAdmin && currentUserID != nil {
		hasAccess, err := h.manager.GetStorage().HasAccess(c.RequestCtx(), branch.ID, *currentUserID, branching.BranchAccessAdmin)
		if err != nil {
			return SendInternalError(c, "Failed to verify branch access")
		}
		if !hasAccess {
			return SendForbidden(c, "You do not have permission to revoke access from this branch", ErrCodeAccessDenied)
		}
	}

	// Revoke access
	if err := h.manager.GetStorage().RevokeAccess(c.RequestCtx(), branch.ID, targetUserID); err != nil {
		log.Error().Err(err).
			Str("branch_id", branch.ID.String()).
			Str("user_id", targetUserID.String()).
			Msg("Failed to revoke branch access")
		return SendInternalError(c, "Failed to revoke access")
	}

	// Log activity
	_ = h.manager.GetStorage().LogActivity(c.RequestCtx(), &branching.ActivityLog{
		BranchID:   branch.ID,
		Action:     branching.ActivityActionAccessRevoked,
		Status:     branching.ActivityStatusSuccess,
		ExecutedBy: currentUserID,
		Details: map[string]any{
			"user_id": targetUserID.String(),
		},
	})

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// fiber:context-methods migrated
