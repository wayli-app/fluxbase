package functions

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/middleware"
	"github.com/nimbleflux/fluxbase/internal/ratelimit"
	"github.com/nimbleflux/fluxbase/internal/runtime"

	"github.com/nimbleflux/fluxbase/internal/database"
	apperrors "github.com/nimbleflux/fluxbase/internal/errors"
	"github.com/nimbleflux/fluxbase/internal/util"
)

// InvokeFunction invokes an edge function
func (h *Handler) InvokeFunction(c fiber.Ctx) error {
	name := c.Params("name")
	namespace := c.Query("namespace")
	if namespace == "default" {
		namespace = ""
	}

	// Get function - if namespace is provided, look up by namespace+name; otherwise find first match by name
	var fn *EdgeFunction
	var err error
	if namespace != "" {
		fn, err = h.storage.GetFunctionByNamespace(middleware.CtxWithTenant(c), name, namespace)
	} else {
		fn, err = h.storage.GetFunction(middleware.CtxWithTenant(c), name)
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
				"error": "Authentication required. Provide an anon key (Bearer token with role=anon), client key (X-Client-Key header), or service key (X-Service-Key header). " +
					"To allow completely unauthenticated access, set allow_unauthenticated=true on the function.",
			})
		}
	}

	// Check function-specific rate limits
	if err := h.checkRateLimit(c, fn); err != nil {
		return err
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
	for key, value := range c.Request().Header.All() {
		req.Headers[string(key)] = string(value)
	}

	// Copy query parameters
	for key, value := range c.Request().URI().QueryArgs().All() {
		req.Params[string(key)] = string(value)
	}

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

	// Set tenant context for token generation and runtime bridge
	req.TenantID = middleware.GetTenantIDFromContext(c)
	req.TenantRole = middleware.GetTenantRoleFromContext(c)
	req.IsInstanceAdmin = middleware.IsInstanceAdminFromContext(c)

	// Check for impersonation token - allows admin to invoke functions as another user
	impersonationToken := c.Get("X-Impersonation-Token")
	if impersonationToken != "" && h.authService != nil {
		// SECURITY: Rate limit impersonation token attempts to prevent brute force attacks
		// Limit: 5 attempts per 5 minutes per IP address
		store := h.rateLimiter
		if store != nil {
			rateLimitKey := "impersonation:" + c.IP()
			result, err := ratelimit.Check(middleware.CtxWithTenant(c), store, rateLimitKey, 5, 5*time.Minute)
			if err != nil {
				log.Error().Err(err).Str("ip", c.IP()).Msg("Failed to check impersonation rate limit")
			} else if !result.Allowed {
				log.Warn().
					Str("ip", c.IP()).
					Int64("limit", result.Limit).
					Time("reset_at", result.ResetAt).
					Msg("SECURITY: Impersonation token rate limit exceeded - possible brute force attack")
				return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
					"error":       "Rate limit exceeded",
					"message":     "Too many impersonation attempts. Please try again in 5 minutes.",
					"retry_after": int(time.Until(result.ResetAt).Seconds()),
				})
			}
		}

		// Trim any whitespace that might have been added
		impersonationToken = strings.TrimSpace(impersonationToken)

		// SECURITY FIX: Reduced from 30 to 8 characters to minimize token exposure in logs
		// 8 chars is enough to identify token format without exposing sensitive data
		tokenPreview := impersonationToken
		if len(tokenPreview) > 8 {
			tokenPreview = tokenPreview[:8] + "..."
		}
		log.Info().
			Str("token_preview", tokenPreview).
			Int("token_length", len(impersonationToken)).
			Bool("starts_with_bearer", strings.HasPrefix(impersonationToken, "Bearer ")).
			Bool("starts_with_ey", strings.HasPrefix(impersonationToken, "ey")).
			Msg("Validating impersonation token")

		impersonationClaims, err := h.authService.ValidateToken(impersonationToken)
		if err != nil {
			log.Warn().
				Err(err).
				Str("token_preview", tokenPreview).
				Int("token_length", len(impersonationToken)).
				Str("ip", c.IP()).
				Msg("SECURITY: Invalid impersonation token in function invocation")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid impersonation token",
			})
		}

		// Override user context with impersonated user
		req.UserID = impersonationClaims.UserID
		req.UserEmail = impersonationClaims.Email
		req.UserRole = impersonationClaims.Role
		req.SessionID = impersonationClaims.SessionID

		log.Info().
			Str("function_name", name).
			Str("impersonated_user_id", impersonationClaims.UserID).
			Str("impersonated_role", impersonationClaims.Role).
			Msg("Function invocation with impersonation")
	}

	// Build permissions
	perms := runtime.DefaultFunctionPermissions()
	perms.AllowNet = fn.AllowNet
	perms.AllowEnv = fn.AllowEnv
	perms.AllowRead = fn.AllowRead
	perms.AllowWrite = fn.AllowWrite
	if fn.AllowedDomains != nil && *fn.AllowedDomains != "" {
		perms.AllowedDomains = strings.Split(*fn.AllowedDomains, ",")
		for i := range perms.AllowedDomains {
			perms.AllowedDomains[i] = strings.TrimSpace(perms.AllowedDomains[i])
		}
	}

	// Log function invocation
	reqID := apperrors.GetRequestID(c)
	log.Info().
		Str("function_name", name).
		Str("execution_id", executionID.String()).
		Str("user_id", req.UserID).
		Str("method", req.Method).
		Str("request_id", reqID).
		Msg("Invoking edge function")

	// Create execution record BEFORE running to enable real-time logging
	// Skip if execution logs are disabled for this function
	if !fn.DisableExecutionLogs {
		if err := h.storage.CreateExecution(middleware.CtxWithTenant(c), executionID, fn.ID, "http"); err != nil {
			log.Error().Err(err).Str("execution_id", executionID.String()).Msg("Failed to create execution record")
			// Continue anyway - logging will still work via stderr fallback
		}
	}

	// Initialize log counter for this execution
	tenantID := middleware.GetTenantIDFromContext(c)
	h.logCounters.Store(executionID, &executionLogContext{tenantID: tenantID})
	defer h.logCounters.Delete(executionID)

	// Build timeout override from function settings
	var timeoutOverride *time.Duration
	if fn.TimeoutSeconds > 0 {
		timeout := time.Duration(fn.TimeoutSeconds) * time.Second
		timeoutOverride = &timeout
	}

	// Load secrets for this function's namespace
	var functionSecrets map[string]string
	if h.secretsStorage != nil {
		var err error
		functionSecrets, err = h.secretsStorage.GetSecretsForNamespace(middleware.CtxWithTenant(c), fn.Namespace)
		if err != nil {
			log.Warn().Err(err).Str("namespace", fn.Namespace).Msg("Failed to load secrets for function execution")
			// Continue without secrets - don't fail the function invocation
		}
	}

	// Load settings secrets (user-specific and system-level)
	// These are injected as FLUXBASE_USER_* and FLUXBASE_SETTING_* env vars
	var userIDPtr *uuid.UUID
	if req.UserID != "" {
		if parsed, err := uuid.Parse(req.UserID); err == nil {
			userIDPtr = &parsed
		}
	}
	settingsSecrets := h.loadSettingsSecrets(middleware.CtxWithTenant(c), userIDPtr)

	// Merge all secrets: function secrets first, then settings secrets (which include the env var prefix already)
	allSecrets := make(map[string]string)
	for k, v := range functionSecrets {
		allSecrets[k] = v
	}
	for k, v := range settingsSecrets {
		allSecrets[k] = v
	}

	// Execute function (nil cancel signal for basic invocation - streaming endpoint will use actual signal)
	result, err := h.runtime.Execute(middleware.CtxWithTenant(c), fn.Code, req, perms, nil, timeoutOverride, allSecrets)

	// Complete execution record
	durationMs := int(result.DurationMs)
	status := "success"
	var errorMessage *string
	if err != nil {
		status = "error"
		errorMessage = &result.Error
	} else if result.Status >= 400 {
		status = "error"
		errMsg := fmt.Sprintf("Function returned HTTP %d", result.Status)
		errorMessage = &errMsg
	}

	var resultBody *string
	if result.Body != "" {
		resultBody = &result.Body
	}

	// Update execution record asynchronously (don't block response)
	// Skip if execution logs are disabled for this function
	if !fn.DisableExecutionLogs {
		tenantID := middleware.GetTenantIDFromContext(c)
		completeCtx := database.ContextWithTenant(context.Background(), tenantID)
		go func() {
			if updateErr := h.storage.CompleteExecution(completeCtx, executionID, status, &result.Status, &durationMs, resultBody, &result.Logs, errorMessage); updateErr != nil {
				log.Error().Err(updateErr).Str("execution_id", executionID.String()).Msg("Failed to complete execution record")
			}
		}()
	}

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
			Str("response_preview", util.TruncateString(result.Body, 200)).
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
func (h *Handler) GetExecutions(c fiber.Ctx) error {
	name := c.Params("name")
	limit := fiber.Query[int](c, "limit", 50)

	if limit > 100 {
		limit = 100
	}

	executions, err := h.storage.GetExecutions(middleware.CtxWithTenant(c), name, limit)
	if err != nil {
		reqID := apperrors.GetRequestID(c)
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
func (h *Handler) ListAllExecutions(c fiber.Ctx) error {
	limit := fiber.Query[int](c, "limit", 25)
	offset := fiber.Query[int](c, "offset", 0)
	namespace := c.Query("namespace")
	if namespace == "default" {
		namespace = ""
	}
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

	executions, total, err := h.storage.ListAllExecutions(middleware.CtxWithTenant(c), filters)
	if err != nil {
		reqID := apperrors.GetRequestID(c)
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
func (h *Handler) GetExecutionLogs(c fiber.Ctx) error {
	if h.loggingService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Logging service not available",
		})
	}

	executionIDStr := c.Params("id")

	_, err := uuid.Parse(executionIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid execution ID",
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
	entries, err := h.loggingService.GetExecutionLogs(middleware.CtxWithTenant(c), executionIDStr, afterLine)
	if err != nil {
		log.Error().Err(err).Str("execution_id", executionIDStr).Msg("Failed to get execution logs")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get execution logs",
		})
	}

	return c.JSON(fiber.Map{
		"entries": entries,
		"count":   len(entries),
	})
}

