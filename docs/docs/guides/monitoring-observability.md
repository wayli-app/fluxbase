---
title: Monitoring & Observability
sidebar_position: 10
---

# Monitoring & Observability

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

#### HTTP Metrics

| Metric                                   | Type      | Labels                     | Description                       |
| ---------------------------------------- | --------- | -------------------------- | --------------------------------- |
| `fluxbase_http_requests_total`           | Counter   | `method`, `path`, `status` | Total number of HTTP requests     |
| `fluxbase_http_request_duration_seconds` | Histogram | `method`, `path`, `status` | HTTP request latency              |
| `fluxbase_http_request_size_bytes`       | Histogram | `method`, `path`           | HTTP request size                 |
| `fluxbase_http_response_size_bytes`      | Histogram | `method`, `path`, `status` | HTTP response size                |
| `fluxbase_http_requests_in_flight`       | Gauge     | -                          | Current number of active requests |

**Histogram Buckets (Latency)**:
`.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10` seconds

#### Database Metrics

| Metric                               | Type      | Labels               | Description                     |
| ------------------------------------ | --------- | -------------------- | ------------------------------- |
| `fluxbase_db_queries_total`          | Counter   | `operation`, `table` | Total database queries executed |
| `fluxbase_db_query_duration_seconds` | Histogram | `operation`, `table` | Database query latency          |
| `fluxbase_db_connections`            | Gauge     | -                    | Current database connections    |
| `fluxbase_db_connections_idle`       | Gauge     | -                    | Idle database connections       |
| `fluxbase_db_connections_max`        | Gauge     | -                    | Maximum database connections    |

**Histogram Buckets (Query Duration)**:
`.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5` seconds

#### Realtime Metrics

| Metric                                      | Type    | Labels         | Description                   |
| ------------------------------------------- | ------- | -------------- | ----------------------------- |
| `fluxbase_realtime_connections`             | Gauge   | -              | Current WebSocket connections |
| `fluxbase_realtime_channels`                | Gauge   | -              | Active realtime channels      |
| `fluxbase_realtime_subscriptions`           | Gauge   | -              | Total realtime subscriptions  |
| `fluxbase_realtime_messages_total`          | Counter | `channel_type` | Messages sent via realtime    |
| `fluxbase_realtime_connection_errors_total` | Counter | `error_type`   | WebSocket connection errors   |

#### Storage Metrics

| Metric                                        | Type      | Labels                          | Description                  |
| --------------------------------------------- | --------- | ------------------------------- | ---------------------------- |
| `fluxbase_storage_bytes_total`                | Counter   | `operation`, `bucket`           | Total bytes stored/retrieved |
| `fluxbase_storage_operations_total`           | Counter   | `operation`, `bucket`, `status` | Storage operations count     |
| `fluxbase_storage_operation_duration_seconds` | Histogram | `operation`, `bucket`           | Storage operation latency    |

**Histogram Buckets (Storage)**:
`.01, .05, .1, .25, .5, 1, 2.5, 5, 10` seconds

#### Authentication Metrics

| Metric                              | Type    | Labels             | Description                |
| ----------------------------------- | ------- | ------------------ | -------------------------- |
| `fluxbase_auth_attempts_total`      | Counter | `method`, `result` | Authentication attempts    |
| `fluxbase_auth_success_total`       | Counter | `method`           | Successful authentications |
| `fluxbase_auth_failure_total`       | Counter | `method`, `reason` | Failed authentications     |
| `fluxbase_auth_tokens_issued_total` | Counter | `token_type`       | Auth tokens issued         |

#### Rate Limiting Metrics

| Metric                           | Type    | Labels                       | Description     |
| -------------------------------- | ------- | ---------------------------- | --------------- |
| `fluxbase_rate_limit_hits_total` | Counter | `limiter_type`, `identifier` | Rate limit hits |

#### System Metrics

| Metric                           | Type  | Labels | Description              |
| -------------------------------- | ----- | ------ | ------------------------ |
| `fluxbase_system_uptime_seconds` | Gauge | -      | System uptime in seconds |

---

## Configuring Prometheus

