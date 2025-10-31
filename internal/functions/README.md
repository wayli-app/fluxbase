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
  const data = JSON.parse(request.body || '{}');

  // Return response
  return {
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ message: "Success", data })
  };
}
```

### Available APIs

Functions have access to:
- **Deno standard library**: All Deno APIs
- **Fluxbase API**: Access via environment variables
  - `FLUXBASE_URL`: API endpoint
  - `FLUXBASE_TOKEN`: Authentication token

### Example: Database Query

```typescript
async function handler(request) {
  const url = Deno.env.get("FLUXBASE_URL");
  const token = Deno.env.get("FLUXBASE_TOKEN");

  // Query Fluxbase REST API
  const response = await fetch(`${url}/api/v1/tables/users`, {
    headers: {
      "Authorization": `Bearer ${token}`,
      "Content-Type": "application/json"
    }
  });

  const users = await response.json();

  return {
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ count: users.length, users })
  };
}
```

## Security Permissions

Functions run in a sandboxed Deno environment with configurable permissions:

| Permission | Description | Default |
|------------|-------------|---------|
| `allow_net` | Network access | `true` |
| `allow_env` | Environment variables | `true` |
| `allow_read` | Filesystem read | `false` |
| `allow_write` | Filesystem write | `false` |

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
edge_function_executions (
  id, function_id, trigger_type, status, status_code,
  duration_ms, result, logs, error_message,
  executed_at, completed_at
)

-- Database triggers (future)
edge_function_triggers (
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
