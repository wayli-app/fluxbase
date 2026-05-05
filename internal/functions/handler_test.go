package functions

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/config"

	"github.com/nimbleflux/fluxbase/internal/util"
)

// =============================================================================
// Handler Construction Tests
// =============================================================================

func TestNewHandler(t *testing.T) {
	t.Run("creates handler with nil dependencies", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "", "", nil, nil, nil, nil)
		assert.NotNil(t, handler)
		assert.Equal(t, "/tmp/functions", handler.functionsDir)
		assert.Equal(t, "http://localhost", handler.publicURL)
	})

	t.Run("creates handler with custom registries", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "https://npm.example.com", "https://jsr.example.com", nil, nil, nil, nil)
		assert.NotNil(t, handler)
		assert.Equal(t, "https://npm.example.com", handler.npmRegistry)
		assert.Equal(t, "https://jsr.example.com", handler.jsrRegistry)
	})
}

func TestHandler_SetScheduler(t *testing.T) {
	t.Run("sets scheduler reference", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "", "", nil, nil, nil, nil)
		assert.Nil(t, handler.scheduler)

		scheduler := &Scheduler{}
		handler.SetScheduler(scheduler)
		assert.NotNil(t, handler.scheduler)
		assert.Equal(t, scheduler, handler.scheduler)
	})
}

func TestHandler_GetRuntime(t *testing.T) {
	t.Run("returns runtime instance", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "", "", nil, nil, nil, nil)
		runtime := handler.GetRuntime()
		assert.NotNil(t, runtime)
	})
}

func TestHandler_GetPublicURL(t *testing.T) {
	t.Run("returns configured public URL", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "https://api.example.com", "", "", nil, nil, nil, nil)
		assert.Equal(t, "https://api.example.com", handler.GetPublicURL())
	})
}

func TestHandler_GetFunctionsDir(t *testing.T) {
	t.Run("returns configured functions directory", func(t *testing.T) {
		handler := NewHandler(nil, "/custom/path", config.CORSConfig{}, "secret", "http://localhost", "", "", nil, nil, nil, nil)
		assert.Equal(t, "/custom/path", handler.GetFunctionsDir())
	})
}

func TestHandler_createBundler(t *testing.T) {
	t.Run("creates bundler without custom registries", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "", "", nil, nil, nil, nil)
		bundler, err := handler.createBundler()
		require.NoError(t, err)
		assert.NotNil(t, bundler)
	})

	t.Run("creates bundler with npm registry", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "https://npm.example.com", "", nil, nil, nil, nil)
		bundler, err := handler.createBundler()
		require.NoError(t, err)
		assert.NotNil(t, bundler)
	})

	t.Run("creates bundler with jsr registry", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "", "https://jsr.example.com", nil, nil, nil, nil)
		bundler, err := handler.createBundler()
		require.NoError(t, err)
		assert.NotNil(t, bundler)
	})

	t.Run("creates bundler with both registries", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "https://npm.example.com", "https://jsr.example.com", nil, nil, nil, nil)
		bundler, err := handler.createBundler()
		require.NoError(t, err)
		assert.NotNil(t, bundler)
	})
}

// =============================================================================
// Log Message Handling Tests
// =============================================================================

func TestHandler_handleLogMessage(t *testing.T) {
	t.Run("handles log without counter", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "", "", nil, nil, nil, nil)
		execID := uuid.New()

		// Should not panic when no counter exists
		handler.handleLogMessage(execID, "info", "test message")
	})

	t.Run("increments counter when exists", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "", "", nil, nil, nil, nil)
		execID := uuid.New()

		ctx := &executionLogContext{}
		handler.logCounters.Store(execID, ctx)

		handler.handleLogMessage(execID, "info", "message 1")
		assert.Equal(t, 1, ctx.lineCounter)

		handler.handleLogMessage(execID, "info", "message 2")
		assert.Equal(t, 2, ctx.lineCounter)
	})

	t.Run("handles invalid counter type", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "", "", nil, nil, nil, nil)
		execID := uuid.New()

		// Store invalid type
		handler.logCounters.Store(execID, "not a pointer")

		// Should not panic
		handler.handleLogMessage(execID, "info", "test message")
	})
}

// =============================================================================
// EdgeFunction Struct Tests
// =============================================================================

