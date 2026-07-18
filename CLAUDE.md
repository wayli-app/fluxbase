# Fluxbase Codebase Guide

Fluxbase is a single-binary Backend-as-a-Service (BaaS). PostgreSQL is the only external dependency.

## Stack

- **Backend:** Go 1.25+, Fiber v3, pgx/v5, golang-migrate, TimescaleDB
- **Admin UI:** React 19, Vite, TanStack Router/Query, Tailwind v4, shadcn/ui
- **SDKs:** TypeScript (`sdk/`), React hooks (`sdk-react/`)
- **Functions Runtime:** Deno (JavaScript/TypeScript edge functions)

## Directory Structure

```
cmd/fluxbase/main.go     # Server entry point (setup + server creation)
 internal/api/module.go   # Module interface + ServiceRegistry (dependency injection)
internal/api/module_*.go # 20 business logic modules (one per domain)
internal/api/server.go   # Server struct, NewServer, module wiring
internal/api/server_init.go # Infrastructure: initCore
internal/api/server_middlewares.go # Middleware setup
internal/api/server_routes.go # Route setup
internal/database/retry.go      # ConnectWithRetry extracted from main.go
internal/tenantdb/bootstrap_keys.go # EnsureDefaultTenantAndKeys, EnsureServiceKey
internal/tenantdb/backfill.go   # BackfillTenantIDToDefault
internal/runtime/execute_helpers.go # buildDenoArgs, startAndStreamOutput
cli/cmd/                 # CLI commands (auth, functions, jobs, migrations, secrets)
internal/                # Core backend modules (see below)
admin/src/routes/        # Admin dashboard pages (file-based routing)
sdk/src/                 # TypeScript SDK source
deploy/helm/             # Kubernetes Helm charts
test/e2e/                # End-to-end tests
```

## Internal Modules (`internal/`)

| Module           | Purpose                                                                                                     |
| ---------------- | ----------------------------------------------------------------------------------------------------------- |
| `adminui/`       | Admin dashboard UI backend management                                                                       |
| `ai/`            | Vector search (pgvector), embeddings, knowledge bases, chatbots                                             |
| `api/`           | HTTP handlers (100+ files) - REST, GraphQL, storage, auth, DDL, webhooks, RPC, bulk operations, data export |
| `auth/`          | Authentication - JWT, OAuth2, OIDC, SAML SSO, magic links, MFA, CAPTCHA, impersonation                      |
| `branching/`     | Database branching - isolated DBs for dev/test environments                                                 |
| `config/`        | YAML + env var configuration loading                                                                        |
| `crypto/`        | Encryption utilities for secret storage                                                                     |
| `database/`      | PostgreSQL connection, schema introspection, migrations                                                     |
| `email/`         | SMTP, SendGrid, Mailgun, AWS SES providers                                                                  |
| `extensions/`    | PostgreSQL extension management system                                                                      |
| `functions/`     | Edge functions - Deno runtime, bundling, loader, scheduler                                                  |
| `jobs/`          | Background jobs - queue, workers, scheduler, progress tracking                                              |
| `logutil/`       | Log utilities (sanitization, formatting)                                                                    |
| `logging/`       | Structured logging with batching and retention policies                                                     |
| `mcp/`           | Model Context Protocol server for AI assistant integration                                                  |
| `middleware/`    | Auth, CORS, rate limiting, logging, branch and tenant context middlewares                                   |
| `migrations/`    | Database migration management                                                                               |
| `observability/` | Prometheus metrics and OpenTelemetry tracing                                                                |
| `pubsub/`        | Distributed pub/sub (local, PostgreSQL, Redis backends)                                                     |
| `query/`         | Shared query building types (FilterCondition, etc.)                                                         |
| `ratelimit/`     | Rate limiting service (memory, PostgreSQL, Redis backends)                                                  |
| `realtime/`      | WebSocket subscriptions via PostgreSQL LISTEN/NOTIFY                                                        |
| `rpc/`           | Remote procedure calls for database functions/procedures                                                    |
| `runtime/`       | Deno runtime wrapper for edge functions                                                                     |
| `scaling/`       | Horizontal scaling and leader election                                                                      |
| `secrets/`       | Secret management for functions/jobs                                                                        |
| `settings/`      | Application settings and custom configuration                                                               |
| `storage/`       | File storage abstraction (local filesystem or S3/MinIO)                                                     |
| `tenantdb/`      | Tenant database routing, FDW connections, separate tenant databases                                         |
| `testcontext/`   | Test context utilities for E2E tests                                                                        |
| `testutil/`      | Test utilities and helpers                                                                                  |
| `webhook/`       | Webhook system for database events (INSERT, UPDATE, DELETE)                                                 |

### Module System (`internal/api/module_*.go`)

Server initialization uses a module-based dependency injection system:

- **`Module` interface** — `Name()` + `Init(ctx, *ServiceRegistry) error`
- **`Shutdowner` interface** — optional `Shutdown(ctx context.Context) error`; modules with stoppable resources implement this for clean teardown
- **`ServiceRegistry`** — stores config, DB, PubSub, RateLimiter, Metrics, and registered services; modules read dependencies via `GetService[T]()`
- **20 modules** in dependency order: Email → Secrets → Storage → Logging → Auth → Webhook → Extensions → Tenancy → Settings → Schema → Functions → Jobs → RPC → AI → Realtime → Branching → GraphQL → MCP → Metrics → BackgroundServices
- Modules shut down in reverse order via `ShutdownModules()`
- `GetService[T]()` uses `reflect.TypeOf` — only works with concrete pointer types (e.g., `*auth.Service`), not interfaces. Use `registry.PubSub` field for `pubsub.PubSub`
- Modules register outputs via `registry.Register()` so downstream modules can find them
- **Deferred email resolution:** `EmailModule` registers a `*email.LazyService` that `AuthModule` and `TenancyModule` use as their `email.Service`. `SettingsModule` creates the real `*email.Manager` (with all deps) and resolves the lazy service. This breaks the Email→Auth→Settings dependency cycle.

**Handler group wiring:** Each module creates its handler group struct during `Init()` and stores it in `m.Handlers`. After `InitModules()`, `NewServer()` assigns `s.X = xMod.Handlers` for each module (~20 lines). Modules with cross-group outputs (Settings→Email/Captcha, AI→Quota, Realtime→Monitoring, Tenancy→Middleware) populate additional handler groups. A few modules have direct Server fields (Auth→`sqlHandler`/`requireAuth`/`optionalAuth`, Schema→`rest`).

**Remaining sideways mutations (by design):** 4 setter calls remain for runtime concerns: `CaptchaService.ReloadFromSettings` (hot-reload), `DDLHandler.SetGraphQLInvalidator` (optional callback), `ChatHandler.SetMCPToolRegistry/SetMCPResources` (late-bound capabilities).

**Remaining Server methods (8, down from 37):** `Start`/`Shutdown`/`App` (lifecycle), `handleHealth` (uses `s.db`), `createMCPAuthMiddleware` (cross-group middleware factory), `SetTenantConfigLoader` (public API from main.go), `GetStorageService`/`GetAuthService` (unexported handler field accessors).

**Key files:** `module.go` (interface + registry), `module_*.go` (one per module), `server.go` (wiring + Shutdown), `server_init.go` (initCore), `server_middlewares.go` (middleware setup), `server_routes.go` (route setup), `routes_*.go` (per-domain route deps), `schema_admin_handler.go` (schema inspection), `schema_handler.go` (schema graph), `policy_handler.go` (RLS policies)

## Database Schemas

- `auth.*` - Users, sessions, identities, client keys
- `storage.*` - Buckets, objects, access policies
- `jobs.*` - Background job storage
- `functions.*` - Edge functions registry
- `branching.*` - Database branch metadata, access control, GitHub config
- `ai.*` - Knowledge bases, documents, chatbots, permissions
- `logging.*` - Centralized logging entries with TimescaleDB hypertable support
- `platform.*` - Multi-tenancy (tenants, service_keys, tenant_admin_assignments, users)
- `public` - User application tables

**Tenant Isolation:** All tenant-scoped tables use Row Level Security (RLS) with the `tenant_service` role for automatic tenant isolation. The `platform.tenants` table stores tenant metadata, and `platform.service_keys` manages API keys per tenant.

## Key Files by Feature

**Authentication:**

- `internal/auth/service.go` - Main auth logic
- `internal/auth/jwt.go` - Token management (only "fluxbase" issuer accepted)
- `internal/auth/scopes.go` - Authorization scopes (JWT auth bypasses scopes; scopes are for API keys only)
- `internal/auth/security_events.go` - Security event logging (extracts tenant_id from context)
- `internal/api/auth_*.go` - Auth HTTP handlers

**REST API:**

- `internal/api/rest_crud.go` - CRUD operations
- `internal/api/query_parser.go` - URL query parsing
- `internal/api/query_builder.go` - SQL generation

**Edge Functions:**

- `internal/functions/handler.go` - Function HTTP handler
- `internal/functions/loader.go` - Load functions from disk
- `internal/runtime/runtime.go` - Deno runtime wrapper

**Background Jobs:**

- `internal/jobs/manager.go` - Job orchestration
- `internal/jobs/worker.go` - Job execution
- `internal/jobs/scheduler.go` - Cron scheduling

**Storage:**

- `internal/storage/service.go` - Storage abstraction
- `internal/api/storage_*.go` - Upload/download handlers

**Realtime:**

- `internal/realtime/manager.go` - Main connection manager
- `internal/realtime/connection.go` - Client connection handling
- `internal/realtime/subscription.go` - Subscription management
- `internal/realtime/presence.go` - User online status tracking

**MCP Server:**

- `internal/mcp/server.go` - JSON-RPC 2.0 protocol handler
- `internal/mcp/handler.go` - HTTP transport layer
- `internal/mcp/auth.go` - Auth context and scope checking
- `internal/mcp/tools/` - Tool implementations (query, storage, functions, jobs, vectors)
- `internal/mcp/resources/` - Resource providers (schema, functions, storage, rpc)

**Database Branching:**

- `internal/branching/manager.go` - Branch lifecycle, tenant-aware clone source resolution, FDW repair
- `internal/branching/storage.go` - Branch metadata CRUD (tenant-scoped via RLS)
- `internal/branching/router.go` - Connection pool per branch
- `internal/branching/types.go` - `TenantResolver`, `FDWRepairer` interfaces, tenant-aware DB name generation
- `internal/api/branch_handler.go` - REST API for branch management
- `internal/api/github_webhook_handler.go` - GitHub PR automation
- `internal/middleware/branch.go` - Branch context extraction
- `internal/tenantdb/fdw.go` - `GetFDWRoleForTenant`, `RepairFDWForBranch` for branch FDW repair
- `cli/cmd/branch.go` - CLI commands

