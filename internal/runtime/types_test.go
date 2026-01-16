package runtime

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// RuntimeType Tests
// =============================================================================

func TestRuntimeType_Constants(t *testing.T) {
	t.Run("RuntimeTypeFunction is 0", func(t *testing.T) {
		assert.Equal(t, RuntimeType(0), RuntimeTypeFunction)
	})

	t.Run("RuntimeTypeJob is 1", func(t *testing.T) {
		assert.Equal(t, RuntimeType(1), RuntimeTypeJob)
	})

	t.Run("types are distinct", func(t *testing.T) {
		assert.NotEqual(t, RuntimeTypeFunction, RuntimeTypeJob)
	})
}

func TestRuntimeType_String(t *testing.T) {
	tests := []struct {
		name     string
		rt       RuntimeType
		expected string
	}{
		{
			name:     "RuntimeTypeFunction returns function",
			rt:       RuntimeTypeFunction,
			expected: "function",
		},
		{
			name:     "RuntimeTypeJob returns job",
			rt:       RuntimeTypeJob,
			expected: "job",
		},
		{
			name:     "unknown type returns unknown",
			rt:       RuntimeType(99),
			expected: "unknown",
		},
		{
			name:     "negative type returns unknown",
			rt:       RuntimeType(-1),
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.rt.String())
		})
	}
}

// =============================================================================
// ExecutionRequest Tests
// =============================================================================

func TestExecutionRequest_Struct(t *testing.T) {
	t.Run("zero value has empty fields", func(t *testing.T) {
		var req ExecutionRequest
		assert.Equal(t, uuid.UUID{}, req.ID)
		assert.Empty(t, req.Name)
		assert.Empty(t, req.Namespace)
		assert.Empty(t, req.UserID)
		assert.Empty(t, req.UserEmail)
		assert.Empty(t, req.UserRole)
		assert.Empty(t, req.BaseURL)
		assert.Empty(t, req.Method)
		assert.Empty(t, req.URL)
		assert.Nil(t, req.Headers)
		assert.Empty(t, req.Body)
		assert.Nil(t, req.Params)
		assert.Empty(t, req.SessionID)
		assert.Nil(t, req.Payload)
		assert.Zero(t, req.RetryCount)
	})

	t.Run("function request with HTTP context", func(t *testing.T) {
		id := uuid.New()
		req := ExecutionRequest{
			ID:        id,
			Name:      "my-function",
			Namespace: "default",
			UserID:    "user-123",
			UserEmail: "user@example.com",
			UserRole:  "authenticated",
			BaseURL:   "https://api.example.com",
			Method:    "POST",
			URL:       "/functions/my-function",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer token",
			},
			Body:      `{"key": "value"}`,
			Params:    map[string]string{"id": "123"},
			SessionID: "sess-456",
		}

		assert.Equal(t, id, req.ID)
		assert.Equal(t, "my-function", req.Name)
		assert.Equal(t, "default", req.Namespace)
		assert.Equal(t, "user-123", req.UserID)
		assert.Equal(t, "user@example.com", req.UserEmail)
		assert.Equal(t, "authenticated", req.UserRole)
		assert.Equal(t, "https://api.example.com", req.BaseURL)
		assert.Equal(t, "POST", req.Method)
		assert.Equal(t, "/functions/my-function", req.URL)
		assert.Len(t, req.Headers, 2)
		assert.Equal(t, "application/json", req.Headers["Content-Type"])
		assert.Equal(t, `{"key": "value"}`, req.Body)
		assert.Equal(t, "123", req.Params["id"])
		assert.Equal(t, "sess-456", req.SessionID)
	})

	t.Run("job request with payload context", func(t *testing.T) {
		id := uuid.New()
		req := ExecutionRequest{
			ID:        id,
			Name:      "my-job",
			Namespace: "production",
			UserID:    "user-789",
			Payload: map[string]interface{}{
				"type":    "process",
				"items":   []int{1, 2, 3},
				"options": map[string]bool{"verbose": true},
			},
			RetryCount: 2,
		}

		assert.Equal(t, id, req.ID)
		assert.Equal(t, "my-job", req.Name)
		assert.Equal(t, "production", req.Namespace)
		assert.Equal(t, "user-789", req.UserID)
		assert.Equal(t, "process", req.Payload["type"])
		assert.Equal(t, 2, req.RetryCount)
	})
}

