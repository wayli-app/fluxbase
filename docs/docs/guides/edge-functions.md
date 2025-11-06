# Edge Functions

Edge Functions are serverless functions powered by the Deno runtime that execute JavaScript/TypeScript code in response to HTTP requests. They enable you to run custom backend logic without managing infrastructure.

## Overview

Edge Functions in Fluxbase provide:

- **Deno Runtime** - Execute TypeScript/JavaScript code with modern ES modules
- **HTTP Triggered** - Invoke functions via REST API
- **Secure Sandbox** - Configurable permissions for network, environment, and filesystem access
- **Database Access** - Query your Fluxbase database via REST API
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

## Quick Start

### 1. Create Your First Function

```bash
curl -X POST http://localhost:8080/api/v1/functions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "name": "hello-world",
    "description": "My first edge function",
    "code": "async function handler(req) { return { status: 200, body: JSON.stringify({ message: \"Hello World!\" }) }; }",
    "enabled": true
  }'
```

### 2. Invoke the Function

```bash
curl -X POST http://localhost:8080/api/v1/functions/hello-world/invoke \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ANON_KEY" \
  -d '{"name": "Alice"}'
```

Response:

```json
{
  "message": "Hello World!"
}
```

> **Note:** Function invocation requires at minimum an **anon key** (JWT token with `role=anon`). See the [Authentication](#authentication) section below for details.

### 3. View Execution History

```bash
curl http://localhost:8080/api/v1/functions/hello-world/executions \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## Authentication

Edge function invocation requires **at minimum an anon key** (JWT token with `role=anon`) by default. This follows the Supabase authentication model where even "public" endpoints require project identification for security and usage tracking.

### Authentication Options

Functions accept multiple authentication methods, in order of privilege:

#### 1. Anon Key (Minimum Required)

An anon key is a JWT token with `role=anon` that allows anonymous access while still identifying your project.

**Generate an anon key:**

```bash
# Using the helper script
./scripts/generate-keys.sh
# Select option 3: Generate Anon Key
```

**Use in requests:**

```bash
curl -X POST http://localhost:8080/api/v1/functions/my-function/invoke \
  -H "Authorization: Bearer YOUR_ANON_KEY" \
  -H "Content-Type: application/json" \
  -d '{"data": "value"}'
```

**Client-side usage:**

```javascript
// Store anon key in your environment
const FLUXBASE_ANON_KEY = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...";

// Use in API calls
fetch("http://localhost:8080/api/v1/functions/my-function/invoke", {
  method: "POST",
  headers: {
    Authorization: `Bearer ${FLUXBASE_ANON_KEY}`,
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ data: "value" }),
});
```

#### 2. User JWT Token

Authenticated user tokens from sign-in provide user context to functions.

```bash
curl -X POST http://localhost:8080/api/v1/functions/my-function/invoke \
  -H "Authorization: Bearer USER_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"data": "value"}'
```

Functions can access the authenticated user ID:

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

#### 3. API Key

API keys provide scoped access with specific permissions.

```bash
curl -X POST http://localhost:8080/api/v1/functions/my-function/invoke \
  -H "X-API-Key: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"data": "value"}'
```

Create an API key with `execute:functions` scope:

```bash
curl -X POST http://localhost:8080/api/v1/auth/api-keys \
  -H "Authorization: Bearer ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My API Key",
    "scopes": ["execute:functions"],
    "rate_limit_per_minute": 100
  }'
```

#### 4. Service Key

Service keys have elevated privileges and bypass RLS policies. Use only for backend services.

```bash
curl -X POST http://localhost:8080/api/v1/functions/my-function/invoke \
  -H "X-Service-Key: YOUR_SERVICE_KEY" \
  -H "Content-Type": "application/json" \
  -d '{"data": "value"}'
