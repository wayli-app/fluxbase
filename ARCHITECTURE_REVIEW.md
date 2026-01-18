# Fluxbase Backend-as-a-Service Architecture Review

**Review Date:** 2026-01-18
**Reviewer:** Architecture Analysis
**Codebase Version:** commit dbded21

---

## Executive Summary

Fluxbase is a well-architected Backend-as-a-Service platform that provides a compelling Supabase alternative for self-hosting scenarios. The codebase demonstrates strong engineering practices in several areas (security, observability, deployment) while showing typical growing pains in others (scalability bottlenecks, N+1 query patterns). This review identifies **8 high-priority issues**, **15 medium-priority improvements**, and **12 nice-to-have enhancements** across security, correctness, scalability, developer experience, and operations.

**Overall Assessment:** Production-ready for small-to-medium deployments (< 1000 concurrent connections, < 100 req/s). Requires targeted improvements for high-scale production use.

---

## 1. Product Scope and Understanding

### 1.1 Core Value Proposition

Fluxbase is a **single-binary Backend-as-a-Service** platform with PostgreSQL as the only external dependency. It provides:

| Feature | Implementation Status |
|---------|----------------------|
| **Authentication** | Complete - JWT, OAuth2/OIDC, SAML, Magic Links, MFA/TOTP |
| **Database Access** | Complete - PostgREST-compatible REST API, GraphQL, Row-Level Security |
| **Storage** | Complete - Local filesystem, S3/MinIO with signed URLs, image transforms |
| **Realtime** | Complete - WebSocket subscriptions via PostgreSQL LISTEN/NOTIFY |
| **Edge Functions** | Complete - Deno runtime with bundling, scheduling |
| **Background Jobs** | Complete - Worker pool, cron scheduling, progress tracking |
| **AI/Chatbot** | Complete - LLM integration, vector search (pgvector), RAG |
| **Database Branching** | Complete - Isolated databases for dev/test environments |
| **MCP Server** | Complete - Model Context Protocol for AI assistant integration |

### 1.2 Explicit Boundaries (What Fluxbase Does NOT Handle)

- **Billing/Subscriptions**: No built-in payment processing or usage metering
- **Multi-Tenant Isolation**: Single-tenant design; multi-tenant requires separate instances
- **Heavy Analytics**: No OLAP/data warehouse features; designed for OLTP workloads
- **CDN/Edge Caching**: No built-in CDN; relies on external solutions (CloudFront, Cloudflare)
- **Email Inbox**: Provides sending only; no email receiving/parsing

### 1.3 Supported Deployment Models

| Model | Support Level | Notes |
|-------|---------------|-------|
| Single Binary | Excellent | Primary use case, minimal dependencies |
| Docker Compose | Excellent | Full stack with optional services |
| Kubernetes (Helm) | Excellent | Production-ready charts with HPA, Ingress |
| Self-hosted VPS | Good | Single binary simplifies deployment |
| Home Lab | Good | Low resource requirements |

### 1.4 Major Capability Gaps for Production BaaS

1. **No API Rate Limiting Per User/Project**: Rate limiting is per-instance only, not per-tenant
2. **No Usage Quotas**: No storage limits, request limits, or compute limits per user
3. **No Webhook Delivery Guarantees**: At-most-once delivery; no retry queue
4. **No GraphQL Subscriptions**: Realtime via separate WebSocket API only
5. **No API Versioning**: No `Accept-Version` header support
6. **No Request Signing**: Webhooks lack HMAC signatures for verification

---

## 2. API and Product Completeness

### 2.1 API Organization Assessment

**Route Structure:**
```
/api/v1/
├── /tables/{schema?}/{table}      # REST CRUD (PostgREST-compatible)
├── /auth/*                        # Authentication endpoints
├── /storage/*                     # File operations
├── /functions/*                   # Edge function invocation
├── /jobs/*                        # Background job operations
├── /realtime/*                    # WebSocket stats/broadcast
├── /graphql                       # GraphQL endpoint
├── /branches/*                    # Database branching
└── /admin/*                       # Dashboard routes
```

