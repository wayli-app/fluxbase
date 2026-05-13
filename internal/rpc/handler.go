package rpc

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/logging"
	"github.com/nimbleflux/fluxbase/internal/middleware"
	"github.com/nimbleflux/fluxbase/internal/observability"
	syncframework "github.com/nimbleflux/fluxbase/internal/sync"
)

// Handler handles RPC-related HTTP endpoints
type Handler struct {
	storage        *Storage
	loader         *Loader
	executor       *Executor
	validator      *Validator
	config         *config.RPCConfig
	baseConfig     *config.Config
	authService    *auth.Service
	scheduler      *Scheduler
	loggingService *logging.Service
}

// SetScheduler sets the scheduler for procedure lifecycle management
func (h *Handler) SetScheduler(scheduler *Scheduler) {
	h.scheduler = scheduler
}

// GetExecutor returns the executor for external use (e.g., scheduler)
func (h *Handler) GetExecutor() *Executor {
	return h.executor
}

// NewHandler creates a new RPC handler
func NewHandler(db *database.Connection, storage *Storage, loader *Loader, metrics *observability.Metrics, cfg *config.RPCConfig, authService *auth.Service, loggingService *logging.Service, baseConfig *config.Config) *Handler {
	return &Handler{
		storage:        storage,
		loader:         loader,
		executor:       NewExecutor(db, storage, metrics, cfg),
		validator:      NewValidator(),
		config:         cfg,
		baseConfig:     baseConfig,
		authService:    authService,
		loggingService: loggingService,
	}
}

// getConfig returns the RPC config to use for the current request.
// It checks for tenant-specific config in fiber context locals and falls back to base config.
//
//nolint:unused // Kept for future tenant-specific config support
func (h *Handler) getConfig(c fiber.Ctx) *config.RPCConfig {
	if tc, ok := c.Locals("tenant_config").(*config.Config); ok && tc != nil {
		return &tc.RPC
	}
	return h.config
}

// ============================================================================
// ADMIN: PROCEDURE MANAGEMENT
// ============================================================================

// ListProcedures returns all procedures (admin view)
// GET /api/v1/admin/rpc/procedures
func (h *Handler) ListProcedures(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	namespace := c.Query("namespace")
	if namespace == "default" {
		namespace = ""
	}

	procedures, err := h.storage.ListProcedures(ctx, namespace)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list procedures")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list procedures",
		})
	}

	// Convert to summaries
	summaries := make([]ProcedureSummary, len(procedures))
	for i, p := range procedures {
		summaries[i] = p.ToSummary()
	}

	return c.JSON(fiber.Map{
		"procedures": summaries,
		"count":      len(summaries),
	})
}

// GetProcedure returns a single procedure by namespace and name
// GET /api/v1/admin/rpc/procedures/:namespace/:name
func (h *Handler) GetProcedure(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	namespace := c.Params("namespace")
	name := c.Params("name")

	procedure, err := h.storage.GetProcedureByName(ctx, namespace, name)
	if err != nil {
		log.Error().Err(err).Str("namespace", namespace).Str("name", name).Msg("Failed to get procedure")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get procedure",
		})
	}

	if procedure == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Procedure not found",
		})
	}

	return c.JSON(procedure)
}

// UpdateProcedureRequest represents the request body for updating a procedure
type UpdateProcedureRequest struct {
	Description             *string  `json:"description,omitempty"`
	Enabled                 *bool    `json:"enabled,omitempty"`
	IsPublic                *bool    `json:"is_public,omitempty"`
	RequireRoles            []string `json:"require_roles,omitempty"`
	MaxExecutionTimeSeconds *int     `json:"max_execution_time_seconds,omitempty"`
	AllowedTables           []string `json:"allowed_tables,omitempty"`
	AllowedSchemas          []string `json:"allowed_schemas,omitempty"`
	Schedule                *string  `json:"schedule,omitempty"`
}

