package functions

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/scheduler"
)

// =============================================================================
// Scheduler Construction Tests
// =============================================================================

func TestNewScheduler(t *testing.T) {
	t.Run("creates scheduler with nil dependencies", func(t *testing.T) {
		s := NewScheduler(nil, "jwt-secret", "http://localhost", nil, nil)
		require.NotNil(t, s)
		assert.NotNil(t, s.inner)
		assert.Equal(t, 10, s.inner.Guard.MaxConcurrent)
		assert.Equal(t, "jwt-secret", s.jwtSecret)
		assert.Equal(t, "http://localhost", s.publicURL)
	})

	t.Run("initializes empty entries", func(t *testing.T) {
		s := NewScheduler(nil, "secret", "http://localhost", nil, nil)
		assert.Equal(t, 0, s.inner.EntryCount())
	})

	t.Run("creates context via inner", func(t *testing.T) {
		s := NewScheduler(nil, "secret", "http://localhost", nil, nil)
		assert.NotNil(t, s.inner.Context())
	})
}

// =============================================================================
// Scheduler Log Message Handling Tests
// =============================================================================

func TestScheduler_handleLogMessage(t *testing.T) {
	t.Run("handles log without counter", func(t *testing.T) {
		s := NewScheduler(nil, "secret", "http://localhost", nil, nil)
		execID := uuid.New()

		s.handleLogMessage(execID, "info", "test message")
	})

	t.Run("increments counter when exists", func(t *testing.T) {
		s := NewScheduler(nil, "secret", "http://localhost", nil, nil)
		execID := uuid.New()

		ctx := &executionLogContext{}
		s.logCounters.Store(execID, ctx)

		s.handleLogMessage(execID, "info", "message 1")
		assert.Equal(t, 1, ctx.lineCounter)

		s.handleLogMessage(execID, "debug", "message 2")
		assert.Equal(t, 2, ctx.lineCounter)

		s.handleLogMessage(execID, "error", "message 3")
		assert.Equal(t, 3, ctx.lineCounter)
	})

	t.Run("handles invalid counter type gracefully", func(t *testing.T) {
		s := NewScheduler(nil, "secret", "http://localhost", nil, nil)
		execID := uuid.New()

		s.logCounters.Store(execID, "not a pointer")

		s.handleLogMessage(execID, "info", "test message")
	})

	t.Run("handles different log levels", func(t *testing.T) {
		s := NewScheduler(nil, "secret", "http://localhost", nil, nil)
		execID := uuid.New()

		levels := []string{"debug", "info", "warn", "error"}
		for _, level := range levels {
			s.handleLogMessage(execID, level, "test message")
		}
	})
}

// =============================================================================
// Cron Parser Tests
// =============================================================================

