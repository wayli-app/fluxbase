# Fluxbase Monitoring Guide

This guide covers the monitoring and observability setup for Fluxbase in production.

## Overview

Fluxbase includes a comprehensive monitoring stack powered by:

- **Prometheus** - Metrics collection and storage
- **Grafana** - Metrics visualization and dashboards
- **PostgreSQL Exporter** - Database metrics
- **Redis Exporter** - Cache metrics
- **MinIO** - Built-in S3 storage metrics
- **Node Exporter** - System-level metrics
- **cAdvisor** - Container metrics

## Architecture

```
┌─────────────┐
│  Fluxbase   │──┐
│ Application │  │
└─────────────┘  │
                 │
┌─────────────┐  │     ┌────────────┐     ┌──────────┐
│ PostgreSQL  │──┼────>│ Prometheus │────>│ Grafana  │
└─────────────┘  │     └────────────┘     └──────────┘
                 │            │
┌─────────────┐  │            │
│   Redis     │──┤            │
└─────────────┘  │            v
                 │     ┌────────────┐
┌─────────────┐  │     │   Alerts   │
│   MinIO     │──┘     └────────────┘
└─────────────┘
```

## Quick Start

### Using Docker Compose

Start the full monitoring stack:

```bash
cd deploy
docker-compose -f docker-compose.production.yml up -d
```

Access the dashboards:

- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Fluxbase**: http://localhost (via NGINX)

### Accessing Metrics

Fluxbase exposes Prometheus metrics at:

```
http://localhost:8080/metrics
```

## Grafana Dashboards

### 1. Application Overview

**Dashboard**: `Fluxbase - Application Overview`

Key metrics:
- Request rate (req/s)
- p95 latency (ms)
- Success rate (%)
- Active database connections
- HTTP requests by endpoint
- HTTP status codes
- Memory and CPU usage

### 2. Database Metrics

**Dashboard**: `Fluxbase - Database Metrics`

Key metrics:
- Database operations (inserts, updates, deletes, fetches)
- Cache hit ratio
- Database locks
- Database size
- Connection states
- Long-running queries

## Key Metrics

### Application Metrics

| Metric | Description | Type |
|--------|-------------|------|
| `http_requests_total` | Total HTTP requests | Counter |
| `http_request_duration_seconds` | HTTP request latency | Histogram |
| `process_resident_memory_bytes` | Memory usage | Gauge |
| `process_cpu_seconds_total` | CPU usage | Counter |
| `go_goroutines` | Active goroutines | Gauge |

### Database Metrics

| Metric | Description | Type |
|--------|-------------|------|
| `pg_stat_database_numbackends` | Active connections | Gauge |
| `pg_stat_database_xact_commit` | Transaction commits | Counter |
| `pg_stat_database_xact_rollback` | Transaction rollbacks | Counter |
| `pg_stat_database_tup_inserted` | Rows inserted | Counter |
| `pg_stat_database_tup_updated` | Rows updated | Counter |
| `pg_stat_database_tup_deleted` | Rows deleted | Counter |
| `pg_database_size_bytes` | Database size | Gauge |

### Cache Metrics (Redis)

| Metric | Description | Type |
|--------|-------------|------|
| `redis_connected_clients` | Connected clients | Gauge |
| `redis_used_memory_bytes` | Memory usage | Gauge |
| `redis_commands_total` | Total commands | Counter |
| `redis_keyspace_hits_total` | Cache hits | Counter |
| `redis_keyspace_misses_total` | Cache misses | Counter |

### Storage Metrics (MinIO)

| Metric | Description | Type |
|--------|-------------|------|
| `minio_s3_requests_total` | S3 API requests | Counter |
| `minio_disk_storage_used_bytes` | Disk usage | Gauge |
| `minio_s3_requests_errors_total` | S3 errors | Counter |

## Alerting (Optional)

### Recommended Alerts

#### High Error Rate

```yaml
- alert: HighErrorRate
  expr: |
    sum(rate(http_requests_total{service="fluxbase",status=~"5.."}[5m]))
    /
    sum(rate(http_requests_total{service="fluxbase"}[5m]))
    > 0.05
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "High HTTP error rate"
    description: "Error rate is {{ $value | humanizePercentage }}"
```

#### High Latency

```yaml
- alert: HighLatency
  expr: |
    histogram_quantile(0.95,
      sum(rate(http_request_duration_seconds_bucket{service="fluxbase"}[5m])) by (le)
    ) > 1
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "High request latency"
    description: "p95 latency is {{ $value }}s"
```

#### Database Connection Pool Exhausted

```yaml
- alert: DatabaseConnectionPoolExhausted
  expr: |
    pg_stat_database_numbackends{datname="fluxbase"}
    /
    pg_settings_max_connections
    > 0.9
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "Database connection pool nearly exhausted"
    description: "Using {{ $value | humanizePercentage }} of max connections"
```

