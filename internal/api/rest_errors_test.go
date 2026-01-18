package api

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleDatabaseError(t *testing.T) {
	testCases := []struct {
		name           string
		err            error
		operation      string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "duplicate key error",
			err:            errors.New("duplicate key value violates unique constraint"),
			operation:      "create",
			expectedStatus: 409,
			expectedError:  "Record with this value already exists",
		},
		{
			name:           "unique constraint error",
			err:            errors.New("ERROR: unique constraint violation on email"),
			operation:      "update",
			expectedStatus: 409,
			expectedError:  "Record with this value already exists",
		},
		{
			name:           "foreign key constraint error",
			err:            errors.New("foreign key constraint violation on user_id"),
			operation:      "delete",
			expectedStatus: 409,
			expectedError:  "Cannot complete operation due to foreign key constraint",
		},
		{
			name:           "null value in column error",
			err:            errors.New("null value in column 'name' violates not-null constraint"),
			operation:      "create",
			expectedStatus: 400,
			expectedError:  "Missing required field",
		},
		{
			name:           "not-null constraint error",
			err:            errors.New("ERROR: not-null constraint violated for email"),
			operation:      "insert",
			expectedStatus: 400,
			expectedError:  "Missing required field",
		},
		{
			name:           "invalid input syntax error",
			err:            errors.New("invalid input syntax for type integer"),
			operation:      "create",
			expectedStatus: 400,
			expectedError:  "Invalid data type provided",
		},
		{
			name:           "check constraint error",
			err:            errors.New("check constraint violation on age"),
			operation:      "update",
			expectedStatus: 400,
			expectedError:  "Data violates table constraints",
		},
		{
			name:           "generic error",
			err:            errors.New("some unknown database error"),
			operation:      "fetch records",
			expectedStatus: 500,
			expectedError:  "Failed to fetch records",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()

			// Create a test handler that uses handleDatabaseError
			app.Get("/test", func(c *fiber.Ctx) error {
				return handleDatabaseError(c, tc.err, tc.operation)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedStatus, resp.StatusCode)
		})
	}
}

