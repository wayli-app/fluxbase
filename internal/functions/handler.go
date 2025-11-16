package functions

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wayli-app/fluxbase/internal/auth"
	"github.com/wayli-app/fluxbase/internal/middleware"
)

// Handler manages HTTP endpoints for edge functions
type Handler struct {
	storage      *Storage
	runtime      *DenoRuntime
	scheduler    *Scheduler
	functionsDir string
}

// NewHandler creates a new edge functions handler
func NewHandler(db *pgxpool.Pool, functionsDir string) *Handler {
	return &Handler{
		storage:      NewStorage(db),
		runtime:      NewDenoRuntime(),
		functionsDir: functionsDir,
	}
}

// SetScheduler sets the scheduler for this handler
func (h *Handler) SetScheduler(scheduler *Scheduler) {
	h.scheduler = scheduler
}

// RegisterRoutes registers all edge function routes with authentication
func (h *Handler) RegisterRoutes(app *fiber.App, authService *auth.Service, apiKeyService *auth.APIKeyService, db *pgxpool.Pool, jwtManager *auth.JWTManager) {
	// Apply authentication middleware to management endpoints
	authMiddleware := middleware.RequireAuthOrServiceKey(authService, apiKeyService, db, jwtManager)

	functions := app.Group("/api/v1/functions")

	// Management endpoints - require authentication
	functions.Post("/", authMiddleware, h.CreateFunction)
	functions.Get("/", authMiddleware, h.ListFunctions)
	functions.Get("/:name", authMiddleware, h.GetFunction)
	functions.Put("/:name", authMiddleware, h.UpdateFunction)
	functions.Delete("/:name", authMiddleware, h.DeleteFunction)

	// Invocation endpoint - auth checked per-function in handler based on allow_unauthenticated
	// We use OptionalAuthMiddleware so auth context is set if token provided,
	// but the handler will check the function's allow_unauthenticated setting
	optionalAuth := middleware.OptionalAPIKeyAuth(authService, apiKeyService)
	functions.Post("/:name/invoke", optionalAuth, h.InvokeFunction)

	// Execution history - require authentication
	functions.Get("/:name/executions", authMiddleware, h.GetExecutions)

	// Admin reload endpoint - handled separately in server.go under admin routes
}