// UpdateProcedure updates a procedure
// PUT /api/v1/admin/rpc/procedures/:namespace/:name
func (h *Handler) UpdateProcedure(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	namespace := c.Params("namespace")
	name := c.Params("name")

	var req UpdateProcedureRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Get existing procedure
	procedure, err := h.storage.GetProcedureByName(ctx, namespace, name)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get procedure")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get procedure",
		})
	}

	if procedure == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Procedure not found",
		})
	}

	// Apply updates
	if req.Description != nil {
		procedure.Description = *req.Description
	}
	if req.Enabled != nil {
		procedure.Enabled = *req.Enabled
	}
	if req.IsPublic != nil {
		procedure.IsPublic = *req.IsPublic
	}
	if len(req.RequireRoles) > 0 {
		procedure.RequireRoles = req.RequireRoles
	}
	if req.MaxExecutionTimeSeconds != nil {
		procedure.MaxExecutionTimeSeconds = *req.MaxExecutionTimeSeconds
	}
	if req.AllowedTables != nil {
		procedure.AllowedTables = req.AllowedTables
	}
	if req.AllowedSchemas != nil {
		procedure.AllowedSchemas = req.AllowedSchemas
	}
	if req.Schedule != nil {
		// Allow clearing schedule with empty string
		if *req.Schedule == "" {
			procedure.Schedule = nil
		} else {
			procedure.Schedule = req.Schedule
		}
	}

	if err := h.storage.UpdateProcedure(ctx, procedure); err != nil {
		log.Error().Err(err).Msg("Failed to update procedure")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update procedure",
		})
	}

	// Reschedule if scheduler is available
	if h.scheduler != nil {
		if err := h.scheduler.RescheduleProcedure(procedure); err != nil {
			log.Warn().Err(err).Str("procedure", procedure.Name).Msg("Failed to reschedule procedure")
		}
	}

	return c.JSON(procedure)
}

// DeleteProcedure deletes a procedure
// DELETE /api/v1/admin/rpc/procedures/:namespace/:name
func (h *Handler) DeleteProcedure(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	namespace := c.Params("namespace")
	name := c.Params("name")

	// Unschedule before deletion
	if h.scheduler != nil {
		h.scheduler.UnscheduleProcedure(namespace, name)
	}

	if err := h.storage.DeleteProcedureByName(ctx, namespace, name); err != nil {
		log.Error().Err(err).Str("namespace", namespace).Str("name", name).Msg("Failed to delete procedure")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete procedure",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Procedure deleted successfully",
	})
}

// ListNamespaces returns all unique namespaces
// GET /api/v1/admin/rpc/namespaces
func (h *Handler) ListNamespaces(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)

	namespaces, err := h.storage.ListNamespaces(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list namespaces")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list namespaces",
		})
	}

	// Ensure we always return at least "default"
	if len(namespaces) == 0 {
		namespaces = []string{"default"}
	}

	// Normalize empty-string namespaces to "default".
	for i := range namespaces {
		if namespaces[i] == "" {
			namespaces[i] = "default"
		}
	}

	return c.JSON(fiber.Map{
		"namespaces": namespaces,
	})
}

// ============================================================================
// ADMIN: SYNC
// ============================================================================

