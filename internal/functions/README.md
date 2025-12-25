# Edge Functions - Deno Runtime (MVP)

Serverless functions powered by Deno runtime for executing JavaScript/TypeScript code.

## Features

✅ **Implemented (MVP)**:

- Deno 2.5.4 runtime integration via CLI
- Function storage in PostgreSQL
- HTTP invocation endpoint
- Configurable permissions (net, env, read, write)
- Execution logging and history
- Timeout enforcement (30s default)
- User authentication integration

⏸️ **Deferred (Future Enhancement)**:

- Cron scheduler for scheduled execution
- Database triggers for event-driven execution
- Admin UI integration
- Function templates and examples

## API Endpoints

### Management Endpoints

```bash
# Create function
POST /api/v1/functions
{
  "name": "hello-world",
  "description": "A simple hello world function",
  "code": "async function handler(req) { return { status: 200, body: 'Hello World!' }; }",
  "enabled": true,
  "timeout_seconds": 30,
  "allow_net": true,
  "allow_env": true
}

# List all functions
GET /api/v1/functions

# Get specific function
GET /api/v1/functions/:name

# Update function
PUT /api/v1/functions/:name
{
  "code": "// Updated code",
  "enabled": true
}

# Delete function
DELETE /api/v1/functions/:name
```

### Invocation Endpoint

```bash
# Invoke function
POST /api/v1/functions/:name/invoke
{
  "key": "value"
}

# View execution history
GET /api/v1/functions/:name/executions?limit=50
```

## Writing Functions

Functions must export a `handler` function that accepts a request object:

```typescript
async function handler(request) {
  // Request object structure:
  // {
  //   method: string,
  //   url: string,
  //   headers: { [key: string]: string },
  //   body: string,
  //   user_id: string (if authenticated)
  // }

  // Your function logic here
  const data = JSON.parse(request.body || "{}");

  // Return response
  return {
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ message: "Success", data }),
  };
}
```

### Available APIs

Functions have access to:

- **Deno standard library**: All Deno APIs
- **Fluxbase API**: Access via environment variables
  - `FLUXBASE_BASE_URL`: API endpoint
  - `FLUXBASE_TOKEN`: Authentication token

### Example: Database Query

```typescript
async function handler(request) {
  const url = Deno.env.get("FLUXBASE_BASE_URL");
  const token = Deno.env.get("FLUXBASE_TOKEN");

  // Query Fluxbase REST API
  const response = await fetch(`${url}/api/v1/tables/users`, {
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
  });

  const users = await response.json();

  return {
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ count: users.length, users }),
  };
}
```

## Security Permissions

Functions run in a sandboxed Deno environment with configurable permissions:

| Permission              | Description                   | Default |
| ----------------------- | ----------------------------- | ------- |
| `allow_net`             | Network access                | `true`  |
| `allow_env`             | Environment variables         | `true`  |
| `allow_read`            | Filesystem read               | `false` |
| `allow_write`           | Filesystem write              | `false` |
| `allow_unauthenticated` | Allow invocation without auth | `false` |

### Authentication Configuration

By default, **all function management endpoints require authentication** (JWT, API key, or service key):

- Creating, listing, updating, and deleting functions
- Viewing execution history

**Function invocation** requires **at minimum an anon key** (JWT token with `role=anon`) by default. This follows the Supabase authentication model where even "public" endpoints require project identification.

#### Authentication Options for Invocation

1. **Anon Key** (JWT with `role=anon`) - Minimum required authentication

   ```bash
   curl -X POST https://fluxbase.example.com/api/v1/functions/my-function/invoke \
     -H "Authorization: Bearer $ANON_KEY" \
     -d '{"data": "value"}'
   ```

2. **User JWT** - Authenticated user token

   ```bash
   curl -X POST https://fluxbase.example.com/api/v1/functions/my-function/invoke \
     -H "Authorization: Bearer $USER_TOKEN" \
     -d '{"data": "value"}'
   ```

3. **API Key** - Scoped API key with `execute:functions` permission

   ```bash
   curl -X POST https://fluxbase.example.com/api/v1/functions/my-function/invoke \
     -H "X-API-Key: $API_KEY" \
     -d '{"data": "value"}'
   ```

4. **Service Key** - Elevated privileges for backend services
   ```bash
   curl -X POST https://fluxbase.example.com/api/v1/functions/my-function/invoke \
     -H "X-Service-Key: $SERVICE_KEY" \
     -d '{"data": "value"}'
   ```

#### Generating Anon Keys

Generate an anon key using the helper script:

```bash
./scripts/generate-keys.sh
# Select option 3: Generate Anon Key
```

The anon key is a JWT token signed with your `JWT_SECRET` containing `role=anon`. Distribute this key to your client applications for public API access.