// CreateFunction creates a new edge function
func (h *Handler) CreateFunction(c *fiber.Ctx) error {
	var req struct {
		Name                 string  `json:"name"`
		Description          *string `json:"description"`
		Code                 string  `json:"code"`
		Enabled              *bool   `json:"enabled"`
		TimeoutSeconds       *int    `json:"timeout_seconds"`
		MemoryLimitMB        *int    `json:"memory_limit_mb"`
		AllowNet             *bool   `json:"allow_net"`
		AllowEnv             *bool   `json:"allow_env"`
		AllowRead            *bool   `json:"allow_read"`
		AllowWrite           *bool   `json:"allow_write"`
		AllowUnauthenticated *bool   `json:"allow_unauthenticated"`
		CronSchedule         *string `json:"cron_schedule"`
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

	// Parse configuration from code comments (if not explicitly set in request)
	var allowUnauthenticated bool
	if req.AllowUnauthenticated != nil {
		// Explicit setting takes precedence
		allowUnauthenticated = *req.AllowUnauthenticated
	} else {
		// Parse from code comments
		config := ParseFunctionConfig(req.Code)
		allowUnauthenticated = config.AllowUnauthenticated
	}

	// Bundle function code if it has imports
	bundler, err := NewBundler()
	bundledCode := req.Code
	originalCode := &req.Code
	isBundled := false
	var bundleError *string

	if err == nil {
		// Bundler available - attempt to bundle
		result, bundleErr := bundler.Bundle(c.Context(), req.Code)
		if bundleErr != nil {
			// Bundling failed - return error to user
			errMsg := fmt.Sprintf("Failed to bundle function: %v", bundleErr)
			return c.Status(400).JSON(fiber.Map{
				"error":   "Bundle error",
				"details": errMsg,
			})
		}

		// Bundling succeeded
		bundledCode = result.BundledCode
		isBundled = result.IsBundled
		if result.Error != "" {
			bundleError = &result.Error
		}
	}
	// If bundler not available (Deno not installed), use unbundled code

	// Create function
	fn := &EdgeFunction{
		Name:                 req.Name,
		Description:          req.Description,
		Code:                 bundledCode,
		OriginalCode:         originalCode,
		IsBundled:            isBundled,
		BundleError:          bundleError,
		Enabled:              req.Enabled != nil && *req.Enabled,
		TimeoutSeconds:       valueOr(req.TimeoutSeconds, 30),
		MemoryLimitMB:        valueOr(req.MemoryLimitMB, 128),
		AllowNet:             valueOr(req.AllowNet, true),
		AllowEnv:             valueOr(req.AllowEnv, true),
		AllowRead:            valueOr(req.AllowRead, false),
		AllowWrite:           valueOr(req.AllowWrite, false),
		AllowUnauthenticated: allowUnauthenticated,
		CronSchedule:         req.CronSchedule,
		CreatedBy:            createdBy,
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

	// Check authentication requirement
	// If function doesn't allow unauthenticated access, require at minimum an anon key
	// Functions can explicitly set allow_unauthenticated=true to bypass this check
	if !fn.AllowUnauthenticated {
		authType := c.Locals("auth_type")
		if authType == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authentication required. Provide an anon key (Bearer token with role=anon), API key (X-API-Key header), or service key (X-Service-Key header). " +
					"To allow completely unauthenticated access, set allow_unauthenticated=true on the function.",
			})
		}
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

// RegisterAdminRoutes registers admin-only routes for functions management
// These routes should be called with UnifiedAuthMiddleware and RequireRole("admin", "dashboard_admin")
func (h *Handler) RegisterAdminRoutes(app *fiber.App) {
	// Admin-only function reload endpoint
	app.Post("/api/v1/admin/functions/reload", h.ReloadFunctions)
}

// ReloadFunctions scans the functions directory and syncs with database
// Admin-only endpoint - requires authentication and admin role
func (h *Handler) ReloadFunctions(c *fiber.Ctx) error {
	ctx := c.Context()

	// Scan functions directory for all .ts files
	functionFiles, err := ListFunctionFiles(h.functionsDir)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to scan functions directory",
		})
	}

	// Get all existing functions from database
	allFunctions, err := h.storage.ListFunctions(ctx)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to list existing functions",
		})
	}

	// Build set of function names on disk
	diskFunctionNames := make(map[string]bool)
	for _, fileInfo := range functionFiles {
		diskFunctionNames[fileInfo.Name] = true
	}

	// Track results
	var created []string
	var updated []string
	var deleted []string
	var errors []string

	// Process each function file
	for _, fileInfo := range functionFiles {
		// Check if function exists in database
		existingFn, err := h.storage.GetFunction(ctx, fileInfo.Name)

		if err != nil {
			// Function doesn't exist in database - create it
			code, err := LoadFunctionCode(h.functionsDir, fileInfo.Name)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s: failed to load code: %v", fileInfo.Name, err))
				continue
			}

			// Parse configuration from code comments
			config := ParseFunctionConfig(code)

			// Bundle function code if it has imports
			bundler, bundlerErr := NewBundler()
			bundledCode := code
			originalCode := &code
			isBundled := false
			var bundleError *string

			if bundlerErr == nil {
				// Bundler available - attempt to bundle
				result, bundleErr := bundler.Bundle(ctx, code)
				if bundleErr != nil {
					// Bundling failed - log error but continue with unbundled code
					errMsg := fmt.Sprintf("bundle failed: %v", bundleErr)
					bundleError = &errMsg
				} else {
					// Bundling succeeded
					bundledCode = result.BundledCode
					isBundled = result.IsBundled
					if result.Error != "" {
						bundleError = &result.Error
					}
				}
			}

			// Create new function with default settings
			fn := &EdgeFunction{
				Name:                 fileInfo.Name,
				Code:                 bundledCode,
				OriginalCode:         originalCode,
				IsBundled:            isBundled,
				BundleError:          bundleError,
				Enabled:              true,
				TimeoutSeconds:       30,
				MemoryLimitMB:        128,
				AllowNet:             true,
				AllowEnv:             true,
				AllowRead:            false,
				AllowWrite:           false,
				AllowUnauthenticated: config.AllowUnauthenticated,
			}

			if err := h.storage.CreateFunction(ctx, fn); err != nil {
				errors = append(errors, fmt.Sprintf("%s: failed to create: %v", fileInfo.Name, err))
				continue
			}

			created = append(created, fileInfo.Name)
		} else {
			// Function exists - update code from filesystem
			code, err := LoadFunctionCode(h.functionsDir, fileInfo.Name)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s: failed to load code: %v", fileInfo.Name, err))
				continue
			}

			// Parse configuration from code comments
			config := ParseFunctionConfig(code)

			// Bundle function code if it has imports
			bundler, bundlerErr := NewBundler()
			bundledCode := code
			originalCode := code
			isBundled := false
			var bundleError *string

			if bundlerErr == nil {
				// Bundler available - attempt to bundle
				result, bundleErr := bundler.Bundle(ctx, code)
				if bundleErr != nil {
					// Bundling failed - log error but continue with unbundled code
					errMsg := fmt.Sprintf("bundle failed: %v", bundleErr)
					bundleError = &errMsg
				} else {
					// Bundling succeeded
					bundledCode = result.BundledCode
					isBundled = result.IsBundled
					if result.Error != "" {
						bundleError = &result.Error
					}
				}
			}

			// Update if code or config has changed
			// Compare with original_code if available, otherwise with code
			compareCode := code
			if existingFn.OriginalCode != nil {
				compareCode = *existingFn.OriginalCode
			}

			if existingFn.Code != bundledCode || compareCode != originalCode || existingFn.AllowUnauthenticated != config.AllowUnauthenticated {
				updates := map[string]interface{}{
					"code":                  bundledCode,
					"original_code":         originalCode,
					"is_bundled":            isBundled,
					"bundle_error":          bundleError,
					"allow_unauthenticated": config.AllowUnauthenticated,
				}

				if err := h.storage.UpdateFunction(ctx, fileInfo.Name, updates); err != nil {
					errors = append(errors, fmt.Sprintf("%s: failed to update: %v", fileInfo.Name, err))
					continue
				}

				updated = append(updated, fileInfo.Name)
			}
		}
	}

	// Delete functions that exist in database but not on disk
	for _, dbFunc := range allFunctions {
		if !diskFunctionNames[dbFunc.Name] {
			// Function exists in DB but not on disk - delete it
			if err := h.storage.DeleteFunction(ctx, dbFunc.Name); err != nil {
				errors = append(errors, fmt.Sprintf("%s: failed to delete: %v", dbFunc.Name, err))
				continue
			}
			deleted = append(deleted, dbFunc.Name)
		}
	}

	return c.JSON(fiber.Map{
		"message": "Functions reloaded from filesystem",
		"created": created,
		"updated": updated,
		"deleted": deleted,
		"errors":  errors,
		"total":   len(functionFiles),
	})
}

