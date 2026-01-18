# Fluxbase Implementation Plan

**Created:** 2026-01-18
**Based on:** ARCHITECTURE_REVIEW.md
**Status:** In Progress

---

## Overview

This document tracks the implementation of improvements identified in the architecture review. Items are organized into phases by priority, with each item including implementation details, test requirements, and progress tracking.

**Excluded from this plan:**
- Terraform modules (not needed currently)
- HIBP password checks (deferred)
- API versioning headers (unnecessary for self-hosted BaaS)
- Chaos testing framework (focus on unit/integration tests first)

---

## Progress Summary

| Phase | Total Items | Completed | In Progress | Remaining |
|-------|-------------|-----------|-------------|-----------|
| Phase 1: Critical Security & Reliability | 8 | 2 | 0 | 6 |
| Phase 2: Scalability & Performance | 8 | 0 | 0 | 8 |
| Phase 3: Maintainability & Correctness | 7 | 0 | 0 | 7 |
| Phase 4: Developer Experience | 5 | 0 | 0 | 5 |
| Phase 5: Operations & Polish | 4 | 0 | 0 | 4 |
| **Total** | **32** | **2** | **0** | **30** |

---

## Phase 1: Critical Security & Reliability

These items address security vulnerabilities and reliability issues that could cause data loss or security breaches.

### 1.1 Wire Up TOTP Encryption Using Global Encryption Key

**Priority:** Critical
**Category:** Security
**Status:** [x] Complete

**Problem:**
TOTP secrets are stored in plaintext because `authService.SetEncryptionKey()` is never called. The global `cfg.EncryptionKey` (already required at startup) should be used.

**Root Cause:**
- `cfg.EncryptionKey` exists and is already validated (32 bytes, required)
- It's used for secrets storage, OAuth config, custom settings
- But `authService.SetEncryptionKey()` is never called in `server.go`
- Result: TOTP secrets stored unencrypted

**Files to Modify:**
- `internal/api/server.go` (add one line after auth service creation)
- `internal/auth/service.go` (remove fallback to plaintext, make encryption required)

**Implementation Steps:**
- [x] In `server.go`, add `authService.SetEncryptionKey(cfg.EncryptionKey)` after line 192
- [x] In `service.go`, remove the `if s.encryptionKey != ""` conditional - always require encryption
- [x] Log warning during migration for secrets that couldn't be decrypted (already plaintext)
- [ ] Add migration to re-encrypt any existing plaintext TOTP secrets (deferred - backward compat handles this)

**Test Requirements:**
- [x] Unit test: TOTP enrollment encrypts secret with provided key
- [x] Unit test: TOTP verification decrypts secret correctly
- [x] Unit test: Missing encryption key returns error (not plaintext fallback)
- [ ] Integration test: Full TOTP flow with encryption (requires DB)

**Test File:** `internal/auth/service_test.go`

---

### 1.2 Add Per-User TOTP Rate Limiting

**Priority:** Critical
**Category:** Security
**Status:** [x] Complete

**Problem:**
6-digit TOTP codes (1M combinations) can be brute-forced without per-user rate limiting.

**Files Modified:**
- `internal/auth/totp_rate_limiter.go` (new file)
- `internal/auth/service.go` (added rate limiter integration)
- `internal/api/server.go` (wire up rate limiter)

**Implementation Steps:**
- [x] Reuse existing `auth.two_factor_recovery_attempts` table (already has timestamp and success columns)
- [x] Add `TOTPRateLimiter` struct with configurable limits (default: 5 attempts per 5 minutes)
- [x] Integrate rate check before TOTP verification in `VerifyTOTP()`
- [x] Return `ErrTOTPRateLimitExceeded` when limit exceeded
- [x] Add lockout duration configuration option (default: 15 minutes)
- [x] Record attempts for rate limiting (success clears counter effectively)
- [x] Add `ClearFailedAttempts()` method for admin use

**Test Requirements:**
- [x] Unit test: Default config values
- [x] Unit test: Custom config values
- [x] Unit test: Negative config values use defaults
- [x] Unit test: Helper functions
- [ ] Integration test: Full flow with rate limiting (requires DB)
- [ ] Integration test: Concurrent attempts handled correctly (requires DB)

**Test File:** `internal/auth/totp_rate_limiter_test.go`

---

### 1.3 Enable Distributed Rate Limiting by Default

**Priority:** Critical
**Category:** Security
**Status:** [ ] Not Started

