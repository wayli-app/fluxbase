package functions

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Handler manages HTTP endpoints for edge functions
type Handler struct {
	storage   *Storage
	runtime   *DenoRuntime
	scheduler *Scheduler
}

// NewHandler creates a new edge functions handler
func NewHandler(db *pgxpool.Pool) *Handler {
	return &Handler{
		storage: NewStorage(db),
		runtime: NewDenoRuntime(),
	}
}

// SetScheduler sets the scheduler for this handler
func (h *Handler) SetScheduler(scheduler *Scheduler) {
	h.scheduler = scheduler
}

// RegisterRoutes registers all edge function routes
func (h *Handler) RegisterRoutes(app *fiber.App) {
	// Management endpoints
	app.Post("/api/v1/functions", h.CreateFunction)
	app.Get("/api/v1/functions", h.ListFunctions)
	app.Get("/api/v1/functions/:name", h.GetFunction)
	app.Put("/api/v1/functions/:name", h.UpdateFunction)
	app.Delete("/api/v1/functions/:name", h.DeleteFunction)

	// Invocation endpoint
	app.Post("/api/v1/functions/:name/invoke", h.InvokeFunction)

	// Execution history
	app.Get("/api/v1/functions/:name/executions", h.GetExecutions)
}

// CreateFunction creates a new edge function
func (h *Handler) CreateFunction(c *fiber.Ctx) error {
	var req struct {
		Name           string  `json:"name"`
		Description    *string `json:"description"`
		Code           string  `json:"code"`
		Enabled        *bool   `json:"enabled"`
		TimeoutSeconds *int    `json:"timeout_seconds"`
		MemoryLimitMB  *int    `json:"memory_limit_mb"`
		AllowNet       *bool   `json:"allow_net"`
		AllowEnv       *bool   `json:"allow_env"`
		AllowRead      *bool   `json:"allow_read"`
		AllowWrite     *bool   `json:"allow_write"`
		CronSchedule   *string `json:"cron_schedule"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Validation
	if req.Name == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Function name is required"})
	}
	if req.Code == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Function code is required"})
	}

	// Get user ID from context (if authenticated)
	var createdBy *uuid.UUID
	if userID := c.Locals("user_id"); userID != nil {
		if uid, ok := userID.(string); ok {
			parsed, err := uuid.Parse(uid)
			if err == nil {
				createdBy = &parsed
			}
		}
	}

	// Create function
	fn := &EdgeFunction{
		Name:           req.Name,
		Description:    req.Description,
		Code:           req.Code,
		Enabled:        req.Enabled != nil && *req.Enabled,
		TimeoutSeconds: valueOr(req.TimeoutSeconds, 30),
		MemoryLimitMB:  valueOr(req.MemoryLimitMB, 128),
		AllowNet:       valueOr(req.AllowNet, true),
		AllowEnv:       valueOr(req.AllowEnv, true),
		AllowRead:      valueOr(req.AllowRead, false),
		AllowWrite:     valueOr(req.AllowWrite, false),
		CronSchedule:   req.CronSchedule,
		CreatedBy:      createdBy,
	}

	if err := h.storage.CreateFunction(c.Context(), fn); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(fn)
}

// ListFunctions lists all edge functions
func (h *Handler) ListFunctions(c *fiber.Ctx) error {
	functions, err := h.storage.ListFunctions(c.Context())
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(functions)
}

// GetFunction gets a single function by name
func (h *Handler) GetFunction(c *fiber.Ctx) error {
	name := c.Params("name")

	fn, err := h.storage.GetFunction(c.Context(), name)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Function not found"})
	}

	return c.JSON(fn)
}

// UpdateFunction updates an existing function
func (h *Handler) UpdateFunction(c *fiber.Ctx) error {
	name := c.Params("name")

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Remove fields that shouldn't be updated directly
	delete(updates, "id")
	delete(updates, "created_at")
	delete(updates, "updated_at")
	delete(updates, "version")

	if err := h.storage.UpdateFunction(c.Context(), name, updates); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Return updated function
	fn, err := h.storage.GetFunction(c.Context(), name)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fn)
}

// DeleteFunction deletes a function
func (h *Handler) DeleteFunction(c *fiber.Ctx) error {
	name := c.Params("name")

	if err := h.storage.DeleteFunction(c.Context(), name); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(204)
}

// InvokeFunction invokes an edge function
func (h *Handler) InvokeFunction(c *fiber.Ctx) error {
	name := c.Params("name")

	// Get function
	fn, err := h.storage.GetFunction(c.Context(), name)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Function not found"})
	}

	// Check if enabled
	if !fn.Enabled {
		return c.Status(403).JSON(fiber.Map{"error": "Function is disabled"})
	}

	// Build execution request
	req := ExecutionRequest{
		Method:  c.Method(),
		URL:     c.OriginalURL(),
		Headers: make(map[string]string),
		Body:    string(c.Body()),
	}

	// Copy headers
	c.Request().Header.VisitAll(func(key, value []byte) {
		req.Headers[string(key)] = string(value)
	})

	// Get user ID if authenticated
	if userID := c.Locals("user_id"); userID != nil {
		if uid, ok := userID.(string); ok {
			req.UserID = uid
		}
	}

	// Build permissions
	perms := Permissions{
		AllowNet:   fn.AllowNet,
		AllowEnv:   fn.AllowEnv,
		AllowRead:  fn.AllowRead,
		AllowWrite: fn.AllowWrite,
	}

	// Execute function
	result, err := h.runtime.Execute(c.Context(), fn.Code, req, perms)

	// Log execution
	now := time.Now()
	durationMs := int(result.DurationMs)
	exec := &EdgeFunctionExecution{
		FunctionID:  fn.ID,
		TriggerType: "http",
		Status:      "success",
		StatusCode:  &result.Status,
		DurationMs:  &durationMs,
		Logs:        &result.Logs,
		CompletedAt: &now,
	}

	if err != nil {
		exec.Status = "error"
		exec.ErrorMessage = &result.Error
	}

	if result.Body != "" {
		exec.Result = &result.Body
	}

	// Log asynchronously (don't block response)
	// Use background context since the Fiber context will be released
	go func() {
		ctx := context.Background()
		_ = h.storage.LogExecution(ctx, exec)
	}()

	// Return function result
	if err != nil {
		return c.Status(result.Status).JSON(fiber.Map{
			"error": result.Error,
			"logs":  result.Logs,
		})
	}

	// Set response headers
	for key, value := range result.Headers {
		c.Set(key, value)
	}

	// Return response
	return c.Status(result.Status).SendString(result.Body)
}

// GetExecutions returns execution history
func (h *Handler) GetExecutions(c *fiber.Ctx) error {
	name := c.Params("name")
	limit := c.QueryInt("limit", 50)

	if limit > 100 {
		limit = 100
	}

	executions, err := h.storage.GetExecutions(c.Context(), name, limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(executions)
}

// Helper functions

func valueOr[T any](ptr *T, defaultVal T) T {
	if ptr != nil {
		return *ptr
	}
	return defaultVal
}