func TestEdgeFunction_Struct(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		fn := EdgeFunction{
			Name: "test-function",
			Code: "export default () => 'hello'",
		}

		assert.Equal(t, "test-function", fn.Name)
		assert.Equal(t, "export default () => 'hello'", fn.Code)
		assert.False(t, fn.Enabled)
		assert.False(t, fn.AllowNet)
		assert.False(t, fn.AllowEnv)
		assert.False(t, fn.AllowUnauthenticated)
	})

	t.Run("with all fields", func(t *testing.T) {
		description := "A test function"
		cronSchedule := "*/5 * * * *"
		corsOrigins := "*"
		rateLimit := 100

		fn := EdgeFunction{
			ID:                   uuid.New(),
			Name:                 "complete-function",
			Namespace:            "default",
			Description:          &description,
			Code:                 "export default () => 'hello'",
			Version:              1,
			CronSchedule:         &cronSchedule,
			Enabled:              true,
			TimeoutSeconds:       30,
			MemoryLimitMB:        128,
			AllowNet:             true,
			AllowEnv:             true,
			AllowRead:            true,
			AllowWrite:           false,
			AllowUnauthenticated: false,
			IsPublic:             true,
			CorsOrigins:          &corsOrigins,
			RateLimitPerMinute:   &rateLimit,
			CreatedAt:            time.Now(),
			UpdatedAt:            time.Now(),
			Source:               "api",
		}

		assert.NotEqual(t, uuid.Nil, fn.ID)
		assert.Equal(t, "complete-function", fn.Name)
		assert.Equal(t, "default", fn.Namespace)
		assert.Equal(t, "A test function", *fn.Description)
		assert.Equal(t, "*/5 * * * *", *fn.CronSchedule)
		assert.True(t, fn.Enabled)
		assert.Equal(t, 30, fn.TimeoutSeconds)
		assert.Equal(t, 128, fn.MemoryLimitMB)
		assert.True(t, fn.AllowNet)
		assert.True(t, fn.AllowEnv)
		assert.Equal(t, 100, *fn.RateLimitPerMinute)
		assert.Equal(t, "api", fn.Source)
	})

	t.Run("JSON serialization", func(t *testing.T) {
		fn := EdgeFunction{
			ID:             uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			Name:           "json-test",
			Code:           "export default () => 'hello'",
			Enabled:        true,
			TimeoutSeconds: 30,
			MemoryLimitMB:  128,
		}

		data, err := json.Marshal(fn)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"name":"json-test"`)
		assert.Contains(t, string(data), `"enabled":true`)
		assert.Contains(t, string(data), `"timeout_seconds":30`)
		assert.Contains(t, string(data), `"memory_limit_mb":128`)
	})
}

// =============================================================================
// EdgeFunctionSummary Struct Tests
// =============================================================================

func TestEdgeFunctionSummary_Struct(t *testing.T) {
	t.Run("excludes code fields", func(t *testing.T) {
		summary := EdgeFunctionSummary{
			ID:             uuid.New(),
			Name:           "summary-test",
			Enabled:        true,
			TimeoutSeconds: 30,
			Source:         "filesystem",
		}

		data, err := json.Marshal(summary)
		require.NoError(t, err)

		// Should not contain code field
		assert.NotContains(t, string(data), `"code"`)
		assert.Contains(t, string(data), `"name":"summary-test"`)
	})
}

// =============================================================================
// EdgeFunctionExecution Struct Tests
// =============================================================================

func TestEdgeFunctionExecution_Struct(t *testing.T) {
	t.Run("pending execution", func(t *testing.T) {
		exec := EdgeFunctionExecution{
			ID:          uuid.New(),
			FunctionID:  uuid.New(),
			TriggerType: "http",
			Status:      "pending",
			ExecutedAt:  time.Now(),
		}

		assert.Equal(t, "pending", exec.Status)
		assert.Equal(t, "http", exec.TriggerType)
		assert.Nil(t, exec.CompletedAt)
	})

	t.Run("completed execution", func(t *testing.T) {
		now := time.Now()
		duration := 150
		statusCode := 200
		result := `{"message": "success"}`

		exec := EdgeFunctionExecution{
			ID:          uuid.New(),
			FunctionID:  uuid.New(),
			TriggerType: "cron",
			Status:      "success",
			StatusCode:  &statusCode,
			DurationMs:  &duration,
			Result:      &result,
			ExecutedAt:  now,
			CompletedAt: &now,
		}

		assert.Equal(t, "success", exec.Status)
		assert.Equal(t, "cron", exec.TriggerType)
		assert.Equal(t, 200, *exec.StatusCode)
		assert.Equal(t, 150, *exec.DurationMs)
		assert.NotNil(t, exec.CompletedAt)
	})

	t.Run("failed execution", func(t *testing.T) {
		errorMsg := "Function timeout exceeded"
		errorStack := "Error: timeout\n    at execute (function.ts:10)"

		exec := EdgeFunctionExecution{
			ID:           uuid.New(),
			FunctionID:   uuid.New(),
			TriggerType:  "http",
			Status:       "error",
			ErrorMessage: &errorMsg,
			ErrorStack:   &errorStack,
			ExecutedAt:   time.Now(),
		}

		assert.Equal(t, "error", exec.Status)
		assert.Equal(t, "Function timeout exceeded", *exec.ErrorMessage)
		assert.Contains(t, *exec.ErrorStack, "timeout")
	})
}

// =============================================================================
// FunctionFile Struct Tests
// =============================================================================

func TestFunctionFile_Struct(t *testing.T) {
	t.Run("supporting file", func(t *testing.T) {
		file := FunctionFile{
			ID:         uuid.New(),
			FunctionID: uuid.New(),
			FilePath:   "utils/helper.ts",
			Content:    "export const add = (a, b) => a + b;",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		assert.Equal(t, "utils/helper.ts", file.FilePath)
		assert.Contains(t, file.Content, "export const add")
	})

	t.Run("JSON serialization", func(t *testing.T) {
		file := FunctionFile{
			ID:         uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			FunctionID: uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
			FilePath:   "db.ts",
			Content:    "export const query = () => {}",
		}

		data, err := json.Marshal(file)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"file_path":"db.ts"`)
		// Note: JSON encoder escapes > as \u003e for HTML safety
		assert.Contains(t, string(data), `"content":"export const query = () =`)
	})
}