**GraphQL:**

- `internal/api/graphql_handler.go` - GraphQL HTTP handler
- `internal/api/graphql_resolvers.go` - Query/mutation resolvers
- `internal/api/graphql_schema.go` - Schema generation

**RPC/Procedures:**

- `internal/rpc/service.go` - Procedure execution
- `internal/api/rpc_handler.go` - RPC HTTP handlers

**Webhooks:**

- `internal/webhook/service.go` - Webhook delivery
- `internal/api/webhook_handler.go` - Webhook management API

**Observability:**

- `internal/observability/metrics.go` - Prometheus metrics
- `internal/observability/tracing.go` - OpenTelemetry tracing

**Multi-Backend Logging:**

- `internal/storage/log_service.go` - Main log service orchestration
- `internal/storage/log_postgres.go` - PostgreSQL native backend
- `internal/storage/log_timescaledb.go` - TimescaleDB backend with compression
- `internal/storage/log_s3.go` - S3/MinIO cloud storage backend
- `internal/storage/loki.go` - Loki integration

**Enhanced AI/Knowledge Base:**

- `internal/ai/knowledge_base.go` - Core data models (incl. per-KB `EntityExtractionEnabled` toggle)
- `internal/ai/knowledge_base_storage.go` - Storage operations
- `internal/ai/provider_anthropic.go` - Native Anthropic Claude provider with explicit `cache_control` prompt caching; supports `tool_choice` (`auto`/`any`/`tool`)
- `internal/ai/provider_openai.go` - OpenAI/Azure provider (parses `prompt_tokens_details.cached_tokens` from automatic prefix caching)
- `internal/ai/chat_handler_message.go` - Turns assemble a static system message + dynamic context message (user ID, time, RAG) so the static prefix is byte-stable across turns for caching. Branches on `ReasoningMode`: `"supervisor"` (default) delegates to the multi-agent graph in `chat_handler_supervisor.go`; `"react"`/`"strict"`/`"none"` use the legacy ReAct loop.
- `internal/ai/chat_handler_supervisor.go` - `runSupervisorTurn` builds the `AgentDeps` bundle, runs the supervisor graph, streams the final response, and falls back to the legacy ReAct loop on any graph error.
- `internal/ai/chat_handler_tools.go` - `execute_sql` / MCP tool execution; both paths emit `query_result` events for parity
- `internal/ai/chat_handler_usage.go` - `GET /api/v1/ai/usage/:chatbotId` (per-user daily quota snapshot)
- `internal/ai/chatbot_limiter.go` - In-memory per-`(chatbotID, userID)` counters; `GetDailyUsage` returns a `DailyUsage` snapshot; `AddTokenUsage` is called with cached-token-discounted spend
- `internal/ai/intent_validator.go` - `GetMatchedRules` surfaces fired rules via the done event's `matched_intent_rules`

**Multi-Agent Supervisor (default ReasoningMode for all chatbots):**

- `internal/ai/graph.go` - Generic graph executor: nodes, edges, conditional edges, parallel fan-out, cycle detection, mutex-safe `State` with typed accessors.
- `internal/ai/supervisor_graph.go` - Concrete graph wiring for one chatbot turn: supervisor → router → specialists → synthesizer → verifier.
- `internal/ai/agent_deps.go` - `AgentDeps` bundle (chatbot, provider, services, sender) + `AgentEventSender` interface for WS events.
- `internal/ai/agent_supervisor.go` - Routing LLM call. Parses JSON `SupervisorPlan` (route, language, investigative flag, min_tool_calls). Applies page-profile whitelist.
- `internal/ai/agent_specialists.go` - SQL, KB, Action, Chat agents. Each has a focused prompt + tool whitelist + bounded internal loop.
- `internal/ai/agent_synthesizer_verifier.go` - Synthesizer (merges multi-agent output) + Verifier (Unicode-script language check + optional LLM grounding check on investigative turns).
- `internal/ai/agent_prompts.go` - Per-agent static system prompts (focused ~100 lines each).
- `internal/ai/page_context.go` - `PageProfile` + `PageProfiles` map + `@fluxbase:page-contexts` JSON parser. Level-2 page-aware chatbot overrides.
- Reasoning modes: `"supervisor"` (default for new + existing chatbots via `PopulateDerivedFields`), `"react"`, `"strict"`, `"none"`. Pin via `@fluxbase:reasoning-mode` annotation.

**Multi-Tenancy:**

- `internal/api/tenant_handler.go` - Tenant CRUD HTTP handlers
- `internal/api/servicekey_handler.go` - Service key management API
- `internal/middleware/tenant.go` - Tenant context extraction middleware
- `internal/database/schema/schemas/platform.sql` - Platform schema with tenants table (declarative)

**Migrations:**

- `internal/migrations/handler.go` - Migrations API HTTP handlers (CRUD, apply, rollback, sync)
- `internal/migrations/executor.go` - Main database migration execution (`ExecuteWithAdminRole`)
- `internal/migrations/tenant_executor.go` - Tenant-scoped execution (`SET LOCAL ROLE tenant_migration_role`)
- `internal/migrations/storage.go` - Migration metadata CRUD (`migrations.app` table)
- `internal/migrations/declarative.go` - Declarative schema service (pgschema plan/apply/dump)
- `internal/tenantdb/declarative.go` - Per-tenant declarative schema service
- `internal/api/routes/migrations.go` - Migration route definitions (`/api/v1/admin/migrations`)
- `internal/middleware/migrations_security.go` - Migrations API auth, IP allowlist, rate limiting
- `internal/database/connection.go` - Filesystem migration runner (`runUserMigrations`)
- `cli/cmd/migrations.go` - CLI commands (`fluxbase migrations sync/list/create/apply/rollback`)