#### Low Cache Hit Ratio

```yaml
- alert: LowCacheHitRatio
  expr: |
    pg_stat_database_blks_hit{datname="fluxbase"}
    /
    (pg_stat_database_blks_hit{datname="fluxbase"} + pg_stat_database_blks_read{datname="fluxbase"})
    < 0.9
  for: 15m
  labels:
    severity: warning
  annotations:
    summary: "Low database cache hit ratio"
    description: "Cache hit ratio is {{ $value | humanizePercentage }}"
```

## Custom Metrics

### Adding Application Metrics

Fluxbase uses the Prometheus Go client. To add custom metrics:

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    myCounter = promauto.NewCounter(prometheus.CounterOpts{
        Name: "fluxbase_custom_events_total",
        Help: "Total number of custom events",
    })

    myHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
        Name: "fluxbase_custom_duration_seconds",
        Help: "Duration of custom operations",
        Buckets: prometheus.DefBuckets,
    })
)

// Usage
myCounter.Inc()
myHistogram.Observe(duration.Seconds())
```

## Performance Tuning

### Prometheus

**Retention**: Default is 30 days. Adjust in [prometheus.yml](monitoring/prometheus.yml):

```yaml
storage:
  tsdb:
    retention:
      time: 30d
      size: 10GB
```

**Scrape Interval**: Default is 15s. Adjust per job:

```yaml
scrape_configs:
  - job_name: 'fluxbase'
    scrape_interval: 15s
```

### Grafana

**Data Source Query Timeout**: Adjust in [datasources/prometheus.yml](monitoring/grafana/provisioning/datasources/prometheus.yml):

```yaml
jsonData:
  queryTimeout: '60s'
```

## Troubleshooting

### Metrics Not Appearing

1. Check if Prometheus can reach the target:
   ```bash
   curl http://localhost:8080/metrics
   ```

2. Check Prometheus targets:
   - Navigate to http://localhost:9090/targets
   - Ensure all targets are "UP"

3. Check Prometheus logs:
   ```bash
   docker-compose -f docker-compose.production.yml logs prometheus
   ```

### Dashboard Not Loading

1. Verify Grafana is running:
   ```bash
   docker-compose -f docker-compose.production.yml ps grafana
   ```

2. Check Grafana logs:
   ```bash
   docker-compose -f docker-compose.production.yml logs grafana
   ```

3. Verify datasource configuration:
   - Navigate to Configuration > Data Sources
   - Test the Prometheus datasource

### High Cardinality Metrics

If Prometheus is using too much memory:

1. Reduce metric cardinality by limiting labels
2. Increase scrape interval
3. Reduce retention period
4. Use recording rules for expensive queries

## Security

### Authentication

**Grafana**: Change default credentials immediately:

```bash
# In docker-compose.production.yml
environment:
  - GF_SECURITY_ADMIN_USER=admin
  - GF_SECURITY_ADMIN_PASSWORD=<strong-password>
```

**Prometheus**: Add basic auth via NGINX or use authentication proxy.

### Network Security

- Expose only necessary ports (NGINX on 80/443)
- Use internal Docker network for service communication
- Enable TLS for external endpoints

## Backup

Prometheus and Grafana data is persisted in Docker volumes:

```bash
# Backup Prometheus data
docker run --rm -v fluxbase_prometheus_data:/data -v $(pwd):/backup alpine tar czf /backup/prometheus-backup.tar.gz /data

# Backup Grafana data
docker run --rm -v fluxbase_grafana_data:/data -v $(pwd):/backup alpine tar czf /backup/grafana-backup.tar.gz /data
```

## Integration with External Systems

### Sending Alerts to Slack

Configure Alertmanager with Slack webhook:

```yaml
receivers:
  - name: 'slack'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/YOUR/WEBHOOK/URL'
        channel: '#alerts'
        title: '{{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'
```

### Sending Metrics to External Prometheus

Use remote write:

```yaml
remote_write:
  - url: https://prometheus.example.com/api/v1/write
    basic_auth:
      username: user
      password: pass
```

## Best Practices

1. **Set appropriate scrape intervals**: Balance between granularity and resource usage
2. **Use recording rules**: Pre-compute expensive queries
3. **Monitor your monitoring**: Set up alerts for Prometheus and Grafana health
4. **Regular backups**: Back up Prometheus and Grafana data regularly
5. **Label hygiene**: Keep labels consistent and avoid high cardinality
6. **Dashboard organization**: Group related metrics, use consistent time ranges
7. **Alert fatigue**: Set appropriate thresholds to avoid false positives

## Resources

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [PostgreSQL Exporter](https://github.com/prometheus-community/postgres_exporter)
- [Redis Exporter](https://github.com/oliver006/redis_exporter)
- [MinIO Monitoring](https://min.io/docs/minio/linux/operations/monitoring.html)
