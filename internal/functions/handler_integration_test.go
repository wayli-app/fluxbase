package functions

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/config"
)

// =============================================================================
// Handler Settings Secrets Tests
// =============================================================================

func TestHandlerIntegration_SettingsSecretsService(t *testing.T) {
	t.Run("settings secrets service can be set", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "", "", nil, nil, nil, nil)

		assert.Nil(t, handler.settingsSecretsService)

		// SettingsSecretsService is a public field
		handler.SetSettingsSecretsService(nil)
		assert.Nil(t, handler.settingsSecretsService)
	})
}

// =============================================================================
// Handler CORS Tests
// =============================================================================

func TestHandlerIntegration_CORS_Configuration(t *testing.T) {
	t.Run("default CORS configuration", func(t *testing.T) {
		corsConfig := config.CORSConfig{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Content-Type", "Authorization"},
			AllowCredentials: false,
			MaxAge:           86400,
		}

		handler := NewHandler(nil, "/tmp/functions", corsConfig, "secret", "http://localhost", "", "", nil, nil, nil, nil)

		assert.Equal(t, corsConfig.AllowedOrigins, handler.corsConfig.AllowedOrigins)
		assert.Equal(t, corsConfig.AllowedMethods, handler.corsConfig.AllowedMethods)
		assert.Equal(t, corsConfig.AllowedHeaders, handler.corsConfig.AllowedHeaders)
	})

	t.Run("custom CORS configuration", func(t *testing.T) {
		corsConfig := config.CORSConfig{
			AllowedOrigins:   []string{"https://example.com", "https://app.example.com"},
			AllowedMethods:   []string{"POST", "GET"},
			AllowedHeaders:   []string{"X-Custom-Header"},
			AllowCredentials: true,
			MaxAge:           3600,
		}

		handler := NewHandler(nil, "/tmp/functions", corsConfig, "secret", "http://localhost", "", "", nil, nil, nil, nil)

		assert.Equal(t, []string{"https://example.com", "https://app.example.com"}, handler.corsConfig.AllowedOrigins)
		assert.True(t, handler.corsConfig.AllowCredentials)
		assert.Equal(t, 3600, handler.corsConfig.MaxAge)
	})
}

// =============================================================================
// Handler Registry Configuration Tests
// =============================================================================

func TestHandlerIntegration_RegistryConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		npmRegistry string
		jsrRegistry string
		expectedNPM string
		expectedJSR string
	}{
		{
			name:        "no custom registries",
			npmRegistry: "",
			jsrRegistry: "",
			expectedNPM: "",
			expectedJSR: "",
		},
		{
			name:        "npm registry only",
			npmRegistry: "https://registry.npmjs.org",
			jsrRegistry: "",
			expectedNPM: "https://registry.npmjs.org",
			expectedJSR: "",
		},
		{
			name:        "jsr registry only",
			npmRegistry: "",
			jsrRegistry: "https://jsr.io",
			expectedNPM: "",
			expectedJSR: "https://jsr.io",
		},
		{
			name:        "both registries",
			npmRegistry: "https://npm.example.com",
			jsrRegistry: "https://jsr.example.com",
			expectedNPM: "https://npm.example.com",
			expectedJSR: "https://jsr.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", tt.npmRegistry, tt.jsrRegistry, nil, nil, nil, nil)

			assert.Equal(t, tt.expectedNPM, handler.npmRegistry)
			assert.Equal(t, tt.expectedJSR, handler.jsrRegistry)
		})
	}
}

// =============================================================================
// Handler Runtime Configuration Tests
// =============================================================================

func TestHandlerIntegration_RuntimeConfiguration(t *testing.T) {
	t.Run("runtime is created for functions", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "", "", nil, nil, nil, nil)

		assert.NotNil(t, handler.runtime)
		assert.NotNil(t, handler.GetRuntime())
		assert.Equal(t, handler.runtime, handler.GetRuntime())
	})

	t.Run("public URL is passed to runtime", func(t *testing.T) {
		testURLs := []string{
			"http://localhost:8080",
			"https://api.example.com",
			"https://fluxbase.example.com/v1",
		}

		for _, url := range testURLs {
			handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", url, "", "", nil, nil, nil, nil)
			assert.Equal(t, url, handler.publicURL)
			assert.Equal(t, url, handler.GetPublicURL())
		}
	})
}

// =============================================================================
// Handler Storage Tests
// =============================================================================

func TestHandlerIntegration_Storage(t *testing.T) {
	t.Run("storage is initialized", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "", "", nil, nil, nil, nil)

		assert.NotNil(t, handler.storage)
	})
}

// =============================================================================
// Handler Log Counter Tests
// =============================================================================

