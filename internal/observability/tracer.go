package observability

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TracerConfig holds configuration for OpenTelemetry tracing
type TracerConfig struct {
	Enabled     bool    `mapstructure:"enabled"`
	Endpoint    string  `mapstructure:"endpoint"`     // OTLP endpoint (e.g., "localhost:4317")
	ServiceName string  `mapstructure:"service_name"` // Service name for traces
	Environment string  `mapstructure:"environment"`  // Environment (development, staging, production)
	SampleRate  float64 `mapstructure:"sample_rate"`  // Sample rate 0.0-1.0 (1.0 = 100%)
	Insecure    bool    `mapstructure:"insecure"`     // Use insecure connection (for local dev)
}

// DefaultTracerConfig returns sensible defaults for tracing
func DefaultTracerConfig() TracerConfig {
	return TracerConfig{
		Enabled:     false,
		Endpoint:    "localhost:4317",
		ServiceName: "fluxbase",
		Environment: "development",
		SampleRate:  1.0,
		Insecure:    true,
	}
}

// Tracer wraps OpenTelemetry tracer functionality
type Tracer struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
	enabled  bool
}

// NewTracer creates a new OpenTelemetry tracer
func NewTracer(ctx context.Context, cfg TracerConfig) (*Tracer, error) {
	if !cfg.Enabled {
		log.Info().Msg("OpenTelemetry tracing is disabled")
		return &Tracer{
			tracer:  otel.Tracer("fluxbase-noop"),
			enabled: false,
		}, nil
	}

	// Set defaults
	if cfg.ServiceName == "" {
		cfg.ServiceName = "fluxbase"
	}
	if cfg.Environment == "" {
		cfg.Environment = "development"
	}
	if cfg.SampleRate <= 0 {
		cfg.SampleRate = 1.0
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = "localhost:4317"
	}

	// Create OTLP exporter
	var opts []otlptracegrpc.Option
	opts = append(opts, otlptracegrpc.WithEndpoint(cfg.Endpoint))

	if cfg.Insecure {
		opts = append(opts, otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion("0.0.1-rc.46"),
			semconv.DeploymentEnvironment(cfg.Environment),
			attribute.String("service.namespace", "fluxbase"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create sampler based on configuration
	var sampler sdktrace.Sampler
	if cfg.SampleRate >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if cfg.SampleRate <= 0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.ParentBased(
			sdktrace.TraceIDRatioBased(cfg.SampleRate),
		)
	}

	// Create trace provider
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set global tracer provider
	otel.SetTracerProvider(provider)

	// Set global propagator for distributed tracing
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	log.Info().
		Str("endpoint", cfg.Endpoint).
		Str("service_name", cfg.ServiceName).
		Str("environment", cfg.Environment).
		Float64("sample_rate", cfg.SampleRate).
		Msg("OpenTelemetry tracing initialized")

	return &Tracer{
		provider: provider,
		tracer:   provider.Tracer("fluxbase"),
		enabled:  true,
	}, nil
}

// Shutdown gracefully shuts down the tracer
func (t *Tracer) Shutdown(ctx context.Context) error {
	if t.provider != nil {
		log.Info().Msg("Shutting down OpenTelemetry tracer")
		return t.provider.Shutdown(ctx)
	}
	return nil
}

// IsEnabled returns whether tracing is enabled
func (t *Tracer) IsEnabled() bool {
	return t.enabled
}

// Tracer returns the underlying OpenTelemetry tracer
func (t *Tracer) Tracer() trace.Tracer {
	return t.tracer
}

// StartSpan starts a new span with the given name
func (t *Tracer) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name, opts...)
}

// SpanFromContext returns the current span from context
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// ContextWithSpan returns a new context with the given span
func ContextWithSpan(ctx context.Context, span trace.Span) context.Context {
	return trace.ContextWithSpan(ctx, span)
}

// RecordError records an error on the current span
func RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// SetSpanAttributes sets attributes on the current span
func SetSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(attrs...)
	}
}

// AddSpanEvent adds an event to the current span
func AddSpanEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

// Database tracing helpers

// StartDBSpan starts a span for a database operation
func StartDBSpan(ctx context.Context, operation, table string) (context.Context, trace.Span) {
	tracer := otel.Tracer("fluxbase-db")
	return tracer.Start(ctx, fmt.Sprintf("db.%s", operation),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			semconv.DBSystemPostgreSQL,
			semconv.DBOperation(operation),
			attribute.String("db.table", table),
		),
	)
}

// EndDBSpan ends a database span and records any error
func EndDBSpan(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()
}

// Storage tracing helpers

// StartStorageSpan starts a span for a storage operation
func StartStorageSpan(ctx context.Context, operation, bucket, key string) (context.Context, trace.Span) {
	tracer := otel.Tracer("fluxbase-storage")
	return tracer.Start(ctx, fmt.Sprintf("storage.%s", operation),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("storage.operation", operation),
			attribute.String("storage.bucket", bucket),
			attribute.String("storage.key", key),
		),
	)
}

// Auth tracing helpers

// StartAuthSpan starts a span for an authentication operation
func StartAuthSpan(ctx context.Context, operation string) (context.Context, trace.Span) {
	tracer := otel.Tracer("fluxbase-auth")
	return tracer.Start(ctx, fmt.Sprintf("auth.%s", operation),
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("auth.operation", operation),
		),
	)
}

