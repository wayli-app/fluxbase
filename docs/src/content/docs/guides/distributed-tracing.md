---
title: "Distributed Tracing"
description: Learn how to configure and use OpenTelemetry distributed tracing in Fluxbase. Monitor performance across services, databases, and edge functions.
---

Distributed tracing provides end-to-end visibility into requests as they travel through your Fluxbase instance. Fluxbase uses OpenTelemetry for standardized, vendor-agnostic tracing.

## What is Distributed Tracing?

Distributed tracing tracks a request as it moves through different services and components. Each unit of work is called a "span", and spans are combined into a "trace" that shows the complete journey of a request.

**Key Benefits:**

- **Performance Analysis**: Identify slow database queries, API calls, and function executions
- **Error Debugging**: Trace errors across service boundaries
- **Architecture Understanding**: Visualize service dependencies and data flow
- **Capacity Planning**: Make data-driven decisions about scaling

## Configuration

Enable OpenTelemetry tracing in your `fluxbase.yaml`:

```yaml
tracing:
  enabled: true
  endpoint: "localhost:4317"        # OTLP collector endpoint
  service_name: "fluxbase"           # Service name for traces
  environment: "production"           # Environment (development, staging, production)
  sample_rate: 1.0                   # Sample rate (0.0-1.0, 1.0 = 100%)
  insecure: false                    # Use TLS for production
```

**Environment Variables:**

```bash
export FLUXBASE_TRACING_ENABLED=true
export FLUXBASE_TRACING_ENDPOINT="collector.example.com:4317"
export FLUXBASE_TRACING_SERVICE_NAME="fluxbase"
export FLUXBASE_TRACING_ENVIRONMENT="production"
export FLUXBASE_TRACING_SAMPLE_RATE=0.1  # Sample 10% of traces
```

## Setting Up Trace Backends

Fluxbase uses the OTLP (OpenTelemetry Protocol) format, which is compatible with many backends:

### 1. Jaeger

Jaeger is a popular open-source tracing backend.

**Run Jaeger with Docker:**

```bash
docker run -d --name jaeger \
  -e COLLECTOR_OTLP_ENABLED=true \
  -p 4317:4317 \
  -p 16686:16686 \
  jaegertracing/all-in-one:latest
```

**Configure Fluxbase:**

```yaml
tracing:
  enabled: true
  endpoint: "localhost:4317"
  insecure: true  # For local development
```

**Access Jaeger UI:**
- Navigate to `http://localhost:16686`
- Browse traces by service, operation, and tags

### 2. Grafana Tempo

Grafana Tempo is a scalable, high-performance distributed tracing backend.

**Run Tempo with Docker:**

```bash
docker run -d --name tempo \
  -p 4317:4317 \
  -p 3200:3200 \
  grafana/tempo:latest \
  -server.http-listen-port=3200 \
  -storage.trace.backend=local \
  -storage.trace.local.path=/tmp/tempo
```

**Configure Fluxbase:**

```yaml
tracing:
  enabled: true
  endpoint: "localhost:4317"
```

**Access Tempo UI:**
- Use Grafana with Tempo data source
- Navigate to Grafana → Explore → Select Tempo data source

### 3. OpenTelemetry Collector

For production deployments, use the OpenTelemetry Collector as a central processing pipeline:

```yaml
# otel-collector-config.yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317

processors:
  batch:
    timeout: 5s
    max_batch_size: 1000

exporters:
  jaeger:
    endpoint: jaeger:14250
    tls:
      insecure: true

  logging:
    loglevel: debug

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [jaeger, logging]
```

**Run Collector:**

```bash
docker run -d --name otel-collector \
  -v $(pwd)/otel-collector-config.yaml:/etc/otelcol/config.yaml \
  -p 4317:4317 \
  otel/opentelemetry-collector:latest
```

### 4. Cloud Providers

**AWS X-Ray:**

```yaml
# Use AWS Distro for OpenTelemetry (ADOT) Collector
tracing:
  enabled: true
  endpoint: "localhost:4317"  # ADOT Collector endpoint
```

**Google Cloud Trace:**

```yaml
# Use OpenTelemetry Collector with Google Cloud Trace exporter
tracing:
  enabled: true
  endpoint: "localhost:4317"
```

**Azure Monitor:**

```yaml
# Use Azure Monitor Application Insights exporter
tracing:
  enabled: true
  endpoint: "localhost:4317"
```

## Automatic Instrumentation

Fluxbase automatically creates spans for:

### Database Operations

All PostgreSQL queries are automatically traced:

```go
// Automatic span created for this query
ctx, span := observability.StartDBSpan(ctx, "SELECT", "users")
defer observability.EndDBSpan(span, err)

rows, err := db.Query(ctx, "SELECT * FROM users WHERE id = $1", userID)
```

**Span Attributes:**
- `db.system`: "postgresql"
- `db.operation`: "SELECT", "INSERT", "UPDATE", "DELETE"
- `db.table`: Table name
- Error status if query fails

### Authentication Operations

All auth operations create spans:

- `auth.signup`
- `auth.signin`
- `auth.signout`
- `auth.oauth`
- `auth.magic_link`

### Storage Operations

File storage operations are traced:

- `storage.upload`
- `storage.download`
- `storage.delete`

## Custom Spans

Create custom spans for additional visibility:

### Basic Custom Span

```go
import "github.com/nimbleflux/fluxbase/internal/observability"

// Start a custom span
ctx, span := observability.StartSpan(ctx, "my-custom-operation")
defer span.End()

// Your code here
result, err := doSomething(ctx)

// Record error if failed
if err != nil {
    observability.RecordError(ctx, err)
}
```

### Span with Attributes

```go
import (
    "go.opentelemetry.io/otel/attribute"
    "github.com/nimbleflux/fluxbase/internal/observability"
)

ctx, span := observability.StartSpan(ctx, "process-data")
defer span.End()

// Add custom attributes
observability.SetSpanAttributes(ctx,
    attribute.String("user.id", userID),
    attribute.Int("record.count", len(records)),
    attribute.String("processing.type", "batch"),
)
```

### Span with Events

```go
// Add events to track progress
observability.AddSpanEvent(ctx, "validation.started",
    attribute.Int("record.count", len(records)),
)

// ... validation code ...

observability.AddSpanEvent(ctx, "validation.completed",
    attribute.Int("valid.records", validCount),
    attribute.Int("invalid.records", invalidCount),
)
```

## Edge Function Tracing

Fluxbase automatically traces edge function execution:

```typescript
// Your Deno function
import { tracer } from "https://deno.land/x/otel@v0.1.0/api.ts";

// Span context is automatically available via environment variables
const traceParent = Deno.env.get("TRACEPARENT");
const traceId = Deno.env.get("OTEL_TRACE_ID");
const spanId = Deno.env.get("OTEL_SPAN_ID");

// Fluxbase automatically creates function spans with attributes:
// - function.execution_id
// - function.name
// - function.namespace
// - user.id (if authenticated)
// - http.method
// - http.url
```

**Function Span Events:**

```typescript
// Add custom events to function spans
await fetch("https://api.example.com/data", {
  headers: {
    "traceparent": traceParent,  // Propagate trace context
  },
});
```

## Background Job Tracing

Jobs are automatically traced with progress tracking:

```go
// Job span is created when job starts
ctx, span := observability.StartJobSpan(ctx, observability.JobSpanConfig{
    JobID:       jobID,
    JobName:     "send-email",
    Namespace:   "notifications",
    Priority:    5,
    ScheduledAt: scheduledAt,
    WorkerID:    workerID,
    WorkerName:  "worker-1",
    UserID:      userID,
})
defer span.End()

// Track job progress
observability.SetJobProgress(ctx, 25, "Email queued")
// ... send email ...
observability.SetJobProgress(ctx, 50, "Email sent")
// ... update database ...
observability.SetJobProgress(ctx, 100, "Completed")

// Set final result
observability.SetJobResult(ctx, "completed", duration, nil)
```

## Trace Context Propagation

Trace context automatically propagates to:

1. **Database Queries**: All queries carry trace context
2. **HTTP Clients**: Use `traceparent` header
3. **Background Jobs**: Jobs inherit parent trace
4. **Edge Functions**: Trace context passed as environment variables

**Manual Propagation:**

```go
// Get trace context for subprocesses
env := observability.GetTraceContextEnv(ctx)

// Pass to subprocess
cmd := exec.CommandContext(ctx, "my-subprocess")
cmd.Env = append(os.Environ(), flattenEnv(env)...)
```

## Sampling Strategies

Reduce tracing overhead with smart sampling:

```yaml
# Sample all traces in development
tracing:
  sample_rate: 1.0  # 100% sampling

# Sample 10% of traces in production
tracing:
  sample_rate: 0.1  # 10% sampling

# Dynamic sampling based on route
tracing:
  sample_rate: 0.01  # 1% baseline
```

**Head-Based Sampling:**

```go
// Always trace slow operations
if duration > time.Second {
    observability.SetSpanAttributes(ctx,
        attribute.Bool("slow.request", true),
    )
}

// Always trace errors
if err != nil {
    observability.RecordError(ctx, err)
}
```