// =============================================================================
// SharedModule Struct Tests
// =============================================================================

func TestSharedModule_Struct(t *testing.T) {
	t.Run("basic shared module", func(t *testing.T) {
		module := SharedModule{
			ID:         uuid.New(),
			ModulePath: "_shared/cors.ts",
			Content:    "export const corsHeaders = { ... };",
			Version:    1,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		assert.Equal(t, "_shared/cors.ts", module.ModulePath)
		assert.Equal(t, 1, module.Version)
	})

	t.Run("with description and creator", func(t *testing.T) {
		description := "CORS utilities for edge functions"
		creatorID := uuid.New()

		module := SharedModule{
			ID:          uuid.New(),
			ModulePath:  "_shared/utils/http.ts",
			Content:     "export const parseBody = (req) => { ... };",
			Description: &description,
			Version:     2,
			CreatedBy:   &creatorID,
		}

		assert.Equal(t, "CORS utilities for edge functions", *module.Description)
		assert.Equal(t, 2, module.Version)
		assert.Equal(t, creatorID, *module.CreatedBy)
	})
}

// =============================================================================
// CORS Configuration Tests
// =============================================================================

func TestCORSConfig(t *testing.T) {
	t.Run("default CORS config", func(t *testing.T) {
		corsConfig := config.CORSConfig{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Content-Type", "Authorization"},
			AllowCredentials: false,
			MaxAge:           3600,
		}

		assert.Contains(t, corsConfig.AllowedOrigins, "*")
		assert.Contains(t, corsConfig.AllowedMethods, "POST")
		assert.False(t, corsConfig.AllowCredentials)
		assert.Equal(t, 3600, corsConfig.MaxAge)
	})

	t.Run("restrictive CORS config", func(t *testing.T) {
		corsConfig := config.CORSConfig{
			AllowedOrigins:   []string{"https://app.example.com", "https://admin.example.com"},
			AllowedMethods:   []string{"GET", "POST"},
			AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Custom-Header"},
			AllowCredentials: true,
			MaxAge:           7200,
			ExposedHeaders:   []string{"X-Request-Id"},
		}

		assert.Contains(t, corsConfig.AllowedOrigins, "https://app.example.com")
		assert.True(t, corsConfig.AllowCredentials)
		assert.Contains(t, corsConfig.ExposedHeaders, "X-Request-Id")
	})
}

// =============================================================================
// Rate Limiting Configuration Tests
// =============================================================================

func TestRateLimitConfiguration(t *testing.T) {
	t.Run("per-minute rate limit", func(t *testing.T) {
		limit := 60
		fn := EdgeFunction{
			Name:               "rate-limited",
			RateLimitPerMinute: &limit,
		}

		assert.Equal(t, 60, *fn.RateLimitPerMinute)
		assert.Nil(t, fn.RateLimitPerHour)
		assert.Nil(t, fn.RateLimitPerDay)
	})

	t.Run("multiple rate limits", func(t *testing.T) {
		perMin := 10
		perHour := 100
		perDay := 1000

		fn := EdgeFunction{
			Name:               "multi-rate-limited",
			RateLimitPerMinute: &perMin,
			RateLimitPerHour:   &perHour,
			RateLimitPerDay:    &perDay,
		}

		assert.Equal(t, 10, *fn.RateLimitPerMinute)
		assert.Equal(t, 100, *fn.RateLimitPerHour)
		assert.Equal(t, 1000, *fn.RateLimitPerDay)
	})

	t.Run("no rate limits (unlimited)", func(t *testing.T) {
		fn := EdgeFunction{
			Name: "unlimited",
		}

		assert.Nil(t, fn.RateLimitPerMinute)
		assert.Nil(t, fn.RateLimitPerHour)
		assert.Nil(t, fn.RateLimitPerDay)
	})
}

// =============================================================================
// Cron Schedule Tests
// =============================================================================

func TestCronSchedule(t *testing.T) {
	t.Run("valid cron expressions", func(t *testing.T) {
		schedules := []string{
			"* * * * *",     // Every minute
			"*/5 * * * *",   // Every 5 minutes
			"0 * * * *",     // Every hour
			"0 0 * * *",     // Every day at midnight
			"0 0 * * 0",     // Every Sunday at midnight
			"0 0 1 * *",     // First of every month
			"0 0 1 1 *",     // January 1st
			"0 */5 * * * *", // Every 5 minutes (6-field with seconds)
			"30 0 0 * * *",  // Every day at midnight + 30 seconds
		}

		for _, schedule := range schedules {
			fn := EdgeFunction{
				Name:         "scheduled-fn",
				CronSchedule: &schedule,
			}
			assert.Equal(t, schedule, *fn.CronSchedule)
		}
	})

	t.Run("function without schedule", func(t *testing.T) {
		fn := EdgeFunction{
			Name: "http-only",
		}

		assert.Nil(t, fn.CronSchedule)
	})
}

// =============================================================================
// Function Permissions Tests
// =============================================================================

func TestFunctionPermissions(t *testing.T) {
	t.Run("minimal permissions", func(t *testing.T) {
		fn := EdgeFunction{
			Name:                 "minimal",
			AllowNet:             false,
			AllowEnv:             false,
			AllowRead:            false,
			AllowWrite:           false,
			AllowUnauthenticated: false,
		}

		assert.False(t, fn.AllowNet)
		assert.False(t, fn.AllowEnv)
		assert.False(t, fn.AllowRead)
		assert.False(t, fn.AllowWrite)
		assert.False(t, fn.AllowUnauthenticated)
	})

	t.Run("full permissions", func(t *testing.T) {
		fn := EdgeFunction{
			Name:                 "full-access",
			AllowNet:             true,
			AllowEnv:             true,
			AllowRead:            true,
			AllowWrite:           true,
			AllowUnauthenticated: true,
			IsPublic:             true,
		}

		assert.True(t, fn.AllowNet)
		assert.True(t, fn.AllowEnv)
		assert.True(t, fn.AllowRead)
		assert.True(t, fn.AllowWrite)
		assert.True(t, fn.AllowUnauthenticated)
		assert.True(t, fn.IsPublic)
	})
}

// =============================================================================
// Resource Limits Tests
// =============================================================================

func TestResourceLimits(t *testing.T) {
	t.Run("default limits", func(t *testing.T) {
		fn := EdgeFunction{
			Name:           "default-limits",
			TimeoutSeconds: 30,
			MemoryLimitMB:  128,
		}

		assert.Equal(t, 30, fn.TimeoutSeconds)
		assert.Equal(t, 128, fn.MemoryLimitMB)
	})

	t.Run("custom limits", func(t *testing.T) {
		fn := EdgeFunction{
			Name:           "custom-limits",
			TimeoutSeconds: 300,
			MemoryLimitMB:  512,
		}

		assert.Equal(t, 300, fn.TimeoutSeconds)
		assert.Equal(t, 512, fn.MemoryLimitMB)
	})

	t.Run("minimal limits", func(t *testing.T) {
		fn := EdgeFunction{
			Name:           "minimal-limits",
			TimeoutSeconds: 1,
			MemoryLimitMB:  32,
		}

		assert.Equal(t, 1, fn.TimeoutSeconds)
		assert.Equal(t, 32, fn.MemoryLimitMB)
	})
}

// =============================================================================
// Trigger Types Tests
// =============================================================================

func TestTriggerTypes(t *testing.T) {
	t.Run("HTTP trigger", func(t *testing.T) {
		exec := EdgeFunctionExecution{
			TriggerType: "http",
		}
		assert.Equal(t, "http", exec.TriggerType)
	})

	t.Run("cron trigger", func(t *testing.T) {
		exec := EdgeFunctionExecution{
			TriggerType: "cron",
		}
		assert.Equal(t, "cron", exec.TriggerType)
	})

	t.Run("webhook trigger", func(t *testing.T) {
		exec := EdgeFunctionExecution{
			TriggerType: "webhook",
		}
		assert.Equal(t, "webhook", exec.TriggerType)
	})
}

// =============================================================================
// Execution Status Tests
// =============================================================================

func TestExecutionStatus(t *testing.T) {
	statuses := []string{
		"pending",
		"running",
		"success",
		"error",
		"timeout",
		"cancelled",
	}

	for _, status := range statuses {
		t.Run(status, func(t *testing.T) {
			exec := EdgeFunctionExecution{
				Status: status,
			}
			assert.Equal(t, status, exec.Status)
		})
	}
}

// =============================================================================
// Source Types Tests
// =============================================================================

func TestSourceTypes(t *testing.T) {
	t.Run("filesystem source", func(t *testing.T) {
		fn := EdgeFunction{
			Name:   "fs-function",
			Source: "filesystem",
		}
		assert.Equal(t, "filesystem", fn.Source)
	})

	t.Run("API source", func(t *testing.T) {
		fn := EdgeFunction{
			Name:   "api-function",
			Source: "api",
		}
		assert.Equal(t, "api", fn.Source)
	})
}

// =============================================================================
// Helper Function Tests - isAdminRole
// =============================================================================

func TestIsAdminRole(t *testing.T) {
	t.Run("admin role is admin", func(t *testing.T) {
		assert.True(t, isAdminRole("admin"))
	})

	t.Run("instance_admin role is admin", func(t *testing.T) {
		assert.True(t, isAdminRole("instance_admin"))
	})

	t.Run("service_role is admin", func(t *testing.T) {
		assert.True(t, isAdminRole("service_role"))
	})

	t.Run("authenticated role is not admin", func(t *testing.T) {
		assert.False(t, isAdminRole("authenticated"))
	})

	t.Run("anon role is not admin", func(t *testing.T) {
		assert.False(t, isAdminRole("anon"))
	})

	t.Run("empty role is not admin", func(t *testing.T) {
		assert.False(t, isAdminRole(""))
	})

	t.Run("user role is not admin", func(t *testing.T) {
		assert.False(t, isAdminRole("user"))
	})

	t.Run("case sensitive - ADMIN is not admin", func(t *testing.T) {
		assert.False(t, isAdminRole("ADMIN"))
	})

	t.Run("case sensitive - Admin is not admin", func(t *testing.T) {
		assert.False(t, isAdminRole("Admin"))
	})
}

// =============================================================================
// Helper Function Tests - valueOr
// =============================================================================

func TestValueOr(t *testing.T) {
	t.Run("returns value when pointer is non-nil (int)", func(t *testing.T) {
		val := 42
		result := util.ValueOr(&val, 0)
		assert.Equal(t, 42, result)
	})

	t.Run("returns default when pointer is nil (int)", func(t *testing.T) {
		var ptr *int
		result := util.ValueOr(ptr, 100)
		assert.Equal(t, 100, result)
	})

	t.Run("returns value when pointer is non-nil (string)", func(t *testing.T) {
		val := "hello"
		result := util.ValueOr(&val, "default")
		assert.Equal(t, "hello", result)
	})

	t.Run("returns default when pointer is nil (string)", func(t *testing.T) {
		var ptr *string
		result := util.ValueOr(ptr, "default")
		assert.Equal(t, "default", result)
	})

	t.Run("returns value when pointer is non-nil (bool)", func(t *testing.T) {
		val := true
		result := util.ValueOr(&val, false)
		assert.True(t, result)
	})

	t.Run("returns default when pointer is nil (bool)", func(t *testing.T) {
		var ptr *bool
		result := util.ValueOr(ptr, true)
		assert.True(t, result)
	})

	t.Run("returns zero value when set", func(t *testing.T) {
		val := 0
		result := util.ValueOr(&val, 42)
		assert.Equal(t, 0, result)
	})

	t.Run("returns empty string when set", func(t *testing.T) {
		val := ""
		result := util.ValueOr(&val, "default")
		assert.Equal(t, "", result)
	})

	t.Run("returns false when set to false", func(t *testing.T) {
		val := false
		result := util.ValueOr(&val, true)
		assert.False(t, result)
	})
}

// =============================================================================
// Helper Function Tests - truncateString
// =============================================================================

func TestTruncateString(t *testing.T) {
	t.Run("returns string unchanged if shorter than maxLen", func(t *testing.T) {
		result := util.TruncateString("hello", 10)
		assert.Equal(t, "hello", result)
	})

	t.Run("returns string unchanged if equal to maxLen", func(t *testing.T) {
		result := util.TruncateString("hello", 5)
		assert.Equal(t, "hello", result)
	})

	t.Run("truncates string if longer than maxLen", func(t *testing.T) {
		result := util.TruncateString("hello world", 5)
		assert.Equal(t, "hello...", result)
	})

	t.Run("handles empty string", func(t *testing.T) {
		result := util.TruncateString("", 10)
		assert.Equal(t, "", result)
	})

	t.Run("handles maxLen of 0", func(t *testing.T) {
		result := util.TruncateString("hello", 0)
		assert.Equal(t, "...", result)
	})

	t.Run("handles maxLen of 1", func(t *testing.T) {
		result := util.TruncateString("hello", 1)
		assert.Equal(t, "h...", result)
	})

	t.Run("handles very long string", func(t *testing.T) {
		longStr := "This is a very long string that should be truncated"
		result := util.TruncateString(longStr, 10)
		assert.Equal(t, "This is a ...", result)
		assert.Len(t, result, 13) // 10 + "..."
	})

	t.Run("handles unicode strings", func(t *testing.T) {
		result := util.TruncateString("hello 世界", 8)
		// Note: truncateString works on byte length, not rune count
		assert.Contains(t, result, "...")
	})
}

// =============================================================================
// Helper Function Tests - toString
// =============================================================================

func TestToString(t *testing.T) {
	t.Run("returns empty string for nil interface", func(t *testing.T) {
		result := util.ToString(nil)
		assert.Equal(t, "", result)
	})

	t.Run("returns string as-is", func(t *testing.T) {
		result := util.ToString("hello")
		assert.Equal(t, "hello", result)
	})

	t.Run("returns empty string for nil uuid pointer", func(t *testing.T) {
		var uid *uuid.UUID
		result := util.ToString(uid)
		assert.Equal(t, "", result)
	})

	t.Run("returns uuid string for non-nil uuid pointer", func(t *testing.T) {
		uid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		result := util.ToString(&uid)
		assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", result)
	})

	t.Run("converts int to string", func(t *testing.T) {
		result := util.ToString(42)
		assert.Equal(t, "42", result)
	})

	t.Run("converts float to string", func(t *testing.T) {
		result := util.ToString(3.14)
		assert.Equal(t, "3.14", result)
	})

	t.Run("converts bool to string", func(t *testing.T) {
		result := util.ToString(true)
		assert.Equal(t, "true", result)
	})

	t.Run("converts struct to string", func(t *testing.T) {
		type testStruct struct {
			Name string
		}
		result := util.ToString(testStruct{Name: "test"})
		assert.Contains(t, result, "test")
	})
}

// =============================================================================
// Helper Function Tests - normalizeSettingsKey
// =============================================================================

func TestNormalizeSettingsKey(t *testing.T) {
	t.Run("simple key with underscores", func(t *testing.T) {
		result := normalizeSettingsKey("openai_api_key")
		assert.Equal(t, "OPENAI_API_KEY", result)
	})

	t.Run("key with dots", func(t *testing.T) {
		result := normalizeSettingsKey("ai.openai.api_key")
		assert.Equal(t, "AI_OPENAI_API_KEY", result)
	})

	t.Run("already uppercase", func(t *testing.T) {
		result := normalizeSettingsKey("API_KEY")
		assert.Equal(t, "API_KEY", result)
	})

	t.Run("mixed case", func(t *testing.T) {
		result := normalizeSettingsKey("apiKey")
		assert.Equal(t, "APIKEY", result)
	})

	t.Run("empty string", func(t *testing.T) {
		result := normalizeSettingsKey("")
		assert.Equal(t, "", result)
	})

	t.Run("multiple dots", func(t *testing.T) {
		result := normalizeSettingsKey("app.config.database.host")
		assert.Equal(t, "APP_CONFIG_DATABASE_HOST", result)
	})

	t.Run("dots and underscores mixed", func(t *testing.T) {
		result := normalizeSettingsKey("app.api_config.max_retries")
		assert.Equal(t, "APP_API_CONFIG_MAX_RETRIES", result)
	})

	t.Run("single character", func(t *testing.T) {
		result := normalizeSettingsKey("a")
		assert.Equal(t, "A", result)
	})

	t.Run("numeric suffix", func(t *testing.T) {
		result := normalizeSettingsKey("retry_limit_v2")
		assert.Equal(t, "RETRY_LIMIT_V2", result)
	})
}

// =============================================================================
// AdminExecutionFilters Struct Tests
// =============================================================================

func TestAdminExecutionFilters_Struct(t *testing.T) {
	t.Run("zero values", func(t *testing.T) {
		filters := AdminExecutionFilters{}

		assert.Empty(t, filters.Namespace)
		assert.Empty(t, filters.FunctionName)
		assert.Empty(t, filters.Status)
		assert.Equal(t, 0, filters.Limit)
		assert.Equal(t, 0, filters.Offset)
	})

	t.Run("all fields set", func(t *testing.T) {
		filters := AdminExecutionFilters{
			Namespace:    "production",
			FunctionName: "my-function",
			Status:       "success",
			Limit:        50,
			Offset:       100,
		}

		assert.Equal(t, "production", filters.Namespace)
		assert.Equal(t, "my-function", filters.FunctionName)
		assert.Equal(t, "success", filters.Status)
		assert.Equal(t, 50, filters.Limit)
		assert.Equal(t, 100, filters.Offset)
	})

	t.Run("namespace filter only", func(t *testing.T) {
		filters := AdminExecutionFilters{
			Namespace: "staging",
			Limit:     25,
		}

		assert.Equal(t, "staging", filters.Namespace)
		assert.Empty(t, filters.FunctionName)
		assert.Empty(t, filters.Status)
		assert.Equal(t, 25, filters.Limit)
	})

	t.Run("status filter only", func(t *testing.T) {
		filters := AdminExecutionFilters{
			Status: "error",
			Limit:  10,
		}

		assert.Empty(t, filters.Namespace)
		assert.Empty(t, filters.FunctionName)
		assert.Equal(t, "error", filters.Status)
	})

	t.Run("pagination only", func(t *testing.T) {
		filters := AdminExecutionFilters{
			Limit:  20,
			Offset: 40,
		}

		assert.Equal(t, 20, filters.Limit)
		assert.Equal(t, 40, filters.Offset)
	})

	t.Run("valid status values", func(t *testing.T) {
		statuses := []string{"pending", "running", "success", "error", "timeout", "cancelled"}
		for _, status := range statuses {
			filters := AdminExecutionFilters{Status: status}
			assert.Equal(t, status, filters.Status)
		}
	})
}

// =============================================================================
// AdminExecution Struct Tests
// =============================================================================

func TestAdminExecution_Struct(t *testing.T) {
	t.Run("success execution", func(t *testing.T) {
		now := time.Now()
		completedAt := now.Add(5 * time.Second)
		duration := 5000
		statusCode := 200
		result := `{"status": "ok"}`

		exec := AdminExecution{
			EdgeFunctionExecution: EdgeFunctionExecution{
				ID:          uuid.New(),
				FunctionID:  uuid.New(),
				TriggerType: "http",
				Status:      "success",
				StatusCode:  &statusCode,
				DurationMs:  &duration,
				Result:      &result,
				ExecutedAt:  now,
				CompletedAt: &completedAt,
			},
			FunctionName: "test-function",
			Namespace:    "default",
		}

		assert.Equal(t, "success", exec.Status)
		assert.Equal(t, 200, *exec.StatusCode)
		assert.Equal(t, 5000, *exec.DurationMs)
		assert.Equal(t, "test-function", exec.FunctionName)
		assert.Equal(t, "default", exec.Namespace)
	})

	t.Run("error execution", func(t *testing.T) {
		errorMsg := "Function failed"
		logs := "Error: something went wrong"

		exec := AdminExecution{
			EdgeFunctionExecution: EdgeFunctionExecution{
				ID:           uuid.New(),
				FunctionID:   uuid.New(),
				TriggerType:  "cron",
				Status:       "error",
				ErrorMessage: &errorMsg,
				Logs:         &logs,
			},
			FunctionName: "scheduled-task",
			Namespace:    "workers",
		}

		assert.Equal(t, "error", exec.Status)
		assert.Equal(t, "Function failed", *exec.ErrorMessage)
		assert.Equal(t, "Error: something went wrong", *exec.Logs)
		assert.Equal(t, "scheduled-task", exec.FunctionName)
		assert.Equal(t, "workers", exec.Namespace)
	})

	t.Run("pending execution", func(t *testing.T) {
		exec := AdminExecution{
			EdgeFunctionExecution: EdgeFunctionExecution{
				ID:          uuid.New(),
				FunctionID:  uuid.New(),
				TriggerType: "http",
				Status:      "pending",
				ExecutedAt:  time.Now(),
			},
			FunctionName: "new-function",
			Namespace:    "test",
		}

		assert.Equal(t, "pending", exec.Status)
		assert.Nil(t, exec.CompletedAt)
		assert.Nil(t, exec.DurationMs)
		assert.Nil(t, exec.StatusCode)
	})
}

// =============================================================================
// Handler SetSettingsSecretsService Tests
// =============================================================================

func TestHandler_SetSettingsSecretsService(t *testing.T) {
	t.Run("sets settings secrets service", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "", "", nil, nil, nil, nil)
		assert.Nil(t, handler.settingsSecretsService)

		// Note: We can't create a real SecretsService without a database,
		// but we verify the method exists and nil case is handled
		handler.SetSettingsSecretsService(nil)
		assert.Nil(t, handler.settingsSecretsService)
	})
}

