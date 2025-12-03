---
title: "Scaling Guide"
---

Learn how to scale Fluxbase to handle increased traffic and optimize performance for production workloads.

## Overview

Fluxbase supports both vertical scaling (bigger machines) and horizontal scaling (more machines). This guide covers:

- Horizontal scaling strategies
- Database optimization
- Caching strategies
- Load balancing
- Performance tuning
- Cost optimization

## Scaling Strategies

### Vertical Scaling (Scale Up)

**When to use**: Quick fix for performance issues, simpler setup

**Pros**:

- Simple to implement
- No code changes needed
- Maintains data consistency easily

**Cons**:

- Limited by hardware
- Single point of failure
- Expensive at scale

**Example**:

```yaml
# Increase resources for single instance
resources:
  requests:
    cpu: 2000m
    memory: 8Gi
  limits:
    cpu: 4000m
    memory: 16Gi
```

### Horizontal Scaling (Scale Out) {#horizontal-scaling}

**When to use**: Long-term growth, high availability needs

**Pros**:

- Nearly unlimited scaling
- High availability
- Cost-effective at scale

**Cons**:

- More complex setup
- Requires load balancer
- Session management considerations

**Example**:

```yaml
# Kubernetes HPA
autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 20
  targetCPU: 70
  targetMemory: 80
```

### Horizontal Scaling Requirements

To scale Fluxbase horizontally across multiple instances, you need to ensure all stateful components are externalized and shared:

#### Prerequisites Checklist

✅ **External PostgreSQL Database**

- Cannot use embedded/local PostgreSQL
- Must be accessible by all Fluxbase instances
- Configure with `DATABASE_URL` environment variable
- **Note**: PostgreSQL also stores authentication sessions (shared across all instances)

✅ **S3-Compatible Object Storage (MinIO/AWS S3/DigitalOcean Spaces)**

- Cannot use local filesystem storage (`provider: local`)
- Must configure S3 provider for shared storage
- All instances must access the same S3 bucket

✅ **Load Balancer with Session Stickiness**

- Required for WebSocket connections (realtime subscriptions)
- Configure sticky sessions based on source IP or cookies
- Ensures WebSocket connections remain on the same instance

#### Current Limitations

⚠️ **Per-Instance State** (not shared across instances):

- **Rate limiting**: Request counters are stored in-memory per instance
- **CSRF tokens**: Anti-CSRF tokens are stored in-memory per instance
- Session stickiness helps mitigate these limitations for individual users

💡 **Future Enhancement**: Redis support planned to enable:

- Shared rate limiting across all instances
- Shared CSRF token validation
- Faster session lookups (compared to PostgreSQL)

#### Configuration Example

```bash
# Fluxbase environment variables for horizontal scaling
DATABASE_URL=postgres://user:pass@postgres.example.com:5432/fluxbase

# S3 Storage (required)
FLUXBASE_STORAGE_PROVIDER=s3
FLUXBASE_STORAGE_S3_ENDPOINT=minio.example.com:9000
FLUXBASE_STORAGE_S3_ACCESS_KEY=minioadmin
FLUXBASE_STORAGE_S3_SECRET_KEY=minioadmin
FLUXBASE_STORAGE_S3_REGION=us-east-1
FLUXBASE_STORAGE_S3_BUCKET=fluxbase
```

#### Load Balancer Configuration

**Nginx with Session Stickiness for Realtime:**

```nginx
upstream fluxbase_backend {
    # Use IP hash for WebSocket sticky sessions
    ip_hash;

    server fluxbase-1:8080 max_fails=3 fail_timeout=30s;
    server fluxbase-2:8080 max_fails=3 fail_timeout=30s;
    server fluxbase-3:8080 max_fails=3 fail_timeout=30s;

    keepalive 32;
}

server {
    listen 443 ssl http2;
    server_name api.example.com;

    location / {
        proxy_pass http://fluxbase_backend;
        proxy_http_version 1.1;

        # WebSocket support
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # Sticky session headers
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;

        # Health check fallback
        proxy_next_upstream error timeout http_500 http_502 http_503;
    }
}
```

**HAProxy with Sticky Sessions:**

