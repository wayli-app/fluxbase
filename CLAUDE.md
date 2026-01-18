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

### Test Organization
- Unit tests: `*_test.go` alongside source
- E2E tests: `test/e2e/`
- Test helpers: `internal/testutil/`

### Coverage Targets
- **Overall:** 20%+ (after exclusions)
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
- Implementation notes: `IMPLEMENTATION_ANALYSIS.md`

**Documentation checklist:**
- [ ] Does the feature documentation match the implementation?
- [ ] Are all configuration options documented?
- [ ] Are error messages and edge cases explained?
- [ ] Are code examples correct and runnable?

### Pre-Commit Checklist

Before committing changes, verify:
1. `make test` passes
2. `make lint` passes (if available)
3. Documentation is updated for user-facing changes
4. New tests are added for new functionality
5. Existing tests are updated if behavior changed