// =============================================================================
// Handler loadSettingsSecrets Tests
// =============================================================================

func TestHandler_loadSettingsSecrets(t *testing.T) {
	t.Run("returns nil when service is nil", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "", "", nil, nil, nil, nil)

		result := handler.loadSettingsSecrets(context.TODO(), nil)
		assert.Nil(t, result)
	})

	t.Run("returns nil when service is nil with user ID", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "", "", nil, nil, nil, nil)

		userID := uuid.New()
		result := handler.loadSettingsSecrets(context.TODO(), &userID)
		assert.Nil(t, result)
	})
}

// =============================================================================
// Handler CORS Configuration Tests
// =============================================================================

func TestHandler_CORSConfig(t *testing.T) {
	t.Run("stores CORS config", func(t *testing.T) {
		corsConfig := config.CORSConfig{
			AllowedOrigins:   []string{"https://example.com"},
			AllowedMethods:   []string{"GET", "POST"},
			AllowedHeaders:   []string{"Content-Type"},
			AllowCredentials: true,
			MaxAge:           3600,
			ExposedHeaders:   []string{"X-Request-Id"},
		}

		handler := NewHandler(nil, "/tmp/functions", corsConfig, "secret", "http://localhost", "", "", nil, nil, nil, nil)

		assert.Equal(t, []string{"https://example.com"}, handler.corsConfig.AllowedOrigins)
		assert.Equal(t, []string{"GET", "POST"}, handler.corsConfig.AllowedMethods)
		assert.True(t, handler.corsConfig.AllowCredentials)
		assert.Equal(t, 3600, handler.corsConfig.MaxAge)
	})
}