**Strengths:**
- Consistent resource naming following REST conventions
- PostgREST-compatible filter syntax for broad ecosystem support
- Clear separation between public API and admin endpoints
- Scope-based authorization per endpoint

**Weaknesses:**
- Path parameter ambiguity: `/tables/users` vs `/tables/auth/users` relies on schema detection
- No explicit API versioning (v1 in path but no version negotiation)
- Inconsistent between POST body formats (array vs object for batch operations)

### 2.2 Error Modeling

**Current Error Response Patterns:**

```json
// Pattern 1: Simple error
{"error": "Record not found"}

// Pattern 2: Detailed error (RLS violations)
{"error": "...", "code": "RLS_POLICY_VIOLATION", "message": "..."}

// Pattern 3: Validation errors
{"error": "Validation failed", "details": {...}}
```

**Issues:**
- **Inconsistent structure**: Some endpoints return `error` string, others return structured objects
- **Missing error codes**: Most endpoints lack machine-readable error codes
- **No request ID in errors**: Makes debugging difficult without correlation

**Recommendation:** Standardize on RFC 7807 Problem Details:
```json
{
  "type": "https://fluxbase.dev/errors/rls-violation",
  "title": "Row Level Security Policy Violation",
  "status": 403,
  "detail": "User lacks permission to access this record",
  "instance": "/api/v1/tables/users/123",
  "request_id": "req_abc123"
}
```

### 2.3 Concrete API Gaps

| Gap | Impact | Severity |
|-----|--------|----------|
| **PUT and PATCH have identical semantics** | Violates HTTP semantics; confuses developers | Medium |
| **RLS denial indistinguishable from 404** | Cannot tell if record doesn't exist vs. no permission | Medium |
| **Silent limit capping** | Users don't know when results are truncated | Medium |
| **No bulk operation response counts** | Batch DELETE/PATCH doesn't return affected count | Low |
| **No conditional requests (ETags)** | No If-Modified-Since, If-None-Match support | Low |
| **Invalid filter operators silently ignored** | Typos in operators default to `=` | Low |
| **No idempotency keys for mutations** | POST requests not safe to retry | Medium |

### 2.4 Pagination Implementation

**Current Implementation:**
- Default page size: 100 (configurable)
- Max page size: 1000 (configurable)
- Max total results: 10,000 (offset + limit cap)
- `Content-Range` header for total counts

**Issues:**
1. Silent capping when limits exceeded (no warning header)
2. 0-based indexing in Content-Range but 1-based row counts (mixing conventions)
3. No cursor-based pagination for large datasets (offset-based only)

### 2.5 SDK Assessment (TypeScript)

**Strengths:**
- Fluent query builder API matching Supabase patterns
- Automatic fallback to POST for complex queries exceeding URL length
- Comprehensive filter operators including vector similarity

**Gaps:**
- No auto-generated TypeScript types from database schema
- Error types not strongly typed (generic `Error | null`)
- No retry logic for transient failures
- No streaming support for large responses

---

## 3. Security Review

### 3.1 Authentication Architecture

**Rating: STRONG (8.5/10)**

| Component | Implementation | Assessment |
|-----------|---------------|------------|
| JWT Tokens | HMAC-SHA256, access (15m) + refresh (7d) | ✅ Proper rotation, JTI tracking |
| Password Hashing | bcrypt cost=12, 12-char minimum | ✅ Modern standards |
| Session Management | SHA-256 hashed tokens in DB | ✅ Database breach protection |
| MFA/TOTP | pquerna/otp, encrypted secrets, backup codes | ✅ Well implemented |
| OAuth/OIDC | Google, GitHub, Microsoft, Apple, etc. | ✅ Standard implementation |
| Account Lockout | 5 failed attempts | ✅ Brute force protection |
| Token Revocation | Blacklist with JTI tracking | ✅ Immediate invalidation |

