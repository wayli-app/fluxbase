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

## Custom Database Functions

PostgreSQL functions allow you to encapsulate complex business logic in the database, improving performance and maintainability.

### Creating PostgreSQL Functions

**Basic Function Example**:

```sql
-- Function to calculate user statistics
CREATE OR REPLACE FUNCTION calculate_user_stats(user_uuid UUID)
RETURNS TABLE(
  total_posts INTEGER,
  total_comments INTEGER,
  account_age_days INTEGER
) AS $$
BEGIN
  RETURN QUERY
  SELECT
    (SELECT COUNT(*)::INTEGER FROM posts WHERE user_id = user_uuid),
    (SELECT COUNT(*)::INTEGER FROM comments WHERE user_id = user_uuid),
    (SELECT EXTRACT(DAY FROM NOW() - created_at)::INTEGER FROM users WHERE id = user_uuid);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;
```

**Calling from SDK**:

```typescript
const { data, error } = await fluxbase
  .rpc('calculate_user_stats', { user_uuid: userId })

console.log('User stats:', data)
// { total_posts: 42, total_comments: 156, account_age_days: 365 }
```

### Function Security

**SECURITY DEFINER vs SECURITY INVOKER**:

```sql
-- SECURITY DEFINER: Runs with function owner's privileges (use with caution!)
CREATE FUNCTION admin_only_operation()
RETURNS VOID AS $$
BEGIN
  -- Bypasses RLS, runs as function owner
  DELETE FROM sensitive_data WHERE expired = true;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- SECURITY INVOKER: Runs with caller's privileges (safer, respects RLS)
CREATE FUNCTION get_my_data()
RETURNS SETOF posts AS $$
BEGIN
  -- Respects RLS policies
  RETURN QUERY SELECT * FROM posts;
END;
$$ LANGUAGE plpgsql SECURITY INVOKER;
```

**Best Practice**: Always use `SECURITY INVOKER` unless you specifically need to bypass RLS.

### Advanced Function Examples

**1. Aggregation Function**:

```sql
CREATE OR REPLACE FUNCTION get_monthly_revenue(org_uuid UUID, year INTEGER)
RETURNS TABLE(
  month INTEGER,
  revenue NUMERIC,
  transaction_count INTEGER
) AS $$
BEGIN
  RETURN QUERY
  SELECT
    EXTRACT(MONTH FROM created_at)::INTEGER AS month,
    SUM(amount) AS revenue,
    COUNT(*)::INTEGER AS transaction_count
  FROM transactions
  WHERE
    organization_id = org_uuid
    AND EXTRACT(YEAR FROM created_at) = year
  GROUP BY EXTRACT(MONTH FROM created_at)
  ORDER BY month;
END;
$$ LANGUAGE plpgsql SECURITY INVOKER;
```

**2. Data Validation Function**:

```sql
CREATE OR REPLACE FUNCTION validate_and_create_order(
  p_user_id UUID,
  p_product_id UUID,
  p_quantity INTEGER
) RETURNS TABLE(
  success BOOLEAN,
  order_id UUID,
  message TEXT
) AS $$
DECLARE
  v_stock INTEGER;
  v_price NUMERIC;
  v_order_id UUID;
BEGIN
  -- Check stock
  SELECT stock_quantity, price INTO v_stock, v_price
  FROM products
  WHERE id = p_product_id;

  IF v_stock < p_quantity THEN
    RETURN QUERY SELECT false, NULL::UUID, 'Insufficient stock'::TEXT;
    RETURN;
  END IF;

  -- Create order
  INSERT INTO orders (user_id, product_id, quantity, total_price)
  VALUES (p_user_id, p_product_id, p_quantity, v_price * p_quantity)
  RETURNING id INTO v_order_id;

  -- Update stock
  UPDATE products
  SET stock_quantity = stock_quantity - p_quantity
  WHERE id = p_product_id;

  RETURN QUERY SELECT true, v_order_id, 'Order created successfully'::TEXT;
END;
$$ LANGUAGE plpgsql SECURITY INVOKER;
```

