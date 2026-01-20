---
title: Custom MCP Tools
description: Create and deploy custom MCP tools and resources to extend AI capabilities in Fluxbase.
---

Custom MCP (Model Context Protocol) tools allow you to extend Fluxbase's AI capabilities with domain-specific functionality. Write tools in TypeScript, deploy them to your Fluxbase instance, and they become available to all AI chatbots configured to use them.

## Overview

MCP tools are functions that AI assistants can invoke during conversations. Custom tools let you:

- Integrate external APIs (weather, payments, notifications)
- Implement business logic accessible to AI
- Create domain-specific data transformations
- Build custom validation and processing pipelines

Custom resources provide read-only data that AI can access during conversations, such as configuration, analytics, or dynamic content.

## Quick Start

### 1. Create a Custom Tool

Create a TypeScript file with your tool implementation:

```typescript
// get_user_orders.ts
// @fluxbase:description Get orders for a user

export async function handler(
  args: { user_id: string; limit?: number },
  fluxbase: any,           // User-scoped client (respects RLS)
  fluxbaseService: any,    // Service-scoped client (bypasses RLS)
  utils: any               // Tool metadata and helpers
) {
  const { user_id, limit = 10 } = args;

  // Same signature as edge functions: handler(args, fluxbase, fluxbaseService, utils)
  const { data: orders } = await fluxbase
    .from("orders")
    .select("id, status, total, created_at")
    .eq("user_id", user_id)
    .order("created_at", { ascending: false })
    .limit(limit)
    .execute();

  return {
    content: [{ type: "text", text: JSON.stringify(orders, null, 2) }]
  };
}
```

### 2. Deploy via CLI

```bash
# Create the tool
fluxbase mcp tools create get_user_orders --code ./get_user_orders.ts

# Or sync a directory of tools (all .ts files become tools)
fluxbase mcp tools sync --dir ./mcp-tools
```

### 3. Use in Chatbots

Configure your chatbot to use the custom tool (note: custom tools are prefixed with `custom_`):

```typescript
// order_assistant.ts
/**
 * @fluxbase:mcp-tools custom_get_user_orders,query_table
 */

export default `You are an order management assistant.
You can look up user orders and query tables.`;
```

## Tool Annotations

All annotations are optional. The `@fluxbase:` prefix is consistent with edge functions and jobs.

| Annotation | Description | Default |
|------------|-------------|---------|
| `@fluxbase:name` | Tool name | Filename (e.g., `weather_forecast.ts` → `weather_forecast`) |
| `@fluxbase:description` | Human-readable description for AI | None |
| `@fluxbase:scopes` | Additional MCP scopes | `execute:custom` |
| `@fluxbase:timeout` | Execution timeout in seconds | 30 |
| `@fluxbase:memory` | Memory limit in MB | 128 |
| `@fluxbase:allow-net` | Allow network access | false |
| `@fluxbase:allow-env` | Allow secrets/environment access | false |

## Input Schema

Define input validation using JSON Schema:

```typescript
// Via API or dashboard, set input_schema:
{
  "type": "object",
  "properties": {
    "location": {
      "type": "string",
      "description": "City name or coordinates"
    },
    "days": {
      "type": "number",
      "default": 3,
      "minimum": 1,
      "maximum": 14
    }
  },
  "required": ["location"]
}
```

## Handler Signature

MCP tools use the same handler signature as edge functions and jobs:

```typescript
handler(args, fluxbase, fluxbaseService, utils)
```

| Parameter | Description |
|-----------|-------------|
| `args` | Input arguments passed to the tool |
| `fluxbase` | User-scoped Fluxbase client (respects RLS) |
| `fluxbaseService` | Service-scoped Fluxbase client (bypasses RLS) |
| `utils` | Tool metadata and helpers |

### Utils Object

