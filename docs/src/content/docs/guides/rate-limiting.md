---
title: "Rate Limiting"
---

Fluxbase includes built-in rate limiting to protect your API from abuse, prevent brute-force attacks, and ensure fair resource usage across users.

## Overview

Rate limiting in Fluxbase provides:

- **IP-based rate limiting** - Limit anonymous requests per IP address
- **User-based rate limiting** - Higher limits for authenticated users
- **API key rate limiting** - Configurable limits per API key
- **Endpoint-specific limits** - Different limits for sensitive endpoints
- **Tiered rate limiting** - Different limits based on authentication level
- **Automatic cleanup** - In-memory storage with garbage collection

### Default Rate Limits

| Endpoint Type | Anonymous (IP) | Authenticated User | API Key |
|---------------|----------------|-------------------|---------|
| **Global API** | 100 req/min | 500 req/min | 1000 req/min |
| **Login** | 5 req/15min | N/A | N/A |
| **Signup** | 3 req/hour | N/A | N/A |
| **Password Reset** | 3 req/hour | N/A | N/A |
| **Magic Link** | 3 req/hour | N/A | N/A |
| **Token Refresh** | 10 req/min | 10 req/min | N/A |
| **Admin Setup** | 5 req/15min | N/A | N/A |
| **Admin Login** | 10 req/min | N/A | N/A |

## Quick Start

### Enable Global Rate Limiting

```bash
# .env or environment variables
FLUXBASE_SECURITY_ENABLE_GLOBAL_RATE_LIMIT=true
```

Or in `fluxbase.yaml`:

```yaml
security:
  enable_global_rate_limit: true
```

This enables **100 requests per minute per IP** across all API endpoints.

### Verify Rate Limiting

```bash
# Make 101 requests in 1 minute
for i in {1..101}; do
  curl http://localhost:8080/health
done

# 101st request will return 429 Too Many Requests
```

Response when rate limit is exceeded:

```json
{
  "error": "Rate limit exceeded",
  "message": "API rate limit exceeded. Maximum 100 requests per minute allowed.",
  "retry_after": 60
}
```

---
## Configuration

### Global API Rate Limiting

Applies to all API endpoints:

```bash
# Enable global rate limiting
FLUXBASE_SECURITY_ENABLE_GLOBAL_RATE_LIMIT=true
```

**Default**: 100 requests per minute per IP (when enabled)

### Authentication Endpoint Rate Limits

#### Login Rate Limiting

```bash
# Configuration
FLUXBASE_SECURITY_AUTH_LOGIN_RATE_LIMIT=10
FLUXBASE_SECURITY_AUTH_LOGIN_RATE_WINDOW=1m
```

```yaml
# fluxbase.yaml
security:
  auth_login_rate_limit: 10
  auth_login_rate_window: 1m
```

**Default**: 10 attempts per minute per IP

#### Admin Setup Rate Limiting

```bash
FLUXBASE_SECURITY_ADMIN_SETUP_RATE_LIMIT=5
FLUXBASE_SECURITY_ADMIN_SETUP_RATE_WINDOW=15m
```

```yaml
security:
  admin_setup_rate_limit: 5
  admin_setup_rate_window: 15m
```

**Default**: 5 attempts per 15 minutes per IP

#### Admin Login Rate Limiting

```bash
FLUXBASE_SECURITY_ADMIN_LOGIN_RATE_LIMIT=10
FLUXBASE_SECURITY_ADMIN_LOGIN_RATE_WINDOW=1m
```

```yaml
security:
  admin_login_rate_limit: 10
  admin_login_rate_window: 1m
```

**Default**: 10 attempts per minute per IP
---

## Rate Limiting by Endpoint

Fluxbase applies different rate limits to different endpoints automatically:

### Authentication Endpoints

#### POST `/api/v1/auth/signin`

**Rate Limit**: 5 requests per 15 minutes per IP

**Purpose**: Prevent brute-force login attacks

**Response on limit**:
```json
{
  "error": "Rate limit exceeded",
  "message": "Too many login attempts. Please try again in 15 minutes.",
  "retry_after": 900
}
```

#### POST `/api/v1/auth/signup`

**Rate Limit**: 3 requests per hour per IP

**Purpose**: Prevent spam account creation

**Response on limit**:
```json
{
  "error": "Rate limit exceeded",
  "message": "Too many signup attempts. Please try again in 1 hour.",
  "retry_after": 3600
}
```

#### POST `/api/v1/auth/password/reset`

**Rate Limit**: 3 requests per hour per IP

**Purpose**: Prevent email bombing and abuse