**Security Concerns:**

1. **TOTP Secret Encryption Optional** (Medium Risk)
   - Location: `internal/auth/totp.go`
   - Issue: If `encryptionKey` not configured, TOTP secrets stored plaintext
   - Recommendation: Make encryption mandatory or warn loudly on startup

2. **No Brute-Force Protection on TOTP Verification** (Medium Risk)
   - Issue: 6-digit codes theoretically brute-forceable (1M combinations)
   - Recommendation: Add per-user rate limiting (5 attempts per 5 minutes)

3. **Service Role Tokens Cannot Be Revoked** (Medium Risk)
   - Issue: By design, service role JWTs skip revocation check
   - Impact: No emergency response for compromised service keys
   - Recommendation: Add optional revocation list for service role tokens

4. **JWT Secret Length Not Validated** (Low Risk)
   - Issue: No minimum entropy requirement for HMAC-SHA256 secret
   - Recommendation: Require 256-bit minimum at startup

### 3.2 Authorization Model

**Row-Level Security Implementation:**
```sql
SET LOCAL ROLE <role>;  -- authenticated | anon | service_role
SET request.jwt.claims = '<json>';  -- Full JWT claims for RLS policies
```

**Strengths:**
- Defense-in-depth: Middleware + Database-level RLS
- PostgreSQL native RLS policies enforced at query execution
- Custom claims support for complex authorization patterns

**Concerns:**
1. Role mapping simplified: All authenticated users → single `authenticated` role
2. No user ID tracking in service role context
3. Custom claims not validated server-side (trusted from JWT)

### 3.3 Input Validation

**Good Practices:**
- SQL injection prevention via parameterized queries ($N placeholders)
- Identifier validation with `^[a-zA-Z_][a-zA-Z0-9_]*$` regex
- GeoJSON structure validation before PostGIS conversion
- Email validation with RFC 5322 compliance + dangerous character blocking

**Gaps:**
- No request body size limits per-endpoint (global limit only)
- No JSON depth limiting (potential stack overflow on deeply nested objects)
- File upload MIME type validation relies on extension, not magic bytes

### 3.4 Rate Limiting

**Critical Issue: In-Memory Rate Limiting Not Distributed**

- Location: `internal/middleware/rate_limiter.go`
- Problem: Per-instance rate limiting; attackers can target different instances
- Impact: Rate limits effectively bypassed in multi-instance deployments
- Status: Documented warning present but commonly overlooked

**Recommendation:** Use Redis/Dragonfly backend for distributed rate limiting (already supported in code, just needs configuration).

### 3.5 Security Headers

**Implemented:**
- CSP (Content Security Policy) with appropriate restrictions
- HSTS (HTTP Strict Transport Security)
- X-Frame-Options: DENY
- X-Content-Type-Options: nosniff
- Server header removed (information disclosure prevention)

**Gap:** Admin UI requires `unsafe-inline` and `unsafe-eval` for React (acceptable trade-off).

### 3.6 Prioritized Security Risks

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Distributed rate limit bypass | High | High (in multi-instance) | Use Redis backend for rate limiting |
| TOTP brute force | Medium | Low | Add per-user attempt limiting |
| Plaintext TOTP secrets | Medium | Low (requires DB breach) | Make encryption mandatory |
| Service key compromise | High | Low | Add optional revocation support |
| Slow-rate enumeration | Low | Medium | Implement account lockout on multiple IPs |

---

## 4. Maintainability and Code Health

### 4.1 Architecture and Modularity

**Layer Separation:**
```
cmd/fluxbase/main.go           # Entry point, CLI flags
internal/api/                   # HTTP handlers (60+ files)
internal/auth/                  # Authentication services
internal/middleware/            # HTTP middleware stack
internal/database/              # PostgreSQL access
internal/storage/               # File storage abstraction
internal/jobs/                  # Background job system
internal/functions/             # Edge functions runtime
internal/realtime/              # WebSocket subscriptions
internal/observability/         # Metrics, tracing
```

