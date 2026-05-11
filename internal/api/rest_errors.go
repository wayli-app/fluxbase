package api

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/middleware"

	apperrors "github.com/nimbleflux/fluxbase/internal/errors"
)

const (
	ErrCodeMissingAuth             = apperrors.ErrCodeMissingAuth
	ErrCodeInvalidToken            = apperrors.ErrCodeInvalidToken
	ErrCodeExpiredToken            = apperrors.ErrCodeExpiredToken
	ErrCodeRevokedToken            = apperrors.ErrCodeRevokedToken
	ErrCodeAuthRequired            = apperrors.ErrCodeAuthRequired
	ErrCodeInvalidUserID           = apperrors.ErrCodeInvalidUserID
	ErrCodeAccountLocked           = apperrors.ErrCodeAccountLocked
	ErrCodeInvalidCredentials      = apperrors.ErrCodeInvalidCredentials
	ErrCodeInsufficientPermissions = apperrors.ErrCodeInsufficientPermissions
	ErrCodeAdminRequired           = apperrors.ErrCodeAdminRequired
	ErrCodeInvalidRole             = apperrors.ErrCodeInvalidRole
	ErrCodeRLSViolation            = apperrors.ErrCodeRLSViolation
	ErrCodeAccessDenied            = apperrors.ErrCodeAccessDenied
	ErrCodeFeatureDisabled         = apperrors.ErrCodeFeatureDisabled
	ErrCodeInvalidBody             = apperrors.ErrCodeInvalidBody
	ErrCodeMissingField            = apperrors.ErrCodeMissingField
	ErrCodeInvalidInput            = apperrors.ErrCodeInvalidInput
	ErrCodeInvalidID               = apperrors.ErrCodeInvalidID
	ErrCodeInvalidFormat           = apperrors.ErrCodeInvalidFormat
	ErrCodeValidationFailed        = apperrors.ErrCodeValidationFailed
	ErrCodeNotFound                = apperrors.ErrCodeNotFound
	ErrCodeAlreadyExists           = apperrors.ErrCodeAlreadyExists
	ErrCodeDuplicateKey            = apperrors.ErrCodeDuplicateKey
	ErrCodeConflict                = apperrors.ErrCodeConflict
	ErrCodeForeignKeyViolation     = apperrors.ErrCodeForeignKeyViolation
	ErrCodeNotNullViolation        = apperrors.ErrCodeNotNullViolation
	ErrCodeCheckViolation          = apperrors.ErrCodeCheckViolation
	ErrCodeInternalError           = apperrors.ErrCodeInternalError
	ErrCodeDatabaseError           = apperrors.ErrCodeDatabaseError
	ErrCodeOperationFailed         = apperrors.ErrCodeOperationFailed
	ErrCodeRateLimited             = apperrors.ErrCodeRateLimited
	ErrCodeTooManyRequests         = apperrors.ErrCodeTooManyRequests
	ErrCodeSetupRequired           = apperrors.ErrCodeSetupRequired
	ErrCodeSetupCompleted          = apperrors.ErrCodeSetupCompleted
	ErrCodeSetupDisabled           = apperrors.ErrCodeSetupDisabled
	ErrCodeInvalidSetupToken       = apperrors.ErrCodeInvalidSetupToken
)

type ErrorResponse = apperrors.ErrorResponse

func getRequestID(c fiber.Ctx) string {
	return apperrors.GetRequestID(c)
}

func SendAppError(c fiber.Ctx, err error) error {
	return apperrors.SendAppError(c, err)
}

func SendError(c fiber.Ctx, statusCode int, errMsg string) error {
	return apperrors.SendError(c, statusCode, errMsg)
}

func SendErrorWithCode(c fiber.Ctx, statusCode int, errMsg string, code string) error {
	return apperrors.SendErrorWithCode(c, statusCode, errMsg, code)
}

func SendErrorWithDetails(c fiber.Ctx, statusCode int, errMsg string, code string, message string, hint string, details interface{}) error {
	return apperrors.SendErrorWithDetails(c, statusCode, errMsg, code, message, hint, details)
}

func SendBadRequest(c fiber.Ctx, errMsg string, code string) error {
	return apperrors.SendBadRequest(c, errMsg, code)
}

func SendUnauthorized(c fiber.Ctx, errMsg string, code string) error {
	return apperrors.SendUnauthorized(c, errMsg, code)
}

func SendForbidden(c fiber.Ctx, errMsg string, code string) error {
	return apperrors.SendForbidden(c, errMsg, code)
}