**Key**: Based on email address (if provided) or IP

**Response on limit**:
```json
{
  "error": "Rate limit exceeded",
  "message": "Too many password reset requests. Please try again in 1 hour.",
  "retry_after": 3600
}
```

#### POST `/api/v1/auth/magiclink`

**Rate Limit**: 3 requests per hour per IP

**Purpose**: Prevent email bombing

**Response on limit**:
```json
{
  "error": "Rate limit exceeded",
  "message": "Too many magic link requests. Please try again in 1 hour.",
  "retry_after": 3600
}
```

#### POST `/api/v1/auth/refresh`

**Rate Limit**: 10 requests per minute per refresh token

**Purpose**: Prevent token abuse

**Key**: Based on refresh token (first 20 chars) or IP

### Admin Endpoints

#### POST `/api/v1/admin/setup`

**Rate Limit**: 5 requests per 15 minutes per IP

**Purpose**: Prevent brute-force on initial setup

**Response on limit**:
```json
{
  "error": "Rate limit exceeded",
  "message": "Too many admin setup attempts. Please try again in 15 minutes.",
  "retry_after": 900
}
```

#### POST `/api/v1/admin/login`

**Rate Limit**: 10 requests per minute per IP

**Purpose**: Prevent admin account takeover attempts

---

## Tiered Rate Limiting

Fluxbase implements tiered rate limiting based on authentication level:

### Anonymous (IP-based)

**Limit**: 100 requests per minute

**Key**: Client IP address

**Use case**: Public endpoints, unauthenticated users

```bash
# Example: Anonymous user
curl http://localhost:8080/api/v1/tables/posts
# Rate limit key: ip:192.168.1.100:100
```

### Authenticated Users

**Limit**: 500 requests per minute

**Key**: User ID from JWT token

**Use case**: Logged-in users with valid session

```bash
# Example: Authenticated user
curl http://localhost:8080/api/v1/tables/posts \
  -H "Authorization: Bearer eyJhbGc..."
# Rate limit key: user:user-uuid-here:500
```

### API Keys

**Limit**: 1,000 requests per minute (configurable per key)

**Key**: API key ID

**Use case**: Server-to-server integrations, higher throughput needs

```bash
# Example: API key authentication
curl http://localhost:8080/api/v1/tables/posts \
  -H "Authorization: Bearer sk_live_xxxxx"
# Rate limit key: apikey:apikey-uuid:1000
```

### Priority Order

When multiple authentication methods are present:

1. **API Key** - Highest priority, highest limits
2. **User JWT** - Medium priority, medium limits
3. **IP Address** - Fallback, lowest limits

---

## Rate Limit Headers

Fluxbase includes rate limit information in response headers:

```http
HTTP/1.1 200 OK
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1640000000
```

**Headers**:

- `X-RateLimit-Limit` - Maximum requests allowed in time window
- `X-RateLimit-Remaining` - Requests remaining in current window
- `X-RateLimit-Reset` - Unix timestamp when the rate limit resets

When rate limit is exceeded:

```http
HTTP/1.1 429 Too Many Requests
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1640000060
Retry-After: 60

{
  "error": "Rate limit exceeded",
  "message": "API rate limit exceeded. Maximum 100 requests per minute allowed.",
  "retry_after": 60
}
```

---

## API Key Rate Limits

When creating API keys, you can set custom rate limits:

```bash
# Create API key with custom rate limit
curl -X POST http://localhost:8080/api/v1/admin/api-keys \
  -H "Authorization: Bearer admin-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "External Integration",
    "rate_limit_rpm": 5000,
    "expires_at": "2025-12-31T23:59:59Z"
  }'
```

Response:
```json
{
  "id": "apikey-uuid",
  "key": "sk_live_xxxxxxxxxx",
  "name": "External Integration",
  "rate_limit_rpm": 5000,
  "created_at": "2024-01-26T10:00:00Z",
  "expires_at": "2025-12-31T23:59:59Z"
}
```

---

## Distributed Rate Limiting

For multi-instance deployments, use a distributed backend for rate limiting:

### Configure Scaling Backend

```bash
# Option 1: PostgreSQL (recommended, no extra dependencies)
FLUXBASE_SCALING_BACKEND=postgres

# Option 2: Redis/Dragonfly (for high-scale, 1000+ req/s)
FLUXBASE_SCALING_BACKEND=redis
FLUXBASE_SCALING_REDIS_URL=redis://dragonfly:6379
```

```yaml
# fluxbase.yaml
scaling:
  backend: postgres  # or "redis"
  redis_url: ""      # only needed if backend is redis
```

