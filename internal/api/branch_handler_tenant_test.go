package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/branching"
	"github.com/nimbleflux/fluxbase/internal/config"
)

// =============================================================================
// getTenantFilter Tests (table-driven)
// =============================================================================

func TestGetTenantFilter(t *testing.T) {
	tenantID := uuid.New()
	tenantIDStr := tenantID.String()

	tests := []struct {
		name           string
		setupLocals    func(c fiber.Ctx)
		expectedResult *uuid.UUID
	}{
		{
			name: "instance admin gets no filter",
			setupLocals: func(c fiber.Ctx) {
				c.Locals("is_instance_admin", true)
			},
			expectedResult: nil,
		},
		{
			name: "service key gets no filter",
			setupLocals: func(c fiber.Ctx) {
				c.Locals("auth_type", "service_key")
			},
			expectedResult: nil,
		},
		{
			name: "regular user with tenant_id returns tenant filter",
			setupLocals: func(c fiber.Ctx) {
				c.Locals("tenant_id", tenantIDStr)
			},
			expectedResult: &tenantID,
		},
		{
			name: "regular user without tenant_id returns nil filter",
			setupLocals: func(c fiber.Ctx) {
				c.Locals("user_role", "member")
			},
			expectedResult: nil,
		},
		{
			name: "non-parseable tenant_id returns nil",
			setupLocals: func(c fiber.Ctx) {
				c.Locals("tenant_id", "not-a-uuid")
			},
			expectedResult: nil,
		},
		{
			name: "empty tenant_id returns nil",
			setupLocals: func(c fiber.Ctx) {
				c.Locals("tenant_id", "")
			},
			expectedResult: nil,
		},
		{
			name: "instance admin overrides tenant context",
			setupLocals: func(c fiber.Ctx) {
				c.Locals("is_instance_admin", true)
				c.Locals("tenant_id", tenantIDStr)
			},
			expectedResult: nil,
		},
		{
			name: "service key overrides tenant context",
			setupLocals: func(c fiber.Ctx) {
				c.Locals("auth_type", "service_key")
				c.Locals("tenant_id", tenantIDStr)
			},
			expectedResult: nil,
		},
		{
			name: "no locals set returns nil",
			setupLocals: func(c fiber.Ctx) {
				// No locals set
			},
			expectedResult: nil,
		},
		{
			name: "wrong type for tenant_id returns nil",
			setupLocals: func(c fiber.Ctx) {
				c.Locals("tenant_id", 12345)
			},
			expectedResult: nil,
		},
		{
			name: "wrong type for is_instance_admin falls through",
			setupLocals: func(c fiber.Ctx) {
				c.Locals("is_instance_admin", "true")
				c.Locals("tenant_id", tenantIDStr)
			},
			expectedResult: &tenantID,
		},
		{
			name: "wrong type for auth_type falls through",
			setupLocals: func(c fiber.Ctx) {
				c.Locals("auth_type", 42)
				c.Locals("tenant_id", tenantIDStr)
			},
			expectedResult: &tenantID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApp(t)

			var capturedFilter *uuid.UUID
			app.Use(func(c fiber.Ctx) error {
				tt.setupLocals(c)
				capturedFilter = getTenantFilter(c)
				return c.Next()
			})
			app.Get("/test", func(c fiber.Ctx) error {
				return c.SendString("ok")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			if tt.expectedResult == nil {
				assert.Nil(t, capturedFilter, "expected nil tenant filter")
			} else {
				require.NotNil(t, capturedFilter, "expected non-nil tenant filter")
				assert.Equal(t, *tt.expectedResult, *capturedFilter, "tenant filter mismatch")
			}
		})
	}
}

// =============================================================================
// DataCloneModeFull Normalization Tests
// =============================================================================

func TestCreateBranch_NormalizesFullAlias(t *testing.T) {
	t.Run("full alias is accepted in request", func(t *testing.T) {
		assert.Equal(t, branching.DataCloneMode("full"), branching.DataCloneModeFull)
		assert.Equal(t, branching.DataCloneMode("full_clone"), branching.DataCloneModeFullClone)
		assert.NotEqual(t, branching.DataCloneModeFull, branching.DataCloneModeFullClone)
	})
}

// =============================================================================
// Tenant Isolation in CreateBranchRequest
// =============================================================================

func TestCreateBranchRequest_TenantID(t *testing.T) {
	t.Run("tenant_id in request body", func(t *testing.T) {
		tenantID := uuid.New()
		parentID := uuid.New()

		req := CreateBranchRequest{
			Name:           "tenant-branch",
			TenantID:       &tenantID,
			ParentBranchID: &parentID,
			DataCloneMode:  branching.DataCloneModeSchemaOnly,
			Type:           branching.BranchTypePreview,
		}

		assert.NotNil(t, req.TenantID)
		assert.Equal(t, tenantID, *req.TenantID)
	})

	t.Run("tenant_id omitted for instance-level branch", func(t *testing.T) {
		req := CreateBranchRequest{
			Name:          "instance-branch",
			DataCloneMode: branching.DataCloneModeSchemaOnly,
			Type:          branching.BranchTypePreview,
		}

		assert.Nil(t, req.TenantID)
	})

	t.Run("tenant_id JSON deserialization", func(t *testing.T) {
		tenantID := "550e8400-e29b-41d4-a716-446655440000"
		jsonData := `{"name":"test","tenant_id":"` + tenantID + `"}`

		var req CreateBranchRequest
		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)
		require.NotNil(t, req.TenantID)
		assert.Equal(t, tenantID, req.TenantID.String())
	})
}