**Strengths:**
- Clear interface-based dependency injection
- Repository pattern for data access
- Handler pattern with `*fiber.Ctx` for HTTP processing
- Feature services conditionally initialized based on config

**Layering Violations:**
1. `internal/api/` handlers sometimes directly access database bypassing service layer
2. `internal/auth/` has circular dependency on settings cache (resolved via SetSettingsCache())
3. Some middleware accesses database directly for RLS enforcement (acceptable for performance)

**Adding New Features:**
Relatively straightforward process:
1. Add config section in `internal/config/`
2. Create service in `internal/<feature>/`
3. Add handler in `internal/api/`
4. Register routes in `server.go`
5. Add migrations in `internal/database/migrations/`

### 4.2 Code Patterns and Clarity

**Consistent Patterns:**
- Error wrapping with context (`fmt.Errorf("operation failed: %w", err)`)
- Structured logging via zerolog
- Context propagation for cancellation
- Graceful shutdown with timeout

**Inconsistent Patterns:**
1. **Error handling in progress updates**: Uses `context.Background()` instead of job context
   - Location: `internal/jobs/worker.go:515`

2. **Column lookup performance**: O(n) scan on every field validation
   - Location: `internal/api/rest_crud.go`

3. **Logging levels**: Mix of `log.Info()` and `log.Debug()` for similar operations

**Duplication Opportunities:**
- Filter parsing logic duplicated between REST and GraphQL handlers
- Pagination handling repeated across multiple endpoints
- Error response formatting inconsistent (should extract to utility)

### 4.3 Testing Strategy

**Current Coverage:**

| Module | Source Lines | Test Lines | Ratio |
|--------|-------------|------------|-------|
| Observability | 847 | 841 | 99% |
| Logging | 1,224 | 2,506 | 205% |
| Auth | ~3,000 | ~1,500 | 50% |
| API | ~8,000 | ~1,200 | 15% |
| Jobs | ~1,500 | ~400 | 27% |

**Test Types Present:**
- Unit tests: Alongside source files (`*_test.go`)
- E2E tests: `test/e2e/` (32 files, 3,510+ lines)
- Test utilities: `internal/testutil/` (mocks, helpers)

**Critical Flows Lacking Tests:**
1. RLS policy enforcement (tested via E2E but limited unit tests)
2. Edge function execution timeout handling
3. Chunked upload resumption
4. Real-time subscription filtering with complex conditions
5. Database branching cleanup scheduling

**Recommended Testing Strategy:**
1. **Immediate**: Add unit tests for `internal/api/rest_crud.go` (core CRUD logic)
2. **Short-term**: Add integration tests for auth token rotation scenarios
3. **Medium-term**: Add chaos tests for database connection failures
4. **CI Enforcement**: Require 30% coverage on new files in critical modules

### 4.4 Maintainability Shape

The codebase is **moderately maintainable** with clear module boundaries but some internal complexity:

- **Onboarding**: New developers can understand high-level structure quickly
- **Feature Addition**: Well-defined patterns for adding new endpoints
- **Debugging**: Good logging and observability aids troubleshooting
- **Refactoring**: Some modules tightly coupled (auth ↔ settings cache)

### 4.5 High-ROI Refactorings

1. **Extract error response utility** (2 days)
   - Standardize error format across all handlers
   - Add request ID to all error responses

2. **Consolidate filter parsing** (3 days)
   - Single parser for REST, GraphQL, and real-time filters
   - Reduce duplication and inconsistency

3. **Add column validation cache** (1 day)
   - Cache column existence checks per table
   - Invalidate with schema cache

4. **Standardize context usage** (2 days)
   - Fix progress update context issue in jobs
   - Ensure consistent cancellation handling

---

## 5. Scalability and Performance

