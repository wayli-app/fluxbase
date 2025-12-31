---
title: "Overview"
description: "Model Context Protocol server for AI assistant integration"
---

Fluxbase includes a built-in MCP (Model Context Protocol) server that enables AI assistants like Claude to interact with your database, storage, functions, and jobs.

## Overview

The MCP server exposes Fluxbase functionality through a standardized JSON-RPC 2.0 protocol, allowing AI assistants to:

- Query and modify database tables (with Row Level Security)
- Upload, download, and manage storage files
- Invoke edge functions and RPC procedures
- Submit and monitor background jobs
- Search vector embeddings for RAG applications

## Configuration

Enable the MCP server in your `fluxbase.yaml`:

```yaml
mcp:
  enabled: true
  base_path: /mcp
  session_timeout: 3600s
  max_message_size: 10485760  # 10MB
  rate_limit_per_min: 100
  allowed_tools: []      # Empty = all tools enabled
  allowed_resources: []  # Empty = all resources enabled
```

### Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| `enabled` | `false` | Enable the MCP server endpoint |
| `base_path` | `/mcp` | URL path for MCP endpoints |
| `session_timeout` | `3600s` | Session timeout for connections |
| `max_message_size` | `10485760` | Maximum request size in bytes |
| `rate_limit_per_min` | `100` | Requests per minute per client |
| `allowed_tools` | `[]` | Whitelist of allowed tools |
| `allowed_resources` | `[]` | Whitelist of allowed resources |

## Authentication

The MCP server requires authentication for all requests. Supported methods:

### Client Keys

Create a client key with specific MCP scopes:

```bash
fluxbase clientkeys create --name "AI Assistant" \
  --scopes "read:tables,write:tables,execute:functions"
```

Use the key in requests:

```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "X-Client-Key: your-client-key" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'
```

### Service Keys

Service keys have full access and bypass Row Level Security:

```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "X-Service-Key: your-service-key" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'
```

## Available Scopes

| Scope | Description |
|-------|-------------|
| `read:tables` | Query database tables |
| `write:tables` | Insert, update, delete records |
| `execute:functions` | Invoke edge functions |
| `execute:rpc` | Execute RPC procedures |
| `read:storage` | List and download files |
| `write:storage` | Upload and delete files |
| `execute:jobs` | Submit and monitor jobs |
| `read:vectors` | Vector similarity search |
| `read:schema` | Access database schema |

## Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/mcp/health` | GET | Health check (no auth required) |
| `/mcp` | POST | JSON-RPC requests |

## Protocol

The MCP server implements JSON-RPC 2.0 with the following methods:

- `initialize` - Protocol handshake
- `ping` - Health check
- `tools/list` - List available tools
- `tools/call` - Execute a tool
- `resources/list` - List available resources
- `resources/read` - Read a resource

### Example Request

```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "query_table",
    "arguments": {
      "table": "users",
      "select": "id,email,created_at",
      "filter": {"is_active": "eq.true"},
      "limit": 10
    }
  },
  "id": 1
}
```

### Example Response

```json
{
  "jsonrpc": "2.0",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "[{\"id\":\"...\",\"email\":\"user@example.com\",\"created_at\":\"...\"}]"
      }
    ],
    "isError": false
  },
  "id": 1
}
```

## Security

- All database operations respect Row Level Security (RLS) policies
- Service keys bypass RLS for administrative operations
- Internal schemas (auth, storage, functions, jobs) are hidden from non-admins
- SQL injection is prevented through parameterized queries and identifier validation
- File downloads are limited to 10MB to prevent memory issues

## Next Steps

- [Available Tools](/guides/mcp/tools) - Complete tool reference
- [Available Resources](/guides/mcp/resources) - Resource reference
- [Security Best Practices](/security/mcp-security) - Security guidelines