**Problem:**
Per-instance rate limiting is bypassed in multi-instance deployments; attackers can target different instances.

**Files to Modify:**
- `internal/middleware/rate_limiter.go`
- `internal/config/config.go`
- `docs/` (configuration documentation)

**Implementation Steps:**
- [ ] Add startup warning when using in-memory rate limiter with `instances > 1` or in Kubernetes
- [ ] Document Redis/Dragonfly requirement for production deployments
- [ ] Add configuration validation that warns about in-memory limiter risks
- [ ] Consider making Redis backend the default when Redis URL is configured
- [ ] Add metrics for rate limit backend type

**Test Requirements:**
- [ ] Unit test: Warning logged for multi-instance + memory backend
- [ ] Unit test: Redis backend properly distributes limits
- [ ] Unit test: Fallback to memory backend when Redis unavailable (with warning)
- [ ] Integration test: Rate limits shared across simulated instances

**Test File:** `internal/middleware/rate_limiter_test.go`

---

### 1.4 Fix Progress Update Context Leak

**Priority:** Critical
**Category:** Correctness
**Status:** [ ] Not Started

**Problem:**
Job progress updates use `context.Background()` instead of job context, orphaning updates when job is cancelled.

**Files to Modify:**
- `internal/jobs/worker.go` (line 515)

**Implementation Steps:**
- [ ] Replace `context.Background()` with job execution context
- [ ] Add timeout for progress updates (prevent blocking on slow DB)
- [ ] Handle context cancellation gracefully in progress update
- [ ] Log warning if progress update fails due to cancellation

**Test Requirements:**
- [ ] Unit test: Progress updates use correct context
- [ ] Unit test: Cancelled job stops progress updates
- [ ] Unit test: Progress update timeout prevents blocking
- [ ] Integration test: Job cancellation doesn't leave orphaned state

**Test File:** `internal/jobs/worker_test.go`

---

### 1.5 Add Per-User WebSocket Connection Limits

**Priority:** High
**Category:** Reliability
**Status:** [ ] Not Started

**Problem:**
Single client can open unlimited WebSocket connections, exhausting server resources.

**Files to Modify:**
- `internal/realtime/manager.go`
- `internal/config/config.go`

**Implementation Steps:**
- [ ] Add `max_connections_per_user` config option (default: 10)
- [ ] Track connection count per user ID in manager
- [ ] Reject new connections when limit exceeded (close with 1008 Policy Violation)
- [ ] Add `max_connections_per_ip` for anonymous connections
- [ ] Add metrics for connection rejections

**Test Requirements:**
- [ ] Unit test: Connections under limit accepted
- [ ] Unit test: Connections over limit rejected with proper code
- [ ] Unit test: Connection count decremented on disconnect
- [ ] Unit test: Anonymous connections limited by IP
- [ ] Integration test: Rapid connection attempts handled correctly

**Test File:** `internal/realtime/manager_test.go`

---

### 1.6 Fix File Descriptor Leak in Function Runtime

**Priority:** High
**Category:** Reliability
**Status:** [ ] Not Started

**Problem:**
Pipes not closed on goroutine panic in function runtime, leaking file descriptors.

**Files to Modify:**
- `internal/runtime/runtime.go` (lines 266-274)

**Implementation Steps:**
- [ ] Add deferred pipe cleanup with recover() in output reading goroutines
- [ ] Ensure stdout/stderr pipes closed in all exit paths
- [ ] Add FD tracking metrics for debugging
- [ ] Consider using errgroup for coordinated goroutine cleanup

**Test Requirements:**
- [ ] Unit test: Pipes closed on normal completion
- [ ] Unit test: Pipes closed on function timeout
- [ ] Unit test: Pipes closed on panic recovery
- [ ] Unit test: No FD leak under stress test (100 rapid invocations)

**Test File:** `internal/runtime/runtime_test.go`

---

### 1.7 Increase and Configure RLS Cache Size

**Priority:** High
**Category:** Performance
**Status:** [ ] Not Started

**Problem:**
10K entry RLS cache insufficient for high-throughput realtime; cache misses cause DB roundtrips.

**Files to Modify:**
- `internal/realtime/manager.go` or `internal/realtime/rls_cache.go`
- `internal/config/config.go`