// SyncProcedures syncs procedures from filesystem or SDK payload
// POST /api/v1/admin/rpc/sync
func (h *Handler) SyncProcedures(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)

	syncCtx := database.ContextWithTenant(ctx, "")
	currentTenantID := database.TenantFromContext(ctx)

	var req SyncRequest
	if err := c.Bind().Body(&req); err != nil {
		req = SyncRequest{}
	}

	namespace := req.Namespace
	if namespace == "" {
		namespace = "default"
	}

	source := "sdk"
	if len(req.Procedures) == 0 {
		source = "filesystem"
	}

	var items []procedureSyncItem

	if len(req.Procedures) == 0 {
		loaded, err := h.loader.LoadProcedures()
		if err != nil {
			log.Error().Err(err).Msg("Failed to load procedures from filesystem")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to load procedures from filesystem",
			})
		}
		for _, lp := range loaded {
			items = append(items, procedureSyncItem{loaded: lp})
		}
	} else {
		for _, spec := range req.Procedures {
			annotations, sqlQuery, err := ParseAnnotations(spec.Code)
			if err != nil {
				items = append(items, procedureSyncItem{loaded: &LoadedProcedure{
					Name: spec.Name, Code: spec.Code, Namespace: namespace,
				}})
				continue
			}
			items = append(items, procedureSyncItem{loaded: &LoadedProcedure{
				Name: spec.Name, Namespace: namespace,
				Code: spec.Code, SQLQuery: sqlQuery, Annotations: annotations,
			}})
		}
	}

	syncer := newRPCSyncer(h, syncCtx, currentTenantID, source)
	opts := syncframework.Options{
		Namespace:     namespace,
		DeleteMissing: req.Options.DeleteMissing,
		DryRun:        req.Options.DryRun,
		TenantID:      currentTenantID,
	}

	syncResult, err := syncframework.Execute(ctx, syncer, items, opts)
	if err != nil {
		log.Error().Err(err).Msg("Sync failed")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	result := &SyncResult{
		Message:   syncResult.Message,
		Namespace: syncResult.Namespace,
		DryRun:    syncResult.DryRun,
		Summary: SyncSummary{
			Created:   syncResult.Summary.Created,
			Updated:   syncResult.Summary.Updated,
			Deleted:   syncResult.Summary.Deleted,
			Unchanged: syncResult.Summary.Unchanged,
			Errors:    syncResult.Summary.Errors,
		},
		Details: SyncDetails{
			Created:   syncResult.Details.Created,
			Updated:   syncResult.Details.Updated,
			Deleted:   syncResult.Details.Deleted,
			Unchanged: syncResult.Details.Unchanged,
		},
		Errors: make([]SyncError, len(syncResult.Errors)),
	}
	for i, e := range syncResult.Errors {
		result.Errors[i] = SyncError{
			Procedure: e.Name,
			Error:     e.Error,
		}
	}

	if source == "filesystem" && result.Message == "" {
		result.Message = "Synced from filesystem"
	} else if source == "sdk" && result.Message == "" {
		result.Message = "Synced from SDK payload"
	}

	return c.JSON(result)
}

// needsUpdate checks if a procedure needs to be updated
func (h *Handler) needsUpdate(existing, new *Procedure) bool {
	if existing.SQLQuery != new.SQLQuery {
		return true
	}
	if existing.OriginalCode != new.OriginalCode {
		return true
	}
	if existing.Description != new.Description {
		return true
	}
	if existing.MaxExecutionTimeSeconds != new.MaxExecutionTimeSeconds {
		return true
	}
	if existing.IsPublic != new.IsPublic {
		return true
	}
	if existing.DisableExecutionLogs != new.DisableExecutionLogs {
		return true
	}
	// Compare require_roles
	if len(existing.RequireRoles) != len(new.RequireRoles) {
		return true
	}
	for i, role := range existing.RequireRoles {
		if i >= len(new.RequireRoles) || role != new.RequireRoles[i] {
			return true
		}
	}
	// Compare schedule
	if (existing.Schedule == nil) != (new.Schedule == nil) {
		return true
	}
	if existing.Schedule != nil && new.Schedule != nil && *existing.Schedule != *new.Schedule {
		return true
	}
	// Compare arrays
	if !stringSlicesEqual(existing.AllowedTables, new.AllowedTables) {
		return true
	}
	if !stringSlicesEqual(existing.AllowedSchemas, new.AllowedSchemas) {
		return true
	}
	return false
}

// stringSlicesEqual compares two string slices for equality
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// ============================================================================
// ADMIN: EXECUTION MANAGEMENT
// ============================================================================

// ListExecutions returns execution history
// GET /api/v1/admin/rpc/executions
func (h *Handler) ListExecutions(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)

	opts := ListExecutionsOptions{
		Namespace:     c.Query("namespace"),
		ProcedureName: c.Query("procedure"),
		UserID:        c.Query("user_id"),
		Limit:         100,
	}
	if opts.Namespace == "default" {
		opts.Namespace = ""
	}

	if status := c.Query("status"); status != "" {
		opts.Status = ExecutionStatus(status)
	}

	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			opts.Limit = l
		}
	}

	if offset := c.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			opts.Offset = o
		}
	}

	executions, err := h.storage.ListExecutions(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list executions")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list executions",
		})
	}

	return c.JSON(fiber.Map{
		"executions": executions,
		"count":      len(executions),
	})
}