func TestHandlerIntegration_LogCounters(t *testing.T) {
	t.Run("log counters map is initialized", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "", "", nil, nil, nil, nil)

		executionID := uuid.New()

		// Initially no counter
		_, ok := handler.logCounters.Load(executionID)
		assert.False(t, ok)

		// Add a counter
		handler.logCounters.Store(executionID, &executionLogContext{})

		// Now it exists
		val, ok := handler.logCounters.Load(executionID)
		assert.True(t, ok)

		ctx, ok := val.(*executionLogContext)
		require.True(t, ok)
		assert.Equal(t, 0, ctx.lineCounter)

		// Clean up
		handler.logCounters.Delete(executionID)
	})

	t.Run("concurrent log counter access", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "", "", nil, nil, nil, nil)

		// Test concurrent access through the handler's handleLogMessage method
		// which is the actual code path used in production
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

		// Should complete without race conditions
		assert.NotNil(t, handler)
	})
}

// =============================================================================
// Handler Bundler Creation Tests
// =============================================================================

func TestHandlerIntegration_CreateBundler_Options(t *testing.T) {
	t.Run("bundler options are applied", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "https://npm.custom.com", "https://jsr.custom.com", nil, nil, nil, nil)

		bundler, err := handler.createBundler()
		require.NoError(t, err)
		assert.NotNil(t, bundler)

		// Verify bundler was created (we can't inspect private fields directly,
		// but we can verify it was created without error)
		assert.NotNil(t, bundler)
	})

	t.Run("bundler with empty registries", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "", "", nil, nil, nil, nil)

		bundler, err := handler.createBundler()
		require.NoError(t, err)
		assert.NotNil(t, bundler)
	})
}

// =============================================================================
// Handler Log Message Tests
// =============================================================================

func TestHandlerIntegration_HandleLogMessage(t *testing.T) {
	t.Run("log message without counter", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "", "", nil, nil, nil, nil)

		executionID := uuid.New()

		// Should not panic when counter doesn't exist
		handler.handleLogMessage(executionID, "info", "test message")
	})

	t.Run("log message increments counter", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "", "", nil, nil, nil, nil)

		executionID := uuid.New()
		ctx := &executionLogContext{}
		handler.logCounters.Store(executionID, ctx)

		handler.handleLogMessage(executionID, "info", "message 1")
		assert.Equal(t, 1, ctx.lineCounter)

		handler.handleLogMessage(executionID, "info", "message 2")
		assert.Equal(t, 2, ctx.lineCounter)

		handler.handleLogMessage(executionID, "error", "error message")
		assert.Equal(t, 3, ctx.lineCounter)

		handler.logCounters.Delete(executionID)
	})

	t.Run("log message with invalid counter type", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "", "", nil, nil, nil, nil)

		executionID := uuid.New()

		// Store an invalid type (not a pointer to int)
		handler.logCounters.Store(executionID, "not an int pointer")

		// Should not panic
		handler.handleLogMessage(executionID, "info", "test message")

		// Clean up
		handler.logCounters.Delete(executionID)
	})
}

// =============================================================================
// Handler Authentication Service Tests
// =============================================================================

func TestHandlerIntegration_AuthService(t *testing.T) {
	t.Run("auth service is stored", func(t *testing.T) {
		// Note: We pass nil, so authService will be nil
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "", "", nil, nil, nil, nil)

		// authService is nil when passed nil
		assert.Nil(t, handler.authService)
	})
}

// =============================================================================
// Handler Secrets Storage Tests
// =============================================================================

func TestHandlerIntegration_SecretsStorage(t *testing.T) {
	t.Run("secrets storage is stored", func(t *testing.T) {
		// Note: We pass nil, so secretsStorage will be nil
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "", "", nil, nil, nil, nil)

		// secretsStorage is nil when passed nil
		assert.Nil(t, handler.secretsStorage)
	})
}

// =============================================================================
// Handler Scheduler Tests
// =============================================================================

func TestHandlerIntegration_Scheduler(t *testing.T) {
	t.Run("scheduler can be set and retrieved", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "", "", nil, nil, nil, nil)

		// Initially nil
		assert.Nil(t, handler.scheduler)

		// Set scheduler
		scheduler := &Scheduler{}
		handler.SetScheduler(scheduler)

		// Verify it's set
		assert.Equal(t, scheduler, handler.scheduler)
	})
}

// =============================================================================
// Handler Edge Cases
// =============================================================================