## Analyzing Traces

### Identify Slow Queries

Look for database spans with high duration:

1. Open Jaeger UI or Grafana Tempo
2. Filter by operation `db.query` or `db.SELECT`
3. Sort by duration
4. Click on slow spans to see SQL query

### Trace Errors Across Services

Follow an error through the system:

1. Find traces with error status
2. Expand the trace timeline
3. Look for red error spans
4. Click on error spans to see stack traces

### Performance Optimization

Identify optimization opportunities:

1. Look for spans with high duration
2. Check if spans run sequentially (could be parallelized)
3. Identify N+1 query patterns
4. Find slow external API calls

## Best Practices

### 1. Use Semantic Attributes

Follow OpenTelemetry semantic conventions:

```go
import semconv "go.opentelemetry.io/otel/semconv/v1.24.0"

observability.SetSpanAttributes(ctx,
    semconv.HTTPMethodKey.String("GET"),
    semconv.HTTPStatusCodeKey.Int(200),
    semconv.EnduserIDKey.String(userID),
)
```

### 2. Add Contextual Events

Track important events in spans:

```go
observability.AddSpanEvent(ctx, "cache.miss",
    attribute.String("cache.key", cacheKey),
)

observability.AddSpanEvent(ctx, "db.query.started")
// ... run query ...
observability.AddSpanEvent(ctx, "db.query.completed",
    attribute.Int("db.row_count", rowCount),
)
```

### 3. Set Appropriate Span Status

```go
if err != nil {
    observability.RecordError(ctx, err)
    span.SetStatus(codes.Error, err.Error())
} else {
    span.SetStatus(codes.Ok, "")
}
```

### 4. Use Span Links

Connect related spans:

```go
// Link to background job span
span.AddLink(trace.Link{
    SpanContext: jobSpan.SpanContext(),
    Attributes: []attribute.KeyValue{
        attribute.String("job.id", jobID),
    },
})
```

### 5. Configure Resource Attributes

Identify the service generating traces:

```yaml
tracing:
  service_name: "fluxbase"
  environment: "production"
```

Resource attributes added automatically:
- `service.name`: Service name
- `service.version`: Fluxbase version
- `deployment.environment`: Environment name
- `service.namespace`: "fluxbase"

## Troubleshooting

### No Traces Appearing

**Check 1: Verify tracing is enabled**

```bash
# Check logs for initialization message
grep "OpenTelemetry tracing initialized" /var/log/fluxbase/fluxbase.log
```

**Check 2: Verify endpoint connectivity**

```bash
# Test connection to collector
telnet localhost 4317
```

**Check 3: Check sample rate**

```yaml
# Ensure sample_rate > 0
tracing:
  sample_rate: 1.0  # Try 100% sampling for testing
```

**Check 4: Verify collector configuration**

```bash
# Check collector logs
docker logs otel-collector
```

### Spans Not Connecting

**Issue**: Spans appear but don't form a complete trace.

**Solution**: Ensure trace context propagation is working:

1. Check that requests include `traceparent` header
2. Verify context is passed through function calls
3. Check that spans use `defer span.End()`

### High Memory Usage

**Issue**: Tracing causes high memory usage.

**Solutions:**

1. **Reduce sample rate:**
   ```yaml
   tracing:
     sample_rate: 0.1  # Sample only 10%
   ```

2. **Use batch processing:**
   ```yaml
   # Collector configuration
   processors:
     batch:
       max_batch_size: 1000
       timeout: 10s
   ```

3. **Limit span attributes:**
   ```go
   // Avoid adding large attributes
   observability.SetSpanAttributes(ctx,
       attribute.String("huge.data", hugeDataString),  // Bad
       attribute.String("data.hash", hashData(hugeData)),  // Good
   )
   ```

## Performance Impact

Tracing overhead is minimal with proper configuration:

| Configuration | Overhead | Use Case |
|--------------|----------|----------|
| Sampling: 100% | ~5-10% | Development, critical paths |
| Sampling: 10% | ~1-2% | Production general |
| Sampling: 1% | <1% | High-traffic production |

**Optimization Tips:**

1. Use sampling in production
2. Disable tracing for health checks
3. Use batch exporters
4. Filter sensitive data from spans
5. Set appropriate span timeout

## Further Reading

- [OpenTelemetry Specification](https://opentelemetry.io/docs/specs/otel/)
- [OTLP Specification](https://opentelemetry.io/docs/specs/otlp/)
- [Jaeger Documentation](https://www.jaegertracing.io/docs/)
- [Grafana Tempo](https://grafana.com/docs/tempo/)
- [Observability Guide](/guides/monitoring-observability/)
