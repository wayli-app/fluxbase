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
      - targets: ["localhost:8080"]
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

Endpoint: `/api/v1/monitoring/health`

```bash
curl http://localhost:8080/api/v1/monitoring/health
```

Returns `200 OK` if healthy, `503` if unhealthy. Checks database, realtime, and storage services.

**Docker Compose:**

```yaml
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost:8080/api/v1/monitoring/health"]
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
FLUXBASE_DEBUG=true   # Enable debug logging
FLUXBASE_DEBUG=false  # Production (info+)
```

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
