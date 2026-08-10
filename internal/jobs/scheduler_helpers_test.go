package jobs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Unit tests for pure helpers in internal/jobs that were at 0% or partial
// coverage: Scheduler.parseScheduleConfig (0%) and Job.CalculateETA guard
// branches (85%). No DB, no Deno, no I/O.
//
// Convention matches scheduler_test.go: testify, package jobs (white-box so
// the unexported parseScheduleConfig is reachable).

// =============================================================================
// Scheduler.parseScheduleConfig (was 0%)
// =============================================================================
//
// Contract source: scheduler.go:140. Parses a schedule string of the form
// "<cron>" or "<cron>|<json-params>". Scans from the RIGHT for the first '|':
//   - no '|'        -> CronExpression = schedule, Params = empty map
//   - '|' found, valid JSON after it -> split cron + params
//   - '|' found, INVALID JSON after it -> log warn, revert to whole string as
//     cron and empty params (the '|' is treated as part of the cron expression)

func TestParseScheduleConfig(t *testing.T) {
	s := &Scheduler{} // parseScheduleConfig only uses the receiver to satisfy the method set

	t.Run("plain cron expression no pipe", func(t *testing.T) {
		cfg := s.parseScheduleConfig("*/5 * * * *")
		assert.Equal(t, "*/5 * * * *", cfg.CronExpression)
		assert.NotNil(t, cfg.Params)
		assert.Empty(t, cfg.Params)
	})

	t.Run("cron with valid json params", func(t *testing.T) {
		cfg := s.parseScheduleConfig(`*/5 * * * *|{"batch_size":100,"dry_run":true}`)
		assert.Equal(t, "*/5 * * * *", cfg.CronExpression)
		require.NotNil(t, cfg.Params)
		assert.Equal(t, float64(100), cfg.Params["batch_size"])
		assert.Equal(t, true, cfg.Params["dry_run"])
	})

	t.Run("cron with empty json object params", func(t *testing.T) {
		cfg := s.parseScheduleConfig(`0 * * * *|{}`)
		assert.Equal(t, "0 * * * *", cfg.CronExpression)
		assert.NotNil(t, cfg.Params)
		assert.Empty(t, cfg.Params)
	})

	t.Run("cron with string param value", func(t *testing.T) {
		cfg := s.parseScheduleConfig(`0 0 * * *|{"name":"daily-report"}`)
		assert.Equal(t, "0 0 * * *", cfg.CronExpression)
		assert.Equal(t, "daily-report", cfg.Params["name"])
	})

	t.Run("invalid json after pipe reverts to whole string as cron", func(t *testing.T) {
		// The '|' is treated as part of the cron expression when params don't parse.
		cfg := s.parseScheduleConfig(`*/5 * * * *|not json`)
		assert.Equal(t, "*/5 * * * *|not json", cfg.CronExpression)
		require.NotNil(t, cfg.Params)
		assert.Empty(t, cfg.Params, "params reset to empty on parse failure")
	})

	t.Run("empty schedule string", func(t *testing.T) {
		cfg := s.parseScheduleConfig("")
		assert.Equal(t, "", cfg.CronExpression)
		assert.NotNil(t, cfg.Params)
		assert.Empty(t, cfg.Params)
	})

	t.Run("pipe at end with empty params", func(t *testing.T) {
		// "|" at the end: paramsJSON is "" which is invalid JSON -> revert.
		cfg := s.parseScheduleConfig(`*/5 * * * *|`)
		// Empty string fails json.Unmarshal -> reverts to whole string.
		assert.Equal(t, "*/5 * * * *|", cfg.CronExpression)
		assert.Empty(t, cfg.Params)
	})

	t.Run("multiple pipes uses rightmost", func(t *testing.T) {
		// Scans right-to-left: the rightmost '|' splits cron from params.
		cfg := s.parseScheduleConfig(`0 0 * * *|{"a":1}|{"b":2}`)
		// Rightmost pipe -> cron = "0 0 * * *|{"a":1}", params = {"b":2}
		assert.Equal(t, `0 0 * * *|{"a":1}`, cfg.CronExpression)
		require.NotNil(t, cfg.Params)
		assert.Equal(t, float64(2), cfg.Params["b"])
	})
}

// =============================================================================
// Job.CalculateETA — remaining guard branches (was 85%)
// =============================================================================
//
// The existing TestJob_CalculateETA covers the happy path and several guards.
// These cases fill the remaining branches: nil progress pointer, empty
// progress string, invalid JSON, and elapsed.Seconds() <= 0 (future start time).

func TestJob_CalculateETA_AdditionalGuards(t *testing.T) {
	t.Run("nil progress pointer is a no-op", func(t *testing.T) {
		start := time.Now().Add(-10 * time.Second)
		job := &Job{Status: JobStatusRunning, StartedAt: &start, Progress: nil}
		job.CalculateETA()
		assert.Nil(t, job.EstimatedSecondsLeft)
		assert.Nil(t, job.EstimatedCompletionAt)
	})

	t.Run("empty progress string is a no-op", func(t *testing.T) {
		start := time.Now().Add(-10 * time.Second)
		empty := ""
		job := &Job{Status: JobStatusRunning, StartedAt: &start, Progress: &empty}
		job.CalculateETA()
		assert.Nil(t, job.EstimatedSecondsLeft)
	})

	t.Run("invalid progress JSON is a no-op", func(t *testing.T) {
		start := time.Now().Add(-10 * time.Second)
		bad := `{not json}`
		job := &Job{Status: JobStatusRunning, StartedAt: &start, Progress: &bad}
		job.CalculateETA()
		assert.Nil(t, job.EstimatedSecondsLeft)
	})

	t.Run("future start time (zero/negative elapsed) is a no-op", func(t *testing.T) {
		// A start time in the future means elapsed.Seconds() < 0 -> guard returns.
		future := time.Now().Add(10 * time.Second)
		progressJSON := `{"percent": 50}`
		job := &Job{Status: JobStatusRunning, StartedAt: &future, Progress: &progressJSON}
		job.CalculateETA()
		assert.Nil(t, job.EstimatedSecondsLeft)
		assert.Nil(t, job.EstimatedCompletionAt)
	})

	t.Run("not running status is a no-op even with valid progress", func(t *testing.T) {
		start := time.Now().Add(-10 * time.Second)
		progressJSON := `{"percent": 50}`
		for _, status := range []JobStatus{JobStatusPending, JobStatusCompleted, JobStatusFailed, JobStatusCancelled} {
			job := &Job{Status: status, StartedAt: &start, Progress: &progressJSON}
			job.CalculateETA()
			assert.Nil(t, job.EstimatedSecondsLeft, "status %s should not compute ETA", status)
		}
	})

	t.Run("negative percent does not compute", func(t *testing.T) {
		start := time.Now().Add(-10 * time.Second)
		progressJSON := `{"percent": -5}`
		job := &Job{Status: JobStatusRunning, StartedAt: &start, Progress: &progressJSON}
		job.CalculateETA()
		assert.Nil(t, job.EstimatedSecondsLeft)
	})
}
