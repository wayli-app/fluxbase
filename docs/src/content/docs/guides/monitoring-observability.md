---
title: Monitoring & Observability
description: Monitor Fluxbase with Prometheus metrics, health checks, and structured logging. Integrate with Grafana dashboards for production observability.
---

Fluxbase provides comprehensive monitoring and observability features to help you track system health, performance, and troubleshoot issues in production.

## Overview

Fluxbase exposes metrics, health checks, and system statistics through multiple endpoints:

- **Prometheus Metrics** (`/metrics`) - Standard Prometheus format metrics
- **System Metrics** (`/api/v1/monitoring/metrics`) - JSON system statistics
- **Health Checks** (`/api/v1/monitoring/health`) - Component health status
- **Logs** - Structured JSON logging with zerolog

---

## Monitoring Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Monitoring Stack                         │
│                                                              │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │  Fluxbase    │───▶│  Prometheus  │───▶│   Grafana    │  │
│  │   /metrics   │    │   (Scraper)  │    │ (Dashboard)  │  │
│  └──────────────┘    └──────────────┘    └──────────────┘  │
│                                                              │
│  ┌──────────────┐    ┌──────────────┐                       │
│  │  Structured  │───▶│  Loki/ELK    │                       │
│  │     Logs     │    │ (Log Agg.)   │                       │
│  └──────────────┘    └──────────────┘                       │
│                                                              │
│  ┌──────────────┐    ┌──────────────┐                       │
│  │    Health    │───▶│  Uptime Mon. │                       │
│  │    Checks    │    │  (AlertMgr)  │                       │
│  └──────────────┘    └──────────────┘                       │
└─────────────────────────────────────────────────────────────┘
```

---

## Prometheus Metrics

Fluxbase exposes Prometheus-compatible metrics at `/metrics` endpoint.

### Available Metrics

| Category | Metric | Type | Labels | Description |
|----------|--------|------|--------|-------------|
| **HTTP** | `fluxbase_http_requests_total` | Counter | `method`, `path`, `status` | Total HTTP requests |
| | `fluxbase_http_request_duration_seconds` | Histogram | `method`, `path`, `status` | HTTP request latency |
| | `fluxbase_http_requests_in_flight` | Gauge | - | Active requests |
| **Database** | `fluxbase_db_queries_total` | Counter | `operation`, `table` | Total database queries |
| | `fluxbase_db_query_duration_seconds` | Histogram | `operation`, `table` | Database query latency |
| | `fluxbase_db_connections` | Gauge | - | Current connections |
| | `fluxbase_db_connections_idle` | Gauge | - | Idle connections |
| | `fluxbase_db_connections_max` | Gauge | - | Maximum connections |
| **Realtime** | `fluxbase_realtime_connections` | Gauge | - | WebSocket connections |
| | `fluxbase_realtime_channels` | Gauge | - | Active channels |
| | `fluxbase_realtime_subscriptions` | Gauge | - | Total subscriptions |
| | `fluxbase_realtime_messages_total` | Counter | `channel_type` | Messages sent |
| **Storage** | `fluxbase_storage_bytes_total` | Counter | `operation`, `bucket` | Bytes stored/retrieved |
| | `fluxbase_storage_operations_total` | Counter | `operation`, `bucket`, `status` | Storage operations |
| | `fluxbase_storage_operation_duration_seconds` | Histogram | `operation`, `bucket` | Storage latency |
| **Auth** | `fluxbase_auth_attempts_total` | Counter | `method`, `result` | Auth attempts |
| | `fluxbase_auth_success_total` | Counter | `method` | Successful auths |
| | `fluxbase_auth_failure_total` | Counter | `method`, `reason` | Failed auths |
| **Rate Limiting** | `fluxbase_rate_limit_hits_total` | Counter | `limiter_type`, `identifier` | Rate limit hits |
| **System** | `fluxbase_system_uptime_seconds` | Gauge | - | System uptime |

---

## Configuring Prometheus

**1. Create `prometheus.yml`:**

```yaml
scrape_configs:
  - job_name: "fluxbase"
    static_configs:
      - targets: ["localhost:9090"]
    metrics_path: "/metrics"