## Branching + Multi-Tenancy Interaction

### Pool Priority Chain

When a request carries both `X-FB-Tenant` and `X-Fluxbase-Branch` headers, the pool selection follows this priority:

```
1. Branch pool (branch-specific database)
2. Tenant pool (tenant's separate database via FDW)
3. Main pool (shared application database)
```

Implemented in `internal/middleware/tenant_db.go:GetPoolForSchema()`.

### Branch Clone Source

When creating a branch for a tenant that has a separate database (`DBName` is set), the branch is cloned from the **tenant's database** (not the main database). For the default tenant (no separate DB), branches clone from the main database as before.

Key interfaces:
- `branching.TenantResolver` — resolves tenant database info for branch cloning
- `branching.FDWRepairer` — repairs FDW mappings after cloning a tenant database

The clone flow:
1. `Manager.resolveTemplateDatabase()` checks if the tenant has a separate DB via `TenantResolver`
2. If yes, uses `CREATE DATABASE ... TEMPLATE <tenant_db>` instead of the main DB
3. After cloning, `repairFDW()` recreates the FDW user mapping in the branch DB using the tenant's FDW role credentials

### FDW + Branching

Tenant databases use `postgres_fdw` to access shared schemas (auth, storage, branching, etc.) from the main database. When a tenant database is cloned for a branch:
- Foreign table definitions are copied but user mappings may be stale
- `tenantdb.Manager.RepairFDWForBranch()` recreates the user mapping with the tenant's FDW role
- The branch's FDW connection correctly enforces RLS with the tenant's `app.current_tenant_id`

FDW schemas imported into tenant databases: `platform`, `auth`, `storage`, `jobs`, `functions`, `realtime`, `ai`, `rpc`, `branching`, `logging`, `mcp`

### Default Tenant

- Slug is hardcoded as `"default"`, identified by `is_default = true` in `platform.tenants`
- `FLUXBASE_TENANTS_DEFAULT_NAME` configures the display name only (not the slug)
- Uses the main database pool (no separate database, no FDW setup needed)
- `UsesMainDatabase()` returns true when `DBName` is nil or empty

### Route Registry Tenant Coverage

Most route groups include `TenantMiddleware` for tenant context. Groups without it are intentionally tenant-agnostic:
- `health` — system-level health checks
- `dashboard-auth` — pre-authentication endpoints
- `github-webhook` — uses repository-to-tenant mapping (no user auth)
- `internal-ai` — server-internal only (`RequireInternal`)
- `invitations` — token-based, pre-tenant-context
- `settings` (base) — global app settings via RLS

### Key Branching + Tenancy Files

- `internal/branching/manager.go` - Branch lifecycle, tenant-aware clone source resolution
- `internal/tenantdb/fdw.go` - FDW setup/repair, `GetFDWRoleForTenant`, `RepairFDWForBranch`
- `internal/api/server.go` - Wires `TenantResolver` and `FDWRepairer` to branch manager
- `internal/middleware/tenant_db.go` - Pool priority (branch > tenant > main)

## Common Commands

### Devcontainer Database Access

When working in the devcontainer, you can use `psql` to query the database directly. The connection credentials are available from environment variables:

```bash
# Connect to the database
psql "postgresql://$FLUXBASE_DATABASE_USER:$FLUXBASE_DATABASE_PASSWORD@$FLUXBASE_DATABASE_HOST:$FLUXBASE_DATABASE_PORT/$FLUXBASE_DATABASE_NAME"
```

Useful queries for debugging:

```sql
-- List all tables in public schema
SELECT tablename FROM pg_tables WHERE schemaname = 'public';

-- Check service_role permissions on public tables
SELECT table_name, privilege_type
FROM information_schema.role_table_grants
WHERE grantee = 'service_role' AND table_schema = 'public';

-- Check tenants
SELECT id, name, slug FROM platform.tenants;

-- Find leftover test tables
SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND tablename LIKE 'test_%';
```

### Build & Development

```bash
# Development
make dev              # Start backend + admin UI dev servers
make build            # Production build with embedded admin

# Database Operations
make db-reset         # Reset database (preserve user data)
make db-reset-full    # Full reset (destroys all data, bootstrap runs on next server start)

# Testing
make test             # Unit tests only (2min)
make test-coverage    # Unit tests with coverage report and enforcement
make test-coverage-unit # Unit tests with coverage (excludes e2e)
make test-full        # All tests including E2E (10min+)
make test-coverage-check  # Check coverage thresholds without running tests
make test-auth        # Authentication tests
make test-rls         # RLS security tests
make test-rest        # REST API tests
make test-storage     # Storage tests
make test-e2e         # E2E tests only
make test-e2e-fast   # Fast E2E tests

# SDK Tests
make test-sdk         # TypeScript SDK tests
make test-sdk-react   # SDK React build and type check

# Code Quality
make lint-go          # Go linting with golangci-lint
make lint-typescript  # TypeScript linting (admin UI + SDKs)

# CLI
make cli-install      # Build and install CLI

# Setup
make setup-dev        # Install dependencies + git hooks
```