```ini
frontend fluxbase_front
    bind *:443 ssl crt /etc/ssl/certs/fluxbase.pem
    default_backend fluxbase_back

backend fluxbase_back
    balance leastconn
    option httpchk GET /health
    http-check expect status 200

    # Sticky sessions based on source IP
    stick-table type ip size 100k expire 30m
    stick on src

    server fluxbase1 10.0.1.10:8080 check inter 5s rise 2 fall 3
    server fluxbase2 10.0.1.11:8080 check inter 5s rise 2 fall 3
    server fluxbase3 10.0.1.12:8080 check inter 5s rise 2 fall 3
```

#### Docker Compose Multi-Instance Example

```yaml
version: "3.8"

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: fluxbase
      POSTGRES_USER: fluxbase
      POSTGRES_PASSWORD: secure-password
    volumes:
      - postgres_data:/var/lib/postgresql/data

  minio:
    image: minio/minio
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    volumes:
      - minio_data:/data

  fluxbase-1:
    image: ghcr.io/wayli-app/fluxbase:latest
    environment:
      DATABASE_URL: postgres://fluxbase:secure-password@postgres:5432/fluxbase
      FLUXBASE_STORAGE_PROVIDER: s3
      FLUXBASE_STORAGE_S3_ENDPOINT: minio:9000
      FLUXBASE_STORAGE_S3_ACCESS_KEY: minioadmin
      FLUXBASE_STORAGE_S3_SECRET_KEY: minioadmin
      FLUXBASE_STORAGE_S3_USE_SSL: "false"
      FLUXBASE_STORAGE_S3_BUCKET: fluxbase
    depends_on:
      - postgres
      - minio

  fluxbase-2:
    image: ghcr.io/wayli-app/fluxbase:latest
    environment:
      DATABASE_URL: postgres://fluxbase:secure-password@postgres:5432/fluxbase
      FLUXBASE_STORAGE_PROVIDER: s3
      FLUXBASE_STORAGE_S3_ENDPOINT: minio:9000
      FLUXBASE_STORAGE_S3_ACCESS_KEY: minioadmin
      FLUXBASE_STORAGE_S3_SECRET_KEY: minioadmin
      FLUXBASE_STORAGE_S3_USE_SSL: "false"
      FLUXBASE_STORAGE_S3_BUCKET: fluxbase
    depends_on:
      - postgres
      - minio

  nginx:
    image: nginx:alpine
    ports:
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
    depends_on:
      - fluxbase-1
      - fluxbase-2

volumes:
  postgres_data:
  minio_data:
```

#### Kubernetes Multi-Instance Example

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: fluxbase
  namespace: fluxbase
spec:
  replicas: 3
  selector:
    matchLabels:
      app: fluxbase
  template:
    metadata:
      labels:
        app: fluxbase
    spec:
      containers:
        - name: fluxbase
          image: ghcr.io/wayli-app/fluxbase:latest
          env:
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: fluxbase-secrets
                  key: database-url
            - name: FLUXBASE_STORAGE_PROVIDER
              value: "s3"
            - name: FLUXBASE_STORAGE_S3_ENDPOINT
              value: "minio.storage.svc.cluster.local:9000"
            - name: FLUXBASE_STORAGE_S3_ACCESS_KEY
              valueFrom:
                secretKeyRef:
                  name: minio-secrets
                  key: access-key
            - name: FLUXBASE_STORAGE_S3_SECRET_KEY
              valueFrom:
                secretKeyRef:
                  name: minio-secrets
                  key: secret-key
            - name: FLUXBASE_STORAGE_S3_BUCKET
              value: "fluxbase"
          resources:
            requests:
              cpu: 500m
              memory: 1Gi
            limits:
              cpu: 2000m
              memory: 4Gi
---
apiVersion: v1
kind: Service
metadata:
  name: fluxbase
  namespace: fluxbase
spec:
  type: ClusterIP
  sessionAffinity: ClientIP # Sticky sessions for WebSocket
  sessionAffinityConfig:
    clientIP:
      timeoutSeconds: 3600
  selector:
    app: fluxbase
  ports:
    - port: 8080
      targetPort: 8080
