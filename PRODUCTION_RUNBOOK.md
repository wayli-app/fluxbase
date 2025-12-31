# Fluxbase Production Runbook

**Version**: 2.0
**Last Updated**: 2025-10-31
**Sprint**: 11 - Production Hardening & Deployment

---

## Table of Contents

1. [System Overview](#system-overview)
2. [Prerequisites](#prerequisites)
3. [Deployment](#deployment)
4. [Configuration](#configuration)
5. [Monitoring & Observability](#monitoring--observability)
6. [Security](#security)
7. [Performance Tuning](#performance-tuning)
8. [Troubleshooting](#troubleshooting)
9. [Maintenance](#maintenance)
10. [Disaster Recovery](#disaster-recovery)

---

## System Overview

Fluxbase is a production-grade Backend-as-a-Service (BaaS) built in Go with:

- **PostgreSQL** for primary data storage
- **S3-compatible storage** for file uploads
- **WebSocket** for realtime subscriptions
- **Embedded Admin UI** (React SPA)
- **Two-tier authentication** (Dashboard admins + Application users)

### Architecture

```
┌─────────────────┐         ┌──────────────────┐
│ Dashboard Admin │         │ Application User │
│   (Admin UI)    │         │  (Frontend App)  │
└────────┬────────┘         └────────┬─────────┘
         │ HTTPS                     │ HTTPS
         │ /dashboard/*              │ /api/v1/*
         │                           │
    ┌────▼───────────────────────────▼────┐
    │       Fluxbase Server                │
    │       (Go Binary)                    │
    │  ┌────────────┐  ┌────────────────┐ │
    │  │ Dashboard  │  │  Application   │ │
    │  │   Auth     │  │     Auth       │ │
    │  └─────┬──────┘  └───────┬────────┘ │
    │        │                 │          │
    │  - Admin UI      - REST API        │
    │  - 2FA Support   - WebSocket       │
    │                  - RLS Policies    │
    └────────┬─────────────────┬──────────┘
             │                 │
        ┌────▼─────────────────▼────┐
        │      PostgreSQL            │
        │  ┌──────────┬──────────┐  │
        │  │dashboard │   auth   │  │
        │  │ schema   │  schema  │  │
        │  │  .users  │  .users  │  │
        │  └──────────┴──────────┘  │
        └────────────┬───────────────┘
                     │
              ┌──────▼──────┐
              │ S3 Storage  │
              └─────────────┘
```

### Key Components

- **Go Binary**: Single stateless binary (~23MB)
- **Database**: PostgreSQL 13+ with pgx/v5 driver
  - `dashboard` schema: Admin users with 2FA support
  - `auth` schema: Application end-users
- **Storage**: Local filesystem or S3-compatible object storage
- **Admin UI**: Embedded React application (served from binary)
- **Authentication**:
  - Dashboard: JWT with `dashboard_admin` role, optional 2FA
  - Application: JWT with `user`/`anon` roles, Row-Level Security

---

## Prerequisites

### Required

- **Go 1.25+** (for building from source)
- **PostgreSQL 13+** with extensions:
  - `uuid-ossp` - UUID generation
  - `pg_trgm` - Fuzzy text search
- **TLS certificates** (Let's Encrypt recommended)

### Optional

- **S3-compatible storage** (AWS S3, MinIO, etc.)
- **SMTP server** (for email auth)
- **Prometheus** (for metrics)
- **Grafana** (for dashboards)

### Minimum Resources

- **CPU**: 2 cores
- **RAM**: 2GB
- **Disk**: 10GB (+ database and storage needs)
- **Network**: 1Gbps

### Recommended Resources (Production)

- **CPU**: 4+ cores
- **RAM**: 8GB+
- **Disk**: SSD with 100GB+
- **Network**: 10Gbps
- **Load Balancer**: HTTPS termination, WebSocket support

---

## Deployment

### 1. Binary Deployment

**Build**:

```bash
git clone https://github.com/your-org/fluxbase
cd fluxbase
go build -ldflags="-s -w" -o fluxbase cmd/fluxbase/main.go
```

**Deploy**:

```bash
# Copy binary to server
scp fluxbase user@server:/usr/local/bin/

# Make executable
chmod +x /usr/local/bin/fluxbase

# Create systemd service
sudo tee /etc/systemd/system/fluxbase.service <<EOF
[Unit]
Description=Fluxbase Backend Service
After=network.target postgresql.service

[Service]
Type=simple
User=fluxbase
WorkingDirectory=/var/lib/fluxbase
EnvironmentFile=/etc/fluxbase/env
ExecStart=/usr/local/bin/fluxbase
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable fluxbase
sudo systemctl start fluxbase
```

### 2. Docker Deployment

**Production Dockerfile** (see [Dockerfile](../Dockerfile)):

```dockerfile
# Multi-stage build with Admin UI
FROM node:25-alpine AS admin-builder
WORKDIR /build/admin
COPY admin/ ./
RUN npm ci --only=production && npm run build

FROM golang:1.22-alpine AS go-builder
WORKDIR /build
COPY --from=admin-builder /build/admin/dist ./admin/dist
COPY . .
RUN go build -ldflags='-w -s' -o fluxbase ./cmd/fluxbase

FROM alpine:3.19
RUN adduser -u 1000 -S fluxbase
COPY --from=go-builder /build/fluxbase /app/fluxbase
USER fluxbase
EXPOSE 8080
CMD ["/app/fluxbase"]
```

**Build & Run**:

```bash
# Production build
docker build -f Dockerfile -t fluxbase:latest .

# Run with environment variables
docker run -d \
  --name fluxbase \
  -p 8080:8080 \
  -e DATABASE_URL="postgresql://..." \
  -e FLUXBASE_JWT_SECRET="..." \
  fluxbase:latest
```

**Production Stack** (see [docker-compose.production.yml](../deploy/docker-compose.production.yml)):

```bash
cd deploy
docker compose -f docker-compose.production.yml up -d
```

Includes:

- PostgreSQL with health checks
- Redis for caching
- MinIO for S3-compatible storage
- Fluxbase (3 replicas)
- Prometheus for metrics
- Grafana for dashboards
- NGINX reverse proxy

### 3. Kubernetes Deployment

**Helm Chart**:

```bash
cd deploy/helm/fluxbase
helm install fluxbase . \
  --set database.host=postgres.example.com \
  --set auth.jwtSecret=<secret> \
  --set ingress.enabled=true \
  --set ingress.host=api.example.com
```

See [deploy/helm/fluxbase/](../deploy/helm/fluxbase/) for full chart configuration.

---

## Configuration

### Environment Variables (Required)

```bash
# Database
DATABASE_URL="postgresql://user:pass@host:5432/fluxbase?sslmode=require"

# JWT Secret (CRITICAL - use strong random value)
FLUXBASE_JWT_SECRET="your-256-bit-secret-key-here"

# Server
FLUXBASE_SERVER_ADDRESS="0.0.0.0:8080"
FLUXBASE_BASE_URL="https://api.example.com"

# Storage (if using S3)
FLUXBASE_STORAGE_PROVIDER="s3"
FLUXBASE_S3_ENDPOINT="s3.amazonaws.com"
FLUXBASE_S3_ACCESS_KEY="AKIAIOSFODNN7EXAMPLE"
FLUXBASE_S3_SECRET_KEY="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
FLUXBASE_S3_BUCKET="my-bucket"
FLUXBASE_S3_REGION="us-east-1"
```

### Configuration File (`fluxbase.yaml`)

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  body_limit: 2147483648 # 2GB

database:
  max_connections: 100
  min_connections: 10
  connection_max_lifetime: "1h"
  max_conn_idle_time: "10m"
  health_check_period: "1m"

security:
  rate_limit:
    enabled: true
    requests_per_minute: 100
  csrf:
    enabled: false # Enable for browser-based apps
    cookie_secure: true # HTTPS only

realtime:
  enabled: true
  max_connections: 1000
  ping_interval: "30s"

log:
  level: "info" # Options: debug, info, warn, error
  format: "json" # Use json in production
```

### Security Checklist

- [ ] Change default JWT secret
- [ ] Enable HTTPS/TLS
- [ ] Enable CSRF protection (if browser clients)
- [ ] Configure rate limiting
- [ ] Set `cookie_secure: true` (HTTPS only)
- [ ] Use strong database password
- [ ] Rotate S3 access keys regularly
- [ ] Enable database connection SSL (`sslmode=require`)
- [ ] Restrict database firewall rules
- [ ] Enable audit logging

---

## Monitoring & Observability

For detailed monitoring setup, see [MONITORING.md](deploy/MONITORING.md).

### Health Checks

**Liveness Probe** (K8s):

```bash
curl http://localhost:8080/health
# Expected: 200 OK
```

**Readiness Probe**:

```bash
curl http://localhost:8080/ready
# Expected: 200 OK (when DB is connected)
```

### Monitoring Stack

The production deployment includes a complete monitoring stack:

- **Prometheus**: Metrics collection and storage
- **Grafana**: Pre-configured dashboards for visualization
- **PostgreSQL Exporter**: Database metrics
- **Redis Exporter**: Cache metrics
- **MinIO**: Built-in S3 storage metrics
- **Node Exporter**: System-level metrics
- **cAdvisor**: Container metrics

**Access**:

- Grafana: http://localhost:3000 (default: admin/admin)
- Prometheus: http://localhost:9090

**Dashboards**:

- Fluxbase - Application Overview
- Fluxbase - Database Metrics

### Prometheus Metrics

**Endpoint**: `http://localhost:8080/metrics`

**Key Metrics**:

- `fluxbase_http_requests_total` - Total HTTP requests
- `fluxbase_http_request_duration_seconds` - Request latency
- `fluxbase_db_queries_total` - Database query count
- `fluxbase_db_query_duration_seconds` - Query latency
- `fluxbase_db_connections` - Active DB connections
- `fluxbase_realtime_connections` - WebSocket connections
- `fluxbase_auth_attempts_total` - Auth attempts
- `fluxbase_rate_limit_hits_total` - Rate limit hits
- `fluxbase_system_uptime_seconds` - System uptime

**Prometheus scrape config**:

```yaml
scrape_configs:
  - job_name: "fluxbase"
    static_configs:
      - targets: ["localhost:8080"]
    metrics_path: "/metrics"
    scrape_interval: 15s
```

### Structured Logging

**Format**: JSON (in production)

**Log Levels**:

- `ERROR`: 5xx responses, failed queries, critical errors
- `WARN`: 4xx responses, slow queries (>1s), rate limit hits
- `INFO`: 2xx responses, startup/shutdown, config changes
- `DEBUG`: Detailed request/response data (dev only)

**Sample Log Entry**:

```json
{
  "level": "info",
  "request_id": "abc123",
  "method": "POST",
  "path": "/api/v1/tables/users",
  "status": 201,
  "duration_ms": 45,
  "ip": "192.168.1.1",
  "user_id": "user-uuid",
  "message": "HTTP request"
}
```

**Viewing Logs**:

```bash
# Systemd
journalctl -u fluxbase -f --output=json

# Docker
docker logs -f fluxbase | jq

# Kubernetes
kubectl logs -f deployment/fluxbase | jq
```

### Audit Logging

Security-sensitive events are logged with `log_type=audit`:

- Authentication (login, logout, token refresh)
- User management (create, delete, update)
- API key operations (create, revoke, regenerate)
- Configuration changes
- Security events (rate limit, CSRF, SQL injection attempts)

**Query audit logs**:

```bash
journalctl -u fluxbase | jq 'select(.log_type=="audit")'
```

---

## Security

### Dashboard Authentication

Fluxbase uses a **two-tier authentication system** to separate dashboard administrators from application end-users:

#### Dashboard Administrators

Dashboard admins access the Admin UI and manage the Fluxbase instance. They are stored in the `dashboard.users` table.

**Features**:

- Email/password authentication
- Optional Two-Factor Authentication (TOTP)
- Account locking after failed login attempts
- Activity logging for security auditing
- Session management
- Password reset via email
- Email verification

**Creating the first admin**:

```bash
# Using the API
curl -X POST http://localhost:8080/dashboard/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "SecurePassword123!",
    "full_name": "Admin User"
  }'
```

**Enabling 2FA**:

1. Log in to Admin UI
2. Navigate to Settings > Account
3. Click "Enable 2FA"
4. Scan QR code with authenticator app (Google Authenticator, Authy, etc.)
5. Enter verification code
6. **Save backup codes** in a secure location

#### Application Users

Application users are the end-users of your frontend application. They are stored in the `auth.users` table and use Row-Level Security policies.

**Features**:

- Email/password or social authentication
- JWT-based auth with refresh tokens
- Row-Level Security (RLS) policies
- Configurable password requirements
- Email verification
- Password reset

**Key Differences**:

| Feature          | Dashboard Admin     | Application User |
| ---------------- | ------------------- | ---------------- |
| Schema           | `dashboard.users`   | `auth.users`     |
| JWT Role         | `dashboard_admin`   | `user` / `anon`  |
| 2FA Support      | ✅ Yes              | ❌ No (future)   |
| Access           | Admin UI            | Application API  |
| Endpoints        | `/dashboard/auth/*` | `/api/v1/auth/*` |
| Account Locking  | ✅ Yes              | ❌ No            |
| Activity Logging | ✅ Yes              | ❌ No            |

### Service Role Keys

Service keys provide **elevated privileges** for backend services, cron jobs, and admin scripts. They bypass Row-Level Security (RLS) policies and have full database access.

**⚠️ CRITICAL WARNING**: Service keys are extremely powerful. Misuse can lead to data breaches.

#### Creating a Service Key

```bash
# 1. Generate random key
openssl rand -base64 32

# 2. Format as sk_live_<random> or sk_test_<random>
SERVICE_KEY="sk_live_abc123xyz456..."

# 3. Hash and store in database
psql -U postgres -d fluxbase <<SQL
INSERT INTO auth.service_keys (name, description, key_hash, key_prefix, enabled, expires_at)
VALUES (
  'Backend Service',
  'Service key for backend cron jobs',
  crypt('sk_live_abc123xyz456...', gen_salt('bf', 12)),
  'sk_live_',
  true,
  NOW() + INTERVAL '1 year'
);
SQL
```

#### Using Service Keys

```bash
# Via X-Service-Key header
curl -H "X-Service-Key: sk_live_abc123..." https://api.example.com/api/v1/tables/users

# Via Authorization header
curl -H "Authorization: ServiceKey sk_live_abc123..." https://api.example.com/api/v1/tables/users
```

**Best Practices**:

- ✅ Store in secrets manager (Vault, AWS Secrets Manager)
- ✅ Rotate every 90 days
- ✅ Monitor via `last_used_at` timestamp
- ✅ Set expiration dates
- ❌ Never expose to clients
- ❌ Never commit to version control
- ❌ Never log in plaintext

For detailed service key documentation, see [docs/guides/authentication.md](docs/docs/guides/authentication.md#client-keys--service-keys).

### Implemented Protections

✅ **OWASP Top 10 Compliance**:

- SQL Injection: Parameterized queries (pgx/v5)
- XSS: Content Security Policy headers
- CSRF: Token-based protection (opt-in)
- Clickjacking: X-Frame-Options: DENY
- MIME Sniffing: X-Content-Type-Options: nosniff

✅ **Rate Limiting**:

- Global: 100 req/min per IP
- Auth endpoints: 5 login attempts per 15 min
- Per-user: 500 req/min (authenticated)
- Per-API-key: 1000 req/min

✅ **Security Headers**:

- `Content-Security-Policy`
- `X-Frame-Options`
- `X-Content-Type-Options`
- `Strict-Transport-Security` (HSTS)
- `Referrer-Policy`

### Security Audit Results

**SQL Injection**: ✅ SECURE

- All queries use parameterized statements
- Column names validated against schema
- No string concatenation with user input

**Dependencies**: Keep updated

```bash
go get -u ./...
go mod tidy
```

### Incident Response

**Rate Limit Hit**:

1. Check logs: `journalctl -u fluxbase | grep rate_limit`
2. Identify source IP
3. Temporarily block if malicious: `iptables -A INPUT -s IP -j DROP`
4. Adjust rate limits if legitimate traffic

**Auth Failures**:

1. Check audit logs: `jq 'select(.log_type=="audit" and .success==false)'`
2. Identify pattern (brute force, credential stuffing)
3. Enable CAPTCHA if needed
4. Block source IPs

**Slow Queries**:

1. Check logs: `jq 'select(.slow_query==true)'`
2. Analyze query plan: `EXPLAIN ANALYZE <query>`
3. Add indexes if needed
4. Optimize query or add caching

---

## Performance Tuning

### Database Connection Pool

**Recommended Settings** (per instance):

```yaml
database:
  max_connections: 100 # Total pool size
  min_connections: 10 # Always keep 10 warm
  connection_max_lifetime: "1h"
  max_conn_idle_time: "10m"
  health_check_period: "1m"
```

**Formula**: `max_connections = (num_cpu_cores * 2) + disk_spindles`

**Monitor**:

```bash
curl localhost:8080/metrics | grep fluxbase_db_connections
```

### Slow Query Monitoring

**Threshold**: 1 second (configurable)

**Identify slow queries**:

```bash
journalctl -u fluxbase | jq 'select(.slow_query==true) | {query, duration_ms}'
```

**PostgreSQL logging** (enable for deep analysis):

```sql
-- Add to postgresql.conf
log_min_duration_statement = 1000  # Log queries > 1s
log_statement = 'all'               # Log all statements (dev only)
```

### Caching Strategy

**Client-side** (future enhancement):

- ETag support for GET requests
- Cache-Control headers

**Server-side**:

- Connection pooling (already implemented)
- Prepared statement caching (pgx default)

**Database**:

- Query result caching (PostgreSQL)
- Materialized views for complex queries

### Load Testing

**Tool**: `k6` (recommended)

**Basic test**:

```javascript
import http from "k6/http";
import { check } from "k6";

export let options = {
  stages: [
    { duration: "2m", target: 100 }, // Ramp up to 100 users
    { duration: "5m", target: 100 }, // Stay at 100 users
    { duration: "2m", target: 0 }, // Ramp down to 0 users
  ],
  thresholds: {
    http_req_duration: ["p(95)<500"], // 95% of requests < 500ms
  },
};

export default function () {
  let response = http.get("http://localhost:8080/health");
  check(response, {
    "status is 200": (r) => r.status === 200,
  });
}
```

**Run**:

```bash
k6 run load-test.js
```

**Target Performance**:

- **Throughput**: 1000 req/s (single instance)
- **Latency**: p95 < 500ms, p99 < 1s
- **Error Rate**: < 0.1%

---

## Troubleshooting

### Common Issues

#### 1. Database Connection Failures

**Symptoms**: `unable to ping database` on startup

**Causes**:

- PostgreSQL not running
- Wrong connection string
- Firewall blocking port 5432
- SSL mode mismatch

**Resolution**:

```bash
# Test connection manually
psql "postgresql://user:pass@host:5432/fluxbase"

# Check PostgreSQL status
sudo systemctl status postgresql

# Check firewall
sudo ufw allow 5432/tcp

# Check SSL requirement
# Change sslmode in DATABASE_URL if needed:
# sslmode=disable (dev only)
# sslmode=require (production)
```

#### 2. High Memory Usage

**Symptoms**: OOM kills, high `fluxbase_db_connections`

**Causes**:

- Too many database connections
- Connection leaks
- Large response payloads

**Resolution**:

```bash
# Reduce max_connections
# Monitor connection usage
curl localhost:8080/metrics | grep fluxbase_db_connections

# Check for leaks
journalctl -u fluxbase | grep "connection"

# Restart if needed
sudo systemctl restart fluxbase
```

#### 3. Slow Requests

**Symptoms**: High `fluxbase_http_request_duration_seconds`

**Causes**:

- Slow database queries
- Missing indexes
- Large datasets without pagination
- Network latency

**Resolution**:

```bash
# Identify slow queries
journalctl -u fluxbase | jq 'select(.slow_query==true)'

# Add indexes
psql -c "CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);"

# Check query plans
psql -c "EXPLAIN ANALYZE SELECT * FROM users WHERE email='...'"
```

#### 4. WebSocket Disconnects

**Symptoms**: Frequent reconnects, `realtime_connection_errors`

**Causes**:

- Load balancer timeout
- Proxy buffering
- Client network issues

**Resolution**:

```nginx
# Nginx config for WebSocket
location /realtime {
    proxy_pass http://fluxbase;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_read_timeout 3600s;  # 1 hour timeout
    proxy_send_timeout 3600s;
}
```

---

## Maintenance

### Routine Tasks

**Daily**:

- [ ] Check error logs
- [ ] Monitor disk space
- [ ] Review rate limit hits

**Weekly**:

- [ ] Review slow queries
- [ ] Check database backup success
- [ ] Analyze performance metrics
- [ ] Review security audit logs

**Monthly**:

- [ ] Update dependencies (`go get -u`)
- [ ] Rotate client keys
- [ ] Review and archive old logs
- [ ] Capacity planning review

### Database Maintenance

**Vacuum** (automatically handled by PostgreSQL):

```sql
-- Manual vacuum if needed
VACUUM ANALYZE;

-- Check table bloat
SELECT schemaname, tablename, pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename))
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

**Index Maintenance**:

```sql
-- Rebuild indexes if needed
REINDEX DATABASE fluxbase;

-- Analyze query performance
SELECT * FROM pg_stat_user_tables;
```

### Upgrades

**Zero-Downtime Strategy**:

1. Deploy new version to staging
2. Run migrations (use `migrate` CLI)
3. Blue-green deployment:
   - Start new instances
   - Health check passes
   - Route traffic to new version
   - Drain old instances
   - Shutdown old instances

**Rollback Plan**:

- Keep previous binary: `/usr/local/bin/fluxbase.backup`
- Database migrations are versioned
- Rollback: `systemctl stop fluxbase && cp fluxbase.backup fluxbase && systemctl start fluxbase`

---

## Disaster Recovery

### Backup Strategy

Fluxbase includes automated backup scripts. See [deploy/scripts/backup.sh](deploy/scripts/backup.sh).

**Automated Backups**:

```bash
# Local backup
./deploy/scripts/backup.sh local

# S3 backup (with environment variables configured)
./deploy/scripts/backup.sh s3
```

**Backup Components**:

- PostgreSQL database (pg_dump format)
- Configuration files
- Local storage (if used)

**Backup Retention**: 30 days (configurable via `RETENTION_DAYS`)

**Manual Database Backups**:

```bash
# Daily full backup
pg_dump -Fc fluxbase > fluxbase-$(date +%Y%m%d).dump

# Continuous archiving (PITR)
# Configure in postgresql.conf:
wal_level = replica
archive_mode = on
archive_command = 'test ! -f /mnt/archive/%f && cp %p /mnt/archive/%f'
```

**Storage Backups**:

- S3: Enable versioning
- Local: Included in backup script

**Configuration Backups**:
Included in backup script or manually:

```bash
tar czf fluxbase-config-$(date +%Y%m%d).tar.gz \
  /etc/fluxbase/env \
  /var/lib/fluxbase/fluxbase.yaml
```

### Recovery Procedures

**Automated Restore**:

```bash
# Restore from local backup
./deploy/scripts/restore.sh 20251031_140530

# Restore from S3
./deploy/scripts/restore.sh 20251031_140530 s3
```

**Manual Database Restore**:

```bash
# Stop Fluxbase
sudo systemctl stop fluxbase

# Restore database
pg_restore -d fluxbase fluxbase-20251031.dump

# Start Fluxbase
sudo systemctl start fluxbase
```

**Complete System Restore**:

1. Provision new server
2. Install PostgreSQL
3. Restore database from backup
4. Copy Fluxbase binary
5. Restore configuration files
6. Start services
7. Verify health checks

**RTO** (Recovery Time Objective): < 1 hour
**RPO** (Recovery Point Objective): < 15 minutes

---

## Support

**Documentation**: https://docs.fluxbase.io
**GitHub Issues**: https://github.com/your-org/fluxbase/issues
**Community Discord**: https://discord.gg/BXPRHkQzkA

---

## Appendix

### Useful Commands

```bash
# Check version
./fluxbase --version

# Validate configuration
./fluxbase --config-test

# Run migrations
./fluxbase migrate

# Generate API key
openssl rand -base64 32

# Monitor real-time logs
journalctl -u fluxbase -f | jq -C

# Check listening ports
sudo netstat -tulpn | grep fluxbase

# Prometheus metrics snapshot
curl -s localhost:8080/metrics | grep -v '^#'
```

### Performance Baselines

**Single Instance** (4 CPU, 8GB RAM):

- Throughput: 1000-1500 req/s
- Latency p50: 50ms
- Latency p95: 200ms
- Latency p99: 500ms
- Max WebSocket connections: 1000
- Database connections: 50-100

**Tuned** (8 CPU, 16GB RAM):

- Throughput: 3000-5000 req/s
- Latency p50: 30ms
- Latency p95: 100ms
- Latency p99: 300ms
- Max WebSocket connections: 5000
- Database connections: 100-200

---

**End of Runbook**

_For questions or updates, contact DevOps team or file an issue on GitHub._