### 5.1 Architecture for Scaling

**Current Model:** Modular monolith with horizontal scaling support

```
                    ┌─────────────────────────────────────┐
                    │         Load Balancer               │
                    └──────────────┬──────────────────────┘
                                   │
        ┌──────────────────────────┼──────────────────────────┐
        │                          │                          │
   ┌────▼────┐               ┌─────▼────┐               ┌─────▼────┐
   │Instance 1│               │Instance 2│               │Instance 3│
   │ API +    │               │ API +    │               │ Worker   │
   │ Worker   │               │ Worker   │               │ Only     │
   └────┬─────┘               └────┬─────┘               └────┬─────┘
        │                          │                          │
        └──────────────────────────┼──────────────────────────┘
                                   │
                    ┌──────────────▼──────────────────────┐
                    │    PostgreSQL (Primary + Replicas)  │
                    └─────────────────────────────────────┘
```

**Scaling Capabilities:**
- `--worker-only` flag for dedicated background job instances
- `--disable-scheduler` for API-only instances
- Leader election via PostgreSQL advisory locks for schedulers
- PubSub for cross-instance communication (memory, Postgres, or Redis backends)

**Stateful Components:**
- WebSocket connections (per-instance, not migratable)
- Rate limiting (per-instance by default, Redis for distributed)
- CSRF tokens (in-memory, session affinity required)
- Schema cache (per-instance with PubSub invalidation)

### 5.2 Data Layer Analysis

**Connection Pooling:**
- Default: 25 max connections, 5 min connections
- Concern: May be insufficient for concurrent workloads (jobs + API + realtime)
- Recommendation: Increase to 50-100 for multi-instance deployments

**N+1 Query Patterns Identified:**

1. **Schema Cache Refresh** (`internal/database/schema_cache.go:66-119`)
   ```
   For N tables: 1 + 4N queries (columns, PKs, FKs, indexes per table)
   200 tables = 801 queries for full refresh
   ```

2. **Function Parameter Loading** (`internal/database/schema_inspector.go:563-634`)
   ```
   For N functions: N additional queries for parameters
   ```

**Query Performance Concerns:**
- `SELECT FOR UPDATE SKIP LOCKED` for job claiming is efficient
- No query result caching (only schema caching)
- `QueryExecModeDescribeExec` adds round-trip per query (required for schema flexibility)

**Migration Handling:**
- Embedded migrations run at startup
- Dirty flag recovery implemented
- No zero-downtime migration support (requires application restart)

### 5.3 Scalability Bottlenecks

| Bottleneck | Component | Impact at Scale | Mitigation |
|------------|-----------|-----------------|------------|
| **Single LISTEN connection** | Realtime | All notifications through one connection | Implement connection pool |
| **RLS cache size (10K)** | Realtime | Cache misses cause DB roundtrips | Increase to 100K+ entries |
| **Broadcast lock contention** | Realtime | RWMutex held during broadcast | Implement async message queues |
| **Schema cache N+1** | Database | Slow startup/refresh with many tables | Batch queries with JOINs |
| **In-memory rate limiting** | API | Bypassed in multi-instance | Use Redis backend |
| **Image transform concurrency** | Storage | 4 concurrent transforms (default) | Separate transform service |

### 5.4 Throughput and Resource Analysis

**Likely Hotspots:**
1. `/api/v1/tables/{table}` - CRUD operations with RLS checking
2. `/realtime` - WebSocket message broadcasting
3. Schema cache refresh on DDL changes
4. Job worker polling (100ms intervals × N workers)

**Missing Resilience Patterns:**
- No circuit breakers for external services (email, S3)
- No backpressure on real-time message queues
- No request queuing for overloaded endpoints
- Limited retry strategies (only database connection has retry)

### 5.5 Recommended Load Tests