```

**2. Run Prometheus:**

```bash
docker run -d -p 9090:9090 -v $(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml prom/prometheus
```

**3. Verify:** Visit http://localhost:9090 and query `fluxbase_http_requests_total`

---

## System Metrics Endpoint

JSON metrics endpoint at `/api/v1/monitoring/metrics` returns system statistics:

```bash
curl http://localhost:8080/api/v1/monitoring/metrics -H "Authorization: Bearer TOKEN"
```

**Response includes:**

| Category | Metrics |
|----------|---------|
| **System** | uptime_seconds, go_version, num_goroutines |
| **Memory** | memory_alloc_mb, memory_sys_mb, num_gc, gc_pause_ms |
| **Database** | acquired_conns, idle_conns, max_conns, acquire_duration_ms |
| **Realtime** | total_connections, active_channels, total_subscriptions |
| **Storage** | total_buckets, total_files, total_size_gb |

---

## Health Checks

Endpoint: `/api/v1/monitoring/health` — requires authentication and the `monitoring:read` scope.

```bash
curl -H "Authorization: Bearer <service-key>" \
     http://localhost:8080/api/v1/monitoring/health
```

Returns `200 OK` if healthy, `503` if unhealthy. Checks database, realtime, and storage services.

**Docker Compose healthcheck:** use the public, unauthenticated `/health` endpoint (the `/monitoring/health` endpoint requires auth and will fail a bare healthcheck):

```yaml
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
  interval: 30s
  timeout: 5s
  retries: 3
```

---

## Setting Up Grafana

**1. Run Grafana:**

```bash
docker run -d -p 3000:3000 grafana/grafana
```

**2. Add Prometheus data source:**

- Open http://localhost:3000 (login: `admin` / `admin`)
- Configuration → Data Sources → Add Prometheus
- URL: `http://prometheus:9090`

**3. Key dashboard queries:**

| Panel | Query |
|-------|-------|
| Request Rate | `rate(fluxbase_http_requests_total[5m])` |
| P95 Latency | `histogram_quantile(0.95, rate(fluxbase_http_request_duration_seconds_bucket[5m]))` |
| Error Rate | `rate(fluxbase_http_requests_total{status=~"5xx"}[5m])` |
| DB Connections | `fluxbase_db_connections` |
| Realtime Connections | `fluxbase_realtime_connections` |

---

## Alerting

Key alert rules for Prometheus:

| Alert | Condition | Description |
|-------|-----------|-------------|
| HighErrorRate | `rate(fluxbase_http_requests_total{status="5xx"}[5m]) > 0.05` | 5xx error rate > 5% |
| HighLatency | `histogram_quantile(0.95, rate(fluxbase_http_request_duration_seconds_bucket[5m])) > 1` | P95 latency > 1s |
| ConnectionPoolExhausted | `fluxbase_db_connections >= fluxbase_db_connections_max * 0.9` | Connection pool > 90% |
| HighAuthFailures | `rate(fluxbase_auth_failure_total[5m]) > 10` | Auth failures > 10/sec |
| FluxbaseDown | `up{job="fluxbase"} == 0` | Instance unreachable |

---

## Logging

Fluxbase uses structured JSON logging (zerolog):

```json
{
  "level": "info",
  "time": "2024-01-15T10:30:00Z",
  "message": "HTTP request",
  "method": "POST",
  "path": "/api/v1/tables/users",
  "status": 200,
  "duration_ms": 25.5
}
```

**Log levels:** debug, info, warn, error, fatal

**Configuration:**

```bash
FLUXBASE_DEBUG=true   # Enable debug mode (verbose output, not the same as log level)
FLUXBASE_LOGGING_CONSOLE_LEVEL=debug  # Set log level to debug
FLUXBASE_LOGGING_CONSOLE_LEVEL=info   # Production default
```

`FLUXBASE_DEBUG` enables verbose debug mode in the application. For controlling the log level independently, use `FLUXBASE_LOGGING_CONSOLE_LEVEL` (trace, debug, info, warn, error).

**Logged events:** HTTP requests, auth events, database queries, realtime connections, storage operations, webhooks, rate limits, security events

---

## Performance Monitoring

Key metrics and targets:

| Metric | Target | Query |
|--------|--------|-------|
| Request Latency | P95 < 200ms, P99 < 500ms | `histogram_quantile(0.95, rate(fluxbase_http_request_duration_seconds_bucket[5m]))` |
| DB Query Latency | P95 < 50ms, P99 < 100ms | `histogram_quantile(0.95, rate(fluxbase_db_query_duration_seconds_bucket[5m]))` |
| Error Rate | < 0.1% | `rate(fluxbase_http_requests_total{status="5xx"}[5m]) / rate(fluxbase_http_requests_total[5m])` |
| Connection Pool | < 80% | `fluxbase_db_connections / fluxbase_db_connections_max` |
| Memory | Stable | Monitor `memory_alloc_mb` over time |
| Goroutines | Stable | Monitor `num_goroutines` over time |

---

## Distributed Tracing

Fluxbase supports OpenTelemetry distributed tracing for end-to-end request visibility across services.

### Overview

Distributed tracing tracks requests as they travel through:
- HTTP API handlers
- Database queries
- Edge functions
- Background jobs
- External API calls

### Configuration

Enable tracing in your `fluxbase.yaml`:

```yaml
tracing:
  enabled: true
  endpoint: "localhost:4317"        # OTLP collector endpoint
  service_name: "fluxbase"
  environment: "production"
  sample_rate: 0.1                   # Sample 10% of traces
  insecure: false                    # Use TLS for production
```

### Setting Up Trace Backends

#### Jaeger (Local Development)

```bash
docker run -d --name jaeger \
  -e COLLECTOR_OTLP_ENABLED=true \
  -p 4317:4317 \
  -p 16686:16686 \
  jaegertracing/all-in-one:latest
```

Configure Fluxbase:
```yaml
tracing:
  enabled: true
  endpoint: "localhost:4317"
  insecure: true
```

Access Jaeger UI: http://localhost:16686

#### Grafana Tempo (Production)

```yaml
tracing:
  enabled: true
  endpoint: "tempo:4317"  # OTLP endpoint
```

#### Cloud Providers

| Provider | Setup |
|----------|-------|
| **AWS X-Ray** | Use AWS Distro for OpenTelemetry (ADOT) Collector |
| **Google Cloud Trace** | Use OpenTelemetry Collector with GCP exporter |
| **Azure Monitor** | Use Azure Monitor Application Insights exporter |

### Trace Analysis

**Identify Slow Queries:**

1. Open Jaeger UI or Grafana Tempo
2. Filter by operation `db.query` or `db.SELECT`
3. Sort by duration
4. Click slow spans to see SQL query

**Trace Errors:**

1. Find traces with error status
2. Expand trace timeline
3. Look for red error spans
4. View stack traces

**Performance Optimization:**

1. Look for spans with high duration
2. Check if spans run sequentially (could parallelize)
3. Identify N+1 query patterns
4. Find slow external API calls

### Automatic Spans

Fluxbase automatically creates spans for:

| Component | Span Name | Attributes |
|-----------|-----------|------------|
| **Database** | `db.SELECT`, `db.INSERT` | `db.system`, `db.table` |
| **Storage** | `storage.upload`, `storage.download` | `storage.bucket`, `storage.key` |
| **Auth** | `auth.login`, `auth.signup` | `auth.operation` |
| **Functions** | `function.myFunction` | `function.name`, `user.id` |
| **Jobs** | `job.sendEmail` | `job.name`, `job.id` |

### Custom Spans

Create custom spans for additional visibility:

```go
import "github.com/nimbleflux/fluxbase/internal/observability"

// Start a custom span
ctx, span := observability.StartSpan(ctx, "my-operation")
defer span.End()

// Add attributes
observability.SetSpanAttributes(ctx,
    attribute.String("user.id", userID),
)

// Add events
observability.AddSpanEvent(ctx, "validation.completed")

// Record errors
if err != nil {
    observability.RecordError(ctx, err)
}
```

### Production Considerations

**Sampling:**

```yaml
# Development: Sample all traces
sample_rate: 1.0  # 100%

# Production: Sample subset to reduce overhead
sample_rate: 0.1  # 10%
```

**Performance Impact:**

| Configuration | Overhead |
|--------------|----------|
| Sampling: 100% | ~5-10% |
| Sampling: 10% | ~1-2% |
| Sampling: 1% | <1% |

For more details, see [Distributed Tracing Guide](/guides/distributed-tracing/).

---

## Troubleshooting

| Issue | Symptoms | Diagnosis | Solutions |
|-------|----------|-----------|-----------|
| **High Latency** | Slow API responses | Check slow endpoints, DB query latency | Add indexes, optimize queries, increase connection pool |
| **High Error Rate** | 5xx errors | Monitor `rate(fluxbase_http_requests_total{status="5xx"}[5m])` | Check logs, verify DB connectivity, review deployments |
| **Memory Leaks** | Increasing memory | Monitor `memory_alloc_mb` and goroutine growth | Review long-running ops, check unclosed connections, update version |
| **Connection Pool Exhaustion** | Slow queries, timeouts | Check `fluxbase_db_connections >= fluxbase_db_connections_max` | Increase max_connections, reduce query time, add replicas |

---

## Best Practices

| Practice | Description |
|----------|-------------|
| **Set up monitoring early** | Configure Prometheus scraping, health checks, log aggregation, and alerting rules before production |
| **Monitor key metrics** | Focus on request latency (P95, P99), error rate, database performance, connection pool usage, auth failures |
| **Set up alerts** | Create alerts for high error rate (> 1%), high latency (P95 > 500ms), service unavailable, connection pool exhaustion |
| **Regular review** | Review dashboards daily, analyze trends weekly, optimize based on metrics, update alert thresholds |
| **Document runbooks** | Create runbooks: High latency → check indexes; 5xx errors → check logs; Memory leaks → restart & investigate |
