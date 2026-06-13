package errors

import (
	stderrors "errors"
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

const (
	ErrCodeMissingAuth        = "MISSING_AUTHENTICATION"
	ErrCodeInvalidToken       = "INVALID_TOKEN"
	ErrCodeExpiredToken       = "EXPIRED_TOKEN"
	ErrCodeRevokedToken       = "REVOKED_TOKEN"
	ErrCodeAuthRequired       = "AUTHENTICATION_REQUIRED"
	ErrCodeInvalidUserID      = "INVALID_USER_ID"
	ErrCodeAccountLocked      = "ACCOUNT_LOCKED"
	ErrCodeInvalidCredentials = "INVALID_CREDENTIALS"

	ErrCodeInsufficientPermissions = "INSUFFICIENT_PERMISSIONS"
	ErrCodeAdminRequired           = "ADMIN_REQUIRED"
	ErrCodeInvalidRole             = "INVALID_ROLE"
	ErrCodeRLSViolation            = "RLS_POLICY_VIOLATION"
	ErrCodeAccessDenied            = "ACCESS_DENIED"
	ErrCodeFeatureDisabled         = "FEATURE_DISABLED"

	ErrCodeInvalidBody      = "INVALID_REQUEST_BODY"
	ErrCodeMissingField     = "MISSING_REQUIRED_FIELD"
	ErrCodeInvalidInput     = "INVALID_INPUT"
	ErrCodeInvalidID        = "INVALID_ID"
	ErrCodeInvalidFormat    = "INVALID_FORMAT"
	ErrCodeValidationFailed = "VALIDATION_FAILED"

	ErrCodeNotFound            = "NOT_FOUND"
	ErrCodeAlreadyExists       = "ALREADY_EXISTS"
	ErrCodeDuplicateKey        = "DUPLICATE_KEY"
	ErrCodeConflict            = "CONFLICT"
	ErrCodeForeignKeyViolation = "FOREIGN_KEY_VIOLATION"

	ErrCodeNotNullViolation = "NOT_NULL_VIOLATION"
	ErrCodeCheckViolation   = "CHECK_VIOLATION"

	ErrCodeInternalError   = "INTERNAL_ERROR"
	ErrCodeDatabaseError   = "DATABASE_ERROR"
	ErrCodeOperationFailed = "OPERATION_FAILED"

	ErrCodeRateLimited     = "RATE_LIMIT_EXCEEDED"
	ErrCodeTooManyRequests = "TOO_MANY_REQUESTS"

	ErrCodePayloadTooLarge = "PAYLOAD_TOO_LARGE"
	ErrCodeJsonTooDeep     = "JSON_TOO_DEEP"

	ErrCodeBranchNotReady    = "BRANCH_NOT_READY"
	ErrCodeBranchNotFound    = "BRANCH_NOT_FOUND"
	ErrCodeBranchingDisabled = "BRANCHING_DISABLED"
	ErrCodeAccessCheckFailed = "ACCESS_CHECK_FAILED"
	ErrCodePoolError         = "POOL_ERROR"
	ErrCodeTenantNotFound    = "TENANT_NOT_FOUND"

	ErrCodeIdempotencyKeyTooLong          = "IDEMPOTENCY_KEY_TOO_LONG"
	ErrCodeIdempotencyKeyMismatch         = "IDEMPOTENCY_KEY_MISMATCH"
	ErrCodeIdempotencyRequestBodyMismatch = "IDEMPOTENCY_REQUEST_BODY_MISMATCH"
	ErrCodeIdempotencyRequestInProgress   = "IDEMPOTENCY_REQUEST_IN_PROGRESS"

	ErrCodeMissingClientKey   = "MISSING_CLIENT_KEY"
	ErrCodeClientKeyRevoked   = "CLIENT_KEY_REVOKED"
	ErrCodeClientKeyExpired   = "CLIENT_KEY_EXPIRED"
	ErrCodeClientKeysDisabled = "CLIENT_KEYS_DISABLED"
	ErrCodeInvalidServiceKey  = "INVALID_SERVICE_KEY"

	ErrCodeFunctionNotFound   = "FUNCTION_NOT_FOUND"
	ErrCodeFunctionDisabled   = "FUNCTION_DISABLED"
	ErrCodeLoggingUnavailable = "LOGGING_SERVICE_UNAVAILABLE"

	ErrCodeCsrfTokenRequired         = "CSRF_TOKEN_REQUIRED"
	ErrCodeCsrfTokenExpired          = "CSRF_TOKEN_EXPIRED"
	ErrCodeCsrfTokenValidationFailed = "CSRF_TOKEN_VALIDATION_FAILED"

	ErrCodeSetupRequired     = "SETUP_REQUIRED"
	ErrCodeSetupCompleted    = "SETUP_ALREADY_COMPLETED"
	ErrCodeSetupDisabled     = "SETUP_DISABLED"
	ErrCodeInvalidSetupToken = "INVALID_SETUP_TOKEN"
)

type ErrorResponse struct {
	Error     string      `json:"error"`
	Code      string      `json:"code,omitempty"`
	Message   string      `json:"message,omitempty"`
	Hint      string      `json:"hint,omitempty"`
	Details   interface{} `json:"details,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
}

func GetRequestID(c fiber.Ctx) string {
	if requestID := requestid.FromContext(c); requestID != "" {
		return requestID
	}
	return c.Get("X-Request-ID", "")
}

func SendAppError(c fiber.Ctx, err error) error {
	var appErr *AppError
	if stderrors.As(err, &appErr) {
		return SendErrorWithCode(c, appErr.HTTPStatus(), appErr.Message(), appErr.Code())
	}
	return SendErrorWithCode(c, 500, "Internal Server Error", ErrCodeInternalError)
}

func SendError(c fiber.Ctx, statusCode int, errMsg string) error {
	return c.Status(statusCode).JSON(ErrorResponse{
		Error:     errMsg,
		RequestID: GetRequestID(c),
	})
}

func SendErrorWithCode(c fiber.Ctx, statusCode int, errMsg string, code string) error {
	return c.Status(statusCode).JSON(ErrorResponse{
		Error:     errMsg,
		Code:      code,
		RequestID: GetRequestID(c),
	})
}

func SendErrorWithDetails(c fiber.Ctx, statusCode int, errMsg string, code string, message string, hint string, details interface{}) error {
	return c.Status(statusCode).JSON(ErrorResponse{
		Error:     errMsg,
		Code:      code,
		Message:   message,
		Hint:      hint,
		Details:   details,
		RequestID: GetRequestID(c),
	})
}

func SendBadRequest(c fiber.Ctx, errMsg string, code string) error {
	return SendErrorWithCode(c, 400, errMsg, code)
}

func SendUnauthorized(c fiber.Ctx, errMsg string, code string) error {
	return SendErrorWithCode(c, 401, errMsg, code)
}

func SendForbidden(c fiber.Ctx, errMsg string, code string) error {
	return SendErrorWithCode(c, 403, errMsg, code)
}

func SendNotFound(c fiber.Ctx, errMsg string) error {
	return SendErrorWithCode(c, 404, errMsg, ErrCodeNotFound)
}

func SendConflict(c fiber.Ctx, errMsg string, code string) error {
	return SendErrorWithCode(c, 409, errMsg, code)
}

func SendInternalError(c fiber.Ctx, errMsg string) error {
	return SendErrorWithCode(c, 500, errMsg, ErrCodeInternalError)
}

func SendValidationError(c fiber.Ctx, errMsg string, details interface{}) error {
	return SendErrorWithDetails(c, 400, errMsg, ErrCodeValidationFailed, "", "", details)
}

func SendMissingAuth(c fiber.Ctx) error {
	return SendErrorWithCode(c, 401, "Missing authentication", ErrCodeMissingAuth)
}

func SendInvalidToken(c fiber.Ctx) error {
	return SendErrorWithCode(c, 401, "Invalid or expired token", ErrCodeInvalidToken)
}

func SendTokenRevoked(c fiber.Ctx) error {
	return SendErrorWithCode(c, 401, "Token has been revoked", ErrCodeRevokedToken)
}

func SendInsufficientPermissions(c fiber.Ctx) error {
	return SendErrorWithCode(c, 403, "Insufficient permissions", ErrCodeInsufficientPermissions)
}

func SendAdminRequired(c fiber.Ctx) error {
	return SendErrorWithCode(c, 403, "Admin role required", ErrCodeAdminRequired)
}

func SendInvalidBody(c fiber.Ctx) error {
	return SendErrorWithCode(c, 400, "Invalid request body", ErrCodeInvalidBody)
}

func SendMissingField(c fiber.Ctx, fieldName string) error {
	return SendErrorWithCode(c, 400, fmt.Sprintf("%s is required", fieldName), ErrCodeMissingField)
}

func SendInvalidID(c fiber.Ctx, idName string) error {
	return SendErrorWithCode(c, 400, fmt.Sprintf("Invalid %s", idName), ErrCodeInvalidID)
}

func SendResourceNotFound(c fiber.Ctx, resourceType string) error {
	return SendErrorWithCode(c, 404, fmt.Sprintf("%s not found", resourceType), ErrCodeNotFound)
}

func SendOperationFailed(c fiber.Ctx, operation string) error {
	return SendErrorWithCode(c, 500, fmt.Sprintf("Failed to %s", operation), ErrCodeOperationFailed)
}

func SendFeatureDisabled(c fiber.Ctx, feature string) error {
	return SendErrorWithCode(c, 403, fmt.Sprintf("%s is currently disabled", feature), ErrCodeFeatureDisabled)
}

func SendNotInitialized(c fiber.Ctx, service string) error {
	return SendErrorWithCode(c, fiber.StatusServiceUnavailable, service+" not initialized", "NOT_INITIALIZED")
}

func SendServiceUnavailable(c fiber.Ctx, msg string) error {
	return SendErrorWithCode(c, fiber.StatusServiceUnavailable, msg, "SERVICE_UNAVAILABLE")
}

func SendErrorWithHint(c fiber.Ctx, statusCode int, errMsg string, code string, hint string) error {
	return SendErrorWithDetails(c, statusCode, errMsg, code, "", hint, nil)
}

func SendSuccess(c fiber.Ctx, message string) error {
	return c.JSON(fiber.Map{
		"success": true,
		"message": message,
	})
}

func SendData(c fiber.Ctx, key string, data interface{}) error {
	return c.JSON(fiber.Map{
		key: data,
	})
}

func SendDataWithCount(c fiber.Ctx, key string, data interface{}, count int) error {
	return c.JSON(fiber.Map{
		key:     data,
		"count": count,
	})
}

func SendPaginated(c fiber.Ctx, key string, data interface{}, total, limit, offset int) error {
	return c.JSON(fiber.Map{
		key:      data,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func SendPaginatedWithMore(c fiber.Ctx, key string, data interface{}, total, limit, offset int, hasMore bool) error {
	return c.JSON(fiber.Map{
		key:       data,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
		"hasMore": hasMore,
	})
}

func SendAffected(c fiber.Ctx, affected int) error {
	return c.JSON(fiber.Map{
		"affected": affected,
	})
}

func SendStatusMessage(c fiber.Ctx, status string, extra fiber.Map) error {
	result := fiber.Map{"status": status}
	for k, v := range extra {
		result[k] = v
	}
	return c.JSON(result)
}

func SendSyncResult(c fiber.Ctx, result interface{}) error {
	return c.JSON(result)
}

func ValueOr[T any](ptr *T, defaultVal T) T {
	if ptr != nil {
		return *ptr
	}
	return defaultVal
}

func ToString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