```typescript
interface ToolUtils {
  // Tool metadata
  tool_name: string;
  namespace: string;

  // User information
  user_id: string;
  user_email: string;
  user_role: string;
  scopes: string[];

  // Secrets accessor (requires allow_env permission)
  secrets: {
    get(name: string): string | undefined;
  };

  // Environment access (requires allow_env permission)
  env: {
    get(name: string): string | undefined;
  };
}

interface FluxbaseClient {
  // Query builder
  from(table: string): QueryBuilder;
  insert(table: string, data: any): InsertBuilder;
  update(table: string, data: any): UpdateBuilder;
  delete(table: string): DeleteBuilder;

  // RPC calls
  rpc(functionName: string, params?: object): Promise<{ data: any; error: null }>;

  // Storage operations
  storage: {
    list(bucket: string, options?: { prefix?: string; limit?: number }): Promise<any>;
    download(bucket: string, path: string): Promise<Response>;
    upload(bucket: string, path: string, file: any, options?: { contentType?: string }): Promise<any>;
    remove(bucket: string, paths: string | string[]): Promise<any>;
    getPublicUrl(bucket: string, path: string): string;
  };

  // Edge functions
  functions: {
    invoke(name: string, options?: { body?: any; headers?: object }): Promise<{ data: any; error: null }>;
  };
}
```

### Accessing Fluxbase Data

```typescript
export async function handler(args: { userId: string }, fluxbase, fluxbaseService, utils) {
  // Query using user context (respects RLS - user can only see their own data)
  const { data: userData } = await fluxbase
    .from("profiles")
    .select("id, name, email")
    .eq("id", args.userId)
    .single()
    .execute();

  // Query using service context (bypasses RLS - admin access)
  const { data: allUsers } = await fluxbaseService
    .from("profiles")
    .select("id, name")
    .limit(10)
    .execute();

  return {
    content: [{ type: "text", text: JSON.stringify({ user: userData, recentUsers: allUsers }) }]
  };
}
```

### Insert, Update, Delete

```typescript
export async function handler(args: { name: string; email: string }, fluxbase, fluxbaseService, utils) {
  // Insert a new record
  const { data: created } = await fluxbaseService
    .insert("users", { name: args.name, email: args.email })
    .select("id, name, email")
    .execute();

  // Update a record
  const { data: updated } = await fluxbaseService
    .update("users", { last_login: new Date().toISOString() })
    .eq("email", args.email)
    .execute();

  // Delete a record
  const { data: deleted } = await fluxbaseService
    .delete("users")
    .eq("id", args.userId)
    .execute();

  return created;
}
```

## Custom Resources

Resources provide read-only data to AI assistants. The URI defaults to `fluxbase://custom/{name}` based on filename.

Resources use the same handler signature:

```typescript
// analytics_summary.ts
// @fluxbase:description Real-time analytics summary

export async function handler(params: {}, fluxbase, fluxbaseService, utils) {
  const { data } = await fluxbase
    .from("analytics_events")
    .select("*")
    .execute();

  return [
    {
      type: "text",
      text: JSON.stringify({
        total_events: data.length,
        last_updated: new Date().toISOString()
      })
    }
  ];
}
```

### Template Resources

For parameterized URIs, specify a custom URI with `{param}` placeholders. Templates are auto-detected:

```typescript
// user_profile.ts
// @fluxbase:uri fluxbase://custom/users/{id}/profile

export async function handler(params: { id: string }, fluxbase, fluxbaseService, utils) {
  const { data: user } = await fluxbase
    .from("users")
    .select("*")
    .eq("id", params.id)
    .single()
    .execute();

  return [{ type: "text", text: JSON.stringify(user) }];
}
```

## API Reference

### List Tools

```bash
GET /api/v1/mcp/tools
```

### Create Tool

```bash
POST /api/v1/mcp/tools
Content-Type: application/json

{
  "name": "weather_forecast",
  "namespace": "default",
  "description": "Get weather forecast",
  "code": "export async function handler...",
  "input_schema": { "type": "object", ... },
  "required_scopes": ["execute:custom"],
  "timeout_seconds": 30,
  "memory_limit_mb": 128,
  "allow_net": true,
  "allow_env": false
}
```