### Prometheus Configuration

Create or update your `prometheus.yml`:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: "fluxbase"
    static_configs:
      - targets: ["localhost:8080"]
    metrics_path: "/metrics"
    scrape_interval: 15s
```

### Run Prometheus

**Using Docker:**

```bash
docker run -d \
  --name prometheus \
  -p 9090:9090 \
  -v $(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml \
  prom/prometheus
```

**Using Docker Compose:**

```yaml
# docker-compose.yml
services:
  fluxbase:
    image: ghcr.io/wayli-app/fluxbase:latest:latest
    ports:
      - "8080:8080"
    environment:
      - FLUXBASE_DATABASE_HOST=postgres

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    command:
      - "--config.file=/etc/prometheus/prometheus.yml"
      - "--storage.tsdb.path=/prometheus"
```

**Verify Prometheus is scraping:**

```bash
# Check targets
curl http://localhost:9090/api/v1/targets

# Query metrics
curl 'http://localhost:9090/api/v1/query?query=fluxbase_http_requests_total'
```

---

## System Metrics Endpoint

Fluxbase provides a JSON metrics endpoint at `/api/v1/monitoring/metrics` that returns detailed system statistics.

### Request

```bash
curl -X GET http://localhost:8080/api/v1/monitoring/metrics \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Response

```json
{
  "uptime_seconds": 86400,
  "go_version": "go1.21.0",
  "num_goroutines": 42,

  "memory_alloc_mb": 25,
  "memory_total_alloc_mb": 150,
  "memory_sys_mb": 50,
  "num_gc": 12,
  "gc_pause_ms": 0.5,

  "database": {
    "acquire_count": 1250,
    "acquired_conns": 5,
    "canceled_acquire_count": 0,
    "constructing_conns": 0,
    "empty_acquire_count": 10,
    "idle_conns": 20,
    "max_conns": 25,
    "total_conns": 25,
    "new_conns_count": 25,
    "max_lifetime_destroy_count": 0,
    "max_idle_destroy_count": 5,
    "acquire_duration_ms": 2.5
  },

  "realtime": {
    "total_connections": 150,
    "active_channels": 25,
    "total_subscriptions": 200
  },

  "storage": {
    "total_buckets": 5,
    "total_files": 1250,
    "total_size_gb": 2.5
  }
}
```

### Metric Descriptions

#### System Metrics

- **uptime_seconds**: Time since server started
- **go_version**: Go runtime version
- **num_goroutines**: Active goroutines (concurrent operations)

#### Memory Metrics

- **memory_alloc_mb**: Current heap allocation in MB
- **memory_total_alloc_mb**: Cumulative bytes allocated
- **memory_sys_mb**: Total memory from OS
- **num_gc**: Number of garbage collections
- **gc_pause_ms**: Latest GC pause time

#### Database Metrics

- **acquire_count**: Total connection acquisitions
- **acquired_conns**: Currently acquired connections
- **idle_conns**: Available idle connections
- **max_conns**: Maximum allowed connections
- **total_conns**: Total connections in pool
- **acquire_duration_ms**: Average time to acquire connection

#### Realtime Metrics

- **total_connections**: Active WebSocket connections
- **active_channels**: Channels with subscriptions
- **total_subscriptions**: Total active subscriptions

#### Storage Metrics

- **total_buckets**: Number of storage buckets
- **total_files**: Total files stored
- **total_size_gb**: Total storage size

---

## Health Checks

Fluxbase provides comprehensive health checks at `/api/v1/monitoring/health`.

### Request

```bash
curl -X GET http://localhost:8080/api/v1/monitoring/health
```

### Response

```json
{
  "status": "healthy",
  "services": {
    "database": {
      "status": "healthy",
      "latency_ms": 5
    },
    "realtime": {
      "status": "healthy",
      "message": "WebSocket server running",
      "latency_ms": 0
    },
    "storage": {
      "status": "healthy",
      "latency_ms": 10
    }
  }
}
```

### Health Status Values

- **healthy**: Service is fully operational
- **degraded**: Service is operational but with issues
- **unhealthy**: Service is down or unresponsive

### HTTP Status Codes

- **200 OK**: System is healthy
- **503 Service Unavailable**: System is unhealthy

### Using Health Checks

**Kubernetes Liveness Probe:**

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: fluxbase
spec:
  containers:
    - name: fluxbase
      image: ghcr.io/wayli-app/fluxbase:latest:latest
      livenessProbe:
        httpGet:
          path: /api/v1/monitoring/health
          port: 8080
        initialDelaySeconds: 30
        periodSeconds: 10
        timeoutSeconds: 5
        failureThreshold: 3
```

**Docker Healthcheck:**

```dockerfile
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
  CMD curl -f http://localhost:8080/api/v1/monitoring/health || exit 1
```

**Docker Compose:**

```yaml
services:
  fluxbase:
    image: ghcr.io/wayli-app/fluxbase:latest:latest
    healthcheck:
      test:
        ["CMD", "curl", "-f", "http://localhost:8080/api/v1/monitoring/health"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 30s
```

---

## Setting Up Grafana

### Install Grafana

**Using Docker:**

```bash
docker run -d \
  --name grafana \
  -p 3000:3000 \
  grafana/grafana
```

**Using Docker Compose:**

```yaml
services:
  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana-storage:/var/lib/grafana

volumes:
  grafana-storage:
```

### Add Prometheus Data Source

1. Open Grafana at http://localhost:3000
2. Login with `admin` / `admin`
3. Navigate to **Configuration** → **Data Sources**
4. Click **Add data source**
5. Select **Prometheus**
6. Set URL to `http://prometheus:9090` (or `http://localhost:9090`)
7. Click **Save & Test**

### Import Fluxbase Dashboard

Create a dashboard with these panels:

#### HTTP Request Rate

```promql
rate(fluxbase_http_requests_total[5m])
```

#### HTTP Request Latency (p95)

```promql
histogram_quantile(0.95, rate(fluxbase_http_request_duration_seconds_bucket[5m]))
```

#### Database Query Latency (p99)

```promql
histogram_quantile(0.99, rate(fluxbase_db_query_duration_seconds_bucket[5m]))
```

#### Active Connections

```promql
fluxbase_http_requests_in_flight
fluxbase_db_connections
fluxbase_realtime_connections
```

#### Error Rate

```promql
rate(fluxbase_http_requests_total{status=~"5xx"}[5m])
```

#### Authentication Failures

```promql
rate(fluxbase_auth_failure_total[5m])
```

---

## Example Dashboards

### System Overview Dashboard

**Panels to include:**

1. **Uptime** - `fluxbase_system_uptime_seconds`
2. **Request Rate** - `rate(fluxbase_http_requests_total[5m])`
3. **Active Requests** - `fluxbase_http_requests_in_flight`
4. **Database Connections** - `fluxbase_db_connections`
5. **Realtime Connections** - `fluxbase_realtime_connections`
6. **Memory Usage** - Query via `/api/v1/monitoring/metrics`
7. **Error Rate** - `rate(fluxbase_http_requests_total{status="5xx"}[5m])`

### Database Performance Dashboard

**Panels to include:**

1. **Query Rate** - `rate(fluxbase_db_queries_total[5m])`
2. **Query Latency p50/p95/p99**
3. **Connection Pool Usage** - `fluxbase_db_connections / fluxbase_db_connections_max`
4. **Idle Connections** - `fluxbase_db_connections_idle`
5. **Slow Queries** - Queries > 1s threshold

### Realtime Performance Dashboard

**Panels to include:**

1. **Active Connections** - `fluxbase_realtime_connections`
2. **Active Channels** - `fluxbase_realtime_channels`
3. **Total Subscriptions** - `fluxbase_realtime_subscriptions`
4. **Message Rate** - `rate(fluxbase_realtime_messages_total[5m])`
5. **Connection Errors** - `rate(fluxbase_realtime_connection_errors_total[5m])`

---

## Alerting

### Prometheus Alertmanager

Create `alert_rules.yml`:

```yaml
groups:
  - name: fluxbase_alerts
    interval: 30s
    rules:
      # High error rate
      - alert: HighErrorRate
        expr: rate(fluxbase_http_requests_total{status="5xx"}[5m]) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value }} errors/sec"

      # High latency
      - alert: HighLatency
        expr: histogram_quantile(0.95, rate(fluxbase_http_request_duration_seconds_bucket[5m])) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High API latency detected"
          description: "P95 latency is {{ $value }} seconds"

      # Database connection pool exhaustion
      - alert: DatabaseConnectionPoolExhausted
        expr: fluxbase_db_connections >= fluxbase_db_connections_max * 0.9
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Database connection pool nearly exhausted"
          description: "{{ $value }} connections in use"

      # Authentication failures
      - alert: HighAuthenticationFailureRate
        expr: rate(fluxbase_auth_failure_total[5m]) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High authentication failure rate"
          description: "{{ $value }} auth failures/sec"

      # Instance down
      - alert: FluxbaseDown
        expr: up{job="fluxbase"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Fluxbase instance is down"
          description: "Instance {{ $labels.instance }} is not reachable"

      # High memory usage
      - alert: HighMemoryUsage
        expr: process_resident_memory_bytes / process_virtual_memory_max_bytes > 0.9
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High memory usage detected"
          description: "Memory usage is {{ $value | humanizePercentage }}"
```

### Configuring Alertmanager

```yaml
# alertmanager.yml
global:
  resolve_timeout: 5m

route:
  group_by: ["alertname", "severity"]
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 12h
  receiver: "email"

receivers:
  - name: "email"
    email_configs:
      - to: "alerts@example.com"
        from: "alertmanager@example.com"
        smarthost: smtp.gmail.com:587
        auth_username: "your-email@gmail.com"
        auth_password: "your-app-password"

  - name: "slack"
    slack_configs:
      - api_url: "YOUR_SLACK_WEBHOOK_URL"
        channel: "#alerts"
        title: "Fluxbase Alert"
        text: "{{ .CommonAnnotations.description }}"
```

---

## Logging Best Practices

### 1. Use Structured Logging

All Fluxbase logs use structured JSON format with zerolog:

```json
{
  "level": "info",
  "time": "2024-01-15T10:30:00Z",
  "message": "HTTP request",
  "method": "POST",
  "path": "/api/v1/tables/users",
  "status": 200,
  "duration_ms": 25.5,
  "ip": "192.168.1.100",
  "user_id": "uuid-here"
}
```

### 2. Log Levels

Fluxbase uses standard log levels:

- **debug**: Detailed diagnostic information
- **info**: General informational messages
- **warn**: Warning messages (degraded state)
- **error**: Error messages (recoverable errors)
- **fatal**: Fatal errors (application crash)

### 3. Configure Log Level

**Environment Variable:**

```bash
FLUXBASE_DEBUG=true  # Enables debug logging
```

**In Production:**

```bash
FLUXBASE_DEBUG=false  # Only info, warn, error, fatal
```

### 4. Important Log Events

Fluxbase automatically logs:

- HTTP requests (method, path, status, duration, IP)
- Authentication events (success/failure, method)
- Database queries (operation, table, duration)
- Realtime connections (connect, disconnect, errors)
- Storage operations (upload, download, delete)
- Webhook deliveries (success, failure, retry)
- Rate limit hits
- Security events (CSRF failures, RLS violations)

---

## Tracing (Distributed Tracing)

For distributed tracing, you can integrate OpenTelemetry or Jaeger.

### OpenTelemetry Integration

**Install OpenTelemetry SDK** (Go):

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/jaeger"
    "go.opentelemetry.io/otel/sdk/trace"
)

// Initialize tracer
func initTracer() (*trace.TracerProvider, error) {
    exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(
        jaeger.WithEndpoint("http://localhost:14268/api/traces"),
    ))
    if err != nil {
        return nil, err
    }

    tp := trace.NewTracerProvider(
        trace.WithBatcher(exporter),
        trace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String("fluxbase"),
        )),
    )

    otel.SetTracerProvider(tp)
    return tp, nil
}
```

**Run Jaeger:**

```bash
docker run -d \
  --name jaeger \
  -p 16686:16686 \
  -p 14268:14268 \
  jaegertracing/all-in-one:latest
