# Fluxbase Advanced Guides

**In-depth guides for production deployments, scaling, and optimization**

This document covers advanced topics for production Fluxbase deployments. Each section includes best practices, code examples, and troubleshooting guidance.

## 📚 Table of Contents

1. [Row-Level Security Patterns](#row-level-security-patterns)
2. [Performance Optimization](#performance-optimization)
3. [Scaling Strategies](#scaling-strategies)
4. [Monitoring & Observability](#monitoring--observability)
5. [Security Hardening](#security-hardening)
6. [Disaster Recovery](#disaster-recovery)
7. [Multi-Tenancy](#multi-tenancy)
8. [Edge Functions Best Practices](#edge-functions-best-practices)

---

## Row-Level Security Patterns

### Understanding RLS

Row-Level Security (RLS) is PostgreSQL's built-in data isolation mechanism. Fluxbase propagates authenticated user context to PostgreSQL via session variables.

**How it Works**:

```
Client Request with JWT
       ↓
Fluxbase Auth Middleware (validates JWT)
       ↓
Extract user_id from token
       ↓
Set PostgreSQL session variables:
  SET LOCAL app.user_id = 'uuid'
  SET LOCAL app.role = 'authenticated'
       ↓
Execute Query
       ↓
PostgreSQL RLS policies filter results
```

### Pattern 1: User Owns Resource

**Use Case**: Users can only access their own data (todos, posts, documents)

```sql
-- Enable RLS
ALTER TABLE posts ENABLE ROW LEVEL SECURITY;

-- Policy: Users see only their posts
CREATE POLICY "users_own_posts"
  ON posts
  FOR ALL
  USING (user_id::text = current_setting('app.user_id', true))
  WITH CHECK (user_id::text = current_setting('app.user_id', true));
```

**Client Code** (automatically enforced):

```typescript
// User only sees their own posts
const { data } = await fluxbase
  .from('posts')
  .select('*')
// RLS automatically filters by user_id
```

### Pattern 2: Organization/Team Access

**Use Case**: Users access data within their organization

```sql
-- Users table with organization
CREATE TABLE users (
  id UUID PRIMARY KEY,
  email TEXT,
  organization_id UUID NOT NULL
);

-- Posts with organization
CREATE TABLE posts (
  id UUID PRIMARY KEY,
  user_id UUID REFERENCES users(id),
  organization_id UUID NOT NULL,
  title TEXT
);

-- Policy: Users see posts in their organization
CREATE POLICY "organization_access"
  ON posts
  FOR SELECT
  USING (
    organization_id IN (
      SELECT organization_id FROM users
      WHERE id::text = current_setting('app.user_id', true)
    )
  );
```

### Pattern 3: Role-Based Access

**Use Case**: Different permissions for admins, editors, viewers

```sql
-- User roles
CREATE TABLE user_roles (
  user_id UUID REFERENCES users(id),
  role TEXT NOT NULL,  -- 'admin', 'editor', 'viewer'
  PRIMARY KEY (user_id, role)
);

-- Admin can see all posts
CREATE POLICY "admin_see_all"
  ON posts
  FOR SELECT
  USING (
    EXISTS (
      SELECT 1 FROM user_roles
      WHERE user_id::text = current_setting('app.user_id', true)
      AND role = 'admin'
    )
  );

-- Editors can modify their own posts
CREATE POLICY "editors_update_own"
  ON posts
  FOR UPDATE
  USING (
    user_id::text = current_setting('app.user_id', true) AND
    EXISTS (
      SELECT 1 FROM user_roles
      WHERE user_id::text = current_setting('app.user_id', true)
      AND role IN ('admin', 'editor')
    )
  );
```

### Pattern 4: Hierarchical Access

**Use Case**: Managers see their team's data

```sql
-- Team hierarchy
CREATE TABLE team_members (
  user_id UUID REFERENCES users(id),
  team_id UUID,
  manager_id UUID REFERENCES users(id)
);

-- Recursive CTE for team hierarchy
CREATE OR REPLACE FUNCTION get_team_members(manager UUID)
RETURNS TABLE (user_id UUID) AS $$
  WITH RECURSIVE team_tree AS (
    -- Base case: direct reports
    SELECT user_id FROM team_members WHERE manager_id = manager
    UNION
    -- Recursive case: reports of reports
    SELECT tm.user_id FROM team_members tm
    INNER JOIN team_tree tt ON tm.manager_id = tt.user_id
  )
  SELECT user_id FROM team_tree;
$$ LANGUAGE sql STABLE;

-- Policy: See own data + team data
CREATE POLICY "hierarchical_access"
  ON posts
  FOR SELECT
  USING (
    user_id::text = current_setting('app.user_id', true) OR
    user_id IN (
      SELECT user_id FROM get_team_members(
        current_setting('app.user_id', true)::uuid
      )
    )
  );
```

### Pattern 5: Time-Based Access

**Use Case**: Scheduled content (embargo dates)

```sql
ALTER TABLE posts ADD COLUMN publish_at TIMESTAMP WITH TIME ZONE;

CREATE POLICY "time_based_access"
  ON posts
  FOR SELECT
  USING (
    published = true AND
    (publish_at IS NULL OR publish_at <= NOW()) AND
    user_id::text = current_setting('app.user_id', true)
  );
```

### Pattern 6: Shared Resources

**Use Case**: Documents shared between users

```sql
-- Sharing table
CREATE TABLE document_shares (
  document_id UUID REFERENCES documents(id),
  shared_with_user_id UUID REFERENCES users(id),
  permission TEXT DEFAULT 'view',  -- 'view', 'edit', 'admin'
  PRIMARY KEY (document_id, shared_with_user_id)
);

-- Policy: Owner + shared users can access
CREATE POLICY "shared_document_access"
  ON documents
  FOR SELECT
  USING (
    owner_id::text = current_setting('app.user_id', true) OR
    id IN (
      SELECT document_id FROM document_shares
      WHERE shared_with_user_id::text = current_setting('app.user_id', true)
    )
  );

-- Policy: Only owner + users with 'edit' can modify
CREATE POLICY "shared_document_edit"
  ON documents
  FOR UPDATE
  USING (
    owner_id::text = current_setting('app.user_id', true) OR
    id IN (
      SELECT document_id FROM document_shares
      WHERE shared_with_user_id::text = current_setting('app.user_id', true)
      AND permission IN ('edit', 'admin')
    )
  );
```

### RLS Performance Tips

1. **Index session variable columns**:
```sql
CREATE INDEX idx_posts_user_id ON posts(user_id);
```

2. **Use SECURITY DEFINER functions for complex checks**:
```sql
CREATE OR REPLACE FUNCTION can_access_post(post_id UUID)
RETURNS BOOLEAN AS $$
  -- Complex access logic here
$$ LANGUAGE sql STABLE SECURITY DEFINER;

CREATE POLICY "complex_access"
  ON posts FOR SELECT
  USING (can_access_post(id));
```

3. **Avoid expensive operations in RLS**:
```sql
-- Bad: Full table scan in policy
CREATE POLICY "bad_policy"
  ON posts FOR SELECT
  USING (id IN (SELECT post_id FROM slow_table));

-- Good: Join with indexed column
CREATE POLICY "good_policy"
  ON posts FOR SELECT
  USING (
    user_id::text = current_setting('app.user_id', true) OR
    EXISTS (
      SELECT 1 FROM shares
      WHERE post_id = posts.id
      AND user_id::text = current_setting('app.user_id', true)
    )
  );
```

---

## Performance Optimization

### Database Optimization

#### 1. Connection Pooling

**fluxbase.yaml**:
```yaml
database:
  max_connections: 100  # Increase for high traffic
  min_connections: 20   # Keep warm connections
  max_idle_time: 30m    # Close idle after 30min
  max_lifetime: 1h      # Recycle connections hourly
```

#### 2. Query Optimization

**Add Indexes**:
```sql
-- Index frequently queried columns
CREATE INDEX idx_posts_created_at ON posts(created_at DESC);
CREATE INDEX idx_posts_user_published ON posts(user_id, published);

-- Partial indexes for common filters
CREATE INDEX idx_published_posts ON posts(created_at DESC)
  WHERE published = true;

-- Composite indexes for multi-column queries
CREATE INDEX idx_posts_category_date ON posts(category_id, created_at DESC);

-- Full-text search indexes
CREATE INDEX idx_posts_search ON posts USING gin(to_tsvector('english', title || ' ' || content));
```

**Analyze Slow Queries**:
```sql
-- Enable query logging
ALTER SYSTEM SET log_min_duration_statement = 1000;  -- Log queries > 1s
SELECT pg_reload_conf();

-- Check slow queries
SELECT
  query,
  calls,
  mean_exec_time,
  max_exec_time
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 20;

-- Explain query plans
EXPLAIN ANALYZE
SELECT * FROM posts WHERE user_id = 'uuid' AND published = true;
```

#### 3. Caching Strategy

**Application-Level Cache**:
```typescript
import { useQuery } from '@tanstack/react-query'

const { data } = useQuery({
  queryKey: ['posts', filters],
  queryFn: () => fluxbase.from('posts').select('*'),
  staleTime: 60000,  // Cache for 1 minute
  gcTime: 300000,    // Keep in cache for 5 minutes
})
```

**Database-Level Cache**:
```sql
-- Materialized views for expensive queries
CREATE MATERIALIZED VIEW popular_posts AS
SELECT
  p.*,
  COUNT(l.id) AS like_count,
  COUNT(c.id) AS comment_count
FROM posts p
LEFT JOIN likes l ON l.post_id = p.id
LEFT JOIN comments c ON c.post_id = p.id
GROUP BY p.id
ORDER BY like_count DESC
LIMIT 100;

-- Refresh periodically (cron job or trigger)
REFRESH MATERIALIZED VIEW CONCURRENTLY popular_posts;
```

#### 4. N+1 Query Prevention

**Bad** (N+1 queries):
```typescript
const { data: posts } = await fluxbase.from('posts').select('*')

// N queries for authors
for (const post of posts) {
  const { data: author } = await fluxbase
    .from('users')
    .select('*')
    .eq('id', post.author_id)
    .single()
}
```

**Good** (1 query with join):
```typescript
const { data: posts } = await fluxbase
  .from('posts')
  .select(`
    *,
    author:users!author_id (
      id,
      name,
      avatar_url
    )
  `)
```

### API Response Optimization

#### 1. Pagination

```typescript
const PAGE_SIZE = 20

const { data, count } = await fluxbase
  .from('posts')
  .select('*', { count: 'exact' })
  .range(page * PAGE_SIZE, (page + 1) * PAGE_SIZE - 1)
```

#### 2. Field Selection

```typescript
// Bad: Return all columns (large payload)
await fluxbase.from('posts').select('*')

// Good: Select only needed fields
await fluxbase.from('posts').select('id, title, excerpt, created_at')
```

#### 3. Compression

**fluxbase.yaml**:
```yaml
server:
  enable_compression: true
  compression_level: 6  # 1-9, higher = better compression, more CPU
```

### Realtime Optimization

#### 1. Channel Filtering

```typescript
// Bad: Subscribe to all changes, filter client-side
fluxbase
  .from('posts')
  .on('*', (payload) => {
    if (payload.record.user_id === currentUserId) {
      // Handle
    }
  })
  .subscribe()

// Good: Filter server-side
fluxbase
  .from('posts')
  .on('*', (payload) => {
    // Only relevant changes
  })
  .eq('user_id', currentUserId)
  .subscribe()
```

#### 2. Batch Updates

```typescript
// Bad: Multiple realtime events
for (const post of posts) {
  await fluxbase.from('posts').update({ published: true }).eq('id', post.id)
  // Triggers N realtime events
}

// Good: Batch update (single event)
await fluxbase
  .from('posts')
  .update({ published: true })
  .in('id', posts.map(p => p.id))
// Triggers 1 realtime event
```

---

## Scaling Strategies

### Horizontal Scaling

#### 1. Application Layer

**Load Balancer** (nginx):

```nginx
upstream fluxbase_backend {
  least_conn;  # Balance by active connections
  server fluxbase1:8080 max_fails=3 fail_timeout=30s;
  server fluxbase2:8080 max_fails=3 fail_timeout=30s;
  server fluxbase3:8080 max_fails=3 fail_timeout=30s;
}

server {
  listen 80;
  server_name api.yourdomain.com;

  location / {
    proxy_pass http://fluxbase_backend;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";  # For WebSocket
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
  }
}
```

**Session Affinity** (for WebSocket):

```nginx
upstream fluxbase_backend {
  ip_hash;  # Same client → same server
  server fluxbase1:8080;
  server fluxbase2:8080;
  server fluxbase3:8080;
}
```

#### 2. Database Layer

**Read Replicas**:

```yaml
# fluxbase.yaml
database:
  primary:
    host: primary-db.yourdomain.com
    port: 5432
  replicas:
    - host: replica1-db.yourdomain.com
      port: 5432
    - host: replica2-db.yourdomain.com
      port: 5432
  read_preference: replica  # Route reads to replicas
```

**Connection Pooler** (PgBouncer):

```ini
# pgbouncer.ini
[databases]
fluxbase = host=postgres port=5432 dbname=fluxbase

[pgbouncer]
listen_addr = 0.0.0.0
listen_port = 6432
auth_type = md5
auth_file = /etc/pgbouncer/userlist.txt
pool_mode = transaction
max_client_conn = 10000
default_pool_size = 25
reserve_pool_size = 10
```

Update Fluxbase config:
```yaml
database:
  host: pgbouncer.yourdomain.com
  port: 6432
```

### Vertical Scaling

**When to Scale Up**:
- CPU usage consistently > 70%
- Memory usage > 80%
- Database connections saturated
- High query latency

**Scaling Guidelines**:

| Tier | vCPU | RAM | Connections | Throughput |
|------|------|-----|-------------|------------|
| Small | 2 | 4GB | 50 | 2K req/s |
| Medium | 4 | 8GB | 100 | 5K req/s |
| Large | 8 | 16GB | 200 | 10K req/s |
| XLarge | 16 | 32GB | 400 | 20K+ req/s |

### Caching Layer

**Redis for Sessions**:

```yaml
# fluxbase.yaml
redis:
  host: redis.yourdomain.com
  port: 6379
  db: 0
  password: ${REDIS_PASSWORD}

cache:
  enabled: true
  ttl: 3600  # 1 hour default
  max_size: 1000MB
```

**Application Code**:

```typescript
// Check cache first
const cached = await redis.get(`post:${id}`)
if (cached) {
  return JSON.parse(cached)
}

// Query database
const { data } = await fluxbase
  .from('posts')
  .select('*')
  .eq('id', id)
  .single()

// Cache result
await redis.setex(`post:${id}`, 3600, JSON.stringify(data))
```

### Content Delivery Network (CDN)

**For Storage (images, files)**:

```yaml
# fluxbase.yaml
storage:
  provider: s3
  s3_bucket: your-bucket
  s3_region: us-east-1
  cdn_url: https://cdn.yourdomain.com  # CloudFront, Cloudflare, etc.
```

**Client Code**:

```typescript
// Automatically uses CDN URL
const { publicURL } = fluxbase.storage
  .from('avatars')
  .getPublicUrl('user-123/avatar.jpg')

// Returns: https://cdn.yourdomain.com/avatars/user-123/avatar.jpg
```

---

## Monitoring & Observability

### Metrics Collection

**Prometheus** (already integrated):

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'fluxbase'
    static_configs:
      - targets: ['fluxbase:9090']
```

**Key Metrics to Track**:

```promql
# Request rate
rate(http_requests_total[5m])

# Error rate
rate(http_requests_total{status=~"5.."}[5m])

# P95 latency
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# Database connection pool usage
database_connections_active / database_connections_max

# WebSocket connections
websocket_connections_active

# Memory usage
process_resident_memory_bytes / node_memory_total_bytes

# CPU usage
rate(process_cpu_seconds_total[5m])
```

### Alerting

**Alertmanager Rules**:

```yaml
groups:
  - name: fluxbase
    rules:
      # High error rate
      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value }}% over the last 5 minutes"

      # High latency
      - alert: HighLatency
        expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "P95 latency is high"
          description: "P95 latency is {{ $value }}s"

      # Database connections saturated
      - alert: DatabaseConnectionsSaturated
        expr: database_connections_active / database_connections_max > 0.9
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Database connection pool nearly full"
```

### Logging

**Structured Logging** (already integrated):

```json
{
  "timestamp": "2025-10-30T12:34:56Z",
  "level": "info",
  "message": "Request completed",
  "method": "GET",
  "path": "/api/v1/rest/posts",
  "status": 200,
  "duration_ms": 45,
  "user_id": "uuid",
  "request_id": "req-123"
}
```

**Log Aggregation** (Loki, Elasticsearch):

```yaml
# promtail-config.yml
clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  - job_name: fluxbase
    static_configs:
      - targets:
          - localhost
        labels:
          job: fluxbase
          __path__: /var/log/fluxbase/*.log
```

### Distributed Tracing

**Jaeger Integration**:

```yaml
# fluxbase.yaml
observability:
  tracing:
    enabled: true
    endpoint: http://jaeger:14268/api/traces
    service_name: fluxbase
    sample_rate: 0.1  # 10% of requests
```

**Trace Context Propagation**:

```typescript
// Automatically propagated via headers
const { data } = await fluxbase
  .from('posts')
  .select('*')
// Trace ID: span-id-123
//   → Database Query: span-id-124
//   → Cache Lookup: span-id-125
```

---

## Security Hardening

### API Security

**Rate Limiting** (already configured):

```yaml
security:
  rate_limit:
    enabled: true
    anonymous: 100  # 100 req/min for anonymous
    authenticated: 500  # 500 req/min for authenticated
    api_key: 1000  # 1000 req/min for API keys
```

**CORS Configuration**:

```yaml
server:
  cors:
    allowed_origins:
      - https://yourdomain.com
      - https://app.yourdomain.com
    allowed_methods:
      - GET
      - POST
      - PUT
      - PATCH
      - DELETE
    allowed_headers:
      - Authorization
      - Content-Type
    expose_headers:
      - X-Total-Count
    max_age: 3600
```

**Content Security Policy**:

```yaml
security:
  headers:
    content_security_policy: "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:;"
    x_frame_options: "DENY"
    x_content_type_options: "nosniff"
```

### Database Security

**SSL/TLS Connections**:

```yaml
database:
  ssl_mode: require  # Or 'verify-full' for cert validation
  ssl_cert: /path/to/client-cert.pem
  ssl_key: /path/to/client-key.pem
  ssl_root_cert: /path/to/ca-cert.pem
```

**Least Privilege**:

```sql
-- Create read-only user for replicas
CREATE USER readonly_user WITH PASSWORD 'secure_password';
GRANT CONNECT ON DATABASE fluxbase TO readonly_user;
GRANT USAGE ON SCHEMA public TO readonly_user;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO readonly_user;

-- Create app user with limited permissions
CREATE USER app_user WITH PASSWORD 'secure_password';
GRANT CONNECT ON DATABASE fluxbase TO app_user;
GRANT USAGE ON SCHEMA public TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO app_user;
```

### Secrets Management

**HashiCorp Vault**:

```yaml
# fluxbase.yaml
secrets:
  provider: vault
  vault:
    address: https://vault.yourdomain.com
    token: ${VAULT_TOKEN}
    path: secret/fluxbase

database:
  password: vault://secret/fluxbase/db_password
auth:
  jwt_secret: vault://secret/fluxbase/jwt_secret
```

**AWS Secrets Manager**:

```yaml
secrets:
  provider: aws_secrets_manager
  aws:
    region: us-east-1

database:
  password: aws://fluxbase/db_password
```

---

## Disaster Recovery

### Backup Strategy

**Automated Backups**:

```bash
#!/bin/bash
# backup.sh

DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backups"
DB_NAME="fluxbase"

# Full backup
pg_dump -h $DB_HOST -U $DB_USER -d $DB_NAME -F c -f "$BACKUP_DIR/full_$DATE.dump"

# Incremental via WAL archiving (PostgreSQL)
# Configure in postgresql.conf:
# wal_level = replica
# archive_mode = on
# archive_command = 'aws s3 cp %p s3://backups/wal/%f'

# Retention: Keep last 7 days
find $BACKUP_DIR -name "full_*.dump" -mtime +7 -delete

# Upload to S3
aws s3 cp "$BACKUP_DIR/full_$DATE.dump" s3://fluxbase-backups/
```

**Cron Schedule**:

```cron
# Daily full backup at 2 AM
0 2 * * * /opt/fluxbase/backup.sh

# Hourly incremental (WAL archiving handles this)
```

### Point-in-Time Recovery

```bash
#!/bin/bash
# restore.sh

RESTORE_TIME="2025-10-30 12:00:00"
BACKUP_FILE="/backups/full_20251030_000000.dump"

# Stop Fluxbase
systemctl stop fluxbase

# Restore base backup
pg_restore -h localhost -U postgres -d fluxbase -c $BACKUP_FILE

# Restore WAL files up to recovery point
# Configure recovery.conf:
cat > /var/lib/postgresql/data/recovery.conf <<EOF
restore_command = 'aws s3 cp s3://backups/wal/%f %p'
recovery_target_time = '$RESTORE_TIME'
EOF

# Start PostgreSQL (enters recovery mode)
systemctl start postgresql

# Wait for recovery to complete
# Check logs: tail -f /var/log/postgresql/postgresql.log

# Start Fluxbase
systemctl start fluxbase
```

### High Availability

**PostgreSQL Replication**:

```bash
# Primary server postgresql.conf
wal_level = replica
max_wal_senders = 10
wal_keep_size = 1GB

# Standby server postgresql.conf
hot_standby = on
primary_conninfo = 'host=primary port=5432 user=replicator password=xxx'
```

**Automatic Failover** (Patroni):

```yaml
# patroni.yml
scope: fluxbase-cluster
name: node1

restapi:
  listen: 0.0.0.0:8008
  connect_address: node1:8008

etcd:
  hosts: etcd1:2379,etcd2:2379,etcd3:2379

bootstrap:
  dcs:
    ttl: 30
    loop_wait: 10
    retry_timeout: 10
    maximum_lag_on_failover: 1048576

postgresql:
  listen: 0.0.0.0:5432
  connect_address: node1:5432
  data_dir: /var/lib/postgresql/data
  authentication:
    replication:
      username: replicator
      password: secure_password
```

---

## Multi-Tenancy

### Schema-Based Isolation

Each tenant gets their own schema:

```sql
-- Create tenant schema
CREATE SCHEMA tenant_acme;

-- Create tables
CREATE TABLE tenant_acme.posts (
  id UUID PRIMARY KEY,
  title TEXT
);

-- Grant access
GRANT USAGE ON SCHEMA tenant_acme TO tenant_user;
```

**Dynamic Schema Selection**:

```typescript
// Middleware to set search_path
app.use((req, res, next) => {
  const tenantId = req.headers['x-tenant-id']
  const schema = `tenant_${tenantId}`

  // Set schema for this request
  req.db.query(`SET search_path TO ${schema}`)
  next()
})
```

### Database-Based Isolation

Each tenant gets their own database:

```yaml
# fluxbase.yaml
tenants:
  - name: acme
    database:
      host: localhost
      name: fluxbase_acme
  - name: foo
    database:
      host: localhost
      name: fluxbase_foo
```

### RLS-Based Isolation

Single database, single schema, isolated via RLS:

```sql
-- Add tenant_id to all tables
ALTER TABLE posts ADD COLUMN tenant_id UUID;

-- RLS policy
CREATE POLICY "tenant_isolation"
  ON posts
  FOR ALL
  USING (tenant_id::text = current_setting('app.tenant_id', true));
```

---

## Edge Functions Best Practices

### Performance

**Cold Start Optimization**:

```typescript
// Keep functions warm with pre-initialized resources
const db = initDatabase()  // Reused across invocations

function handler(request) {
  // Use pre-initialized db
  const result = db.query(...)
  return { status: 200, body: JSON.stringify(result) }
}
```

**Async Operations**:

```typescript
// Good: Run operations in parallel
async function handler(request) {
  const [users, posts, comments] = await Promise.all([
    fetchUsers(),
    fetchPosts(),
    fetchComments()
  ])

  return { status: 200, body: JSON.stringify({ users, posts, comments }) }
}
```

### Error Handling

```typescript
async function handler(request) {
  try {
    const result = await riskyOperation()
    return { status: 200, body: JSON.stringify(result) }
  } catch (error) {
    console.error('Function error:', error)

    if (error.code === 'NOT_FOUND') {
      return { status: 404, body: JSON.stringify({ error: 'Not found' }) }
    }

    // Don't leak internal errors
    return { status: 500, body: JSON.stringify({ error: 'Internal server error' }) }
  }
}
```

### Authentication

```typescript
function handler(request) {
  const userId = request.user_id

  if (!userId) {
    return { status: 401, body: JSON.stringify({ error: 'Unauthorized' }) }
  }

  // Proceed with authenticated logic
  const data = fetchUserData(userId)
  return { status: 200, body: JSON.stringify(data) }
}
```

---

## 📚 Additional Resources

- [PostgreSQL Performance](https://www.postgresql.org/docs/current/performance-tips.html)
- [Prometheus Best Practices](https://prometheus.io/docs/practices/)
- [Kubernetes Scaling](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/)
- [AWS RDS Best Practices](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/CHAP_BestPractices.html)

---

**Last Updated**: 2025-10-30
**Status**: Complete ✅
**Coverage**: Production-ready patterns for scaling to 100K+ users