func SendNotFound(c fiber.Ctx, errMsg string) error {
	return apperrors.SendNotFound(c, errMsg)
}

func SendConflict(c fiber.Ctx, errMsg string, code string) error {
	return apperrors.SendConflict(c, errMsg, code)
}

func SendInternalError(c fiber.Ctx, errMsg string) error {
	return apperrors.SendInternalError(c, errMsg)
}

func SendValidationError(c fiber.Ctx, errMsg string, details interface{}) error {
	return apperrors.SendValidationError(c, errMsg, details)
}

func SendMissingAuth(c fiber.Ctx) error {
	return apperrors.SendMissingAuth(c)
}

func SendInvalidToken(c fiber.Ctx) error {
	return apperrors.SendInvalidToken(c)
}

func SendTokenRevoked(c fiber.Ctx) error {
	return apperrors.SendTokenRevoked(c)
}

func SendInsufficientPermissions(c fiber.Ctx) error {
	return apperrors.SendInsufficientPermissions(c)
}

func SendAdminRequired(c fiber.Ctx) error {
	return apperrors.SendAdminRequired(c)
}

func SendInvalidBody(c fiber.Ctx) error {
	return apperrors.SendInvalidBody(c)
}

func SendMissingField(c fiber.Ctx, fieldName string) error {
	return apperrors.SendMissingField(c, fieldName)
}

func SendInvalidID(c fiber.Ctx, idName string) error {
	return apperrors.SendInvalidID(c, idName)
}

func SendResourceNotFound(c fiber.Ctx, resourceType string) error {
	return apperrors.SendResourceNotFound(c, resourceType)
}

func SendOperationFailed(c fiber.Ctx, operation string) error {
	return apperrors.SendOperationFailed(c, operation)
}

func SendFeatureDisabled(c fiber.Ctx, feature string) error {
	return apperrors.SendFeatureDisabled(c, feature)
}

func SendNotInitialized(c fiber.Ctx, service string) error {
	return apperrors.SendNotInitialized(c, service)
}

func SendServiceUnavailable(c fiber.Ctx, msg string) error {
	return apperrors.SendServiceUnavailable(c, msg)
}

func handleDatabaseError(c fiber.Ctx, err error, operation string) error {
	errMsg := err.Error()
	requestID := getRequestID(c)

	if strings.Contains(errMsg, "duplicate key") || strings.Contains(errMsg, "unique constraint") {
		return SendErrorWithCode(c, 409, "Record with this value already exists", ErrCodeDuplicateKey)
	}

	if strings.Contains(errMsg, "foreign key constraint") {
		return SendErrorWithCode(c, 409, "Cannot complete operation due to foreign key constraint", ErrCodeForeignKeyViolation)
	}

	if strings.Contains(errMsg, "null value in column") || strings.Contains(errMsg, "not-null constraint") {
		return SendErrorWithCode(c, 400, "Missing required field", ErrCodeNotNullViolation)
	}

	if strings.Contains(errMsg, "invalid input syntax") {
		return SendErrorWithCode(c, 400, "Invalid data type provided", ErrCodeInvalidInput)
	}

	if strings.Contains(errMsg, "check constraint") {
		return SendErrorWithCode(c, 400, "Data violates table constraints", ErrCodeCheckViolation)
	}

	log.Error().
		Err(err).
		Str("operation", operation).
		Str("request_id", requestID).
		Msg("Database operation failed")

	return SendErrorWithCode(c, 500, fmt.Sprintf("Failed to %s", operation), ErrCodeDatabaseError)
}

func isUserAuthenticated(c fiber.Ctx) bool {
	role := c.Locals("rls_role")
	if role == nil {
		return false
	}
	roleStr, ok := role.(string)
	if !ok {
		return false
	}
	return roleStr != "anon" && roleStr != ""
}

func (h *RESTHandler) handleRLSViolation(c fiber.Ctx, operation string, tableName string) error {
	ctx := c.RequestCtx()
	requestID := getRequestID(c)

	authenticated := isUserAuthenticated(c)

	middleware.LogRLSViolation(ctx, h.db, c, operation, tableName)

	if !authenticated {
		log.Warn().
			Str("operation", operation).
			Str("table", tableName).
			Str("role", "anon").
			Str("request_id", requestID).
			Msg("RLS violation: Anonymous user attempted operation")

		return SendErrorWithCode(c, 401, "Authentication required", ErrCodeAuthRequired)
	}

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
