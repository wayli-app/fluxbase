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
| Phase 1: Critical Security & Reliability | 8 | 8 | 0 | 0 |
| Phase 2: Scalability & Performance | 8 | 8 | 0 | 0 |
| Phase 3: Maintainability & Correctness | 7 | 0 | 0 | 7 |
| Phase 4: Developer Experience | 5 | 0 | 0 | 5 |
| Phase 5: Operations & Polish | 4 | 0 | 0 | 4 |
| **Total** | **32** | **16** | **0** | **16** |

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
**Status:** [x] Complete

**Problem:**
Per-instance rate limiting is bypassed in multi-instance deployments; attackers can target different instances.

**Files Modified:**
- `internal/middleware/rate_limiter.go`
- `internal/middleware/rate_limiter_warning_test.go` (new file)

**Implementation Steps:**
- [x] Add startup warning when using in-memory rate limiter in Kubernetes/Docker Compose
- [x] Detect multi-instance environment via env vars (KUBERNETES_SERVICE_HOST, POD_NAME, COMPOSE_PROJECT_NAME)
- [x] Suppress warning when Redis/Dragonfly is configured (FLUXBASE_REDIS_URL, FLUXBASE_DRAGONFLY_URL)
- [x] Warning logged only once per process to avoid log spam
- [ ] Document Redis/Dragonfly requirement for production deployments (docs update)
- [ ] Add metrics for rate limit backend type (future enhancement)

**Test Requirements:**
- [x] Unit test: Warning not displayed when Redis is configured
- [x] Unit test: Warning not displayed when Dragonfly is configured
- [ ] Integration test: Rate limits shared across simulated instances (requires Redis)

**Test File:** `internal/middleware/rate_limiter_warning_test.go`

---

### 1.4 ~~Fix Progress Update Context Leak~~ (Not a Bug)

**Priority:** ~~Critical~~ N/A
**Category:** Correctness
**Status:** [x] Reviewed - Not a Bug

**Original Concern:**
Job progress updates use `context.Background()` instead of job context.

**Review Finding:**
The `context.Background()` usage is **intentional and correct**. The code at `internal/jobs/worker.go:515` includes an explicit comment explaining the design:

```go
// Update in database with a short timeout to avoid blocking on slow DB
// Using a timeout context instead of job context since progress updates
// are async and should complete even if job is finishing
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
```

**Rationale:**
Progress updates are auxiliary operations that should complete independently of the job lifecycle. Using the job context would cause progress updates to be aborted when the job finishes or is cancelled, potentially leaving stale/incomplete progress data in the database. The 5-second timeout prevents blocking on slow DB operations.

**Conclusion:** No changes needed. Current implementation is correct.

---

### 1.5 Add Per-User WebSocket Connection Limits

**Priority:** High
**Category:** Reliability
**Status:** [x] Complete

**Problem:**
Single client can open unlimited WebSocket connections, exhausting server resources.

**Files Modified:**
- `internal/realtime/manager.go`
- `internal/realtime/manager_test.go`
- `internal/config/config.go`
- `internal/api/server.go`

**Implementation Steps:**
- [x] Add `max_connections_per_user` config option (default: 10)
- [x] Add `max_connections_per_ip` for anonymous connections (default: 20)
- [x] Track connection count per user ID in manager (`userConnections` map)
- [x] Track connection count per IP in manager (`ipConnections` map)
- [x] Reject new connections when limit exceeded with `ErrMaxUserConnectionsReached` / `ErrMaxIPConnectionsReached`
- [x] Added `AddConnectionWithIP()` method for explicit IP tracking
- [x] Added `GetUserConnectionCount()` and `GetIPConnectionCount()` methods
- [x] Added `SetConnectionLimits()` method for runtime updates
- [x] Add metrics for connection rejections (recorded via `RecordRealtimeError`)

**Test Requirements:**
- [x] Unit test: Connections under limit accepted
- [x] Unit test: Connections over limit rejected with proper error
- [x] Unit test: Connection count decremented on disconnect
- [x] Unit test: Anonymous connections limited by IP
- [x] Unit test: Authenticated users not affected by IP limit
- [x] Unit test: Different users have independent limits
- [x] Unit test: Global limit takes precedence
- [x] Unit test: Shutdown clears tracking maps
- [ ] Integration test: Rapid connection attempts handled correctly

**Test File:** `internal/realtime/manager_test.go`

---