// GetExecution returns a single execution by ID
// GET /api/v1/admin/rpc/executions/:id
func (h *Handler) GetExecution(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	id := c.Params("id")

	execution, err := h.storage.GetExecution(ctx, id)
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to get execution")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get execution",
		})
	}

	if execution == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Execution not found",
		})
	}

	return c.JSON(execution)
}

// GetExecutionLogs returns logs for an execution
// GET /api/v1/admin/rpc/executions/:id/logs
func (h *Handler) GetExecutionLogs(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	id := c.Params("id")

	// Check if execution exists
	execution, err := h.storage.GetExecution(ctx, id)
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to get execution")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get execution",
		})
	}

	if execution == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Execution not found",
		})
	}

	// Parse after_line query param for pagination
	afterLine := 0
	if afterLineStr := c.Query("after_line"); afterLineStr != "" {
		if l, err := strconv.Atoi(afterLineStr); err == nil {
			afterLine = l
		}
	}

	// Query logs from central logging
	entries, err := h.loggingService.GetExecutionLogs(ctx, id, afterLine)
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to get execution logs")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get execution logs",
		})
	}

	return c.JSON(fiber.Map{
		"entries": entries,
		"count":   len(entries),
	})
}

// CancelExecution cancels a pending or running execution
// POST /api/v1/admin/rpc/executions/:id/cancel
func (h *Handler) CancelExecution(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	id := c.Params("id")

	// Get execution to check status
	execution, err := h.storage.GetExecution(ctx, id)
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to get execution")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get execution",
		})
	}

	if execution == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Execution not found",
		})
	}

	// Can only cancel pending or running executions
	if execution.Status != StatusPending && execution.Status != StatusRunning {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Execution cannot be cancelled",
			"status": execution.Status,
		})
	}

	// Cancel the execution
	if err := h.storage.CancelExecution(ctx, id); err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to cancel execution")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to cancel execution",
		})
	}

	// Get the updated execution
	execution, _ = h.storage.GetExecution(ctx, id)

	return c.JSON(execution)
}

// ============================================================================
// PUBLIC: PROCEDURE LISTING
// ============================================================================

// ListPublicProcedures returns public, enabled procedures
// GET /api/v1/rpc/procedures
func (h *Handler) ListPublicProcedures(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	namespace := c.Query("namespace")
	if namespace == "default" {
		namespace = ""
	}

	procedures, err := h.storage.ListPublicProcedures(ctx, namespace)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list public procedures")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list procedures",
		})
	}

	return c.JSON(fiber.Map{
		"procedures": procedures,
		"count":      len(procedures),
	})
}

// ============================================================================
// PUBLIC: INVOCATION
// ============================================================================

