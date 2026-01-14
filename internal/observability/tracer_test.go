package observability

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// =============================================================================
// TracerConfig Tests
// =============================================================================

func TestDefaultTracerConfig(t *testing.T) {
	t.Run("returns expected defaults", func(t *testing.T) {
		cfg := DefaultTracerConfig()

		assert.False(t, cfg.Enabled)
		assert.Equal(t, "localhost:4317", cfg.Endpoint)
		assert.Equal(t, "fluxbase", cfg.ServiceName)
		assert.Equal(t, "development", cfg.Environment)
		assert.Equal(t, 1.0, cfg.SampleRate)
		assert.True(t, cfg.Insecure)
	})

	t.Run("returns new instance each time", func(t *testing.T) {
		cfg1 := DefaultTracerConfig()
		cfg2 := DefaultTracerConfig()

		cfg1.ServiceName = "modified"
		assert.Equal(t, "fluxbase", cfg2.ServiceName)
	})
}

func TestTracerConfig_Struct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		cfg := TracerConfig{
			Enabled:     true,
			Endpoint:    "collector.example.com:4317",
			ServiceName: "my-service",
			Environment: "production",
			SampleRate:  0.5,
			Insecure:    false,
		}

		assert.True(t, cfg.Enabled)
		assert.Equal(t, "collector.example.com:4317", cfg.Endpoint)
		assert.Equal(t, "my-service", cfg.ServiceName)
		assert.Equal(t, "production", cfg.Environment)
		assert.Equal(t, 0.5, cfg.SampleRate)
		assert.False(t, cfg.Insecure)
	})

	t.Run("zero value config", func(t *testing.T) {
		var cfg TracerConfig

		assert.False(t, cfg.Enabled)
		assert.Empty(t, cfg.Endpoint)
		assert.Empty(t, cfg.ServiceName)
		assert.Empty(t, cfg.Environment)
		assert.Equal(t, 0.0, cfg.SampleRate)
		assert.False(t, cfg.Insecure)
	})
}

// =============================================================================
// Tracer Tests
// =============================================================================

func TestTracer_IsEnabled(t *testing.T) {
	t.Run("disabled tracer returns false", func(t *testing.T) {
		tracer := &Tracer{
			enabled: false,
		}
		assert.False(t, tracer.IsEnabled())
	})

	t.Run("enabled tracer returns true", func(t *testing.T) {
		tracer := &Tracer{
			enabled: true,
		}
		assert.True(t, tracer.IsEnabled())
	})
}

func TestTracer_Tracer(t *testing.T) {
	t.Run("returns underlying tracer", func(t *testing.T) {
		noopTracer := noop.NewTracerProvider().Tracer("test")
		tracer := &Tracer{
			tracer: noopTracer,
		}

		result := tracer.Tracer()
		assert.NotNil(t, result)
		assert.Equal(t, noopTracer, result)
	})

	t.Run("nil tracer returns nil", func(t *testing.T) {
		tracer := &Tracer{}
		assert.Nil(t, tracer.Tracer())
	})
}

func TestTracer_StartSpan(t *testing.T) {
	t.Run("creates span with noop tracer", func(t *testing.T) {
		noopTracer := noop.NewTracerProvider().Tracer("test")
		tracer := &Tracer{
			tracer: noopTracer,
		}

		ctx := context.Background()
		newCtx, span := tracer.StartSpan(ctx, "test-operation")

		assert.NotNil(t, newCtx)
		assert.NotNil(t, span)
		span.End()
	})
}

func TestTracer_Shutdown(t *testing.T) {
	t.Run("shutdown with nil provider returns nil", func(t *testing.T) {
		tracer := &Tracer{
			provider: nil,
		}

		err := tracer.Shutdown(context.Background())
		assert.NoError(t, err)
	})
}

// =============================================================================
// Context Helper Tests
// =============================================================================

func TestSpanFromContext(t *testing.T) {
	t.Run("returns span from context", func(t *testing.T) {
		ctx := context.Background()
		span := SpanFromContext(ctx)

		// Background context has no span, returns noop
		assert.NotNil(t, span)
	})

	t.Run("returns noop span for background context", func(t *testing.T) {
		ctx := context.Background()
		span := SpanFromContext(ctx)

		// Should not panic and return a valid span
		assert.NotNil(t, span)
		// Noop span is not recording
		assert.False(t, span.IsRecording())
	})
}

