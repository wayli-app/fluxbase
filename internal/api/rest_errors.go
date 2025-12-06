package api

import (
	"fmt"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// handleDatabaseError returns an appropriate HTTP error response based on the database error.
// This centralizes error handling logic for all REST operations.
func handleDatabaseError(c *fiber.Ctx, err error, operation string) error {
	errMsg := err.Error()

	// Duplicate key violation (unique constraint)
	if strings.Contains(errMsg, "duplicate key") || strings.Contains(errMsg, "unique constraint") {
		return c.Status(409).JSON(fiber.Map{
			"error": "Record with this value already exists",
		})
	}

	// Foreign key constraint violation
	if strings.Contains(errMsg, "foreign key constraint") {
		return c.Status(409).JSON(fiber.Map{
			"error": "Cannot complete operation due to foreign key constraint",
		})
	}

	// NOT NULL constraint violation (missing required field)
	if strings.Contains(errMsg, "null value in column") || strings.Contains(errMsg, "not-null constraint") {
		return c.Status(400).JSON(fiber.Map{
			"error": "Missing required field",
		})
	}

	// Invalid input syntax (type mismatch, invalid data)
	if strings.Contains(errMsg, "invalid input syntax") {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid data type provided",
		})
	}

	// Check constraint violation
	if strings.Contains(errMsg, "check constraint") {
		return c.Status(400).JSON(fiber.Map{
			"error": "Data violates table constraints",
		})
	}

	// Generic server error for other cases
	log.Error().Err(err).Str("operation", operation).Msg("Database operation failed")
	return c.Status(500).JSON(fiber.Map{
		"error": fmt.Sprintf("Failed to %s", operation),
	})
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
func (h *RESTHandler) handleRLSViolation(c *fiber.Ctx, operation string, tableName string) error {
	ctx := c.Context()

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
			Msg("RLS violation: Anonymous user attempted operation")

		return c.Status(401).JSON(fiber.Map{
			"error": "Authentication required",
			"code":  "AUTHENTICATION_REQUIRED",
		})
	}

	// Authenticated users get 403 - insufficient permissions
	userID := c.Locals("rls_user_id")
	role := c.Locals("rls_role")

	log.Warn().
		Interface("user_id", userID).
		Interface("role", role).
		Str("operation", operation).
		Str("table", tableName).
		Msg("RLS violation: Insufficient permissions")

	return c.Status(403).JSON(fiber.Map{
		"error":   "Insufficient permissions",
		"code":    "RLS_POLICY_VIOLATION",
		"message": "Row-level security policy blocks this operation",
		"hint":    "Verify your authentication and table access policies",
	})
}