// Invoke invokes an RPC procedure
// POST /api/v1/rpc/:namespace/:name
func (h *Handler) Invoke(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	namespace := c.Params("namespace")
	name := c.Params("name")

	// Get procedure
	procedure, err := h.storage.GetProcedureByName(ctx, namespace, name)
	if err != nil {
		log.Error().Err(err).Str("namespace", namespace).Str("name", name).Msg("Failed to get procedure")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get procedure",
		})
	}

	if procedure == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Procedure not found",
		})
	}

	// Check if procedure is enabled
	if !procedure.Enabled {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Procedure not found",
		})
	}

	// Get user context from locals
	userID := ""
	userRole := "anon"
	userEmail := ""
	var claims *auth.TokenClaims
	isAuthenticated := false

	if uid := middleware.GetUserID(c); uid != "" {
		userID = uid
		isAuthenticated = true
	}
	if role := middleware.GetUserRole(c); role != "" {
		userRole = role
	}
	// Check both "email" and "user_email" for compatibility
	if email, ok := c.Locals("user_email").(string); ok {
		userEmail = email
	} else if email, ok := c.Locals("email").(string); ok {
		userEmail = email
	}
	// Check both "jwt_claims" and "claims" for compatibility
	if tc, ok := c.Locals("jwt_claims").(*auth.TokenClaims); ok {
		claims = tc
	} else if tc, ok := c.Locals("claims").(*auth.TokenClaims); ok {
		claims = tc
	}

	// Check for impersonation token - allows admin to invoke RPC as another user
	impersonationToken := c.Get("X-Impersonation-Token")
	if impersonationToken != "" && h.authService != nil {
		impersonationClaims, err := h.authService.JWTManager().ValidateToken(impersonationToken)
		if err != nil {
			log.Warn().Err(err).Msg("Invalid impersonation token in RPC invocation")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid impersonation token",
			})
		}

		// Override user context with impersonated user
		userID = impersonationClaims.UserID
		userRole = impersonationClaims.Role
		userEmail = impersonationClaims.Email
		claims = impersonationClaims
		isAuthenticated = true

		log.Info().
			Str("procedure", name).
			Str("impersonated_user_id", impersonationClaims.UserID).
			Str("impersonated_role", impersonationClaims.Role).
			Msg("RPC invocation with impersonation")
	}

	// Validate access
	if err := h.validator.ValidateAccess(procedure, userRole, isAuthenticated); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Parse request body
	var req InvokeRequest
	if err := c.Bind().Body(&req); err != nil {
		// Body is optional
		req = InvokeRequest{}
	}

	// Build execution context
	execCtx := &ExecuteContext{
		Procedure:            procedure,
		Params:               req.Params,
		UserID:               userID,
		UserRole:             userRole,
		UserEmail:            userEmail,
		Claims:               claims,
		IsAsync:              req.Async,
		DisableExecutionLogs: procedure.DisableExecutionLogs,
	}

	// Execute
	var result *ExecuteResult
	if req.Async {
		result, err = h.executor.ExecuteAsync(ctx, execCtx)
	} else {
		result, err = h.executor.Execute(ctx, execCtx)
	}

	if err != nil {
		log.Error().Err(err).Str("procedure", name).Msg("Failed to execute procedure")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to execute procedure",
		})
	}

	return c.JSON(result)
}

// GetPublicExecution returns execution status for user's own execution
// GET /api/v1/rpc/executions/:id
func (h *Handler) GetPublicExecution(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	id := c.Params("id")

	userID := middleware.GetUserID(c)

	execution, err := h.storage.GetExecution(ctx, id)
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to get execution")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get execution",
		})
	}

	if execution == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Execution not found",
		})
	}

	// Check ownership (unless service role)
	role := middleware.GetUserRole(c)
	if role != "service_role" && role != "instance_admin" && role != "tenant_service" {
		if execution.UserID == nil || *execution.UserID != userID {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Execution not found",
			})
		}
	}

	return c.JSON(execution)
}

// GetPublicExecutionLogs returns logs for user's own execution
// GET /api/v1/rpc/executions/:id/logs
func (h *Handler) GetPublicExecutionLogs(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	id := c.Params("id")

	userID := middleware.GetUserID(c)

	// Check execution exists and belongs to user
	execution, err := h.storage.GetExecution(ctx, id)
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to get execution")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get execution",
		})
	}

	if execution == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Execution not found",
		})
	}

	// Check ownership (unless service_role or instance_admin)
	role := middleware.GetUserRole(c)
	if role != "service_role" && role != "instance_admin" && role != "tenant_service" {
		if execution.UserID == nil || *execution.UserID != userID {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Execution not found",
			})
		}
	}

	// Parse after_line query param for pagination
	afterLine := 0
	if afterLineStr := c.Query("after_line"); afterLineStr != "" {
		if l, err := strconv.Atoi(afterLineStr); err == nil {
			afterLine = l
		}
	}

	// Query logs from central logging
	entries, err := h.loggingService.GetExecutionLogs(ctx, id, afterLine)
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to get execution logs")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get execution logs",
		})
	}

	return c.JSON(fiber.Map{
		"entries": entries,
		"count":   len(entries),
	})
}

// fiber:context-methods migrated
