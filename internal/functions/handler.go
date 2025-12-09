package functions

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/fluxbase-eu/fluxbase/internal/middleware"
	"github.com/fluxbase-eu/fluxbase/internal/runtime"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// Handler manages HTTP endpoints for edge functions
type Handler struct {
	storage      *Storage
	runtime      *runtime.DenoRuntime
	scheduler    *Scheduler
	functionsDir string
	corsConfig   config.CORSConfig
	publicURL    string
	logCounters  sync.Map // map[uuid.UUID]*int for tracking log line numbers per execution
}

// NewHandler creates a new edge functions handler
func NewHandler(db *database.Connection, functionsDir string, corsConfig config.CORSConfig, jwtSecret, publicURL string) *Handler {
	h := &Handler{
		storage:      NewStorage(db),
		runtime:      runtime.NewRuntime(runtime.RuntimeTypeFunction, jwtSecret, publicURL),
		functionsDir: functionsDir,
		corsConfig:   corsConfig,
		publicURL:    publicURL,
	}

	// Set up log callback to capture console.log output
	h.runtime.SetLogCallback(h.handleLogMessage)

	return h
}

// SetScheduler sets the scheduler for this handler
func (h *Handler) SetScheduler(scheduler *Scheduler) {
	h.scheduler = scheduler
}

// handleLogMessage is called when a function outputs a log message via console.log/console.error
func (h *Handler) handleLogMessage(executionID uuid.UUID, level string, message string) {
	// Get and increment the line counter for this execution
	counterVal, ok := h.logCounters.Load(executionID)
	if !ok {
		// No counter means execution wasn't set up for logging (e.g., old code path)
		log.Debug().
			Str("execution_id", executionID.String()).
			Str("level", level).
			Str("message", message).
			Msg("Function log (no counter)")
		return
	}

	counterPtr, ok := counterVal.(*int)
	if !ok {
		log.Warn().Str("execution_id", executionID.String()).Msg("Invalid log counter type")
		return
	}

	lineNumber := *counterPtr
	*counterPtr = lineNumber + 1

	// Insert log line into database
	ctx := context.Background()
	if err := h.storage.AppendExecutionLog(ctx, executionID, lineNumber, level, message); err != nil {
		log.Error().Err(err).Str("execution_id", executionID.String()).Msg("Failed to insert execution log")
	}
}

// applyCorsHeaders applies CORS headers to the response with fallback to global config
func (h *Handler) applyCorsHeaders(c *fiber.Ctx, fn *EdgeFunction) {
	// Determine CORS values with fallback: function settings > global config
	origins := h.corsConfig.AllowedOrigins
	if fn.CorsOrigins != nil {
		origins = *fn.CorsOrigins
	}

	methods := h.corsConfig.AllowedMethods
	if fn.CorsMethods != nil {
		methods = *fn.CorsMethods
	}

	headers := h.corsConfig.AllowedHeaders
	if fn.CorsHeaders != nil {
		headers = *fn.CorsHeaders
	}

	credentials := h.corsConfig.AllowCredentials
	if fn.CorsCredentials != nil {
		credentials = *fn.CorsCredentials
	}

	maxAge := h.corsConfig.MaxAge
	if fn.CorsMaxAge != nil {
		maxAge = *fn.CorsMaxAge
	}

	// Apply CORS headers
	// Handle Access-Control-Allow-Origin properly:
	// - If origins is "*", use "*"
	// - If origins contains multiple comma-separated values, check if request origin matches
	// - Browsers only accept a single origin or "*", not comma-separated lists
	allowedOrigin := origins
	if origins != "*" && strings.Contains(origins, ",") {
		requestOrigin := c.Get("Origin")
		if requestOrigin != "" {
			// Check if request origin is in the allowed list
			for _, allowed := range strings.Split(origins, ",") {
				if strings.TrimSpace(allowed) == requestOrigin {
					allowedOrigin = requestOrigin
					break
				}
			}
		}
	}
	c.Set("Access-Control-Allow-Origin", allowedOrigin)
	c.Set("Access-Control-Allow-Methods", methods)
	c.Set("Access-Control-Allow-Headers", headers)

	if credentials && allowedOrigin != "*" {
		c.Set("Access-Control-Allow-Credentials", "true")
	}

	if maxAge > 0 {
		c.Set("Access-Control-Max-Age", strconv.Itoa(maxAge))
	}

	// Expose headers if configured
	if h.corsConfig.ExposedHeaders != "" {
		c.Set("Access-Control-Expose-Headers", h.corsConfig.ExposedHeaders)
	}
}

