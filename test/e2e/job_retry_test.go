//go:build integration

package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/test"
)

const failingJobCode = `export default {
  async handler(payload) {
    throw new Error("deliberate failure for retry testing");
  }
};`

func createFailingJobFunction(t *testing.T, tc *test.TestContext, namespace, jobName string, maxRetries int) {
	t.Helper()
	tenantID := tc.GetDefaultTenantID()
	ctx := database.ContextWithTenant(context.Background(), tenantID)

	err := database.WrapWithServiceRoleAndTenant(ctx, tc.DB, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO jobs.functions (namespace, name, code, enabled, timeout_seconds, max_retries, progress_timeout_seconds)
			VALUES ($1, $2, $3, true, 30, $4, 60)
			ON CONFLICT (namespace, name) DO UPDATE SET code = $3, enabled = true, max_retries = $4
		`, namespace, jobName, failingJobCode, maxRetries)
		return err
	})
	require.NoError(t, err)
}

func execSQL(t *testing.T, tc *test.TestContext, sql string, args ...interface{}) {
	t.Helper()
	_, err := tc.DB.Exec(context.Background(), sql, args...)
	require.NoError(t, err)
}

func querySQL(t *testing.T, tc *test.TestContext, sql string, args ...interface{}) []map[string]interface{} {
	t.Helper()
	return tc.QuerySQL(sql, args...)
}

func markJobRunning(t *testing.T, tc *test.TestContext, jobID string) {
	t.Helper()
	execSQL(t, tc, `UPDATE jobs.queue SET status = 'running', started_at = NOW(), worker_id = gen_random_uuid() WHERE id = $1 AND status = 'pending'`, jobID)
}

func markJobFailed(t *testing.T, tc *test.TestContext, jobID string) {
	t.Helper()
	execSQL(t, tc, `UPDATE jobs.queue SET status = 'failed', error_message = 'deliberate failure', completed_at = NOW() WHERE id = $1 AND status = 'running'`, jobID)
}

func requeueForRetry(t *testing.T, tc *test.TestContext, jobID string) bool {
	t.Helper()
	results := querySQL(t, tc, `
		UPDATE jobs.queue
		SET status = 'pending', retry_count = retry_count + 1, worker_id = NULL,
		    started_at = NULL, last_progress_at = NULL, completed_at = NULL,
		    error_message = 'deliberate failure',
		    scheduled_at = NOW() + make_interval(secs => 5.0 * POWER(2::float8, LEAST(retry_count, 6)))
		WHERE id = $1 AND status = 'running' AND retry_count < max_retries
		RETURNING id
	`, jobID)
	return len(results) > 0
}

func toInt64(v interface{}) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case int32:
		return int64(val)
	case int:
		return int64(val)
	default:
		return -1
	}
}

func submitRetryJob(t *testing.T, tc *test.TestContext, token, namespace, jobName string) map[string]interface{} {
	t.Helper()
	resp := tc.NewRequest("POST", "/api/v1/jobs/submit").
		WithAuth(token).
		WithJSON(map[string]interface{}{
			"job_name":  jobName,
			"namespace": namespace,
			"payload":   map[string]interface{}{"test": true},
		}).
		Send()
	require.Equal(t, fiber.StatusCreated, resp.Status(), "Failed to submit job: %s", string(resp.Body()))
	var result map[string]interface{}
	resp.JSON(&result)
	return result
}

func TestJobRetry_ExponentialBackoff(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()
	tc.EnsureAuthSchema()
	tc.EnsureSystemSettings()

	suffix := uuid.New().String()[:8]
	namespace := fmt.Sprintf("test-retry-%s", suffix)
	jobName := fmt.Sprintf("failing-job-%s", suffix)

	createFailingJobFunction(t, tc, namespace, jobName, 3)

	userEmail := test.RandomEmail()
	_, userToken := tc.CreateTestUser(userEmail, "password123")

	job := submitRetryJob(t, tc, userToken, namespace, jobName)
	jobID := job["id"].(string)
	require.NotEmpty(t, jobID)

	defer func() {
		execSQL(t, tc, "DELETE FROM jobs.queue WHERE id = $1", jobID)
		execSQL(t, tc, "DELETE FROM jobs.functions WHERE namespace = $1 AND name = $2", namespace, jobName)
	}()

	results := querySQL(t, tc, "SELECT retry_count, status, scheduled_at FROM jobs.queue WHERE id = $1", jobID)
	require.Len(t, results, 1)
	assert.Equal(t, int64(0), toInt64(results[0]["retry_count"]))
	assert.Equal(t, "pending", results[0]["status"])
	assert.Nil(t, results[0]["scheduled_at"])

	markJobRunning(t, tc, jobID)
	requeued := requeueForRetry(t, tc, jobID)
	require.True(t, requeued, "Job should be requeued (retry 1 of 3)")

	results = querySQL(t, tc, "SELECT retry_count, status, scheduled_at, NOW() as now_ts FROM jobs.queue WHERE id = $1", jobID)
	require.Len(t, results, 1)
	assert.Equal(t, int64(1), toInt64(results[0]["retry_count"]))
	assert.Equal(t, "pending", results[0]["status"])
	scheduledAt, ok := results[0]["scheduled_at"].(time.Time)
	require.True(t, ok, "scheduled_at should be a time.Time")
	nowTs, ok := results[0]["now_ts"].(time.Time)
	require.True(t, ok, "now_ts should be a time.Time")
	delay := scheduledAt.Sub(nowTs).Seconds()
	assert.GreaterOrEqual(t, delay, 4.0, "First backoff should be ~5s")
	assert.LessOrEqual(t, delay, 7.0, "First backoff should be ~5s")

	markJobRunning(t, tc, jobID)
	requeued = requeueForRetry(t, tc, jobID)
	require.True(t, requeued, "Job should be requeued (retry 2 of 3)")

	results = querySQL(t, tc, "SELECT retry_count, status, scheduled_at, NOW() as now_ts FROM jobs.queue WHERE id = $1", jobID)
	require.Len(t, results, 1)
	assert.Equal(t, int64(2), toInt64(results[0]["retry_count"]))
	scheduledAt, ok = results[0]["scheduled_at"].(time.Time)
	require.True(t, ok)
	nowTs, ok = results[0]["now_ts"].(time.Time)
	require.True(t, ok)
	delay = scheduledAt.Sub(nowTs).Seconds()
	assert.GreaterOrEqual(t, delay, 9.0, "Second backoff should be ~10s")
	assert.LessOrEqual(t, delay, 12.0, "Second backoff should be ~10s")

	markJobRunning(t, tc, jobID)
	requeued = requeueForRetry(t, tc, jobID)
	require.True(t, requeued, "Job should be requeued (retry 3 of 3)")

	results = querySQL(t, tc, "SELECT retry_count, status, scheduled_at, NOW() as now_ts FROM jobs.queue WHERE id = $1", jobID)
	require.Len(t, results, 1)
	assert.Equal(t, int64(3), toInt64(results[0]["retry_count"]))
	scheduledAt, ok = results[0]["scheduled_at"].(time.Time)
	require.True(t, ok)
	nowTs, ok = results[0]["now_ts"].(time.Time)
	require.True(t, ok)
	delay = scheduledAt.Sub(nowTs).Seconds()
	assert.GreaterOrEqual(t, delay, 19.0, "Third backoff should be ~20s")
	assert.LessOrEqual(t, delay, 22.0, "Third backoff should be ~20s")

	markJobRunning(t, tc, jobID)
	requeued = requeueForRetry(t, tc, jobID)
	assert.False(t, requeued, "Job should NOT be requeued when retry_count (3) reaches max_retries (3)")

	execSQL(t, tc, `UPDATE jobs.queue SET status = 'failed', error_message = 'deliberate failure', completed_at = NOW() WHERE id = $1 AND status = 'running'`, jobID)

	results = querySQL(t, tc, "SELECT retry_count, status, error_message, completed_at FROM jobs.queue WHERE id = $1", jobID)
	require.Len(t, results, 1)
	assert.Equal(t, int64(3), toInt64(results[0]["retry_count"]))
	assert.Equal(t, "failed", results[0]["status"])
	assert.NotNil(t, results[0]["error_message"])
	assert.NotNil(t, results[0]["completed_at"])
}

func TestJobRetry_AdminRetryEndpoint(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()
	tc.EnsureAuthSchema()
	tc.EnsureSystemSettings()

	suffix := uuid.New().String()[:8]
	namespace := fmt.Sprintf("test-retry-api-%s", suffix)
	jobName := fmt.Sprintf("failing-api-job-%s", suffix)

	createFailingJobFunction(t, tc, namespace, jobName, 2)

	_, adminToken := tc.CreateDashboardAdminUser(test.E2ETestEmailWithSuffix("admin-retry"), "adminpassword123")

	job := submitRetryJob(t, tc, adminToken, namespace, jobName)
	jobID := job["id"].(string)
	require.NotEmpty(t, jobID)

	defer func() {
		execSQL(t, tc, "DELETE FROM jobs.queue WHERE id = $1", jobID)
		execSQL(t, tc, "DELETE FROM jobs.functions WHERE namespace = $1 AND name = $2", namespace, jobName)
	}()

	markJobRunning(t, tc, jobID)
	markJobFailed(t, tc, jobID)

	results := querySQL(t, tc, "SELECT status, retry_count FROM jobs.queue WHERE id = $1", jobID)
	require.Len(t, results, 1)
	assert.Equal(t, "failed", results[0]["status"])
	assert.Equal(t, int64(0), toInt64(results[0]["retry_count"]))

	resp := tc.NewRequest("POST", fmt.Sprintf("/api/v1/admin/jobs/queue/%s/retry", jobID)).
		WithAuth(adminToken).
		Send()
	assert.Equal(t, fiber.StatusOK, resp.Status())

	results = querySQL(t, tc, "SELECT retry_count, status, scheduled_at, NOW() as now_ts FROM jobs.queue WHERE id = $1", jobID)
	require.Len(t, results, 1)
	assert.Equal(t, int64(1), toInt64(results[0]["retry_count"]))
	assert.Equal(t, "pending", results[0]["status"])
	scheduledAt, ok := results[0]["scheduled_at"].(time.Time)
	require.True(t, ok)
	nowTs, ok := results[0]["now_ts"].(time.Time)
	require.True(t, ok)
	delay := scheduledAt.Sub(nowTs).Seconds()
	assert.GreaterOrEqual(t, delay, 4.0)
	assert.LessOrEqual(t, delay, 7.0)

	resp = tc.NewRequest("POST", fmt.Sprintf("/api/v1/admin/jobs/queue/%s/retry", jobID)).
		WithAuth(adminToken).
		Send()
	assert.Equal(t, fiber.StatusBadRequest, resp.Status())

	markJobRunning(t, tc, jobID)
	markJobFailed(t, tc, jobID)

	resp = tc.NewRequest("POST", fmt.Sprintf("/api/v1/admin/jobs/queue/%s/retry", jobID)).
		WithAuth(adminToken).
		Send()
	assert.Equal(t, fiber.StatusOK, resp.Status())

	results = querySQL(t, tc, "SELECT retry_count, status FROM jobs.queue WHERE id = $1", jobID)
	require.Len(t, results, 1)
	assert.Equal(t, int64(2), toInt64(results[0]["retry_count"]))
	assert.Equal(t, "pending", results[0]["status"])

	markJobRunning(t, tc, jobID)
	markJobFailed(t, tc, jobID)

	results = querySQL(t, tc, "SELECT retry_count, status FROM jobs.queue WHERE id = $1", jobID)
	require.Len(t, results, 1)
	assert.Equal(t, int64(2), toInt64(results[0]["retry_count"]))
	assert.Equal(t, "failed", results[0]["status"])

	resp = tc.NewRequest("POST", fmt.Sprintf("/api/v1/admin/jobs/queue/%s/retry", jobID)).
		WithAuth(adminToken).
		Send()
	assert.Equal(t, fiber.StatusInternalServerError, resp.Status())
}

func TestJobRetry_BackoffDoublesEachAttempt(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()
	tc.EnsureAuthSchema()
	tc.EnsureSystemSettings()

	suffix := uuid.New().String()[:8]
	namespace := fmt.Sprintf("test-backoff-%s", suffix)
	jobName := fmt.Sprintf("backoff-job-%s", suffix)

	createFailingJobFunction(t, tc, namespace, jobName, 5)

	userEmail := test.RandomEmail()
	_, userToken := tc.CreateTestUser(userEmail, "password123")

	job := submitRetryJob(t, tc, userToken, namespace, jobName)
	jobID := job["id"].(string)
	require.NotEmpty(t, jobID)

	defer func() {
		execSQL(t, tc, "DELETE FROM jobs.queue WHERE id = $1", jobID)
		execSQL(t, tc, "DELETE FROM jobs.functions WHERE namespace = $1 AND name = $2", namespace, jobName)
	}()

	expectedBackoffs := []float64{5.0, 10.0, 20.0, 40.0, 80.0}

	for i, expectedBackoff := range expectedBackoffs {
		markJobRunning(t, tc, jobID)

		results := querySQL(t, tc, `
			UPDATE jobs.queue
			SET status = 'pending', retry_count = retry_count + 1, worker_id = NULL,
			    started_at = NULL, last_progress_at = NULL, completed_at = NULL,
			    error_message = 'deliberate failure',
			    scheduled_at = NOW() + make_interval(secs => 5.0 * POWER(2::float8, LEAST(retry_count, 6)))
			WHERE id = $1 AND status = 'running' AND retry_count < max_retries
			RETURNING retry_count, scheduled_at, NOW() as now_ts
		`, jobID)
		require.Len(t, results, 1, "Retry %d should succeed", i+1)

		assert.Equal(t, int64(i+1), toInt64(results[0]["retry_count"]), "retry_count after attempt %d", i+1)

		scheduledAt, ok := results[0]["scheduled_at"].(time.Time)
		require.True(t, ok)
		nowTs, ok := results[0]["now_ts"].(time.Time)
		require.True(t, ok)

		delay := scheduledAt.Sub(nowTs).Seconds()
		tolerance := 3.0
		assert.GreaterOrEqual(t, delay, expectedBackoff-tolerance,
			"Retry %d backoff should be ~%.0fs (5 * 2^%d)", i+1, expectedBackoff, i)
		assert.LessOrEqual(t, delay, expectedBackoff+tolerance,
			"Retry %d backoff should be ~%.0fs (5 * 2^%d)", i+1, expectedBackoff, i)
	}

	markJobRunning(t, tc, jobID)
	results := querySQL(t, tc, `
		UPDATE jobs.queue
		SET status = 'pending', retry_count = retry_count + 1, worker_id = NULL,
		    started_at = NULL, last_progress_at = NULL, completed_at = NULL,
		    error_message = 'deliberate failure',
		    scheduled_at = NOW() + make_interval(secs => 5.0 * POWER(2::float8, LEAST(retry_count, 6)))
		WHERE id = $1 AND status = 'running' AND retry_count < max_retries
		RETURNING id
	`, jobID)
	assert.Len(t, results, 0, "Should not requeue after max_retries (5) reached")
}
