# Future Plans

This document outlines planned features and enhancements for Fluxbase.

---

## MCP Server (Model Context Protocol)

**Status:** Planned
**Priority:** Medium
**Estimated Effort:** 2-4 weeks

### Overview

Implement an embedded MCP server in Fluxbase that enables AI assistants to interact with user data through a standardized protocol. MCP is becoming the standard way for AI tools (Claude, GPT-based assistants, etc.) to interact with external services.

### Benefits

1. **Standardized AI Integration**: Any MCP-compatible AI client can interact with Fluxbase
2. **Security**: Inherits Fluxbase auth (JWT, API keys, RLS)
3. **Unified Interface**: Single protocol for database, functions, storage, and more
4. **Tool Discovery**: AI agents can discover available capabilities dynamically
5. **Competitive Advantage**: Positions Fluxbase as an "AI-native" BaaS

### Design Decisions

| Decision   | Choice                      | Rationale                           |
| ---------- | --------------------------- | ----------------------------------- |
| Use case   | AI assistants for end-users | Primary value proposition           |
| Deployment | Embedded in Fluxbase        | Simpler deployment, single binary   |
| Language   | Go                          | Matches backend, easier integration |
| Transport  | HTTP + JSON-RPC 2.0         | Standard MCP transport              |

### API Endpoints

```
POST /mcp          # JSON-RPC 2.0 requests
GET  /mcp/sse      # Server-sent events (notifications)
GET  /mcp/health   # Health check (no auth)
```

### Tools to Implement

| Tool              | Description                                       | Scope Required      |
| ----------------- | ------------------------------------------------- | ------------------- |
| `query_table`     | Query database with filters, ordering, pagination | `read:tables`       |
| `insert_record`   | Insert new records                                | `write:tables`      |
| `update_record`   | Update existing records                           | `write:tables`      |
| `delete_record`   | Delete records                                    | `write:tables`      |
| `invoke_function` | Call edge functions                               | `execute:functions` |
| `invoke_rpc`      | Call RPC procedures                               | `execute:rpc`       |
| `search_vectors`  | Semantic similarity search                        | `read:vectors`      |
| `upload_file`     | Upload to storage                                 | `write:storage`     |
| `download_file`   | Download from storage                             | `read:storage`      |
| `submit_job`      | Submit background job                             | `execute:jobs`      |

### Resources to Expose

| Resource URI                 | Description                              |
| ---------------------------- | ---------------------------------------- |
| `fluxbase://schema/tables`   | Database schema (tables, columns, types) |
| `fluxbase://functions`       | Available edge functions                 |
| `fluxbase://rpc`             | Available RPC procedures                 |
| `fluxbase://storage/buckets` | Storage buckets                          |

### Authentication

The MCP server reuses Fluxbase's existing authentication:

1. **API Key** (recommended for AI assistants)

   ```http
   POST /mcp
   X-API-Key: fb_pk_xxxxxxxxxxxx
   ```

2. **JWT Bearer Token** (for user sessions)

   ```http
   POST /mcp
   Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
   ```

3. **Service Key** (admin operations)
   ```http
   POST /mcp
   X-Service-Key: sb_service_xxxxxxxxxxxx
   ```

**Scope-based access**: Each tool requires specific API key scopes. Tools are rejected if the API key lacks required scopes.

**RLS enforcement**: Database operations respect PostgreSQL Row Level Security. When an API key is bound to a user, all queries execute with that user's RLS context.

### Configuration

```yaml
mcp:
  enabled: true
  base_path: "/mcp"
  session_timeout: "30m"
  max_message_size: 10485760 # 10MB
  allowed_tools: [] # Empty = all enabled
  allowed_resources: [] # Empty = all enabled
  rate_limit_per_min: 100
```

### Directory Structure