### 1.6 Fix File Descriptor Leak in Function Runtime

**Priority:** High
**Category:** Reliability
**Status:** [x] Complete

**Problem:**
Pipes not closed when command fails to start, leaking file descriptors.

**Files Modified:**
- `internal/runtime/runtime.go` (lines 265-286)

**Implementation Steps:**
- [x] Close stdout pipe if stderr pipe creation fails
- [x] Close both pipes if `cmd.Start()` fails
- [x] Add comments explaining pipe ownership (managed by Wait() after Start succeeds)
- [ ] Add FD tracking metrics for debugging (deferred - requires monitoring infrastructure)

**Analysis:**
The original concern mentioned goroutine panics, but the goroutines already had `recover()` in place. The actual FD leak occurred when:
1. `StderrPipe()` failed after `StdoutPipe()` succeeded
2. `cmd.Start()` failed after both pipes were created

After `cmd.Start()` succeeds, Go's `exec` package manages the pipes and closes them when `Wait()` is called. The fix ensures pipes are properly closed in error paths before `Start()`.

**Test Requirements:**
- [x] Verified: Pipes closed on command start failure (code review)
- [x] Verified: Pipes closed on stderr pipe creation failure (code review)
- [N/A] Goroutine panics already handled by existing `recover()`
- [ ] Integration test: No FD leak under stress test (requires Deno binary - manual testing)

**Test File:** N/A - Fix is in error handling code paths that are difficult to unit test without mocking exec

---

### 1.7 Increase and Configure RLS Cache Size

**Priority:** High
**Category:** Performance
**Status:** [x] Complete

**Problem:**
10K entry RLS cache with 2-second TTL was insufficient for high-throughput realtime; cache misses caused excessive DB roundtrips.

**Files Modified:**
- `internal/realtime/subscription.go` (RLS cache implementation)
- `internal/realtime/subscription_test.go` (unit tests)
- `internal/config/config.go` (config options)
- `internal/api/server.go` (wire up config)

**Implementation Steps:**
- [x] Add `realtime.rls_cache_size` config option (default: 100,000)
- [x] Add `realtime.rls_cache_ttl` config option (default: 30s)
- [x] Create `RLSCacheConfig` struct for cache configuration
- [x] Create `newRLSCacheWithConfig()` to accept custom settings
- [x] Update `SubscriptionManager` to use per-instance RLS cache (not global)
- [x] Add `NewSubscriptionManagerWithConfig()` for custom cache config
- [x] Wire up config in server.go
- [ ] Add cache hit/miss metrics (deferred - requires metrics infrastructure)
- [ ] Consider LRU vs LFU eviction strategy (current: evict expired on size limit)

**Changes from defaults:**
- Cache size: 10,000 → 100,000 entries (10x increase)
- Cache TTL: 2 seconds → 30 seconds (15x increase)
- Cache is now per-SubscriptionManager (not global), allowing better isolation

**Test Requirements:**
- [x] Unit test: Cache uses default config when none provided
- [x] Unit test: Cache uses custom config values
- [x] Unit test: Zero/negative config values fall back to defaults
- [x] Unit test: SubscriptionManager created with custom RLS cache
- [ ] Benchmark: Cache performance at 100K entries (manual testing)

**Test File:** `internal/realtime/subscription_test.go`

---

### 1.8 Add Correlation IDs to Error Responses

**Priority:** High
**Category:** Operations
**Status:** [x] Complete

**Problem:**
Cannot correlate client errors to server logs for debugging.

**Files Modified:**
- `internal/api/rest_errors.go` (new error helpers with request ID)
- `internal/api/rest_errors_test.go` (comprehensive tests)

**Pre-existing Infrastructure:**
- Fiber's `requestid` middleware was already in place (server.go:1075)
- Request ID already propagated to logs via structured logger
- X-Request-ID header already in CORS allowed/exposed headers