```

#### Architecture Diagram

```mermaid
graph TB
    A[Client 1] -->|HTTPS| LB[Load Balancer<br/>Session Stickiness]
    B[Client 2] -->|HTTPS| LB
    C[Client 3] -->|HTTPS| LB

    LB -->|Sticky Session| FB1[Fluxbase Instance 1]
    LB -->|Sticky Session| FB2[Fluxbase Instance 2]
    LB -->|Sticky Session| FB3[Fluxbase Instance 3]

    FB1 -->|SQL + Sessions| DB[(PostgreSQL<br/>Shared Database)]
    FB2 -->|SQL + Sessions| DB
    FB3 -->|SQL + Sessions| DB

    FB1 -->|S3 API| S3[MinIO / S3<br/>Shared Storage]
    FB2 -->|S3 API| S3
    FB3 -->|S3 API| S3

    style LB fill:#ff6b6b,color:#fff
    style FB1 fill:#3178c6,color:#fff
    style FB2 fill:#3178c6,color:#fff
    style FB3 fill:#3178c6,color:#fff
    style DB fill:#336791,color:#fff
    style S3 fill:#c92a2a,color:#fff
```

**Key Points:**

- Each Fluxbase instance shares state via external PostgreSQL and S3/MinIO
- **PostgreSQL** stores: Database records + authentication sessions + token blacklist
- **MinIO/S3** stores: User-uploaded files and objects
- Load balancer uses session stickiness to route WebSocket connections consistently
- Rate limiting and CSRF tokens are per-instance (not shared)

#### Why Session Stickiness for Realtime?

Fluxbase uses PostgreSQL's `LISTEN/NOTIFY` for realtime subscriptions. Each instance maintains its own PostgreSQL connection and LISTEN channel. When a database change occurs:

1. PostgreSQL broadcasts `pg_notify()` to **all** connected instances
2. Each instance forwards the notification to its connected WebSocket clients
3. If a client's WebSocket connection is on Instance 1, only Instance 1 can send the update to that client

Without sticky sessions, a client might establish a WebSocket on Instance 1, but subsequent requests could route to Instance 2, breaking the realtime connection.

**Alternative:** Implement Redis Pub/Sub for realtime (future feature) to eliminate the need for sticky sessions.
---

## Application Scaling

### Kubernetes Horizontal Pod Autoscaling

#### Basic CPU/Memory Scaling

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: fluxbase-hpa
  namespace: fluxbase
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: fluxbase
  minReplicas: 3
  maxReplicas: 20
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
        - type: Percent
          value: 50
          periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 0
      policies:
        - type: Percent
          value: 100
          periodSeconds: 30
        - type: Pods
          value: 4
          periodSeconds: 30
      selectPolicy: Max
```

#### Custom Metrics Scaling

Scale based on request rate:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: fluxbase-hpa-custom
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: fluxbase
  minReplicas: 3
  maxReplicas: 30
  metrics:
    - type: Pods
      pods:
        metric:
          name: http_requests_per_second
        target:
          type: AverageValue
          averageValue: "1000"
```

**Install Prometheus Adapter for custom metrics**:

```bash
helm install prometheus-adapter prometheus-community/prometheus-adapter \
  --namespace monitoring \
  --set prometheus.url=http://prometheus-server.monitoring.svc \
  --set rules.default=false
```

### Docker Swarm Scaling

```bash
# Scale service to 10 replicas
docker service scale fluxbase=10

# Auto-scaling (requires external tool like Orbiter)
# Example with resource constraints
docker service update \
  --replicas-max-per-node 2 \
  --reserve-cpu 500m \
  --reserve-memory 1G \
  fluxbase
```

---

## Database Scaling

### Connection Pooling

**Critical for horizontal scaling** - prevents connection exhaustion.

#### Fluxbase Configuration

```bash
# Scale based on number of app instances
# Formula: max_connections = (num_pods × connections_per_pod) + buffer
# Example: 10 pods × 25 connections + 25 buffer = 275 connections