**Usage**:

```typescript
const { data } = await fluxbase.rpc('validate_and_create_order', {
  p_user_id: userId,
  p_product_id: productId,
  p_quantity: 2
})

if (data[0].success) {
  console.log('Order ID:', data[0].order_id)
} else {
  console.error('Error:', data[0].message)
}
```

**3. Trigger Function for Audit Logging**:

```sql
-- Create audit log table
CREATE TABLE audit_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  table_name TEXT NOT NULL,
  operation TEXT NOT NULL,
  old_data JSONB,
  new_data JSONB,
  user_id UUID,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Audit function
CREATE OR REPLACE FUNCTION audit_trigger()
RETURNS TRIGGER AS $$
DECLARE
  v_user_id TEXT;
BEGIN
  -- Get current user from session
  v_user_id := current_setting('app.user_id', true);

  IF (TG_OP = 'DELETE') THEN
    INSERT INTO audit_logs (table_name, operation, old_data, user_id)
    VALUES (TG_TABLE_NAME, TG_OP, row_to_json(OLD), v_user_id::UUID);
    RETURN OLD;
  ELSIF (TG_OP = 'UPDATE') THEN
    INSERT INTO audit_logs (table_name, operation, old_data, new_data, user_id)
    VALUES (TG_TABLE_NAME, TG_OP, row_to_json(OLD), row_to_json(NEW), v_user_id::UUID);
    RETURN NEW;
  ELSIF (TG_OP = 'INSERT') THEN
    INSERT INTO audit_logs (table_name, operation, new_data, user_id)
    VALUES (TG_TABLE_NAME, TG_OP, row_to_json(NEW), v_user_id::UUID);
    RETURN NEW;
  END IF;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Attach trigger to table
CREATE TRIGGER posts_audit
  AFTER INSERT OR UPDATE OR DELETE ON posts
  FOR EACH ROW EXECUTE FUNCTION audit_trigger();
```

### Performance Optimization for Functions

**1. Use IMMUTABLE When Possible**:

```sql
-- Immutable function (result only depends on input)
CREATE FUNCTION calculate_discount(price NUMERIC, percent INTEGER)
RETURNS NUMERIC AS $$
  SELECT price * (1 - percent / 100.0);
$$ LANGUAGE SQL IMMUTABLE;

-- PostgreSQL can cache result for same inputs
```

**2. Use PARALLEL SAFE for Large Datasets**:

```sql
CREATE FUNCTION expensive_calculation(input INTEGER)
RETURNS INTEGER AS $$
  -- Complex calculation
  SELECT input * input + input;
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;
```

**3. Return SETOF for Batch Operations**:

```sql
-- Efficient: Returns multiple rows in one call
CREATE FUNCTION get_user_feed(user_uuid UUID, page_size INTEGER)
RETURNS SETOF posts AS $$
  SELECT p.*
  FROM posts p
  JOIN followers f ON f.following_id = p.user_id
  WHERE f.follower_id = user_uuid
  ORDER BY p.created_at DESC
  LIMIT page_size;
$$ LANGUAGE SQL SECURITY INVOKER;
```

### Error Handling in Functions

```sql
CREATE OR REPLACE FUNCTION safe_transfer(
  from_account UUID,
  to_account UUID,
  amount NUMERIC
) RETURNS TABLE(success BOOLEAN, message TEXT) AS $$
DECLARE
  v_balance NUMERIC;
BEGIN
  -- Check balance
  SELECT balance INTO v_balance
  FROM accounts
  WHERE id = from_account
  FOR UPDATE; -- Lock row

  IF v_balance < amount THEN
    RETURN QUERY SELECT false, 'Insufficient funds'::TEXT;
    RETURN;
  END IF;

  -- Perform transfer
  UPDATE accounts SET balance = balance - amount WHERE id = from_account;
  UPDATE accounts SET balance = balance + amount WHERE id = to_account;

  RETURN QUERY SELECT true, 'Transfer successful'::TEXT;

EXCEPTION
  WHEN OTHERS THEN
    RETURN QUERY SELECT false, SQLERRM::TEXT;
END;
$$ LANGUAGE plpgsql SECURITY INVOKER;
```