**Implementation Steps:**
- [ ] Add `realtime.rls_cache_size` config option (default: 100000)
- [ ] Add `realtime.rls_cache_ttl` config option (default: 5m)
- [ ] Add cache hit/miss metrics
- [ ] Add cache eviction metrics
- [ ] Consider LRU vs LFU eviction strategy

**Test Requirements:**
- [ ] Unit test: Cache respects configured size
- [ ] Unit test: Cache evicts oldest entries when full
- [ ] Unit test: Cache TTL expires entries correctly
- [ ] Benchmark: Cache performance at 100K entries

**Test File:** `internal/realtime/rls_cache_test.go`

---

### 1.8 Add Correlation IDs to Error Responses

**Priority:** High
**Category:** Operations
**Status:** [ ] Not Started

**Problem:**
Cannot correlate client errors to server logs for debugging.

**Files to Modify:**
- `internal/middleware/request_id.go` (new file)
- `internal/api/server.go`
- `internal/api/rest_errors.go`

**Implementation Steps:**
- [ ] Create request ID middleware that generates/extracts X-Request-ID header
- [ ] Store request ID in fiber context locals
- [ ] Include request ID in all error responses
- [ ] Include request ID in all log entries for the request
- [ ] Return request ID in response header

**Test Requirements:**
- [ ] Unit test: Request ID generated when not provided
- [ ] Unit test: Request ID extracted from header when provided
- [ ] Unit test: Request ID included in error responses
- [ ] Unit test: Request ID propagated to logs
- [ ] Integration test: Full request traced by ID

**Test File:** `internal/middleware/request_id_test.go`

---

## Phase 2: Scalability & Performance

These items address bottlenecks that will impact performance as usage grows.

### 2.1 Batch Schema Introspection Queries

**Priority:** High
**Category:** Scalability
**Status:** [ ] Not Started

**Problem:**
N+1 query pattern: 5N queries for N tables during schema cache refresh.

**Files to Modify:**
- `internal/database/schema_cache.go` (lines 66-119)
- `internal/database/schema_inspector.go`

**Implementation Steps:**
- [ ] Combine column queries into single query with table filter
- [ ] Combine primary key queries into single query
- [ ] Combine foreign key queries into single query
- [ ] Combine index queries into single query
- [ ] Group results by table in Go code
- [ ] Add query timing metrics

**Test Requirements:**
- [ ] Unit test: Batch query returns same results as individual queries
- [ ] Unit test: Empty table set handled correctly
- [ ] Unit test: Large table count (500+) handled efficiently
- [ ] Benchmark: Compare query count before/after (should be O(1) vs O(N))

**Test File:** `internal/database/schema_cache_test.go`

---

### 2.2 Implement LISTEN Connection Pooling

**Priority:** High
**Category:** Scalability
**Status:** [ ] Not Started

**Problem:**
Single PostgreSQL LISTEN connection is bottleneck for realtime subscriptions.

**Files to Modify:**
- `internal/realtime/listener.go`
- `internal/config/config.go`

**Implementation Steps:**
- [ ] Add `realtime.listener_pool_size` config option (default: 4)
- [ ] Create pool of LISTEN connections
- [ ] Distribute channel subscriptions across pool (consistent hashing)
- [ ] Handle connection failures with automatic reconnection
- [ ] Add pool health metrics

**Test Requirements:**
- [ ] Unit test: Connections distributed across pool
- [ ] Unit test: Channel consistently routes to same connection
- [ ] Unit test: Failed connection triggers reconnection
- [ ] Unit test: Subscriptions rebalanced on pool resize
- [ ] Load test: Compare throughput with 1 vs 4 connections

**Test File:** `internal/realtime/listener_test.go`

---

### 2.3 Add Async Message Broadcasting

**Priority:** High
**Category:** Scalability
**Status:** [ ] Not Started

**Problem:**
RWMutex held during WebSocket broadcast blocks other operations.

**Files to Modify:**
- `internal/realtime/manager.go`

**Implementation Steps:**
- [ ] Add per-client message queue (buffered channel)
- [ ] Broadcast adds to queues without holding global lock
- [ ] Dedicated goroutine per client drains queue
- [ ] Add queue depth metrics
- [ ] Handle slow clients (drop messages or disconnect)

**Test Requirements:**
- [ ] Unit test: Messages queued without blocking
- [ ] Unit test: Slow client doesn't block other clients
- [ ] Unit test: Queue overflow handled gracefully
- [ ] Unit test: Client disconnect drains queue
- [ ] Load test: Broadcast latency under high connection count