## Configuration

Three-layer system: defaults → `fluxbase.yaml` → `FLUXBASE_*` env vars

Key config sections: server, database, auth, storage, realtime, functions, jobs, email, ai, mcp, branching, graphql, rpc, scaling, observability (metrics, tracing), security, cors, api, logging, migrations, tenants

**Database Configuration (relevant to migrations):**

```yaml
database:
  user_migrations_path: "/migrations/user"  # Local path for filesystem migrations (env: FLUXBASE_DATABASE_USER_MIGRATIONS_PATH)
```

**Migrations API Configuration:**

```yaml
migrations:
  enabled: true
  allowed_ip_ranges: []  # IP CIDR allowlist (default: Docker/private networks)
```

**Tenant Declarative Schema Configuration:**

```yaml
tenants:
  declarative:
    enabled: false               # Disabled by default
    schema_dir: ""               # Directory: {schema_dir}/{tenant-slug}/public.sql
    on_create: false             # Apply on tenant creation
    on_startup: false            # Apply on server startup
    allow_destructive: false     # Allow DROP/ALTER in tenant schemas
```

**MCP Configuration:**

```yaml
mcp:
  enabled: true
  base_path: /mcp
  rate_limit_per_min: 100
  allowed_tools: [] # Empty = all tools
  allowed_resources: [] # Empty = all resources
```

**Branching Configuration:**

```yaml
branching:
  enabled: true
  max_branches_per_user: 5
  max_branches_per_tenant: 0      # 0 = unlimited
  max_total_branches: 50
  default_data_clone_mode: schema_only
  auto_delete_after: 0            # 0 = never
  database_prefix: branch_
  default_branch: main
  max_total_connections: 500
  pool_eviction_age: 1h
```

When multi-tenancy is enabled, branches for tenants with separate databases clone from the tenant's database (not the main DB). The FDW user mapping is automatically repaired after cloning.

**Logging Backend Configuration:**

```yaml
logging:
  backend: postgres # postgres, timescaledb, s3, local, loki, elasticsearch, clickhouse
  batch_size: 100
  flush_interval: 1s
  retention_days: 90
  compression_days: 7 # For TimescaleDB
  s3:
    bucket: logs
    prefix: logs/
```

## Code Quality Standards

**MANDATORY REQUIREMENTS:** All code must pass these checks before committing.

### Go Code Quality

```bash
# Formatting (REQUIRED)
go fmt ./...

# Linting (REQUIRED - must pass)
golangci-lint run ./...

# Type Checking
golangci-lint run ./...  # Includes type checking
```

**What gets checked:**

- **gofmt**: Standard Go formatting (auto-fixed by pre-commit hook)
- **golangci-lint**: Comprehensive linting including:
  - gocritic: Code improvement suggestions
  - misspell: Spell checking
  - govet: Static analysis
  - type checking: Type safety verification

**Configuration:** `.golangci.yml`

- Timeout: 5 minutes
- Tests included
- Integration build tags enabled

### TypeScript Code Quality

```bash
# Admin UI
cd admin && bun run type-check
cd admin && bun run lint

# SDK
cd sdk && bun run type-check
cd sdk && bun run lint

# SDK React
cd sdk-react && bun run type-check  # Uses tsc --noEmit
```

**What gets checked:**

- **ESLint**: TypeScript ESLint, React Hooks, React Refresh, TanStack Query
- **Prettier**: Code formatting with import sorting and Tailwind integration
- **TypeScript**: No unused vars, type-only imports enforced
1
### Pre-Commit Hook Enforcement

Git pre-commit hooks automatically run:

1. `go fmt ./...` - Auto-stages formatted files
2. `golangci-lint run ./...` - Blocks commit if fails
3. Admin UI type-check - Blocks commit if fails
4. SDK type-check - Blocks commit if fails
5. SDK React type-check - Blocks commit if fails

### CI/CD Enforcement

- **Go**: Formatting check + golangci-lint must pass
- **TypeScript**: ESLint must pass
- **Tests**: Coverage thresholds enforced (25% overall)
- **Build**: Cross-platform builds (Linux/amd64 + Linux/arm64)

## Patterns

- Interface-based dependency injection
- Handler pattern with `*fiber.Ctx`
- Repository pattern for data access
- PostgreSQL Row Level Security (RLS) for authorization
- PostgREST-compatible REST API conventions

## Security Hardening

### Sensitive Value Handling

- **OTP codes** are stored as SHA-256 hashes (`auth.otp_codes.code_hash`). The `PlaintextCode` field uses `json:"-"` to prevent API leakage
- **Invitation tokens** are stored as SHA-256 hashes (`platform.invitation_tokens.token_hash`). Dual-read with lazy migration supports existing plaintext tokens during upgrade
- **Edge function env vars**: `internal/runtime/env.go` blocks sensitive vars (`FLUXBASE_DATABASE_URL`, `FLUXBASE_AUTH_JWT_SECRET`, email API keys, etc.) from being passed to Deno functions
- **Function update columns**: `internal/functions/storage.go` uses a whitelist (`allowedFunctionColumns`) to prevent overwriting protected columns (`id`, `tenant_id`, `created_at`, `updated_at`)

### Database Connection Safety

