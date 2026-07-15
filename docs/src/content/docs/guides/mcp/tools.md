---
title: "Tools Reference"
description: "Complete reference of the 46 built-in MCP tools across database, DDL, storage, functions, jobs, vectors, knowledge graph, branching, GitHub, sync, and utility domains"
---

The MCP server exposes **46 built-in tools** for interacting with Fluxbase. Each tool requires one or more [scopes](/guides/mcp/) granted to the calling client.

## Tool Index

| Tool | Scope | Description |
|------|-------|-------------|
| **Database** | | |
| `query_table` | `tables:read` | Query a table with filters, ordering, pagination |
| `insert_record` | `tables:write` | Insert a new record |
| `update_record` | `tables:write` | Update records matching a filter |
| `delete_record` | `tables:write` | Delete records matching a filter |
| `execute_sql` | `execute:sql` | Run a read-only SQL query |
| **DDL** | | |
| `list_schemas` | `tables:read` | List database schemas |
| `create_schema` | `admin:ddl` | Create a new schema |
| `create_table` | `admin:ddl` | Create a table with columns |
| `drop_table` | `admin:ddl` | Drop a table |
| `rename_table` | `admin:ddl` | Rename a table |
| `add_column` | `admin:ddl` | Add a column to a table |
| `drop_column` | `admin:ddl` | Drop a column from a table |
| **Storage** | | |
| `list_objects` | `storage:read` | List objects in a bucket |
| `download_object` | `storage:read` | Download a file |
| `upload_object` | `storage:write` | Upload a file |
| `delete_object` | `storage:write` | Delete a file |
| **Functions & RPC** | | |
| `invoke_function` | `functions:execute` | Invoke an edge function |
| `invoke_rpc` | `rpc:execute` | Invoke an RPC procedure |
| **Jobs** | | |
| `submit_job` | `execute:jobs` | Submit a background job |
| `get_job_status` | `execute:jobs` | Get job status |
| **Vectors** | | |
| `search_vectors` | `read:vectors` | Semantic similarity search |
| **Knowledge Graph** | | |
| `query_knowledge_graph` | `read:vectors` | Query entities with filters |
| `browse_knowledge_graph` | `read:vectors` | Browse the full knowledge graph |
| `find_related_entities` | `read:vectors` | Find entities related to a given entity |
| **Branching — Access** | | |
| `list_branches` | `branch:read` | List branches |
| `get_branch` | `branch:read` | Get branch details |
| `get_active_branch` | `branch:read` | Get the active branch |
| `grant_branch_access` | `branch:access` | Grant a user branch access |
| `revoke_branch_access` | `branch:access` | Revoke branch access |
| **Branching — Lifecycle** | | |
| `create_branch` | `branch:write` | Create a new branch |
| `delete_branch` | `branch:write` | Delete a branch |
| `reset_branch` | `branch:write` | Reset a branch to its template |
| `set_active_branch` | `branch:write` | Set the active branch |
| **GitHub** | | |
| `list_github_issues` | `github:read` | List issues in a repository |
| `get_github_issue` | `github:read` | Get issue details |
| `create_github_issue` | `github:write` | Create an issue |
| `create_github_issue_comment` | `github:write` | Comment on an issue |
| `update_github_issue_labels` | `github:write` | Add/remove issue labels |
| `trigger_claude_fix` | `github:write` | Trigger Claude Fix automation |
| **Sync** | | |
| `sync_chatbot` | `sync:chatbots` | Create/update a chatbot from code |
| `sync_function` | `sync:functions` | Deploy/update an edge function |
| `sync_job` | `sync:jobs` | Create/update a background job |
| `sync_rpc` | `sync:rpc` | Create/update an RPC procedure |
| `sync_migration` | `sync:migrations` | Create/apply a migration |
| **Utility** | | |
| `think` | _(none)_ | Plan an investigation before acting |
| `http_request` | `execute:http` | HTTP GET to a whitelisted domain |

:::note[Tool availability]
Custom tools (see [Custom MCP Tools](/guides/mcp/custom-mcp-tools/)) are registered in the same registry and appear alongside these built-in tools. Tools can be restricted globally via `mcp.allowed_tools`.
:::

## Database Tools

### query_table

Query a database table with filters, ordering, and pagination. **Scope:** `tables:read`

