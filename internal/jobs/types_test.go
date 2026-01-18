package jobs

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobStatus_Constants(t *testing.T) {
	assert.Equal(t, JobStatus("pending"), JobStatusPending)
	assert.Equal(t, JobStatus("running"), JobStatusRunning)
	assert.Equal(t, JobStatus("completed"), JobStatusCompleted)
	assert.Equal(t, JobStatus("failed"), JobStatusFailed)
	assert.Equal(t, JobStatus("cancelled"), JobStatusCancelled)
	assert.Equal(t, JobStatus("interrupted"), JobStatusInterrupted)
}

func TestWorkerStatus_Constants(t *testing.T) {
	assert.Equal(t, WorkerStatus("active"), WorkerStatusActive)
	assert.Equal(t, WorkerStatus("draining"), WorkerStatusDraining)
	assert.Equal(t, WorkerStatus("stopped"), WorkerStatusStopped)
}

func TestJobFunction_Struct(t *testing.T) {
	t.Run("creates job function with all fields", func(t *testing.T) {
		id := uuid.New()
		createdBy := uuid.New()
		desc := "Test function"
		code := "console.log('hello');"
		schedule := "*/5 * * * *"
		requireRoles := []string{"admin", "editor"}
		now := time.Now()

		fn := &JobFunction{
			ID:                     id,
			Name:                   "test-function",
			Namespace:              "default",
			Description:            &desc,
			Code:                   &code,
			OriginalCode:           &code,
			IsBundled:              true,
			Enabled:                true,
			Schedule:               &schedule,
			TimeoutSeconds:         30,
			MemoryLimitMB:          256,
			MaxRetries:             3,
			ProgressTimeoutSeconds: 60,
			AllowNet:               true,
			AllowEnv:               false,
			AllowRead:              true,
			AllowWrite:             false,
			RequireRoles:           requireRoles,
			Version:                1,
			CreatedBy:              &createdBy,
			Source:                 "filesystem",
			CreatedAt:              now,
			UpdatedAt:              now,
		}

		assert.Equal(t, id, fn.ID)
		assert.Equal(t, "test-function", fn.Name)
		assert.Equal(t, "default", fn.Namespace)
		assert.Equal(t, &desc, fn.Description)
		assert.Equal(t, &code, fn.Code)
		assert.True(t, fn.IsBundled)
		assert.True(t, fn.Enabled)
		assert.Equal(t, &schedule, fn.Schedule)
		assert.Equal(t, 30, fn.TimeoutSeconds)
		assert.Equal(t, 256, fn.MemoryLimitMB)
		assert.Equal(t, 3, fn.MaxRetries)
		assert.True(t, fn.AllowNet)
		assert.False(t, fn.AllowEnv)
		assert.Equal(t, "filesystem", fn.Source)
	})

	t.Run("marshals to JSON", func(t *testing.T) {
		fn := &JobFunction{
			ID:             uuid.New(),
			Name:           "json-test",
			Namespace:      "default",
			Enabled:        true,
			TimeoutSeconds: 60,
		}

		data, err := json.Marshal(fn)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "json-test", result["name"])
		assert.Equal(t, "default", result["namespace"])
		assert.Equal(t, true, result["enabled"])
	})
}

func TestJob_FlattenProgress(t *testing.T) {
	t.Run("flattens progress with all fields", func(t *testing.T) {
		progressJSON := `{"percent": 75, "message": "Processing...", "data": {"items": 100}}`
		job := &Job{
			Progress: &progressJSON,
		}

		job.FlattenProgress()

		require.NotNil(t, job.ProgressPercent)
		assert.Equal(t, 75, *job.ProgressPercent)
		require.NotNil(t, job.ProgressMessage)
		assert.Equal(t, "Processing...", *job.ProgressMessage)
		assert.NotNil(t, job.ProgressData)
		assert.Equal(t, float64(100), job.ProgressData["items"])
	})

	t.Run("handles nil progress", func(t *testing.T) {
		job := &Job{
			Progress: nil,
		}

		job.FlattenProgress()

		assert.Nil(t, job.ProgressPercent)
		assert.Nil(t, job.ProgressMessage)
		assert.Nil(t, job.ProgressData)
	})

	t.Run("handles empty progress string", func(t *testing.T) {
		emptyStr := ""
		job := &Job{
			Progress: &emptyStr,
		}

		job.FlattenProgress()

		assert.Nil(t, job.ProgressPercent)
	})

	t.Run("handles invalid JSON", func(t *testing.T) {
		invalidJSON := `{invalid json}`
		job := &Job{
			Progress: &invalidJSON,
		}

		job.FlattenProgress()

		assert.Nil(t, job.ProgressPercent)
	})

	t.Run("handles progress with only percent", func(t *testing.T) {
		progressJSON := `{"percent": 50}`
		job := &Job{
			Progress: &progressJSON,
		}

		job.FlattenProgress()

		require.NotNil(t, job.ProgressPercent)
		assert.Equal(t, 50, *job.ProgressPercent)
		assert.Nil(t, job.ProgressMessage)
		assert.Nil(t, job.ProgressData)
	})
}