- **Pool mutex**: `internal/database/connection.go` uses `sync.RWMutex` on all pool access (`BeginTx`, `Query`, `Exec`, `Stats`, `Pool`). `Close()` acquires write lock, nils pool, then closes outside lock
- **Advisory locks**: Migrations use `pg_try_advisory_xact_lock` (transaction-scoped) to prevent concurrent migration execution. Lock auto-releases on commit/rollback

### Admin UI Auth

- **Single source of truth**: Zustand store (`admin/src/stores/auth-store.ts`) manages tokens with safe cookie parsing (try/catch with cleanup on malformed cookies)
- **Retry guard**: Consolidated `refreshAndRetry()` helper handles all auth refresh paths (401, auth-like response body) with deduplication via `failedQueue`
- **Error helper**: `admin/src/lib/get-error-message.ts` centralizes error extraction from Axios responses

### Path Safety

- **Log file paths**: `internal/storage/log_local.go` validates components via `validatePathComponent` (rejects `..`, `/`, null bytes, absolute paths)
- **SQL substitution**: `internal/database/bootstrap/substitute.go` validates `APP_USER` identifier with `^[a-zA-Z_][a-zA-Z0-9_]*$` before SQL substitution
- **Prometheus metrics**: `normalizePath` replaces UUIDs, numeric IDs, and slug-like segments with `:id` to prevent cardinality explosion

### SAML

- **Logout signature verification**: Enabled by default. `RequireLogoutSignature` config can disable for development
- **Nil-safe parsing**: `ParseLogoutRequest`/`ParseLogoutResponse` guard against nil `NameID`, `Issuer`, `Status` fields

### Email

- **HTML escaping**: All dynamic values in email templates pass through `html.EscapeString` to prevent injection

### SSRF Protection

- **BlockedDomains**: `DefaultFunctionPermissions()` and `DefaultJobPermissions()` include blocked domains for metadata endpoints (`169.254.169.254`, `metadata.google.internal`, etc.). Applied to all function, job, and MCP executions
- **Filesystem restriction**: `--allow-read` and `--allow-write` are restricted to `/tmp` (not full filesystem)

### Realtime Tenant Isolation

- **Broadcast filtering**: `BroadcastToChannel` and `BroadcastGlobal` filter by `TenantID` — messages only delivered to connections of the same tenant
- **Presence filtering**: Presence join/leave/sync events scoped by tenant
- **GlobalBroadcast**: Includes `TenantID` field for cross-instance propagation via pub/sub

### Leader Election

- **Dedicated connection**: `internal/scaling/leader.go` uses a dedicated `*pgxpool.Conn` (not the pool) for advisory locks, preventing premature lock release
- **Idempotent start**: `Start()` guarded by `started` flag to prevent duplicate election loops

## Migrations

Fluxbase uses a **hybrid migration system** with three subsystems:

### Internal Schema (Declarative)

Internal Fluxbase tables (auth, storage, functions, jobs, etc.) are managed declaratively:

- **Bootstrap:** `internal/database/bootstrap/bootstrap.sql` - Creates schemas, extensions, roles, default privileges
- **Schema files:** `internal/database/schema/schemas/*.sql` - Declarative SQL files for each schema
- **Applied automatically:** Server applies bootstrap + declarative schema on startup

### User Schema - Imperative Migrations

Imperative migrations are tracked in the `migrations.app` table and can be delivered via a local filesystem path or the API:

**Local Filesystem:**

- Config: `database.user_migrations_path` (default: `/migrations/user`)
- Env var: `FLUXBASE_DATABASE_USER_MIGRATIONS_PATH`
- Files: `{name}.up.sql` / `{name}.down.sql` pairs, sorted alphabetically (e.g. `001_create_users.up.sql`)
- Applied at startup against the main database as admin user
- Tracked with `namespace='filesystem'`
- SQL validated via `pg_query.Parse()` before execution

**Migrations API (`/api/v1/admin/migrations`):**

- Requires service key or `service_role` JWT with `admin`, `instance_admin`, or `tenant_admin` role
- Endpoints: CRUD, apply, rollback, apply-pending, sync
- `POST /sync` accepts a batch of migrations, deduplicates by SHA256 of up/down SQL, and optionally auto-applies
- Configurable namespaces (default: `"default"`)

**Tenant-aware routing (backward compatible):**

- **No tenant context** (no `X-FB-Tenant` header, not `tenant_admin` JWT): runs via `db.ExecuteWithAdminRole()` against the main database with full DDL privileges
- **Default tenant** (`X-FB-Tenant` points to default tenant): `Router.GetPool()` returns the main pool, `TenantExecutor` runs with `SET LOCAL ROLE tenant_migration_role` (restricted to `public` schema)
- **Named tenant**: `Router.GetPool()` returns a pool to the tenant's separate database, same `tenant_migration_role` restriction to `public` schema

### User Schema - Declarative (pgschema)

Per-tenant declarative schema management using pgschema for diff-based application to the `public` schema:

**Local Filesystem:**

- Config: `tenants.declarative.schema_dir`
- Structure: `{SchemaDir}/{tenant-slug}/public.sql`
- Applied on tenant creation (`on_create`), server startup (`on_startup`), or on-demand via API

**Tenant Schema API:**

- `GET /tenants/:id/schema` - Get schema status and pending changes
- `POST /tenants/:id/schema/content` - Upload schema SQL (stored in `platform.tenant_schemas`)
- `POST /tenants/:id/schema/content/apply` - Diff and apply uploaded content via pgschema
- `POST /tenants/:id/schema/apply` - Apply from local filesystem
- `DELETE /tenants/:id/schema/content` - Delete stored schema