**Test File:** `internal/realtime/manager_test.go`

---

### 2.4 Add Column Validation Cache

**Priority:** Medium
**Category:** Performance
**Status:** [ ] Not Started

**Problem:**
O(n) column lookup per field on every request.

**Files to Modify:**
- `internal/api/rest_crud.go`
- `internal/database/schema_cache.go`

**Implementation Steps:**
- [ ] Add `ColumnExists(schema, table, column) bool` method to schema cache
- [ ] Use map lookup instead of slice iteration
- [ ] Invalidate on schema cache refresh
- [ ] Add cache hit metrics

**Test Requirements:**
- [ ] Unit test: Column existence check returns correct result
- [ ] Unit test: Invalid column rejected
- [ ] Unit test: Cache invalidated on schema change
- [ ] Benchmark: Compare lookup time O(1) vs O(n)

**Test File:** `internal/database/schema_cache_test.go`

---

### 2.5 Increase Default Connection Pool

**Priority:** Medium
**Category:** Scalability
**Status:** [ ] Not Started

**Problem:**
25 max connections insufficient for concurrent workloads (jobs + API + realtime).

**Files to Modify:**
- `internal/config/config.go`
- `docs/` (configuration documentation)

**Implementation Steps:**
- [ ] Increase default `database.max_connections` from 25 to 50
- [ ] Add guidance for sizing based on instance count
- [ ] Add connection pool exhaustion alerting recommendation
- [ ] Document connection requirements per feature (API, jobs, realtime)

**Test Requirements:**
- [ ] Unit test: Config accepts new default
- [ ] Integration test: Pool handles 50 concurrent connections

**Test File:** `internal/config/config_test.go`

---

### 2.6 Add Streaming Result Parsing for Functions

**Priority:** Medium
**Category:** Scalability
**Status:** [ ] Not Started

**Problem:**
1MB buffer per line in function output can OOM on large results.

**Files to Modify:**
- `internal/runtime/runtime.go` (line 296)

**Implementation Steps:**
- [ ] Replace fixed buffer with streaming JSON parser
- [ ] Add `functions.max_output_size` config (default: 10MB)
- [ ] Truncate output with warning when limit exceeded
- [ ] Add output size metrics

**Test Requirements:**
- [ ] Unit test: Small output parsed correctly
- [ ] Unit test: Large output truncated with warning
- [ ] Unit test: Malformed output handled gracefully
- [ ] Unit test: Memory usage bounded during parsing

**Test File:** `internal/runtime/runtime_test.go`

---

### 2.7 Implement Cursor-Based Pagination

**Priority:** Medium
**Category:** Performance
**Status:** [ ] Not Started

**Problem:**
Offset pagination inefficient for large datasets; performance degrades linearly.

**Files to Modify:**
- `internal/api/query_parser.go`
- `internal/api/query_builder.go`
- `internal/api/rest_crud.go`
- `sdk/src/` (TypeScript SDK)

**Implementation Steps:**
- [ ] Add `cursor` query parameter (base64 encoded last row identifier)
- [ ] Add `cursor_column` parameter (default: primary key)
- [ ] Implement keyset pagination in query builder
- [ ] Return `next_cursor` in response headers
- [ ] Update SDK with cursor pagination support
- [ ] Document cursor vs offset trade-offs

**Test Requirements:**
- [ ] Unit test: Cursor decoded correctly
- [ ] Unit test: Query uses keyset condition
- [ ] Unit test: Next cursor generated correctly
- [ ] Unit test: Invalid cursor returns 400
- [ ] Integration test: Full pagination through dataset
- [ ] Benchmark: Compare performance at offset 10K vs cursor

**Test File:** `internal/api/query_parser_test.go`, `internal/api/query_builder_test.go`

---

### 2.8 Auto-Disconnect Slow WebSocket Clients

**Priority:** Medium
**Category:** Reliability
**Status:** [ ] Not Started

**Problem:**
Slow clients are tracked but not acted upon; they accumulate and waste resources.

**Files to Modify:**
- `internal/realtime/manager.go`

**Implementation Steps:**
- [ ] Add `realtime.slow_client_threshold` config (default: 100 pending messages)
- [ ] Add `realtime.slow_client_timeout` config (default: 30s)
- [ ] Disconnect clients exceeding threshold for timeout duration
- [ ] Send close frame with 1008 Policy Violation before disconnect
- [ ] Add slow client disconnect metrics

