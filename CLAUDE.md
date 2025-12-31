# Fluxbase Codebase Guide

Fluxbase is a single-binary Backend-as-a-Service (BaaS) - a lightweight Supabase alternative. PostgreSQL is the only external dependency.

## Stack

- **Backend:** Go 1.25+, Fiber v2, pgx/v5, golang-migrate
- **Admin UI:** React 19, Vite, TanStack Router/Query, Tailwind v4, shadcn/ui
- **SDKs:** TypeScript (`sdk/`), React hooks (`sdk-react/`), Go (`pkg/client/`)
- **Functions Runtime:** Deno (JavaScript/TypeScript edge functions)

## Directory Structure

```
cmd/fluxbase/main.go     # Server entry point
cli/cmd/                 # CLI commands (auth, functions, jobs, migrations, secrets)
internal/                # Core backend modules (see below)
admin/src/routes/        # Admin dashboard pages (file-based routing)
sdk/src/                 # TypeScript SDK source
deploy/helm/             # Kubernetes Helm charts
test/e2e/                # End-to-end tests
```

## Internal Modules (`internal/`)

| Module | Purpose |
|--------|---------|
| `api/` | HTTP handlers (60+ files) - REST CRUD, storage, auth, DDL, webhooks |
| `auth/` | Authentication - JWT, OAuth2, OIDC, magic links, MFA, scopes |
| `branching/` | Database branching - isolated DBs for dev/test environments |
| `database/` | PostgreSQL connection, schema introspection, migrations |
| `functions/` | Edge functions - Deno runtime, bundling, loader, scheduler |
| `jobs/` | Background jobs - queue, workers, scheduler, progress tracking |
| `mcp/` | Model Context Protocol server for AI assistant integration |
| `realtime/` | WebSocket subscriptions via PostgreSQL LISTEN/NOTIFY |
| `storage/` | File storage abstraction (local filesystem or S3/MinIO) |
| `middleware/` | Auth, CORS, rate limiting, logging, branch context middlewares |
| `secrets/` | Secret management for functions/jobs |
| `config/` | YAML + env var configuration loading |
| `email/` | SMTP, SendGrid, Mailgun, AWS SES providers |
| `ai/` | Vector search (pgvector), embeddings |
| `query/` | Shared query building types (FilterCondition, etc.) |

## Database Schemas

- `auth.*` - Users, sessions, identities, client keys
- `storage.*` - Buckets, objects, access policies
- `jobs.*` - Background job storage
- `functions.*` - Edge functions registry
- `branching.*` - Database branch metadata, access control, GitHub config
- `public` - User application tables

## Key Files by Feature

**Authentication:**
- `internal/auth/service.go` - Main auth logic
- `internal/auth/jwt.go` - Token management
- `internal/auth/scopes.go` - Authorization scopes
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
- `internal/realtime/hub.go` - WebSocket connection hub
- `internal/realtime/client.go` - Client management

**MCP Server:**
- `internal/mcp/server.go` - JSON-RPC 2.0 protocol handler
- `internal/mcp/handler.go` - HTTP transport layer
- `internal/mcp/auth.go` - Auth context and scope checking
- `internal/mcp/tools/` - Tool implementations (query, storage, functions, jobs, vectors)
- `internal/mcp/resources/` - Resource providers (schema, functions, storage, rpc)

**Database Branching:**
- `internal/branching/manager.go` - CREATE/DROP DATABASE operations
- `internal/branching/storage.go` - Branch metadata CRUD
- `internal/branching/router.go` - Connection pool per branch
- `internal/api/branch_handler.go` - REST API for branch management
- `internal/api/github_webhook_handler.go` - GitHub PR automation
- `internal/middleware/branch.go` - Branch context extraction
- `cli/cmd/branch.go` - CLI commands

## Common Commands

```bash
make dev              # Start backend + admin UI dev servers
make build            # Production build with embedded admin
make test             # Run tests with race detector
make migrate-up       # Run database migrations
make cli-install      # Build and install CLI
```

## Configuration

Three-layer system: defaults → `fluxbase.yaml` → `FLUXBASE_*` env vars

Key config sections: server, database, auth, storage, realtime, functions, jobs, email, ai, mcp, branching

**MCP Configuration:**
```yaml
mcp:
  enabled: true
  base_path: /mcp
  rate_limit_per_min: 100
  allowed_tools: []      # Empty = all tools
  allowed_resources: []  # Empty = all resources
```

**Branching Configuration:**
```yaml
branching:
  enabled: true
  max_branches_per_user: 5
  max_total_branches: 50
  default_data_clone_mode: schema_only
  auto_delete_after: 24h
  database_prefix: branch_
  admin_database_url: "postgresql://..."
```

## Patterns

- Interface-based dependency injection
- Handler pattern with `*fiber.Ctx`
- Repository pattern for data access
- PostgreSQL Row Level Security (RLS) for authorization
- PostgREST-compatible REST API conventions

## Migrations

SQL files in `internal/database/migrations/` numbered sequentially (001-084+).
Format: `NNN_description.up.sql` / `NNN_description.down.sql`

## Testing

- Unit tests: `*_test.go` alongside source
- E2E tests: `test/e2e/`
- Test helpers: `internal/testutil/`