// =============================================================================
// ExecutionResult Tests
// =============================================================================

func TestExecutionResult_Struct(t *testing.T) {
	t.Run("zero value is failure with no data", func(t *testing.T) {
		var result ExecutionResult
		assert.False(t, result.Success)
		assert.Empty(t, result.Error)
		assert.Empty(t, result.Logs)
		assert.Zero(t, result.DurationMs)
		assert.Zero(t, result.Status)
		assert.Nil(t, result.Headers)
		assert.Empty(t, result.Body)
		assert.Nil(t, result.Result)
	})

	t.Run("successful function result", func(t *testing.T) {
		result := ExecutionResult{
			Success:    true,
			Logs:       "Function executed successfully\n",
			DurationMs: 150,
			Status:     200,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: `{"data": "response"}`,
		}

		assert.True(t, result.Success)
		assert.Empty(t, result.Error)
		assert.Equal(t, "Function executed successfully\n", result.Logs)
		assert.Equal(t, int64(150), result.DurationMs)
		assert.Equal(t, 200, result.Status)
		assert.Equal(t, "application/json", result.Headers["Content-Type"])
		assert.Equal(t, `{"data": "response"}`, result.Body)
	})

	t.Run("failed function result", func(t *testing.T) {
		result := ExecutionResult{
			Success:    false,
			Error:      "Timeout exceeded",
			Logs:       "Starting execution...\nProcessing...\n",
			DurationMs: 30000,
			Status:     504,
		}

		assert.False(t, result.Success)
		assert.Equal(t, "Timeout exceeded", result.Error)
		assert.Contains(t, result.Logs, "Starting execution...")
		assert.Equal(t, int64(30000), result.DurationMs)
		assert.Equal(t, 504, result.Status)
	})

	t.Run("successful job result", func(t *testing.T) {
		result := ExecutionResult{
			Success:    true,
			Logs:       "Job completed",
			DurationMs: 5000,
			Result: map[string]interface{}{
				"processed": 100,
				"failed":    0,
				"output":    "batch_result.csv",
			},
		}

		assert.True(t, result.Success)
		assert.Equal(t, 100, result.Result["processed"])
		assert.Equal(t, 0, result.Result["failed"])
		assert.Equal(t, "batch_result.csv", result.Result["output"])
	})
}

// =============================================================================
// Progress Tests
// =============================================================================

func TestProgress_Struct(t *testing.T) {
	t.Run("zero value", func(t *testing.T) {
		var progress Progress
		assert.Zero(t, progress.Percent)
		assert.Empty(t, progress.Message)
		assert.Nil(t, progress.Data)
		assert.Nil(t, progress.EstimatedSecondsLeft)
	})

	t.Run("progress with all fields", func(t *testing.T) {
		estimatedSeconds := 120
		progress := Progress{
			Percent:              50,
			Message:              "Processing items...",
			Data:                 map[string]interface{}{"current": 500, "total": 1000},
			EstimatedSecondsLeft: &estimatedSeconds,
		}

		assert.Equal(t, 50, progress.Percent)
		assert.Equal(t, "Processing items...", progress.Message)
		assert.Equal(t, 500, progress.Data["current"])
		assert.Equal(t, 1000, progress.Data["total"])
		assert.NotNil(t, progress.EstimatedSecondsLeft)
		assert.Equal(t, 120, *progress.EstimatedSecondsLeft)
	})

	t.Run("progress at 0 percent", func(t *testing.T) {
		progress := Progress{
			Percent: 0,
			Message: "Starting...",
		}

		assert.Zero(t, progress.Percent)
		assert.Equal(t, "Starting...", progress.Message)
	})

	t.Run("progress at 100 percent", func(t *testing.T) {
		progress := Progress{
			Percent: 100,
			Message: "Complete",
		}

		assert.Equal(t, 100, progress.Percent)
		assert.Equal(t, "Complete", progress.Message)
	})

	t.Run("progress can exceed 100 percent", func(t *testing.T) {
		// This tests edge case where progress might be miscalculated
		progress := Progress{
			Percent: 150,
			Message: "Overcounted",
		}

		assert.Equal(t, 150, progress.Percent)
	})

	t.Run("progress can be negative", func(t *testing.T) {
		// Edge case - should be handled by validation elsewhere
		progress := Progress{
			Percent: -10,
			Message: "Invalid",
		}

		assert.Equal(t, -10, progress.Percent)
	})
}