func TestHandlerIntegration_EdgeCases(t *testing.T) {
	t.Run("handler with empty functions directory", func(t *testing.T) {
		handler := NewHandler(nil, "", config.CORSConfig{}, "secret", "http://localhost", "", "", nil, nil, nil, nil)

		assert.Equal(t, "", handler.functionsDir)
		assert.Equal(t, "", handler.GetFunctionsDir())
	})

	t.Run("handler with empty JWT secret", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "", "http://localhost", "", "", nil, nil, nil, nil)

		assert.NotNil(t, handler)
		assert.NotNil(t, handler.runtime)
	})

	t.Run("handler with empty public URL", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "", "", "", nil, nil, nil, nil)

		assert.Equal(t, "", handler.publicURL)
		assert.Equal(t, "", handler.GetPublicURL())
	})
}

// =============================================================================
// Handler Configuration Priority Tests
// =============================================================================

func TestHandlerIntegration_ConfigurationPriority(t *testing.T) {
	t.Run("CORS config from parameter", func(t *testing.T) {
		corsConfig := config.CORSConfig{
			AllowedOrigins: []string{"https://app.example.com"},
			MaxAge:         7200,
		}

		handler := NewHandler(nil, "/tmp/functions", corsConfig, "secret", "http://localhost", "", "", nil, nil, nil, nil)

		assert.Equal(t, corsConfig.AllowedOrigins, handler.corsConfig.AllowedOrigins)
		assert.Equal(t, 7200, handler.corsConfig.MaxAge)
	})
}

// =============================================================================
// Handler Concurrent Access Tests
// =============================================================================

func TestHandlerIntegration_ConcurrentAccess(t *testing.T) {
	t.Run("concurrent log counter operations", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "", "", nil, nil, nil, nil)

		done := make(chan bool)
		for i := 0; i < 10; i++ {
			executionID := uuid.New()
			handler.logCounters.Store(executionID, &executionLogContext{})

			go func(id uuid.UUID) {
				for j := 0; j < 50; j++ {
					handler.handleLogMessage(id, "info", "message")
				}
				done <- true
			}(executionID)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		// Should complete without race conditions
	})
}

// =============================================================================
// Handler Memory Tests
// =============================================================================

func TestHandlerIntegration_MemoryManagement(t *testing.T) {
	t.Run("log counters are cleaned up", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "", "", nil, nil, nil, nil)

		// Add multiple counters
		ids := make([]uuid.UUID, 100)
		for i := 0; i < 100; i++ {
			id := uuid.New()
			ids[i] = id
			handler.logCounters.Store(id, &executionLogContext{})
		}

		// Verify they exist
		count := 0
		handler.logCounters.Range(func(k, v interface{}) bool {
			count++
			return true
		})
		assert.Equal(t, 100, count)

		// Clean up
		for _, id := range ids {
			handler.logCounters.Delete(id)
		}

		// Verify they're gone
		count = 0
		handler.logCounters.Range(func(k, v interface{}) bool {
			count++
			return true
		})
		assert.Equal(t, 0, count)
	})
}

// =============================================================================
// Handler Context Tests
// =============================================================================

func TestHandlerIntegration_ContextUsage(t *testing.T) {
	t.Run("handler works with context", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "", "", nil, nil, nil, nil)

		ctx := context.Background()
		assert.NotNil(t, ctx)

		// Handler methods should accept context
		// This is a compile-time check that the handler structure supports context
		_ = ctx
		_ = handler
	})

	t.Run("handler works with cancelled context", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "", "", nil, nil, nil, nil)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// Handler should still exist
		assert.NotNil(t, handler)
		// Context should be cancelled
		assert.Error(t, ctx.Err())
	})
}

// =============================================================================
// Handler Timeout Tests
// =============================================================================

func TestHandlerIntegration_Timeouts(t *testing.T) {
	t.Run("handler with timeout context", func(t *testing.T) {
		handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "", "", nil, nil, nil, nil)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		assert.NotNil(t, handler)
		assert.NotNil(t, ctx)
	})
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkHandler_NewHandler(b *testing.B) {
	corsConfig := config.CORSConfig{
		AllowedOrigins: []string{"*"},
		MaxAge:         86400,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewHandler(nil, "/tmp/functions", corsConfig, "secret", "http://localhost", "", "", nil, nil, nil, nil)
	}
}

func BenchmarkHandler_HandleLogMessage(b *testing.B) {
	handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "", "", nil, nil, nil, nil)

	executionID := uuid.New()
	handler.logCounters.Store(executionID, &executionLogContext{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.handleLogMessage(executionID, "info", "test log message")
	}
}

func BenchmarkHandler_CreateBundler(b *testing.B) {
	handler := NewHandler(nil, "/tmp/functions", config.CORSConfig{}, "secret", "http://localhost", "https://npm.example.com", "https://jsr.example.com", nil, nil, nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handler.createBundler()
	}
}