```json
{
  "name": "query_table",
  "arguments": {
    "table": "public.users",
    "select": "id,email,created_at",
    "filter": {
      "is_active": "eq.true",
      "age": "gt.18"
    },
    "order": "created_at.desc",
    "limit": 100,
    "offset": 0
  }
}
```

**Filter operators:** `eq`, `neq`, `gt`, `gte`, `lt`, `lte`, `like`, `ilike`, `is`, `in`.

### insert_record

Insert a new record. **Scope:** `tables:write`

```json
{
  "name": "insert_record",
  "arguments": {
    "table": "public.products",
    "data": { "name": "Widget", "price": 29.99, "category": "electronics" },
    "returning": "id,created_at"
  }
}
```

### update_record

Update records matching a filter. **Scope:** `tables:write`

```json
{
  "name": "update_record",
  "arguments": {
    "table": "public.products",
    "data": { "price": 24.99 },
    "filter": { "id": "eq.123" },
    "returning": "id,price,updated_at"
  }
}
```

:::warning
The `filter` parameter is required to prevent accidental bulk updates.
:::

### delete_record

Delete records matching a filter. **Scope:** `tables:write`

```json
{
  "name": "delete_record",
  "arguments": {
    "table": "public.products",
    "filter": { "id": "eq.123" },
    "returning": "id"
  }
}
```

:::warning
The `filter` parameter is required to prevent accidental bulk deletes.
:::

### execute_sql

Run a read-only SQL query against the database. Only `SELECT` queries are allowed by default. **Scope:** `execute:sql`

```json
{
  "name": "execute_sql",
  "arguments": {
    "sql": "SELECT count(*) FROM public.users WHERE created_at > now() - interval '7 days'",
    "description": "Count new users in the last 7 days"
  }
}
```

