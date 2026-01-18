package api

import (
	"fmt"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// Standard error codes for consistent API error responses.
// These codes are returned in the "code" field of error responses.
const (
	// Authentication errors (401)
	ErrCodeMissingAuth        = "MISSING_AUTHENTICATION"
	ErrCodeInvalidToken       = "INVALID_TOKEN"
	ErrCodeExpiredToken       = "EXPIRED_TOKEN"
	ErrCodeRevokedToken       = "REVOKED_TOKEN"
	ErrCodeAuthRequired       = "AUTHENTICATION_REQUIRED"
	ErrCodeInvalidUserID      = "INVALID_USER_ID"
	ErrCodeAccountLocked      = "ACCOUNT_LOCKED"
	ErrCodeInvalidCredentials = "INVALID_CREDENTIALS"

	// Authorization errors (403)
	ErrCodeInsufficientPermissions = "INSUFFICIENT_PERMISSIONS"
	ErrCodeAdminRequired           = "ADMIN_REQUIRED"
	ErrCodeInvalidRole             = "INVALID_ROLE"
	ErrCodeRLSViolation            = "RLS_POLICY_VIOLATION"
	ErrCodeAccessDenied            = "ACCESS_DENIED"
	ErrCodeFeatureDisabled         = "FEATURE_DISABLED"

	// Validation errors (400)
	ErrCodeInvalidBody      = "INVALID_REQUEST_BODY"
	ErrCodeMissingField     = "MISSING_REQUIRED_FIELD"
	ErrCodeInvalidInput     = "INVALID_INPUT"
	ErrCodeInvalidID        = "INVALID_ID"
	ErrCodeInvalidFormat    = "INVALID_FORMAT"
	ErrCodeValidationFailed = "VALIDATION_FAILED"

	// Resource errors (404, 409)
	ErrCodeNotFound            = "NOT_FOUND"
	ErrCodeAlreadyExists       = "ALREADY_EXISTS"
	ErrCodeDuplicateKey        = "DUPLICATE_KEY"
	ErrCodeConflict            = "CONFLICT"
	ErrCodeForeignKeyViolation = "FOREIGN_KEY_VIOLATION"

	// Constraint errors (400)
	ErrCodeNotNullViolation = "NOT_NULL_VIOLATION"
	ErrCodeCheckViolation   = "CHECK_VIOLATION"

	// Server errors (500)
	ErrCodeInternalError   = "INTERNAL_ERROR"
	ErrCodeDatabaseError   = "DATABASE_ERROR"
	ErrCodeOperationFailed = "OPERATION_FAILED"

	// Rate limiting (429)
	ErrCodeRateLimited     = "RATE_LIMIT_EXCEEDED"
	ErrCodeTooManyRequests = "TOO_MANY_REQUESTS"

	// Setup/config errors
	ErrCodeSetupRequired     = "SETUP_REQUIRED"
	ErrCodeSetupCompleted    = "SETUP_ALREADY_COMPLETED"
	ErrCodeSetupDisabled     = "SETUP_DISABLED"
	ErrCodeInvalidSetupToken = "INVALID_SETUP_TOKEN"
)

// getRequestID extracts the request ID from the Fiber context.
// It first checks the requestid middleware local, then falls back to the X-Request-ID header.
func getRequestID(c *fiber.Ctx) string {
	if requestID := c.Locals("requestid"); requestID != nil {
		if id, ok := requestID.(string); ok && id != "" {
			return id
		}
	}
	return c.Get("X-Request-ID", "")
}

// ErrorResponse represents a standardized API error response
type ErrorResponse struct {
	Error     string      `json:"error"`
	Code      string      `json:"code,omitempty"`
	Message   string      `json:"message,omitempty"`
	Hint      string      `json:"hint,omitempty"`
	Details   interface{} `json:"details,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
}

// SendError sends a standardized error response with request ID
func SendError(c *fiber.Ctx, statusCode int, errMsg string) error {
	return c.Status(statusCode).JSON(ErrorResponse{
		Error:     errMsg,
		RequestID: getRequestID(c),
	})
}

// SendErrorWithCode sends a standardized error response with error code and request ID
func SendErrorWithCode(c *fiber.Ctx, statusCode int, errMsg string, code string) error {
	return c.Status(statusCode).JSON(ErrorResponse{
		Error:     errMsg,
		Code:      code,
		RequestID: getRequestID(c),
	})
}

// SendErrorWithDetails sends a detailed error response with request ID
func SendErrorWithDetails(c *fiber.Ctx, statusCode int, errMsg string, code string, message string, hint string, details interface{}) error {
	return c.Status(statusCode).JSON(ErrorResponse{
		Error:     errMsg,
		Code:      code,
		Message:   message,
		Hint:      hint,
		Details:   details,
		RequestID: getRequestID(c),
	})
}

// =============================================================================
// Convenience functions for common error patterns
// =============================================================================

// SendBadRequest sends a 400 Bad Request error with the given message and code
func SendBadRequest(c *fiber.Ctx, errMsg string, code string) error {
	return SendErrorWithCode(c, 400, errMsg, code)
}

// SendUnauthorized sends a 401 Unauthorized error
func SendUnauthorized(c *fiber.Ctx, errMsg string, code string) error {
	return SendErrorWithCode(c, 401, errMsg, code)
}

// SendForbidden sends a 403 Forbidden error
func SendForbidden(c *fiber.Ctx, errMsg string, code string) error {
	return SendErrorWithCode(c, 403, errMsg, code)
}

// SendNotFound sends a 404 Not Found error
func SendNotFound(c *fiber.Ctx, errMsg string) error {
	return SendErrorWithCode(c, 404, errMsg, ErrCodeNotFound)
}

// SendConflict sends a 409 Conflict error
func SendConflict(c *fiber.Ctx, errMsg string, code string) error {
	return SendErrorWithCode(c, 409, errMsg, code)
}

// SendInternalError sends a 500 Internal Server Error
func SendInternalError(c *fiber.Ctx, errMsg string) error {
	return SendErrorWithCode(c, 500, errMsg, ErrCodeInternalError)
}

// SendValidationError sends a 400 error for validation failures with details
func SendValidationError(c *fiber.Ctx, errMsg string, details interface{}) error {
	return SendErrorWithDetails(c, 400, errMsg, ErrCodeValidationFailed, "", "", details)
}

// SendMissingAuth sends a 401 for missing authentication
func SendMissingAuth(c *fiber.Ctx) error {
	return SendErrorWithCode(c, 401, "Missing authentication", ErrCodeMissingAuth)
}

// SendInvalidToken sends a 401 for invalid or expired token
func SendInvalidToken(c *fiber.Ctx) error {
	return SendErrorWithCode(c, 401, "Invalid or expired token", ErrCodeInvalidToken)
}

// SendTokenRevoked sends a 401 for revoked token
func SendTokenRevoked(c *fiber.Ctx) error {
	return SendErrorWithCode(c, 401, "Token has been revoked", ErrCodeRevokedToken)
}

// SendInsufficientPermissions sends a 403 for insufficient permissions
func SendInsufficientPermissions(c *fiber.Ctx) error {
	return SendErrorWithCode(c, 403, "Insufficient permissions", ErrCodeInsufficientPermissions)
}

// SendAdminRequired sends a 403 when admin role is required
func SendAdminRequired(c *fiber.Ctx) error {
	return SendErrorWithCode(c, 403, "Admin role required", ErrCodeAdminRequired)
}

// SendInvalidBody sends a 400 for invalid request body
func SendInvalidBody(c *fiber.Ctx) error {
	return SendErrorWithCode(c, 400, "Invalid request body", ErrCodeInvalidBody)
}

// SendMissingField sends a 400 for missing required field
func SendMissingField(c *fiber.Ctx, fieldName string) error {
	return SendErrorWithCode(c, 400, fmt.Sprintf("%s is required", fieldName), ErrCodeMissingField)
}

// SendInvalidID sends a 400 for invalid ID format
func SendInvalidID(c *fiber.Ctx, idName string) error {
	return SendErrorWithCode(c, 400, fmt.Sprintf("Invalid %s", idName), ErrCodeInvalidID)
}

// SendResourceNotFound sends a 404 for a specific resource type
func SendResourceNotFound(c *fiber.Ctx, resourceType string) error {
	return SendErrorWithCode(c, 404, fmt.Sprintf("%s not found", resourceType), ErrCodeNotFound)
}

// SendOperationFailed sends a 500 for a failed operation with context
func SendOperationFailed(c *fiber.Ctx, operation string) error {
	return SendErrorWithCode(c, 500, fmt.Sprintf("Failed to %s", operation), ErrCodeOperationFailed)
}

// SendFeatureDisabled sends a 403 when a feature is disabled
func SendFeatureDisabled(c *fiber.Ctx, feature string) error {
	return SendErrorWithCode(c, 403, fmt.Sprintf("%s is currently disabled", feature), ErrCodeFeatureDisabled)
}

// handleDatabaseError returns an appropriate HTTP error response based on the database error.
// This centralizes error handling logic for all REST operations.
// All responses include the request ID for correlation with logs.
func handleDatabaseError(c *fiber.Ctx, err error, operation string) error {
	errMsg := err.Error()
	requestID := getRequestID(c)

	// Duplicate key violation (unique constraint)
	if strings.Contains(errMsg, "duplicate key") || strings.Contains(errMsg, "unique constraint") {
		return SendErrorWithCode(c, 409, "Record with this value already exists", ErrCodeDuplicateKey)
	}

	// Foreign key constraint violation
	if strings.Contains(errMsg, "foreign key constraint") {
		return SendErrorWithCode(c, 409, "Cannot complete operation due to foreign key constraint", ErrCodeForeignKeyViolation)
	}

	// NOT NULL constraint violation (missing required field)
	if strings.Contains(errMsg, "null value in column") || strings.Contains(errMsg, "not-null constraint") {
		return SendErrorWithCode(c, 400, "Missing required field", ErrCodeNotNullViolation)
	}

	// Invalid input syntax (type mismatch, invalid data)
	if strings.Contains(errMsg, "invalid input syntax") {
		return SendErrorWithCode(c, 400, "Invalid data type provided", ErrCodeInvalidInput)
	}

	// Check constraint violation
	if strings.Contains(errMsg, "check constraint") {
		return SendErrorWithCode(c, 400, "Data violates table constraints", ErrCodeCheckViolation)
	}

	// Generic server error for other cases
	log.Error().
		Err(err).
		Str("operation", operation).
		Str("request_id", requestID).
		Msg("Database operation failed")

	return SendErrorWithCode(c, 500, fmt.Sprintf("Failed to %s", operation), ErrCodeDatabaseError)
}

// isUserAuthenticated checks if the user is authenticated based on RLS context
func isUserAuthenticated(c *fiber.Ctx) bool {
	role := c.Locals("rls_role")
	if role == nil {
		return false
	}
	roleStr, ok := role.(string)
	if !ok {
		return false
	}
	// User is authenticated if they have any role other than "anon"
	return roleStr != "anon" && roleStr != ""
}

// handleRLSViolation returns appropriate error response for RLS policy violations.
// For mutations (INSERT/UPDATE/DELETE), when a query succeeds but returns 0 rows,
// it's likely an RLS policy blocking the operation rather than the record not existing.
// All responses include the request ID for correlation with logs.
func (h *RESTHandler) handleRLSViolation(c *fiber.Ctx, operation string, tableName string) error {
	ctx := c.Context()
	requestID := getRequestID(c)

	// Check if user is authenticated
	authenticated := isUserAuthenticated(c)

	// Log the violation to the audit table
	// Note: This is synchronous to avoid Fiber context issues with goroutines
	// The DB insert is fast enough to not significantly impact response time
	middleware.LogRLSViolation(ctx, h.db, c, operation, tableName)

	if !authenticated {
		// Anonymous users get 401 - need authentication
		log.Warn().
			Str("operation", operation).
			Str("table", tableName).
			Str("role", "anon").
			Str("request_id", requestID).
			Msg("RLS violation: Anonymous user attempted operation")

		return SendErrorWithCode(c, 401, "Authentication required", ErrCodeAuthRequired)
	}

	// Authenticated users get 403 - insufficient permissions
	userID := c.Locals("rls_user_id")
	role := c.Locals("rls_role")

	log.Warn().
		Interface("user_id", userID).
		Interface("role", role).
		Str("operation", operation).
		Str("table", tableName).
		Str("request_id", requestID).
		Msg("RLS violation: Insufficient permissions")

	return SendErrorWithDetails(c, 403, "Insufficient permissions", ErrCodeRLSViolation,
		"Row-level security policy blocks this operation",
		"Verify your authentication and table access policies",
		nil)
}