```

Access Jaeger UI at http://localhost:16686

---

## Performance Monitoring

### Key Metrics to Monitor

#### 1. Request Latency

**Target**: P95 < 200ms, P99 < 500ms

```promql
histogram_quantile(0.95, rate(fluxbase_http_request_duration_seconds_bucket[5m]))
histogram_quantile(0.99, rate(fluxbase_http_request_duration_seconds_bucket[5m]))
```

#### 2. Database Query Performance

**Target**: P95 < 50ms, P99 < 100ms

```promql
histogram_quantile(0.95, rate(fluxbase_db_query_duration_seconds_bucket[5m]))
```

#### 3. Error Rate

**Target**: < 0.1% (1 error per 1000 requests)

```promql
rate(fluxbase_http_requests_total{status="5xx"}[5m]) / rate(fluxbase_http_requests_total[5m])
```

#### 4. Connection Pool Health

**Target**: < 80% utilization

```promql
fluxbase_db_connections / fluxbase_db_connections_max
```

#### 5. Memory Usage

**Target**: Stable, no memory leaks

Monitor: `memory_alloc_mb` over time

#### 6. Goroutines

**Target**: Stable count

Monitor: `num_goroutines` over time

---

## Troubleshooting

### High Latency

**Symptoms**: Slow API responses

**Diagnosis**:

```promql
# Check which endpoints are slow
topk(10, histogram_quantile(0.95, rate(fluxbase_http_request_duration_seconds_bucket[5m])))