**Test Requirements:**
- [ ] Unit test: Client below threshold not disconnected
- [ ] Unit test: Client above threshold for duration disconnected
- [ ] Unit test: Client recovering before timeout not disconnected
- [ ] Unit test: Disconnect sends proper close frame

**Test File:** `internal/realtime/manager_test.go`

---

## Phase 3: Maintainability & Correctness

These items improve code quality and fix correctness issues.

### 3.1 Standardize Error Response Format

**Priority:** High
**Category:** Developer Experience
**Status:** [ ] Not Started

**Problem:**
Inconsistent error structures confuse SDK developers and complicate client error handling.

**Files to Modify:**
- `internal/api/rest_errors.go`
- `internal/api/*.go` (all handlers)

**Implementation Steps:**
- [ ] Define standard `APIError` struct with code, message, details, request_id
- [ ] Create error response helper: `SendError(c, statusCode, errorCode, message, details)`
- [ ] Migrate all handlers to use helper
- [ ] Document error codes in OpenAPI spec
- [ ] Update SDK to parse structured errors

**Test Requirements:**
- [ ] Unit test: Error helper produces correct format
- [ ] Unit test: All error codes documented
- [ ] Integration test: Various error scenarios return consistent format
- [ ] SDK test: Error parsing works for all error types

**Test File:** `internal/api/rest_errors_test.go`

---

### 3.2 Extract Unified Filter Parser

**Priority:** Medium
**Category:** Maintainability
**Status:** [ ] Not Started

**Problem:**
Filter logic duplicated across REST, GraphQL, and realtime handlers.

**Files to Modify:**
- Create `internal/query/filter_parser.go`
- `internal/api/query_parser.go`
- `internal/api/graphql_handler.go`
- `internal/realtime/subscription.go`

**Implementation Steps:**
- [ ] Extract `FilterParser` interface and implementation
- [ ] Support both PostgREST and structured filter formats
- [ ] Migrate REST handler to use unified parser
- [ ] Migrate GraphQL handler to use unified parser
- [ ] Migrate realtime subscription to use unified parser
- [ ] Add comprehensive operator validation

**Test Requirements:**
- [ ] Unit test: All filter operators parsed correctly
- [ ] Unit test: Invalid operators return descriptive error
- [ ] Unit test: Nested logical groups handled
- [ ] Unit test: Format compatibility (PostgREST vs structured)
- [ ] Integration test: Same filter works across REST, GraphQL, realtime

**Test File:** `internal/query/filter_parser_test.go`

---

### 3.3 Implement Graceful Job Shutdown

**Priority:** Medium
**Category:** Correctness
**Status:** [ ] Not Started

**Problem:**
Hard 30-second timeout kills long-running jobs without cleanup.

**Files to Modify:**
- `internal/jobs/worker.go` (lines 120-154)
- `internal/jobs/manager.go`

**Implementation Steps:**
- [ ] Add `jobs.graceful_shutdown_timeout` config (default: 5m)
- [ ] On shutdown signal, stop accepting new jobs
- [ ] Wait for running jobs to complete up to timeout
- [ ] Mark incomplete jobs as "interrupted" not "failed"
- [ ] Add job interrupt handling callback for cleanup

**Test Requirements:**
- [ ] Unit test: Shutdown waits for running jobs
- [ ] Unit test: New jobs rejected during shutdown
- [ ] Unit test: Timeout forces termination
- [ ] Unit test: Interrupted jobs marked correctly
- [ ] Integration test: Full graceful shutdown flow

**Test File:** `internal/jobs/worker_test.go`

---

### 3.4 Implement Idempotency Keys for Mutations

**Priority:** Medium
**Category:** Correctness
**Status:** [ ] Not Started

**Problem:**
POST requests not safe to retry; network failures can cause duplicate operations.

**Files to Modify:**
- `internal/middleware/idempotency.go` (new file)
- `internal/api/server.go`
- `internal/database/migrations/` (new migration)

**Implementation Steps:**
- [ ] Add `Idempotency-Key` header support
- [ ] Create `idempotency_keys` table (key, response, expires_at)
- [ ] Check for existing key before processing request
- [ ] Store response on completion
- [ ] Return cached response for duplicate keys
- [ ] Add TTL for key expiration (default: 24h)

**Test Requirements:**
- [ ] Unit test: Request without key processed normally
- [ ] Unit test: First request with key processed and cached
- [ ] Unit test: Duplicate request returns cached response
- [ ] Unit test: Expired keys allow new requests
- [ ] Integration test: Concurrent duplicate requests handled correctly