// RegisterRoutes registers all edge function routes with authentication
func (h *Handler) RegisterRoutes(app *fiber.App, authService *auth.Service, apiKeyService *auth.APIKeyService, db *pgxpool.Pool, jwtManager *auth.JWTManager) {
	// Apply authentication middleware to management endpoints
	authMiddleware := middleware.RequireAuthOrServiceKey(authService, apiKeyService, db, jwtManager)

	// Apply feature flag middleware to all functions routes
	functions := app.Group("/api/v1/functions",
		middleware.RequireFunctionsEnabled(authService.GetSettingsCache()),
	)

	// Management endpoints - require authentication
	functions.Post("/", authMiddleware, h.CreateFunction)
	functions.Get("/", authMiddleware, h.ListFunctions)
	functions.Get("/:name", authMiddleware, h.GetFunction)
	functions.Put("/:name", authMiddleware, h.UpdateFunction)
	functions.Delete("/:name", authMiddleware, h.DeleteFunction)

	// Invocation endpoint - auth checked per-function in handler based on allow_unauthenticated
	// We use OptionalAuthOrServiceKey so auth context is set if token provided (including anon key JWTs),
	// but the handler will check the function's allow_unauthenticated setting
	optionalAuth := middleware.OptionalAuthOrServiceKey(authService, apiKeyService, db, jwtManager)
	functions.Post("/:name/invoke", optionalAuth, h.InvokeFunction)
	functions.Get("/:name/invoke", optionalAuth, h.InvokeFunction) // Also support GET for health checks

	// Execution history - require authentication
	functions.Get("/:name/executions", authMiddleware, h.GetExecutions)

	// Shared modules endpoints - require authentication
	shared := app.Group("/api/v1/functions/shared")
	shared.Post("/", authMiddleware, h.CreateSharedModule)
	shared.Get("/", authMiddleware, h.ListSharedModules)
	shared.Get("/*", authMiddleware, h.GetSharedModule) // Use /* to capture full path
	shared.Put("/*", authMiddleware, h.UpdateSharedModule)
	shared.Delete("/*", authMiddleware, h.DeleteSharedModule)

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
		IsPublic             *bool   `json:"is_public"`
		CorsOrigins          *string `json:"cors_origins"`
		CorsMethods          *string `json:"cors_methods"`
		CorsHeaders          *string `json:"cors_headers"`
		CorsCredentials      *bool   `json:"cors_credentials"`
		CorsMaxAge           *int    `json:"cors_max_age"`
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
	config := ParseFunctionConfig(req.Code)

	var allowUnauthenticated bool
	if req.AllowUnauthenticated != nil {
		// Explicit setting takes precedence
		allowUnauthenticated = *req.AllowUnauthenticated
	} else {
		// Parse from code comments
		allowUnauthenticated = config.AllowUnauthenticated
	}

	var isPublic bool
	if req.IsPublic != nil {
		// Explicit setting takes precedence
		isPublic = *req.IsPublic
	} else {
		// Parse from code comments (defaults to true)
		isPublic = config.IsPublic
	}

	// Apply CORS config with priority: API request > annotations > nil (use global defaults)
	var corsOrigins *string
	if req.CorsOrigins != nil {
		corsOrigins = req.CorsOrigins
	} else {
		corsOrigins = config.CorsOrigins
	}

	var corsMethods *string
	if req.CorsMethods != nil {
		corsMethods = req.CorsMethods
	} else {
		corsMethods = config.CorsMethods
	}

	var corsHeaders *string
	if req.CorsHeaders != nil {
		corsHeaders = req.CorsHeaders
	} else {
		corsHeaders = config.CorsHeaders
	}

	var corsCredentials *bool
	if req.CorsCredentials != nil {
		corsCredentials = req.CorsCredentials
	} else {
		corsCredentials = config.CorsCredentials
	}

	var corsMaxAge *int
	if req.CorsMaxAge != nil {
		corsMaxAge = req.CorsMaxAge
	} else {
		corsMaxAge = config.CorsMaxAge
	}

	// Bundle function code if it has imports
	bundler, err := NewBundler()
	bundledCode := req.Code
	originalCode := &req.Code
	isBundled := false
	var bundleError *string

	if err == nil {
		// Check if code imports from _shared/ modules
		hasSharedImports := strings.Contains(req.Code, "from \"_shared/") ||
			strings.Contains(req.Code, "from '_shared/")

		var result *BundleResult
		var bundleErr error

		if hasSharedImports {
			// Load all shared modules from database
			sharedModules, err := h.storage.ListSharedModules(c.Context())
			if err != nil {
				log.Warn().Err(err).Msg("Failed to load shared modules, proceeding with regular bundle")
				result, bundleErr = bundler.Bundle(c.Context(), req.Code)
			} else {
				// Build map of shared module paths to content
				sharedModulesMap := make(map[string]string)
				for _, module := range sharedModules {
					sharedModulesMap[module.ModulePath] = module.Content
				}

				// Bundle with shared modules (no supporting files for now)
				supportingFiles := make(map[string]string)
				result, bundleErr = bundler.BundleWithFiles(c.Context(), req.Code, supportingFiles, sharedModulesMap)
			}
		} else {
			// No shared imports - use regular bundling
			result, bundleErr = bundler.Bundle(c.Context(), req.Code)
		}

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
		IsPublic:             isPublic,
		CorsOrigins:          corsOrigins,
		CorsMethods:          corsMethods,
		CorsHeaders:          corsHeaders,
		CorsCredentials:      corsCredentials,
		CorsMaxAge:           corsMaxAge,
		CronSchedule:         req.CronSchedule,
		CreatedBy:            createdBy,
		Source:               "api",
	}

	if err := h.storage.CreateFunction(c.Context(), fn); err != nil {
		reqID := getRequestID(c)
		log.Error().
			Err(err).
			Str("function_name", fn.Name).
			Str("request_id", reqID).
			Str("user_id", toString(createdBy)).
			Msg("Failed to create edge function in database")

		return c.Status(500).JSON(fiber.Map{
			"error":      "Failed to create function",
			"details":    err.Error(),
			"request_id": reqID,
		})
	}

	return c.Status(201).JSON(fn)
}

// ListFunctions lists all edge functions
func (h *Handler) ListFunctions(c *fiber.Ctx) error {
	// Check if namespace filter is provided
	namespace := c.Query("namespace")

	var functions []EdgeFunctionSummary
	var err error

	if namespace != "" {
		// If namespace is specified, list functions in that namespace
		functions, err = h.storage.ListFunctionsByNamespace(c.Context(), namespace)
	} else {
		// Otherwise, list all public functions
		functions, err = h.storage.ListFunctions(c.Context())
	}

	if err != nil {
		reqID := getRequestID(c)
		log.Error().
			Err(err).
			Str("request_id", reqID).
			Msg("Failed to list edge functions from database")

		return c.Status(500).JSON(fiber.Map{
			"error":      "Failed to list functions",
			"details":    err.Error(),
			"request_id": reqID,
		})
	}

	return c.JSON(functions)
}