| Test | Setup | Metrics to Watch |
|------|-------|------------------|
| REST CRUD throughput | 100 concurrent users, mixed CRUD | P95 latency, error rate, DB connections |
| WebSocket scale | 5000 concurrent connections | Memory per connection, broadcast latency |
| Real-time RLS | 1000 subscriptions, high churn | RLS cache hit rate, DB query rate |
| Job throughput | 10 workers, 1000 queued jobs | Job completion rate, worker utilization |
| Schema refresh | 500 tables, force invalidation | Refresh duration, query count |
| Mixed workload | 80% reads, 20% writes | Connection pool exhaustion, deadlocks |
| Function concurrency | 50 concurrent function invocations | Memory usage, process count, FD usage |
| Chunked upload | 10 concurrent 1GB uploads | Disk I/O, memory usage |
| Auth token refresh storm | 1000 simultaneous refreshes | Token generation rate, session table locks |
| Chaos: DB disconnect | Kill DB connection during operation | Recovery time, data consistency |

### 5.6 Scalability Improvement Strategies

**Short-term (configuration changes):**
1. Increase connection pool to 50-100
2. Enable Redis/Dragonfly backend for rate limiting and PubSub
3. Increase RLS cache to 100K entries
4. Deploy dedicated worker-only instances

**Medium-term (code changes):**
1. Implement LISTEN connection pooling
2. Batch schema introspection queries
3. Add async message queuing for broadcasts
4. Implement cursor-based pagination

**Long-term (architecture changes):**
1. Separate real-time service for horizontal scaling
2. Dedicated image transformation service
3. Event sourcing for audit logs
4. Read replica support for query offloading

---

## 6. Observability, Operations, and Deployment

### 6.1 Logging Assessment

**Strengths:**
- Structured logging via zerolog (JSON output)
- Multi-category system (system, HTTP, security, execution, AI, custom)
- Async batching with configurable flush intervals
- Retention policies per category
- PubSub notifications for real-time streaming

**Gaps:**
- No request ID propagation to all log entries
- No log sampling for high-volume endpoints
- No automatic PII redaction

**Recommendation:** Add correlation ID middleware that injects request ID into all log entries.

### 6.2 Metrics Assessment

**Comprehensive Coverage (70+ metrics):**
- HTTP: request rate, latency histograms, response sizes, status codes
- Database: query counts, latency, connection pool stats
- Realtime: connections, channels, subscriptions, message throughput
- Storage: bytes transferred, operation latency by bucket
- Auth: attempt counts, success/failure by type
- AI: request duration, token usage, provider latency

**Missing Metrics:**
- Job queue depth by namespace/priority
- Worker utilization percentage
- Function execution duration distribution
- Cache hit rates (schema cache, RLS cache)
- Rate limiting rejection counts

### 6.3 Tracing Assessment

**Implementation:**
- Full OpenTelemetry integration with OTLP gRPC exporter
- Configurable sampling (AlwaysSample, TraceIDRatioBased, ParentBased)
- W3C Trace Context propagation
- Helper functions for DB, storage, auth spans

**Coverage Gaps:**
- Edge function execution not traced
- Background job processing not traced
- Real-time message flow not traced

**Recommendation:** Add spans for:
- `StartFunctionSpan()` - Deno execution tracing
- `StartJobSpan()` - Background job processing
- `StartRealtimeSpan()` - Message broadcast latency

### 6.4 Deployment Assessment

**Docker (Excellent):**
- Multi-stage build (Deno, Node.js/Admin, Go, Runtime)
- Non-root user (UID 1000)
- Health checks with startup probes
- Volume mounts for persistence

**Kubernetes/Helm (Excellent):**
- 18 templates covering all common patterns
- HorizontalPodAutoscaler support
- Multiple ingress options (Ingress, HTTPRoute, Gateway API)
- ServiceMonitor for Prometheus integration
- Resource presets (nano through 2xlarge)

**Self-Hosting Ergonomics:**
- Single `docker compose up` for full stack
- Comprehensive environment variable configuration
- Database migration via init containers
- Health checks for all services

