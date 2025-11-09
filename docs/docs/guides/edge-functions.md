# Edge Functions

Edge Functions are serverless functions powered by the Deno runtime that execute JavaScript/TypeScript code in response to HTTP requests. They enable you to run custom backend logic without managing infrastructure.

## Overview

Edge Functions in Fluxbase provide:

- **Deno Runtime** - Execute TypeScript/JavaScript code with modern ES modules
- **HTTP Triggered** - Invoke functions via REST API or SDK
- **Secure Sandbox** - Configurable permissions for network, environment, and filesystem access
- **Database Access** - Query your Fluxbase database directly
- **Execution Logging** - Track function invocations and debug issues
- **Version Control** - Each function update increments version
- **Timeout Protection** - Configurable execution limits

## Use Cases

- **Webhooks** - Process incoming webhooks from third-party services
- **Data Processing** - Transform and validate data before storage
- **API Integrations** - Connect to external APIs (payment gateways, analytics, etc.)
- **Scheduled Tasks** - Run periodic jobs (data cleanup, reports, notifications)
- **Custom Business Logic** - Implement complex rules that can't be expressed in SQL
- **Authentication Extensions** - Custom OAuth flows, SSO integration
- **Email Templates** - Generate and send personalized emails

---

## Using the SDK (Recommended)

### Installation

**TypeScript/JavaScript:**

```bash
npm install @fluxbase/sdk
```

**Python:**

```bash
pip install fluxbase
```

### Quick Start

#### TypeScript/JavaScript

```typescript
import { FluxbaseClient } from "@fluxbase/sdk";

// Initialize client (requires authentication)
const client = new FluxbaseClient({
  url: "http://localhost:8080",
  apiKey: process.env.FLUXBASE_API_KEY,
});

// Create an edge function
const func = await client.functions.create({
  name: "hello-world",
  description: "My first edge function",
  code: `
    async function handler(req) {
      const data = JSON.parse(req.body || '{}');
      return {
        status: 200,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          message: \`Hello \${data.name || 'World'}!\`
        })
      };
    }
  `,
  enabled: true,
});

console.log("Function created:", func.name);

// Invoke the function
const result = await client.functions.invoke("hello-world", {
  name: "Alice",
});

console.log("Result:", result); // { message: "Hello Alice!" }

// List all functions
const functions = await client.functions.list();

// Get function details
const details = await client.functions.get("hello-world");

// View execution history
const executions = await client.functions.getExecutions("hello-world", {
  limit: 10,
});
```

#### Python

```python
from fluxbase import FluxbaseClient
import os

# Initialize client
client = FluxbaseClient(
    url="http://localhost:8080",
    api_key=os.environ['FLUXBASE_API_KEY']
)

# Create an edge function
func = client.functions.create(
    name="hello-world",
    description="My first edge function",
    code="""
async function handler(req) {
  const data = JSON.parse(req.body || '{}');
  return {
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      message: `Hello ${data.name || 'World'}!`
    })
  };
}
    """,
    enabled=True
)

print(f"Function created: {func['name']}")

# Invoke the function
result = client.functions.invoke("hello-world", {"name": "Alice"})
print(f"Result: {result}")  # { message: "Hello Alice!" }

# List all functions
functions = client.functions.list()

# Get function details
details = client.functions.get("hello-world")

# View execution history
executions = client.functions.get_executions("hello-world", limit=10)
```

---

### Managing Edge Functions

#### Create a Function

**TypeScript:**

```typescript
const func = await client.functions.create({
  name: "process-webhook",
  description: "Process incoming webhooks from Stripe",
  code: `
    async function handler(request) {
      const payload = JSON.parse(request.body || '{}');

      // Validate webhook signature
      const signature = request.headers['stripe-signature'];
      if (!signature) {
        return {
          status: 401,
          body: JSON.stringify({ error: 'Missing signature' })
        };
      }

      // Process webhook event
      console.log('Webhook event:', payload.type);

      return {
        status: 200,
        body: JSON.stringify({ received: true })
      };
    }
  `,
  enabled: true,
  timeout_seconds: 30,
  memory_limit_mb: 128,
  allow_net: true,
  allow_env: true,
  allow_read: false,
  allow_write: false,
});