func TestCronParser(t *testing.T) {
	parser := scheduler.StandardParser

	t.Run("parses standard 5-field cron expressions", func(t *testing.T) {
		expressions := []struct {
			expr        string
			description string
		}{
			{"* * * * *", "every minute"},
			{"*/5 * * * *", "every 5 minutes"},
			{"0 * * * *", "every hour at minute 0"},
			{"0 0 * * *", "every day at midnight"},
			{"0 12 * * *", "every day at noon"},
			{"0 0 * * 0", "every Sunday at midnight"},
			{"0 0 1 * *", "first of every month"},
			{"0 0 1 1 *", "January 1st"},
			{"30 4 1,15 * *", "1st and 15th at 4:30"},
			{"0 22 * * 1-5", "weekdays at 10pm"},
		}

		for _, tc := range expressions {
			t.Run(tc.description, func(t *testing.T) {
				schedule, err := parser.Parse(tc.expr)
				require.NoError(t, err, "Failed to parse: %s", tc.expr)
				assert.NotNil(t, schedule)
			})
		}
	})

	t.Run("parses 6-field cron expressions with seconds", func(t *testing.T) {
		expressions := []struct {
			expr        string
			description string
		}{
			{"0 * * * * *", "every minute at second 0"},
			{"30 * * * * *", "every minute at second 30"},
			{"0 */5 * * * *", "every 5 minutes at second 0"},
			{"*/10 * * * * *", "every 10 seconds"},
			{"0 0 * * * *", "every hour at minute 0, second 0"},
		}

		for _, tc := range expressions {
			t.Run(tc.description, func(t *testing.T) {
				schedule, err := parser.Parse(tc.expr)
				require.NoError(t, err, "Failed to parse: %s", tc.expr)
				assert.NotNil(t, schedule)
			})
		}
	})

	t.Run("parses descriptors", func(t *testing.T) {
		descriptors := []string{
			"@yearly",
			"@annually",
			"@monthly",
			"@weekly",
			"@daily",
			"@midnight",
			"@hourly",
		}

		for _, desc := range descriptors {
			t.Run(desc, func(t *testing.T) {
				schedule, err := parser.Parse(desc)
				require.NoError(t, err, "Failed to parse: %s", desc)
				assert.NotNil(t, schedule)
			})
		}
	})

	t.Run("rejects invalid expressions", func(t *testing.T) {
		invalidExprs := []string{
			"invalid",
			"* * *",         // too few fields
			"* * * * * * *", // too many fields
			"60 * * * *",    // invalid minute
			"* 25 * * *",    // invalid hour
			"* * 32 * *",    // invalid day
			"* * * 13 *",    // invalid month
			"* * * * 8",     // invalid day of week
		}

		for _, expr := range invalidExprs {
			t.Run(expr, func(t *testing.T) {
				_, err := parser.Parse(expr)
				assert.Error(t, err, "Should reject: %s", expr)
			})
		}
	})
}

// =============================================================================
// Schedule Calculation Tests
// =============================================================================

func TestScheduleCalculation(t *testing.T) {
	parser := scheduler.StandardParser

	t.Run("every minute schedule", func(t *testing.T) {
		schedule, err := parser.Parse("* * * * *")
		require.NoError(t, err)

		now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		next := schedule.Next(now)

		assert.Equal(t, 2024, next.Year())
		assert.Equal(t, time.January, next.Month())
		assert.Equal(t, 15, next.Day())
		assert.Equal(t, 10, next.Hour())
		assert.Equal(t, 31, next.Minute())
	})

	t.Run("every 5 minutes schedule", func(t *testing.T) {
		schedule, err := parser.Parse("*/5 * * * *")
		require.NoError(t, err)

		now := time.Date(2024, 1, 15, 10, 32, 0, 0, time.UTC)
		next := schedule.Next(now)

		assert.Equal(t, 35, next.Minute())
	})

	t.Run("daily at midnight schedule", func(t *testing.T) {
		schedule, err := parser.Parse("0 0 * * *")
		require.NoError(t, err)

		now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		next := schedule.Next(now)

		assert.Equal(t, 16, next.Day())
		assert.Equal(t, 0, next.Hour())
		assert.Equal(t, 0, next.Minute())
	})

	t.Run("weekly schedule", func(t *testing.T) {
		schedule, err := parser.Parse("@weekly")
		require.NoError(t, err)

		now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		next := schedule.Next(now)

		assert.Equal(t, time.Sunday, next.Weekday())
	})
}

// =============================================================================
// Concurrent Execution Limits Tests
// =============================================================================

func TestConcurrentExecutionLimits(t *testing.T) {
	t.Run("default max concurrent is 10", func(t *testing.T) {
		s := NewScheduler(nil, "secret", "http://localhost", nil, nil)
		assert.Equal(t, 10, s.inner.Guard.MaxConcurrent)
	})

	t.Run("guard allows acquire when idle", func(t *testing.T) {
		s := NewScheduler(nil, "secret", "http://localhost", nil, nil)
		assert.True(t, s.inner.Guard.Acquire("test"))
		s.inner.Guard.Release()
	})
}

// =============================================================================
// Entry Tracking via Inner CronScheduler Tests
// =============================================================================