```
internal/mcp/
  handler.go           # Fiber HTTP routes
  server.go            # MCP server orchestration
  transport.go         # JSON-RPC 2.0 handling
  session.go           # Session management
  types.go             # MCP/JSON-RPC type definitions
  auth.go              # Auth context extraction
  config.go            # MCP configuration

  tools/
    registry.go        # Tool registration
    query_table.go     # Database query tool
    crud.go            # insert/update/delete tools
    invoke_function.go # Edge function invocation
    invoke_rpc.go      # RPC procedure invocation
    search_vectors.go  # Vector similarity search
    storage.go         # File upload/download
    submit_job.go      # Background job submission

  resources/
    registry.go        # Resource registration
    schema.go          # Database schema resource
    functions.go       # Edge functions listing
    rpc.go             # RPC procedures listing
    buckets.go         # Storage buckets listing
```

### Implementation Steps

#### Phase 1: Core Infrastructure

- [ ] Create `internal/mcp/` directory
- [ ] Add `MCPConfig` to `internal/config/config.go`
- [ ] Implement JSON-RPC types in `types.go`
- [ ] Implement Fiber handler in `handler.go`
- [ ] Implement server dispatch in `server.go`

#### Phase 2: Authentication

- [ ] Implement auth context extraction from Fiber locals
- [ ] Add scope checking for tools
- [ ] Reuse existing auth middleware

#### Phase 3: Tools

- [ ] Implement tool registry
- [ ] `query_table` - reuse `internal/api/query_builder.go`
- [ ] `insert_record`, `update_record`, `delete_record` - reuse `internal/api/rest_crud.go`
- [ ] `invoke_function` - use `internal/runtime/runtime.go`
- [ ] `invoke_rpc` - use `internal/rpc/executor.go`
- [ ] `search_vectors` - use `internal/ai/rag_service.go`
- [ ] `upload_file`, `download_file` - use `internal/storage/service.go`
- [ ] `submit_job` - use `internal/jobs/manager.go`

#### Phase 4: Resources

- [ ] Implement resource registry
- [ ] Schema resource - use `internal/database/schema_cache.go`
- [ ] Functions resource - use `internal/functions/storage.go`
- [ ] RPC resource - use `internal/rpc/storage.go`
- [ ] Buckets resource - use `internal/storage/service.go`

#### Phase 5: Integration & Testing

- [ ] Register routes in `internal/api/server.go`
- [ ] Add to server initialization
- [ ] Write unit tests
- [ ] Write integration tests

### Files to Modify

| File                        | Change                 |
| --------------------------- | ---------------------- |
| `internal/config/config.go` | Add `MCPConfig` struct |
| `internal/api/server.go`    | Register MCP handler   |

### Files to Reference

| File                                 | Purpose                   |
| ------------------------------------ | ------------------------- |
| `internal/api/query_builder.go`      | Query building logic      |
| `internal/api/rest_crud.go`          | CRUD operations           |
| `internal/middleware/apikey_auth.go` | Auth middleware pattern   |
| `internal/ai/provider.go`            | Tool/ToolFunction structs |
| `internal/rpc/types.go`              | RPC types                 |
| `internal/rpc/executor.go`           | RPC execution             |

### Example Usage

```json
// Request: Query users table
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "query_table",
    "arguments": {
      "table": "users",
      "select": "id, email, created_at",
      "filter": {"is_active": "eq.true"},
      "order": "created_at.desc",
      "limit": 10
    }
  }
}

// Response
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [{
      "type": "text",
      "text": "[{\"id\": 1, \"email\": \"user@example.com\", ...}]"
    }]
  }
}
```

### Security Considerations

1. **RLS Enforcement**: All database ops respect PostgreSQL Row Level Security
2. **Scope-Based Access**: Tools require API key scopes
3. **Rate Limiting**: Reuse existing rate limit infrastructure
4. **Input Validation**: Validate arguments against JSON Schema
5. **Audit Logging**: Log all MCP operations

### References