# Per-pod settings
FLUXBASE_DATABASE_MAX_CONNECTIONS=25
FLUXBASE_DATABASE_MIN_CONNECTIONS=5
FLUXBASE_DATABASE_MAX_CONN_LIFETIME=1h
FLUXBASE_DATABASE_MAX_CONN_IDLE_TIME=30m
```

#### PostgreSQL Configuration

```sql
-- Calculate max_connections based on your setup
-- Default: 100
-- Recommended for production: 200-500

ALTER SYSTEM SET max_connections = 300;
ALTER SYSTEM SET shared_buffers = '4GB';
ALTER SYSTEM SET effective_cache_size = '12GB';
ALTER SYSTEM SET maintenance_work_mem = '1GB';
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
ALTER SYSTEM SET wal_buffers = '16MB';
ALTER SYSTEM SET default_statistics_target = 100;
ALTER SYSTEM SET random_page_cost = 1.1;
ALTER SYSTEM SET effective_io_concurrency = 200;
ALTER SYSTEM SET work_mem = '16MB';
ALTER SYSTEM SET min_wal_size = '1GB';
ALTER SYSTEM SET max_wal_size = '4GB';

-- Reload configuration
SELECT pg_reload_conf();
```

### Read Replicas

Distribute read traffic across multiple database replicas.

#### PostgreSQL Replication

```bash
# On primary
ALTER SYSTEM SET wal_level = replica;
ALTER SYSTEM SET max_wal_senders = 10;
ALTER SYSTEM SET max_replication_slots = 10;
ALTER SYSTEM SET hot_standby = on;

# Create replication user
CREATE USER replicator WITH REPLICATION ENCRYPTED PASSWORD 'secure-password';
```

#### Fluxbase Read Replica Support (Future)

```bash
# Configure read replica endpoints
FLUXBASE_DATABASE_READ_REPLICAS=replica1.example.com:5432,replica2.example.com:5432

# Read operations will be distributed across replicas
# Writes always go to primary
```

**Current workaround**: Use PgBouncer or HAProxy for read replica routing.

### Database Sharding (Advanced)

For massive scale (millions of users):

1. **Horizontal Partitioning**: Split tables by user ID range

   ```sql
   -- Example: Partition users table
   CREATE TABLE users_0_1000000 PARTITION OF users
     FOR VALUES FROM (0) TO (1000000);

   CREATE TABLE users_1000000_2000000 PARTITION OF users
     FOR VALUES FROM (1000000) TO (2000000);
   ```

2. **Separate Databases**: Different databases per tenant/region
   ```bash
   # Configure multiple databases
   FLUXBASE_DATABASE_SHARD_0=postgres://db1.example.com/fluxbase
   FLUXBASE_DATABASE_SHARD_1=postgres://db2.example.com/fluxbase
   ```

---

## Caching Strategies

### Redis Caching

#### Setup Redis

```yaml
# Kubernetes
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis
spec:
  replicas: 1
  template:
    spec:
      containers:
        - name: redis
          image: redis:7-alpine
          resources:
            requests:
              memory: "2Gi"
              cpu: "500m"
            limits:
              memory: "4Gi"
              cpu: "2000m"
---
# Redis Cluster for HA
apiVersion: v1
kind: Service
metadata:
  name: redis-cluster
spec:
  clusterIP: None
  ports:
    - port: 6379
  selector:
    app: redis
```

#### Fluxbase Configuration

```bash
FLUXBASE_REDIS_ENABLED=true
FLUXBASE_REDIS_HOST=redis-cluster
FLUXBASE_REDIS_PORT=6379
FLUXBASE_REDIS_PASSWORD=secure-password
FLUXBASE_REDIS_DB=0
FLUXBASE_REDIS_POOL_SIZE=10
```

#### What to Cache

1. **Session Data**: User sessions and JWT tokens
2. **API Responses**: Frequently accessed data
3. **Rate Limiting State**: Request counters
4. **Database Query Results**: Expensive queries

#### Cache Invalidation

```go
// Example: Cache with TTL
// Fluxbase handles this internally, but for reference:
cache.Set("user:123", userData, 15*time.Minute)