// =============================================================================
// Permissions Tests
// =============================================================================

func TestPermissions_Struct(t *testing.T) {
	t.Run("zero value is all false/zero", func(t *testing.T) {
		var perms Permissions
		assert.False(t, perms.AllowNet)
		assert.False(t, perms.AllowEnv)
		assert.False(t, perms.AllowRead)
		assert.False(t, perms.AllowWrite)
		assert.Zero(t, perms.MemoryLimitMB)
	})

	t.Run("all permissions enabled", func(t *testing.T) {
		perms := Permissions{
			AllowNet:      true,
			AllowEnv:      true,
			AllowRead:     true,
			AllowWrite:    true,
			MemoryLimitMB: 1024,
		}

		assert.True(t, perms.AllowNet)
		assert.True(t, perms.AllowEnv)
		assert.True(t, perms.AllowRead)
		assert.True(t, perms.AllowWrite)
		assert.Equal(t, 1024, perms.MemoryLimitMB)
	})

	t.Run("restrictive permissions", func(t *testing.T) {
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
		assert.Equal(t, 128, perms.MemoryLimitMB)
	})
}

func TestDefaultPermissions(t *testing.T) {
	t.Run("returns expected defaults", func(t *testing.T) {
		perms := DefaultPermissions()

		assert.True(t, perms.AllowNet)
		assert.True(t, perms.AllowEnv)
		assert.False(t, perms.AllowRead)
		assert.False(t, perms.AllowWrite)
		assert.Zero(t, perms.MemoryLimitMB) // No memory limit set
	})

	t.Run("returns new instance each call", func(t *testing.T) {
		perms1 := DefaultPermissions()
		perms2 := DefaultPermissions()

		// Modify one
		perms1.AllowRead = true

		// Other should be unchanged
		assert.False(t, perms2.AllowRead)
	})
}

func TestDefaultFunctionPermissions(t *testing.T) {
	t.Run("returns expected defaults", func(t *testing.T) {
		perms := DefaultFunctionPermissions()

		assert.True(t, perms.AllowNet)
		assert.True(t, perms.AllowEnv)
		assert.False(t, perms.AllowRead)
		assert.False(t, perms.AllowWrite)
		assert.Equal(t, 512, perms.MemoryLimitMB) // Functions have 512MB default
	})

	t.Run("returns new instance each call", func(t *testing.T) {
		perms1 := DefaultFunctionPermissions()
		perms2 := DefaultFunctionPermissions()

		perms1.MemoryLimitMB = 1024

		assert.Equal(t, 512, perms2.MemoryLimitMB)
	})
}

func TestDefaultJobPermissions(t *testing.T) {
	t.Run("returns expected defaults", func(t *testing.T) {
		perms := DefaultJobPermissions()

		assert.True(t, perms.AllowNet)
		assert.True(t, perms.AllowEnv)
		assert.False(t, perms.AllowRead)
		assert.False(t, perms.AllowWrite)
		assert.Equal(t, 512, perms.MemoryLimitMB) // Jobs have 512MB default
	})

	t.Run("matches function permissions by default", func(t *testing.T) {
		funcPerms := DefaultFunctionPermissions()
		jobPerms := DefaultJobPermissions()

		assert.Equal(t, funcPerms.AllowNet, jobPerms.AllowNet)
		assert.Equal(t, funcPerms.AllowEnv, jobPerms.AllowEnv)
		assert.Equal(t, funcPerms.AllowRead, jobPerms.AllowRead)
		assert.Equal(t, funcPerms.AllowWrite, jobPerms.AllowWrite)
		assert.Equal(t, funcPerms.MemoryLimitMB, jobPerms.MemoryLimitMB)
	})

	t.Run("returns new instance each call", func(t *testing.T) {
		perms1 := DefaultJobPermissions()
		perms2 := DefaultJobPermissions()

		perms1.AllowWrite = true

		assert.False(t, perms2.AllowWrite)
	})
}