// HTTP tracing helpers

// ExtractTraceID extracts the trace ID from context as a string
func ExtractTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasTraceID() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// ExtractSpanID extracts the span ID from context as a string
func ExtractSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasSpanID() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// Function tracing helpers

// FunctionSpanConfig holds configuration for function span attributes
type FunctionSpanConfig struct {
	ExecutionID string
	Name        string
	Namespace   string
	UserID      string
	Method      string
	URL         string
}

// StartFunctionSpan starts a span for an edge function invocation
func StartFunctionSpan(ctx context.Context, cfg FunctionSpanConfig) (context.Context, trace.Span) {
	tracer := otel.Tracer("fluxbase-functions")

	spanName := fmt.Sprintf("function.%s", cfg.Name)
	if cfg.Namespace != "" {
		spanName = fmt.Sprintf("function.%s.%s", cfg.Namespace, cfg.Name)
	}

	attrs := []attribute.KeyValue{
		attribute.String("function.execution_id", cfg.ExecutionID),
		attribute.String("function.name", cfg.Name),
		attribute.String("function.namespace", cfg.Namespace),
		attribute.String("http.method", cfg.Method),
		attribute.String("http.url", cfg.URL),
	}

	if cfg.UserID != "" {
		attrs = append(attrs, attribute.String("user.id", cfg.UserID))
	}

	return tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(attrs...),
	)
}

// AddFunctionEvent adds an event to a function span
func AddFunctionEvent(ctx context.Context, eventName string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		defaultAttrs := []attribute.KeyValue{
			attribute.String("component", "function"),
		}
		span.AddEvent(eventName, trace.WithAttributes(append(defaultAttrs, attrs...)...))
	}
}

// SetFunctionResult sets the result attributes on a function span
func SetFunctionResult(ctx context.Context, statusCode int, duration time.Duration, err error) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(
			attribute.Int("http.status_code", statusCode),
			attribute.Int64("function.duration_ms", duration.Milliseconds()),
		)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else if statusCode >= 400 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", statusCode))
		} else {
			span.SetStatus(codes.Ok, "")
		}
	}
}

// Job tracing helpers

// JobSpanConfig holds configuration for job span attributes
type JobSpanConfig struct {
	JobID       string
	JobName     string
	Namespace   string
	Priority    int
	ScheduledAt string
	UserID      string
	WorkerID    string
	WorkerName  string
}

// StartJobSpan starts a span for a background job execution
func StartJobSpan(ctx context.Context, cfg JobSpanConfig) (context.Context, trace.Span) {
	tracer := otel.Tracer("fluxbase-jobs")

	spanName := fmt.Sprintf("job.%s", cfg.JobName)
	if cfg.Namespace != "" {
		spanName = fmt.Sprintf("job.%s.%s", cfg.Namespace, cfg.JobName)
	}

	attrs := []attribute.KeyValue{
		attribute.String("job.id", cfg.JobID),
		attribute.String("job.name", cfg.JobName),
		attribute.String("job.namespace", cfg.Namespace),
		attribute.Int("job.priority", cfg.Priority),
		attribute.String("job.scheduled_at", cfg.ScheduledAt),
		attribute.String("worker.id", cfg.WorkerID),
		attribute.String("worker.name", cfg.WorkerName),
	}

	if cfg.UserID != "" {
		attrs = append(attrs, attribute.String("user.id", cfg.UserID))
	}

	return tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(attrs...),
	)
}

// AddJobEvent adds an event to a job span
func AddJobEvent(ctx context.Context, eventName string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		defaultAttrs := []attribute.KeyValue{
			attribute.String("component", "job"),
		}
		span.AddEvent(eventName, trace.WithAttributes(append(defaultAttrs, attrs...)...))
	}
}

// SetJobProgress updates job progress on the span
func SetJobProgress(ctx context.Context, progress int, message string) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.AddEvent("job.progress", trace.WithAttributes(
			attribute.Int("job.progress_percent", progress),
			attribute.String("job.progress_message", message),
		))
	}
}

// SetJobResult sets the result attributes on a job span
func SetJobResult(ctx context.Context, status string, duration time.Duration, err error) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("job.status", status),
			attribute.Int64("job.duration_ms", duration.Milliseconds()),
		)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else if status == "failed" || status == "cancelled" {
			span.SetStatus(codes.Error, status)
		} else {
			span.SetStatus(codes.Ok, "")
		}
	}
}

// GetTraceContextEnv returns trace context as environment variables for propagation
// This is useful for propagating trace context to subprocesses like Deno
func GetTraceContextEnv(ctx context.Context) map[string]string {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return nil
	}

	env := make(map[string]string)
	sc := span.SpanContext()

	// W3C Trace Context format
	if sc.HasTraceID() {
		env["TRACEPARENT"] = fmt.Sprintf("00-%s-%s-%s",
			sc.TraceID().String(),
			sc.SpanID().String(),
			sc.TraceFlags().String(),
		)
	}

	// Also include individual fields for easier access
	if sc.HasTraceID() {
		env["OTEL_TRACE_ID"] = sc.TraceID().String()
	}
	if sc.HasSpanID() {
		env["OTEL_SPAN_ID"] = sc.SpanID().String()
	}

	return env
}