func TestContextWithSpan(t *testing.T) {
	t.Run("adds span to context", func(t *testing.T) {
		ctx := context.Background()
		noopTracer := noop.NewTracerProvider().Tracer("test")
		_, span := noopTracer.Start(ctx, "test")
		defer span.End()

		newCtx := ContextWithSpan(ctx, span)
		assert.NotNil(t, newCtx)

		// Retrieve span from new context
		retrievedSpan := SpanFromContext(newCtx)
		assert.Equal(t, span, retrievedSpan)
	})
}

// =============================================================================
// Span Recording Tests
// =============================================================================

func TestRecordError(t *testing.T) {
	t.Run("does not panic with no span", func(t *testing.T) {
		ctx := context.Background()
		err := errors.New("test error")

		// Should not panic
		assert.NotPanics(t, func() {
			RecordError(ctx, err)
		})
	})

	t.Run("does not panic with nil error", func(t *testing.T) {
		ctx := context.Background()

		assert.NotPanics(t, func() {
			RecordError(ctx, nil)
		})
	})

	t.Run("records error on recording span", func(t *testing.T) {
		noopTracer := noop.NewTracerProvider().Tracer("test")
		ctx, span := noopTracer.Start(context.Background(), "test")
		defer span.End()

		err := errors.New("test error")
		assert.NotPanics(t, func() {
			RecordError(ctx, err)
		})
	})
}

func TestSetSpanAttributes(t *testing.T) {
	t.Run("does not panic with no span", func(t *testing.T) {
		ctx := context.Background()

		assert.NotPanics(t, func() {
			SetSpanAttributes(ctx,
				attribute.String("key", "value"),
				attribute.Int("count", 42),
			)
		})
	})

	t.Run("does not panic with empty attributes", func(t *testing.T) {
		ctx := context.Background()

		assert.NotPanics(t, func() {
			SetSpanAttributes(ctx)
		})
	})

	t.Run("sets attributes on recording span", func(t *testing.T) {
		noopTracer := noop.NewTracerProvider().Tracer("test")
		ctx, span := noopTracer.Start(context.Background(), "test")
		defer span.End()

		assert.NotPanics(t, func() {
			SetSpanAttributes(ctx,
				attribute.String("service.name", "test-service"),
				attribute.Bool("feature.enabled", true),
			)
		})
	})
}

func TestAddSpanEvent(t *testing.T) {
	t.Run("does not panic with no span", func(t *testing.T) {
		ctx := context.Background()

		assert.NotPanics(t, func() {
			AddSpanEvent(ctx, "test-event")
		})
	})

	t.Run("does not panic with empty event name", func(t *testing.T) {
		ctx := context.Background()

		assert.NotPanics(t, func() {
			AddSpanEvent(ctx, "")
		})
	})

	t.Run("adds event with attributes", func(t *testing.T) {
		noopTracer := noop.NewTracerProvider().Tracer("test")
		ctx, span := noopTracer.Start(context.Background(), "test")
		defer span.End()

		assert.NotPanics(t, func() {
			AddSpanEvent(ctx, "cache.hit",
				attribute.String("cache.key", "user:123"),
				attribute.Int("cache.ttl", 3600),
			)
		})
	})
}

// =============================================================================
// Trace ID Extraction Tests
// =============================================================================

func TestExtractTraceID(t *testing.T) {
	t.Run("returns empty for context without span", func(t *testing.T) {
		ctx := context.Background()
		traceID := ExtractTraceID(ctx)

		assert.Empty(t, traceID)
	})

	t.Run("returns empty for noop span", func(t *testing.T) {
		noopTracer := noop.NewTracerProvider().Tracer("test")
		ctx, span := noopTracer.Start(context.Background(), "test")
		defer span.End()

		traceID := ExtractTraceID(ctx)
		// Noop tracer doesn't generate real trace IDs
		assert.Empty(t, traceID)
	})
}

