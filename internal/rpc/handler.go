package rpc

import (
	"strconv"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/fluxbase-eu/fluxbase/internal/observability"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// Handler handles RPC-related HTTP endpoints
type Handler struct {
	storage   *Storage
	loader    *Loader
	executor  *Executor
	validator *Validator
	config    *config.RPCConfig
}

// NewHandler creates a new RPC handler
func NewHandler(db *database.Connection, storage *Storage, loader *Loader, metrics *observability.Metrics, cfg *config.RPCConfig) *Handler {
	return &Handler{
		storage:   storage,
		loader:    loader,
		executor:  NewExecutor(db, storage, metrics, cfg),
		validator: NewValidator(),
		config:    cfg,
	}
}

// ============================================================================
// ADMIN: PROCEDURE MANAGEMENT
// ============================================================================

// ListProcedures returns all procedures (admin view)
// GET /api/v1/admin/rpc/procedures
func (h *Handler) ListProcedures(c *fiber.Ctx) error {
	ctx := c.Context()
	namespace := c.Query("namespace")

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
func (h *Handler) GetProcedure(c *fiber.Ctx) error {
	ctx := c.Context()
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
	RequireRole             *string  `json:"require_role,omitempty"`
	MaxExecutionTimeSeconds *int     `json:"max_execution_time_seconds,omitempty"`
	AllowedTables           []string `json:"allowed_tables,omitempty"`
	AllowedSchemas          []string `json:"allowed_schemas,omitempty"`
}

// UpdateProcedure updates a procedure
// PUT /api/v1/admin/rpc/procedures/:namespace/:name
func (h *Handler) UpdateProcedure(c *fiber.Ctx) error {
	ctx := c.Context()
	namespace := c.Params("namespace")
	name := c.Params("name")

	var req UpdateProcedureRequest
	if err := c.BodyParser(&req); err != nil {
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
	if req.RequireRole != nil {
		procedure.RequireRole = req.RequireRole
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

	if err := h.storage.UpdateProcedure(ctx, procedure); err != nil {
		log.Error().Err(err).Msg("Failed to update procedure")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update procedure",
		})
	}

	return c.JSON(procedure)
}

// DeleteProcedure deletes a procedure
// DELETE /api/v1/admin/rpc/procedures/:namespace/:name
func (h *Handler) DeleteProcedure(c *fiber.Ctx) error {
	ctx := c.Context()
	namespace := c.Params("namespace")
	name := c.Params("name")

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
func (h *Handler) ListNamespaces(c *fiber.Ctx) error {
	ctx := c.Context()

	namespaces, err := h.storage.ListNamespaces(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list namespaces")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list namespaces",
		})
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
func (h *Handler) SyncProcedures(c *fiber.Ctx) error {
	ctx := c.Context()

	var req SyncRequest
	if err := c.BodyParser(&req); err != nil {
		// Body is optional, continue with defaults
		req = SyncRequest{}
	}

	namespace := req.Namespace
	if namespace == "" {
		namespace = "default"
	}

	result := &SyncResult{
		Namespace: namespace,
		DryRun:    req.Options.DryRun,
		Summary:   SyncSummary{},
		Details: SyncDetails{
			Created:   []string{},
			Updated:   []string{},
			Deleted:   []string{},
			Unchanged: []string{},
		},
		Errors: []SyncError{},
	}

	var specsToSync []*LoadedProcedure

	// Determine source: filesystem or SDK payload
	if len(req.Procedures) == 0 {
		// Load from filesystem
		loaded, err := h.loader.LoadProcedures()
		if err != nil {
			log.Error().Err(err).Msg("Failed to load procedures from filesystem")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to load procedures from filesystem",
			})
		}
		specsToSync = loaded
		result.Message = "Synced from filesystem"
	} else {
		// Use SDK payload
		for _, spec := range req.Procedures {
			annotations, sqlQuery, err := ParseAnnotations(spec.Code)
			if err != nil {
				result.Errors = append(result.Errors, SyncError{
					Procedure: spec.Name,
					Error:     err.Error(),
				})
				result.Summary.Errors++
				continue
			}

			loaded := &LoadedProcedure{
				Name:        spec.Name,
				Namespace:   namespace,
				Code:        spec.Code,
				SQLQuery:    sqlQuery,
				Annotations: annotations,
			}
			specsToSync = append(specsToSync, loaded)
		}
		result.Message = "Synced from SDK payload"
	}

	// Get existing procedures in namespace
	existing, err := h.storage.ListProcedures(ctx, namespace)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list existing procedures")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list existing procedures",
		})
	}

	existingMap := make(map[string]*Procedure)
	for _, p := range existing {
		existingMap[p.Name] = p
	}

	// Track which procedures are in the sync
	syncedNames := make(map[string]bool)

	// Process each procedure
	for _, spec := range specsToSync {
		syncedNames[spec.Name] = true

		proc := spec.ToProcedure()
		proc.Namespace = namespace
		proc.Source = "sdk"
		if len(req.Procedures) == 0 {
			proc.Source = "filesystem"
		}

		existingProc, exists := existingMap[spec.Name]

		if !exists {
			// Create new procedure
			if !req.Options.DryRun {
				if err := h.storage.CreateProcedure(ctx, proc); err != nil {
					result.Errors = append(result.Errors, SyncError{
						Procedure: spec.Name,
						Error:     err.Error(),
					})
					result.Summary.Errors++
					continue
				}
			}
			result.Details.Created = append(result.Details.Created, spec.Name)
			result.Summary.Created++
		} else {
			// Check if update is needed
			if h.needsUpdate(existingProc, proc) {
				proc.ID = existingProc.ID
				if !req.Options.DryRun {
					if err := h.storage.UpdateProcedure(ctx, proc); err != nil {
						result.Errors = append(result.Errors, SyncError{
							Procedure: spec.Name,
							Error:     err.Error(),
						})
						result.Summary.Errors++
						continue
					}
				}
				result.Details.Updated = append(result.Details.Updated, spec.Name)
				result.Summary.Updated++
			} else {
				result.Details.Unchanged = append(result.Details.Unchanged, spec.Name)
				result.Summary.Unchanged++
			}
		}
	}

	// Handle deletion of missing procedures
	if req.Options.DeleteMissing {
		for name, proc := range existingMap {
			if !syncedNames[name] && proc.Source != "api" {
				if !req.Options.DryRun {
					if err := h.storage.DeleteProcedure(ctx, proc.ID); err != nil {
						result.Errors = append(result.Errors, SyncError{
							Procedure: name,
							Error:     err.Error(),
						})
						result.Summary.Errors++
						continue
					}
				}
				result.Details.Deleted = append(result.Details.Deleted, name)
				result.Summary.Deleted++
			}
		}
	}

	return c.JSON(result)
}