```

### Allowing Completely Unauthenticated Access

For truly public endpoints that don't require any authentication (not even an anon key), set `allow_unauthenticated: true`. This is useful for public webhooks or endpoints that must be accessible without credentials.

**Via API:**

```json
{
  "name": "public-webhook",
  "code": "async function handler(req) { ... }",
  "allow_unauthenticated": true
}
```

**Via code comment:**

```typescript
// @fluxbase:allow-unauthenticated
async function handler(request) {
  // Process webhook without authentication
  return {
    status: 200,
    body: JSON.stringify({ success: true }),
  };
}
```

> **Security Note:** Use `allow_unauthenticated: true` sparingly. Anon keys provide better security, rate limiting, and usage tracking for most public endpoints.

### Authentication Error Responses

**No authentication provided (401):**

```json
{
  "error": "Authentication required. Provide an anon key (Bearer token with role=anon), API key (X-API-Key header), or service key (X-Service-Key header). To allow completely unauthenticated access, set allow_unauthenticated=true on the function."
}
```

**Function disabled (403):**

```json
{
  "error": "Function is disabled"
}
```

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

## Examples

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

## Deployment Methods

Fluxbase supports two ways to deploy edge functions:

### 1. API-Based Deployment (Default)

Create and manage functions via the REST API. This is ideal for:

- Dynamic function creation from admin dashboards
- Programmatic deployment workflows
- Quick prototyping and testing

See the [Management API](#management-api) section below for details.

### 2. File-Based Deployment

Deploy functions as TypeScript files mounted to the functions directory. This is ideal for:

- GitOps workflows and version control
- Docker/Kubernetes deployments
- CI/CD pipelines
- Team collaboration

#### Configuration

Enable file-based functions in your `fluxbase.yaml`:

```yaml
functions:
  enabled: true
  functions_dir: "./functions" # Path to functions directory
  default_timeout: 30 # Default timeout in seconds
  max_timeout: 300 # Maximum timeout (5 minutes)
  default_memory_limit: 128 # Default memory limit in MB
  max_memory_limit: 1024 # Maximum memory limit (1GB)
```

Or via environment variables:

```bash
FLUXBASE_FUNCTIONS_ENABLED=true
FLUXBASE_FUNCTIONS_DIR=./functions
FLUXBASE_FUNCTIONS_DEFAULT_TIMEOUT=30
FLUXBASE_FUNCTIONS_MAX_TIMEOUT=300
FLUXBASE_FUNCTIONS_DEFAULT_MEMORY_LIMIT=128
FLUXBASE_FUNCTIONS_MAX_MEMORY_LIMIT=1024
```

#### Creating Functions

Create a TypeScript file in your functions directory:

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

Mount your functions directory as a volume:

```yaml
# docker-compose.yml
services:
  fluxbase:
    image: fluxbase/fluxbase:latest
    volumes:
      - ./functions:/app/functions
    environment:
      FLUXBASE_FUNCTIONS_ENABLED: "true"
      FLUXBASE_FUNCTIONS_DIR: /app/functions
```

#### Kubernetes Deployment

Use a PersistentVolumeClaim or ConfigMap:

```yaml
# Using PVC (recommended for production)
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: fluxbase-functions
spec:
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 1Gi
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: fluxbase
spec:
  template:
    spec:
      containers:
        - name: fluxbase
          image: fluxbase/fluxbase:latest
          env:
            - name: FLUXBASE_FUNCTIONS_ENABLED
              value: "true"
            - name: FLUXBASE_FUNCTIONS_DIR
              value: /app/functions
          volumeMounts:
            - name: functions
              mountPath: /app/functions
      volumes:
        - name: functions
          persistentVolumeClaim:
            claimName: fluxbase-functions
```

#### Reloading Functions

Functions are automatically detected from the filesystem at runtime. To reload functions after updating files:

**Via API (Admin Only):**

```bash
curl -X POST http://localhost:8080/api/v1/admin/functions/reload \
  -H "Authorization: Bearer ADMIN_TOKEN"
```

**Via Dashboard:**

1. Navigate to the Functions page
2. Functions are automatically reloaded when the page loads
3. Or click the "Reload from Disk" button (if available)

The reload endpoint will:

- Scan the functions directory for `.ts` files
- Create new functions in the database
- Update existing functions if code has changed
- Skip invalid or malformed files
- Return a summary of created, updated, and error counts

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

#### File Naming Rules

- Function names are derived from filenames
- Only `.ts` files are processed
- Valid characters: `a-z`, `A-Z`, `0-9`, `-`, `_`
- Reserved names are blocked: `.`, `..`, `index`, `main`, `handler`, `_`, `-`
- Path traversal attempts (e.g., `../malicious.ts`) are rejected

#### Security

File-based deployment includes security measures:

- **Path traversal prevention**: Strict validation prevents directory traversal attacks
- **Name validation**: Function names are validated against a regex pattern
- **Admin-only reload**: Only dashboard admins can trigger function reloads
- **Filesystem as source of truth**: Files override database entries
- **Audit logging**: All reload operations are logged

#### GitOps Workflow Example

```bash
# 1. Add function to Git repository
git add functions/new-feature.ts
git commit -m "Add new-feature function"
git push