console.log("Created function:", func.id);
```

**Python:**

```python
func = client.functions.create(
    name="process-webhook",
    description="Process incoming webhooks from Stripe",
    code="""
async function handler(request) {
  const payload = JSON.parse(request.body || '{}');

  // Validate webhook signature
  const signature = request.headers['stripe-signature'];
  if (!signature) {
    return {
      status: 401,
      body: JSON.stringify({ error: 'Missing signature' })
    };
  }

  // Process webhook event
  console.log('Webhook event:', payload.type);

  return {
    status: 200,
    body: JSON.stringify({ received: true })
  };
}
    """,
    enabled=True,
    timeout_seconds=30,
    memory_limit_mb=128,
    allow_net=True,
    allow_env=True,
    allow_read=False,
    allow_write=False
)

print(f"Created function: {func['id']}")
```

**Configuration Options:**

| Field             | Type    | Description                           | Default |
| ----------------- | ------- | ------------------------------------- | ------- |
| `name`            | string  | Function name (used in URL)           | -       |
| `description`     | string  | Optional description                  | -       |
| `code`            | string  | Function code (must export `handler`) | -       |
| `enabled`         | boolean | Whether function is active            | `true`  |
| `timeout_seconds` | number  | Max execution time (max 300s)         | `30`    |
| `memory_limit_mb` | number  | Max memory usage (max 1024MB)         | `128`   |
| `allow_net`       | boolean | Network access (fetch, WebSocket)     | `true`  |
| `allow_env`       | boolean | Environment variables access          | `true`  |
| `allow_read`      | boolean | Filesystem read access                | `false` |
| `allow_write`     | boolean | Filesystem write access               | `false` |
| `cron_schedule`   | string  | Cron schedule (future feature)        | `null`  |

#### Invoke a Function

**TypeScript:**

```typescript
// Simple invocation
const result = await client.functions.invoke("my-function", {
  email: "user@example.com",
  action: "send_welcome",
});

console.log("Function result:", result);

// With custom headers
const resultWithHeaders = await client.functions.invoke(
  "my-function",
  { data: "value" },
  {
    headers: {
      "X-Custom-Header": "value",
    },
  },
);
```

**Python:**

```python
# Simple invocation
result = client.functions.invoke(
    "my-function",
    {
        "email": "user@example.com",
        "action": "send_welcome"
    }
)

print(f"Function result: {result}")

# With custom headers
result_with_headers = client.functions.invoke(
    "my-function",
    {"data": "value"},
    headers={"X-Custom-Header": "value"}
)
```

#### List Functions

**TypeScript:**

```typescript
const functions = await client.functions.list();

functions.forEach((func) => {
  console.log(`${func.name} (v${func.version})`);
  console.log(`  Status: ${func.enabled ? "Enabled" : "Disabled"}`);
  console.log(`  Timeout: ${func.timeout_seconds}s`);
  console.log(`  Memory: ${func.memory_limit_mb}MB`);
});
```

**Python:**

```python
functions = client.functions.list()

for func in functions:
    status = "Enabled" if func['enabled'] else "Disabled"
    print(f"{func['name']} (v{func['version']})")
    print(f"  Status: {status}")
    print(f"  Timeout: {func['timeout_seconds']}s")
    print(f"  Memory: {func['memory_limit_mb']}MB")
```

#### Get Function Details

**TypeScript:**

```typescript
const func = await client.functions.get("my-function");

console.log("Name:", func.name);
console.log("Version:", func.version);
console.log("Code:", func.code);
console.log("Created:", func.created_at);
console.log("Updated:", func.updated_at);
```

**Python:**

```python
func = client.functions.get("my-function")

print(f"Name: {func['name']}")
print(f"Version: {func['version']}")
print(f"Code: {func['code']}")
print(f"Created: {func['created_at']}")
```

#### Update Function

**TypeScript:**

```typescript
// Update function code
await client.functions.update("my-function", {
  code: `
    async function handler(req) {
      // Updated implementation
      return {
        status: 200,
        body: JSON.stringify({ version: 2 })
      };
    }
  `,
});

// Enable/disable function
await client.functions.update("my-function", {
  enabled: false,
});