// LoadFromFilesystem loads functions from filesystem at boot time
// This is called from main.go if auto_load_on_boot is enabled
func (h *Handler) LoadFromFilesystem(ctx context.Context) error {
	// Scan functions directory for all .ts files
	functionFiles, err := ListFunctionFiles(h.functionsDir)
	if err != nil {
		return fmt.Errorf("failed to scan functions directory: %w", err)
	}

	// Track results
	var created []string
	var updated []string
	var errors []string

	// Process each function file
	for _, fileInfo := range functionFiles {
		// Check if function exists in database
		existingFn, err := h.storage.GetFunction(ctx, fileInfo.Name)

		if err != nil {
			// Function doesn't exist in database - create it
			code, err := LoadFunctionCode(h.functionsDir, fileInfo.Name)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s: failed to load code: %v", fileInfo.Name, err))
				continue
			}

			// Parse configuration from code comments
			config := ParseFunctionConfig(code)

			// Bundle function code if it has imports
			bundler, bundlerErr := NewBundler()
			bundledCode := code
			originalCode := &code
			isBundled := false
			var bundleError *string

			if bundlerErr == nil {
				// Bundler available - attempt to bundle
				result, bundleErr := bundler.Bundle(ctx, code)
				if bundleErr != nil {
					// Bundling failed - log error but continue with unbundled code
					errMsg := fmt.Sprintf("bundle failed: %v", bundleErr)
					bundleError = &errMsg
				} else {
					// Bundling succeeded
					bundledCode = result.BundledCode
					isBundled = result.IsBundled
					if result.Error != "" {
						bundleError = &result.Error
					}
				}
			}

			// Create new function with default settings
			fn := &EdgeFunction{
				Name:                 fileInfo.Name,
				Code:                 bundledCode,
				OriginalCode:         originalCode,
				IsBundled:            isBundled,
				BundleError:          bundleError,
				Enabled:              true,
				TimeoutSeconds:       30,
				MemoryLimitMB:        128,
				AllowNet:             true,
				AllowEnv:             true,
				AllowRead:            false,
				AllowWrite:           false,
				AllowUnauthenticated: config.AllowUnauthenticated,
			}

			if err := h.storage.CreateFunction(ctx, fn); err != nil {
				errors = append(errors, fmt.Sprintf("%s: failed to create: %v", fileInfo.Name, err))
				continue
			}

			created = append(created, fileInfo.Name)
		} else {
			// Function exists - update code from filesystem
			code, err := LoadFunctionCode(h.functionsDir, fileInfo.Name)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s: failed to load code: %v", fileInfo.Name, err))
				continue
			}

			// Parse configuration from code comments
			config := ParseFunctionConfig(code)

			// Bundle function code if it has imports
			bundler, bundlerErr := NewBundler()
			bundledCode := code
			originalCode := code
			isBundled := false
			var bundleError *string

			if bundlerErr == nil {
				// Bundler available - attempt to bundle
				result, bundleErr := bundler.Bundle(ctx, code)
				if bundleErr != nil {
					// Bundling failed - log error but continue with unbundled code
					errMsg := fmt.Sprintf("bundle failed: %v", bundleErr)
					bundleError = &errMsg
				} else {
					// Bundling succeeded
					bundledCode = result.BundledCode
					isBundled = result.IsBundled
					if result.Error != "" {
						bundleError = &result.Error
					}
				}
			}

			// Update if code or config has changed
			// Compare with original_code if available, otherwise with code
			compareCode := code
			if existingFn.OriginalCode != nil {
				compareCode = *existingFn.OriginalCode
			}

			if existingFn.Code != bundledCode || compareCode != originalCode || existingFn.AllowUnauthenticated != config.AllowUnauthenticated {
				updates := map[string]interface{}{
					"code":                  bundledCode,
					"original_code":         originalCode,
					"is_bundled":            isBundled,
					"bundle_error":          bundleError,
					"allow_unauthenticated": config.AllowUnauthenticated,
				}

				if err := h.storage.UpdateFunction(ctx, fileInfo.Name, updates); err != nil {
					errors = append(errors, fmt.Sprintf("%s: failed to update: %v", fileInfo.Name, err))
					continue
				}

				updated = append(updated, fileInfo.Name)
			}
		}
	}

	// Note: Auto-load does NOT delete functions missing from filesystem
	// This prevents data loss when UI-created functions exist alongside file-based functions
	// Use the manual reload endpoint to perform full sync including deletions

	// Log results
	if len(created) > 0 || len(updated) > 0 {
		fmt.Printf("Functions loaded from filesystem: %d created, %d updated\n", len(created), len(updated))
	}
	if len(errors) > 0 {
		fmt.Printf("Errors loading functions: %v\n", errors)
	}

	return nil
}

// Helper functions

func valueOr[T any](ptr *T, defaultVal T) T {
	if ptr != nil {
		return *ptr
	}
	return defaultVal
}