// =============================================================================
// Handler Log Counter Tests
// =============================================================================

func TestHandler_LogCounters(t *testing.T) {
	t.Run("log counters are thread-safe", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "", "", nil, nil, nil, nil)

		// Run concurrent operations
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func(id int) {
				execID := uuid.New()
				handler.logCounters.Store(execID, &executionLogContext{})

				for j := 0; j < 100; j++ {
					handler.handleLogMessage(execID, "info", "message")
				}

				handler.logCounters.Delete(execID)
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		// Should complete without panics
		assert.NotNil(t, handler)
	})
}

// =============================================================================
// Bundled Status Tests
// =============================================================================

func TestEdgeFunction_BundleStatus(t *testing.T) {
	t.Run("pre-bundled from client", func(t *testing.T) {
		original := "import { x } from './utils'; export default () => x;"
		bundled := "const x = 1; export default () => x;"

		fn := EdgeFunction{
			Name:         "pre-bundled",
			Code:         bundled,
			OriginalCode: &original,
			IsBundled:    true,
			Source:       "api",
		}

		assert.True(t, fn.IsBundled)
		assert.Equal(t, bundled, fn.Code)
		assert.Equal(t, original, *fn.OriginalCode)
	})

	t.Run("server-side bundled", func(t *testing.T) {
		original := "import { serve } from 'https://deno.land/std/http/server.ts';"

		fn := EdgeFunction{
			Name:         "server-bundled",
			Code:         "// bundled content",
			OriginalCode: &original,
			IsBundled:    true,
			Source:       "filesystem",
		}

		assert.True(t, fn.IsBundled)
		assert.NotEqual(t, fn.Code, *fn.OriginalCode)
	})

	t.Run("unbundled simple function", func(t *testing.T) {
		code := "export default () => ({ hello: 'world' });"

		fn := EdgeFunction{
			Name:         "simple",
			Code:         code,
			OriginalCode: &code,
			IsBundled:    false,
		}

		assert.False(t, fn.IsBundled)
		assert.Equal(t, code, fn.Code)
		assert.Equal(t, code, *fn.OriginalCode)
	})
}
