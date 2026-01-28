# Fluxbase Codebase Improvement Analysis

**Date:** January 2026
**Scope:** Security, Maintainability, Functionality, Developer Experience

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Security Improvements](#1-security-improvements)
3. [Maintainability Improvements](#2-maintainability-improvements)
4. [Functionality: Feature Gaps & New Features](#3-functionality-feature-gaps--new-features)
5. [Developer Experience Improvements](#4-developer-experience-improvements)
6. [Feature Deep-Dive: Visual Table Viewer with Relations](#5-feature-deep-dive-visual-table-viewer-with-relations)
7. [Feature Deep-Dive: Policy Editor & Security Warning System](#6-feature-deep-dive-policy-editor--security-warning-system)
8. [Feature Parity Matrix vs Supabase](#7-feature-parity-matrix-vs-supabase)
9. [Prioritized Roadmap](#8-prioritized-roadmap)

---

## Executive Summary

Fluxbase is a well-architected Backend-as-a-Service with strong fundamentals: a comprehensive auth system (JWT, OAuth2, OIDC, SAML, MFA, magic links), parameterized SQL queries, proper RLS enforcement, path traversal protection, and good observability. However, there are significant opportunities in four areas:

- **Security:** Fix SQL injection in pub/sub, standardize error handling to prevent info leakage, add startup config validation
- **Maintainability:** Eliminate ~993 inconsistent error handling patterns, improve test coverage on handlers, add missing godoc
- **Functionality:** Two critical gaps vs Supabase (visual schema viewer, policy editor), plus several medium-priority features
- **Developer Experience:** Missing Go SDK, incomplete React SDK coverage, CLI-only migrations

The codebase has **500+ Go files**, **150+ TypeScript files**, **71 database migrations**, **100+ RLS policies**, and covers 30+ internal modules. The analysis below is based on a thorough review of every module.

---

## 1. Security Improvements

### 1.1 CRITICAL: SQL Injection in PostgreSQL Pub/Sub

**File:** `internal/pubsub/postgres.go:165`

```go
_, err = p.pool.Exec(ctx, fmt.Sprintf("SELECT pg_notify('%s', %s)", pgChannel, payloadJSON))
```

The channel name is interpolated directly into SQL using `fmt.Sprintf`. While the channel name is sanitized by replacing colons with underscores (lines 230-241), this does NOT properly quote PostgreSQL identifiers. A malformed channel name could inject SQL.

**Recommendation:** Use parameterized queries or PostgreSQL's `quote_ident()`:
```go
_, err = p.pool.Exec(ctx, "SELECT pg_notify($1, $2)", pgChannel, payloadJSON)
```

**Risk:** Medium (internal-only channel names, but defense-in-depth requires fixing)

### 1.2 HIGH: Rate Limiting Bypass in Multi-Instance Deployments

**File:** `internal/middleware/rate_limiter.go`

Rate limiting uses Fiber's in-memory storage (lines 74-93), which means each server instance has independent counters. In load-balanced deployments, an attacker can bypass rate limits by targeting different instances.

**Recommendation:**
- Require Redis/Dragonfly backend when running multiple instances
- Detect multi-instance mode (Kubernetes, Docker Swarm) and fail-start if Redis is not configured
- Document this clearly in production runbook

### 1.3 HIGH: Missing Startup Configuration Validation

**File:** `internal/config/config.go`

Several security-critical settings are validated only at runtime (with warnings), not at startup:

| Setting | Risk | Current Behavior |
|---------|------|-----------------|
| `SetupToken` empty | Dashboard accessible without auth | Logs warning only |
| `CORS AllowedOrigins="*"` | Cross-origin attacks | No warning |
| `EncryptionKey` wrong length | OAuth secrets stored unencrypted | Clears key silently |
| `JWTSecret` default value | Token forgery | Rejects at startup (good) |

**Recommendation:** Add a `validateSecurityConfig()` function called at startup that:
- **Fails fast** on insecure `EncryptionKey` (wrong length or missing in production)
- **Warns loudly** (stderr + structured log) when CORS allows `*`
- **Fails fast** if `SetupToken` is empty and `FLUXBASE_ENV=production`

### 1.4 MEDIUM: Inconsistent Error Response Exposure

**File:** All 48+ handler files in `internal/api/`

~993 instances of bare `c.Status(X).JSON(fiber.Map{"error": "..."})` return unstructured errors. Some expose internal details:

```go
// Example from multiple handlers:
return c.Status(500).JSON(fiber.Map{"error": err.Error()})
```

This can leak database error messages, file paths, or internal state to clients.

**Recommendation:**
- Mandate use of `SendError()` / `SendBadRequest()` / etc. from `rest_errors.go`
- Add a linter rule or middleware to reject bare `fiber.Map{"error":...}` patterns
- Ensure 500 errors always use generic messages with request IDs for correlation

### 1.5 MEDIUM: WebSocket Connection Authentication Gaps

**Files:** `internal/realtime/handler.go`, `internal/realtime/auth_adapter.go`

WebSocket connections validate tokens at connection time but there's no documented re-validation when tokens expire during long-lived connections.

**Recommendation:**
- Implement periodic token validation (every N minutes) on active WebSocket connections
- Disconnect clients whose tokens have expired or been revoked
- Add heartbeat-based session validation

### 1.6 LOW: Audit Log for Sensitive Admin Operations

**Current:** RLS violations are logged to `auth.rls_audit_log`. However, admin operations (user deletion, settings changes, key rotation, impersonation) are not logged to a dedicated audit trail.

**Recommendation:** Create an `admin.audit_log` table tracking:
- Who performed the action (dashboard user ID)
- What action (CRUD operation type)
- What resource (table, user, setting)
- When (timestamp)
- From where (IP, user agent)

---

## 2. Maintainability Improvements

### 2.1 CRITICAL: Error Handling Standardization

**Current state:** `rest_errors.go` defines 67 error codes and 6+ convenience functions (`SendError`, `SendBadRequest`, `SendUnauthorized`, `SendNotFound`, etc.), but only ~50 out of ~1,000 error returns use them.

**Impact:** Inconsistent error formats, missing request ID correlation, no structured logging on 95% of errors.

**Recommendation:**
1. Create a middleware that wraps all handler responses and normalizes error formats
2. Refactor handlers to use structured error returns: `return SendBadRequest(c, "Invalid email")`
3. Add a CI lint check for `fiber.Map{"error":` patterns in handler files

### 2.2 HIGH: Request Validation Middleware

**Current state:** Every handler independently parses and validates request bodies:

```go
// Repeated in 20+ handlers
var req SomeRequest
if err := c.BodyParser(&req); err != nil {
    return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
}
```

**Recommendation:** Create a generic validation middleware:
```go
func ParseAndValidate[T any](c *fiber.Ctx) (*T, error) {
    var req T
    if err := c.BodyParser(&req); err != nil {
        return nil, SendInvalidBody(c)
    }
    if err := validate.Struct(req); err != nil {
        return nil, SendValidationError(c, err)
    }
    return &req, nil
}
```

### 2.3 HIGH: Test Coverage for Handler Files

**Current state:** Many API handlers lack dedicated test files:
- `clientkey_handler.go` - No `_test.go`
- `storage_handler.go` - Limited coverage
- `webhook_handler.go` - No `_test.go`
- `branch_handler.go` - No `_test.go`
- `servicekey_handler.go` - No `_test.go`

**Coverage targets (from CLAUDE.md):** Critical modules should have 70%+ coverage. Many handlers are below 50%.

**Recommendation:**
1. Prioritize test creation for auth and storage handlers (security-critical)
2. Create a shared test harness for handler testing (mock Fiber context, DB, auth service)
3. Add table-driven tests for all error paths

### 2.4 MEDIUM: Server Struct Decomposition

**File:** `internal/api/server.go` (lines 44-132)

The `Server` struct has 40+ fields, coupling all concerns together. This makes testing difficult and violates single responsibility.

**Recommendation:** Split into sub-handlers:
```go
type Server struct {
    config     *config.Config
    auth       *AuthHandlers
    storage    *StorageHandlers
    crud       *CRUDHandlers
    admin      *AdminHandlers
    // ...
}
```

### 2.5 MEDIUM: Godoc Coverage on Exported Functions

Multiple handler files lack comments on exported functions. Examples:
- `clientkey_handler.go`: `CreateClientKey`, `ListClientKeys` - no godoc
- `oauth_handler.go`: `OAuthHandler` struct - no godoc
- `branch_handler.go`: Handler methods - no godoc

**Recommendation:** Add brief godoc comments on all exported types and functions, focusing on behavior contracts and error conditions.

### 2.6 LOW: Database Query Timeout Enforcement

**File:** `internal/database/executor.go`

No explicit query timeout mechanism is visible. While pgx supports context cancellation, handlers don't consistently set timeouts.

**Recommendation:**
- Add a default query timeout (e.g., 30s) in the executor
- Allow per-query overrides via context
- Add metrics for slow queries (>1s)

---

## 3. Functionality: Feature Gaps & New Features

### 3.1 Visual Schema Designer / ERD View (HIGH - User Requested)

**Gap:** No visual representation of database schema, table relationships, or foreign key connections. Users must mentally model relationships from raw column data.

**What exists in backend:** Schema introspector already returns complete `ForeignKey` data (name, column, referenced table/column, ON DELETE/UPDATE actions), `IndexInfo`, `PrimaryKey`, and `RLSEnabled` per table. The GraphQL engine already resolves FK relationships.

**What's missing:** Frontend visualization component. See [Section 5](#5-feature-deep-dive-visual-table-viewer-with-relations) for detailed design.

### 3.2 RLS Policy Editor & Security Warnings (HIGH - User Requested)

**Gap:** No UI to create, edit, view, or test RLS policies. No warnings for tables without RLS. Users must write raw SQL.

**What exists:** Full RLS middleware enforcement, 100+ existing policies, impersonation system, RLS audit logging, and SQL editor with RLS testing support.

**What's missing:** Policy introspection API (`pg_policies`), visual policy builder, security advisor. See [Section 6](#6-feature-deep-dive-policy-editor--security-warning-system) for detailed design.

### 3.3 Enhanced DDL Operations (HIGH)

**Current:** `CreateTableRequest` only supports `Name`, `Type`, `Nullable`, `PrimaryKey`, `DefaultValue`.

**Missing DDL operations via API/UI:**
- Create/drop foreign keys
- Create/drop indexes
- Create/drop unique constraints
- Create/drop check constraints
- Alter column types (ALTER COLUMN ... TYPE)
- Create/alter/drop views and materialized views
- Create/alter/drop triggers
- Create/alter/drop database functions (PL/pgSQL)

**Recommendation:** Extend `ddl_handler.go` to support these operations. Add corresponding UI components in the admin table editor.

### 3.4 Go SDK (MEDIUM)

**Gap:** `CLAUDE.md` mentions a Go SDK at `pkg/client/` but the directory does not exist. Go developers must use the REST API directly.

**Recommendation:** Implement a Go client SDK with:
- Auth (sign up, sign in, token management)
- Query builder (similar to TypeScript SDK)
- Realtime subscriptions (WebSocket)
- Storage operations
- Function invocation

### 3.5 Migration Management UI (MEDIUM)

**Current:** Migrations are CLI-only (`cli/cmd/migrations.go`).

**Recommendation:** Add an admin dashboard page for:
- Viewing migration history and status
- Applying pending migrations
- Rolling back migrations
- Creating new migration files (with SQL editor)
- Diff view between migration versions

### 3.6 Database Message Queue (MEDIUM)

**Current:** The jobs system handles task execution but there's no general-purpose message queue for inter-service communication.

**Recommendation:** Implement a PostgreSQL-backed message queue (similar to pgmq):
- Durable message storage with guaranteed delivery
- Pull-based consumption model
- Dead letter queues
- Admin UI for queue monitoring
- SDK methods for enqueue/dequeue

### 3.7 Data Export/Import (MEDIUM)

**Gap:** No bulk data export or import capabilities in the admin UI.

**Recommendation:**
- CSV/JSON export from table viewer (per table or filtered)
- CSV/JSON import with column mapping
- pg_dump/pg_restore integration for full database export
- Data seeding support via admin UI

### 3.8 Security Advisor Dashboard (MEDIUM)

**Gap:** No automated scanning for common security misconfigurations.

**Recommendation:** Build a security advisor that checks:
- Tables in `public` schema without RLS enabled
- Missing indexes on foreign key columns
- Overly permissive policies (e.g., `USING (true)`)
- Unused or redundant policies
- Tables accessible via `anon` key without explicit policies
- Weak password policy configuration
- Missing MFA enforcement for admin users
- Expired or unused API keys

### 3.9 Performance Advisor (LOW)

**Recommendation:** Build a performance advisor that checks:
- Missing indexes on frequently queried columns
- Slow query log analysis (queries >1s)
- Unused indexes consuming disk space
- Table bloat (dead tuples)
- Connection pool utilization
- N+1 query detection in RLS policies

### 3.10 Database Vault / Column Encryption (LOW)

**Gap:** No in-database transparent column encryption.

**Recommendation:** Implement column-level encryption using PostgreSQL's `pgcrypto`:
- Encrypt/decrypt columns transparently
- Key management UI in admin dashboard
- Support for encrypting specific columns (PII, secrets)

### 3.11 Foreign Data Wrappers UI (LOW)

**Gap:** No support for querying external data sources.

**Recommendation:** Add FDW management UI:
- Connect external PostgreSQL, MySQL, MongoDB, S3, BigQuery
- Map external tables to local schemas
- Query external data via standard REST API

### 3.12 Identity Provider Mode (LOW)

**Gap:** Fluxbase can authenticate users but cannot serve as an identity provider for other applications.

**Recommendation:** Add OAuth2 server mode:
- Register external applications as OAuth clients
- Issue access tokens for third-party apps
- "Sign in with [Your App]" capability

### 3.13 Webhook Retry Dashboard (LOW)

**Current:** Webhooks support retry policies but monitoring is limited.

**Recommendation:**
- Visual delivery history with success/failure indicators
- Retry queue visibility
- Payload inspection for debugging
- Response body logging

---

## 4. Developer Experience Improvements

### 4.1 React SDK Coverage Gaps

**Missing hooks for:**
| Feature | Hook Status |
|---------|-------------|
| Edge functions management | Missing |
| Background jobs | Missing |
| Migrations | Missing |
| RPC management | Missing |
| DDL operations | Missing |
| Vector operations | Missing |
| Database branching | Missing |
| Settings (detailed) | Partial |

**Recommendation:** Add React hooks for the missing features, following the existing pattern (`useMutation` + `useQuery` from TanStack).

### 4.2 TypeScript SDK Type Safety

**File:** `admin/src/lib/api.ts` (lines 362-382)

The `TableInfo` interface types `foreign_keys` and `indexes` as `unknown`:
```typescript
foreign_keys: unknown  // Should be ForeignKey[]
indexes: unknown       // Should be IndexInfo[]
```

**Recommendation:** Add proper TypeScript interfaces:
```typescript
interface ForeignKey {
  name: string
  column_name: string
  referenced_table: string
  referenced_column: string
  on_delete: string
  on_update: string
}

interface IndexInfo {
  name: string
  columns: string[]
  is_unique: boolean
  is_primary: boolean
}
```

### 4.3 Admin UI Keyboard Shortcuts

**Current:** The admin UI has a command menu (`command-menu.tsx`) but limited keyboard shortcuts.

**Recommendation:** Add shortcuts for common operations:
- `Ctrl+K` - Command palette (exists)
- `Ctrl+S` - Save current form
- `Ctrl+Enter` - Execute SQL query
- `Ctrl+Shift+I` - Toggle impersonation
- `Ctrl+N` - New record/function/job
- `Esc` - Close dialogs

### 4.4 Admin UI: Table Viewer Enhancements

**Missing features in the table viewer:**
- Column resizing
- Column reordering (drag-and-drop)
- Inline JSON editor for JSONB columns
- Date picker for timestamp columns
- Foreign key lookup (dropdown showing referenced records)
- Cell-level copy to clipboard
- Row-level JSON view
- Column statistics (min, max, avg, null count)

### 4.5 OpenAPI Documentation Enhancement

**File:** `internal/api/openapi.go`

**Recommendation:** Generate interactive API documentation in the admin UI:
- Swagger UI or Redoc embedded in `/api/rest` page
- Auto-generated from schema (tables become endpoints)
- Try-it-out functionality with authentication
- SDK code generation examples (TypeScript, Go, Python, cURL)

### 4.6 CLI Improvements

**Missing CLI features:**
- `fluxbase doctor` - Validate config, check database connectivity, verify security settings
- `fluxbase benchmark` - Run performance benchmarks
- `fluxbase export` - Export schema and/or data
- `fluxbase import` - Import data from CSV/JSON
- `fluxbase policy list` - List all RLS policies
- `fluxbase policy check` - Run security checks on policies

---

## 5. Feature Deep-Dive: Visual Table Viewer with Relations

### 5.1 Current State

**Backend (ready):**
- `internal/database/schema_inspector.go` returns complete `ForeignKey` data per table
- `ForeignKey` struct includes: `Name`, `ColumnName`, `ReferencedTable`, `ReferencedColumn`, `OnDelete`, `OnUpdate`
- Batch queries prevent N+1 when fetching FK metadata
- Schema cache (60s TTL) provides fast access
- API endpoint `/api/v1/admin/tables` returns all this data

**Frontend (data available but unused):**
- `TableInfo.foreign_keys` is typed as `unknown` - the data flows through but is never rendered
- Table viewer shows columns and data but no FK indicators
- No relation navigation or schema visualization

### 5.2 Proposed Implementation

#### Phase 1: FK Indicators in Table Viewer
- Add FK badge/icon on columns that are foreign keys
- Show tooltip: "→ users.id (ON DELETE CASCADE)"
- Click FK value → navigate to referenced record in referenced table
- Show reverse relationships: "Referenced by: orders.user_id (12 records)"

#### Phase 2: Schema Diagram / ERD View
- Add a "Schema" tab alongside "Data" tab in the tables page
- Use ReactFlow (or similar) for interactive canvas
- Render tables as nodes with columns listed inside
- Draw edges for FK relationships with cardinality indicators
- Color-code by schema (public=blue, auth=red, storage=green)
- Interactive: click table to navigate to data view
- Zoom, pan, auto-layout (dagre/elk algorithm)
- Export as PNG/SVG

#### Phase 3: Visual Schema Editor
- Drag to create new FK relationships between tables
- Right-click menu: add column, add index, add constraint
- Double-click table to edit columns/properties
- Undo/redo support for schema changes
- Generate migration SQL from visual changes
- Preview migration before applying

### 5.3 Backend Changes Needed

1. **New endpoint:** `GET /api/v1/admin/schema/relationships`
   - Returns all FK relationships across all schemas in a graph format
   - Includes reverse relationships (which tables reference a given table)
   - Optimized single-query implementation

2. **DDL handler extensions:**
   - `POST /api/v1/admin/tables/{schema}/{table}/foreign-keys` - Create FK
   - `DELETE /api/v1/admin/tables/{schema}/{table}/foreign-keys/{name}` - Drop FK
   - `POST /api/v1/admin/tables/{schema}/{table}/indexes` - Create index
   - `DELETE /api/v1/admin/tables/{schema}/{table}/indexes/{name}` - Drop index

3. **Type definitions:** Properly type `foreign_keys` and `indexes` in the API response

### 5.4 UI Component Architecture

```
SchemaPage
├── SchemaViewToggle (Data | ERD | Columns)
├── ERDCanvas (ReactFlow)
│   ├── TableNode (per table)
│   │   ├── ColumnList
│   │   │   ├── ColumnRow (name, type, PK icon, FK icon, nullable)
│   │   │   └── ...
│   │   └── TableActions (edit, delete, add column)
│   ├── RelationEdge (per FK)
│   │   └── EdgeLabel (constraint name, ON DELETE action)
│   └── CanvasControls (zoom, fit, layout, export)
├── RelationPanel (sidebar)
│   ├── OutgoingRelations (FKs from this table)
│   └── IncomingRelations (FKs pointing to this table)
└── SchemaSelector (filter by schema)
```

---

## 6. Feature Deep-Dive: Policy Editor & Security Warning System

### 6.1 Current State

**RLS is fully enforced at the database level:**
- `internal/middleware/rls.go` sets `LOCAL ROLE` and `request.jwt.claims` per request
- 100+ RLS policies defined across migrations 018-025
- Three roles: `anon`, `authenticated`, `service_role` (BYPASSRLS)
- Helper functions: `auth.current_user_id()`, `auth.current_user_role()`, `auth.is_admin()`
- Audit logging: `auth.rls_audit_log` tracks violations

**What's missing:**
- No policy introspection API (does not query `pg_policies`)
- No visual policy editor or management UI
- No security warnings for misconfigured policies
- Policy creation requires raw SQL in the SQL editor

### 6.2 Proposed Implementation

#### Phase 1: Policy Viewer & RLS Status

**New backend endpoint:** `GET /api/v1/admin/policies`
```sql
SELECT
    schemaname, tablename, policyname,
    permissive,  -- 'PERMISSIVE' or 'RESTRICTIVE'
    roles,       -- array of roles
    cmd,         -- ALL, SELECT, INSERT, UPDATE, DELETE
    qual,        -- USING expression
    with_check   -- WITH CHECK expression
FROM pg_policies
WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
ORDER BY schemaname, tablename, policyname
```

**New backend endpoint:** `GET /api/v1/admin/policies/{schema}/{table}`
Returns policies for a specific table plus RLS status.

**New backend endpoint:** `POST /api/v1/admin/policies/{schema}/{table}/toggle-rls`
Enables or disables RLS on a table.

**Admin UI - Policy Viewer page:**
- Table listing with RLS status toggle (enabled/disabled)
- Expandable rows showing policies per table
- Policy details: name, type (permissive/restrictive), roles, command, USING expression, WITH CHECK expression
- Color coding: green (has policies), yellow (RLS enabled but no policies), red (RLS disabled on public table)

#### Phase 2: Policy Editor

**New backend endpoint:** `POST /api/v1/admin/policies`
```json
{
  "schema": "public",
  "table": "posts",
  "name": "users_can_read_own_posts",
  "command": "SELECT",
  "permissive": true,
  "roles": ["authenticated"],
  "using": "auth.uid() = author_id",
  "with_check": null
}
```

**New backend endpoint:** `DELETE /api/v1/admin/policies/{schema}/{table}/{policy_name}`

**Admin UI - Policy Editor:**
- "Create Policy" dialog with:
  - Table selector (with schema)
  - Policy name input
  - Command selector: ALL, SELECT, INSERT, UPDATE, DELETE
  - Type: Permissive / Restrictive
  - Roles: Multi-select (anon, authenticated, custom roles)
  - USING expression: Code editor with syntax highlighting + helper function autocomplete
  - WITH CHECK expression: Same as above
- Policy templates (pre-built common patterns):
  - "User can only access own rows" → `auth.uid() = user_id`
  - "Authenticated users can read all" → `auth.role() = 'authenticated'`
  - "Admin full access" → `auth.is_admin()`
  - "Public read-only" → `true` (for SELECT only)
  - "Owner can modify" → `auth.uid() = owner_id`
- SQL preview: Show the exact `CREATE POLICY` statement before executing
- Edit existing policies (DROP + CREATE with new definition)

#### Phase 3: Security Warning System

**Security Advisor checks (automated):**

| Warning | Severity | Condition |
|---------|----------|-----------|
| RLS disabled on public table | Critical | `public.*` tables with `relrowsecurity = false` |
| No policies on RLS-enabled table | High | RLS enabled but `pg_policies` returns empty |
| USING (true) on non-SELECT policy | High | Policy allows unrestricted writes |
| Policy uses service_role | Medium | Service role bypasses RLS anyway |
| Missing WITH CHECK on INSERT/UPDATE | Medium | No write validation |
| Multiple permissive policies (OR) | Low | May grant broader access than intended |
| Subquery in policy (performance) | Low | N+1 risk in USING/WITH CHECK |
| Policy references non-existent column | Critical | Broken policy (runtime error) |
| Anon has write access | High | Anonymous users can modify data |

**Admin UI - Security Dashboard:**
- Top-level summary: "X tables secured, Y warnings, Z critical"
- Per-table breakdown with warning badges
- Severity filtering
- One-click fix suggestions (e.g., "Enable RLS" button, "Add basic policy" template)
- Scheduled background scan (every hour or on schema change)

#### Phase 4: Policy Testing UI

**Interactive policy tester:**
- Select a table and policy
- Choose a test role (anon, authenticated, specific user via impersonation)
- Run sample queries (SELECT, INSERT, UPDATE, DELETE)
- Show which rows pass/fail the policy
- Visual diff: "This user can see 10 rows, this user can see 3 rows"
- Uses existing impersonation infrastructure

### 6.3 Backend Changes Needed

1. **Policy introspection queries** - Query `pg_policies` and `pg_class` system catalogs
2. **Policy CRUD API** - Execute `CREATE POLICY`, `ALTER POLICY`, `DROP POLICY`
3. **Policy validation** - Parse and validate USING/WITH CHECK expressions before execution
4. **Security scanner service** - Periodic scan comparing `pg_policies` against best practices
5. **Extend schema cache** - Cache policy information alongside table metadata

### 6.4 UI Component Architecture

```
PoliciesPage
├── PolicyOverview
│   ├── SecurityScoreBadge (A/B/C/D/F)
│   ├── WarningCount (critical, high, medium, low)
│   └── QuickActions (fix all, scan now)
├── TablePolicyList
│   ├── TableRow (per table)
│   │   ├── RLSToggle (enable/disable)
│   │   ├── PolicyCount badge
│   │   ├── SecurityWarnings
│   │   └── ExpandedPolicies
│   │       ├── PolicyCard (per policy)
│   │       │   ├── PolicyHeader (name, type, command)
│   │       │   ├── PolicyRoles (role badges)
│   │       │   ├── PolicyExpression (syntax-highlighted USING/WITH CHECK)
│   │       │   └── PolicyActions (edit, delete, test)
│   │       └── AddPolicyButton
│   └── ...
├── PolicyEditor (dialog)
│   ├── PolicyForm
│   │   ├── TableSelector
│   │   ├── NameInput
│   │   ├── CommandSelect
│   │   ├── TypeToggle (permissive/restrictive)
│   │   ├── RoleMultiSelect
│   │   ├── UsingExpressionEditor (Monaco)
│   │   └── WithCheckExpressionEditor (Monaco)
│   ├── TemplateSelector
│   ├── SQLPreview
│   └── TestButton (run policy against sample data)
├── SecurityAdvisor (sidebar/panel)
│   ├── WarningList
│   │   ├── WarningCard (severity, description, fix action)
│   │   └── ...
│   └── ScanButton
└── PolicyTester (dialog)
    ├── RoleSelector
    ├── UserPicker (for impersonation)
    ├── QueryPreview
    └── ResultsTable (rows that pass/fail)
```

---

## 7. Feature Parity Matrix vs Supabase

| Feature | Supabase | Fluxbase | Gap |
|---------|----------|----------|-----|
| PostgreSQL Database | Full | Full | None |
| REST API (PostgREST-compatible) | Full | Full | None |
| Auth (JWT, OAuth, SAML, MFA) | Full | Full | None |
| Realtime (WebSocket) | Full | Full | None |
| Storage (S3/local) | Full | Full | None |
| Edge Functions (Deno) | Full | Full | None |
| GraphQL API | Full | Full | None |
| RLS Enforcement | Full | Full | None |
| Client SDKs (TypeScript) | Full | Full | None |
| MCP Server | Full | Full | None |
| Type Generation | Full | Full | None |
| OpenAPI Spec | Full | Full | None |
| **Visual Schema Designer/ERD** | **Full** | **None** | **Large** |
| **RLS Policy Editor** | **Full** | **None** | **Large** |
| **Security/Performance Advisor** | **Full** | **None** | **Medium** |
| **FK management in table editor** | **Full** | **None** | **Medium** |
| Foreign Data Wrappers | Full | None | Medium |
| Database Vault / Column Encryption | Full | Partial (app-level) | Small |
| Message Queues (pgmq) | Full | None (has jobs) | Small |
| Cron UI (pg_cron) | Full | Partial (app-level) | Small |
| Migration UI | Full | CLI only | Small |
| Backups / PITR | Full | None (infra-level) | Small |
| One-click Integrations | Partial | None | Small |
| Analytics Buckets / CDC | Alpha | None | Future |
| **User Impersonation (deep)** | Partial | **Full** | **Fluxbase wins** |
| **AI Chatbots / RAG** | Minimal | **Full** | **Fluxbase wins** |
| **Background Jobs (app-level)** | pg_cron/pgmq | **Full** | **Fluxbase wins** |
| **Database Branching (self-hosted)** | Cloud only | **Full** | **Fluxbase wins** |
| **Single Binary Deploy** | No (13+ containers) | **Yes** | **Fluxbase wins** |
| **RPC Functions Framework** | Basic | **Full** | **Fluxbase wins** |
| **Custom MCP Tools** | No | **Yes** | **Fluxbase wins** |

---

## 8. Prioritized Roadmap

### Tier 1: Critical (Security & Competitive Parity)

| # | Item | Type | Effort |
|---|------|------|--------|
| 1 | Fix SQL injection in `pubsub/postgres.go` | Security | Small |
| 2 | Add startup security config validation | Security | Small |
| 3 | Standardize error handling (use `SendError` consistently) | Maintainability | Medium |
| 4 | Visual Table Viewer with Relations (Phase 1: FK indicators) | Feature | Medium |
| 5 | Policy Viewer & RLS Status page | Feature | Medium |

### Tier 2: High Priority (Feature Completion)

| # | Item | Type | Effort |
|---|------|------|--------|
| 6 | Policy Editor (Phase 2: create/edit/delete policies) | Feature | Large |
| 7 | Visual Schema Diagram / ERD (Phase 2) | Feature | Large |
| 8 | Security Warning System (Phase 3) | Feature | Medium |
| 9 | Enhanced DDL operations (FK, indexes, constraints) | Feature | Medium |
| 10 | Handler test coverage improvement (auth, storage) | Maintainability | Medium |
| 11 | Fix rate limiting for multi-instance deployments | Security | Medium |

### Tier 3: Medium Priority (Developer Experience)

| # | Item | Type | Effort |
|---|------|------|--------|
| 12 | Go SDK implementation | DX | Large |
| 13 | Migration management UI | Feature | Medium |
| 14 | Data export/import in admin UI | Feature | Medium |
| 15 | React SDK coverage expansion | DX | Medium |
| 16 | TypeScript types for foreign_keys/indexes | DX | Small |
| 17 | Admin audit logging | Security | Medium |
| 18 | Request validation middleware | Maintainability | Small |

### Tier 4: Lower Priority (Polish & Differentiation)

| # | Item | Type | Effort |
|---|------|------|--------|
| 19 | Performance Advisor | Feature | Medium |
| 20 | Visual Schema Editor (Phase 3: drag-to-create) | Feature | Large |
| 21 | Policy Testing UI (Phase 4) | Feature | Medium |
| 22 | Database message queue (pgmq equivalent) | Feature | Large |
| 23 | Database Vault / column encryption | Feature | Large |
| 24 | Foreign Data Wrappers UI | Feature | Large |
| 25 | Identity Provider mode (OAuth2 server) | Feature | Large |
| 26 | Webhook retry dashboard | Feature | Small |
| 27 | CLI `doctor` and `policy` commands | DX | Small |
| 28 | Table viewer enhancements (resize, FK lookup) | DX | Medium |
| 29 | WebSocket token re-validation | Security | Small |
| 30 | Server struct decomposition | Maintainability | Medium |
