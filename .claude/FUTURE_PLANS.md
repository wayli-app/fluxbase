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