**Test File:** `internal/middleware/idempotency_test.go`

---

### 3.5 Add OAuth State Persistence for Multi-Instance

**Priority:** Medium
**Category:** Correctness
**Status:** [ ] Not Started

**Problem:**
In-memory OAuth state breaks with load balancing; callback may hit different instance.

**Files to Modify:**
- `internal/auth/oauth.go`
- `internal/database/migrations/` (new migration)

**Implementation Steps:**
- [ ] Create `auth.oauth_states` table (state, provider, redirect_uri, expires_at)
- [ ] Store state in database instead of memory
- [ ] Validate state from database on callback
- [ ] Delete state after use (prevent replay)
- [ ] Add cleanup job for expired states

**Test Requirements:**
- [ ] Unit test: State stored in database
- [ ] Unit test: Valid state accepted on callback
- [ ] Unit test: Invalid state rejected
- [ ] Unit test: Used state cannot be replayed
- [ ] Integration test: OAuth flow across "different instances"

**Test File:** `internal/auth/oauth_test.go`

---

### 3.6 Add Request/Response Size Limits Per Endpoint

**Priority:** Medium
**Category:** Security
**Status:** [ ] Not Started

**Problem:**
Global limits only; no per-endpoint control for different use cases.

**Files to Modify:**
- `internal/middleware/body_limit.go` (new file)
- `internal/api/server.go`
- `internal/config/config.go`

**Implementation Steps:**
- [ ] Add configurable limits per route pattern
- [ ] Default limits by endpoint type (REST: 1MB, Upload: 100MB, etc.)
- [ ] Add JSON depth limiting to prevent stack overflow
- [ ] Return 413 Payload Too Large with clear message

**Test Requirements:**
- [ ] Unit test: Requests under limit accepted
- [ ] Unit test: Requests over limit rejected with 413
- [ ] Unit test: Different endpoints have different limits
- [ ] Unit test: Deeply nested JSON rejected

**Test File:** `internal/middleware/body_limit_test.go`

---

### 3.7 Add Service Role Token Revocation

**Priority:** Low
**Category:** Security
**Status:** [ ] Not Started

**Problem:**
Cannot emergency-revoke compromised service keys.

**Files to Modify:**
- `internal/auth/token_blacklist.go`
- `internal/auth/service.go`

**Implementation Steps:**
- [ ] Add optional service role to blacklist check
- [ ] Add admin endpoint to revoke service role tokens
- [ ] Add `service_keys` table for tracking issued keys
- [ ] Support key rotation with grace period

**Test Requirements:**
- [ ] Unit test: Non-revoked service token accepted
- [ ] Unit test: Revoked service token rejected
- [ ] Unit test: Key rotation grace period works
- [ ] Integration test: Full revocation flow

**Test File:** `internal/auth/token_blacklist_test.go`

---

## Phase 4: Developer Experience

These items improve the experience for developers using Fluxbase.

### 4.1 Generate TypeScript Types from Schema

**Priority:** High
**Category:** Developer Experience
**Status:** [ ] Not Started

**Problem:**
Manual type definitions error-prone and quickly outdated.

**Files to Modify:**
- Create `sdk/scripts/generate-types.ts`
- `internal/api/schema_export.go` (new endpoint)

**Implementation Steps:**
- [ ] Add `/api/v1/schema/typescript` endpoint returning TypeScript definitions
- [ ] Generate types for all tables with column types
- [ ] Generate types for RPC functions
- [ ] Add CLI command: `fluxbase types generate`
- [ ] Document type generation workflow

**Test Requirements:**
- [ ] Unit test: Type generation produces valid TypeScript
- [ ] Unit test: All PostgreSQL types mapped correctly
- [ ] Unit test: Nullable columns marked optional
- [ ] Integration test: Generated types compile without errors

**Test File:** `internal/api/schema_export_test.go`, `sdk/scripts/generate-types.test.ts`

---

### 4.2 Add Conditional Requests (ETags)

**Priority:** Medium
**Category:** Performance
**Status:** [ ] Not Started

**Problem:**
No client-side caching support; clients always fetch full response.

**Files to Modify:**
- `internal/api/rest_crud.go`
- `internal/middleware/etag.go` (new file)