### Testing Functions

```sql
-- Create test function
CREATE OR REPLACE FUNCTION test_calculate_user_stats()
RETURNS VOID AS $$
DECLARE
  v_result RECORD;
  v_test_user UUID := gen_random_uuid();
BEGIN
  -- Setup test data
  INSERT INTO users (id, email, created_at)
  VALUES (v_test_user, 'test@example.com', NOW() - INTERVAL '30 days');

  INSERT INTO posts (user_id, title)
  SELECT v_test_user, 'Post ' || i
  FROM generate_series(1, 5) i;

  -- Run function
  SELECT * INTO v_result FROM calculate_user_stats(v_test_user);

  -- Assert results
  IF v_result.total_posts != 5 THEN
    RAISE EXCEPTION 'Expected 5 posts, got %', v_result.total_posts;
  END IF;

  IF v_result.account_age_days != 30 THEN
    RAISE EXCEPTION 'Expected 30 days, got %', v_result.account_age_days;
  END IF;

  -- Cleanup
  DELETE FROM posts WHERE user_id = v_test_user;
  DELETE FROM users WHERE id = v_test_user;

  RAISE NOTICE 'All tests passed!';
END;
$$ LANGUAGE plpgsql;

-- Run test
SELECT test_calculate_user_stats();
```

---

## Advanced Queries

Master complex PostgreSQL queries for sophisticated data operations.

### Complex JOINs

**Multiple JOINs with Filtering**:

```typescript
// Get users with their posts and comment counts
const { data } = await fluxbase
  .from('users')
  .select(`
    id,
    email,
    posts:posts(
      id,
      title,
      created_at,
      comments:comments(count)
    )
  `)
  .eq('posts.published', true)
  .order('posts.created_at', { ascending: false })
  .limit(10)
```

**Lateral JOIN (PostgreSQL 9.3+)**:

```sql
-- Get each user's 3 most recent posts
SELECT u.email, recent.title, recent.created_at
FROM users u
CROSS JOIN LATERAL (
  SELECT title, created_at
  FROM posts p
  WHERE p.user_id = u.id
  ORDER BY created_at DESC
  LIMIT 3
) recent;
```

### Aggregations

**Basic Aggregation**:

```typescript
// Count posts by category
const { data } = await fluxbase
  .from('posts')
  .select('category, count:id.count()')
  .groupBy('category')
```

**Complex Aggregation with Filters**:

```sql
SELECT
  DATE_TRUNC('day', created_at) AS date,
  COUNT(*) FILTER (WHERE status = 'published') AS published_count,
  COUNT(*) FILTER (WHERE status = 'draft') AS draft_count,
  AVG(view_count) AS avg_views,
  PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY view_count) AS median_views
FROM posts
WHERE created_at >= NOW() - INTERVAL '30 days'
GROUP BY DATE_TRUNC('day', created_at)
ORDER BY date;
```

**Grouping Sets**:

```sql
-- Multiple aggregation levels in one query
SELECT
  category,
  author_id,
  COUNT(*) AS post_count,
  SUM(view_count) AS total_views
FROM posts
GROUP BY GROUPING SETS (
  (category),              -- By category
  (author_id),            -- By author
  (category, author_id),  -- By both
  ()                      -- Grand total
)
ORDER BY category, author_id;
```

### Full-Text Search

**Basic Text Search**:

```typescript
// Simple text search
const { data } = await fluxbase
  .from('posts')
  .select('*')
  .textSearch('title', 'postgresql', { type: 'plain' })
```

