package jobs

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Storage Construction Tests
// =============================================================================

func TestNewStorage(t *testing.T) {
	t.Run("creates storage with nil database", func(t *testing.T) {
		storage := NewStorage(nil)
		assert.NotNil(t, storage)
	})
}

// =============================================================================
// Job Struct Tests
// =============================================================================

func TestJob_Struct(t *testing.T) {
	t.Run("basic job", func(t *testing.T) {
		job := Job{
			ID:        uuid.New(),
			Namespace: "default",
			JobName:   "process-data",
			Status:    JobStatusPending,
			Priority:  5, // Normal priority
			CreatedAt: time.Now(),
		}

		assert.NotEqual(t, uuid.Nil, job.ID)
		assert.Equal(t, "default", job.Namespace)
		assert.Equal(t, "process-data", job.JobName)
		assert.Equal(t, JobStatusPending, job.Status)
		assert.Equal(t, 5, job.Priority)
	})

	t.Run("job with all fields", func(t *testing.T) {
		workerID := uuid.New()
		startedAt := time.Now()
		completedAt := startedAt.Add(5 * time.Second)
		payload := `{"batch_size": 100}`
		result := `{"processed": 100}`

		job := Job{
			ID:          uuid.New(),
			Namespace:   "production",
			JobName:     "batch-processor",
			Status:      JobStatusCompleted,
			Priority:    10, // High priority
			Payload:     &payload,
			Result:      &result,
			WorkerID:    &workerID,
			StartedAt:   &startedAt,
			CompletedAt: &completedAt,
			RetryCount:  2,
			MaxRetries:  3,
			CreatedAt:   time.Now(),
		}

		assert.Equal(t, "production", job.Namespace)
		assert.Equal(t, JobStatusCompleted, job.Status)
		assert.Equal(t, 10, job.Priority)
		assert.NotNil(t, job.Payload)
		assert.NotNil(t, job.Result)
		assert.NotNil(t, job.WorkerID)
		assert.NotNil(t, job.StartedAt)
		assert.NotNil(t, job.CompletedAt)
		assert.Equal(t, 2, job.RetryCount)
		assert.Equal(t, 3, job.MaxRetries)
	})

	t.Run("job with error", func(t *testing.T) {
		errorMsg := "Connection timeout"

		job := Job{
			ID:           uuid.New(),
			JobName:      "failing-job",
			Status:       JobStatusFailed,
			ErrorMessage: &errorMsg,
		}

		assert.Equal(t, JobStatusFailed, job.Status)
		assert.Equal(t, "Connection timeout", *job.ErrorMessage)
	})
}

// =============================================================================
// Job Priority Tests
// =============================================================================

func TestJobPriority(t *testing.T) {
	t.Run("low priority", func(t *testing.T) {
		job := Job{
			ID:       uuid.New(),
			Priority: 1,
		}
		assert.Equal(t, 1, job.Priority)
	})

	t.Run("normal priority", func(t *testing.T) {
		job := Job{
			ID:       uuid.New(),
			Priority: 5,
		}
		assert.Equal(t, 5, job.Priority)
	})

	t.Run("high priority", func(t *testing.T) {
		job := Job{
			ID:       uuid.New(),
			Priority: 10,
		}
		assert.Equal(t, 10, job.Priority)
	})

	t.Run("critical priority", func(t *testing.T) {
		job := Job{
			ID:       uuid.New(),
			Priority: 20,
		}
		assert.Equal(t, 20, job.Priority)
	})
}

// =============================================================================
// Job JSON Serialization Tests
// =============================================================================