**Implementation Steps:**
- [ ] Calculate ETag from response content hash
- [ ] Add `ETag` header to GET responses
- [ ] Check `If-None-Match` header on requests
- [ ] Return 304 Not Modified when ETag matches
- [ ] Add `Last-Modified` header using row timestamps if available

**Test Requirements:**
- [ ] Unit test: ETag generated for responses
- [ ] Unit test: Matching If-None-Match returns 304
- [ ] Unit test: Non-matching If-None-Match returns full response
- [ ] Unit test: Last-Modified header set when available

**Test File:** `internal/middleware/etag_test.go`

---

### 4.3 Add GraphQL Subscriptions

**Priority:** Medium
**Category:** Feature
**Status:** [ ] Not Started

**Problem:**
Realtime only available via separate WebSocket API; GraphQL users expect subscriptions.

**Files to Modify:**
- `internal/api/graphql_handler.go`
- `internal/api/graphql_subscription.go` (new file)

**Implementation Steps:**
- [ ] Implement GraphQL WebSocket protocol (graphql-ws)
- [ ] Map subscriptions to PostgreSQL LISTEN/NOTIFY
- [ ] Support subscription filters
- [ ] Integrate with existing realtime infrastructure
- [ ] Add subscription depth limiting

**Test Requirements:**
- [ ] Unit test: Subscription connection established
- [ ] Unit test: Subscription receives database changes
- [ ] Unit test: Subscription filters applied correctly
- [ ] Unit test: Subscription disconnection cleanup
- [ ] Integration test: Full subscription lifecycle

**Test File:** `internal/api/graphql_subscription_test.go`

---

### 4.4 Add Webhook Request Signing

**Priority:** Medium
**Category:** Security
**Status:** [ ] Not Started

**Problem:**
Webhooks lack verification mechanism; recipients can't verify authenticity.

**Files to Modify:**
- `internal/webhook/sender.go`
- `internal/config/config.go`

**Implementation Steps:**
- [ ] Add `X-Fluxbase-Signature` header (HMAC-SHA256)
- [ ] Include timestamp in signature to prevent replay
- [ ] Add per-webhook secret configuration
- [ ] Document signature verification for webhook consumers
- [ ] Add SDK helper for signature verification

**Test Requirements:**
- [ ] Unit test: Signature generated correctly
- [ ] Unit test: Timestamp included in signature
- [ ] Unit test: Different secrets produce different signatures
- [ ] SDK test: Verification helper works correctly

**Test File:** `internal/webhook/sender_test.go`

---

### 4.5 Bulk Operation Response Counts

**Priority:** Low
**Category:** Developer Experience
**Status:** [ ] Not Started

**Problem:**
Batch DELETE/PATCH doesn't return affected count; clients can't verify operation success.

**Files to Modify:**
- `internal/api/rest_batch.go`
- `internal/api/rest_crud.go`

**Implementation Steps:**
- [ ] Add `Prefer: return=representation` header support for counts
- [ ] Return `{ affected: number }` for batch operations
- [ ] Add `X-Affected-Count` header as alternative
- [ ] Document batch operation responses

**Test Requirements:**
- [ ] Unit test: Affected count returned correctly
- [ ] Unit test: Zero affected handled (may indicate RLS)
- [ ] Unit test: Header and body both return count

**Test File:** `internal/api/rest_batch_test.go`

---

## Phase 5: Operations & Polish

These items improve operational capabilities.

### 5.1 Document Backup and Restore Procedures

**Priority:** High
**Category:** Operations
**Status:** [ ] Not Started

**Problem:**
No documented recovery process for disasters.

**Files to Create:**
- `docs/operations/backup.md`
- `scripts/backup.sh`
- `scripts/restore.sh`

**Implementation Steps:**
- [ ] Document PostgreSQL backup strategies (pg_dump, WAL archiving)
- [ ] Document storage backup (S3 versioning, local rsync)
- [ ] Create backup script with configurable retention
- [ ] Create restore script with verification
- [ ] Document point-in-time recovery
- [ ] Add backup verification checklist

**Test Requirements:**
- [ ] Manual test: Backup script creates valid backup
- [ ] Manual test: Restore script recovers data
- [ ] Manual test: Partial restore works

---

### 5.2 Add Tracing to Functions and Jobs

**Priority:** Medium
**Category:** Operations
**Status:** [ ] Not Started

**Problem:**
Blind spots in distributed tracing for edge functions and background jobs.

**Files to Modify:**
- `internal/functions/handler.go`
- `internal/jobs/worker.go`
- `internal/observability/tracer.go`