**Advanced Full-Text Search**:

```sql
-- Create text search column
ALTER TABLE posts ADD COLUMN search_vector tsvector;

-- Update index with triggers
CREATE OR REPLACE FUNCTION posts_search_trigger()
RETURNS TRIGGER AS $$
BEGIN
  NEW.search_vector :=
    setweight(to_tsvector('english', COALESCE(NEW.title, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(NEW.content, '')), 'B') ||
    setweight(to_tsvector('english', COALESCE(NEW.tags, '')), 'C');
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER posts_search_update
  BEFORE INSERT OR UPDATE ON posts
  FOR EACH ROW EXECUTE FUNCTION posts_search_trigger();

-- Create GIN index
CREATE INDEX posts_search_idx ON posts USING GIN(search_vector);

-- Query with ranking
SELECT
  id,
  title,
  ts_rank(search_vector, query) AS rank
FROM posts,
     to_tsquery('english', 'postgresql & performance') query
WHERE search_vector @@ query
ORDER BY rank DESC
LIMIT 20;
```

**Fuzzy Search with Trigrams**:

```sql
-- Enable pg_trgm extension
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Create index
CREATE INDEX posts_title_trgm_idx ON posts USING GIN (title gin_trgm_ops);

-- Fuzzy search
SELECT
  title,
  similarity(title, 'posgresql') AS score
FROM posts
WHERE title % 'posgresql'  -- % operator for similarity
ORDER BY score DESC
LIMIT 10;
```

### JSON Operations

**Querying JSONB Data**:

```typescript
// Query nested JSON
const { data } = await fluxbase
  .from('products')
  .select('*')
  .contains('metadata', { featured: true })
  .gte('metadata->price', 100)
```

**Complex JSON Queries**:

```sql
-- Extract and filter JSON data
SELECT
  id,
  name,
  metadata->>'category' AS category,
  (metadata->>'price')::NUMERIC AS price,
  jsonb_array_elements_text(metadata->'tags') AS tag
FROM products
WHERE
  metadata @> '{"featured": true}'
  AND (metadata->>'price')::NUMERIC < 1000;

-- JSON aggregation
SELECT
  metadata->>'category' AS category,
  jsonb_agg(
    jsonb_build_object(
      'id', id,
      'name', name,
      'price', metadata->'price'
    )
  ) AS products
FROM products
GROUP BY metadata->>'category';
```

### Window Functions

**Ranking and Row Numbers**:

```sql
-- Rank posts within each category
SELECT
  category,
  title,
  view_count,
  RANK() OVER (PARTITION BY category ORDER BY view_count DESC) AS rank,
  DENSE_RANK() OVER (PARTITION BY category ORDER BY view_count DESC) AS dense_rank,
  ROW_NUMBER() OVER (PARTITION BY category ORDER BY view_count DESC) AS row_num
FROM posts;
```

**Running Totals and Moving Averages**:

```sql
-- Calculate cumulative revenue
SELECT
  DATE_TRUNC('day', created_at) AS date,
  amount,
  SUM(amount) OVER (ORDER BY created_at) AS cumulative_total,
  AVG(amount) OVER (
    ORDER BY created_at
    ROWS BETWEEN 6 PRECEDING AND CURRENT ROW
  ) AS moving_avg_7_days
FROM transactions
ORDER BY created_at;
```

**Lead and Lag**:

```sql
-- Compare with previous/next values
SELECT
  user_id,
  login_at,
  LAG(login_at) OVER (PARTITION BY user_id ORDER BY login_at) AS previous_login,
  LEAD(login_at) OVER (PARTITION BY user_id ORDER BY login_at) AS next_login,
  login_at - LAG(login_at) OVER (PARTITION BY user_id ORDER BY login_at) AS time_since_last_login
FROM user_logins
ORDER BY user_id, login_at;
```

### Common Table Expressions (CTEs)

**Recursive CTE for Hierarchical Data**:

```sql
-- Get all descendants of a category
WITH RECURSIVE category_tree AS (
  -- Base case: start with parent category
  SELECT id, name, parent_id, 0 AS level
  FROM categories
  WHERE id = 'root-category-uuid'

  UNION ALL

  -- Recursive case: get children
  SELECT c.id, c.name, c.parent_id, ct.level + 1
  FROM categories c
  JOIN category_tree ct ON c.parent_id = ct.id
)
SELECT * FROM category_tree
ORDER BY level, name;
```

**Multiple CTEs**:

```sql
WITH
  active_users AS (
    SELECT id, email
    FROM users
    WHERE last_login_at > NOW() - INTERVAL '30 days'
  ),
  popular_posts AS (
    SELECT user_id, COUNT(*) AS post_count
    FROM posts
    WHERE view_count > 100
    GROUP BY user_id
  )
SELECT
  au.email,
  COALESCE(pp.post_count, 0) AS popular_post_count
FROM active_users au
LEFT JOIN popular_posts pp ON au.id = pp.user_id
ORDER BY popular_post_count DESC;
```

### N+1 Query Prevention

**Problem - N+1 Queries**:

```typescript
// ❌ BAD: Makes 1 + N queries
const users = await fluxbase.from('users').select('*')

for (const user of users.data) {
  // Additional query for each user!
  const posts = await fluxbase
    .from('posts')
    .select('*')
    .eq('user_id', user.id)
}
```

**Solution - JOIN or Nested Select**:

```typescript
// ✅ GOOD: Single query with JOIN
const { data } = await fluxbase
  .from('users')
  .select(`
    *,
    posts:posts(*)
  `)

// All data loaded in one query
data.forEach(user => {
  console.log(user.email, user.posts.length)
})
```

### Query Optimization Tips

**1. Use EXPLAIN ANALYZE**:

```sql
EXPLAIN ANALYZE
SELECT *
FROM posts p
JOIN users u ON p.user_id = u.id
WHERE p.created_at > NOW() - INTERVAL '7 days';
```

**2. Avoid SELECT ***:

```typescript
// ❌ BAD: Fetches all columns
const { data } = await fluxbase.from('posts').select('*')

// ✅ GOOD: Only fetch what you need
const { data } = await fluxbase
  .from('posts')
  .select('id, title, created_at')
```

**3. Use Appropriate Indexes**:

```sql
-- B-tree for equality and range queries
CREATE INDEX idx_posts_created_at ON posts(created_at);

-- Partial index for frequently queried subset
CREATE INDEX idx_published_posts ON posts(created_at)
WHERE status = 'published';

-- Composite index for multiple columns
CREATE INDEX idx_posts_user_status ON posts(user_id, status);
```

**4. Batch Operations**:

```typescript
// ❌ BAD: Multiple individual inserts
for (const item of items) {
  await fluxbase.from('posts').insert(item)
}

// ✅ GOOD: Single batch insert
await fluxbase.from('posts').insert(items)
```

---

## Schema Design Best Practices

Design scalable and maintainable database schemas.

### Normalization vs Denormalization

**Normalized Design** (Recommended for most cases):

```sql
-- Normalized: Separate tables, no duplication
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT UNIQUE NOT NULL,
  name TEXT NOT NULL
);

CREATE TABLE posts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  title TEXT NOT NULL,
  content TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Query requires JOIN
SELECT p.title, u.name
FROM posts p
JOIN users u ON p.user_id = u.id;
```

**When to Denormalize**:

```sql
-- Denormalized: Store username in posts for performance
CREATE TABLE posts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  user_name TEXT NOT NULL,  -- Denormalized!
  title TEXT NOT NULL,
  content TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Faster query (no JOIN needed)
SELECT title, user_name FROM posts;

-- Maintain consistency with trigger
CREATE OR REPLACE FUNCTION sync_user_name()
RETURNS TRIGGER AS $$
BEGIN
  IF (TG_OP = 'UPDATE' AND OLD.name != NEW.name) THEN
    UPDATE posts SET user_name = NEW.name WHERE user_id = NEW.id;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_name_update
  AFTER UPDATE ON users
  FOR EACH ROW EXECUTE FUNCTION sync_user_name();
```