// ListNamespaces lists all unique namespaces with edge functions
func (h *Handler) ListNamespaces(c *fiber.Ctx) error {
	namespaces, err := h.storage.ListFunctionNamespaces(c.Context())
	if err != nil {
		reqID := getRequestID(c)
		log.Error().
			Err(err).
			Str("request_id", reqID).
			Msg("Failed to list function namespaces")

		return c.Status(500).JSON(fiber.Map{
			"error":      "Failed to list function namespaces",
			"request_id": reqID,
		})
	}

	// Ensure we always return at least "default"
	if len(namespaces) == 0 {
		namespaces = []string{"default"}
	}

	return c.JSON(fiber.Map{"namespaces": namespaces})
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

	// If code is being updated, re-bundle with shared modules
	if codeUpdate, ok := updates["code"].(string); ok && codeUpdate != "" {
		bundler, err := NewBundler()
		if err == nil {
			// Check if code imports from _shared/ modules
			hasSharedImports := strings.Contains(codeUpdate, "from \"_shared/") ||
				strings.Contains(codeUpdate, "from '_shared/")

			var result *BundleResult
			var bundleErr error

			if hasSharedImports {
				// Load all shared modules from database
				sharedModules, err := h.storage.ListSharedModules(c.Context())
				if err != nil {
					log.Warn().Err(err).Msg("Failed to load shared modules for update, proceeding with regular bundle")
					result, bundleErr = bundler.Bundle(c.Context(), codeUpdate)
				} else {
					// Build map of shared module paths to content
					sharedModulesMap := make(map[string]string)
					for _, module := range sharedModules {
						sharedModulesMap[module.ModulePath] = module.Content
					}

					// Bundle with shared modules
					supportingFiles := make(map[string]string)
					result, bundleErr = bundler.BundleWithFiles(c.Context(), codeUpdate, supportingFiles, sharedModulesMap)
				}
			} else {
				// No shared imports - use regular bundling
				result, bundleErr = bundler.Bundle(c.Context(), codeUpdate)
			}

			if bundleErr != nil {
				// Bundling failed - return error to user
				errMsg := fmt.Sprintf("Failed to bundle function: %v", bundleErr)
				return c.Status(400).JSON(fiber.Map{
					"error":   "Bundle error",
					"details": errMsg,
				})
			}

			// Update with bundled code
			updates["code"] = result.BundledCode
			updates["original_code"] = codeUpdate
			updates["is_bundled"] = result.IsBundled
			if result.Error != "" {
				updates["bundle_error"] = result.Error
			} else {
				updates["bundle_error"] = nil
			}
		}
	}

	reqID := getRequestID(c)
	if err := h.storage.UpdateFunction(c.Context(), name, updates); err != nil {
		log.Error().
			Err(err).
			Str("function_name", name).
			Str("request_id", reqID).
			Msg("Failed to update edge function in database")

		return c.Status(500).JSON(fiber.Map{
			"error":      "Failed to update function",
			"details":    err.Error(),
			"request_id": reqID,
		})
	}

	// Return updated function
	fn, err := h.storage.GetFunction(c.Context(), name)
	if err != nil {
		log.Error().
			Err(err).
			Str("function_name", name).
			Str("request_id", reqID).
			Msg("Failed to retrieve updated edge function from database")

		return c.Status(500).JSON(fiber.Map{
			"error":      "Failed to retrieve updated function",
			"details":    err.Error(),
			"request_id": reqID,
		})
	}

	return c.JSON(fn)
}

// DeleteFunction deletes a function
func (h *Handler) DeleteFunction(c *fiber.Ctx) error {
	name := c.Params("name")

	if err := h.storage.DeleteFunction(c.Context(), name); err != nil {
		reqID := getRequestID(c)
		log.Error().
			Err(err).
			Str("function_name", name).
			Str("request_id", reqID).
			Msg("Failed to delete edge function from database")

		return c.Status(500).JSON(fiber.Map{
			"error":      "Failed to delete function",
			"details":    err.Error(),
			"request_id": reqID,
		})
	}

	return c.SendStatus(204)
}