- [MCP Specification](https://modelcontextprotocol.io/specification/2025-11-25)
- [MCP Server Development Guide](https://github.com/cyanheads/model-context-protocol-resources)

---

## Database Branching

**Status:** Planned
**Priority:** Medium
**Estimated Effort:** 4-6 weeks

### Overview

Implement database branching to enable isolated development/testing environments, similar to Supabase's branching feature. Each branch is a separate PostgreSQL database with full data clone capability.

### Benefits

1. **Isolated Development**: Test schema changes and migrations without affecting production
2. **CI/CD Integration**: Automatic preview environments for pull requests
3. **Safe Experimentation**: Try database changes with real data copies
4. **Team Collaboration**: Multiple developers can work on different features simultaneously

### Design Decisions

| Decision      | Choice                         | Rationale                        |
| ------------- | ------------------------------ | -------------------------------- |
| Use case      | Development/Testing            | Primary value proposition        |
| Storage       | Separate PostgreSQL databases  | Full isolation, easier cleanup   |
| Data handling | Full clone option              | Debug with real data when needed |
| Scope         | CLI + Dashboard + API + GitHub | Full-featured like Supabase      |

### Database Schema

**New migration**: `internal/database/migrations/046_branching_support.up.sql`

Create `branching` schema with tables:

| Table                         | Purpose                                                                      |
| ----------------------------- | ---------------------------------------------------------------------------- |
| `branching.branches`          | Branch metadata (id, name, slug, database_name, status, type, github fields) |
| `branching.migration_history` | Track migrations per branch                                                  |
| `branching.activity_log`      | Audit trail for branch operations                                            |
| `branching.github_config`     | GitHub integration settings per repository                                   |

**Branch statuses**: `creating`, `ready`, `migrating`, `error`, `deleting`, `deleted`
**Branch types**: `main`, `preview`, `persistent`
**Data clone modes**: `schema_only`, `full_clone`, `seed_data`

### Configuration

```yaml
branching:
  enabled: true
  max_branches_per_user: 5
  max_total_branches: 20
  default_data_clone_mode: "schema_only"
  auto_delete_after: "168h" # 7 days
  database_prefix: "fluxbase_branch_"
  admin_database_url: "" # Connection with CREATE DATABASE privileges
```

### CLI Commands

| Command                                  | Description                                          |
| ---------------------------------------- | ---------------------------------------------------- |
| `fluxbase branch create <name>`          | Create new branch (`--from`, `--clone-data`, `--pr`) |
| `fluxbase branch list`                   | List all branches with status                        |
| `fluxbase branch delete <name>`          | Delete branch (`--force`)                            |
| `fluxbase branch status <name>`          | Show branch details and health                       |
| `fluxbase branch switch <name>`          | Set active branch for CLI session                    |
| `fluxbase branch reset <name>`           | Reset to parent state                                |
| `fluxbase branch migrate <name>`         | Run pending migrations                               |
| `fluxbase branch diff <source> <target>` | Show schema differences                              |

### API Endpoints

| Endpoint                             | Method | Description            |
| ------------------------------------ | ------ | ---------------------- |
| `/api/v1/admin/branches`             | POST   | Create branch          |
| `/api/v1/admin/branches`             | GET    | List branches          |
| `/api/v1/admin/branches/:id`         | GET    | Get branch details     |
| `/api/v1/admin/branches/:id`         | DELETE | Delete branch          |
| `/api/v1/admin/branches/:id/reset`   | POST   | Reset to parent        |
| `/api/v1/admin/branches/:id/migrate` | POST   | Run migrations         |
| `/api/v1/admin/branches/:id/diff`    | GET    | Get schema diff        |
| `/webhooks/github`                   | POST   | GitHub webhook handler |

### Directory Structure

```
internal/branching/
  types.go              # Branch, BranchStatus, BranchType, DataCloneMode
  storage.go            # CRUD operations for branch metadata
  manager.go            # Database operations (CREATE DATABASE, pg_dump/restore)
  router.go             # Connection pool management per branch
  clone.go              # Data cloning logic with PII handling

internal/api/
  branch_handler.go     # REST API endpoints
  github_webhook_handler.go  # GitHub PR webhooks

internal/middleware/
  branch.go             # Branch context middleware

cli/cmd/
  branch.go             # CLI commands

admin/src/
  routes/_authenticated/branches/
    index.tsx           # Branch list
    $id/index.tsx       # Branch details
    new.tsx             # Create form
    settings.tsx        # GitHub config
  features/branches/
    branch-list.tsx
    branch-card.tsx
    branch-create-dialog.tsx
    branch-delete-dialog.tsx
    branch-status-badge.tsx
    branch-switcher.tsx  # Global header selector
    hooks/
      use-branches.ts
      use-branch-switch.ts
```

### Connection Routing

Branch context extracted from:

1. `X-Fluxbase-Branch` header
2. `?branch=` query parameter
3. Subdomain (optional)

Middleware stores in fiber context: `c.Locals("branch")`, `c.Locals("branch_pool")`

### Data Cloning Strategy

Use `pg_dump`/`pg_restore` for cloning (handles active connections):

```bash
# Schema only
pg_dump --schema-only source_db | pg_restore target_db

# Full clone
pg_dump --no-owner source_db | pg_restore target_db
```

**PII Protection** - Default excluded tables for full clone:

- `auth.sessions`
- `auth.refresh_tokens`
- `auth.mfa_factors`
- `auth.identities`

### Feature Integration

| Feature          | Integration                                                  |
| ---------------- | ------------------------------------------------------------ |
| Storage          | Branch-specific paths: `branches/{slug}/path/to/file`        |
| Realtime         | Branch-specific channels: `realtime_changes_{branch_id[:8]}` |
| Functions & Jobs | Execute with branch connection pool from context             |

### Implementation Steps

#### Phase 1: Core Infrastructure

- [ ] Create `internal/branching/` directory
- [ ] Add `BranchingConfig` to `internal/config/config.go`
- [ ] Create migration `046_branching_support.up.sql`
- [ ] Implement types in `types.go`
- [ ] Implement storage in `storage.go`
- [ ] Implement manager in `manager.go`
- [ ] Implement router in `router.go`
- [ ] Add branch middleware in `internal/middleware/branch.go`

#### Phase 2: CLI Commands

- [ ] Create `cli/cmd/branch.go`
- [ ] Implement `branch create`
- [ ] Implement `branch list`
- [ ] Implement `branch delete`
- [ ] Implement `branch status`
- [ ] Implement `branch switch`
- [ ] Implement `branch reset`
- [ ] Implement `branch migrate`
- [ ] Implement `branch diff`

#### Phase 3: REST API

- [ ] Create `internal/api/branch_handler.go`
- [ ] Implement CRUD endpoints
- [ ] Implement reset/migrate/diff actions
- [ ] Register routes in `server.go`

#### Phase 4: Dashboard UI

- [ ] Create branch routes in admin dashboard
- [ ] Implement branch list page
- [ ] Implement branch details page
- [ ] Implement create/delete dialogs
- [ ] Add branch switcher to header

#### Phase 5: GitHub Integration

- [ ] Create `internal/api/github_webhook_handler.go`
- [ ] Implement PR webhook handling
- [ ] Add GitHub config management UI
- [ ] Test auto-create/delete on PR lifecycle

### Files to Modify

| File                              | Change                                |
| --------------------------------- | ------------------------------------- |
| `internal/config/config.go`       | Add `BranchingConfig` struct          |
| `internal/api/server.go`          | Register branch routes and middleware |
| `cli/cmd/root.go`                 | Add branch command                    |
| `internal/database/connection.go` | Branch-aware pool management          |
| `internal/storage/service.go`     | Branch path prefixing                 |
| `internal/realtime/hub.go`        | Branch-specific channels              |

### Security Considerations

1. **Access Control**: Branch operations require dashboard admin or service_role
2. **Creator Access**: Branch creators can access their own branches
3. **PII Protection**: Data clone excludes sensitive auth tables by default
4. **Webhook Security**: GitHub webhook signature verification
5. **Encrypted Storage**: Connection strings and secrets stored encrypted

### References

- [Supabase Branching Docs](https://supabase.com/docs/guides/deployment/branching)
- [Supabase Branching 2.0](https://joshuaberkowitz.us/blog/news-1/supabase-branching-2-0-flexible-database-experimentation-without-git-531)