**Gaps:**
- No Terraform/CloudFormation templates for cloud deployments
- No GitOps examples (ArgoCD, Flux)
- No backup/restore procedures documented
- No runbook for common operational issues

### 6.5 Operational Readiness Assessment

| Aspect | Rating | Notes |
|--------|--------|-------|
| **Development/Demo** | ✅ Ready | Single command startup, good DX |
| **Small Production** | ✅ Ready | <1000 users, single instance |
| **Medium Production** | ⚠️ Conditional | Requires Redis, tuning, monitoring |
| **Large Production** | ❌ Gaps | Scalability bottlenecks need addressing |

### 6.6 Prioritized Operations Improvements

1. **Add correlation IDs to all logs** (High)
2. **Create operational runbook** (High)
3. **Add job queue depth metrics** (Medium)
4. **Document backup procedures** (Medium)
5. **Add tracing to functions/jobs** (Medium)
6. **Create Grafana dashboard presets** (Low)
7. **Add GitOps deployment examples** (Low)

---

## 7. Concrete Next Steps Roadmap

### High Priority (Now) — Security, Correctness, Reliability

| # | Title | Category | Why It Matters | Where to Start |
|---|-------|----------|----------------|----------------|
| 1 | **Make TOTP encryption mandatory** | Security | Plaintext TOTP secrets in DB = account compromise on breach | `internal/auth/totp.go` - Add startup validation |
| 2 | **Add per-user TOTP rate limiting** | Security | 6-digit codes brute-forceable without limits | `internal/auth/service.go:820-1050` |
| 3 | **Enable distributed rate limiting by default** | Security | Per-instance limits trivially bypassed | `internal/middleware/rate_limiter.go` - Use Redis backend |
| 4 | **Fix progress update context leak** | Correctness | Uses `context.Background()`, orphans updates on cancellation | `internal/jobs/worker.go:515` |
| 5 | **Add per-user WebSocket connection limits** | Reliability | Single client can open unlimited connections | `internal/realtime/manager.go` |
| 6 | **Fix file descriptor leak in function runtime** | Reliability | Pipes not closed on goroutine panic | `internal/runtime/runtime.go:266-274` |
| 7 | **Increase RLS cache size** | Performance | 10K entries insufficient for high-throughput realtime | `internal/realtime/` - Make configurable |
| 8 | **Add correlation IDs to error responses** | Operations | Cannot correlate errors to logs | `internal/api/` - Middleware extraction |

### Medium Priority (Soon) — Maintainability, Scalability

| # | Title | Category | Why It Matters | Where to Start |
|---|-------|----------|----------------|----------------|
| 9 | **Standardize error response format** | DX | Inconsistent error structures confuse SDK developers | `internal/api/rest_errors.go` |
| 10 | **Batch schema introspection queries** | Scalability | N+1 pattern: 5N queries for N tables | `internal/database/schema_cache.go` |
| 11 | **Implement LISTEN connection pooling** | Scalability | Single connection bottleneck for realtime | `internal/realtime/listener.go` |
| 12 | **Add async message broadcasting** | Scalability | RWMutex held during broadcast blocks other operations | `internal/realtime/manager.go` |
| 13 | **Extract unified filter parser** | Maintainability | Filter logic duplicated across REST, GraphQL, realtime | Create `internal/query/filter_parser.go` |
| 14 | **Add column validation caching** | Performance | O(n) lookup per field on every request | `internal/api/rest_crud.go` |
| 15 | **Implement graceful job shutdown** | Correctness | Hard 30-second timeout kills long-running jobs | `internal/jobs/worker.go:120-154` |
| 16 | **Add streaming result parsing for functions** | Scalability | 1MB buffer per line can OOM on large results | `internal/runtime/runtime.go:296` |
| 17 | **Document backup and restore procedures** | Operations | No documented recovery process | Create `docs/operations/backup.md` |
| 18 | **Add tracing to functions and jobs** | Operations | Blind spots in distributed tracing | `internal/functions/handler.go`, `internal/jobs/worker.go` |
| 19 | **Increase default connection pool** | Scalability | 25 connections insufficient for concurrent workloads | `internal/config/config.go` |
| 20 | **Add request/response size limits per endpoint** | Security | Global limits only; no per-endpoint control | `internal/api/server.go` |
| 21 | **Implement idempotency keys for mutations** | Correctness | POST requests not safe to retry | `internal/api/rest_crud.go` |
| 22 | **Add OAuth state persistence for multi-instance** | Correctness | In-memory state breaks with load balancing | `internal/auth/oauth.go` |
| 23 | **Auto-disconnect slow WebSocket clients** | Reliability | Slow clients tracked but not acted upon | `internal/realtime/manager.go` |