func TestJob_CalculateETA(t *testing.T) {
	t.Run("calculates ETA for running job with progress", func(t *testing.T) {
		// Started 10 seconds ago, 50% done -> expect about 10 more seconds
		startTime := time.Now().Add(-10 * time.Second)
		progressJSON := `{"percent": 50}`

		job := &Job{
			Status:    JobStatusRunning,
			StartedAt: &startTime,
			Progress:  &progressJSON,
		}

		job.CalculateETA()

		require.NotNil(t, job.EstimatedSecondsLeft)
		// Allow some tolerance for timing
		assert.InDelta(t, 10, *job.EstimatedSecondsLeft, 2)
		assert.NotNil(t, job.EstimatedCompletionAt)
	})

	t.Run("does not calculate for non-running job", func(t *testing.T) {
		startTime := time.Now().Add(-10 * time.Second)
		progressJSON := `{"percent": 50}`

		job := &Job{
			Status:    JobStatusCompleted,
			StartedAt: &startTime,
			Progress:  &progressJSON,
		}

		job.CalculateETA()

		assert.Nil(t, job.EstimatedSecondsLeft)
		assert.Nil(t, job.EstimatedCompletionAt)
	})

	t.Run("does not calculate without start time", func(t *testing.T) {
		progressJSON := `{"percent": 50}`

		job := &Job{
			Status:   JobStatusRunning,
			Progress: &progressJSON,
		}

		job.CalculateETA()

		assert.Nil(t, job.EstimatedSecondsLeft)
	})

	t.Run("does not calculate with 0% progress", func(t *testing.T) {
		startTime := time.Now().Add(-10 * time.Second)
		progressJSON := `{"percent": 0}`

		job := &Job{
			Status:    JobStatusRunning,
			StartedAt: &startTime,
			Progress:  &progressJSON,
		}

		job.CalculateETA()

		assert.Nil(t, job.EstimatedSecondsLeft)
	})

	t.Run("does not calculate with 100% progress", func(t *testing.T) {
		startTime := time.Now().Add(-10 * time.Second)
		progressJSON := `{"percent": 100}`

		job := &Job{
			Status:    JobStatusRunning,
			StartedAt: &startTime,
			Progress:  &progressJSON,
		}

		job.CalculateETA()

		assert.Nil(t, job.EstimatedSecondsLeft)
	})

	t.Run("does not calculate without progress", func(t *testing.T) {
		startTime := time.Now().Add(-10 * time.Second)

		job := &Job{
			Status:    JobStatusRunning,
			StartedAt: &startTime,
		}

		job.CalculateETA()

		assert.Nil(t, job.EstimatedSecondsLeft)
	})
}

func TestProgress_Struct(t *testing.T) {
	t.Run("creates progress with all fields", func(t *testing.T) {
		estimatedLeft := 30

		p := Progress{
			Percent:              75,
			Message:              "Processing items...",
			EstimatedSecondsLeft: &estimatedLeft,
			Data: map[string]interface{}{
				"processed": 150,
				"total":     200,
			},
		}

		assert.Equal(t, 75, p.Percent)
		assert.Equal(t, "Processing items...", p.Message)
		assert.Equal(t, 30, *p.EstimatedSecondsLeft)
		assert.Equal(t, 150, p.Data["processed"])
	})

	t.Run("marshals to JSON", func(t *testing.T) {
		p := Progress{
			Percent: 50,
			Message: "Halfway",
		}

		data, err := json.Marshal(p)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, float64(50), result["percent"])
		assert.Equal(t, "Halfway", result["message"])
	})
}

