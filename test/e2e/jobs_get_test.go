package e2e

import (
	"context"
	"fmt"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/test"
)

const testJobCode = `export default {
  async handler(payload) {
    console.log("test job executed", JSON.stringify(payload));
    return { success: true, received: payload };
  }
};`

func createTestJobFunction(t *testing.T, tc *test.TestContext, namespace, jobName string) {
	t.Helper()
	tenantID := tc.GetDefaultTenantID()
	ctx := database.ContextWithTenant(context.Background(), tenantID)

	err := database.WrapWithServiceRoleAndTenant(ctx, tc.DB, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO jobs.functions (namespace, name, code, enabled, timeout_seconds, max_retries, progress_timeout_seconds)
			VALUES ($1, $2, $3, true, 30, 0, 60)
			ON CONFLICT (namespace, name) DO UPDATE SET code = $3, enabled = true
		`, namespace, jobName, testJobCode)
		return err
	})
	require.NoError(t, err, "Failed to create test job function")
}

func submitTestJob(t *testing.T, tc *test.TestContext, token, namespace, jobName string) map[string]interface{} {
	t.Helper()
	resp := tc.NewRequest("POST", "/api/v1/jobs/submit").
		WithAuth(token).
		WithJSON(map[string]interface{}{
			"job_name":  jobName,
			"namespace": namespace,
			"payload":   map[string]interface{}{"test": "data"},
		}).
		Send()
	require.Equal(t, fiber.StatusCreated, resp.Status(), "Failed to submit job: %s", string(resp.Body()))

	var result map[string]interface{}
	resp.JSON(&result)
	return result
}

func TestJobGetByID_ReturnsSubmittedJob(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()
	tc.EnsureAuthSchema()
	tc.EnsureSystemSettings()

	suffix := uuid.New().String()[:8]
	namespace := "test-jobs-get"
	jobName := fmt.Sprintf("test-get-job-%s", suffix)

	createTestJobFunction(t, tc, namespace, jobName)

	userEmail := "user-" + test.RandomEmail()
	_, userToken := tc.CreateTestUser(userEmail, "password123")

	job := submitTestJob(t, tc, userToken, namespace, jobName)
	jobID := job["id"].(string)
	require.NotEmpty(t, jobID)

	t.Run("GET /api/v1/jobs/:id returns the submitted job", func(t *testing.T) {
		resp := tc.NewRequest("GET", fmt.Sprintf("/api/v1/jobs/%s", jobID)).
			WithAuth(userToken).
			Send()
		assert.Equal(t, fiber.StatusOK, resp.Status(), "Expected 200, got %d: %s", resp.Status(), string(resp.Body()))

		var fetched map[string]interface{}
		resp.JSON(&fetched)
		assert.Equal(t, jobID, fetched["id"])
		assert.Equal(t, jobName, fetched["job_name"])
	})

	t.Run("GET /api/v1/jobs/:id with default tenant header returns the job", func(t *testing.T) {
		resp := tc.NewRequest("GET", fmt.Sprintf("/api/v1/jobs/%s", jobID)).
			WithAuth(userToken).
			WithDefaultTenant().
			Send()
		assert.Equal(t, fiber.StatusOK, resp.Status(), "Expected 200 with tenant header, got %d: %s", resp.Status(), string(resp.Body()))
	})

	t.Run("GET /api/v1/jobs/:id/logs returns job found (not 404)", func(t *testing.T) {
		resp := tc.NewRequest("GET", fmt.Sprintf("/api/v1/jobs/%s/logs", jobID)).
			WithAuth(userToken).
			Send()
		assert.NotEqual(t, fiber.StatusNotFound, resp.Status(),
			"Should not return 404 'Job not found', got: %s", string(resp.Body()))
	})

	t.Run("GET /api/v1/jobs lists the submitted job", func(t *testing.T) {
		resp := tc.NewRequest("GET", fmt.Sprintf("/api/v1/jobs?namespace=%s&limit=10", namespace)).
			WithAuth(userToken).
			Send()
		assert.Equal(t, fiber.StatusOK, resp.Status(), "Expected 200 from list, got %d: %s", resp.Status(), string(resp.Body()))

		var result map[string]interface{}
		resp.JSON(&result)
		jobs, ok := result["jobs"].([]interface{})
		require.True(t, ok, "Response should contain jobs array")
		assert.NotEmpty(t, jobs, "Jobs list should not be empty")
	})
}

func TestJobGetByID_NonexistentJobReturns404(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()

	email := "user-" + test.RandomEmail()
	_, token := tc.CreateTestUser(email, "password123")

	resp := tc.NewRequest("GET", "/api/v1/jobs/00000000-0000-0000-0000-000000000000").
		WithAuth(token).
		Send()
	assert.Equal(t, fiber.StatusNotFound, resp.Status())
}
