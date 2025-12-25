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
| **Login** | 10 req/15min | N/A | N/A |
| **Signup** | 10 req/15min | N/A | N/A |
| **Password Reset** | 5 req/15min | N/A | N/A |
| **Magic Link** | 5 req/15min | N/A | N/A |
| **2FA Verification** | 5 req/5min | N/A | N/A |
| **Token Refresh** | 10 req/min | 10 req/min | N/A |
| **Admin Setup** | 5 req/15min | N/A | N/A |
| **Admin Login** | 4 req/min | N/A | N/A |

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

**Rate Limit**: 10 requests per 15 minutes per IP

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

**Rate Limit**: 10 requests per 15 minutes per IP

**Purpose**: Prevent spam account creation

**Response on limit**:
```json
{
  "error": "Rate limit exceeded",
  "message": "Too many signup attempts. Please try again in 15 minutes.",
  "retry_after": 900
}
```

#### POST `/api/v1/auth/password/reset`

**Rate Limit**: 5 requests per 15 minutes per IP

**Purpose**: Prevent email bombing and abuse

**Key**: Based on email address (if provided) or IP

**Response on limit**:
```json
{
  "error": "Rate limit exceeded",
  "message": "Too many password reset requests. Please try again in 15 minutes.",
  "retry_after": 900
}
```

#### POST `/api/v1/auth/magiclink`

**Rate Limit**: 5 requests per 15 minutes per IP

**Purpose**: Prevent email bombing

**Response on limit**:
```json
{
  "error": "Rate limit exceeded",
  "message": "Too many magic link requests. Please try again in 15 minutes.",
  "retry_after": 900
}
```

#### POST `/api/v1/auth/2fa/verify`

**Rate Limit**: 5 requests per 5 minutes per IP

**Purpose**: Prevent brute-force attacks on 6-digit TOTP codes

**Response on limit**:
```json
{
  "error": "Rate limit exceeded",
  "message": "Too many 2FA verification attempts. Please try again in 5 minutes.",
  "retry_after": 300
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

**Rate Limit**: 4 requests per minute per IP

**Purpose**: Prevent admin account takeover attempts. Set to 4 to trigger rate limiting before account lockout (which happens at 5 failed attempts).

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

## Multi-Instance Deployments

:::caution[Important Limitation]
Fluxbase's built-in rate limiting uses **in-memory storage per instance**. In multi-instance deployments, each instance maintains its own rate limit counters independently. This means attackers could potentially bypass rate limits by targeting different instances.
:::

### Current Behavior

| Deployment | Rate Limiting Behavior |
|------------|------------------------|
| Single instance | Full protection - all requests share counters |
| Multi-instance | Per-instance only - counters are NOT shared |

### Recommended Solutions for Multi-Instance

For production environments with horizontal scaling, use one of these approaches:

**Option 1: Reverse Proxy Rate Limiting (Recommended)**

Use your load balancer or reverse proxy for centralized rate limiting:

```nginx
# nginx example
limit_req_zone $binary_remote_addr zone=api:10m rate=100r/m;

server {
    location /api/ {
        limit_req zone=api burst=20 nodelay;
        proxy_pass http://fluxbase;
    }
}
```

**Option 2: Kubernetes Ingress**

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    nginx.ingress.kubernetes.io/limit-rps: "100"
    nginx.ingress.kubernetes.io/limit-connections: "10"
```

**Option 3: API Gateway**

Use an API gateway like Kong, Traefik, or AWS API Gateway with built-in distributed rate limiting.

### Architecture with External Rate Limiting

```
┌─────────────┐
│  Client     │
└──────┬──────┘
       │
  ┌────▼─────────────────────┐
  │   Load Balancer          │
  │   (Rate Limiting Here)   │
  └────┬─────────────────────┘
       │
  ┌────┴─────┬────────┬──────┐
  │          │        │      │
┌─▼──┐   ┌──▼─┐   ┌──▼─┐  ┌─▼──┐
│FB 1│   │FB 2│   │FB 3│  │FB 4│
└────┘   └────┘   └────┘  └────┘
```

This approach provides:

- Centralized rate limit state
- Consistent rate limiting across all instances
- No sticky sessions required

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

The SDK automatically handles rate limit responses with exponential backoff:

```typescript
import { createClient } from '@fluxbase/sdk'

const client = createClient({
  baseUrl: 'http://localhost:8080',
})

// The SDK automatically retries on 429 responses
const { data: posts } = await client.from('posts').select()
```

If you need custom retry logic, you can catch the error:

```typescript
try {
  const { data } = await client.from('posts').select()
} catch (error) {
  if (error.status === 429) {
    const retryAfter = error.headers?.get('Retry-After') || 60
    console.warn(`Rate limited. Retry after ${retryAfter} seconds`)
  }
}
```

### Batch Requests

Reduce rate limit impact by using efficient queries:

```typescript
// Bad: Many individual requests
for (const id of userIds) {
  await client.from('users').select().eq('id', id).single()
}

// Good: Single query with filter
const { data: users } = await client
  .from('users')
  .select()
  .in('id', userIds)
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
- Login: 10 attempts per 15 minutes
- Signup: 10 per 15 minutes
- Password reset: 5 per 15 minutes
- 2FA verification: 5 per 5 minutes

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

### Each Instance Has Separate Rate Limits

**Issue**: In multi-instance deployments, rate limits are not shared across instances.

**Solution**: This is expected behavior. Fluxbase uses in-memory rate limiting per instance. For centralized rate limiting, use a reverse proxy or API gateway. See [Multi-Instance Deployments](#multi-instance-deployments) above.

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
