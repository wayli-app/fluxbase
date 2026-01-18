package api

import (
	"fmt"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
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

// handleDatabaseError returns an appropriate HTTP error response based on the database error.
// This centralizes error handling logic for all REST operations.
// All responses include the request ID for correlation with logs.
func handleDatabaseError(c *fiber.Ctx, err error, operation string) error {
	errMsg := err.Error()
	requestID := getRequestID(c)

	// Duplicate key violation (unique constraint)
	if strings.Contains(errMsg, "duplicate key") || strings.Contains(errMsg, "unique constraint") {
		return SendErrorWithCode(c, 409, "Record with this value already exists", "DUPLICATE_KEY")
	}

	// Foreign key constraint violation
	if strings.Contains(errMsg, "foreign key constraint") {
		return SendErrorWithCode(c, 409, "Cannot complete operation due to foreign key constraint", "FOREIGN_KEY_VIOLATION")
	}

	// NOT NULL constraint violation (missing required field)
	if strings.Contains(errMsg, "null value in column") || strings.Contains(errMsg, "not-null constraint") {
		return SendErrorWithCode(c, 400, "Missing required field", "NOT_NULL_VIOLATION")
	}

	// Invalid input syntax (type mismatch, invalid data)
	if strings.Contains(errMsg, "invalid input syntax") {
		return SendErrorWithCode(c, 400, "Invalid data type provided", "INVALID_INPUT")
	}

	// Check constraint violation
	if strings.Contains(errMsg, "check constraint") {
		return SendErrorWithCode(c, 400, "Data violates table constraints", "CHECK_VIOLATION")
	}

	// Generic server error for other cases
	log.Error().
		Err(err).
		Str("operation", operation).
		Str("request_id", requestID).
		Msg("Database operation failed")

	return SendErrorWithCode(c, 500, fmt.Sprintf("Failed to %s", operation), "DATABASE_ERROR")
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

		return SendErrorWithCode(c, 401, "Authentication required", "AUTHENTICATION_REQUIRED")
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

	return SendErrorWithDetails(c, 403, "Insufficient permissions", "RLS_POLICY_VIOLATION",
		"Row-level security policy blocks this operation",
		"Verify your authentication and table access policies",
		nil)
}
