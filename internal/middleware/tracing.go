package middleware

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// TracingConfig holds configuration for the tracing middleware
type TracingConfig struct {
	// Enabled controls whether tracing is active
	Enabled bool

	// ServiceName is the name of the service for spans
	ServiceName string

	// SkipPaths are paths that should not be traced (e.g., /health, /metrics)
	SkipPaths []string

	// RecordRequestBody if true, records the request body as a span attribute (be careful with sensitive data)
	RecordRequestBody bool

	// RecordResponseBody if true, records the response body as a span attribute
	RecordResponseBody bool
}

// DefaultTracingConfig returns sensible defaults
func DefaultTracingConfig() TracingConfig {
	return TracingConfig{
		Enabled:            true,
		ServiceName:        "fluxbase",
		SkipPaths:          []string{"/health", "/ready", "/metrics"},
		RecordRequestBody:  false,
		RecordResponseBody: false,
	}
}

// TracingMiddleware returns a Fiber middleware that creates spans for HTTP requests
func TracingMiddleware(cfg TracingConfig) fiber.Handler {
	if !cfg.Enabled {
		return func(c *fiber.Ctx) error {
			return c.Next()
		}
	}

	tracer := otel.Tracer("fluxbase-http")

	// Build skip paths map for O(1) lookup
	skipPaths := make(map[string]bool)
	for _, path := range cfg.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *fiber.Ctx) error {
		path := c.Path()

		// Skip tracing for certain paths
		if skipPaths[path] {
			return c.Next()
		}

		// Extract parent context from incoming request headers
		ctx := otel.GetTextMapPropagator().Extract(
			c.Context(),
			propagation.HeaderCarrier(c.GetReqHeaders()),
		)

		// Determine span name - use route pattern if available, otherwise path
		spanName := c.Route().Path
		if spanName == "" {
			spanName = path
		}
		spanName = fmt.Sprintf("%s %s", c.Method(), spanName)

		// Start span
		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPMethod(c.Method()),
				semconv.HTTPURL(c.OriginalURL()),
				semconv.HTTPRoute(c.Route().Path),
				semconv.HTTPScheme(c.Protocol()),
				semconv.NetHostName(c.Hostname()),
				attribute.String("http.user_agent", c.Get("User-Agent")),
				attribute.String("http.request_id", c.Get("X-Request-ID")),
				attribute.String("net.peer.ip", c.IP()),
			),
		)
		defer span.End()

		// Store trace context in Fiber locals for downstream use
		c.Locals("trace_ctx", ctx)
		c.Locals("trace_span", span)

		// Add trace ID to response headers for debugging
		if span.SpanContext().HasTraceID() {
			c.Set("X-Trace-ID", span.SpanContext().TraceID().String())
		}

		// Record request body if configured (be careful with sensitive data)
		if cfg.RecordRequestBody && len(c.Body()) > 0 && len(c.Body()) < 4096 {
			span.SetAttributes(attribute.String("http.request.body", string(c.Body())))
		}

		// Process the request
		err := c.Next()

		// Record response attributes
		statusCode := c.Response().StatusCode()
		span.SetAttributes(
			semconv.HTTPStatusCode(statusCode),
			attribute.Int("http.response_size", len(c.Response().Body())),
		)

		// Record response body if configured
		if cfg.RecordResponseBody && len(c.Response().Body()) > 0 && len(c.Response().Body()) < 4096 {
			span.SetAttributes(attribute.String("http.response.body", string(c.Response().Body())))
		}

		// Set span status based on HTTP status code
		if statusCode >= 400 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", statusCode))
		} else {
			span.SetStatus(codes.Ok, "")
		}

		// Record any error that occurred
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}

		// Add user context if available
		if userID := c.Locals("user_id"); userID != nil {
			span.SetAttributes(attribute.String("user.id", fmt.Sprintf("%v", userID)))
		}
		if userRole := c.Locals("user_role"); userRole != nil {
			span.SetAttributes(attribute.String("user.role", fmt.Sprintf("%v", userRole)))
		}

		return err
	}
}

// GetTraceContext returns the trace context from Fiber context
func GetTraceContext(c *fiber.Ctx) trace.SpanContext {
	if span, ok := c.Locals("trace_span").(trace.Span); ok {
		return span.SpanContext()
	}
	return trace.SpanContext{}
}

// GetTraceID returns the trace ID from the Fiber context
func GetTraceID(c *fiber.Ctx) string {
	ctx := GetTraceContext(c)
	if ctx.HasTraceID() {
		return ctx.TraceID().String()
	}
	return ""
}

// GetSpanID returns the span ID from the Fiber context
func GetSpanID(c *fiber.Ctx) string {
	ctx := GetTraceContext(c)
	if ctx.HasSpanID() {
		return ctx.SpanID().String()
	}
	return ""
}

// StartChildSpan starts a child span from the Fiber context
func StartChildSpan(c *fiber.Ctx, name string, opts ...trace.SpanStartOption) (trace.Span, func()) {
	tracer := otel.Tracer("fluxbase-http")

	// Get parent context from Fiber locals
	var parentCtx interface{}
	if ctx := c.Locals("trace_ctx"); ctx != nil {
		parentCtx = ctx
	}

	// If we have a parent context, use it
	if pctx, ok := parentCtx.(interface{ Done() <-chan struct{} }); ok {
		_, span := tracer.Start(pctx.(interface {
			Done() <-chan struct{}
			Err() error
			Value(key interface{}) interface{}
			Deadline() (interface{}, bool)
		}).(interface {
			Done() <-chan struct{}
			Err() error
			Value(key interface{}) interface{}
		}), name, opts...)
		return span, span.End
	}

	// Otherwise create a new span
	_, span := tracer.Start(c.Context(), name, opts...)
	return span, span.End
}

// AddSpanEvent adds an event to the current span
func AddSpanEvent(c *fiber.Ctx, name string, attrs ...attribute.KeyValue) {
	if span, ok := c.Locals("trace_span").(trace.Span); ok && span.IsRecording() {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

// SetSpanError records an error on the current span
func SetSpanError(c *fiber.Ctx, err error) {
	if span, ok := c.Locals("trace_span").(trace.Span); ok && span.IsRecording() {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// SetSpanAttributes sets attributes on the current span
func SetSpanAttributes(c *fiber.Ctx, attrs ...attribute.KeyValue) {
	if span, ok := c.Locals("trace_span").(trace.Span); ok && span.IsRecording() {
		span.SetAttributes(attrs...)
	}
}
