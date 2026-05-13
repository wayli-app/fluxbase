package functions

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/database"
	apperrors "github.com/nimbleflux/fluxbase/internal/errors"
	"github.com/nimbleflux/fluxbase/internal/logging"
	"github.com/nimbleflux/fluxbase/internal/middleware"
	"github.com/nimbleflux/fluxbase/internal/ratelimit"
	"github.com/nimbleflux/fluxbase/internal/runtime"
	"github.com/nimbleflux/fluxbase/internal/secrets"
	"github.com/nimbleflux/fluxbase/internal/settings"
	"github.com/nimbleflux/fluxbase/internal/util"
)

// Handler manages HTTP endpoints for edge functions
type Handler struct {
	storage                *Storage
	runtime                *runtime.DenoRuntime
	scheduler              *Scheduler
	authService            *auth.Service
	loggingService         *logging.Service
	secretsStorage         *secrets.Storage
	settingsSecretsService *settings.SecretsService
	rateLimiter            ratelimit.Store
	functionsDir           string
	corsConfig             config.CORSConfig
	publicURL              string
	npmRegistry            string
	jsrRegistry            string
	logCounters            sync.Map
	baseConfig             *config.Config
}

// NewHandler creates a new edge functions handler
func NewHandler(db *database.Connection, functionsDir string, corsConfig config.CORSConfig, jwtSecret, publicURL, npmRegistry, jsrRegistry string, authService *auth.Service, loggingService *logging.Service, secretsStorage *secrets.Storage, baseConfig *config.Config) *Handler {
	opts := []runtime.Option{}
	if baseConfig != nil && baseConfig.Functions.MaxOutputSize > 0 {
		opts = append(opts, runtime.WithMaxOutputSize(baseConfig.Functions.MaxOutputSize))
	}
	h := &Handler{
		storage:        NewStorage(db),
		runtime:        runtime.NewRuntime(runtime.RuntimeTypeFunction, jwtSecret, publicURL, opts...),
		authService:    authService,
		loggingService: loggingService,
		secretsStorage: secretsStorage,
		functionsDir:   functionsDir,
		corsConfig:     corsConfig,
		publicURL:      publicURL,
		npmRegistry:    npmRegistry,
		jsrRegistry:    jsrRegistry,
		baseConfig:     baseConfig,
	}

	// Set up log callback to capture console.log output
	h.runtime.SetLogCallback(h.handleLogMessage)

	return h
}

// SetScheduler sets the scheduler for this handler
func (h *Handler) SetScheduler(scheduler *Scheduler) {
	h.scheduler = scheduler
}

// SetSettingsSecretsService sets the settings secrets service for accessing user/system secrets
func (h *Handler) SetSettingsSecretsService(svc *settings.SecretsService) {
	h.settingsSecretsService = svc
}

func (h *Handler) SetRateLimiter(store ratelimit.Store) {
	h.rateLimiter = store
}

// GetRuntime returns the Deno runtime for external use (e.g., MCP tools)
func (h *Handler) GetRuntime() *runtime.DenoRuntime {
	return h.runtime
}

// GetPublicURL returns the public URL configured for this handler
func (h *Handler) GetPublicURL() string {
	return h.publicURL
}

// createBundler creates a new bundler with the handler's registry configuration
func (h *Handler) createBundler() (*Bundler, error) {
	var opts []BundlerOption
	if h.npmRegistry != "" {
		opts = append(opts, WithNpmRegistry(h.npmRegistry))
	}
	if h.jsrRegistry != "" {
		opts = append(opts, WithJsrRegistry(h.jsrRegistry))
	}
	return NewBundler(opts...)
}

// GetFunctionsDir returns the functions directory path
func (h *Handler) GetFunctionsDir() string {
	return h.functionsDir
}

// getConfig returns the functions config to use for the current request.
// It checks for tenant-specific config in fiber context locals and falls back to base config.
func (h *Handler) getConfig(c fiber.Ctx) *config.FunctionsConfig {
	if tc, ok := c.Locals("tenant_config").(*config.Config); ok && tc != nil {
		return &tc.Functions
	}
	return &h.baseConfig.Functions
}

type executionLogContext struct {
	lineCounter int
	tenantID    string
}