// Update timeout and permissions
await client.functions.update("my-function", {
  timeout_seconds: 60,
  allow_net: false,
});
```

**Python:**

```python
# Update function code
client.functions.update(
    "my-function",
    code="""
async function handler(req) {
  // Updated implementation
  return {
    status: 200,
    body: JSON.stringify({ version: 2 })
  };
}
    """
)

# Enable/disable function
client.functions.update("my-function", enabled=False)

# Update timeout and permissions
client.functions.update(
    "my-function",
    timeout_seconds=60,
    allow_net=False
)
```

#### Delete Function

**TypeScript:**

```typescript
await client.functions.delete("my-function");
console.log("Function deleted");
```

**Python:**

```python
client.functions.delete("my-function")
print("Function deleted")
```

#### View Execution History

**TypeScript:**

```typescript
const executions = await client.functions.getExecutions("my-function", {
  limit: 50,
  offset: 0,
});

executions.forEach((exec) => {
  console.log(`${exec.executed_at}: ${exec.status}`);
  console.log(`  Duration: ${exec.duration_ms}ms`);
  console.log(`  Status Code: ${exec.status_code}`);
  if (exec.error_message) {
    console.log(`  Error: ${exec.error_message}`);
  }
  if (exec.logs) {
    console.log(`  Logs: ${exec.logs}`);
  }
});
```

**Python:**

```python
executions = client.functions.get_executions(
    "my-function",
    limit=50,
    offset=0
)

for exec in executions:
    print(f"{exec['executed_at']}: {exec['status']}")
    print(f"  Duration: {exec['duration_ms']}ms")
    print(f"  Status Code: {exec['status_code']}")
    if exec.get('error_message'):
        print(f"  Error: {exec['error_message']}")
    if exec.get('logs'):
        print(f"  Logs: {exec['logs']}")
```

---

## Writing Functions

### Function Structure

Every edge function must export a `handler` function:

```typescript
async function handler(request: Request): Promise<Response> {
  // Your code here
  return {
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ message: "Success" }),
  };
}
```

### Request Object

The `request` parameter contains:

```typescript
interface Request {
  method: string; // HTTP method (GET, POST, etc.)
  url: string; // Full request URL
  headers: Record<string, string>; // Request headers
  body: string; // Request body as string
  user_id?: string; // Authenticated user ID (if available)
}
```

### Response Object

Return a response object:

```typescript
interface Response {
  status: number; // HTTP status code (200, 404, 500, etc.)
  headers?: Record<string, string>; // Response headers
  body: string; // Response body as string
}
```

---

## Function Examples

### Simple Hello World

```typescript
async function handler(request) {
  const data = JSON.parse(request.body || "{}");
  const name = data.name || "World";

  return {
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      message: `Hello ${name}!`,
      timestamp: new Date().toISOString(),
    }),
  };
}
```

### Query Fluxbase Database

```typescript
async function handler(request) {
  const url = Deno.env.get("FLUXBASE_URL");
  const token = Deno.env.get("FLUXBASE_TOKEN");

  // Query your database via REST API
  const response = await fetch(`${url}/api/v1/tables/users?select=id,email`, {
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
  });

  if (!response.ok) {
    return {
      status: response.status,
      body: JSON.stringify({ error: "Failed to fetch users" }),
    };
  }

  const users = await response.json();

  return {
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      count: users.length,
      users: users,
    }),
  };
}
```

### Process Webhook

```typescript
async function handler(request) {
  // Parse webhook payload
  const payload = JSON.parse(request.body || "{}");

  // Validate webhook signature (example)
  const signature = request.headers["x-webhook-signature"];
  if (!signature) {
    return {
      status: 401,
      body: JSON.stringify({ error: "Missing signature" }),
    };
  }

  // Process the webhook
  console.log("Received webhook:", payload.event);

  // Store in database
  const url = Deno.env.get("FLUXBASE_URL");
  const token = Deno.env.get("FLUXBASE_TOKEN");

  await fetch(`${url}/api/v1/tables/webhook_events`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      event_type: payload.event,
      data: payload,
      received_at: new Date().toISOString(),
    }),
  });

  return {
    status: 200,
    body: JSON.stringify({ success: true }),
  };
}
```

### Send Email Notification

```typescript
async function handler(request) {
  const data = JSON.parse(request.body || "{}");

  // Send email via external service (e.g., SendGrid)
  const response = await fetch("https://api.sendgrid.com/v3/mail/send", {
    method: "POST",
    headers: {
      Authorization: `Bearer ${Deno.env.get("SENDGRID_API_KEY")}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      personalizations: [
        {
          to: [{ email: data.email }],
          subject: data.subject,
        },
      ],
      from: { email: "noreply@yourapp.com" },
      content: [
        {
          type: "text/html",
          value: data.html_body,
        },
      ],
    }),
  });

  return {
    status: response.ok ? 200 : 500,
    body: JSON.stringify({
      success: response.ok,
      message: response.ok ? "Email sent" : "Failed to send email",
    }),
  };
}
```

### Data Validation

```typescript
async function handler(request) {
  const data = JSON.parse(request.body || "{}");

  // Validate required fields
  if (!data.email || !data.name) {
    return {
      status: 400,
      body: JSON.stringify({
        error: "Missing required fields",
        required: ["email", "name"],
      }),
    };
  }

  // Validate email format
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  if (!emailRegex.test(data.email)) {
    return {
      status: 400,
      body: JSON.stringify({ error: "Invalid email format" }),
    };
  }

  // Process validated data
  return {
    status: 200,
    body: JSON.stringify({
      valid: true,
      data: {
        email: data.email.toLowerCase(),
        name: data.name.trim(),
      },
    }),
  };
}
```

### API Proxy

```typescript
async function handler(request) {
  const data = JSON.parse(request.body || "{}");

  // Proxy request to external API
  const response = await fetch(`https://api.example.com/data/${data.id}`, {
    headers: {
      Authorization: `Bearer ${Deno.env.get("EXTERNAL_API_KEY")}`,
    },
  });

  const result = await response.json();

  // Transform and return
  return {
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      data: result,
      cached_at: new Date().toISOString(),
    }),
  };
}
```

---

## Authentication

Edge function invocation requires **at minimum an anon key** (JWT token with `role=anon`) by default for security and usage tracking.

### Using SDK (Automatic)

The SDK automatically handles authentication when you initialize the client:

```typescript
const client = new FluxbaseClient({
  url: "http://localhost:8080",
  apiKey: process.env.FLUXBASE_API_KEY, // Anon key, user token, or API key
});