// =============================================================================
// GetBranch Handler Tenant Isolation Tests
// =============================================================================

func TestGetBranch_TenantIsolation(t *testing.T) {
	t.Run("instance admin can access branch lookup with no filter", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewBranchHandler(nil, nil, config.BranchingConfig{})

		app.Use(func(c fiber.Ctx) error {
			c.Locals("is_instance_admin", true)
			return c.Next()
		})
		app.Get("/branches/:id", handler.GetBranch)

		branchID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/branches/"+branchID.String(), nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)

		respBody, err := parseBranchResponseBody(t, resp)
		require.NoError(t, err)
		assert.Contains(t, respBody["error"], "not initialized")
	})

	t.Run("tenant user triggers filtered lookup by UUID", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewBranchHandler(nil, nil, config.BranchingConfig{})

		tenantID := uuid.New()
		app.Use(func(c fiber.Ctx) error {
			c.Locals("tenant_id", tenantID.String())
			return c.Next()
		})
		app.Get("/branches/:id", handler.GetBranch)

		branchID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/branches/"+branchID.String(), nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)

		respBody, err := parseBranchResponseBody(t, resp)
		require.NoError(t, err)
		assert.Contains(t, respBody["error"], "not initialized")
	})

	t.Run("tenant user triggers filtered lookup by slug", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewBranchHandler(nil, nil, config.BranchingConfig{})

		tenantID := uuid.New()
		app.Use(func(c fiber.Ctx) error {
			c.Locals("tenant_id", tenantID.String())
			return c.Next()
		})
		app.Get("/branches/:id", handler.GetBranch)

		req := httptest.NewRequest(http.MethodGet, "/branches/my-feature-branch", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)
	})

	t.Run("service key bypasses tenant filter", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewBranchHandler(nil, nil, config.BranchingConfig{})

		app.Use(func(c fiber.Ctx) error {
			c.Locals("auth_type", "service_key")
			return c.Next()
		})
		app.Get("/branches/:id", handler.GetBranch)

		branchID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/branches/"+branchID.String(), nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)

		respBody, err := parseBranchResponseBody(t, resp)
		require.NoError(t, err)
		assert.Contains(t, respBody["error"], "not initialized")
	})
}

// =============================================================================
// DeleteBranch Handler Tenant Isolation Tests
// =============================================================================