func (h *Handler) handleLogMessage(executionID uuid.UUID, level string, message string) {
	counterVal, ok := h.logCounters.Load(executionID)
	if !ok {
		log.Info().
			Str("execution_id", executionID.String()).
			Str("execution_type", "function").
			Str("level", level).
			Msg(message)
		return
	}

	ctx, ok := counterVal.(*executionLogContext)
	if !ok {
		log.Warn().Str("execution_id", executionID.String()).Msg("Invalid log counter type")
		return
	}

	lineNumber := ctx.lineCounter
	ctx.lineCounter = lineNumber + 1

	event := log.Info().
		Str("execution_id", executionID.String()).
		Str("execution_type", "function").
		Str("level", level).
		Int("line_number", lineNumber)

	if ctx.tenantID != "" {
		event = event.Str("tenant_id", ctx.tenantID)
	}

	event.Msg(message)
}

// applyCorsHeaders applies CORS headers to the response with fallback to global config
func (h *Handler) applyCorsHeaders(c fiber.Ctx, fn *EdgeFunction) {
	// Determine CORS values with fallback: function settings > global config
	// Function settings are stored as comma-separated strings, global config uses slices
	origins := h.corsConfig.AllowedOrigins
	if fn.CorsOrigins != nil {
		origins = strings.Split(*fn.CorsOrigins, ",")
		for i := range origins {
			origins[i] = strings.TrimSpace(origins[i])
		}
	}

	methods := h.corsConfig.AllowedMethods
	if fn.CorsMethods != nil {
		methods = strings.Split(*fn.CorsMethods, ",")
		for i := range methods {
			methods[i] = strings.TrimSpace(methods[i])
		}
	}

	headers := h.corsConfig.AllowedHeaders
	if fn.CorsHeaders != nil {
		headers = strings.Split(*fn.CorsHeaders, ",")
		for i := range headers {
			headers[i] = strings.TrimSpace(headers[i])
		}
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
	// - If origins contains "*", use "*"
	// - If origins contains multiple values, check if request origin matches
	// - Browsers only accept a single origin or "*", not lists
	hasWildcard := false
	for _, o := range origins {
		if o == "*" {
			hasWildcard = true
			break
		}
	}

	var allowedOrigin string
	switch {
	case hasWildcard:
		allowedOrigin = "*"
	case len(origins) == 1:
		allowedOrigin = origins[0]
	default:
		requestOrigin := c.Get("Origin")
		if requestOrigin != "" {
			// Check if request origin is in the allowed list
			for _, allowed := range origins {
				if allowed == requestOrigin {
					allowedOrigin = requestOrigin
					break
				}
			}
		}
		// If no match, use first allowed origin (will cause CORS failure, but that's expected)
		if allowedOrigin == "" && len(origins) > 0 {
			allowedOrigin = origins[0]
		}
	}

	c.Set("Access-Control-Allow-Origin", allowedOrigin)
	c.Set("Access-Control-Allow-Methods", strings.Join(methods, ", "))
	c.Set("Access-Control-Allow-Headers", strings.Join(headers, ", "))

	if credentials && allowedOrigin != "*" {
		c.Set("Access-Control-Allow-Credentials", "true")
	}

	if maxAge > 0 {
		c.Set("Access-Control-Max-Age", strconv.Itoa(maxAge))
	}

	// Expose headers if configured
	if len(h.corsConfig.ExposedHeaders) > 0 {
		c.Set("Access-Control-Expose-Headers", strings.Join(h.corsConfig.ExposedHeaders, ", "))
	}
}

// checkRateLimit checks function-specific rate limits and returns an error response if exceeded.
// Rate limits are checked per user ID (authenticated) or per IP (anonymous).
// Uses the global rate limit store which supports memory, PostgreSQL, or Redis backends.
func (h *Handler) checkRateLimit(c fiber.Ctx, fn *EdgeFunction) error {
	// Skip if no rate limits configured
	if fn.RateLimitPerMinute == nil && fn.RateLimitPerHour == nil && fn.RateLimitPerDay == nil {
		return nil
	}

	store := h.rateLimiter
	if store == nil {
		// No rate limit store available, fail open
		log.Warn().Msg("Rate limit store not available, skipping function rate limit check")
		return nil
	}

	// Build rate limit key: fn:{function_id}:{user_id} or fn:{function_id}:ip:{ip}
	var identifier string
	if uid := middleware.GetUserID(c); uid != "" {
		identifier = uid
	}
	if identifier == "" {
		identifier = "ip:" + c.IP()
	}

	baseKey := fmt.Sprintf("fn:%s:%s", fn.ID.String(), identifier)

	// Check each rate limit window (most restrictive first for efficiency)
	type limitCheck struct {
		limit    *int
		window   time.Duration
		suffix   string
		unitName string
	}

	checks := []limitCheck{
		{fn.RateLimitPerMinute, time.Minute, ":min", "minute"},
		{fn.RateLimitPerHour, time.Hour, ":hour", "hour"},
		{fn.RateLimitPerDay, 24 * time.Hour, ":day", "day"},
	}

	for _, check := range checks {
		if check.limit == nil || *check.limit <= 0 {
			continue
		}

		key := baseKey + check.suffix
		result, err := ratelimit.Check(middleware.CtxWithTenant(c), store, key, int64(*check.limit), check.window)
		if err != nil {
			// Fail open on rate limit errors
			log.Error().Err(err).Str("key", key).Msg("Rate limit check failed")
			continue
		}

		if !result.Allowed {
			retryAfter := int(time.Until(result.ResetAt).Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}

			c.Set("Retry-After", strconv.Itoa(retryAfter))
			c.Set("X-RateLimit-Limit", strconv.FormatInt(result.Limit, 10))
			c.Set("X-RateLimit-Remaining", "0")
			c.Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))

			return apperrors.SendErrorWithDetails(c, fiber.StatusTooManyRequests, fmt.Sprintf("Rate limit exceeded: %d requests per %s", *check.limit, check.unitName), apperrors.ErrCodeRateLimited, "", "", fiber.Map{"retry_after": retryAfter})
		}
	}

	return nil
}