### Sync Tool (Upsert)

```bash
POST /api/v1/mcp/tools/sync
Content-Type: application/json

{
  "name": "weather_forecast",
  "namespace": "default",
  "code": "...",
  "upsert": true
}
```

### Test Tool

```bash
POST /api/v1/mcp/tools/:id/test
Content-Type: application/json

{
  "args": { "location": "New York" }
}
```

## CLI Commands

```bash
# List tools
fluxbase mcp tools list

# Get tool details
fluxbase mcp tools get weather_forecast

# Create tool
fluxbase mcp tools create weather_forecast --code ./weather.ts

# Update tool
fluxbase mcp tools update weather_forecast --code ./weather.ts

# Delete tool
fluxbase mcp tools delete weather_forecast

# Sync directory
fluxbase mcp tools sync --dir ./mcp-tools --namespace production

# Test tool
fluxbase mcp tools test weather_forecast --args '{"location": "NYC"}'

# Resources
fluxbase mcp resources list
fluxbase mcp resources create analytics --uri "fluxbase://custom/analytics" --code ./analytics.ts
fluxbase mcp resources sync --dir ./mcp-resources
```

## Security

### Deno Sandbox

Custom tools run in a sandboxed Deno environment with explicit permissions:

- **Network (`allow_net`)**: Required for external API calls
- **Environment (`allow_env`)**: Access to environment variables and secrets
- **Read (`allow_read`)**: File system read access
- **Write (`allow_write`)**: File system write access

### Scopes

Tools require the `execute:custom` scope plus any additional scopes you specify. Users must have these scopes to invoke the tool.

### Secrets

Access secrets securely via `context.secrets.get("SECRET_NAME")`. Secrets are:
- Encrypted at rest
- Never exposed in logs
- Only available when `allow_env` is enabled

## Integration with Chatbots

Custom tools are automatically available to chatbots. Configure which tools a chatbot can use:

```typescript
// my_chatbot.ts
/**
 * @fluxbase:mcp-tools custom_check_order_status,query_table
 */
export default `You are a customer service assistant.`;
```

The `custom_` prefix is automatically added to tool names to distinguish them from built-in tools. So `check_order_status.ts` becomes `custom_check_order_status` in chatbot configuration.

## Best Practices

1. **Validate inputs** - Use JSON Schema for input validation
2. **Handle errors gracefully** - Return `isError: true` with helpful messages
3. **Keep tools focused** - One tool, one responsibility
4. **Use timeouts** - Set appropriate timeouts for external API calls
5. **Secure secrets** - Never hardcode API keys; use the secrets system
6. **Test thoroughly** - Use the test endpoint before deploying to production
7. **Document tools** - Clear descriptions help AI use tools correctly

## Example: Complete Integration

```typescript
// check_order_status.ts
// @fluxbase:description Check the status of a customer order

export async function handler(args: { order_id: string }, fluxbase, fluxbaseService, utils) {
  const { order_id } = args;

  // Validate input
  if (!order_id || !order_id.match(/^ORD-\d+$/)) {
    return {
      content: [{ type: "text", text: "Invalid order ID format. Expected: ORD-XXXXX" }],
      isError: true
    };
  }

  // Query Fluxbase
  const { data: orders } = await fluxbase
    .from("orders")
    .select("id, status, created_at, total, items")
    .eq("id", order_id)
    .execute();

  if (!orders || orders.length === 0) {
    return {
      content: [{ type: "text", text: `Order ${order_id} not found` }],
      isError: true
    };
  }

  const order = orders[0];

  return {
    content: [{
      type: "text",
      text: `Order ${order.id}:
- Status: ${order.status}
- Created: ${order.created_at}
- Total: $${order.total}
- Items: ${order.items.length} item(s)`
    }]
  };
}
```

Deploy and test:

```bash
fluxbase mcp tools create check_order_status --code ./check_order_status.ts
fluxbase mcp tools test check_order_status --args '{"order_id": "ORD-12345"}'
```