**Implementation Steps:**
- [x] Verified request ID middleware already set up (Fiber's built-in `requestid.New()`)
- [x] Created `getRequestID()` helper to extract request ID from context
- [x] Created `ErrorResponse` struct with standardized fields including `request_id`
- [x] Created `SendError()` helper for simple errors with request ID
- [x] Created `SendErrorWithCode()` helper for errors with error codes
- [x] Created `SendErrorWithDetails()` helper for detailed errors (hint, details)
- [x] Updated `handleDatabaseError()` to use new helpers (includes request ID + error codes)
- [x] Updated `handleRLSViolation()` to use new helpers (includes request ID)
- [x] Added request ID to log entries in error handlers

**Error Response Format:**
```json
{
  "error": "Human-readable error message",
  "code": "ERROR_CODE",
  "message": "Additional context (optional)",
  "hint": "Suggestion for resolution (optional)",
  "details": {...},
  "request_id": "uuid-from-request"
}
```

**Test Requirements:**
- [x] Unit test: Request ID extracted from locals (requestid middleware)
- [x] Unit test: Request ID extracted from X-Request-ID header (fallback)
- [x] Unit test: Locals preferred over header when both present
- [x] Unit test: Empty request ID when none provided
- [x] Unit test: SendError includes request ID
- [x] Unit test: SendErrorWithCode includes request ID and code
- [x] Unit test: SendErrorWithDetails includes all fields
- [x] Unit test: handleDatabaseError includes request ID and error code

**Test File:** `internal/api/rest_errors_test.go`

---

## Phase 2: Scalability & Performance

These items address bottlenecks that will impact performance as usage grows.

### 2.1 Batch Schema Introspection Queries

**Priority:** High
**Category:** Scalability
**Status:** [x] Complete

**Problem:**
N+1 query pattern: 5N queries for N tables during schema cache refresh.

**Files Modified:**
- `internal/database/schema_inspector.go` (batch query functions)
- `internal/database/schema_inspector_test.go` (unit tests)

**Implementation Steps:**
- [x] Created `batchFetchTableMetadata()` to orchestrate batched metadata fetching
- [x] Created `batchGetColumns()` for batch column retrieval (uses information_schema)
- [x] Created `batchGetMaterializedViewColumns()` for materialized views (uses pg_catalog)
- [x] Created `batchGetPrimaryKeys()` for batch primary key retrieval
- [x] Created `batchGetForeignKeys()` for batch foreign key retrieval
- [x] Created `batchGetIndexes()` for batch index retrieval
- [x] Updated `GetAllTables()` to use batched queries instead of N individual calls
- [x] Updated `GetAllViews()` to use batched queries
- [x] Updated `GetAllMaterializedViews()` to use batched queries
- [x] Group results by "schema.table" key in Go code
- [ ] Add query timing metrics (deferred - requires metrics infrastructure)

**Query Count Reduction:**
- Before: 1 + 4N queries (list + columns/pk/fk/indexes per table)
- After: 5 queries (list + batched columns/pk/fk/indexes)
- For 100 tables: 401 queries → 5 queries (99% reduction)

**Test Requirements:**
- [x] Unit test: Batch column aggregation by table key
- [x] Unit test: Batch primary key aggregation (including composite keys)
- [x] Unit test: Batch foreign key aggregation
- [x] Unit test: Batch index aggregation
- [x] Unit test: Table map merging with metadata
- [x] Unit test: Result order preserved after batch merge
- [x] Unit test: Empty schema handled correctly
- [x] Unit test: Views don't get primary/foreign keys
- [x] Unit test: Materialized views can have indexes
- [ ] Benchmark: Compare query count before/after (requires DB)

**Test File:** `internal/database/schema_inspector_test.go`

---

### 2.2 Implement LISTEN Connection Pooling

**Priority:** High
**Category:** Scalability
**Status:** [x] Complete

**Problem:**
Single PostgreSQL LISTEN connection is bottleneck for realtime subscriptions.

**Files Modified:**
- `internal/realtime/listener.go` (added RealtimeListener interface)
- `internal/realtime/listener_pool.go` (new file - pooled listener implementation)
- `internal/realtime/listener_pool_test.go` (comprehensive tests)
- `internal/config/config.go` (config options)
- `internal/api/server.go` (wire up ListenerPool)

**Implementation Steps:**
- [x] Add `realtime.listener_pool_size` config option (default: 2)
- [x] Add `realtime.notification_workers` config option (default: 4)
- [x] Add `realtime.notification_queue_size` config option (default: 1000)
- [x] Create `RealtimeListener` interface for both Listener and ListenerPool
- [x] Create `ListenerPool` with configurable pool of LISTEN connections
- [x] Implement parallel notification processing with worker goroutines
- [x] Handle connection failures with automatic exponential backoff reconnection
- [x] Add pool health metrics (active connections, notifications received/processed, failures, reconnections)
- [x] Wire up ListenerPool in server.go with config values

**Key Features:**
- Multiple redundant LISTEN connections for fault tolerance
- Worker pool for parallel notification processing (avoids single-threaded bottleneck)
- Non-blocking notification queue with configurable size
- Automatic reconnection with exponential backoff
- Comprehensive metrics for monitoring

**Test Requirements:**
- [x] Unit test: Default config values
- [x] Unit test: Custom config values respected
- [x] Unit test: Negative/zero config values fall back to defaults
- [x] Unit test: Notification channel capacity calculation
- [x] Unit test: Stop before start doesn't panic
- [x] Unit test: Metrics initial values
- [x] Unit test: Metrics queue capacity
- [x] Unit test: Atomic counters thread-safe
- [x] Unit test: ListenerPool implements RealtimeListener interface
- [x] Unit test: Listener implements RealtimeListener interface
- [x] Unit test: EnrichJobWithETA edge cases
- [x] Benchmark: GetMetrics performance
- [x] Benchmark: EnrichJobWithETA performance
- [ ] Integration test: Full listener pool with database (requires DB)

**Test File:** `internal/realtime/listener_pool_test.go`

---

### 2.3 Add Async Message Broadcasting

**Priority:** High
**Category:** Scalability
**Status:** [x] Complete

**Problem:**
RWMutex held during WebSocket broadcast blocks other operations.

**Files Modified:**
- `internal/realtime/connection.go` (async message queue per connection)
- `internal/realtime/connection_test.go` (comprehensive tests)
- `internal/realtime/manager.go` (queue size config)
- `internal/config/config.go` (client_message_queue_size option)
- `internal/api/server.go` (wire up config)

**Implementation Steps:**
- [x] Add per-client message queue (buffered channel) in Connection struct
- [x] Add writer goroutine per client that drains queue
- [x] SendMessage now queues messages non-blocking (O(1) instead of O(write time))
- [x] Add `realtime.client_message_queue_size` config option (default: 256)
- [x] Add queue depth metrics (GetQueueStats method)
- [x] Handle slow clients - return ErrQueueFull, track dropped messages
- [x] Mark connections as slow after multiple queue full events
- [x] Support sync mode for backward compatibility in tests (NewConnectionSync)
- [x] Graceful shutdown - drain queue before closing
- [x] Wire up config in server.go

**Key Benefits:**
- Broadcast no longer holds RWMutex while writing to clients
- Slow clients don't block other clients
- Non-blocking message queueing (returns immediately)
- Configurable queue size per environment
- Automatic slow client detection and handling

**Test Requirements:**
- [x] Unit test: NewConnectionWithQueueSize creates correct queue
- [x] Unit test: Default queue size on zero/negative values
- [x] Unit test: NewConnectionSync for sync mode
- [x] Unit test: SendMessage to closed connection returns error
- [x] Unit test: GetQueueStats returns correct values
- [x] Unit test: Close multiple times doesn't panic
- [x] Unit test: IsSlowClient detection
- [x] Unit test: ConnectionQueueStats struct
- [x] Unit test: SendMessage with slow client marked
- [x] Benchmark: Subscribe/Unsubscribe
- [x] Benchmark: IsSubscribed
- [x] Benchmark: GetQueueStats
- [ ] Load test: Broadcast latency comparison (requires full E2E setup)

**Test File:** `internal/realtime/connection_test.go`

---

### 2.4 Add Column Validation Cache

**Priority:** Medium
**Category:** Performance
**Status:** [x] Complete

**Problem:**
O(n) column lookup per field on every request.

**Files Modified:**
- `internal/database/schema_inspector.go` - Added ColumnMap to TableInfo, BuildColumnMap(), GetColumn(), HasColumn()
- `internal/api/rest_query.go` - Updated columnExists() to use O(1) HasColumn lookup
- `internal/database/schema_inspector_test.go` - Added tests and benchmarks for column map

**Implementation:**
- [x] Added `ColumnMap map[string]*ColumnInfo` field to TableInfo struct
- [x] Added `BuildColumnMap()` method to populate map from Columns slice
- [x] Added `GetColumn(name string) *ColumnInfo` method for O(1) lookup with fallback
- [x] Added `HasColumn(name string) bool` method for existence checks
- [x] Map automatically built during schema cache refresh (batchFetchTableMetadata, GetTableInfo)
- [x] Updated RESTHandler.columnExists to use HasColumn for O(1) lookups
- [x] Optimized duplicate column iteration pattern in rest_query.go

**Test Requirements:**
- [x] Unit test: BuildColumnMap creates correct map from columns
- [x] Unit test: GetColumn returns correct column or nil
- [x] Unit test: HasColumn returns correct boolean
- [x] Unit test: Fallback works when map not built
- [x] Benchmark: Compare O(1) lookup vs O(n) fallback

**Test File:** `internal/database/schema_inspector_test.go`

---

### 2.5 Increase Default Connection Pool

**Priority:** Medium
**Category:** Scalability
**Status:** [x] Complete

**Problem:**
25 max connections insufficient for concurrent workloads (jobs + API + realtime).

**Files Modified:**
- `internal/config/config.go` - Updated defaults with sizing guidance
- `internal/config/config_test.go` - Updated test fixtures

**Implementation:**
- [x] Increased default `database.max_connections` from 25 to 50
- [x] Increased default `database.min_connections` from 5 to 10 (warmer pool for production)
- [x] Added inline documentation with sizing guidance:
  - Single-instance: 50 connections
  - Multi-instance: divide by instance count (e.g., 3 instances = 17 per instance)
  - Approximate breakdown: API (20), Jobs (15), Realtime (10), Schema cache (5)
  - Recommendation to monitor pg_stat_activity and pool exhaustion metrics

**Test Requirements:**
- [x] Unit test: Config accepts new default (updated fixture)
- [x] Existing validation tests still pass

**Test File:** `internal/config/config_test.go`

---

### 2.6 Add Streaming Result Parsing for Functions

**Priority:** Medium
**Category:** Scalability
**Status:** [x] Complete

**Problem:**
1MB buffer per line in function output can OOM on large results.

**Files Modified:**
- `internal/runtime/runtime.go` - Added output size limiting with truncation
- `internal/config/config.go` - Added `functions.max_output_size` config
- `internal/runtime/runtime_test.go` - Added tests for output size options

**Implementation:**
- [x] Added `maxOutputSize` field to `DenoRuntime` struct
- [x] Added `WithMaxOutputSize(bytes int) Option` function
- [x] Set defaults: 10MB for functions, 50MB for jobs
- [x] Added `functions.max_output_size` config option (default: 10MB)
- [x] Implemented output tracking and truncation in stdout processing
- [x] Preserved `__RESULT__::` line even when output is truncated
- [x] Added warning log when truncation occurs
- [x] Progress updates and log callbacks continue even during truncation

**Test Requirements:**
- [x] Unit test: WithMaxOutputSize option works correctly
- [x] Unit test: Default output sizes set correctly per runtime type
- [x] Unit test: Custom option overrides default
- [x] Unit test: WithMemoryLimit and WithTimeout options work

**Test File:** `internal/runtime/runtime_test.go`

---

### 2.7 Implement Cursor-Based Pagination

**Priority:** Medium
**Category:** Performance
**Status:** [x] Complete

**Problem:**
Offset pagination inefficient for large datasets; performance degrades linearly.

**Files Modified:**
- `internal/api/query_parser.go` - Added cursor and cursor_column parameters
- `internal/api/query_builder.go` - Added cursor encoding/decoding and keyset condition building
- `internal/api/query_parser_test.go` - Added cursor parsing tests
- `internal/api/query_builder_test.go` - Added cursor encoding/decoding and query building tests

**Implementation:**
- [x] Added `CursorData` struct with Column, Value, and Desc fields
- [x] Added `EncodeCursor()` function to create base64-encoded cursors
- [x] Added `DecodeCursor()` function to parse cursors with validation
- [x] Added `Cursor` and `CursorColumn` fields to QueryParams
- [x] Added `cursor` and `cursor_column` query parameter parsing
- [x] Added `WithCursor()` method to QueryBuilder
- [x] Implemented `buildCursorCondition()` for keyset WHERE conditions
- [x] Cursor supports both ascending (>) and descending (<) orders

**Note:** SDK update and response header additions deferred to separate task.

**Test Requirements:**
- [x] Unit test: Cursor encoded/decoded correctly
- [x] Unit test: Query uses keyset condition (ascending and descending)
- [x] Unit test: Cursor column override works
- [x] Unit test: Invalid cursor returns error
- [x] Unit test: Cursor combines with filters correctly

**Test File:** `internal/api/query_parser_test.go`, `internal/api/query_builder_test.go`

---

### 2.8 Auto-Disconnect Slow WebSocket Clients

**Priority:** Medium
**Category:** Reliability
**Status:** [x] Complete

**Problem:**
Slow clients are tracked but not acted upon; they accumulate and waste resources.

**Files Modified:**
- `internal/realtime/manager.go` - Added slow client checking goroutine and disconnect logic
- `internal/config/config.go` - Added slow_client_threshold and slow_client_timeout config
- `internal/realtime/manager_test.go` - Added tests for slow client config

**Implementation:**
- [x] Added `realtime.slow_client_threshold` config (default: 100 pending messages)
- [x] Added `realtime.slow_client_timeout` config (default: 30s)
- [x] Added `SlowClientThreshold` and `SlowClientTimeout` to ManagerConfig
- [x] Added `slowClientFirstSeen` map to track when clients first became slow
- [x] Added `slowClientChecker()` goroutine that runs every 5 seconds
- [x] Implemented `checkAndDisconnectSlowClients()` with proper lock handling
- [x] Implemented `disconnectSlowClient()` with 1008 Policy Violation close frame
- [x] Added `slowClientsDisconnected` metric counter
- [x] Clients that recover before timeout are automatically untracked

**Test Requirements:**
- [x] Unit test: Config applies default slow client settings
- [x] Unit test: Config applies custom slow client settings
- [x] Unit test: Tracking map is initialized
- [x] Unit test: Disconnect counter starts at 0

**Test File:** `internal/realtime/manager_test.go`

---

## Phase 3: Maintainability & Correctness

These items improve code quality and fix correctness issues.

### 3.1 Standardize Error Response Format

**Priority:** High
**Category:** Developer Experience
**Status:** [x] Complete

**Problem:**
Inconsistent error structures confuse SDK developers and complicate client error handling.

**Files Modified:**
- `internal/api/rest_errors.go` (error codes + convenience functions)
- `internal/api/rest_errors_test.go` (comprehensive tests)
- `internal/api/auth_middleware.go` (migrated)
- `internal/api/admin_auth_handler.go` (migrated)
- `internal/api/storage_files.go` (migrated)
- `internal/api/ddl_handler.go` (migrated)
- `internal/api/realtime_admin_handler.go` (migrated)
- `internal/api/server.go` (migrated)

**Implementation Steps:**
- [x] Define 30+ standard error code constants (ErrCodeMissingAuth, ErrCodeInvalidToken, etc.)
- [x] Create convenience helpers: SendBadRequest, SendUnauthorized, SendForbidden, SendNotFound,
      SendConflict, SendInternalError, SendValidationError, SendMissingAuth, SendInvalidToken,
      SendTokenRevoked, SendInsufficientPermissions, SendAdminRequired, SendInvalidBody,
      SendMissingField, SendInvalidID, SendResourceNotFound, SendOperationFailed, SendFeatureDisabled
- [x] Migrate all 204 fiber.Map error responses to use helpers
- [ ] Document error codes in OpenAPI spec (deferred)
- [ ] Update SDK to parse structured errors (deferred)

**Test Requirements:**
- [x] Unit test: Error helper produces correct format (20+ tests)
- [x] Unit test: All error code constants verified
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
**Status:** [x] Complete

**Problem:**
Hard 30-second timeout kills long-running jobs without cleanup.

**Files Modified:**
- `internal/config/config.go` (added GracefulShutdownTimeout config)
- `internal/jobs/types.go` (added JobStatusInterrupted)
- `internal/jobs/types_test.go` (updated tests)
- `internal/jobs/storage.go` (added InterruptJob method)
- `internal/jobs/worker.go` (draining mode, configurable timeout, interrupt handling)

**Implementation Steps:**
- [x] Add `jobs.graceful_shutdown_timeout` config (default: 5m)
- [x] On shutdown signal, stop accepting new jobs (draining mode)
- [x] Update worker status to "draining" in database
- [x] Wait for running jobs to complete up to configurable timeout
- [x] Mark incomplete jobs as "interrupted" not "failed"
- [x] Add interruptAllJobs() for graceful shutdown timeout
- [ ] Add job interrupt handling callback for cleanup (deferred - requires job-level changes)

**Test Requirements:**
- [x] Unit test: JobStatusInterrupted constant
- [ ] Unit test: Shutdown waits for running jobs (requires integration)
- [ ] Unit test: New jobs rejected during shutdown (requires integration)
- [ ] Unit test: Timeout forces termination (requires integration)
- [ ] Unit test: Interrupted jobs marked correctly (requires integration)
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