// CreateFunction creates a new edge function
func (h *Handler) CreateFunction(c fiber.Ctx) error {
	var req struct {
		Name                 string  `json:"name"`
		Description          *string `json:"description"`
		Code                 string  `json:"code"`
		OriginalCode         *string `json:"original_code"` // Original code if pre-bundled (for editing)
		IsBundled            *bool   `json:"is_bundled"`    // If true, skip server-side bundling
		Enabled              *bool   `json:"enabled"`
		TimeoutSeconds       *int    `json:"timeout_seconds"`
		MemoryLimitMB        *int    `json:"memory_limit_mb"`
		AllowNet             *bool   `json:"allow_net"`
		AllowEnv             *bool   `json:"allow_env"`
		AllowRead            *bool   `json:"allow_read"`
		AllowWrite           *bool   `json:"allow_write"`
		AllowedDomains       *string `json:"allowed_domains"`
		AllowUnauthenticated *bool   `json:"allow_unauthenticated"`
		IsPublic             *bool   `json:"is_public"`
		CorsOrigins          *string `json:"cors_origins"`
		CorsMethods          *string `json:"cors_methods"`
		CorsHeaders          *string `json:"cors_headers"`
		CorsCredentials      *bool   `json:"cors_credentials"`
		CorsMaxAge           *int    `json:"cors_max_age"`
		RateLimitPerMinute   *int    `json:"rate_limit_per_minute"`
		RateLimitPerHour     *int    `json:"rate_limit_per_hour"`
		RateLimitPerDay      *int    `json:"rate_limit_per_day"`
		CronSchedule         *string `json:"cron_schedule"`
	}

	if err := c.Bind().Body(&req); err != nil {
		return apperrors.SendInvalidBody(c)
	}

	// Validation
	if req.Name == "" {
		return apperrors.SendMissingField(c, "Function name")
	}
	if req.Code == "" {
		return apperrors.SendMissingField(c, "Function code")
	}

	var createdBy *uuid.UUID
	if uid := middleware.GetUserID(c); uid != "" {
		parsed, err := uuid.Parse(uid)
		if err == nil {
			createdBy = &parsed
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

	// Apply rate limit config with priority: API request > annotations
	var rateLimitPerMinute *int
	if req.RateLimitPerMinute != nil {
		rateLimitPerMinute = req.RateLimitPerMinute
	} else {
		rateLimitPerMinute = config.RateLimitPerMinute
	}

	var rateLimitPerHour *int
	if req.RateLimitPerHour != nil {
		rateLimitPerHour = req.RateLimitPerHour
	} else {
		rateLimitPerHour = config.RateLimitPerHour
	}

	var rateLimitPerDay *int
	if req.RateLimitPerDay != nil {
		rateLimitPerDay = req.RateLimitPerDay
	} else {
		rateLimitPerDay = config.RateLimitPerDay
	}

	var allowedDomains *string
	if req.AllowedDomains != nil {
		allowedDomains = req.AllowedDomains
	} else {
		allowedDomains = config.AllowedDomains
	}

	// Bundle function code if it has imports
	bundledCode := req.Code
	originalCode := &req.Code
	isBundled := false
	var bundleError *string

	// If client sent pre-bundled code, skip server-side bundling
	if req.IsBundled != nil && *req.IsBundled {
		// Code is already bundled by the client
		isBundled = true
		// Use original_code if provided (for editing), otherwise use code as both
		if req.OriginalCode != nil && *req.OriginalCode != "" {
			originalCode = req.OriginalCode
		}
	} else {
		// Bundle the function code server-side
		bundler, err := h.createBundler()
		if err == nil {
			// Check if code imports from _shared/ modules
			hasSharedImports := strings.Contains(req.Code, "from \"_shared/") ||
				strings.Contains(req.Code, "from '_shared/")

			var result *BundleResult
			var bundleErr error

			if hasSharedImports {
				// Load all shared modules from database
				sharedModules, err := h.storage.ListSharedModules(middleware.CtxWithTenant(c))
				if err != nil {
					log.Warn().Err(err).Msg("Failed to load shared modules, proceeding with regular bundle")
					result, bundleErr = bundler.Bundle(middleware.CtxWithTenant(c), req.Code)
				} else {
					// Build map of shared module paths to content
					sharedModulesMap := make(map[string]string)
					for _, module := range sharedModules {
						sharedModulesMap[module.ModulePath] = module.Content
					}

					// Bundle with shared modules (no supporting files for now)
					supportingFiles := make(map[string]string)
					result, bundleErr = bundler.BundleWithFiles(middleware.CtxWithTenant(c), req.Code, supportingFiles, sharedModulesMap)
				}
			} else {
				// No shared imports - use regular bundling
				result, bundleErr = bundler.Bundle(middleware.CtxWithTenant(c), req.Code)
			}

			if bundleErr != nil {
				errMsg := fmt.Sprintf("Failed to bundle function: %v", bundleErr)
				return apperrors.SendErrorWithDetails(c, 400, "Bundle error", apperrors.ErrCodeInvalidInput, "", "", errMsg)
			}

			// Bundling succeeded
			bundledCode = result.BundledCode
			isBundled = result.IsBundled
			if result.Error != "" {
				bundleError = &result.Error
			}
		}
		// If bundler not available (Deno not installed), use unbundled code
	}

	// Create function - get tenant-specific config for defaults
	cfg := h.getConfig(c)
	defaultTimeout := cfg.DefaultTimeout
	if defaultTimeout <= 0 {
		defaultTimeout = 30
	}
	defaultMemory := cfg.DefaultMemoryLimit
	if defaultMemory <= 0 {
		defaultMemory = 128
	}
	// Validate against max limits
	timeoutSeconds := util.ValueOr(req.TimeoutSeconds, defaultTimeout)
	if cfg.MaxTimeout > 0 && timeoutSeconds > cfg.MaxTimeout {
		timeoutSeconds = cfg.MaxTimeout
	}
	memoryLimitMB := util.ValueOr(req.MemoryLimitMB, defaultMemory)
	if cfg.MaxMemoryLimit > 0 && memoryLimitMB > cfg.MaxMemoryLimit {
		memoryLimitMB = cfg.MaxMemoryLimit
	}

	fn := &EdgeFunction{
		Name:                 req.Name,
		Description:          req.Description,
		Code:                 bundledCode,
		OriginalCode:         originalCode,
		IsBundled:            isBundled,
		BundleError:          bundleError,
		Enabled:              req.Enabled != nil && *req.Enabled,
		TimeoutSeconds:       timeoutSeconds,
		MemoryLimitMB:        memoryLimitMB,
		AllowNet:             util.ValueOr(req.AllowNet, true),
		AllowEnv:             util.ValueOr(req.AllowEnv, true),
		AllowRead:            util.ValueOr(req.AllowRead, false),
		AllowWrite:           util.ValueOr(req.AllowWrite, false),
		AllowedDomains:       allowedDomains,
		AllowUnauthenticated: allowUnauthenticated,
		IsPublic:             isPublic,
		CorsOrigins:          corsOrigins,
		CorsMethods:          corsMethods,
		CorsHeaders:          corsHeaders,
		CorsCredentials:      corsCredentials,
		CorsMaxAge:           corsMaxAge,
		RateLimitPerMinute:   rateLimitPerMinute,
		RateLimitPerHour:     rateLimitPerHour,
		RateLimitPerDay:      rateLimitPerDay,
		CronSchedule:         req.CronSchedule,
		CreatedBy:            createdBy,
		Source:               "api",
	}

	if err := h.storage.CreateFunction(middleware.CtxWithTenant(c), fn); err != nil {
		reqID := apperrors.GetRequestID(c)
		log.Error().
			Err(err).
			Str("function_name", fn.Name).
			Str("request_id", reqID).
			Str("user_id", util.ToString(createdBy)).
			Msg("Failed to create edge function in database")

		return apperrors.SendErrorWithDetails(c, 500, "Failed to create function", apperrors.ErrCodeOperationFailed, "", "", err.Error())
	}

	return c.Status(201).JSON(fn)
}

// ListFunctions lists all edge functions
// Admin-only endpoint - non-admin users receive 403 Forbidden
func (h *Handler) ListFunctions(c fiber.Ctx) error {
	role := middleware.GetUserRole(c)
	if !isAdminRole(role) {
		return apperrors.SendAdminRequired(c)
	}

	// Check if namespace filter is provided
	namespace := c.Query("namespace")

	// Normalize "default" to empty string — functions without an explicit
	// namespace are stored as namespace="" but the UI presents them as "default".
	if namespace == "default" {
		namespace = ""
	}

	var functions []EdgeFunctionSummary
	var err error

	if namespace != "" {
		// If namespace is specified, list functions in that namespace
		functions, err = h.storage.ListFunctionsByNamespace(middleware.CtxWithTenant(c), namespace)
	} else {
		// Otherwise, list all functions (admin can see all)
		functions, err = h.storage.ListAllFunctions(middleware.CtxWithTenant(c))
	}

	if err != nil {
		reqID := apperrors.GetRequestID(c)
		log.Error().
			Err(err).
			Str("request_id", reqID).
			Msg("Failed to list edge functions from database")

		return apperrors.SendErrorWithDetails(c, 500, "Failed to list functions", apperrors.ErrCodeOperationFailed, "", "", err.Error())
	}

	return c.JSON(functions)
}

// ListNamespaces lists all unique namespaces with edge functions
func (h *Handler) ListNamespaces(c fiber.Ctx) error {
	namespaces, err := h.storage.ListFunctionNamespaces(middleware.CtxWithTenant(c))
	if err != nil {
		reqID := apperrors.GetRequestID(c)
		log.Error().
			Err(err).
			Str("request_id", reqID).
			Msg("Failed to list function namespaces")

		return apperrors.SendOperationFailed(c, "list function namespaces")
	}

	// Ensure we always return at least "default"
	if len(namespaces) == 0 {
		namespaces = []string{"default"}
	}

	// Normalize empty-string namespaces to "default" so the UI can present
	// them meaningfully and use the value in subsequent queries.
	for i := range namespaces {
		if namespaces[i] == "" {
			namespaces[i] = "default"
		}
	}

	return c.JSON(fiber.Map{"namespaces": namespaces})
}

// GetFunction gets a single function by name
// Admin-only endpoint - non-admin users receive 403 Forbidden
func (h *Handler) GetFunction(c fiber.Ctx) error {
	role := middleware.GetUserRole(c)
	if !isAdminRole(role) {
		return apperrors.SendAdminRequired(c)
	}

	name := c.Params("name")

	fn, err := h.storage.GetFunction(middleware.CtxWithTenant(c), name)
	if err != nil {
		return apperrors.SendResourceNotFound(c, "Function")
	}

	return c.JSON(fn)
}

// UpdateFunction updates an existing function
func (h *Handler) UpdateFunction(c fiber.Ctx) error {
	name := c.Params("name")

	var updates map[string]interface{}
	if err := c.Bind().Body(&updates); err != nil {
		return apperrors.SendInvalidBody(c)
	}

	// Remove fields that shouldn't be updated directly
	delete(updates, "id")
	delete(updates, "created_at")
	delete(updates, "updated_at")
	delete(updates, "version")

	// If code is being updated, re-bundle with shared modules
	if codeUpdate, ok := updates["code"].(string); ok && codeUpdate != "" {
		bundler, err := h.createBundler()
		if err == nil {
			// Check if code imports from _shared/ modules
			hasSharedImports := strings.Contains(codeUpdate, "from \"_shared/") ||
				strings.Contains(codeUpdate, "from '_shared/")

			var result *BundleResult
			var bundleErr error

			if hasSharedImports {
				// Load all shared modules from database
				sharedModules, err := h.storage.ListSharedModules(middleware.CtxWithTenant(c))
				if err != nil {
					log.Warn().Err(err).Msg("Failed to load shared modules for update, proceeding with regular bundle")
					result, bundleErr = bundler.Bundle(middleware.CtxWithTenant(c), codeUpdate)
				} else {
					// Build map of shared module paths to content
					sharedModulesMap := make(map[string]string)
					for _, module := range sharedModules {
						sharedModulesMap[module.ModulePath] = module.Content
					}

					// Bundle with shared modules
					supportingFiles := make(map[string]string)
					result, bundleErr = bundler.BundleWithFiles(middleware.CtxWithTenant(c), codeUpdate, supportingFiles, sharedModulesMap)
				}
			} else {
				// No shared imports - use regular bundling
				result, bundleErr = bundler.Bundle(middleware.CtxWithTenant(c), codeUpdate)
			}

			if bundleErr != nil {
				errMsg := fmt.Sprintf("Failed to bundle function: %v", bundleErr)
				return apperrors.SendErrorWithDetails(c, 400, "Bundle error", apperrors.ErrCodeInvalidInput, "", "", errMsg)
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

	reqID := apperrors.GetRequestID(c)
	if err := h.storage.UpdateFunction(middleware.CtxWithTenant(c), name, updates); err != nil {
		log.Error().
			Err(err).
			Str("function_name", name).
			Str("request_id", reqID).
			Msg("Failed to update edge function in database")

		return apperrors.SendErrorWithDetails(c, 500, "Failed to update function", apperrors.ErrCodeOperationFailed, "", "", err.Error())
	}

	// Return updated function
	fn, err := h.storage.GetFunction(middleware.CtxWithTenant(c), name)
	if err != nil {
		log.Error().
			Err(err).
			Str("function_name", name).
			Str("request_id", reqID).
			Msg("Failed to retrieve updated edge function from database")

		return apperrors.SendErrorWithDetails(c, 500, "Failed to retrieve updated function", apperrors.ErrCodeOperationFailed, "", "", err.Error())
	}

	return c.JSON(fn)
}

// DeleteFunction deletes a function
func (h *Handler) DeleteFunction(c fiber.Ctx) error {
	name := c.Params("name")

	if err := h.storage.DeleteFunction(middleware.CtxWithTenant(c), name); err != nil {
		reqID := apperrors.GetRequestID(c)
		log.Error().
			Err(err).
			Str("function_name", name).
			Str("request_id", reqID).
			Msg("Failed to delete edge function from database")

		return apperrors.SendErrorWithDetails(c, 500, "Failed to delete function", apperrors.ErrCodeOperationFailed, "", "", err.Error())
	}

	return c.SendStatus(204)
}

// Helper functions

// isAdminRole checks if the given role has admin privileges
func isAdminRole(role string) bool {
	return role == "admin" || role == "instance_admin" || role == "service_role" || role == "tenant_service"
}