// needsUpdate checks if a procedure needs to be updated
func (h *Handler) needsUpdate(existing, new *Procedure) bool {
	if existing.SQLQuery != new.SQLQuery {
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
	// Compare require_role
	if (existing.RequireRole == nil) != (new.RequireRole == nil) {
		return true
	}
	if existing.RequireRole != nil && new.RequireRole != nil && *existing.RequireRole != *new.RequireRole {
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
func (h *Handler) ListExecutions(c *fiber.Ctx) error {
	ctx := c.Context()

	opts := ListExecutionsOptions{
		Namespace:     c.Query("namespace"),
		ProcedureName: c.Query("procedure"),
		UserID:        c.Query("user_id"),
		Limit:         100,
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
func (h *Handler) GetExecution(c *fiber.Ctx) error {
	ctx := c.Context()
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
func (h *Handler) GetExecutionLogs(c *fiber.Ctx) error {
	ctx := c.Context()
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

	// Get logs
	afterLine := 0
	if after := c.Query("after"); after != "" {
		if a, err := strconv.Atoi(after); err == nil {
			afterLine = a
		}
	}

	var logs []*ExecutionLog
	if afterLine > 0 {
		logs, err = h.storage.GetExecutionLogsSince(ctx, id, afterLine)
	} else {
		logs, err = h.storage.GetExecutionLogs(ctx, id)
	}

	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to get execution logs")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get execution logs",
		})
	}

	return c.JSON(fiber.Map{
		"logs":  logs,
		"count": len(logs),
	})
}

// ============================================================================
// PUBLIC: PROCEDURE LISTING
// ============================================================================

// ListPublicProcedures returns public, enabled procedures
// GET /api/v1/rpc/procedures
func (h *Handler) ListPublicProcedures(c *fiber.Ctx) error {
	ctx := c.Context()
	namespace := c.Query("namespace")

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
func (h *Handler) Invoke(c *fiber.Ctx) error {
	ctx := c.Context()
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

	if uid, ok := c.Locals("user_id").(string); ok && uid != "" {
		userID = uid
		isAuthenticated = true
	}
	if role, ok := c.Locals("role").(string); ok && role != "" {
		userRole = role
	}
	if email, ok := c.Locals("email").(string); ok {
		userEmail = email
	}
	if tc, ok := c.Locals("claims").(*auth.TokenClaims); ok {
		claims = tc
	}

	// Validate access
	if err := h.validator.ValidateAccess(procedure, userRole, isAuthenticated); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Parse request body
	var req InvokeRequest
	if err := c.BodyParser(&req); err != nil {
		// Body is optional
		req = InvokeRequest{}
	}

	// Build execution context
	execCtx := &ExecuteContext{
		Procedure: procedure,
		Params:    req.Params,
		UserID:    userID,
		UserRole:  userRole,
		UserEmail: userEmail,
		Claims:    claims,
		IsAsync:   req.Async,
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
func (h *Handler) GetPublicExecution(c *fiber.Ctx) error {
	ctx := c.Context()
	id := c.Params("id")

	// Get user context
	userID := ""
	if uid, ok := c.Locals("user_id").(string); ok {
		userID = uid
	}

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
	role, _ := c.Locals("role").(string)
	if role != "service_role" && role != "dashboard_admin" {
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
func (h *Handler) GetPublicExecutionLogs(c *fiber.Ctx) error {
	ctx := c.Context()
	id := c.Params("id")

	// Get user context
	userID := ""
	if uid, ok := c.Locals("user_id").(string); ok {
		userID = uid
	}

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

	// Check ownership
	role, _ := c.Locals("role").(string)
	if role != "service_role" && role != "dashboard_admin" {
		if execution.UserID == nil || *execution.UserID != userID {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Execution not found",
			})
		}
	}

	// Get logs
	afterLine := 0
	if after := c.Query("after"); after != "" {
		if a, err := strconv.Atoi(after); err == nil {
			afterLine = a
		}
	}

	var logs []*ExecutionLog
	if afterLine > 0 {
		logs, err = h.storage.GetExecutionLogsSince(ctx, id, afterLine)
	} else {
		logs, err = h.storage.GetExecutionLogs(ctx, id)
	}

	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to get execution logs")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get execution logs",
		})
	}

	return c.JSON(fiber.Map{
		"logs":  logs,
		"count": len(logs),
	})
}