// loadSettingsSecrets loads settings secrets (user-specific and system-level) for function execution.
// Returns a map of environment variable name -> decrypted value.
// User secrets use prefix FLUXBASE_USER_, system secrets use prefix FLUXBASE_SETTING_.
func (h *Handler) loadSettingsSecrets(ctx context.Context, userID *uuid.UUID) map[string]string {
	if h.settingsSecretsService == nil {
		return nil
	}

	envVars := make(map[string]string)

	// Load system-level settings secrets
	systemSecrets, err := h.settingsSecretsService.GetSystemSecrets(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load system settings secrets for function execution")
	} else {
		for key, value := range systemSecrets {
			envName := "FLUXBASE_SETTING_" + normalizeSettingsKey(key)
			envVars[envName] = value
		}
	}

	// Load user-specific settings secrets (if user is authenticated)
	if userID != nil {
		userSecrets, err := h.settingsSecretsService.GetUserSecrets(ctx, *userID)
		if err != nil {
			log.Warn().Err(err).Str("user_id", userID.String()).Msg("Failed to load user settings secrets for function execution")
		} else {
			for key, value := range userSecrets {
				envName := "FLUXBASE_USER_" + normalizeSettingsKey(key)
				envVars[envName] = value
			}
		}
	}

	return envVars
}

// normalizeSettingsKey converts a settings key to an environment variable suffix.
// Example: "openai_api_key" -> "OPENAI_API_KEY", "ai.openai.api_key" -> "AI_OPENAI_API_KEY"
func normalizeSettingsKey(key string) string {
	// Replace dots with underscores, then uppercase
	normalized := strings.ReplaceAll(key, ".", "_")
	return strings.ToUpper(normalized)
}