`sql` and `description` are both required (the description explains the query's intent).

## DDL Tools

All DDL tools (except `list_schemas`) require the `admin:ddl` scope.

### create_table

Create a new table with the given columns. **Scope:** `admin:ddl`

```json
{
  "name": "create_table",
  "arguments": {
    "name": "public.tasks",
    "columns": [
      { "name": "id", "type": "uuid", "primary_key": true, "default": "gen_random_uuid()" },
      { "name": "title", "type": "text", "nullable": false },
      { "name": "done", "type": "boolean", "default": "false" }
    ]
  }
}
```

`drop_table`, `rename_table`, `add_column` (requires `table`, `name`, `type`), and `drop_column` follow the same shape. `list_schemas` takes no arguments (`tables:read`).

## Storage Tools

### list_objects

List objects in a bucket. **Scope:** `storage:read`

```json
{
  "name": "list_objects",
  "arguments": { "bucket": "uploads", "prefix": "images/", "limit": 100, "start_after": "images/file099.jpg" }
}
```

### download_object

Download a file. **Scope:** `storage:read`

```json
{ "name": "download_object", "arguments": { "bucket": "uploads", "key": "documents/report.pdf" } }
```

Text files are returned as-is; binary files are returned base64-encoded (max 10MB).

### upload_object

Upload a file. **Scope:** `storage:write`

```json
{
  "name": "upload_object",
  "arguments": {
    "bucket": "uploads",
    "key": "documents/report.txt",
    "content": "Report content here...",
    "content_type": "text/plain",
    "encoding": "text"
  }
}
```

`encoding` is `text` (plain) or `base64` (binary).

### delete_object

Delete a file. **Scope:** `storage:write`

```json
{ "name": "delete_object", "arguments": { "bucket": "uploads", "key": "documents/old-report.pdf" } }
```

## Function & RPC Tools

### invoke_function

Invoke an edge function. **Scope:** `functions:execute`

```json
{
  "name": "invoke_function",
  "arguments": {
    "name": "send-email",
    "namespace": "default",
    "method": "POST",
    "body": { "to": "user@example.com", "subject": "Hello" },
    "headers": { "X-Custom-Header": "value" }
  }
}
```

**Response:** `{ "status", "headers", "body", "execution_id" }`.

### invoke_rpc

Invoke an RPC procedure. **Scope:** `rpc:execute`

```json
{
  "name": "invoke_rpc",
  "arguments": { "name": "calculate_total", "namespace": "default", "params": { "order_id": "ord-123" } }
}
```

**Response:** `{ "data", "rows_returned", "duration_ms" }`.

## Job Tools

### submit_job

Submit a background job. **Scope:** `execute:jobs`

```json
{
  "name": "submit_job",
  "arguments": {
    "job_name": "process-report",
    "namespace": "default",
    "payload": { "report_id": "rpt-123", "format": "pdf" },
    "priority": 5,
    "scheduled_at": "2024-01-15T10:00:00Z"
  }
}
```

### get_job_status

Get the status of a background job. **Scope:** `execute:jobs`

```json
{ "name": "get_job_status", "arguments": { "job_id": "job-456" } }
```

## Vector Search Tools

### search_vectors

Perform semantic similarity search. **Scope:** `read:vectors`

```json
{
  "name": "search_vectors",
  "arguments": {
    "query": "How do I reset my password?",
    "chatbot_id": "cb-123",
    "knowledge_bases": ["kb-docs", "kb-faq"],
    "limit": 5,
    "threshold": 0.7,
    "tags": ["support"]
  }
}
```

## Knowledge Graph Tools

All knowledge-graph tools require the `read:vectors` scope and take a `knowledge_base_id`.

### query_knowledge_graph

Query entities with optional filtering by entity type and a search query.

```json
{
  "name": "query_knowledge_graph",
  "arguments": { "knowledge_base_id": "kb-123", "type": "person", "search": "Ada" }
}
```

`browse_knowledge_graph` returns the full graph for a knowledge base; `find_related_entities` returns entities related to a given entity (requires `knowledge_base_id` and `entity_id`).

## Branching Tools

### create_branch

Create an isolated database branch. **Scope:** `branch:write`

```json
{
  "name": "create_branch",
  "arguments": {
    "name": "feature-x",
    "parent_branch_id": "main",
    "data_clone_mode": "schema_only"
  }
}
```

`delete_branch` and `reset_branch` take `branch_id` or `slug` (`branch:write`). `list_branches`, `get_branch`, and `get_active_branch` are read-only (`branch:read`). `set_active_branch` (`branch:write`) takes a branch id/slug. `grant_branch_access` and `revoke_branch_access` (`branch:access`) manage per-user branch grants.

## GitHub Tools

Require a configured GitHub token. Listing and reading use `github:read`; creating, commenting, relabeling, and triggering use `github:write`.

### list_github_issues

```json
{
  "name": "list_github_issues",
  "arguments": { "repository": "owner/repo", "state": "open", "labels": "bug,priority:high", "limit": 30 }
}
```

Other GitHub tools: `get_github_issue`, `create_github_issue`, `create_github_issue_comment`, `update_github_issue_labels`, and `trigger_claude_fix` (adds the `claude-fix` label to kick off automation).

## Sync Tools

Deploy or update definitions from code. Each parses `@fluxbase:` annotations from the supplied code.

| Tool | Scope | Required args |
|------|-------|---------------|
| `sync_function` | `sync:functions` | `name`, `code` |
| `sync_job` | `sync:jobs` | `name`, `code` (with `@fluxbase:schedule`) |
| `sync_chatbot` | `sync:chatbots` | `name`, `code` |
| `sync_rpc` | `sync:rpc` | `name`, `sql_code` |
| `sync_migration` | `sync:migrations` | `name`, `up_sql` |

```json
{
  "name": "sync_function",
  "arguments": { "name": "send-email", "code": "/** @fluxbase:public */ export default async () => { ... }" }
}
```

`sync_migration` defaults to `auto_apply: false` and `dry_run: true` for safety.

## Utility Tools

### think

Plan your investigation approach before executing queries — analyze the question, break it into steps, and choose tools. **No scope required** (always available).

```json
{
  "name": "think",
  "arguments": { "analysis": "...", "plan": "...", "tool_choice": "..." }
}
```

`analysis`, `plan`, and `tool_choice` are all required.

### http_request

Make an HTTP GET request to an external API. Only domains whitelisted for the current context are permitted; JSON responses only. **Scope:** `execute:http`

```json
{
  "name": "http_request",
  "arguments": { "url": "https://api.example.com/data", "headers": { "Accept": "application/json" } }
}
```

`url` is required.

## Error Handling

All tools return errors in a consistent format:

```json
{
  "content": [{ "type": "text", "text": "Error message here" }],
  "isError": true
}
```

Common error codes:

- `-32002` — Resource not found
- `-32003` — Tool not found
- `-32004` — Tool execution error
- `-32005` — Unauthorized
- `-32006` — Forbidden
- `-32007` — Rate limited