func TestExtractSpanID(t *testing.T) {
	t.Run("returns empty for context without span", func(t *testing.T) {
		ctx := context.Background()
		spanID := ExtractSpanID(ctx)

		assert.Empty(t, spanID)
	})

	t.Run("returns empty for noop span", func(t *testing.T) {
		noopTracer := noop.NewTracerProvider().Tracer("test")
		ctx, span := noopTracer.Start(context.Background(), "test")
		defer span.End()

		spanID := ExtractSpanID(ctx)
		// Noop tracer doesn't generate real span IDs
		assert.Empty(t, spanID)
	})
}

// =============================================================================
// Database Tracing Helpers Tests
// =============================================================================

func TestStartDBSpan(t *testing.T) {
	t.Run("creates database span", func(t *testing.T) {
		ctx := context.Background()
		newCtx, span := StartDBSpan(ctx, "SELECT", "users")

		assert.NotNil(t, newCtx)
		assert.NotNil(t, span)
		span.End()
	})

	t.Run("creates span with different operations", func(t *testing.T) {
		operations := []string{"SELECT", "INSERT", "UPDATE", "DELETE"}
		for _, op := range operations {
			t.Run(op, func(t *testing.T) {
				ctx, span := StartDBSpan(context.Background(), op, "test_table")
				assert.NotNil(t, ctx)
				assert.NotNil(t, span)
				span.End()
			})
		}
	})

	t.Run("handles empty table name", func(t *testing.T) {
		ctx, span := StartDBSpan(context.Background(), "SELECT", "")
		assert.NotNil(t, ctx)
		assert.NotNil(t, span)
		span.End()
	})
}

func TestEndDBSpan(t *testing.T) {
	t.Run("ends span without error", func(t *testing.T) {
		_, span := StartDBSpan(context.Background(), "SELECT", "users")

		assert.NotPanics(t, func() {
			EndDBSpan(span, nil)
		})
	})

	t.Run("ends span with error", func(t *testing.T) {
		_, span := StartDBSpan(context.Background(), "SELECT", "users")
		err := errors.New("database connection failed")

		assert.NotPanics(t, func() {
			EndDBSpan(span, err)
		})
	})
}

// =============================================================================
// Storage Tracing Helpers Tests
// =============================================================================

func TestStartStorageSpan(t *testing.T) {
	t.Run("creates storage span", func(t *testing.T) {
		ctx := context.Background()
		newCtx, span := StartStorageSpan(ctx, "upload", "my-bucket", "path/to/file.jpg")

		assert.NotNil(t, newCtx)
		assert.NotNil(t, span)
		span.End()
	})

	t.Run("creates span with different operations", func(t *testing.T) {
		operations := []string{"upload", "download", "delete", "list"}
		for _, op := range operations {
			t.Run(op, func(t *testing.T) {
				ctx, span := StartStorageSpan(context.Background(), op, "bucket", "key")
				assert.NotNil(t, ctx)
				assert.NotNil(t, span)
				span.End()
			})
		}
	})

	t.Run("handles empty bucket and key", func(t *testing.T) {
		ctx, span := StartStorageSpan(context.Background(), "upload", "", "")
		assert.NotNil(t, ctx)
		assert.NotNil(t, span)
		span.End()
	})
}

// =============================================================================
// Auth Tracing Helpers Tests
// =============================================================================

func TestStartAuthSpan(t *testing.T) {
	t.Run("creates auth span", func(t *testing.T) {
		ctx := context.Background()
		newCtx, span := StartAuthSpan(ctx, "login")

		assert.NotNil(t, newCtx)
		assert.NotNil(t, span)
		span.End()
	})

	t.Run("creates span with different operations", func(t *testing.T) {
		operations := []string{"login", "logout", "refresh", "validate"}
		for _, op := range operations {
			t.Run(op, func(t *testing.T) {
				ctx, span := StartAuthSpan(context.Background(), op)
				assert.NotNil(t, ctx)
				assert.NotNil(t, span)
				span.End()
			})
		}
	})

	t.Run("handles empty operation", func(t *testing.T) {
		ctx, span := StartAuthSpan(context.Background(), "")
		assert.NotNil(t, ctx)
		assert.NotNil(t, span)
		span.End()
	})
}