# Check database query latency
histogram_quantile(0.95, rate(fluxbase_db_query_duration_seconds_bucket[5m]))
```

**Solutions**:

- Add database indexes
- Enable query caching
- Increase connection pool size
- Optimize slow queries

### High Error Rate

**Symptoms**: 5xx errors

**Diagnosis**:

```promql
rate(fluxbase_http_requests_total{status="5xx"}[5m])
```

**Solutions**:

- Check application logs
- Verify database connectivity
- Check storage availability
- Review recent deployments

### Memory Leaks

**Symptoms**: Increasing memory usage

**Diagnosis**:

- Monitor `memory_alloc_mb` over time
- Check goroutine count growth

**Solutions**:

- Review long-running operations
- Check for unclosed connections
- Update to latest version
- Enable pprof profiling

### Connection Pool Exhaustion

**Symptoms**: Slow queries, timeouts

**Diagnosis**:

```promql
fluxbase_db_connections >= fluxbase_db_connections_max
```

**Solutions**:

- Increase `max_connections` in config
- Reduce query execution time
- Check for connection leaks
- Add read replicas

---

## Best Practices

### 1. Set Up Monitoring Early

Configure monitoring before deploying to production:

- ✅ Prometheus scraping
- ✅ Health checks
- ✅ Log aggregation
- ✅ Alerting rules

### 2. Monitor Key Metrics

Focus on:

- Request latency (P95, P99)
- Error rate
- Database performance
- Connection pool usage
- Authentication failures

### 3. Set Up Alerts

Create alerts for:

- High error rate (> 1%)
- High latency (P95 > 500ms)
- Service unavailable
- Connection pool exhaustion
- Authentication failures

### 4. Regular Review

- Review dashboards daily
- Analyze trends weekly
- Optimize based on metrics
- Update alert thresholds

### 5. Document Runbooks

Create runbooks for common issues:

- High latency → Check database indexes
- 5xx errors → Check logs and database
- Memory leaks → Restart and investigate
- Connection issues → Scale up pool

---

## Summary

Fluxbase provides comprehensive monitoring and observability:

- ✅ **Prometheus metrics** for all components
- ✅ **Health checks** for service status
- ✅ **Structured logging** with JSON format
- ✅ **System statistics** via JSON API
- ✅ **Pre-built Grafana dashboards**
- ✅ **Alert rules** for critical issues

Set up monitoring early, create dashboards, configure alerts, and regularly review metrics to ensure optimal performance and reliability.
