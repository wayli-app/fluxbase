package api

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
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
