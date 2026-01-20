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
// weather_forecast.ts
// @mcp:tool
// @mcp:name weather_forecast
// @mcp:description Get weather forecast for a location
// @mcp:scopes execute:custom

export async function handler(
  args: { location: string; days?: number },
  context: any
) {
  const { location, days = 3 } = args;

  // Access secrets securely
  const apiKey = context.secrets.get("WEATHER_API_KEY");

  const response = await fetch(
    `https://api.weather.com/forecast?location=${encodeURIComponent(location)}&days=${days}`,
    { headers: { "X-API-Key": apiKey } }
  );

  if (!response.ok) {
    return {
      content: [{ type: "text", text: `Weather API error: ${response.status}` }],
      isError: true
    };
  }

  const data = await response.json();

  return {
    content: [{ type: "text", text: JSON.stringify(data, null, 2) }]
  };
}
```

### 2. Deploy via CLI

```bash
# Create the tool
fluxbase mcp tools create weather_forecast --code ./weather_forecast.ts \
  --description "Get weather forecast for a location" \
  --allow-net

# Or sync a directory of tools
fluxbase mcp tools sync --dir ./mcp-tools
```

### 3. Use in Chatbots

Configure your chatbot to use the custom tool:

```typescript
// chatbot.ts
// @fluxbase:mcp-tools custom_weather_forecast,query_table
// @fluxbase:use-mcp-schema

export default async function handler(request: Request, context: any) {
  return context.chat({
    systemPrompt: "You are a helpful assistant that can check weather forecasts."
  });
}
```

## Tool Annotations

Use annotations in your TypeScript files to configure tool behavior:

| Annotation | Description | Example |
|------------|-------------|---------|
| `@mcp:tool` | Marks file as MCP tool | `// @mcp:tool` |
| `@mcp:name` | Tool name (alphanumeric + underscore) | `// @mcp:name weather_forecast` |
| `@mcp:description` | Human-readable description | `// @mcp:description Get weather forecast` |
| `@mcp:scopes` | Required MCP scopes (comma-separated) | `// @mcp:scopes execute:custom,read:tables` |
| `@mcp:timeout` | Execution timeout in seconds | `// @mcp:timeout 30` |
| `@mcp:memory` | Memory limit in MB | `// @mcp:memory 128` |
| `@mcp:allow-net` | Allow network access | `// @mcp:allow-net` |
| `@mcp:allow-env` | Allow environment variable access | `// @mcp:allow-env` |

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

## Tool Context

Tools receive a context object with two Fluxbase clients and utilities:

```typescript
interface ToolContext {
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

  // User-scoped Fluxbase client (respects RLS policies)
  fluxbase: FluxbaseClient;

  // Service-scoped Fluxbase client (bypasses RLS - use with caution!)
  fluxbaseService: FluxbaseClient;

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
export async function handler(args: { userId: string }, context: any) {
  // Query using user context (respects RLS - user can only see their own data)
  const { data: userData } = await context.fluxbase
    .from("profiles")
    .select("id, name, email")
    .eq("id", args.userId)
    .single()
    .execute();

  // Query using service context (bypasses RLS - admin access)
  const { data: allUsers } = await context.fluxbaseService
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
export async function handler(args: { name: string; email: string }, context: any) {
  // Insert a new record
  const { data: created } = await context.fluxbaseService
    .insert("users", { name: args.name, email: args.email })
    .select("id, name, email")
    .execute();

  // Update a record
  const { data: updated } = await context.fluxbaseService
    .update("users", { last_login: new Date().toISOString() })
    .eq("email", args.email)
    .execute();

  // Delete a record
  const { data: deleted } = await context.fluxbaseService
    .delete("users")
    .eq("id", args.userId)
    .execute();

  return created;
}
```

## Custom Resources

Resources provide read-only data to AI assistants:

```typescript
// analytics_summary.ts
// @mcp:resource
// @mcp:uri fluxbase://custom/analytics/summary
// @mcp:name Analytics Summary
// @mcp:description Real-time analytics summary

export async function handler(params: {}, context: any) {
  const data = await context.fluxbase
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

For parameterized URIs:

```typescript
// user_profile.ts
// @mcp:resource
// @mcp:uri fluxbase://custom/users/{id}/profile
// @mcp:name User Profile
// @mcp:template

export async function handler(params: { id: string }, context: any) {
  const user = await context.fluxbase
    .from("users")
    .select("*")
    .eq("id", params.id)
    .execute();

  return [{ type: "text", text: JSON.stringify(user[0]) }];
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
// @fluxbase:mcp-tools custom_weather_forecast,custom_send_notification,query_table
```

The `custom_` prefix is automatically added to tool names to distinguish them from built-in tools.

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
// order_status.ts
// @mcp:tool
// @mcp:name check_order_status
// @mcp:description Check the status of a customer order
// @mcp:scopes execute:custom,read:tables
// @mcp:timeout 10

export async function handler(
  args: { order_id: string },
  context: any
) {
  const { order_id } = args;

  // Validate input
  if (!order_id || !order_id.match(/^ORD-\d+$/)) {
    return {
      content: [{ type: "text", text: "Invalid order ID format. Expected: ORD-XXXXX" }],
      isError: true
    };
  }

  // Query Fluxbase
  const orders = await context.fluxbase
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
fluxbase mcp tools create check_order_status --code ./order_status.ts
fluxbase mcp tools test check_order_status --args '{"order_id": "ORD-12345"}'
```