**Backend Comparison:**

| Backend | Use Case | Performance |
|---------|----------|-------------|
| `local` | Single instance (default) | Fastest, in-memory |
| `postgres` | Multi-instance, < 1000 req/s | Good, uses UPSERT |
| `redis` | High-scale, > 1000 req/s | Best, in-memory distributed |

**Dragonfly Recommended:** For the `redis` backend, we recommend [Dragonfly](https://dragonflydb.io/) - a Redis-compatible datastore that is 25x faster with 80% less memory than Redis.

**Benefits**:
- Shared rate limit state across all Fluxbase instances
- Consistent rate limiting in horizontal scaling
- No sticky sessions required

**Architecture**:

```
┌─────────────┐
│  Client     │
└──────┬──────┘
       │
  ┌────▼─────────────────────┐
  │   Load Balancer          │
  └────┬─────────────────────┘
       │
  ┌────┴─────┬────────┬──────┐
  │          │        │      │
┌─▼──┐   ┌──▼─┐   ┌──▼─┐  ┌─▼──┐
│FB 1│   │FB 2│   │FB 3│  │FB 4│
└─┬──┘   └──┬─┘   └──┬─┘  └─┬──┘
  │         │        │      │
  └─────────┴────────┴──────┘
            │
   ┌────────▼─────────┐
   │ PostgreSQL or    │
   │ Dragonfly/Redis  │
   │  (Shared State)  │
   └──────────────────┘
```

---

## Monitoring Rate Limits

### Prometheus Metrics

Fluxbase exposes rate limiting metrics:

```txt
# Total rate limit hits
rate_limit_hits_total{endpoint="/api/v1/auth/signin"}

# Rate limit by status
rate_limit_status{status="allowed"}
rate_limit_status{status="blocked"}

# Current rate limit usage
rate_limit_current_usage{key_type="ip"}
rate_limit_current_usage{key_type="user"}
rate_limit_current_usage{key_type="apikey"}
```

### View Rate Limit Logs

```bash
# Enable debug logging
FLUXBASE_DEBUG=true

# View logs
docker logs fluxbase | grep "rate limit"

# Kubernetes
kubectl logs -f deployment/fluxbase -n fluxbase | grep "rate limit"
```

Sample log output:
```
{"level":"warn","time":"2024-01-26T10:30:00Z","ip":"192.168.1.100","endpoint":"/api/v1/auth/signin","message":"rate limit exceeded"}
```

---

## Client-Side Handling

### Respect Rate Limits

Implement exponential backoff when receiving 429 responses:

```typescript
// TypeScript/JavaScript client
async function fetchWithRetry(url: string, options: RequestInit, maxRetries = 3) {
  for (let i = 0; i < maxRetries; i++) {
    const response = await fetch(url, options);

    if (response.status === 429) {
      const retryAfter = parseInt(response.headers.get('Retry-After') || '60');
      console.warn(`Rate limited. Retrying after ${retryAfter} seconds...`);
      await new Promise(resolve => setTimeout(resolve, retryAfter * 1000));
      continue;
    }

    return response;
  }

  throw new Error('Max retries exceeded');
}

// Usage
const response = await fetchWithRetry('http://localhost:8080/api/v1/tables/posts', {
  headers: { 'Authorization': 'Bearer token' }
});
```

### Check Rate Limit Status

```typescript
const response = await fetch('http://localhost:8080/api/v1/tables/posts');

const limit = parseInt(response.headers.get('X-RateLimit-Limit') || '0');
const remaining = parseInt(response.headers.get('X-RateLimit-Remaining') || '0');
const reset = parseInt(response.headers.get('X-RateLimit-Reset') || '0');

console.log(`Rate limit: ${remaining}/${limit} remaining`);
console.log(`Resets at: ${new Date(reset * 1000)}`);

if (remaining < 10) {
  console.warn('Approaching rate limit!');
}
```

### Batch Requests

Reduce rate limit impact by batching requests:

```typescript
// Bad: Many individual requests
for (const id of userIds) {
  await fetch(`/api/v1/users/${id}`);
}

// Good: Single batched request
const users = await fetch('/api/v1/users', {
  method: 'POST',
  body: JSON.stringify({ ids: userIds })
});
```

---

## Security Best Practices

### 1. Enable Global Rate Limiting in Production

```bash
# Always enable in production
FLUXBASE_SECURITY_ENABLE_GLOBAL_RATE_LIMIT=true
```

### 2. Use Stricter Limits for Sensitive Endpoints

Authentication endpoints have built-in strict limits:
- Login: 5 attempts per 15 minutes
- Signup: 3 per hour
- Password reset: 3 per hour

**Do not disable these** unless absolutely necessary.

### 3. Monitor for Abuse

Set up alerts for high rate limit violations:

```yaml
# Prometheus alert rule
groups:
  - name: rate_limiting
    rules:
      - alert: HighRateLimitViolations
        expr: rate(rate_limit_hits_total{status="blocked"}[5m]) > 10
        for: 5m
        annotations:
          summary: "High rate limit violations detected"
          description: "More than 10 rate limit violations per second for 5 minutes"
```

### 4. Use API Keys for Server-to-Server

For integrations requiring high throughput:

```bash
# Create API key with appropriate limits
curl -X POST /api/v1/admin/api-keys \
  -d '{"name": "Partner API", "rate_limit_rpm": 5000}'
```

---

## Troubleshooting

### Rate Limit Too Strict

**Symptom**: Legitimate users getting rate limited

**Solution 1**: Increase limits for authenticated users

```bash
# Users get 500 req/min by default
# If using custom implementation, adjust accordingly
```

**Solution 2**: Use API keys for high-traffic clients

```bash
# Create API key with higher limit
curl -X POST /api/v1/admin/api-keys \
  -H "Authorization: Bearer admin-token" \
  -d '{"name": "High Traffic Client", "rate_limit_rpm": 10000}'
```

### Rate Limit Not Working

**Check 1**: Verify rate limiting is enabled

```bash
# Check configuration
FLUXBASE_SECURITY_ENABLE_GLOBAL_RATE_LIMIT=true
```

**Check 2**: Check logs

```bash
# Enable debug mode
FLUXBASE_DEBUG=true

# View logs
docker logs fluxbase | grep -i "rate"
```

**Check 3**: Test manually

```bash
# Send 101 requests quickly
for i in {1..101}; do
  curl http://localhost:8080/health
done

# 101st should return 429
```

### Distributed Setup Not Working

**Issue**: Each instance has separate rate limits

**Solution**: Enable Redis

```bash
FLUXBASE_REDIS_ENABLED=true
FLUXBASE_REDIS_HOST=redis
FLUXBASE_REDIS_PORT=6379
```

**Verify**:
```bash
# Connect to Redis
redis-cli

# Check rate limit keys
KEYS rate_limit:*
```

### False Positives from Load Balancer

**Issue**: All requests appear from same IP (load balancer IP)

**Solution**: Configure load balancer to forward real client IP

**Nginx**:
```nginx
proxy_set_header X-Real-IP $remote_addr;
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
```

**Kubernetes Ingress**:
```yaml
annotations:
  nginx.ingress.kubernetes.io/use-forwarded-headers: "true"
```

Fluxbase automatically uses `X-Forwarded-For` header when available.

---

## Performance Impact

Rate limiting adds minimal overhead:

- **In-memory storage**: < 1ms per request
- **Redis storage**: < 5ms per request (network latency)
- **Memory usage**: ~100 bytes per unique key

**Benchmarks** (single instance):

| Storage | Requests/sec | Avg Latency | p99 Latency |
|---------|--------------|-------------|-------------|
| Memory | 10,000 | 0.5ms | 2ms |
| Redis | 8,000 | 2ms | 10ms |

---

## Migration from Other Systems

### From Supabase

Supabase uses Kong for rate limiting. Fluxbase provides similar functionality:

| Supabase (Kong) | Fluxbase |
|----------------|----------|
| Anonymous: 60 req/min | Anonymous: 100 req/min |
| Authenticated: 600 req/min | Authenticated: 500 req/min |
| Service key: Unlimited | API Key: 1000 req/min (configurable) |

**Migration steps**:

1. Enable rate limiting:
   ```bash
   FLUXBASE_SECURITY_ENABLE_GLOBAL_RATE_LIMIT=true
   ```

2. Adjust limits to match Supabase (if needed):
   ```go
   // Custom configuration (if extending)
   GlobalAPILimiter(60) // Match Supabase anonymous limit
   ```

3. Test with existing clients

### From Firebase

Firebase rate limits are per-function. Fluxbase applies global + endpoint-specific limits.

**Recommended approach**:

1. Start with default limits
2. Monitor usage patterns
3. Adjust per-endpoint limits as needed

---

## Next Steps

- [Authentication](/docs/guides/authentication) - Secure your API endpoints
- [SDK Admin Documentation](/docs/api/sdk/classes/APIKeysManager) - Create API keys for integrations
- [Monitoring](/docs/guides/monitoring-observability) - Set up observability
- [Security](/docs/security/overview) - Additional security measures