**Implementation Steps:**
- [ ] Add `StartFunctionSpan()` helper
- [ ] Add `StartJobSpan()` helper
- [ ] Propagate trace context to Deno runtime via environment
- [ ] Add span events for function/job lifecycle stages
- [ ] Include function/job metadata in span attributes

**Test Requirements:**
- [ ] Unit test: Function spans created with correct attributes
- [ ] Unit test: Job spans created with correct attributes
- [ ] Unit test: Trace context propagated correctly
- [ ] Integration test: Full trace visible in collector

**Test File:** `internal/functions/handler_test.go`, `internal/jobs/worker_test.go`

---

### 5.3 Create Operational Runbook

**Priority:** Medium
**Category:** Operations
**Status:** [ ] Not Started

**Problem:**
No incident response documentation.

**Files to Create:**
- `docs/operations/runbook.md`

**Implementation Steps:**
- [ ] Document common failure scenarios and remediation
- [ ] Add database troubleshooting section
- [ ] Add performance debugging section
- [ ] Add security incident response section
- [ ] Include escalation procedures
- [ ] Add monitoring dashboard interpretation guide

---

### 5.4 Add Job Queue Depth Metrics

**Priority:** Low
**Category:** Operations
**Status:** [ ] Not Started

**Problem:**
Cannot observe job queue health.

**Files to Modify:**
- `internal/jobs/manager.go`
- `internal/observability/metrics.go`

**Implementation Steps:**
- [ ] Add `fluxbase_jobs_queue_depth` gauge by namespace/priority
- [ ] Add `fluxbase_jobs_processing` gauge for active jobs
- [ ] Add `fluxbase_jobs_worker_utilization` gauge
- [ ] Add recommended alerting thresholds to documentation

**Test Requirements:**
- [ ] Unit test: Queue depth metric updated on enqueue/dequeue
- [ ] Unit test: Processing count accurate
- [ ] Unit test: Worker utilization calculated correctly

**Test File:** `internal/jobs/manager_test.go`

---

## Appendix: Implementation Order

Recommended implementation order optimizing for dependencies and impact:

```
Week 1-2: Phase 1 (Critical Security)
├── 1.1 TOTP Encryption (no dependencies)
├── 1.8 Correlation IDs (no dependencies)
├── 1.3 Distributed Rate Limiting (no dependencies)
├── 1.4 Progress Update Context (no dependencies)
├── 1.5 WebSocket Limits (no dependencies)
├── 1.6 FD Leak Fix (no dependencies)
├── 1.7 RLS Cache Size (no dependencies)
└── 1.2 TOTP Rate Limiting (depends on 1.1)

Week 3-4: Phase 2 (Scalability)
├── 2.4 Column Validation Cache (no dependencies)
├── 2.5 Connection Pool Increase (no dependencies)
├── 2.1 Batch Schema Queries (depends on 2.4)
├── 2.3 Async Broadcasting (no dependencies)
├── 2.8 Slow Client Disconnect (depends on 2.3)
├── 2.2 LISTEN Pooling (no dependencies)
├── 2.6 Streaming Function Output (no dependencies)
└── 2.7 Cursor Pagination (no dependencies)

Week 5-6: Phase 3 (Maintainability)
├── 3.1 Error Response Format (no dependencies)
├── 3.2 Unified Filter Parser (no dependencies)
├── 3.4 Idempotency Keys (depends on 1.8)
├── 3.5 OAuth State Persistence (no dependencies)
├── 3.6 Body Size Limits (no dependencies)
├── 3.3 Graceful Job Shutdown (no dependencies)
└── 3.7 Service Token Revocation (no dependencies)

Week 7-8: Phase 4-5 (DX & Ops)
├── 5.1 Backup Documentation (no dependencies)
├── 5.2 Function/Job Tracing (no dependencies)
├── 4.1 TypeScript Type Generation (no dependencies)
├── 4.2 ETags (no dependencies)
├── 4.4 Webhook Signing (no dependencies)
├── 4.5 Bulk Response Counts (no dependencies)
├── 5.3 Runbook (no dependencies)
├── 5.4 Job Queue Metrics (no dependencies)
└── 4.3 GraphQL Subscriptions (depends on 2.2, 2.3)
```

---

## Change Log

| Date | Change | Author |
|------|--------|--------|
| 2026-01-18 | Initial plan created | Architecture Review |