#### Allowing Completely Unauthenticated Access

For truly public endpoints that don't require any authentication (not even an anon key), set `allow_unauthenticated: true`:

1. **API Request** - Set `allow_unauthenticated` when creating/updating:

   ```json
   {
     "name": "public-webhook",
     "code": "...",
     "allow_unauthenticated": true
   }
   ```

2. **Code Comment** - Add directive in your function code:
   ```typescript
   // @fluxbase:allow-unauthenticated
   async function handler(request) {
     return { status: 200, body: "Truly public endpoint" };
   }
   ```

The comment directive is parsed when functions are loaded via the `/api/v1/admin/functions/reload` endpoint.

**Note**: Use `allow_unauthenticated: true` only for public webhooks or endpoints that must be accessible without any credentials. For most use cases, anon keys provide better security and usage tracking.

### Rate Limiting

Functions support per-user/IP rate limiting via code annotations or API:

1. **Code Comment** - Add directive in your function code:

   ```typescript
   // @fluxbase:rate-limit 100/min
   async function handler(request) {
     return { status: 200, body: "Rate limited endpoint" };
   }
   ```

2. **API Request** - Set rate limits when creating/updating:

   ```json
   {
     "name": "api-endpoint",
     "code": "...",
     "rate_limit_per_minute": 100,
     "rate_limit_per_hour": 1000,
     "rate_limit_per_day": 10000
   }
   ```

**Supported time windows:**

- `N/min` - Requests per minute
- `N/hour` - Requests per hour
- `N/day` - Requests per day

**Rate limit key:**

- Authenticated users: Limited per user ID
- Anonymous requests: Limited per IP address

When rate limit is exceeded, the function returns `429 Too Many Requests` with headers:

- `Retry-After`: Seconds until reset
- `X-RateLimit-Limit`: Configured limit
- `X-RateLimit-Remaining`: Remaining requests
- `X-RateLimit-Reset`: Unix timestamp of reset

## Execution Limits

- **Timeout**: 30 seconds (configurable up to 300s)
- **Memory**: 128MB (configurable up to 1024MB)
- **Concurrency**: No limit (runs in separate Deno process per invocation)

## Database Schema

```sql
-- Functions table
edge_functions (
  id, name, code, version, enabled,
  timeout_seconds, memory_limit_mb,
  allow_net, allow_env, allow_read, allow_write,
  cron_schedule, created_at, updated_at, created_by
)

-- Execution logs (30-day retention)
edge_executions (
  id, function_id, trigger_type, status, status_code,
  duration_ms, result, logs, error_message,
  executed_at, completed_at
)

-- Database triggers (future)
edge_triggers (
  id, function_id, table_name, events, condition
)
```

## Architecture

**Runtime**: Deno CLI integration (shell execution)

- Simple, no CGO dependencies
- Easy to deploy (just install Deno binary)
- Can be optimized with embedded Deno core later

**Execution Flow**:

1. HTTP request → `/api/v1/functions/:name/invoke`
2. Load function code from PostgreSQL
3. Wrap code with runtime bridge
4. Execute via `deno run` with permissions
5. Capture stdout (response) and stderr (logs)
6. Log execution to database
7. Return response to client

**Storage**: PostgreSQL for function code and execution logs

## Testing Functions

```bash
# Create a test function
curl -X POST http://localhost:8080/api/v1/functions \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test",
    "code": "async function handler(req) { return { status: 200, body: JSON.stringify({ message: \"Hello from Deno!\" }) }; }",
    "enabled": true
  }'

# Invoke the function
curl -X POST http://localhost:8080/api/v1/functions/test/invoke \
  -H "Content-Type: application/json" \
  -d '{"name": "World"}'

# View execution logs
curl http://localhost:8080/api/v1/functions/test/executions
```

## Future Enhancements

- **Cron Scheduler**: Scheduled function execution
- **Database Triggers**: Execute functions on table changes (INSERT/UPDATE/DELETE)
- **Admin UI**: Monaco Editor for code editing, execution logs viewer
- **Function Templates**: Pre-built examples (webhook handler, email sender, etc.)
- **Performance**: Function code caching, connection pooling
- **Monitoring**: Real-time execution metrics dashboard

## Migration Path from Supabase

Fluxbase edge functions are compatible with Supabase's approach:

- TypeScript/JavaScript runtime (Deno)
- HTTP-triggered execution
- Access to database via REST API
- Environment variable support

Main difference: Fluxbase uses HTTP invocation instead of URL routing (can be added later).

## Files

- `runtime.go` - Deno execution manager
- `storage.go` - PostgreSQL persistence
- `handler.go` - HTTP API endpoints
- `migrations/012_edge_functions.up.sql` - Database schema

## License

MIT