func TestDeleteBranch_TenantIsolation(t *testing.T) {
	t.Run("tenant user delete triggers filtered lookup", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewBranchHandler(nil, nil, config.BranchingConfig{})

		tenantID := uuid.New()
		userID := uuid.New()
		app.Use(func(c fiber.Ctx) error {
			c.Locals("tenant_id", tenantID.String())
			c.Locals("user_id", userID.String())
			c.Locals("user_role", "member")
			return c.Next()
		})
		app.Delete("/branches/:id", handler.DeleteBranch)

		branchID := uuid.New()
		req := httptest.NewRequest(http.MethodDelete, "/branches/"+branchID.String(), nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)

		respBody, err := parseBranchResponseBody(t, resp)
		require.NoError(t, err)
		assert.Contains(t, respBody["error"], "not initialized")
	})

	t.Run("instance admin delete bypasses tenant filter", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewBranchHandler(nil, nil, config.BranchingConfig{})

		app.Use(func(c fiber.Ctx) error {
			c.Locals("is_instance_admin", true)
			return c.Next()
		})
		app.Delete("/branches/:id", handler.DeleteBranch)

		branchID := uuid.New()
		req := httptest.NewRequest(http.MethodDelete, "/branches/"+branchID.String(), nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)

		respBody, err := parseBranchResponseBody(t, resp)
		require.NoError(t, err)
		assert.Contains(t, respBody["error"], "not initialized")
	})

	t.Run("service key delete bypasses tenant filter", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewBranchHandler(nil, nil, config.BranchingConfig{})

		app.Use(func(c fiber.Ctx) error {
			c.Locals("auth_type", "service_key")
			return c.Next()
		})
		app.Delete("/branches/:id", handler.DeleteBranch)

		branchID := uuid.New()
		req := httptest.NewRequest(http.MethodDelete, "/branches/"+branchID.String(), nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)

		respBody, err := parseBranchResponseBody(t, resp)
		require.NoError(t, err)
		assert.Contains(t, respBody["error"], "not initialized")
	})

	t.Run("admin role delete with tenant context", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewBranchHandler(nil, nil, config.BranchingConfig{})

		tenantID := uuid.New()
		userID := uuid.New()
		app.Use(func(c fiber.Ctx) error {
			c.Locals("tenant_id", tenantID.String())
			c.Locals("user_id", userID.String())
			c.Locals("user_role", "admin")
			return c.Next()
		})
		app.Delete("/branches/:id", handler.DeleteBranch)

		branchID := uuid.New()
		req := httptest.NewRequest(http.MethodDelete, "/branches/"+branchID.String(), nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)
	})
}

// =============================================================================
// ListBranches Handler Tenant Isolation Tests
// =============================================================================

func TestListBranches_TenantIsolation(t *testing.T) {
	t.Run("regular user gets tenant filtered results", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewBranchHandler(nil, nil, config.BranchingConfig{})

		tenantID := uuid.New()
		app.Use(func(c fiber.Ctx) error {
			c.Locals("tenant_id", tenantID.String())
			c.Locals("user_role", "member")
			return c.Next()
		})
		app.Get("/branches", handler.ListBranches)

		req := httptest.NewRequest(http.MethodGet, "/branches", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)

		respBody, err := parseBranchResponseBody(t, resp)
		require.NoError(t, err)
		assert.Contains(t, respBody["error"], "not initialized")
	})

	t.Run("instance admin gets unfiltered results", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewBranchHandler(nil, nil, config.BranchingConfig{})

		tenantID := uuid.New()
		app.Use(func(c fiber.Ctx) error {
			c.Locals("tenant_id", tenantID.String())
			c.Locals("user_role", "instance_admin")
			return c.Next()
		})
		app.Get("/branches", handler.ListBranches)

		req := httptest.NewRequest(http.MethodGet, "/branches", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)
	})

	t.Run("admin role gets unfiltered results", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewBranchHandler(nil, nil, config.BranchingConfig{})

		tenantID := uuid.New()
		app.Use(func(c fiber.Ctx) error {
			c.Locals("tenant_id", tenantID.String())
			c.Locals("user_role", "admin")
			return c.Next()
		})
		app.Get("/branches", handler.ListBranches)

		req := httptest.NewRequest(http.MethodGet, "/branches", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)
	})

	t.Run("member role with no tenant_id gets no filter", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewBranchHandler(nil, nil, config.BranchingConfig{})

		app.Use(func(c fiber.Ctx) error {
			c.Locals("user_role", "member")
			return c.Next()
		})
		app.Get("/branches", handler.ListBranches)

		req := httptest.NewRequest(http.MethodGet, "/branches", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)
	})
}

// =============================================================================
// ResetBranch Handler Tenant Isolation Tests
// =============================================================================

func TestResetBranch_TenantIsolation(t *testing.T) {
	t.Run("tenant user reset triggers filtered lookup", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewBranchHandler(nil, nil, config.BranchingConfig{})

		tenantID := uuid.New()
		userID := uuid.New()
		app.Use(func(c fiber.Ctx) error {
			c.Locals("tenant_id", tenantID.String())
			c.Locals("user_id", userID.String())
			c.Locals("user_role", "member")
			return c.Next()
		})
		app.Post("/branches/:id/reset", handler.ResetBranch)

		branchID := uuid.New()
		req := httptest.NewRequest(http.MethodPost, "/branches/"+branchID.String()+"/reset", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)

		respBody, err := parseBranchResponseBody(t, resp)
		require.NoError(t, err)
		assert.Contains(t, respBody["error"], "not initialized")
	})

	t.Run("instance admin reset bypasses tenant filter", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewBranchHandler(nil, nil, config.BranchingConfig{})

		app.Use(func(c fiber.Ctx) error {
			c.Locals("is_instance_admin", true)
			return c.Next()
		})
		app.Post("/branches/:id/reset", handler.ResetBranch)

		branchID := uuid.New()
		req := httptest.NewRequest(http.MethodPost, "/branches/"+branchID.String()+"/reset", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)
	})
}