// Invalidate on update
cache.Delete("user:123")
```

### CDN Caching

**For static assets and API responses**:

```nginx
# Nginx caching configuration
proxy_cache_path /var/cache/nginx levels=1:2 keys_zone=api_cache:10m max_size=1g inactive=60m;

server {
    location /api/v1/public {
        proxy_cache api_cache;
        proxy_cache_valid 200 5m;
        proxy_cache_use_stale error timeout updating http_500 http_502 http_503 http_504;
        proxy_cache_background_update on;
        proxy_cache_lock on;

        add_header X-Cache-Status $upstream_cache_status;
        proxy_pass http://fluxbase;
    }
}
```

**CDN Providers**:

- Cloudflare (Free tier available)
- AWS CloudFront
- Google Cloud CDN
- Fastly

---

## Load Balancing

### Layer 7 (Application) Load Balancing

#### Nginx

```nginx
upstream fluxbase_backend {
    least_conn;  # Algorithm: least_conn, ip_hash, random

    server fluxbase-1:8080 max_fails=3 fail_timeout=30s;
    server fluxbase-2:8080 max_fails=3 fail_timeout=30s;
    server fluxbase-3:8080 max_fails=3 fail_timeout=30s;

    keepalive 32;
}

server {
    listen 443 ssl http2;
    server_name api.example.com;

    location / {
        proxy_pass http://fluxbase_backend;
        proxy_http_version 1.1;
        proxy_set_header Connection "";

        # Health check
        proxy_next_upstream error timeout http_500 http_502 http_503;
    }

    # Health check endpoint
    location /health {
        proxy_pass http://fluxbase_backend/health;
        access_log off;
    }
}
```

#### HAProxy

```ini
frontend fluxbase_front
    bind *:443 ssl crt /etc/ssl/certs/fluxbase.pem
    default_backend fluxbase_back

backend fluxbase_back
    balance leastconn
    option httpchk GET /health
    http-check expect status 200

    server fluxbase1 10.0.1.10:8080 check inter 5s rise 2 fall 3
    server fluxbase2 10.0.1.11:8080 check inter 5s rise 2 fall 3
    server fluxbase3 10.0.1.12:8080 check inter 5s rise 2 fall 3
```

### Layer 4 (TCP) Load Balancing

```yaml
# AWS Network Load Balancer (NLB)
apiVersion: v1
kind: Service
metadata:
  name: fluxbase
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
    service.beta.kubernetes.io/aws-load-balancer-cross-zone-load-balancing-enabled: "true"
spec:
  type: LoadBalancer
  selector:
    app: fluxbase
  ports:
    - port: 443
      targetPort: 8080
```

### Session Affinity

For sticky sessions (if needed):

```yaml
# Kubernetes
service:
  sessionAffinity: ClientIP
  sessionAffinityConfig:
    clientIP:
      timeoutSeconds: 3600
```

```nginx
# Nginx
upstream fluxbase_backend {
    ip_hash;  # Sticky sessions based on IP
    server fluxbase-1:8080;
    server fluxbase-2:8080;
}
```

**Note**: Fluxbase is stateless by default (sessions in Redis), so sticky sessions are usually not needed.

---

## Performance Optimization

### Application-Level Optimization

#### Request Batching

```typescript
// Client-side: Batch requests
const [users, posts, comments] = await Promise.all([
  fluxbase.from("users").select("*").limit(10),
  fluxbase.from("posts").select("*").limit(10),
  fluxbase.from("comments").select("*").limit(10),
]);
```

#### Connection Pooling

```bash
# Optimize pool size
FLUXBASE_DATABASE_MAX_CONNECTIONS=25  # Per pod
FLUXBASE_DATABASE_MIN_CONNECTIONS=5

# Adjust based on load
# Low traffic: 10-25 connections
# Medium traffic: 25-50 connections
# High traffic: 50-100 connections (scale horizontally instead)
```

#### Query Optimization

```sql
-- Add indexes for frequently queried columns
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_posts_user_id ON posts(user_id);
CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(created_at DESC);

-- Composite indexes for multi-column queries
CREATE INDEX IF NOT EXISTS idx_posts_user_created ON posts(user_id, created_at DESC);