Works for the default tenant too — `Router.GetPool()` returns the main pool when `UsesMainDatabase()` is true.

### Common Commands

```bash
# Database Operations
make db-reset         # Reset database (preserve user data)
make db-reset-full    # Full reset - bootstrap runs on next server start

# CLI Migrations (interact with server API)
fluxbase migrations list [--namespace]      # List migrations
fluxbase migrations create <name>           # Create migration
fluxbase migrations apply <name>            # Apply a migration
fluxbase migrations rollback <name>         # Rollback a migration
fluxbase migrations apply-pending           # Apply all pending
fluxbase migrations sync [--dir] [--namespace] [--no-apply]  # Sync from directory
```

## Testing

### Test Organization

- Unit tests: `*_test.go` alongside source
- E2E tests: `test/e2e/`
- Test helpers: `internal/testutil/`

### Coverage Targets

- **Overall:** 25%+ (starting point, will increase)
- **Core business logic:** 50%+ per file
- **Critical modules (auth, API):** 70%+ per file

### Excluded from Coverage

Files containing only type definitions, interfaces, or requiring external system dependencies are excluded from coverage calculations. See [.testcoverage.yml](.testcoverage.yml) for the complete list:

- Pure type definition files (e.g., `internal/*/types.go`, `internal/*/errors.go`)
- Interface-only files (e.g., `internal/auth/interfaces.go`)
- Infrastructure code requiring system dependencies (leader election, database connections, OCR)
- CLI commands (tested via integration tests)
- Entry points and embedded assets

### Running Tests

```bash
make test             # Unit tests only (2min)
make test-coverage    # Unit tests with coverage report and enforcement
make test-full        # All tests including E2E (10min)
make test-coverage-check  # Check coverage thresholds without running tests
```

### Coverage Enforcement