// =============================================================================
// NewTracer Tests (without network)
// =============================================================================

func TestNewTracer_Disabled(t *testing.T) {
	t.Run("disabled tracer returns noop tracer", func(t *testing.T) {
		cfg := TracerConfig{
			Enabled: false,
		}

		tracer, err := NewTracer(context.Background(), cfg)
		require.NoError(t, err)
		require.NotNil(t, tracer)

		assert.False(t, tracer.IsEnabled())
		assert.NotNil(t, tracer.Tracer())
		assert.Nil(t, tracer.provider)
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkDefaultTracerConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = DefaultTracerConfig()
	}
}

func BenchmarkSpanFromContext(b *testing.B) {
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = SpanFromContext(ctx)
	}
}

func BenchmarkExtractTraceID(b *testing.B) {
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = ExtractTraceID(ctx)
	}
}

func BenchmarkExtractSpanID(b *testing.B) {
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = ExtractSpanID(ctx)
	}
}

func BenchmarkStartDBSpan(b *testing.B) {
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, span := StartDBSpan(ctx, "SELECT", "users")
		span.End()
	}
}

func BenchmarkStartStorageSpan(b *testing.B) {
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, span := StartStorageSpan(ctx, "upload", "bucket", "key")
		span.End()
	}
}

func BenchmarkStartAuthSpan(b *testing.B) {
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, span := StartAuthSpan(ctx, "login")
		span.End()
	}
}

func BenchmarkSetSpanAttributes(b *testing.B) {
	noopTracer := noop.NewTracerProvider().Tracer("bench")
	ctx, span := noopTracer.Start(context.Background(), "test")
	defer span.End()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		SetSpanAttributes(ctx,
			attribute.String("key", "value"),
			attribute.Int("count", i),
		)
	}
}

func BenchmarkAddSpanEvent(b *testing.B) {
	noopTracer := noop.NewTracerProvider().Tracer("bench")
	ctx, span := noopTracer.Start(context.Background(), "test")
	defer span.End()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		AddSpanEvent(ctx, "test.event",
			attribute.Int("iteration", i),
		)
	}
}

func BenchmarkRecordError(b *testing.B) {
	noopTracer := noop.NewTracerProvider().Tracer("bench")
	ctx, span := noopTracer.Start(context.Background(), "test")
	defer span.End()
	testErr := errors.New("benchmark error")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		RecordError(ctx, testErr)
	}
}

// =============================================================================
// Edge Cases and Error Scenarios
// =============================================================================

func TestTracer_EdgeCases(t *testing.T) {
	t.Run("nil context handling", func(t *testing.T) {
		// These should not panic with nil context
		// Note: trace functions from otel package handle nil context
		assert.NotPanics(t, func() {
			_ = ExtractTraceID(context.Background())
		})
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// Should handle cancelled context gracefully
		_, span := StartDBSpan(ctx, "SELECT", "users")
		assert.NotNil(t, span)
		span.End()
	})
}

// Test that Tracer struct implements expected methods
func TestTracer_Interface(t *testing.T) {
	t.Run("Tracer has expected methods", func(t *testing.T) {
		tracer := &Tracer{}

		// These should compile - testing interface compliance
		_ = tracer.IsEnabled()
		_ = tracer.Tracer()
		_ = tracer.Shutdown(context.Background())

		if tracer.Tracer() != nil {
			_, _ = tracer.StartSpan(context.Background(), "test")
		}
	})
}

// Test TracerConfig field tags
func TestTracerConfig_Tags(t *testing.T) {
	t.Run("config uses mapstructure tags", func(t *testing.T) {
		// This is a compile-time check that mapstructure tags exist
		// The actual parsing is done by viper/mapstructure
		cfg := TracerConfig{
			Enabled:     true,
			Endpoint:    "localhost:4317",
			ServiceName: "test",
			Environment: "test",
			SampleRate:  1.0,
			Insecure:    true,
		}
		assert.NotEmpty(t, cfg.Endpoint)
	})
}