func TestIsUserAuthenticated(t *testing.T) {
	testCases := []struct {
		name     string
		role     interface{}
		expected bool
	}{
		{
			name:     "authenticated user",
			role:     "authenticated",
			expected: true,
		},
		{
			name:     "service role",
			role:     "service_role",
			expected: true,
		},
		{
			name:     "admin role",
			role:     "admin",
			expected: true,
		},
		{
			name:     "anonymous user",
			role:     "anon",
			expected: false,
		},
		{
			name:     "empty role",
			role:     "",
			expected: false,
		},
		{
			name:     "nil role",
			role:     nil,
			expected: false,
		},
		{
			name:     "wrong type (int)",
			role:     123,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()

			var result bool
			app.Get("/test", func(c *fiber.Ctx) error {
				if tc.role != nil {
					c.Locals("rls_role", tc.role)
				}
				result = isUserAuthenticated(c)
				return c.SendStatus(200)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			_, err := app.Test(req)
			require.NoError(t, err)

			assert.Equal(t, tc.expected, result)
		})
	}
}

// Tests for request ID / correlation ID support

func TestGetRequestID_FromLocals(t *testing.T) {
	app := fiber.New()

	var result string
	app.Get("/test", func(c *fiber.Ctx) error {
		c.Locals("requestid", "test-request-id-123")
		result = getRequestID(c)
		return c.SendStatus(200)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	_, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, "test-request-id-123", result)
}

func TestGetRequestID_FromHeader(t *testing.T) {
	app := fiber.New()

	var result string
	app.Get("/test", func(c *fiber.Ctx) error {
		result = getRequestID(c)
		return c.SendStatus(200)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "header-request-id-456")
	_, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, "header-request-id-456", result)
}

func TestGetRequestID_LocalsPreferred(t *testing.T) {
	app := fiber.New()

	var result string
	app.Get("/test", func(c *fiber.Ctx) error {
		c.Locals("requestid", "local-id")
		result = getRequestID(c)
		return c.SendStatus(200)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "header-id") // Should be ignored
	_, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, "local-id", result)
}

func TestGetRequestID_Empty(t *testing.T) {
	app := fiber.New()

	var result string
	app.Get("/test", func(c *fiber.Ctx) error {
		result = getRequestID(c)
		return c.SendStatus(200)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	_, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, "", result)
}

func TestSendError_IncludesRequestID(t *testing.T) {
	app := fiber.New()
	app.Use(requestid.New())

	app.Get("/test", func(c *fiber.Ctx) error {
		return SendError(c, 400, "Bad request")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var response ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "Bad request", response.Error)
	assert.NotEmpty(t, response.RequestID) // Should have a generated request ID
}

func TestSendErrorWithCode_IncludesRequestIDAndCode(t *testing.T) {
	app := fiber.New()
	app.Use(requestid.New())

	app.Get("/test", func(c *fiber.Ctx) error {
		return SendErrorWithCode(c, 404, "Not found", "NOT_FOUND")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)

	var response ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "Not found", response.Error)
	assert.Equal(t, "NOT_FOUND", response.Code)
	assert.NotEmpty(t, response.RequestID)
}

func TestSendErrorWithDetails_FullResponse(t *testing.T) {
	app := fiber.New()
	app.Use(requestid.New())

	app.Get("/test", func(c *fiber.Ctx) error {
		details := map[string]string{"field": "email", "reason": "invalid format"}
		return SendErrorWithDetails(c, 422, "Validation failed", "VALIDATION_ERROR", "The request body contains invalid data", "Check the 'details' field for more info", details)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 422, resp.StatusCode)

	var response ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "Validation failed", response.Error)
	assert.Equal(t, "VALIDATION_ERROR", response.Code)
	assert.Equal(t, "The request body contains invalid data", response.Message)
	assert.Equal(t, "Check the 'details' field for more info", response.Hint)
	assert.NotNil(t, response.Details)
	assert.NotEmpty(t, response.RequestID)
}

func TestSendError_UsesProvidedRequestID(t *testing.T) {
	app := fiber.New()

	app.Get("/test", func(c *fiber.Ctx) error {
		c.Locals("requestid", "custom-id-xyz")
		return SendError(c, 500, "Server error")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	var response ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "custom-id-xyz", response.RequestID)
}

func TestHandleDatabaseError_IncludesRequestID(t *testing.T) {
	app := fiber.New()
	app.Use(requestid.New())

	app.Get("/test", func(c *fiber.Ctx) error {
		return handleDatabaseError(c, errors.New("duplicate key violation"), "create")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 409, resp.StatusCode)

	var response ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "Record with this value already exists", response.Error)
	assert.Equal(t, "DUPLICATE_KEY", response.Code)
	assert.NotEmpty(t, response.RequestID)
}

// =============================================================================
// Tests for convenience error functions
// =============================================================================

func TestSendBadRequest(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return SendBadRequest(c, "Invalid parameter", ErrCodeInvalidInput)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var response ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "Invalid parameter", response.Error)
	assert.Equal(t, ErrCodeInvalidInput, response.Code)
}

func TestSendUnauthorized(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return SendUnauthorized(c, "Token expired", ErrCodeExpiredToken)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode)

	var response ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "Token expired", response.Error)
	assert.Equal(t, ErrCodeExpiredToken, response.Code)
}

func TestSendForbidden(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return SendForbidden(c, "Access denied", ErrCodeAccessDenied)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode)

	var response ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "Access denied", response.Error)
	assert.Equal(t, ErrCodeAccessDenied, response.Code)
}

func TestSendNotFound(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return SendNotFound(c, "Resource not found")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)

	var response ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "Resource not found", response.Error)
	assert.Equal(t, ErrCodeNotFound, response.Code)
}

func TestSendConflict(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return SendConflict(c, "Resource already exists", ErrCodeAlreadyExists)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 409, resp.StatusCode)

	var response ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "Resource already exists", response.Error)
	assert.Equal(t, ErrCodeAlreadyExists, response.Code)
}

func TestSendInternalError(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return SendInternalError(c, "Something went wrong")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)

	var response ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "Something went wrong", response.Error)
	assert.Equal(t, ErrCodeInternalError, response.Code)
}

func TestSendValidationError(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		details := map[string]string{"email": "invalid format"}
		return SendValidationError(c, "Validation failed", details)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var response ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "Validation failed", response.Error)
	assert.Equal(t, ErrCodeValidationFailed, response.Code)
	assert.NotNil(t, response.Details)
}

func TestSendMissingAuth(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return SendMissingAuth(c)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode)

	var response ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "Missing authentication", response.Error)
	assert.Equal(t, ErrCodeMissingAuth, response.Code)
}

func TestSendInvalidToken(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return SendInvalidToken(c)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode)

	var response ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "Invalid or expired token", response.Error)
	assert.Equal(t, ErrCodeInvalidToken, response.Code)
}

func TestSendTokenRevoked(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return SendTokenRevoked(c)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode)

	var response ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "Token has been revoked", response.Error)
	assert.Equal(t, ErrCodeRevokedToken, response.Code)
}