**Denormalization Guidelines**:
- ✅ Read-heavy data (e.g., display names, counts)
- ✅ Data that rarely changes
- ✅ Significant performance gain needed
- ❌ Frequently updated data
- ❌ Critical data requiring strong consistency

### Foreign Keys and Relationships

**One-to-Many**:

```sql
CREATE TABLE organizations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL
);

CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  email TEXT UNIQUE NOT NULL
);
```

**Many-to-Many with Junction Table**:

```sql
CREATE TABLE posts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title TEXT NOT NULL
);

CREATE TABLE tags (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT UNIQUE NOT NULL
);

-- Junction table
CREATE TABLE post_tags (
  post_id UUID REFERENCES posts(id) ON DELETE CASCADE,
  tag_id UUID REFERENCES tags(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  PRIMARY KEY (post_id, tag_id)
);

-- Query: Get posts with tags
SELECT
  p.title,
  ARRAY_AGG(t.name) AS tags
FROM posts p
LEFT JOIN post_tags pt ON p.id = pt.post_id
LEFT JOIN tags t ON pt.tag_id = t.id
GROUP BY p.id, p.title;
```

**Self-Referential (Hierarchical)**:

```sql
CREATE TABLE categories (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  parent_id UUID REFERENCES categories(id),
  name TEXT NOT NULL,
  level INTEGER NOT NULL DEFAULT 0,
  path TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[]
);

-- Maintain hierarchy with trigger
CREATE OR REPLACE FUNCTION update_category_hierarchy()
RETURNS TRIGGER AS $$
BEGIN
  IF NEW.parent_id IS NULL THEN
    NEW.level := 0;
    NEW.path := ARRAY[NEW.id::TEXT];
  ELSE
    SELECT level + 1, path || NEW.id::TEXT
    INTO NEW.level, NEW.path
    FROM categories
    WHERE id = NEW.parent_id;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER category_hierarchy
  BEFORE INSERT OR UPDATE ON categories
  FOR EACH ROW EXECUTE FUNCTION update_category_hierarchy();
```

### Audit Columns

**Standard Audit Columns**:

```sql
CREATE TABLE posts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Business columns
  title TEXT NOT NULL,
  content TEXT,
  status TEXT NOT NULL DEFAULT 'draft',

  -- Audit columns (add to every table)
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_by UUID REFERENCES users(id),
  updated_by UUID REFERENCES users(id),

  -- Soft delete
  deleted_at TIMESTAMPTZ,
  deleted_by UUID REFERENCES users(id)
);

-- Auto-update updated_at
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at := NOW();
  NEW.updated_by := NULLIF(current_setting('app.user_id', true), '')::UUID;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER posts_updated_at
  BEFORE UPDATE ON posts
  FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Set created_by on insert
CREATE OR REPLACE FUNCTION set_created_by()
RETURNS TRIGGER AS $$
BEGIN
  NEW.created_by := NULLIF(current_setting('app.user_id', true), '')::UUID;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER posts_created_by
  BEFORE INSERT ON posts
  FOR EACH ROW EXECUTE FUNCTION set_created_by();
```

### Soft Deletes

**Implementation**:

```sql
-- Soft delete function
CREATE OR REPLACE FUNCTION soft_delete()
RETURNS TRIGGER AS $$
BEGIN
  -- Instead of deleting, update deleted_at
  UPDATE posts
  SET
    deleted_at = NOW(),
    deleted_by = NULLIF(current_setting('app.user_id', true), '')::UUID
  WHERE id = OLD.id;

  -- Prevent actual deletion
  RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER posts_soft_delete
  BEFORE DELETE ON posts
  FOR EACH ROW EXECUTE FUNCTION soft_delete();

-- View excluding soft-deleted records
CREATE VIEW active_posts AS
SELECT * FROM posts WHERE deleted_at IS NULL;

-- Grant access to view instead of table
GRANT SELECT, INSERT, UPDATE ON active_posts TO authenticated;
```