// InvokeFunction invokes an edge function
func (h *Handler) InvokeFunction(c *fiber.Ctx) error {
	name := c.Params("name")
	namespace := c.Query("namespace")

	// Get function - if namespace is provided, look up by namespace+name; otherwise find first match by name
	var fn *EdgeFunction
	var err error
	if namespace != "" {
		fn, err = h.storage.GetFunctionByNamespace(c.Context(), name, namespace)
	} else {
		fn, err = h.storage.GetFunction(c.Context(), name)
	}
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Function not found"})
	}

	// Apply CORS headers to all responses (including errors)
	h.applyCorsHeaders(c, fn)

	// Handle CORS preflight (OPTIONS) requests automatically
	if c.Method() == "OPTIONS" {
		return c.SendStatus(204)
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

	// Generate execution ID for tracking
	executionID := uuid.New()

	// Build execution request with unified runtime types
	req := runtime.ExecutionRequest{
		ID:        executionID,
		Name:      fn.Name,
		Namespace: fn.Namespace,
		Method:    c.Method(),
		URL:       h.publicURL + c.OriginalURL(),
		BaseURL:   h.publicURL,
		Headers:   make(map[string]string),
		Body:      string(c.Body()),
		Params:    make(map[string]string),
	}

	// Copy headers
	c.Request().Header.VisitAll(func(key, value []byte) {
		req.Headers[string(key)] = string(value)
	})

	// Copy query parameters
	c.Request().URI().QueryArgs().VisitAll(func(key, value []byte) {
		req.Params[string(key)] = string(value)
	})

	// Get user context if authenticated
	if userID := c.Locals("user_id"); userID != nil {
		if uid, ok := userID.(string); ok {
			req.UserID = uid
		}
	}
	if userEmail := c.Locals("user_email"); userEmail != nil {
		if email, ok := userEmail.(string); ok {
			req.UserEmail = email
		}
	}
	if userRole := c.Locals("user_role"); userRole != nil {
		if role, ok := userRole.(string); ok {
			req.UserRole = role
		}
	}
	if sessionID := c.Locals("session_id"); sessionID != nil {
		if sid, ok := sessionID.(string); ok {
			req.SessionID = sid
		}
	}

	// Build permissions
	perms := runtime.Permissions{
		AllowNet:   fn.AllowNet,
		AllowEnv:   fn.AllowEnv,
		AllowRead:  fn.AllowRead,
		AllowWrite: fn.AllowWrite,
	}

	// Log function invocation
	reqID := getRequestID(c)
	log.Info().
		Str("function_name", name).
		Str("execution_id", executionID.String()).
		Str("user_id", req.UserID).
		Str("method", req.Method).
		Str("request_id", reqID).
		Msg("Invoking edge function")

	// Create execution record BEFORE running to enable real-time logging
	if err := h.storage.CreateExecution(c.Context(), executionID, fn.ID, "http"); err != nil {
		log.Error().Err(err).Str("execution_id", executionID.String()).Msg("Failed to create execution record")
		// Continue anyway - logging will still work via stderr fallback
	}

	// Initialize log counter for this execution
	lineCounter := 0
	h.logCounters.Store(executionID, &lineCounter)
	defer h.logCounters.Delete(executionID)

	// Execute function (nil cancel signal for basic invocation - streaming endpoint will use actual signal)
	result, err := h.runtime.Execute(c.Context(), fn.Code, req, perms, nil)

	// Complete execution record
	durationMs := int(result.DurationMs)
	status := "success"
	var errorMessage *string
	if err != nil {
		status = "error"
		errorMessage = &result.Error
	}

	var resultBody *string
	if result.Body != "" {
		resultBody = &result.Body
	}

	// Update execution record asynchronously (don't block response)
	go func() {
		ctx := context.Background()
		if updateErr := h.storage.CompleteExecution(ctx, executionID, status, &result.Status, &durationMs, resultBody, &result.Logs, errorMessage); updateErr != nil {
			log.Error().Err(updateErr).Str("execution_id", executionID.String()).Msg("Failed to complete execution record")
		}
	}()

	// Return function result
	if err != nil {
		// Log execution error with full context
		log.Error().
			Err(err).
			Str("function_name", name).
			Str("user_id", req.UserID).
			Str("request_id", reqID).
			Int("status", result.Status).
			Str("error_message", result.Error).
			Str("logs", result.Logs).
			Int64("duration_ms", result.DurationMs).
			Msg("Edge function execution failed")

		return c.Status(result.Status).JSON(fiber.Map{
			"error":      result.Error,
			"logs":       result.Logs,
			"request_id": reqID,
		})
	}

	// Log non-2xx responses even when execution succeeded
	if result.Status >= 400 {
		log.Warn().
			Str("function_name", name).
			Str("user_id", req.UserID).
			Str("request_id", reqID).
			Int("status", result.Status).
			Str("logs", result.Logs).
			Str("response_preview", truncateString(result.Body, 200)).
			Int64("duration_ms", result.DurationMs).
			Msg("Edge function returned error status")
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
		reqID := getRequestID(c)
		log.Error().
			Err(err).
			Str("function_name", name).
			Str("request_id", reqID).
			Int("limit", limit).
			Msg("Failed to retrieve edge function execution history from database")

		return c.Status(500).JSON(fiber.Map{
			"error":      "Failed to retrieve execution history",
			"details":    err.Error(),
			"request_id": reqID,
		})
	}

	return c.JSON(executions)
}

// ListAllExecutions returns execution history across all functions (admin only)
func (h *Handler) ListAllExecutions(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 25)
	offset := c.QueryInt("offset", 0)
	namespace := c.Query("namespace")
	functionName := c.Query("function_name")
	status := c.Query("status")

	if limit > 100 {
		limit = 100
	}

	filters := AdminExecutionFilters{
		Namespace:    namespace,
		FunctionName: functionName,
		Status:       status,
		Limit:        limit,
		Offset:       offset,
	}

	executions, total, err := h.storage.ListAllExecutions(c.Context(), filters)
	if err != nil {
		reqID := getRequestID(c)
		log.Error().
			Err(err).
			Str("request_id", reqID).
			Interface("filters", filters).
			Msg("Failed to list all edge function executions")

		return c.Status(500).JSON(fiber.Map{
			"error":      "Failed to list executions",
			"details":    err.Error(),
			"request_id": reqID,
		})
	}

	return c.JSON(fiber.Map{
		"executions": executions,
		"count":      total,
	})
}