func TestSendInsufficientPermissions(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return SendInsufficientPermissions(c)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode)

	var response ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "Insufficient permissions", response.Error)
	assert.Equal(t, ErrCodeInsufficientPermissions, response.Code)
}

func TestSendAdminRequired(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return SendAdminRequired(c)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode)

	var response ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "Admin role required", response.Error)
	assert.Equal(t, ErrCodeAdminRequired, response.Code)
}

func TestSendInvalidBody(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return SendInvalidBody(c)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var response ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "Invalid request body", response.Error)
	assert.Equal(t, ErrCodeInvalidBody, response.Code)
}

func TestSendMissingField(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return SendMissingField(c, "email")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var response ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "email is required", response.Error)
	assert.Equal(t, ErrCodeMissingField, response.Code)
}

func TestSendInvalidID(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return SendInvalidID(c, "user ID")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var response ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "Invalid user ID", response.Error)
	assert.Equal(t, ErrCodeInvalidID, response.Code)
}

func TestSendResourceNotFound(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return SendResourceNotFound(c, "User")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)

	var response ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "User not found", response.Error)
	assert.Equal(t, ErrCodeNotFound, response.Code)
}

func TestSendOperationFailed(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return SendOperationFailed(c, "create user")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)

	var response ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "Failed to create user", response.Error)
	assert.Equal(t, ErrCodeOperationFailed, response.Code)
}

func TestSendFeatureDisabled(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return SendFeatureDisabled(c, "User registration")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode)

	var response ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "User registration is currently disabled", response.Error)
	assert.Equal(t, ErrCodeFeatureDisabled, response.Code)
}

// =============================================================================
// Tests for error code constants
// =============================================================================

func TestErrorCodeConstants(t *testing.T) {
	// Verify error code constants are defined correctly
	tests := []struct {
		constant string
		expected string
	}{
		{ErrCodeMissingAuth, "MISSING_AUTHENTICATION"},
		{ErrCodeInvalidToken, "INVALID_TOKEN"},
		{ErrCodeExpiredToken, "EXPIRED_TOKEN"},
		{ErrCodeRevokedToken, "REVOKED_TOKEN"},
		{ErrCodeAuthRequired, "AUTHENTICATION_REQUIRED"},
		{ErrCodeInvalidCredentials, "INVALID_CREDENTIALS"},
		{ErrCodeAccountLocked, "ACCOUNT_LOCKED"},
		{ErrCodeInsufficientPermissions, "INSUFFICIENT_PERMISSIONS"},
		{ErrCodeAdminRequired, "ADMIN_REQUIRED"},
		{ErrCodeInvalidRole, "INVALID_ROLE"},
		{ErrCodeRLSViolation, "RLS_POLICY_VIOLATION"},
		{ErrCodeAccessDenied, "ACCESS_DENIED"},
		{ErrCodeFeatureDisabled, "FEATURE_DISABLED"},
		{ErrCodeInvalidBody, "INVALID_REQUEST_BODY"},
		{ErrCodeMissingField, "MISSING_REQUIRED_FIELD"},
		{ErrCodeInvalidInput, "INVALID_INPUT"},
		{ErrCodeInvalidID, "INVALID_ID"},
		{ErrCodeValidationFailed, "VALIDATION_FAILED"},
		{ErrCodeNotFound, "NOT_FOUND"},
		{ErrCodeAlreadyExists, "ALREADY_EXISTS"},
		{ErrCodeDuplicateKey, "DUPLICATE_KEY"},
		{ErrCodeConflict, "CONFLICT"},
		{ErrCodeForeignKeyViolation, "FOREIGN_KEY_VIOLATION"},
		{ErrCodeNotNullViolation, "NOT_NULL_VIOLATION"},
		{ErrCodeCheckViolation, "CHECK_VIOLATION"},
		{ErrCodeInternalError, "INTERNAL_ERROR"},
		{ErrCodeDatabaseError, "DATABASE_ERROR"},
		{ErrCodeOperationFailed, "OPERATION_FAILED"},
		{ErrCodeRateLimited, "RATE_LIMIT_EXCEEDED"},
		{ErrCodeSetupRequired, "SETUP_REQUIRED"},
		{ErrCodeSetupCompleted, "SETUP_ALREADY_COMPLETED"},
		{ErrCodeSetupDisabled, "SETUP_DISABLED"},
		{ErrCodeInvalidSetupToken, "INVALID_SETUP_TOKEN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.constant)
		})
	}
}