// Authentication is automatic
const result = await client.functions.invoke("my-function", { data: "value" });
```

### Authentication Options

Functions accept multiple authentication methods, in order of privilege:

1. **Anon Key** (Minimum required) - JWT token with `role=anon`
2. **User JWT Token** - Authenticated user tokens from sign-in
3. **API Key** - Scoped access with specific permissions
4. **Service Key** - Elevated privileges (backend only)

**Accessing User Context in Functions:**

```typescript
async function handler(request) {
  const userId = request.user_id; // Available when user is authenticated

  if (userId) {
    console.log("Authenticated user:", userId);
  } else {
    console.log("Anonymous user (anon key)");
  }

  // Your logic here
}
```

### Allowing Unauthenticated Access

For truly public endpoints (e.g., webhooks):

```typescript
await client.functions.create({
  name: "public-webhook",
  code: `...`,
  allow_unauthenticated: true,
});
```

---

## Deployment Methods

Fluxbase supports two deployment methods:

### 1. API-Based Deployment (SDK)

Create and manage functions via the SDK (shown above). Ideal for:

- Dynamic function creation from admin dashboards
- Programmatic deployment workflows
- Quick prototyping and testing

### 2. File-Based Deployment

Deploy functions as TypeScript files for GitOps workflows.

#### Configuration

Enable file-based functions in your `fluxbase.yaml`:

```yaml
functions:
  enabled: true
  functions_dir: "./functions" # Path to functions directory
  default_timeout: 30
  max_timeout: 300
  default_memory_limit: 128
  max_memory_limit: 1024
```

#### Creating Functions

```bash
# Create functions directory
mkdir -p ./functions