-- Partial indexes for filtered queries
CREATE INDEX IF NOT EXISTS idx_active_users ON users(email) WHERE active = true;

-- Monitor slow queries
SELECT query, calls, total_time, mean_time
FROM pg_stat_statements
ORDER BY mean_time DESC
LIMIT 20;
```

### Database-Level Optimization

#### PostgreSQL Tuning

```sql
-- Vacuum and analyze regularly
VACUUM ANALYZE;

-- Auto-vacuum settings
ALTER SYSTEM SET autovacuum_vacuum_scale_factor = 0.1;
ALTER SYSTEM SET autovacuum_analyze_scale_factor = 0.05;

-- Increase statistics for better query plans
ALTER TABLE users ALTER COLUMN email SET STATISTICS 1000;

-- Parallel query execution
ALTER SYSTEM SET max_parallel_workers_per_gather = 4;
ALTER SYSTEM SET max_parallel_workers = 8;
```

#### Monitoring Query Performance

```sql
-- Enable pg_stat_statements
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- Find slow queries
SELECT
  query,
  calls,
  total_exec_time,
  mean_exec_time,
  max_exec_time
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 20;

-- Find queries with high I/O
SELECT
  query,
  calls,
  shared_blks_hit,
  shared_blks_read,
  shared_blks_read / NULLIF(shared_blks_hit + shared_blks_read, 0)::float AS cache_miss_ratio
FROM pg_stat_statements
ORDER BY cache_miss_ratio DESC
LIMIT 20;
```

### Network Optimization

#### HTTP/2 and Compression

```nginx
# Nginx
server {
    listen 443 ssl http2;

    # Compression
    gzip on;
    gzip_vary on;
    gzip_min_length 1000;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml;

    # Brotli (if available)
    brotli on;
    brotli_comp_level 6;
    brotli_types text/plain text/css application/json application/javascript text/xml application/xml;
}
```

#### Connection Pooling at Load Balancer

```nginx
upstream fluxbase_backend {
    keepalive 64;  # Keep 64 idle connections
    keepalive_requests 100;
    keepalive_timeout 60s;
}
```

---

## Monitoring for Scaling Decisions

### Key Metrics to Monitor

#### Application Metrics

```txt
# Request rate
rate(http_requests_total[5m])

# Response time (p95)
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# Error rate
rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m])

# Active connections
fluxbase_active_connections
```

#### Database Metrics

```txt
# Connection pool usage
pg_stat_database_connections / pg_settings_max_connections

# Query latency
rate(pg_stat_statements_total_time[5m]) / rate(pg_stat_statements_calls[5m])

# Cache hit ratio
pg_stat_database_blks_hit / (pg_stat_database_blks_hit + pg_stat_database_blks_read)
```

#### System Metrics

```txt
# CPU usage
rate(container_cpu_usage_seconds_total[5m])

# Memory usage
container_memory_working_set_bytes / container_spec_memory_limit_bytes

# Disk I/O
rate(container_fs_reads_bytes_total[5m])
rate(container_fs_writes_bytes_total[5m])
```

### Scaling Triggers

| Metric               | Threshold       | Action                         |
| -------------------- | --------------- | ------------------------------ |
| CPU usage            | > 70% for 5 min | Scale up pods                  |
| Memory usage         | > 80% for 5 min | Scale up pods                  |
| Request queue        | > 100 requests  | Scale up pods                  |
| Response time (p95)  | > 1s            | Scale up or optimize           |
| Database connections | > 80% of max    | Scale database or add replicas |
| Error rate           | > 1%            | Investigate and fix            |

---

## Scaling Roadmap by User Count

### < 1,000 Users

**Setup**:

- 1-2 Fluxbase instances
- Single PostgreSQL instance
- No Redis (optional)

```yaml
resources:
  requests:
    cpu: 250m
    memory: 512Mi
  limits:
    cpu: 500m
    memory: 1Gi
```

### 1,000 - 10,000 Users

**Setup**:

- 2-3 Fluxbase instances
- PostgreSQL with 2-4 cores
- Redis for caching

```yaml
resources:
  requests:
    cpu: 500m
    memory: 1Gi
  limits:
    cpu: 1000m
    memory: 2Gi