**Usage**:

```typescript
// Works automatically - "deleted" records still in table but hidden
await fluxbase.from('posts').delete().eq('id', postId)

// Query only shows non-deleted records
const { data } = await fluxbase.from('posts').select('*')

// Admin can query all including deleted
const { data } = await fluxbase
  .from('posts')
  .select('*')
  .is('deleted_at', null)  // null = not deleted
```

### Versioning Strategies

**Event Sourcing Approach**:

```sql
-- Current state table
CREATE TABLE documents (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title TEXT NOT NULL,
  content TEXT,
  version INTEGER NOT NULL DEFAULT 1,
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Version history table
CREATE TABLE document_versions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  document_id UUID NOT NULL REFERENCES documents(id),
  version INTEGER NOT NULL,
  title TEXT NOT NULL,
  content TEXT,
  changed_by UUID REFERENCES users(id),
  changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  change_description TEXT,
  UNIQUE(document_id, version)
);

-- Auto-version on update
CREATE OR REPLACE FUNCTION version_document()
RETURNS TRIGGER AS $$
BEGIN
  -- Save old version
  INSERT INTO document_versions (document_id, version, title, content, changed_by)
  VALUES (
    OLD.id,
    OLD.version,
    OLD.title,
    OLD.content,
    NULLIF(current_setting('app.user_id', true), '')::UUID
  );

  -- Increment version
  NEW.version := OLD.version + 1;
  NEW.updated_at := NOW();

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER document_versioning
  BEFORE UPDATE ON documents
  FOR EACH ROW EXECUTE FUNCTION version_document();
```

**Usage**:

```typescript
// Get current version
const { data: current } = await fluxbase
  .from('documents')
  .select('*')
  .eq('id', docId)
  .single()

// Get version history
const { data: history } = await fluxbase
  .from('document_versions')
  .select('*')
  .eq('document_id', docId)
  .order('version', { ascending: false })

// Restore old version
const { data: oldVersion } = await fluxbase
  .from('document_versions')
  .select('title, content')
  .eq('document_id', docId)
  .eq('version', 5)
  .single()

await fluxbase
  .from('documents')
  .update({ title: oldVersion.title, content: oldVersion.content })
  .eq('id', docId)
```

### Enums vs Lookup Tables

**ENUMs (Simple, Static Values)**:

```sql
-- Use for fixed, rarely-changing values
CREATE TYPE post_status AS ENUM ('draft', 'published', 'archived');

CREATE TABLE posts (
  id UUID PRIMARY KEY,
  status post_status NOT NULL DEFAULT 'draft'
);

-- ✅ Advantages:
-- - Type-safe
-- - Compact storage
-- - Fast queries

-- ❌ Disadvantages:
-- - Hard to modify (requires ALTER TYPE)
-- - No additional metadata
```

**Lookup Tables (Flexible, Metadata)**:

```sql
-- Use when values change or need metadata
CREATE TABLE post_statuses (
  id TEXT PRIMARY KEY,  -- or use UUID
  name TEXT NOT NULL,
  description TEXT,
  color TEXT,
  display_order INTEGER,
  is_active BOOLEAN DEFAULT true
);

INSERT INTO post_statuses (id, name, description, color, display_order) VALUES
  ('draft', 'Draft', 'Work in progress', '#gray', 1),
  ('review', 'In Review', 'Pending approval', '#yellow', 2),
  ('published', 'Published', 'Live on site', '#green', 3),
  ('archived', 'Archived', 'No longer visible', '#red', 4);

CREATE TABLE posts (
  id UUID PRIMARY KEY,
  status_id TEXT NOT NULL REFERENCES post_statuses(id)
);

-- ✅ Advantages:
-- - Easy to add/modify values
-- - Can store metadata
-- - Can disable without deleting

-- ❌ Disadvantages:
-- - Requires JOIN for queries
-- - Slightly more storage
```