# Create a function file
cat > ./functions/hello-world.ts << 'EOF'
async function handler(req) {
  const data = JSON.parse(req.body || '{}');
  return {
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      message: `Hello ${data.name || 'World'}!`
    })
  };
}
EOF
```

#### Docker Deployment

```yaml
# docker-compose.yml
services:
  fluxbase:
    image: ghcr.io/wayli-app/fluxbase:latest:latest
    volumes:
      - ./functions:/app/functions
    environment:
      FLUXBASE_FUNCTIONS_ENABLED: "true"
      FLUXBASE_FUNCTIONS_DIR: /app/functions
```

#### Reloading Functions

Use the SDK to reload functions from disk:

```typescript
// Reload all functions from filesystem
await client.admin.reloadFunctions();
```

**Response:**

```json
{
  "message": "Functions reloaded from filesystem",
  "total": 5,
  "created": ["new-function"],
  "updated": ["existing-function"],
  "errors": []
}
```

---

## Best Practices

### 1. Always Validate Input

```typescript
async function handler(request) {
  const data = JSON.parse(request.body || "{}");

  if (!data.email) {
    return {
      status: 400,
      body: JSON.stringify({ error: "email is required" }),
    };
  }

  // Process valid input
}
```

### 2. Use Environment Variables for Secrets

```typescript
// ❌ Bad - hardcoded secrets
const apiKey = "sk_live_123456789";

// ✅ Good - use environment variables
const apiKey = Deno.env.get("EXTERNAL_API_KEY");
```

### 3. Set Appropriate Timeouts

```typescript
await client.functions.create({
  name: "long-running-task",
  timeout_seconds: 120,
  code: "...",
});
```

### 4. Handle Errors Gracefully

```typescript
async function handler(request) {
  try {
    // Main logic
  } catch (error) {
    console.error(error);
    return {
      status: 500,
      body: JSON.stringify({ error: "Internal server error" }),
    };
  }
}
```

### 5. Return Proper HTTP Status Codes

```typescript
// Success
return { status: 200, body: "..." };

// Created
return { status: 201, body: "..." };

// Bad Request
return { status: 400, body: "..." };

// Not Found
return { status: 404, body: "..." };

// Internal Server Error
return { status: 500, body: "..." };
```

### 6. Minimize Cold Start Time

- Keep function code concise
- Avoid large dependencies
- Cache external API responses when possible

---

## Debugging

### View Execution Logs

Use the SDK to check execution history:

```typescript
const executions = await client.functions.getExecutions("my-function", {
  limit: 10,
});

