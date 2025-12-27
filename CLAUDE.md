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
| `database/` | PostgreSQL connection, schema introspection, migrations |
| `functions/` | Edge functions - Deno runtime, bundling, loader, scheduler |
| `jobs/` | Background jobs - queue, workers, scheduler, progress tracking |
| `realtime/` | WebSocket subscriptions via PostgreSQL LISTEN/NOTIFY |
| `storage/` | File storage abstraction (local filesystem or S3/MinIO) |
| `middleware/` | Auth, CORS, rate limiting, logging middlewares |
| `secrets/` | Secret management for functions/jobs |
| `config/` | YAML + env var configuration loading |
| `email/` | SMTP, SendGrid, Mailgun, AWS SES providers |
| `ai/` | Vector search (pgvector), embeddings |

## Database Schemas

- `auth.*` - Users, sessions, identities, API keys
- `storage.*` - Buckets, objects, access policies
- `jobs.*` - Background job storage
- `functions.*` - Edge functions registry
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

Key config sections: server, database, auth, storage, realtime, functions, jobs, email, ai

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
