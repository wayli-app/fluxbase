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