### Nice to Have (Later) — Polish, DX, Tooling

| # | Title | Category | Why It Matters | Where to Start |
|---|-------|----------|----------------|----------------|
| 24 | **Generate TypeScript types from schema** | DX | Manual type definitions error-prone | `sdk/` - Add codegen script |
| 25 | **Add cursor-based pagination** | DX | Offset pagination inefficient for large datasets | `internal/api/query_parser.go` |
| 26 | **Implement conditional requests (ETags)** | Performance | No client-side caching support | `internal/api/rest_crud.go` |
| 27 | **Add GraphQL subscriptions** | Feature | Realtime via separate WebSocket only | `internal/api/graphql_handler.go` |
| 28 | **Create Terraform modules** | Operations | No cloud deployment templates | Create `deploy/terraform/` |
| 29 | **Add load testing benchmarks** | Quality | No performance baselines | Create `test/bench/` |
| 30 | **Implement webhook request signing** | Security | Webhooks lack verification mechanism | `internal/webhook/` |
| 31 | **Add Have I Been Pwned password check** | Security | No compromised password detection | `internal/auth/password.go` |
| 32 | **Create operational runbook** | Operations | No incident response documentation | Create `docs/operations/runbook.md` |
| 33 | **Add API versioning headers** | DX | No version negotiation support | `internal/middleware/` |
| 34 | **Implement service role token revocation** | Security | Cannot emergency-revoke service keys | `internal/auth/token_blacklist.go` |
| 35 | **Add chaos testing framework** | Quality | No resilience verification | Create `test/chaos/` |

---

## Appendix A: File Reference

Key files for each priority item:

```
# High Priority
internal/auth/totp.go                    # TOTP encryption
internal/auth/service.go                 # Auth rate limiting
internal/middleware/rate_limiter.go      # Distributed rate limiting
internal/jobs/worker.go                  # Job context handling
internal/realtime/manager.go             # WebSocket limits
internal/runtime/runtime.go              # Function FD leak

# Medium Priority
internal/api/rest_errors.go              # Error standardization
internal/database/schema_cache.go        # Schema N+1
internal/realtime/listener.go            # LISTEN pooling
internal/api/rest_crud.go                # Column caching
internal/config/config.go                # Connection pool defaults

# Entry Points
cmd/fluxbase/main.go                     # Server startup
internal/api/server.go                   # Route registration
```

---

## Appendix B: Deployment Checklist

### Pre-Production
- [ ] Generate JWT secrets (`scripts/generate-keys.sh`)
- [ ] Configure PostgreSQL credentials
- [ ] Enable Redis/Dragonfly for distributed state
- [ ] Set appropriate resource limits
- [ ] Configure Prometheus scraping
- [ ] Set up TLS termination
- [ ] Enable audit logging

### Production Monitoring
- [ ] Grafana dashboards deployed
- [ ] Alert rules configured (error rate, latency, pool exhaustion)
- [ ] Log aggregation configured
- [ ] Backup schedule established
- [ ] Incident response procedures documented

---

*Review completed. This document should be updated as issues are addressed and new findings emerge.*