// GetExecutionLogs returns logs for a specific function execution
func (h *Handler) GetExecutionLogs(c *fiber.Ctx) error {
	executionIDStr := c.Params("executionId")

	executionID, err := uuid.Parse(executionIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid execution ID",
		})
	}

	// Get optional after parameter for pagination/streaming
	afterLine := 0
	if after := c.Query("after"); after != "" {
		if a, err := strconv.Atoi(after); err == nil {
			afterLine = a
		}
	}

	var logs []ExecutionLog
	if afterLine > 0 {
		logs, err = h.storage.GetExecutionLogsSince(c.Context(), executionID, afterLine)
	} else {
		logs, err = h.storage.GetExecutionLogs(c.Context(), executionID)
	}

	if err != nil {
		reqID := getRequestID(c)
		log.Error().
			Err(err).
			Str("request_id", reqID).
			Str("execution_id", executionIDStr).
			Msg("Failed to get execution logs")

		return c.Status(500).JSON(fiber.Map{
			"error":      "Failed to get execution logs",
			"details":    err.Error(),
			"request_id": reqID,
		})
	}

	return c.JSON(fiber.Map{
		"logs":  logs,
		"count": len(logs),
	})
}

// RegisterAdminRoutes registers admin-only routes for functions management
// These routes should be called with UnifiedAuthMiddleware and RequireRole("admin", "dashboard_admin")
func (h *Handler) RegisterAdminRoutes(app *fiber.App) {
	// Admin-only function reload endpoint
	app.Post("/api/v1/admin/functions/reload", h.ReloadFunctions)
}