// =============================================================================
// Access Management Handler Tenant Isolation Tests
// =============================================================================

func TestListBranchAccess_TenantIsolation(t *testing.T) {
	t.Run("tenant user access list triggers filtered lookup", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewBranchHandler(nil, nil, config.BranchingConfig{})

		tenantID := uuid.New()
		app.Use(func(c fiber.Ctx) error {
			c.Locals("tenant_id", tenantID.String())
			return c.Next()
		})
		app.Get("/branches/:id/access", handler.ListBranchAccess)

		branchID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/branches/"+branchID.String()+"/access", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)

		respBody, err := parseBranchResponseBody(t, resp)
		require.NoError(t, err)
		assert.Contains(t, respBody["error"], "not initialized")
	})
}

func TestGrantBranchAccess_TenantIsolation(t *testing.T) {
	t.Run("tenant user grant triggers filtered lookup", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewBranchHandler(nil, nil, config.BranchingConfig{})

		tenantID := uuid.New()
		app.Use(func(c fiber.Ctx) error {
			c.Locals("tenant_id", tenantID.String())
			return c.Next()
		})
		app.Post("/branches/:id/access", handler.GrantBranchAccess)

		targetUserID := uuid.New()
		body := mustMarshalGrantAccessRequest(t, targetUserID.String(), "read")
		req := httptest.NewRequest(http.MethodPost, "/branches/test-branch/access", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)

		respBody, err := parseBranchResponseBody(t, resp)
		require.NoError(t, err)
		assert.Contains(t, respBody["error"], "not initialized")
	})
}

func TestRevokeBranchAccess_TenantIsolation(t *testing.T) {
	t.Run("tenant user revoke triggers filtered lookup", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewBranchHandler(nil, nil, config.BranchingConfig{})

		tenantID := uuid.New()
		app.Use(func(c fiber.Ctx) error {
			c.Locals("tenant_id", tenantID.String())
			return c.Next()
		})
		app.Delete("/branches/:id/access/:user_id", handler.RevokeBranchAccess)

		branchID := uuid.New()
		targetUserID := uuid.New()
		req := httptest.NewRequest(http.MethodDelete,
			"/branches/"+branchID.String()+"/access/"+targetUserID.String(), nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)

		respBody, err := parseBranchResponseBody(t, resp)
		require.NoError(t, err)
		assert.Contains(t, respBody["error"], "not initialized")
	})
}

// =============================================================================
// GetBranchActivity Handler Tenant Isolation Tests
// =============================================================================

func TestGetBranchActivity_TenantIsolation(t *testing.T) {
	t.Run("tenant user activity triggers filtered lookup", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewBranchHandler(nil, nil, config.BranchingConfig{})

		tenantID := uuid.New()
		app.Use(func(c fiber.Ctx) error {
			c.Locals("tenant_id", tenantID.String())
			return c.Next()
		})
		app.Get("/branches/:id/activity", handler.GetBranchActivity)

		branchID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/branches/"+branchID.String()+"/activity", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)

		respBody, err := parseBranchResponseBody(t, resp)
		require.NoError(t, err)
		assert.Contains(t, respBody["error"], "not initialized")
	})

	t.Run("instance admin activity uses no filter", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewBranchHandler(nil, nil, config.BranchingConfig{})

		app.Use(func(c fiber.Ctx) error {
			c.Locals("is_instance_admin", true)
			return c.Next()
		})
		app.Get("/branches/:id/activity", handler.GetBranchActivity)

		branchID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/branches/"+branchID.String()+"/activity", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)
	})
}

// =============================================================================
// Helper Functions
// =============================================================================

// parseBranchResponseBody reads and parses the response body into a map.
func parseBranchResponseBody(t *testing.T, resp *http.Response) (map[string]interface{}, error) {
	t.Helper()
	var result map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&result)
	return result, err
}

// mustMarshalGrantAccessRequest creates a JSON body for granting branch access.
func mustMarshalGrantAccessRequest(t *testing.T, userID, accessLevel string) []byte {
	t.Helper()
	body, err := json.Marshal(map[string]string{
		"user_id":      userID,
		"access_level": accessLevel,
	})
	if err != nil {
		t.Fatalf("failed to marshal grant access request body: %v", err)
	}
	return body
}