```

### 10,000 - 100,000 Users

**Setup**:

- 3-5 Fluxbase instances with HPA
- PostgreSQL with 4-8 cores
- PostgreSQL read replica
- Redis cluster
- CDN for static assets

```yaml
resources:
  requests:
    cpu: 1000m
    memory: 2Gi
  limits:
    cpu: 2000m
    memory: 4Gi

autoscaling:
  minReplicas: 3
  maxReplicas: 10
```

### 100,000 - 1,000,000 Users

**Setup**:

- 5-10 Fluxbase instances with aggressive HPA
- PostgreSQL cluster (primary + 2+ replicas)
- Redis Cluster (3+ nodes)
- Multi-region deployment
- Advanced monitoring and alerting

```yaml
resources:
  requests:
    cpu: 2000m
    memory: 4Gi
  limits:
    cpu: 4000m
    memory: 8Gi

autoscaling:
  minReplicas: 5
  maxReplicas: 20
```

### 1,000,000+ Users

**Setup**:

- 10-50+ Fluxbase instances across regions
- PostgreSQL sharding or distributed database (CockroachDB, Yugabyte)
- Redis Cluster with persistence
- Global CDN
- Multi-region active-active deployment
- Dedicated monitoring infrastructure

```yaml
resources:
  requests:
    cpu: 4000m
    memory: 8Gi
  limits:
    cpu: 8000m
    memory: 16Gi

autoscaling:
  minReplicas: 10
  maxReplicas: 50
```

---

## Cost Optimization

### Right-sizing Resources

```bash
# Monitor actual usage
kubectl top pods -n fluxbase

# Adjust based on actual needs (start conservative, scale up)
resources:
  requests:
    cpu: 500m      # Start with actual usage + 20%
    memory: 1Gi
  limits:
    cpu: 1000m     # 2x requests for burst
    memory: 2Gi
```

### Autoscaling for Cost Savings

```yaml
autoscaling:
  enabled: true
  minReplicas: 3 # Minimum for HA during business hours
  maxReplicas: 10
  targetCPU: 70

  # Scale down during off-hours (custom schedule)
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300 # Wait 5 min before scaling down
```

### Use Spot Instances (Non-critical Workloads)

```yaml
# Kubernetes node affinity for spot instances
nodeSelector:
  node.kubernetes.io/instance-type: spot

tolerations:
  - key: "spot"
    operator: "Equal"
    value: "true"
    effect: "NoSchedule"
```

### Database Cost Optimization

1. **Use read replicas** instead of larger primary
2. **Archive old data** to cheaper storage
3. **Enable compression** for tables
4. **Use connection pooling** to reduce instance size

---

## Troubleshooting Performance Issues

### High CPU Usage

**Diagnosis**:

```bash
# Check CPU usage per pod
kubectl top pods -n fluxbase

# Profile application
FLUXBASE_DEBUG=true
```

**Solutions**:

1. Scale horizontally (add more pods)
2. Optimize expensive operations
3. Enable caching

### High Memory Usage

**Diagnosis**:

```bash
# Check memory usage
kubectl top pods -n fluxbase

# Check for memory leaks
kubectl logs <pod> -n fluxbase | grep -i "memory"
```

**Solutions**:

1. Increase memory limits
2. Check for memory leaks
3. Optimize query results (pagination)

### Slow Database Queries

**Diagnosis**:

```sql
SELECT * FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 10;
```

**Solutions**:

1. Add indexes
2. Optimize query
3. Use caching
4. Add read replicas

### Connection Pool Exhaustion

**Diagnosis**:

```bash
# Check database connections
kubectl logs <pod> -n fluxbase | grep "connection pool"
```

**Solutions**:

1. Increase `max_connections` in PostgreSQL
2. Increase `FLUXBASE_DATABASE_MAX_CONNECTIONS`
3. Scale horizontally instead of increasing connections

---

## Next Steps

- [Production Checklist](production-checklist) - Pre-deployment verification
- [Monitoring Guide](../guides/monitoring-observability) - Set up observability
- [Database Guide](../guides/typescript-sdk/database) - Database operations and best practices