Coverage thresholds are enforced in CI via [go-test-coverage](https://github.com/vladopajic/go-test-coverage). Pull requests must meet minimum thresholds for affected files. The tool automatically excludes files that shouldn't be counted (pure type definitions, infrastructure code, etc.).

### Playwright UI Tests

Playwright tests cover the admin UI with a real browser. They run against a dedicated `fluxbase_playwright` database to avoid interfering with development data.

**Test Structure:**

Tests are in `admin/tests/e2e/` with a 3-phase pipeline:

1. **Setup** (`setup.spec.ts`) — Creates the admin user on a fresh database
2. **Provisioning** (`_provisioning.spec.ts`) — Creates tenants, tenant admin user, test data
3. **E2E tests** (all other `*.spec.ts`) — Run with provisioned data

**Running Tests:**

```bash
make test-e2e-ui           # Reset DB, start server, run all tests
make test-e2e-ui-headed    # Run with visible browser
make test-e2e-ui-debug     # Debug mode
make test-e2e-ui-dev       # Reuse running server for fast iteration
make test-e2e-ui-server    # Start test servers (Go :8082 + Vite :5050)
```

**Key Test Files:**

| File | Coverage |
|---|---|
| `setup.spec.ts` | Initial setup/onboarding |
| `login.spec.ts` | Login/logout flows |
| `auth-guard.spec.ts` | Route protection, token expiry |
| `middleware.spec.ts` | Auth headers, JS console errors |
| `tenant-crud.spec.ts` | Create/edit/delete tenants via UI |
| `tenant-switching.spec.ts` | Tenant selector, X-FB-Tenant propagation |
| `tenant-default.spec.ts` | Default tenant behavior |
| `tenant-keys.spec.ts` | Service key CRUD per tenant |
| `tenant-keys-isolation.spec.ts` | Key visibility/usage/lifecycle isolation |
| `tenant-admin-isolation.spec.ts` | Tenant admin JWT claims, route access, API isolation |
| `tenant-service-isolation.spec.ts` | Cross-tenant isolation for functions, secrets, knowledge bases |
| `tenant-service-admin-isolation.spec.ts` | Service isolation from tenant admin perspective |
| `tenant-members.spec.ts` | Member add/list/remove |
| `impersonation.spec.ts` | Impersonation UI flow (header selector + inline popover) |
| `impersonation-tenant-isolation.spec.ts` | Impersonation + tenant isolation |
| `functions-execution.spec.ts` | Edge function creation, invocation, deletion via UI |
| `jobs-execution.spec.ts` | Background jobs management via UI |
| `chatbots-execution.spec.ts` | Chatbot management via UI |

**Test Infrastructure:**

- `fixtures.ts` — Playwright fixtures (`adminPage`, `tenantAdminPage`, `adminToken`, `tenantAdminToken`, etc.)
- `helpers/api.ts` — API request helpers with tenant-scoped variants (`rawXxx` for fetch, `xxx` for Playwright context)
- `helpers/db.ts` — Direct PostgreSQL helpers for verifying database state
- `helpers/constants.ts` — Shared test credentials and tenant slugs
- `helpers/selectors.ts` — Common UI selectors and navigation helpers
- `helpers/mailhog.ts` — Email testing helpers
- `scripts/start-e2e-ui.sh` — Starts Go backend (:8082) + Vite (:5050) against `fluxbase_playwright` DB

**Writing New Tests:**

1. Add API helpers to `helpers/api.ts` if the endpoint isn't covered
2. Add fixtures to `fixtures.ts` if you need new authenticated contexts
3. Create `*.spec.ts` following the existing naming conventions
4. Use `rawXxx` helpers for API setup/teardown, browser interactions for UI verification
5. Track created resources for cleanup in `afterAll`

## Development Workflow Requirements

### Writing Tests

**IMPORTANT:** When making code changes, always consider writing or updating tests:

1. **New features** - Write unit tests covering the main functionality and edge cases
2. **Bug fixes** - Add a regression test that would have caught the bug
3. **Refactoring** - Ensure existing tests still pass; add tests if coverage gaps exist

**Test file locations:**

- Unit tests: Place `*_test.go` files alongside the source file being tested
- E2E tests: Add to `test/e2e/` for integration scenarios
- Test helpers: Use `internal/testutil/` for shared test utilities

**Test naming conventions:**

```go
func TestFunctionName_Scenario_ExpectedBehavior(t *testing.T)
// Example: TestCreateBranch_ExceedsUserLimit_ReturnsError
```

**When to skip tests:**

- Pure type definitions or interface files
- Simple configuration structs with no logic
- Code that only wraps external dependencies (but do test the integration)

### Updating Documentation

**IMPORTANT:** When making code changes, always consider updating documentation:

1. **New features** - Add documentation in `docs/src/content/docs/guides/`
2. **API changes** - Update SDK documentation in `docs/src/content/docs/api/`
3. **Configuration changes** - Update the relevant guide and CLAUDE.md if needed
4. **Breaking changes** - Document migration steps clearly

**Documentation locations:**

- Feature guides: `docs/src/content/docs/guides/<feature>.md`
- API reference: `docs/src/content/docs/api/` (auto-generated from SDK)
- Project overview: `CLAUDE.md` (this file)

**Documentation checklist:**

- [ ] Does the feature documentation match the implementation?
- [ ] Are all configuration options documented?
- [ ] Are error messages and edge cases explained?
- [ ] Are code examples correct and runnable?

### Pre-Commit Checklist

Before committing changes, verify:

1. `go fmt ./...` passes (or auto-fixed by hook)
2. `golangci-lint run ./...` passes
3. TypeScript type-check passes (admin UI + SDKs)
4. `make test` passes
5. Documentation is updated for user-facing changes
6. New tests are added for new functionality
7. Existing tests are updated if behavior changed

## Browser Testing with Firefox MCP

The Firefox DevTools MCP enables headless browser testing of the admin UI. Use it to catch JS errors, verify rendering, and validate form flows that API-level tests can't cover.

### Prerequisites

- `make dev` running (backend on :8080, Vite HMR on :5050)
- Firefox ESR pre-installed in devcontainer (`.devcontainer/Dockerfile`)
- MCP configured in `~/.claude.json` under `mcpServers.firefox-devtools`

### Dev URLs

Always use port 5050 (Vite dev server) for browser testing — it proxies API calls to 8080.

| Service | URL |
| ------- | --- |
| Admin UI | `http://localhost:5050/admin/` |
| Backend API | `http://localhost:8080` |
| MailHog | `http://localhost:8025` |
| MinIO Console | `http://localhost:9001` |

### Admin Login

After a fresh `make db-reset-full`, the admin UI shows a setup page at `/admin/setup`. The setup token is in `.env` (`FLUXBASE_SECURITY_SETUP_TOKEN`). Once an admin user exists, login is at `/admin/login` via `POST /dashboard/auth/login` with `{email, password}`.

### Common Testing Patterns

**Login and navigate:**

1. `navigate_page` to `http://localhost:5050/admin/login`
2. `take_snapshot` to get DOM with UIDs
3. `fill_by_uid` for email/password fields
4. `click_by_uid` the submit button
5. `screenshot_page` to verify dashboard loaded

**Check for JS errors:**

1. `clear_console_messages`
2. Navigate to target page
3. `list_console_messages` with `level: "error"` — any errors indicate a bug

**Check for failed API calls:**

1. `list_network_requests` with `statusMin: 400`
2. `get_network_request` by ID to inspect details

**Visual verification:**

- `screenshot_page` for full page capture
- `screenshot_by_uid` for specific components
- Use `saveTo` parameter to persist to disk

### MCP Tool Quick Reference

| Tool | Purpose |
| ---- | ------- |
| `navigate_page` | Go to a URL |
| `take_snapshot` | Get DOM tree with stable UIDs for interaction |
| `click_by_uid` | Click an element |
| `fill_by_uid` | Type into an input |
| `fill_form_by_uid` | Fill multiple fields at once |
| `screenshot_page` | Full page screenshot |
| `list_console_messages` | Check JS console (filter by level) |
| `list_network_requests` | See API calls (filter by status) |
| `get_network_request` | Inspect request/response details |
| `evaluate_script` | Run JS in page context |

### When to Use Browser vs API Testing

**Use browser testing (Firefox MCP) when:**

- Changes touch `admin/src/` React components or pages
- Verifying new UI features render correctly
- Checking for JS console errors after frontend changes
- Validating form submission flows through the actual UI

**Use API testing (Go E2E, curl) when:**

- Testing `internal/` Go backend code
- Verifying business logic, auth flows, database operations
- Running the existing test suite (`make test`, `make test-e2e`)

**Use both when:**

- A feature spans frontend and backend (e.g., a settings page that writes to the database)