// =============================================================================
// Security Permission Combinations Tests
// =============================================================================

func TestPermissions_SecurityCombinations(t *testing.T) {
	t.Run("network-only is typical for APIs", func(t *testing.T) {
		perms := Permissions{
			AllowNet: true,
		}

		assert.True(t, perms.AllowNet)
		assert.False(t, perms.AllowRead)
		assert.False(t, perms.AllowWrite)
	})

	t.Run("read-only access pattern", func(t *testing.T) {
		perms := Permissions{
			AllowRead: true,
		}

		assert.True(t, perms.AllowRead)
		assert.False(t, perms.AllowWrite)
	})

	t.Run("write implies need for read usually", func(t *testing.T) {
		// Just testing struct allows it - validation is elsewhere
		perms := Permissions{
			AllowRead:  false,
			AllowWrite: true,
		}

		assert.False(t, perms.AllowRead)
		assert.True(t, perms.AllowWrite)
	})
}

// =============================================================================
// ExecutionRequest Field Combinations Tests
// =============================================================================

func TestExecutionRequest_FieldCombinations(t *testing.T) {
	t.Run("anonymous function request", func(t *testing.T) {
		req := ExecutionRequest{
			ID:     uuid.New(),
			Name:   "public-function",
			Method: "GET",
			URL:    "/functions/public-function",
			// No UserID, UserEmail, UserRole
		}

		assert.Empty(t, req.UserID)
		assert.Empty(t, req.UserEmail)
		assert.Empty(t, req.UserRole)
		assert.NotEmpty(t, req.Name)
	})

	t.Run("system-triggered job", func(t *testing.T) {
		req := ExecutionRequest{
			ID:         uuid.New(),
			Name:       "cleanup-job",
			Namespace:  "system",
			Payload:    map[string]interface{}{"trigger": "cron"},
			RetryCount: 0,
			// No user context
		}

		assert.Empty(t, req.UserID)
		assert.Equal(t, "system", req.Namespace)
		assert.Equal(t, "cron", req.Payload["trigger"])
	})

	t.Run("retry job with context", func(t *testing.T) {
		req := ExecutionRequest{
			ID:         uuid.New(),
			Name:       "email-job",
			Namespace:  "notifications",
			UserID:     "user-123",
			Payload:    map[string]interface{}{"to": "test@example.com"},
			RetryCount: 3,
		}

		assert.Equal(t, 3, req.RetryCount)
		assert.Equal(t, "user-123", req.UserID)
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkRuntimeType_String(b *testing.B) {
	types := []RuntimeType{RuntimeTypeFunction, RuntimeTypeJob, RuntimeType(99)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, t := range types {
			_ = t.String()
		}
	}
}

func BenchmarkDefaultPermissions(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DefaultPermissions()
	}
}

func BenchmarkDefaultFunctionPermissions(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DefaultFunctionPermissions()
	}
}

func BenchmarkDefaultJobPermissions(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DefaultJobPermissions()
	}
}

func BenchmarkExecutionRequest_Creation(b *testing.B) {
	id := uuid.New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ExecutionRequest{
			ID:        id,
			Name:      "test-function",
			Namespace: "default",
			Method:    "POST",
			URL:       "/functions/test",
			Headers:   map[string]string{"Content-Type": "application/json"},
			Body:      `{"test": true}`,
		}
	}
}

func BenchmarkExecutionResult_Creation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ExecutionResult{
			Success:    true,
			DurationMs: 150,
			Status:     200,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       `{"data": "result"}`,
		}
	}
}