executions.forEach((exec) => {
  console.log("Status:", exec.status);
  console.log("Duration:", exec.duration_ms, "ms");
  console.log("Logs:", exec.logs);
  if (exec.error_message) {
    console.log("Error:", exec.error_message);
  }
});
```

### Console Logging

Use `console.log()`, `console.error()`, `console.warn()` in your function:

```typescript
async function handler(request) {
  console.log("Request received:", request.method);
  console.log("Body:", request.body);

  const result = processData();
  console.log("Result:", result);

  return {
    status: 200,
    body: JSON.stringify(result),
  };
}
```

All console output is captured in the execution logs.

### Error Handling

Wrap your code in try-catch blocks:

```typescript
async function handler(request) {
  try {
    const data = JSON.parse(request.body || "{}");

    // Your logic here
    const result = await someAsyncOperation(data);

    return {
      status: 200,
      body: JSON.stringify(result),
    };
  } catch (error) {
    console.error("Function error:", error);

    return {
      status: 500,
      body: JSON.stringify({
        error: error.message,
        stack: error.stack,
      }),
    };
  }
}
```

---

## Limitations

- **Execution Time**: Maximum 300 seconds (5 minutes)
- **Memory**: Maximum 1024MB
- **Concurrency**: Each invocation runs in a separate process
- **Code Size**: Recommended &lt;1MB for faster cold starts
- **Log Retention**: 30 days

---

## Security Considerations

### Sandbox Isolation

- Functions run in isolated Deno processes
- Filesystem access is disabled by default
- Network access can be restricted
- No access to other functions' data

### Authentication

- Functions inherit authentication from the invoking request
- Access `request.user_id` to get authenticated user
- Use service tokens for background tasks

### Best Practices

1. **Validate all inputs** - Never trust user input
2. **Use HTTPS** - Always use TLS for external API calls
3. **Rotate secrets** - Update API keys and tokens regularly
4. **Minimal permissions** - Only enable required permissions
5. **Rate limiting** - Implement rate limits for public endpoints
6. **Error messages** - Don't leak sensitive information in errors

---

## Advanced: REST API Reference

For direct HTTP access or custom integrations, Fluxbase provides a complete REST API.

### Create Function

**POST** `/api/v1/functions`

**Headers:**

```
Authorization: Bearer {access_token}
Content-Type: application/json
```

**Request Body:**

```json
{
  "name": "my-function",
  "description": "Function description",
  "code": "async function handler(req) { ... }",
  "enabled": true,
  "timeout_seconds": 30,
  "memory_limit_mb": 128,
  "allow_net": true,
  "allow_env": true,
  "allow_read": false,
  "allow_write": false,
  "cron_schedule": null
}
```

**Response (201 Created):**

```json
{
  "id": "uuid",
  "name": "my-function",
  "description": "Function description",
  "code": "...",
  "version": 1,
  "enabled": true,
  "timeout_seconds": 30,
  "memory_limit_mb": 128,
  "allow_net": true,
  "allow_env": true,
  "allow_read": false,
  "allow_write": false,
  "cron_schedule": null,
  "created_at": "2024-10-26T10:00:00Z",
  "updated_at": "2024-10-26T10:00:00Z",
  "created_by": "user-uuid"
}
```

---

### List Functions

**GET** `/api/v1/functions`

**Response (200 OK):**

```json
[
  {
    "id": "uuid",
    "name": "my-function",
    "description": "Function description",
    "version": 1,
    "enabled": true,
    "created_at": "2024-10-26T10:00:00Z",
    "updated_at": "2024-10-26T10:00:00Z"
  }
]
```

---

### Get Function

**GET** `/api/v1/functions/:name`

**Response (200 OK):**

Returns full function details including code.

---

### Update Function

**PUT** `/api/v1/functions/:name`

```json
{
  "code": "async function handler(req) { /* updated */ }",
  "enabled": true,
  "timeout_seconds": 60
}
```

**Response (200 OK):**

Returns updated function with incremented version.

---

### Delete Function

**DELETE** `/api/v1/functions/:name`

**Response (204 No Content)**

---

### Invoke Function

**POST** `/api/v1/functions/:name/invoke`

```json
{
  "key": "value",
  "data": ["array", "of", "values"]
}
```

**Response (200 OK):**

Returns whatever your function's handler returns.

---

### Get Execution History

**GET** `/api/v1/functions/:name/executions?limit=50`

**Response (200 OK):**

```json
[
  {
    "id": "uuid",
    "function_id": "uuid",
    "trigger_type": "http",
    "status": "success",
    "status_code": 200,
    "duration_ms": 125,
    "result": "{\"message\":\"Success\"}",
    "logs": "console.log output here",
    "error_message": null,
    "executed_at": "2024-10-26T10:00:00Z",
    "completed_at": "2024-10-26T10:00:01Z"
  }
]
```

---

## Troubleshooting

### Function Not Executing

1. Check if function is enabled via SDK: `await client.functions.get("name")`
2. View execution logs: `await client.functions.getExecutions("name")`
3. Verify Deno is installed: `deno --version`

### Timeout Errors

Increase `timeout_seconds` using SDK:

```typescript
await client.functions.update("my-function", {
  timeout_seconds: 60,
});
```

### Permission Denied

Enable required permissions:

```typescript
await client.functions.update("my-function", {
  allow_net: true,
  allow_env: true,
});
```

### Network Errors

Check that `allow_net` is enabled and external API is accessible.

---

## Migration from Supabase

Fluxbase Edge Functions are similar to Supabase Edge Functions with some differences.

**Migration Example:**

Supabase:

```typescript
// functions/hello/index.ts
Deno.serve(async (req) => {
  return new Response(JSON.stringify({ message: "Hello" }), {
    headers: { "Content-Type": "application/json" },
  });
});
```

Fluxbase:

```typescript
async function handler(req) {
  return {
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ message: "Hello" }),
  };
}
```

---

## Learn More

- [Authentication](/docs/guides/authentication) - Authenticate to manage functions
- [Database Operations](/docs/guides/typescript-sdk/database) - Query data from functions
- [Webhooks](/docs/guides/webhooks) - Alternative event handling