### Indexes Best Practices

**1. B-tree Indexes (Default)**:

```sql
-- Single column
CREATE INDEX idx_posts_created_at ON posts(created_at);

-- Multiple columns (order matters!)
CREATE INDEX idx_posts_user_status ON posts(user_id, status);

-- Use for: =, <, <=, >, >=, BETWEEN, IN, ORDER BY
```

**2. Partial Indexes**:

```sql
-- Index only published posts
CREATE INDEX idx_published_posts ON posts(created_at)
WHERE status = 'published';

-- Much smaller index, faster queries for published posts
-- ✅ Use when querying a subset frequently
```

**3. Expression Indexes**:

```sql
-- Index on expression
CREATE INDEX idx_users_lower_email ON users(LOWER(email));

-- Query must match expression
SELECT * FROM users WHERE LOWER(email) = 'test@example.com';
```

**4. GIN Indexes (Arrays, JSONB, Full-Text)**:

```sql
-- For JSONB
CREATE INDEX idx_products_metadata ON products USING GIN(metadata);

-- For arrays
CREATE INDEX idx_posts_tags ON posts USING GIN(tags);

-- For full-text search
CREATE INDEX idx_posts_search ON posts USING GIN(search_vector);
```

**5. When NOT to Index**:

- ❌ Small tables (< 1000 rows)
- ❌ Frequently updated columns
- ❌ Columns with low cardinality (few distinct values)
- ❌ Columns rarely used in WHERE/ORDER BY

### Constraints and Data Integrity

**Check Constraints**:

```sql
CREATE TABLE products (
  id UUID PRIMARY KEY,
  name TEXT NOT NULL,
  price NUMERIC NOT NULL CHECK (price >= 0),
  stock INTEGER NOT NULL CHECK (stock >= 0),
  discount_percent INTEGER CHECK (discount_percent BETWEEN 0 AND 100),
  email TEXT CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}$'),

  -- Table-level constraint
  CONSTRAINT valid_discount CHECK (
    discount_percent IS NULL OR
    (price * (1 - discount_percent / 100.0)) >= 0
  )
);
```

**Unique Constraints**:

```sql
-- Simple unique
CREATE TABLE users (
  email TEXT UNIQUE NOT NULL
);

-- Composite unique
CREATE TABLE user_posts (
  user_id UUID,
  post_id UUID,
  UNIQUE(user_id, post_id)
);

-- Partial unique (unique only where deleted_at IS NULL)
CREATE UNIQUE INDEX idx_active_emails
ON users(email) WHERE deleted_at IS NULL;
```

**Exclusion Constraints**:

```sql
-- Prevent overlapping date ranges
CREATE EXTENSION btree_gist;

CREATE TABLE bookings (
  id UUID PRIMARY KEY,
  room_id UUID NOT NULL,
  start_date DATE NOT NULL,
  end_date DATE NOT NULL,
  EXCLUDE USING GIST (
    room_id WITH =,
    daterange(start_date, end_date, '[]') WITH &&
  )
);

-- Prevents: booking same room on overlapping dates
```

---

## 📚 Additional Resources

- [PostgreSQL Performance](https://www.postgresql.org/docs/current/performance-tips.html)
- [Prometheus Best Practices](https://prometheus.io/docs/practices/)
- [Kubernetes Scaling](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/)
- [AWS RDS Best Practices](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/CHAP_BestPractices.html)

---

**Last Updated**: 2025-11-04
**Status**: Complete ✅
**Coverage**: Comprehensive advanced patterns including custom functions, complex queries, and schema design best practices