func TestEntryTracking(t *testing.T) {
	t.Run("empty on initialization", func(t *testing.T) {
		s := NewScheduler(nil, "secret", "http://localhost", nil, nil)
		assert.Equal(t, 0, s.inner.EntryCount())
	})

	t.Run("can schedule and check via IsScheduled", func(t *testing.T) {
		s := NewScheduler(nil, "secret", "http://localhost", nil, nil)

		sched := "*/5 * * * *"
		fn := EdgeFunctionSummary{
			ID:           uuid.New(),
			Name:         "test-function",
			Enabled:      true,
			CronSchedule: &sched,
		}
		err := s.ScheduleFunction(fn)
		require.NoError(t, err)

		assert.True(t, s.inner.IsScheduled("test-function"))
		assert.False(t, s.inner.IsScheduled("non-existent"))
	})

	t.Run("can remove scheduled function", func(t *testing.T) {
		s := NewScheduler(nil, "secret", "http://localhost", nil, nil)

		sched := "*/5 * * * *"
		fn := EdgeFunctionSummary{
			ID:           uuid.New(),
			Name:         "to-remove",
			Enabled:      true,
			CronSchedule: &sched,
		}
		err := s.ScheduleFunction(fn)
		require.NoError(t, err)
		assert.True(t, s.inner.IsScheduled("to-remove"))

		s.UnscheduleFunction("to-remove")
		assert.False(t, s.inner.IsScheduled("to-remove"))
	})
}

// =============================================================================
// Stop Tests
// =============================================================================

func TestScheduler_Stop(t *testing.T) {
	t.Run("stop cancels inner context", func(t *testing.T) {
		s := NewScheduler(nil, "secret", "http://localhost", nil, nil)

		s.Start()
		time.Sleep(50 * time.Millisecond)
		s.Stop()

		select {
		case <-s.inner.Context().Done():
			// Expected
		default:
			t.Error("Context should be cancelled after Stop()")
		}
	})
}

// =============================================================================
// Edge Function Scheduling Tests
// =============================================================================

func TestEdgeFunctionForScheduling(t *testing.T) {
	t.Run("function with cron schedule", func(t *testing.T) {
		schedule := "*/5 * * * *"
		fn := EdgeFunctionSummary{
			ID:           uuid.New(),
			Name:         "scheduled-function",
			Enabled:      true,
			CronSchedule: &schedule,
		}

		assert.True(t, fn.Enabled)
		assert.NotNil(t, fn.CronSchedule)
		assert.Equal(t, "*/5 * * * *", *fn.CronSchedule)
	})

	t.Run("disabled function with schedule", func(t *testing.T) {
		schedule := "0 * * * *"
		fn := EdgeFunctionSummary{
			ID:           uuid.New(),
			Name:         "disabled-scheduled",
			Enabled:      false,
			CronSchedule: &schedule,
		}

		assert.False(t, fn.Enabled)
		assert.NotNil(t, fn.CronSchedule)
	})

	t.Run("enabled function without schedule", func(t *testing.T) {
		fn := EdgeFunctionSummary{
			ID:      uuid.New(),
			Name:    "http-only",
			Enabled: true,
		}

		assert.True(t, fn.Enabled)
		assert.Nil(t, fn.CronSchedule)
	})
}

// =============================================================================
// Scheduler ScheduleFunction Validation Tests
// =============================================================================

