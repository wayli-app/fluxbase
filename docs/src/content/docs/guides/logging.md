---
title: Logging
---

Fluxbase provides comprehensive structured logging using [zerolog](https://github.com/rs/zerolog), a fast and lightweight logging library that outputs JSON-formatted logs for easy parsing and analysis.

## Overview

All Fluxbase logs are structured JSON messages that include:

- **Timestamp**: ISO 8601 format with timezone
- **Level**: Log level (debug, info, warn, error, fatal)
- **Message**: Human-readable description
- **Context Fields**: Additional structured data (user_id, request_id, etc.)

---

## Log Levels

Fluxbase uses standard log levels from least to most severe:

| Level     | Description                        | Use Case              | Production  |
| --------- | ---------------------------------- | --------------------- | ----------- |
| **debug** | Detailed diagnostic information    | Development debugging | ❌ Disabled |
| **info**  | General informational messages     | Normal operations     | ✅ Enabled  |
| **warn**  | Warning messages, degraded state   | Non-critical issues   | ✅ Enabled  |
| **error** | Error messages, recoverable errors | Failed operations     | ✅ Enabled  |
| **fatal** | Fatal errors, application crash    | Critical failures     | ✅ Enabled  |

---

## Configuration

### Enable Debug Logging

**Environment Variable:**

```bash
# Enable debug logging (development)
FLUXBASE_DEBUG=true

# Disable debug logging (production - default)
FLUXBASE_DEBUG=false
```

**Docker:**

```bash
docker run -e FLUXBASE_DEBUG=true fluxbase/fluxbase:latest
```

**Docker Compose:**

```yaml
services:
  fluxbase:
    image: ghcr.io/wayli-app/fluxbase:latest:latest
    environment:
      - FLUXBASE_DEBUG=true
```

**Kubernetes:**

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: fluxbase-config
data:
  FLUXBASE_DEBUG: "false"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: fluxbase
spec:
  template:
    spec:
      containers:
        - name: fluxbase
          image: ghcr.io/wayli-app/fluxbase:latest:latest
          envFrom:
            - configMapRef:
                name: fluxbase-config
```

---

## Log Format

### JSON Structure

All logs are output as single-line JSON:

```json
{
  "level": "info",
  "time": "2024-01-15T10:30:00.123Z",
  "message": "HTTP request completed",
  "method": "POST",
  "path": "/api/v1/tables/users",
  "status": 200,
  "duration_ms": 25.5,
  "ip": "192.168.1.100",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "request_id": "req_abc123"
}
```

### Field Descriptions

| Field         | Type    | Description                                 |
| ------------- | ------- | ------------------------------------------- |
| `level`       | string  | Log level (debug, info, warn, error, fatal) |
| `time`        | string  | ISO 8601 timestamp with timezone            |
| `message`     | string  | Human-readable log message                  |
| `method`      | string  | HTTP method (GET, POST, etc.)               |
| `path`        | string  | Request path                                |
| `status`      | integer | HTTP status code                            |
| `duration_ms` | float   | Request duration in milliseconds            |
| `ip`          | string  | Client IP address                           |
| `user_id`     | string  | Authenticated user ID (if available)        |
| `request_id`  | string  | Unique request identifier                   |
| `error`       | string  | Error message (for error logs)              |

---

## Log Events

### HTTP Requests

Every HTTP request is logged with details:

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
  "user_agent": "Mozilla/5.0...",
  "request_id": "req_abc123"
}
```

### Authentication Events

**Successful Login:**

```json
{
  "level": "info",
  "time": "2024-01-15T10:30:00Z",
  "message": "User authenticated successfully",
  "method": "email",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "ip": "192.168.1.100"
}
```

**Failed Login:**

```json
{
  "level": "warn",
  "time": "2024-01-15T10:30:00Z",
  "message": "Authentication failed",
  "method": "email",
  "reason": "invalid_credentials",
  "email": "user@example.com",
  "ip": "192.168.1.100"
}
```

### Database Operations

**Query Execution:**

```json
{
  "level": "debug",
  "time": "2024-01-15T10:30:00Z",
  "message": "Database query executed",
  "operation": "SELECT",
  "table": "users",
  "duration_ms": 5.2,
  "rows_affected": 10
}
```

**Slow Query:**

```json
{
  "level": "warn",
  "time": "2024-01-15T10:30:00Z",
  "message": "Slow database query detected",
  "operation": "SELECT",
  "table": "posts",
  "duration_ms": 1250.5,
  "query": "SELECT * FROM posts WHERE..."
}
```

### Realtime Events

**WebSocket Connection:**

```json
{
  "level": "info",
  "time": "2024-01-15T10:30:00Z",
  "message": "WebSocket connection established",
  "connection_id": "conn_xyz789",
  "ip": "192.168.1.100",
  "user_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Channel Subscription:**

```json
{
  "level": "info",
  "time": "2024-01-15T10:30:00Z",
  "message": "Channel subscription created",
  "connection_id": "conn_xyz789",
  "channel": "public:posts",
  "user_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**WebSocket Error:**

```json
{
  "level": "error",
  "time": "2024-01-15T10:30:00Z",
  "message": "WebSocket error",
  "connection_id": "conn_xyz789",
  "error": "connection closed unexpectedly",
  "error_type": "connection_error"
}
```

### Storage Operations

**File Upload:**

```json
{
  "level": "info",
  "time": "2024-01-15T10:30:00Z",
  "message": "File uploaded",
  "bucket": "avatars",
  "file_path": "user-123/avatar.png",
  "size_bytes": 524288,
  "content_type": "image/png",
  "duration_ms": 125.5,
  "user_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**File Download:**

```json
{
  "level": "info",
  "time": "2024-01-15T10:30:00Z",
  "message": "File downloaded",
  "bucket": "avatars",
  "file_path": "user-123/avatar.png",
  "size_bytes": 524288,
  "duration_ms": 45.2
}
```

### Webhook Events

**Webhook Triggered:**

```json
{
  "level": "info",
  "time": "2024-01-15T10:30:00Z",
  "message": "Webhook triggered",
  "webhook_id": "webhook_123",
  "event": "insert",
  "table": "users",
  "url": "https://example.com/webhooks/users"
}
```

**Webhook Delivery Success:**

```json
{
  "level": "info",
  "time": "2024-01-15T10:30:00Z",
  "message": "Webhook delivered successfully",
  "webhook_id": "webhook_123",
  "delivery_id": "delivery_456",
  "url": "https://example.com/webhooks/users",
  "status": 200,
  "duration_ms": 250.5
}
```

**Webhook Delivery Failure:**

```json
{
  "level": "error",
  "time": "2024-01-15T10:30:00Z",
  "message": "Webhook delivery failed",
  "webhook_id": "webhook_123",
  "delivery_id": "delivery_456",
  "url": "https://example.com/webhooks/users",
  "status": 500,
  "error": "connection timeout",
  "retry_count": 2,
  "duration_ms": 5000
}
```

### Security Events

**CSRF Validation Failure:**

```json
{
  "level": "warn",
  "time": "2024-01-15T10:30:00Z",
  "message": "CSRF token validation failed",
  "ip": "192.168.1.100",
  "path": "/api/v1/tables/users",
  "method": "POST"
}
```

**Rate Limit Hit:**

```json
{
  "level": "warn",
  "time": "2024-01-15T10:30:00Z",
  "message": "Rate limit exceeded",
  "ip": "192.168.1.100",
  "path": "/api/v1/auth/login",
  "limit": 10,
  "window": "1m"
}
```

**RLS Policy Violation:**

```json
{
  "level": "warn",
  "time": "2024-01-15T10:30:00Z",
  "message": "Row Level Security policy violation",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "table": "private_data",
  "operation": "SELECT",
  "policy": "user_isolation"
}
```

### System Events

**Server Started:**

```json
{
  "level": "info",
  "time": "2024-01-15T10:00:00Z",
  "message": "Fluxbase server started",
  "version": "v1.0.0",
  "address": ":8080",
  "environment": "production"
}
```

**Database Connected:**

```json
{
  "level": "info",
  "time": "2024-01-15T10:00:01Z",
  "message": "Database connection established",
  "host": "postgres",
  "port": 5432,
  "database": "fluxbase",
  "max_connections": 25
}
```

**Graceful Shutdown:**

```json
{
  "level": "info",
  "time": "2024-01-15T18:00:00Z",
  "message": "Graceful shutdown initiated",
  "uptime_seconds": 28800
}
```

---

## Log Aggregation

### Sending Logs to External Services

#### 1. Docker Logs

**View Logs:**

```bash
# Follow logs
docker logs -f fluxbase

# Last 100 lines
docker logs --tail 100 fluxbase

# Since 1 hour ago
docker logs --since 1h fluxbase
```

**Filter Logs:**

```bash
# Only error logs
docker logs fluxbase 2>&1 | grep '"level":"error"'

# Only authentication events
docker logs fluxbase 2>&1 | grep '"message":"User authenticated"'
```

#### 2. Loki (Grafana Loki)

**Docker Compose Setup:**

```yaml
# docker-compose.yml
services:
  fluxbase:
    image: ghcr.io/wayli-app/fluxbase:latest:latest
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
    labels:
      logging: "promtail"

  promtail:
    image: grafana/promtail:latest
    volumes:
      - /var/lib/docker/containers:/var/lib/docker/containers:ro
      - ./promtail-config.yml:/etc/promtail/config.yml
    command: -config.file=/etc/promtail/config.yml

  loki:
    image: grafana/loki:latest
    ports:
      - "3100:3100"

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
```

**Promtail Configuration (`promtail-config.yml`):**

```yaml
server:
  http_listen_port: 9080
  grpc_listen_port: 0

positions:
  filename: /tmp/positions.yaml

clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  - job_name: fluxbase
    docker_sd_configs:
      - host: unix:///var/run/docker.sock
        refresh_interval: 5s
    relabel_configs:
      - source_labels: ["__meta_docker_container_name"]
        regex: "/(.*)"
        target_label: "container"
      - source_labels: ["__meta_docker_container_log_stream"]
        target_label: "stream"
```

#### 3. Elasticsearch (ELK Stack)

**Filebeat Configuration:**

```yaml
# filebeat.yml
filebeat.inputs:
  - type: container
    paths:
      - "/var/lib/docker/containers/*/*.log"
    json.keys_under_root: true
    json.add_error_key: true

processors:
  - add_docker_metadata: ~

output.elasticsearch:
  hosts: ["localhost:9200"]
  index: "fluxbase-%{+yyyy.MM.dd}"

setup.kibana:
  host: "localhost:5601"
```

#### 4. CloudWatch Logs (AWS)

**Docker Log Driver:**

```yaml
# docker-compose.yml
services:
  fluxbase:
    image: ghcr.io/wayli-app/fluxbase:latest:latest
    logging:
      driver: awslogs
      options:
        awslogs-region: us-east-1
        awslogs-group: /fluxbase/production
        awslogs-stream: fluxbase-app
```

#### 5. Google Cloud Logging

**Docker Log Driver:**

```yaml
# docker-compose.yml
services:
  fluxbase:
    image: ghcr.io/wayli-app/fluxbase:latest:latest
    logging:
      driver: gcplogs
      options:
        gcp-project: your-project-id
        gcp-log-cmd: true
```

---

## Querying Logs

### Using jq (Command Line)

**Install jq:**

```bash
# macOS
brew install jq

# Ubuntu/Debian
sudo apt-get install jq

# Alpine
apk add jq
```

**Query Examples:**

```bash
# Filter by log level
docker logs fluxbase 2>&1 | jq 'select(.level == "error")'

# Filter by message
docker logs fluxbase 2>&1 | jq 'select(.message | contains("authentication"))'

# Extract specific fields
docker logs fluxbase 2>&1 | jq '{time, level, message, user_id}'

# Count errors by type
docker logs fluxbase 2>&1 | jq -r 'select(.level == "error") | .error' | sort | uniq -c

# Find slow requests (> 1000ms)
docker logs fluxbase 2>&1 | jq 'select(.duration_ms > 1000)'

# Get unique IP addresses
docker logs fluxbase 2>&1 | jq -r '.ip' | sort | uniq

# Calculate average request duration
docker logs fluxbase 2>&1 | jq -s 'map(.duration_ms) | add / length'
```

### Using Grafana Loki (LogQL)

**Query Examples:**

```logql
# All logs from fluxbase
{container="fluxbase"}

# Only error logs
{container="fluxbase"} |= "error"

# Authentication failures
{container="fluxbase"} | json | level="warn" | message="Authentication failed"

# Requests taking > 1 second
{container="fluxbase"} | json | duration_ms > 1000

# Rate of errors
rate({container="fluxbase"} | json | level="error" [5m])

# Top 10 slowest requests
topk(10, sum by (path) (avg_over_time({container="fluxbase"} | json | unwrap duration_ms [5m])))
```

### Using Elasticsearch (Kibana)

**Query Examples:**

```json
// All error logs
{
  "query": {
    "match": {
      "level": "error"
    }
  }
}

// Authentication failures in last hour
{
  "query": {
    "bool": {
      "must": [
        { "match": { "message": "Authentication failed" }},
        { "range": { "time": { "gte": "now-1h" }}}
      ]
    }
  }
}

// Slow requests
{
  "query": {
    "range": {
      "duration_ms": {
        "gte": 1000
      }
    }
  }
}
```

---

## Log Retention

### Docker Log Rotation

Configure log rotation to prevent disk space issues:

```yaml
# docker-compose.yml
services:
  fluxbase:
    image: ghcr.io/wayli-app/fluxbase:latest:latest
    logging:
      driver: "json-file"
      options:
        max-size: "10m" # Max size per log file
        max-file: "3" # Keep 3 log files
        compress: "true" # Compress rotated logs
```

### Kubernetes Log Rotation

Kubernetes automatically rotates logs:

```yaml
# Pod configuration
apiVersion: v1
kind: Pod
metadata:
  name: fluxbase
spec:
  containers:
    - name: fluxbase
      image: ghcr.io/wayli-app/fluxbase:latest:latest
      # Logs are automatically rotated by kubelet
      # Default: 10MB per file, max 5 files
```

### External Log Storage

**Loki Retention:**

```yaml
# loki-config.yml
table_manager:
  retention_deletes_enabled: true
  retention_period: 720h # 30 days
```

**Elasticsearch Retention:**

```json
// Index Lifecycle Policy
{
  "policy": {
    "phases": {
      "hot": {
        "actions": {
          "rollover": {
            "max_size": "50GB",
            "max_age": "7d"
          }
        }
      },
      "delete": {
        "min_age": "30d",
        "actions": {
          "delete": {}
        }
      }
    }
  }
}
```

---

## Best Practices

### 1. Production Configuration

**Disable Debug Logging:**

```bash
FLUXBASE_DEBUG=false
```

**Configure Log Rotation:**

```yaml
logging:
  driver: "json-file"
  options:
    max-size: "10m"
    max-file: "3"
```

**Send to External Service:**

Use Loki, Elasticsearch, or CloudWatch for long-term storage and analysis.

### 2. Monitoring Critical Events

Set up alerts for:

```logql
# High error rate
rate({container="fluxbase"} | json | level="error" [5m]) > 1

# Authentication failures
rate({container="fluxbase"} | json | message="Authentication failed" [5m]) > 10

# Slow requests
rate({container="fluxbase"} | json | duration_ms > 1000 [5m]) > 5
```

### 3. Log Sampling

For high-traffic applications, consider sampling:

```go
// Sample 10% of debug logs
if level == "debug" && rand.Float64() > 0.1 {
    return
}
```

### 4. Redact Sensitive Data

Fluxbase automatically redacts:

- Passwords
- API keys
- JWT tokens
- Credit card numbers

Never log:

- ❌ User passwords (plaintext or hashed)
- ❌ API keys or secrets
- ❌ JWT tokens (except for debugging)
- ❌ Credit card information
- ❌ Social security numbers
- ❌ Personal health information

### 5. Use Structured Fields

Always use structured fields instead of string concatenation:

```go
// ✅ GOOD: Structured logging
log.Info().
    Str("user_id", userID).
    Str("action", "login").
    Msg("User logged in")

// ❌ BAD: String concatenation
log.Info().Msg(fmt.Sprintf("User %s logged in", userID))
```

### 6. Include Context

Always include relevant context:

```go
log.Info().
    Str("request_id", requestID).
    Str("user_id", userID).
    Str("ip", clientIP).
    Int("status", statusCode).
    Float64("duration_ms", duration).
    Msg("Request completed")
```

### 7. Avoid Log Spam

Rate limit or sample high-frequency events:

```go
// Rate limit "user online" events to once per minute
if time.Since(lastLog) < time.Minute {
    return
}
```

---

## Troubleshooting

### No Logs Appearing

**Check log level:**

```bash
# Ensure debug mode is enabled if needed
FLUXBASE_DEBUG=true
```

**Check log driver:**

```bash
# View Docker log driver
docker inspect fluxbase | jq '.[0].HostConfig.LogConfig'
```

**Check log permissions:**

```bash
# Ensure Fluxbase can write to stdout/stderr
ls -la /dev/stdout /dev/stderr
```

### Logs Too Verbose

**Disable debug logging:**

```bash
FLUXBASE_DEBUG=false
```

**Filter logs:**

```bash
# Only show warnings and errors
docker logs fluxbase 2>&1 | jq 'select(.level == "warn" or .level == "error")'
```

### Disk Space Issues

**Enable log rotation:**

```yaml
logging:
  options:
    max-size: "10m"
    max-file: "3"
```

**Clean old logs:**

```bash
# Docker
docker system prune -a

# Kubernetes
kubectl logs --tail=100 pod-name
```

---

## Summary

Fluxbase provides comprehensive structured logging:

- ✅ **Structured JSON logs** for easy parsing
- ✅ **Multiple log levels** (debug, info, warn, error, fatal)
- ✅ **Automatic request logging** with detailed context
- ✅ **Security event logging** (auth, CSRF, RLS)
- ✅ **Integration with log aggregators** (Loki, ELK, CloudWatch)
- ✅ **Redaction of sensitive data**

Configure appropriate log levels, set up log rotation, send logs to an aggregator, and use structured queries to monitor and troubleshoot your Fluxbase instance.