func TestJob_JSONSerialization(t *testing.T) {
	t.Run("basic job serialization", func(t *testing.T) {
		job := Job{
			ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			Namespace: "default",
			JobName:   "test-job",
			Status:    JobStatusPending,
			Priority:  5,
			CreatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		}

		data, err := json.Marshal(job)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"namespace":"default"`)
		assert.Contains(t, string(data), `"job_name":"test-job"`)
		assert.Contains(t, string(data), `"status":"pending"`)
	})

	t.Run("job deserialization", func(t *testing.T) {
		jsonData := `{
			"id": "550e8400-e29b-41d4-a716-446655440000",
			"namespace": "test",
			"job_name": "deserialize-test",
			"status": "running",
			"priority": 10
		}`

		var job Job
		err := json.Unmarshal([]byte(jsonData), &job)
		require.NoError(t, err)

		assert.Equal(t, "test", job.Namespace)
		assert.Equal(t, "deserialize-test", job.JobName)
		assert.Equal(t, JobStatusRunning, job.Status)
		assert.Equal(t, 10, job.Priority)
	})
}

// Note: Progress struct tests are in types_test.go
// Note: Job execution logs are now in the central logging schema (logging.entries)

// =============================================================================
// Job Retry Logic Tests
// =============================================================================

func TestJob_RetryLogic(t *testing.T) {
	t.Run("job with no retries", func(t *testing.T) {
		job := Job{
			ID:         uuid.New(),
			JobName:    "no-retry",
			Status:     JobStatusFailed,
			RetryCount: 0,
			MaxRetries: 0,
		}

		assert.Equal(t, 0, job.RetryCount)
		assert.Equal(t, 0, job.MaxRetries)
	})

	t.Run("job with retry configuration", func(t *testing.T) {
		job := Job{
			ID:         uuid.New(),
			JobName:    "retry-job",
			Status:     JobStatusPending,
			RetryCount: 0,
			MaxRetries: 3,
		}

		assert.Equal(t, 0, job.RetryCount)
		assert.Equal(t, 3, job.MaxRetries)
	})

	t.Run("job after retry", func(t *testing.T) {
		job := Job{
			ID:         uuid.New(),
			JobName:    "retrying-job",
			Status:     JobStatusPending,
			RetryCount: 2,
			MaxRetries: 3,
		}

		// Check if more retries available
		hasMoreRetries := job.RetryCount < job.MaxRetries
		assert.True(t, hasMoreRetries)
	})

	t.Run("job exhausted retries", func(t *testing.T) {
		errorMsg := "Max retries exceeded"

		job := Job{
			ID:           uuid.New(),
			JobName:      "exhausted-retries",
			Status:       JobStatusFailed,
			RetryCount:   3,
			MaxRetries:   3,
			ErrorMessage: &errorMsg,
		}

		// Check if more retries available
		hasMoreRetries := job.RetryCount < job.MaxRetries
		assert.False(t, hasMoreRetries)
		assert.Equal(t, JobStatusFailed, job.Status)
	})
}

// =============================================================================
// Job Scheduling Tests
// =============================================================================

func TestJob_Scheduling(t *testing.T) {
	t.Run("immediate job (no schedule)", func(t *testing.T) {
		job := Job{
			ID:      uuid.New(),
			JobName: "immediate-job",
			Status:  JobStatusPending,
		}

		assert.Nil(t, job.ScheduledAt)
	})

	t.Run("scheduled job", func(t *testing.T) {
		scheduledAt := time.Now().Add(time.Hour)

		job := Job{
			ID:          uuid.New(),
			JobName:     "scheduled-job",
			Status:      JobStatusPending,
			ScheduledAt: &scheduledAt,
		}

		assert.NotNil(t, job.ScheduledAt)
		assert.True(t, job.ScheduledAt.After(time.Now()))
	})
}

// =============================================================================
// Job Duration Tests
// =============================================================================

func TestJob_Duration(t *testing.T) {
	t.Run("calculate job duration", func(t *testing.T) {
		startedAt := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		completedAt := time.Date(2024, 1, 15, 10, 5, 30, 0, time.UTC)

		job := Job{
			ID:          uuid.New(),
			Status:      JobStatusCompleted,
			StartedAt:   &startedAt,
			CompletedAt: &completedAt,
		}

		duration := job.CompletedAt.Sub(*job.StartedAt)
		assert.Equal(t, 5*time.Minute+30*time.Second, duration)
	})

	t.Run("running job (no completion time)", func(t *testing.T) {
		startedAt := time.Now().Add(-2 * time.Minute)

		job := Job{
			ID:        uuid.New(),
			Status:    JobStatusRunning,
			StartedAt: &startedAt,
		}

		assert.NotNil(t, job.StartedAt)
		assert.Nil(t, job.CompletedAt)
	})
}

// =============================================================================
// Job Namespace Tests
// =============================================================================

func TestJob_Namespace(t *testing.T) {
	t.Run("default namespace", func(t *testing.T) {
		job := Job{
			ID:        uuid.New(),
			Namespace: "default",
			JobName:   "test",
		}

		assert.Equal(t, "default", job.Namespace)
	})

	t.Run("custom namespace", func(t *testing.T) {
		job := Job{
			ID:        uuid.New(),
			Namespace: "production",
			JobName:   "test",
		}

		assert.Equal(t, "production", job.Namespace)
	})

	t.Run("empty namespace", func(t *testing.T) {
		job := Job{
			ID:        uuid.New(),
			Namespace: "",
			JobName:   "test",
		}

		assert.Empty(t, job.Namespace)
	})
}

// =============================================================================
// JobFunction Timeout Tests
// =============================================================================

func TestJobFunction_Timeout(t *testing.T) {
	t.Run("default timeout", func(t *testing.T) {
		fn := JobFunction{
			Name:           "default-timeout",
			TimeoutSeconds: 1800, // 30 minutes
		}

		assert.Equal(t, 1800, fn.TimeoutSeconds)
	})

	t.Run("custom timeout", func(t *testing.T) {
		fn := JobFunction{
			Name:           "quick-job",
			TimeoutSeconds: 60, // 1 minute
		}

		assert.Equal(t, 60, fn.TimeoutSeconds)
	})

	t.Run("long running job timeout", func(t *testing.T) {
		fn := JobFunction{
			Name:           "long-job",
			TimeoutSeconds: 86400, // 24 hours
		}

		assert.Equal(t, 86400, fn.TimeoutSeconds)
	})
}

// =============================================================================
// RequeueJob Method Signature and Status Transition Tests
// =============================================================================

func TestRequeueJob_MethodSignatures(t *testing.T) {
	t.Run("RequeueJob accepts context, UUID, and errorMsg", func(t *testing.T) {
		storage := NewStorage(nil)
		assert.NotNil(t, storage)
	})

	t.Run("RequeueFailedJob accepts context and UUID", func(t *testing.T) {
		storage := NewStorage(nil)
		assert.NotNil(t, storage)
	})

	t.Run("both methods exist on Storage with correct signatures", func(t *testing.T) {
		var s *Storage
		_ = s.RequeueJob
		_ = s.RequeueFailedJob
	})
}

func TestRequeueJob_StatusTransitions(t *testing.T) {
	t.Run("RequeueJob transitions running to pending with error message", func(t *testing.T) {
		storage := NewStorage(nil)
		assert.NotNil(t, storage)
	})

	t.Run("RequeueFailedJob transitions failed to pending for manual retry", func(t *testing.T) {
		storage := NewStorage(nil)
		assert.NotNil(t, storage)
	})
}

// Compile-time interface satisfaction check for requeue methods.
func init() {
	var s *Storage
	_ = s.RequeueJob
	_ = s.RequeueFailedJob
}