// bundleFunctionFromFilesystem loads function code with supporting files and shared modules,
// then bundles it. Returns bundled code, original code, bundled status, and any error.
func (h *Handler) bundleFunctionFromFilesystem(ctx context.Context, functionName string) (bundledCode string, originalCode string, isBundled bool, bundleError *string, err error) {
	// Load main code and supporting files
	mainCode, supportingFiles, err := LoadFunctionCodeWithFiles(h.functionsDir, functionName)
	if err != nil {
		return "", "", false, nil, fmt.Errorf("failed to load code: %w", err)
	}

	// Load shared modules from filesystem
	sharedModules, err := LoadSharedModulesFromFilesystem(h.functionsDir)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load shared modules from filesystem, continuing without them")
		sharedModules = make(map[string]string)
	}

	// Create bundler
	bundler, bundlerErr := NewBundler()
	if bundlerErr != nil {
		// No bundler available - return unbundled code
		return mainCode, mainCode, false, nil, nil
	}

	// Determine if we need to use BundleWithFiles (multi-file or shared imports)
	hasSharedImports := strings.Contains(mainCode, "from \"_shared/") || strings.Contains(mainCode, "from '_shared/")
	hasMultipleFiles := len(supportingFiles) > 0

	var result *BundleResult
	var bundleErr error

	if hasSharedImports || hasMultipleFiles {
		// Use BundleWithFiles for multi-file or shared module support
		result, bundleErr = bundler.BundleWithFiles(ctx, mainCode, supportingFiles, sharedModules)
	} else {
		// Simple single-file bundle
		result, bundleErr = bundler.Bundle(ctx, mainCode)
	}

	if bundleErr != nil {
		// Bundling failed - return unbundled code with error
		errMsg := fmt.Sprintf("bundle failed: %v", bundleErr)
		return mainCode, mainCode, false, &errMsg, nil
	}

	// Bundling succeeded
	var bundleErrPtr *string
	if result.Error != "" {
		bundleErrPtr = &result.Error
	}

	return result.BundledCode, mainCode, result.IsBundled, bundleErrPtr, nil
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
	created := []string{}
	updated := []string{}
	deleted := []string{}
	errors := []string{}

	// Process each function file
	for _, fileInfo := range functionFiles {
		// Check if function exists in database
		existingFn, err := h.storage.GetFunction(ctx, fileInfo.Name)

		if err != nil {
			// Function doesn't exist in database - create it
			bundledCode, originalCode, isBundled, bundleError, err := h.bundleFunctionFromFilesystem(ctx, fileInfo.Name)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", fileInfo.Name, err))
				continue
			}

			// Parse configuration from code comments
			config := ParseFunctionConfig(originalCode)

			// Create new function with default settings
			fn := &EdgeFunction{
				Name:                 fileInfo.Name,
				Code:                 bundledCode,
				OriginalCode:         &originalCode,
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
				IsPublic:             config.IsPublic,
				Source:               "filesystem",
			}

			if err := h.storage.CreateFunction(ctx, fn); err != nil {
				errors = append(errors, fmt.Sprintf("%s: failed to create: %v", fileInfo.Name, err))
				continue
			}

			created = append(created, fileInfo.Name)
		} else {
			// Function exists - update code from filesystem
			bundledCode, originalCode, isBundled, bundleError, err := h.bundleFunctionFromFilesystem(ctx, fileInfo.Name)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", fileInfo.Name, err))
				continue
			}

			// Parse configuration from code comments
			config := ParseFunctionConfig(originalCode)

			// Update if code or config has changed
			// Compare with original_code if available, otherwise with code
			compareCode := originalCode
			if existingFn.OriginalCode != nil {
				compareCode = *existingFn.OriginalCode
			}

			if existingFn.Code != bundledCode || compareCode != originalCode || existingFn.AllowUnauthenticated != config.AllowUnauthenticated || existingFn.IsPublic != config.IsPublic {
				updates := map[string]interface{}{
					"code":                  bundledCode,
					"original_code":         originalCode,
					"is_bundled":            isBundled,
					"bundle_error":          bundleError,
					"allow_unauthenticated": config.AllowUnauthenticated,
					"is_public":             config.IsPublic,
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
	// Only delete filesystem-sourced functions, preserve API-created ones
	for _, dbFunc := range allFunctions {
		if !diskFunctionNames[dbFunc.Name] && dbFunc.Source == "filesystem" {
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

// SyncFunctions syncs a list of functions to a specific namespace
// Admin-only endpoint - requires authentication and admin role
func (h *Handler) SyncFunctions(c *fiber.Ctx) error {
	var req struct {
		Namespace string `json:"namespace"`
		Functions []struct {
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
			IsPublic             *bool   `json:"is_public"`
			CronSchedule         *string `json:"cron_schedule"`
		} `json:"functions"`
		Options struct {
			DeleteMissing bool `json:"delete_missing"`
			DryRun        bool `json:"dry_run"`
		} `json:"options"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Default namespace to "default" if not specified
	namespace := req.Namespace
	if namespace == "" {
		namespace = "default"
	}

	ctx := c.Context()

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

	// Get all existing functions in this namespace
	existingFunctions, err := h.storage.ListFunctionsByNamespace(ctx, namespace)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to list existing functions in namespace",
		})
	}

	// Build set of existing function names
	existingNames := make(map[string]*EdgeFunctionSummary)
	for i := range existingFunctions {
		existingNames[existingFunctions[i].Name] = &existingFunctions[i]
	}

	// Build set of payload function names
	payloadNames := make(map[string]bool)
	for _, spec := range req.Functions {
		payloadNames[spec.Name] = true
	}

	// Determine operations
	toCreate := []string{}
	toUpdate := []string{}
	toDelete := []string{}

	for _, spec := range req.Functions {
		if _, exists := existingNames[spec.Name]; exists {
			toUpdate = append(toUpdate, spec.Name)
		} else {
			toCreate = append(toCreate, spec.Name)
		}
	}

	if req.Options.DeleteMissing {
		for name := range existingNames {
			if !payloadNames[name] {
				toDelete = append(toDelete, name)
			}
		}
	}

	// Track results
	created := []string{}
	updated := []string{}
	deleted := []string{}
	unchanged := []string{}
	errorList := []fiber.Map{}

	// If dry run, return what would be done without making changes
	if req.Options.DryRun {
		return c.JSON(fiber.Map{
			"message":   "Dry run - no changes made",
			"namespace": namespace,
			"summary": fiber.Map{
				"created":   len(toCreate),
				"updated":   len(toUpdate),
				"deleted":   len(toDelete),
				"unchanged": 0,
			},
			"details": fiber.Map{
				"created":   toCreate,
				"updated":   toUpdate,
				"deleted":   toDelete,
				"unchanged": []string{},
			},
			"errors":  []string{},
			"dry_run": true,
		})
	}

	// Bundle and create/update functions in parallel
	type bundleResult struct {
		Name         string
		BundledCode  string
		OriginalCode string
		IsBundled    bool
		BundleError  *string
		Err          error
	}

	// Use semaphore to limit concurrent bundling to 10
	sem := make(chan struct{}, 10)
	var wg sync.WaitGroup
	resultsChan := make(chan bundleResult, len(req.Functions))

	// Load shared modules once (used by all bundles)
	sharedModules, _ := h.storage.ListSharedModules(ctx)
	sharedModulesMap := make(map[string]string)
	for _, module := range sharedModules {
		sharedModulesMap[module.ModulePath] = module.Content
	}

	// Bundle all functions in parallel
	for i := range req.Functions {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			spec := req.Functions[i]

			// Bundle the function code
			bundler, err := NewBundler()
			if err != nil {
				resultsChan <- bundleResult{
					Name: spec.Name,
					Err:  fmt.Errorf("failed to create bundler: %w", err),
				}
				return
			}

			bundledCode := spec.Code
			originalCode := spec.Code
			isBundled := false
			var bundleError *string

			// Check if code imports from _shared/ modules
			hasSharedImports := strings.Contains(spec.Code, "from \"_shared/") ||
				strings.Contains(spec.Code, "from '_shared/")

			var result *BundleResult
			var bundleErr error

			if hasSharedImports {
				supportingFiles := make(map[string]string)
				result, bundleErr = bundler.BundleWithFiles(context.Background(), spec.Code, supportingFiles, sharedModulesMap)
			} else {
				result, bundleErr = bundler.Bundle(context.Background(), spec.Code)
			}

			if bundleErr != nil {
				resultsChan <- bundleResult{
					Name: spec.Name,
					Err:  fmt.Errorf("bundle error: %w", bundleErr),
				}
				return
			}

			if result != nil {
				bundledCode = result.BundledCode
				isBundled = result.IsBundled
				if result.Error != "" {
					bundleError = &result.Error
				}
			}

			resultsChan <- bundleResult{
				Name:         spec.Name,
				BundledCode:  bundledCode,
				OriginalCode: originalCode,
				IsBundled:    isBundled,
				BundleError:  bundleError,
				Err:          nil,
			}
		}(i)
	}

	// Wait for all bundling to complete
	wg.Wait()
	close(resultsChan)

	// Collect bundling results
	bundleResults := make(map[string]bundleResult)
	for result := range resultsChan {
		bundleResults[result.Name] = result
		if result.Err != nil {
			errorList = append(errorList, fiber.Map{
				"function": result.Name,
				"error":    result.Err.Error(),
				"action":   "bundle",
			})
		}
	}

	// Create/Update functions
	for _, spec := range req.Functions {
		result, ok := bundleResults[spec.Name]
		if !ok || result.Err != nil {
			// Skip if bundling failed
			continue
		}

		// Parse configuration from code comments
		config := ParseFunctionConfig(spec.Code)

		// Determine values (request takes precedence over config)
		allowUnauthenticated := config.AllowUnauthenticated
		if spec.AllowUnauthenticated != nil {
			allowUnauthenticated = *spec.AllowUnauthenticated
		}

		isPublic := config.IsPublic
		if spec.IsPublic != nil {
			isPublic = *spec.IsPublic
		}

		if _, exists := existingNames[spec.Name]; exists {
			// Update existing function
			updates := map[string]interface{}{
				"code":                  result.BundledCode,
				"original_code":         result.OriginalCode,
				"is_bundled":            result.IsBundled,
				"bundle_error":          result.BundleError,
				"allow_unauthenticated": allowUnauthenticated,
				"is_public":             isPublic,
			}

			if spec.Description != nil {
				updates["description"] = spec.Description
			}
			if spec.Enabled != nil {
				updates["enabled"] = *spec.Enabled
			}
			if spec.TimeoutSeconds != nil {
				updates["timeout_seconds"] = *spec.TimeoutSeconds
			}
			if spec.MemoryLimitMB != nil {
				updates["memory_limit_mb"] = *spec.MemoryLimitMB
			}
			if spec.AllowNet != nil {
				updates["allow_net"] = *spec.AllowNet
			}
			if spec.AllowEnv != nil {
				updates["allow_env"] = *spec.AllowEnv
			}
			if spec.AllowRead != nil {
				updates["allow_read"] = *spec.AllowRead
			}
			if spec.AllowWrite != nil {
				updates["allow_write"] = *spec.AllowWrite
			}
			if spec.CronSchedule != nil {
				updates["cron_schedule"] = *spec.CronSchedule
			}

			if err := h.storage.UpdateFunctionByNamespace(ctx, spec.Name, namespace, updates); err != nil {
				errorList = append(errorList, fiber.Map{
					"function": spec.Name,
					"error":    err.Error(),
					"action":   "update",
				})
				continue
			}

			updated = append(updated, spec.Name)
		} else {
			// Create new function
			fn := &EdgeFunction{
				Name:                 spec.Name,
				Namespace:            namespace,
				Description:          spec.Description,
				Code:                 result.BundledCode,
				OriginalCode:         &result.OriginalCode,
				IsBundled:            result.IsBundled,
				BundleError:          result.BundleError,
				Enabled:              valueOr(spec.Enabled, true),
				TimeoutSeconds:       valueOr(spec.TimeoutSeconds, 30),
				MemoryLimitMB:        valueOr(spec.MemoryLimitMB, 128),
				AllowNet:             valueOr(spec.AllowNet, true),
				AllowEnv:             valueOr(spec.AllowEnv, true),
				AllowRead:            valueOr(spec.AllowRead, false),
				AllowWrite:           valueOr(spec.AllowWrite, false),
				AllowUnauthenticated: allowUnauthenticated,
				IsPublic:             isPublic,
				CronSchedule:         spec.CronSchedule,
				CreatedBy:            createdBy,
			}

			if err := h.storage.CreateFunction(ctx, fn); err != nil {
				errorList = append(errorList, fiber.Map{
					"function": spec.Name,
					"error":    err.Error(),
					"action":   "create",
				})
				continue
			}

			created = append(created, spec.Name)
		}
	}

	// Delete removed functions (after successful creates/updates for safety)
	if req.Options.DeleteMissing {
		for _, name := range toDelete {
			if err := h.storage.DeleteFunctionByNamespace(ctx, name, namespace); err != nil {
				errorList = append(errorList, fiber.Map{
					"function": name,
					"error":    err.Error(),
					"action":   "delete",
				})
				continue
			}
			deleted = append(deleted, name)
		}
	}

	return c.JSON(fiber.Map{
		"message":   "Functions synced successfully",
		"namespace": namespace,
		"summary": fiber.Map{
			"created":   len(created),
			"updated":   len(updated),
			"deleted":   len(deleted),
			"unchanged": len(unchanged),
			"errors":    len(errorList),
		},
		"details": fiber.Map{
			"created":   created,
			"updated":   updated,
			"deleted":   deleted,
			"unchanged": unchanged,
		},
		"errors":  errorList,
		"dry_run": false,
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
	created := []string{}
	updated := []string{}
	errors := []string{}

	// Process each function file
	for _, fileInfo := range functionFiles {
		// Check if function exists in database
		existingFn, err := h.storage.GetFunction(ctx, fileInfo.Name)

		if err != nil {
			// Function doesn't exist in database - create it
			bundledCode, originalCode, isBundled, bundleError, err := h.bundleFunctionFromFilesystem(ctx, fileInfo.Name)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", fileInfo.Name, err))
				continue
			}

			// Parse configuration from code comments
			config := ParseFunctionConfig(originalCode)

			// Create new function with default settings
			fn := &EdgeFunction{
				Name:                 fileInfo.Name,
				Code:                 bundledCode,
				OriginalCode:         &originalCode,
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
				IsPublic:             config.IsPublic,
				Source:               "filesystem",
			}

			if err := h.storage.CreateFunction(ctx, fn); err != nil {
				errors = append(errors, fmt.Sprintf("%s: failed to create: %v", fileInfo.Name, err))
				continue
			}

			created = append(created, fileInfo.Name)
		} else {
			// Function exists - update code from filesystem
			bundledCode, originalCode, isBundled, bundleError, err := h.bundleFunctionFromFilesystem(ctx, fileInfo.Name)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", fileInfo.Name, err))
				continue
			}

			// Parse configuration from code comments
			config := ParseFunctionConfig(originalCode)

			// Update if code or config has changed
			// Compare with original_code if available, otherwise with code
			compareCode := originalCode
			if existingFn.OriginalCode != nil {
				compareCode = *existingFn.OriginalCode
			}

			if existingFn.Code != bundledCode || compareCode != originalCode || existingFn.AllowUnauthenticated != config.AllowUnauthenticated || existingFn.IsPublic != config.IsPublic {
				updates := map[string]interface{}{
					"code":                  bundledCode,
					"original_code":         originalCode,
					"is_bundled":            isBundled,
					"bundle_error":          bundleError,
					"allow_unauthenticated": config.AllowUnauthenticated,
					"is_public":             config.IsPublic,
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

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// getRequestID extracts the request ID from the fiber context
func getRequestID(c *fiber.Ctx) string {
	requestID := c.Locals("requestid")
	if requestID != nil {
		if reqIDStr, ok := requestID.(string); ok {
			return reqIDStr
		}
	}
	return c.Get("X-Request-ID", "")
}

// toString converts a value to string for logging
func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	if uid, ok := v.(*uuid.UUID); ok {
		if uid == nil {
			return ""
		}
		return uid.String()
	}
	return fmt.Sprintf("%v", v)
}

// CreateSharedModule creates a new shared module
func (h *Handler) CreateSharedModule(c *fiber.Ctx) error {
	var req struct {
		ModulePath  string  `json:"module_path"`
		Content     string  `json:"content"`
		Description *string `json:"description"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Validate module_path starts with _shared/
	if !strings.HasPrefix(req.ModulePath, "_shared/") {
		return c.Status(400).JSON(fiber.Map{"error": "Module path must start with '_shared/'"})
	}

	// Get user ID from context (if authenticated)
	var userID *uuid.UUID
	if uid := c.Locals("user_id"); uid != nil {
		if parsedUID, ok := uid.(uuid.UUID); ok {
			userID = &parsedUID
		}
	}

	module := &SharedModule{
		ModulePath:  req.ModulePath,
		Content:     req.Content,
		Description: req.Description,
		CreatedBy:   userID,
	}

	if err := h.storage.CreateSharedModule(c.Context(), module); err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "already exists") {
			return c.Status(409).JSON(fiber.Map{"error": "Shared module already exists"})
		}
		log.Error().Err(err).Str("module_path", req.ModulePath).Msg("Failed to create shared module")
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create shared module"})
	}

	log.Info().
		Str("module_path", module.ModulePath).
		Str("user_id", toString(userID)).
		Msg("Shared module created")

	return c.Status(201).JSON(module)
}

// ListSharedModules returns all shared modules
func (h *Handler) ListSharedModules(c *fiber.Ctx) error {
	modules, err := h.storage.ListSharedModules(c.Context())
	if err != nil {
		log.Error().Err(err).Msg("Failed to list shared modules")
		return c.Status(500).JSON(fiber.Map{"error": "Failed to list shared modules"})
	}

	return c.JSON(modules)
}

// GetSharedModule retrieves a shared module by path
func (h *Handler) GetSharedModule(c *fiber.Ctx) error {
	// Get full path from wildcard (e.g., "cors.ts" from "/shared/cors.ts")
	modulePath := strings.TrimPrefix(c.Params("*"), "/")

	// Ensure it starts with _shared/
	if !strings.HasPrefix(modulePath, "_shared/") {
		modulePath = "_shared/" + modulePath
	}

	module, err := h.storage.GetSharedModule(c.Context(), modulePath)
	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(404).JSON(fiber.Map{"error": "Shared module not found"})
		}
		log.Error().Err(err).Str("module_path", modulePath).Msg("Failed to get shared module")
		return c.Status(500).JSON(fiber.Map{"error": "Failed to get shared module"})
	}

	return c.JSON(module)
}

// UpdateSharedModule updates an existing shared module
func (h *Handler) UpdateSharedModule(c *fiber.Ctx) error {
	// Get full path from wildcard
	modulePath := strings.TrimPrefix(c.Params("*"), "/")

	// Ensure it starts with _shared/
	if !strings.HasPrefix(modulePath, "_shared/") {
		modulePath = "_shared/" + modulePath
	}

	var req struct {
		Content     string  `json:"content"`
		Description *string `json:"description"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := h.storage.UpdateSharedModule(c.Context(), modulePath, req.Content, req.Description); err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(404).JSON(fiber.Map{"error": "Shared module not found"})
		}
		log.Error().Err(err).Str("module_path", modulePath).Msg("Failed to update shared module")
		return c.Status(500).JSON(fiber.Map{"error": "Failed to update shared module"})
	}

	// Get updated module
	module, err := h.storage.GetSharedModule(c.Context(), modulePath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Module updated but failed to retrieve"})
	}

	log.Info().
		Str("module_path", modulePath).
		Int("version", module.Version).
		Msg("Shared module updated")

	return c.JSON(module)
}

// DeleteSharedModule deletes a shared module
func (h *Handler) DeleteSharedModule(c *fiber.Ctx) error {
	// Get full path from wildcard
	modulePath := strings.TrimPrefix(c.Params("*"), "/")

	// Ensure it starts with _shared/
	if !strings.HasPrefix(modulePath, "_shared/") {
		modulePath = "_shared/" + modulePath
	}

	if err := h.storage.DeleteSharedModule(c.Context(), modulePath); err != nil {
		log.Error().Err(err).Str("module_path", modulePath).Msg("Failed to delete shared module")
		return c.Status(500).JSON(fiber.Map{"error": "Failed to delete shared module"})
	}

	log.Info().Str("module_path", modulePath).Msg("Shared module deleted")

	return c.JSON(fiber.Map{"message": "Shared module deleted successfully"})
}