func TestJobFilters_Struct(t *testing.T) {
	t.Run("creates filters with all options", func(t *testing.T) {
		status := JobStatusRunning
		name := "my-job"
		ns := "production"
		createdBy := uuid.New()
		workerID := uuid.New()
		limit := 50
		offset := 10
		includeResult := true

		filters := JobFilters{
			Status:        &status,
			JobName:       &name,
			Namespace:     &ns,
			CreatedBy:     &createdBy,
			WorkerID:      &workerID,
			Limit:         &limit,
			Offset:        &offset,
			IncludeResult: &includeResult,
		}

		assert.Equal(t, JobStatusRunning, *filters.Status)
		assert.Equal(t, "my-job", *filters.JobName)
		assert.Equal(t, "production", *filters.Namespace)
		assert.Equal(t, 50, *filters.Limit)
		assert.Equal(t, 10, *filters.Offset)
		assert.True(t, *filters.IncludeResult)
	})
}

func TestJobStats_Struct(t *testing.T) {
	t.Run("creates job stats", func(t *testing.T) {
		stats := JobStats{
			TotalJobs:          100,
			PendingJobs:        10,
			RunningJobs:        5,
			CompletedJobs:      80,
			FailedJobs:         4,
			CancelledJobs:      1,
			AvgDurationSeconds: 45.5,
			JobsByStatus: []JobStatusCount{
				{Status: "completed", Count: 80},
				{Status: "pending", Count: 10},
			},
			JobsByDay: []JobDayCount{
				{Date: "2024-01-01", Count: 25},
				{Date: "2024-01-02", Count: 30},
			},
			JobsByFunction: []JobFunctionCount{
				{Name: "process-images", Count: 50},
				{Name: "send-emails", Count: 50},
			},
		}

		assert.Equal(t, 100, stats.TotalJobs)
		assert.Equal(t, 10, stats.PendingJobs)
		assert.Equal(t, 5, stats.RunningJobs)
		assert.Equal(t, 80, stats.CompletedJobs)
		assert.Equal(t, 45.5, stats.AvgDurationSeconds)
		assert.Len(t, stats.JobsByStatus, 2)
		assert.Len(t, stats.JobsByDay, 2)
		assert.Len(t, stats.JobsByFunction, 2)
	})
}

func TestWorkerRecord_Struct(t *testing.T) {
	t.Run("creates worker record", func(t *testing.T) {
		id := uuid.New()
		name := "worker-1"
		hostname := "node-1.cluster.local"
		metadata := `{"version": "1.0"}`
		now := time.Now()

		worker := WorkerRecord{
			ID:                id,
			Name:              &name,
			Hostname:          &hostname,
			Status:            WorkerStatusActive,
			MaxConcurrentJobs: 10,
			CurrentJobCount:   3,
			LastHeartbeatAt:   now,
			StartedAt:         now.Add(-time.Hour),
			Metadata:          &metadata,
		}

		assert.Equal(t, id, worker.ID)
		assert.Equal(t, "worker-1", *worker.Name)
		assert.Equal(t, "node-1.cluster.local", *worker.Hostname)
		assert.Equal(t, WorkerStatusActive, worker.Status)
		assert.Equal(t, 10, worker.MaxConcurrentJobs)
		assert.Equal(t, 3, worker.CurrentJobCount)
	})
}

func TestPermissions_Struct(t *testing.T) {
	t.Run("creates permissions with all options", func(t *testing.T) {
		perms := Permissions{
			AllowNet:      true,
			AllowEnv:      false,
			AllowRead:     true,
			AllowWrite:    true,
			MemoryLimitMB: 512,
		}

		assert.True(t, perms.AllowNet)
		assert.False(t, perms.AllowEnv)
		assert.True(t, perms.AllowRead)
		assert.True(t, perms.AllowWrite)
		assert.Equal(t, 512, perms.MemoryLimitMB)
	})

	t.Run("creates restrictive permissions", func(t *testing.T) {
		perms := Permissions{
			AllowNet:      false,
			AllowEnv:      false,
			AllowRead:     false,
			AllowWrite:    false,
			MemoryLimitMB: 128,
		}

		assert.False(t, perms.AllowNet)
		assert.False(t, perms.AllowEnv)
		assert.False(t, perms.AllowRead)
		assert.False(t, perms.AllowWrite)
	})
}

func TestJobFunctionFile_Struct(t *testing.T) {
	t.Run("creates job function file", func(t *testing.T) {
		id := uuid.New()
		fnID := uuid.New()
		now := time.Now()

		file := JobFunctionFile{
			ID:            id,
			JobFunctionID: fnID,
			FilePath:      "utils/helpers.js",
			Content:       "export function helper() { return 42; }",
			CreatedAt:     now,
		}

		assert.Equal(t, id, file.ID)
		assert.Equal(t, fnID, file.JobFunctionID)
		assert.Equal(t, "utils/helpers.js", file.FilePath)
		assert.Contains(t, file.Content, "helper")
	})
}