# 2. Deploy to staging (CI/CD pipeline)
kubectl apply -f k8s/staging/
# Wait for pods to restart with new volume

# 3. Reload functions via API
curl -X POST https://staging-api.example.com/api/v1/admin/functions/reload \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# 4. Test the function
curl -X POST https://staging-api.example.com/api/v1/functions/new-feature/invoke \
  -H "Content-Type: application/json" \
  -d '{"test": true}'

# 5. Deploy to production
kubectl apply -f k8s/production/
curl -X POST https://api.example.com/api/v1/admin/functions/reload \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

## Management API

### Create Function

**POST** `/api/v1/functions`

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

### Get Function

**GET** `/api/v1/functions/:name`

**Response (200 OK):**

Returns full function details including code.

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

### Delete Function

**DELETE** `/api/v1/functions/:name`

**Response (204 No Content)**

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

## Configuration Options

### Permissions

Configure what your function can access:

| Permission    | Description                       | Default |
| ------------- | --------------------------------- | ------- |
| `allow_net`   | Network access (fetch, WebSocket) | `true`  |
| `allow_env`   | Environment variables             | `true`  |
| `allow_read`  | Filesystem read access            | `false` |
| `allow_write` | Filesystem write access           | `false` |

**Example:**

```json
{
  "name": "secure-function",
  "code": "...",
  "allow_net": false,
  "allow_env": false,
  "allow_read": false,
  "allow_write": false
}
```

### Execution Limits

| Setting           | Description            | Default | Max    |
| ----------------- | ---------------------- | ------- | ------ |
| `timeout_seconds` | Maximum execution time | 30s     | 300s   |
| `memory_limit_mb` | Maximum memory usage   | 128MB   | 1024MB |

### Environment Variables

Functions have access to these environment variables:

- `FLUXBASE_URL` - Your Fluxbase API endpoint
- `FLUXBASE_TOKEN` - Service token for API access
- `DENO_DEPLOYMENT_ID` - Unique ID for this deployment

You can also set custom environment variables via your function configuration (future feature).

## Debugging

### View Execution Logs

Check execution history to see logs and errors:

```bash
curl http://localhost:8080/api/v1/functions/my-function/executions?limit=10
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

```json
{
  "name": "long-running-task",
  "timeout_seconds": 120,
  "code": "..."
}
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

## Limitations

- **Execution Time**: Maximum 300 seconds (5 minutes)
- **Memory**: Maximum 1024MB
- **Concurrency**: Each invocation runs in a separate process
- **Code Size**: Recommended &lt;1MB for faster cold starts
- **Log Retention**: 30 days

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

## Future Features

The following features are planned for future releases:

- **Cron Scheduler** - Schedule periodic function execution
- **Database Triggers** - Execute functions on table INSERT/UPDATE/DELETE
- **Function Templates** - Pre-built examples for common use cases
- **Admin UI** - Monaco editor for code editing with syntax highlighting
- **NPM Packages** - Import external packages from npm/deno.land
- **Function Versioning** - Deploy and rollback specific versions
- **A/B Testing** - Split traffic between function versions

## Migration from Supabase

Fluxbase Edge Functions are similar to Supabase Edge Functions:

**Similarities:**

- Deno runtime
- TypeScript/JavaScript support
- HTTP invocation
- Database access via REST API
- Environment variables

**Differences:**

- **Invocation**: Fluxbase uses `/functions/:name/invoke` instead of URL routing
- **Deployment**: Managed via REST API instead of CLI
- **Triggers**: Database triggers coming in future release

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

## Troubleshooting

### Function Not Executing

1. Check if function is enabled: `GET /api/v1/functions/:name`
2. View execution logs for errors: `GET /api/v1/functions/:name/executions`
3. Verify Deno is installed: `deno --version`

### Timeout Errors

Increase `timeout_seconds`:

```json
{
  "timeout_seconds": 60
}
```

### Permission Denied

Enable required permissions:

```json
{
  "allow_net": true,
  "allow_env": true
}
```

### Network Errors

Check that `allow_net` is enabled and external API is accessible.

## Next Steps

- [API Reference](/api/edge-functions) - Complete API documentation
- [Database Access](database) - Query your data from functions
- [Authentication](authentication) - Secure your functions
- [Examples Repository](https://github.com/your-org/fluxbase-examples) - More code examples