func TestScheduleFunction_Validation(t *testing.T) {
	t.Run("valid schedule", func(t *testing.T) {
		s := NewScheduler(nil, "secret", "http://localhost", nil, nil)
		schedule := "*/5 * * * *"
		fn := EdgeFunctionSummary{
			ID:           uuid.New(),
			Name:         "valid-scheduled",
			Enabled:      true,
			CronSchedule: &schedule,
		}

		err := s.ScheduleFunction(fn)
		assert.NoError(t, err)
	})

	t.Run("invalid schedule expression", func(t *testing.T) {
		s := NewScheduler(nil, "secret", "http://localhost", nil, nil)
		invalidSchedule := "invalid cron"
		fn := EdgeFunctionSummary{
			ID:           uuid.New(),
			Name:         "invalid-scheduled",
			Enabled:      true,
			CronSchedule: &invalidSchedule,
		}

		err := s.ScheduleFunction(fn)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expected 5 to 6 fields")
	})

	t.Run("nil schedule", func(t *testing.T) {
		s := NewScheduler(nil, "secret", "http://localhost", nil, nil)
		fn := EdgeFunctionSummary{
			ID:      uuid.New(),
			Name:    "no-schedule",
			Enabled: true,
		}

		err := s.ScheduleFunction(fn)
		assert.NoError(t, err)
	})

	t.Run("empty schedule string", func(t *testing.T) {
		s := NewScheduler(nil, "secret", "http://localhost", nil, nil)
		emptySchedule := ""
		fn := EdgeFunctionSummary{
			ID:           uuid.New(),
			Name:         "empty-schedule",
			Enabled:      true,
			CronSchedule: &emptySchedule,
		}

		err := s.ScheduleFunction(fn)
		assert.NoError(t, err)
	})
}

// =============================================================================
// Scheduler UnscheduleFunction Tests
// =============================================================================

func TestUnscheduleFunction(t *testing.T) {
	t.Run("unschedule existing function", func(t *testing.T) {
		s := NewScheduler(nil, "secret", "http://localhost", nil, nil)

		schedule := "*/5 * * * *"
		fn := EdgeFunctionSummary{
			ID:           uuid.New(),
			Name:         "to-unschedule",
			Enabled:      true,
			CronSchedule: &schedule,
		}

		err := s.ScheduleFunction(fn)
		require.NoError(t, err)
		assert.True(t, s.inner.IsScheduled(fn.Name))

		s.UnscheduleFunction(fn.Name)
		assert.False(t, s.inner.IsScheduled(fn.Name))
	})

	t.Run("unschedule non-existent function", func(t *testing.T) {
		s := NewScheduler(nil, "secret", "http://localhost", nil, nil)

		s.UnscheduleFunction("non-existent")
	})
}

// =============================================================================
// Scheduler IsScheduled Tests
// =============================================================================

func TestIsScheduled(t *testing.T) {
	t.Run("returns true for scheduled function", func(t *testing.T) {
		s := NewScheduler(nil, "secret", "http://localhost", nil, nil)

		schedule := "*/5 * * * *"
		fn := EdgeFunctionSummary{
			ID:           uuid.New(),
			Name:         "scheduled-check",
			Enabled:      true,
			CronSchedule: &schedule,
		}

		err := s.ScheduleFunction(fn)
		require.NoError(t, err)

		assert.True(t, s.IsScheduled(fn.Name))
	})

	t.Run("returns false for unscheduled function", func(t *testing.T) {
		s := NewScheduler(nil, "secret", "http://localhost", nil, nil)
		assert.False(t, s.IsScheduled("not-scheduled"))
	})
}

// =============================================================================
// Scheduler GetScheduledFunctions Tests
// =============================================================================

func TestGetScheduledFunctions(t *testing.T) {
	t.Run("returns empty list initially", func(t *testing.T) {
		s := NewScheduler(nil, "secret", "http://localhost", nil, nil)
		functions := s.GetScheduledFunctions()
		assert.Empty(t, functions)
	})

	t.Run("returns scheduled function names", func(t *testing.T) {
		s := NewScheduler(nil, "secret", "http://localhost", nil, nil)

		schedules := []struct {
			name string
			cron string
		}{
			{"func-1", "*/5 * * * *"},
			{"func-2", "0 * * * *"},
			{"func-3", "0 0 * * *"},
		}

		for _, sc := range schedules {
			fn := EdgeFunctionSummary{
				ID:           uuid.New(),
				Name:         sc.name,
				Enabled:      true,
				CronSchedule: &sc.cron,
			}
			err := s.ScheduleFunction(fn)
			require.NoError(t, err)
		}

		functions := s.GetScheduledFunctions()
		assert.Len(t, functions, 3)
		assert.Contains(t, functions, "func-1")
		assert.Contains(t, functions, "func-2")
		assert.Contains(t, functions, "func-3")
	})
}
