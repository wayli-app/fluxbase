---
title: "RPC (Remote Procedures)"
---

Fluxbase provides a Remote Procedure Call (RPC) system that allows you to define custom SQL-based procedures with type-safe input/output schemas, access control, and execution tracking.

## Overview

RPC procedures in Fluxbase enable:

- **Custom SQL Logic**: Define reusable SQL queries as named procedures
- **Type-Safe Schemas**: Define input/output schemas for validation
- **Access Control**: Restrict procedures by role or make them public
- **Execution Tracking**: Monitor procedure executions with logs
- **Async Support**: Execute long-running procedures asynchronously

Common use cases include complex queries, business logic encapsulation, data aggregations, and secure access to sensitive operations.

## Creating Procedures

Procedures are defined in SQL files with `@fluxbase:` annotations in comments.

### File Structure

Create procedure files in your RPC directory (default: `./rpc/`):

```
rpc/
├── get-user-orders.sql
├── generate-report.sql
└── namespace/
    └── custom-procedure.sql
```

### Basic Procedure

```sql
-- @fluxbase:name get-user-orders
-- @fluxbase:description Get orders for a specific user with pagination
-- @fluxbase:public true
-- @fluxbase:input {"user_id": "uuid", "limit?": "integer", "offset?": "integer"}
-- @fluxbase:output {"id": "uuid", "total": "decimal", "status": "text", "created_at": "timestamp"}

SELECT id, total, status, created_at
FROM orders
WHERE user_id = $user_id
ORDER BY created_at DESC
LIMIT COALESCE($limit, 10)
OFFSET COALESCE($offset, 0);
```

### Configuration Annotations

| Annotation | Description | Default |
|------------|-------------|---------|
| `@fluxbase:name` | Procedure name (used in API calls) | Filename |
| `@fluxbase:description` | Human-readable description | Empty |
| `@fluxbase:input` | Input parameter schema (JSON or `field:type` format) | `any` |
| `@fluxbase:output` | Output schema (JSON or `field:type` format) | `any` |
| `@fluxbase:allowed-tables` | Comma-separated tables the procedure can access | All |
| `@fluxbase:allowed-schemas` | Comma-separated database schemas | `public` |
| `@fluxbase:max-execution-time` | Maximum execution time | `30s` |
| `@fluxbase:require-role` | Required role to execute (e.g., `authenticated`, `admin`) | None |
| `@fluxbase:public` | Show in public procedure list | `false` |
| `@fluxbase:version` | Procedure version number | `1` |

### Schema Types

Available types for input/output schemas:

| Type | PostgreSQL Type |
|------|-----------------|
| `uuid` | UUID |
| `string`, `text` | TEXT |
| `number`, `int`, `integer` | INTEGER |
| `float`, `double`, `decimal` | NUMERIC |
| `boolean`, `bool` | BOOLEAN |
| `timestamp`, `datetime` | TIMESTAMPTZ |
| `date` | DATE |
| `time` | TIME |
| `json`, `jsonb`, `object` | JSONB |
| `array` | JSONB |

Optional fields are indicated with a `?` suffix: `"limit?": "integer"`

### Parameter Substitution

Use `$parameter_name` in your SQL to reference input parameters:

```sql
-- @fluxbase:input {"user_id": "uuid", "status": "text"}

SELECT * FROM orders
WHERE user_id = $user_id
  AND status = $status;
```

## SDK Usage

### Listing Procedures

```typescript
import { createClient } from '@fluxbase/sdk'

const client = createClient('http://localhost:8080', 'your-anon-key')

// List available public procedures
const { data: procedures, error } = await client.rpc.list()
console.log('Available procedures:', procedures)

// Filter by namespace
const { data } = await client.rpc.list('my-namespace')
```

### Invoking Procedures

```typescript
// Synchronous invocation
const { data, error } = await client.rpc.invoke('get-user-orders', {
  user_id: '123e4567-e89b-12d3-a456-426614174000',
  limit: 10,
  offset: 0
})

if (data) {
  console.log('Status:', data.status) // 'completed'
  console.log('Results:', data.result)
  console.log('Rows:', data.rows_returned)
  console.log('Duration:', data.duration_ms, 'ms')
}
```

### Async Execution

For long-running procedures:

```typescript
// Start async execution
const { data: asyncResult } = await client.rpc.invoke('generate-report', {
  start_date: '2024-01-01',
  end_date: '2024-12-31'
}, { async: true })

console.log('Execution ID:', asyncResult.execution_id)

// Poll for status
const { data: status } = await client.rpc.getStatus(asyncResult.execution_id)
console.log('Status:', status.status) // 'pending', 'running', 'completed', 'failed'

// Wait for completion with automatic polling
const { data: final } = await client.rpc.waitForCompletion(asyncResult.execution_id, {
  maxWaitMs: 60000, // Wait up to 1 minute
  onProgress: (exec) => console.log(`Status: ${exec.status}`)
})

console.log('Final result:', final.result)
```

### Execution Logs

```typescript
const { data: logs } = await client.rpc.getLogs('execution-uuid')

for (const log of logs) {
  console.log(`[${log.level}] ${log.message}`)
}
```

### Namespaces

Procedures can be organized into namespaces:

```typescript
// Invoke procedure in a specific namespace
const { data } = await client.rpc.invoke('my-procedure', params, {
  namespace: 'reports'
})
```

## Admin Management

Administrators can manage procedures via the admin API.

### Syncing Procedures

```typescript
// Sync from filesystem
const { data, error } = await client.admin.rpc.sync()

// Sync with provided procedure code
const { data, error } = await client.admin.rpc.sync({
  namespace: 'default',
  procedures: [{
    name: 'my-procedure',
    code: `
      -- @fluxbase:description My custom procedure
      -- @fluxbase:public true
      SELECT * FROM users WHERE active = true;
    `,
  }],
  options: {
    delete_missing: false, // Don't remove procedures not in this sync
    dry_run: false,        // Preview changes without applying
  }
})

if (data) {
  console.log(`Created: ${data.summary.created}`)
  console.log(`Updated: ${data.summary.updated}`)
  console.log(`Deleted: ${data.summary.deleted}`)
}
```

### Managing Procedures

```typescript
// List all procedures (including private)
const { data: procedures } = await client.admin.rpc.list()

// Get procedure details
const { data: procedure } = await client.admin.rpc.get('default', 'get-user-orders')
console.log('SQL:', procedure.sql_query)

// Update procedure settings
const { data } = await client.admin.rpc.update('default', 'get-user-orders', {
  enabled: true,
  max_execution_time_seconds: 60,
  is_public: true,
})

// Enable/disable procedure
await client.admin.rpc.toggle('default', 'get-user-orders', false)

// Delete procedure
await client.admin.rpc.delete('default', 'get-user-orders')
```

### Monitoring Executions

```typescript
// List all executions
const { data: executions } = await client.admin.rpc.listExecutions()

// Filter executions
const { data } = await client.admin.rpc.listExecutions({
  namespace: 'default',
  procedure: 'get-user-orders',
  status: 'failed',
  limit: 50,
})

// Get execution details
const { data: execution } = await client.admin.rpc.getExecution('execution-uuid')

// Get execution logs
const { data: logs } = await client.admin.rpc.getExecutionLogs('execution-uuid')

// Cancel running execution
await client.admin.rpc.cancelExecution('execution-uuid')
```

## API Reference

### Public Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/rpc/procedures` | List public procedures |
| `POST` | `/api/v1/rpc/:namespace/:name` | Invoke procedure |
| `GET` | `/api/v1/rpc/executions/:id` | Get execution status |
| `GET` | `/api/v1/rpc/executions/:id/logs` | Get execution logs |

### Admin Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/admin/rpc/namespaces` | List namespaces |
| `GET` | `/api/v1/admin/rpc/procedures` | List all procedures |
| `GET` | `/api/v1/admin/rpc/procedures/:namespace/:name` | Get procedure |
| `PUT` | `/api/v1/admin/rpc/procedures/:namespace/:name` | Update procedure |
| `DELETE` | `/api/v1/admin/rpc/procedures/:namespace/:name` | Delete procedure |
| `POST` | `/api/v1/admin/rpc/sync` | Sync procedures |
| `GET` | `/api/v1/admin/rpc/executions` | List executions |
| `GET` | `/api/v1/admin/rpc/executions/:id` | Get execution |
| `GET` | `/api/v1/admin/rpc/executions/:id/logs` | Get execution logs |

### Invoke Request

```typescript
{
  params: {           // Input parameters
    user_id: "uuid",
    limit: 10
  },
  async: false        // Run asynchronously (default: false)
}
```

### Invoke Response

```typescript
{
  execution_id: "uuid",
  status: "completed",     // pending, running, completed, failed, cancelled, timeout
  result: [...],           // Query results (when completed)
  rows_returned: 10,
  duration_ms: 45,
  error: null              // Error message (when failed)
}
```

## Security

### Access Control

Control who can execute procedures:

```sql
-- Only authenticated users
-- @fluxbase:require-role authenticated

-- Only admins
-- @fluxbase:require-role admin

-- Public (no auth required)
-- @fluxbase:public true
```

### Table Restrictions

Limit which tables a procedure can access:

```sql
-- @fluxbase:allowed-tables orders,order_items,products
-- @fluxbase:allowed-schemas public

SELECT * FROM orders
JOIN order_items ON orders.id = order_items.order_id
JOIN products ON order_items.product_id = products.id;
```

### Best Practices

1. **Use Parameterized Queries**: Always use `$param` syntax for user input
2. **Limit Access**: Restrict tables and schemas to only what's needed
3. **Set Timeouts**: Use `@fluxbase:max-execution-time` to prevent runaway queries
4. **Validate Input**: Define input schemas for type validation
5. **Monitor Executions**: Review execution logs for failed procedures

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `FLUXBASE_RPC_ENABLED` | Enable RPC functionality | `true` |
| `FLUXBASE_RPC_PROCEDURES_DIR` | Directory for procedure files | `./rpc` |
| `FLUXBASE_RPC_AUTO_LOAD_ON_BOOT` | Load procedures on startup | `true` |
| `FLUXBASE_RPC_DEFAULT_MAX_EXECUTION_TIME` | Default timeout | `30s` |
| `FLUXBASE_RPC_MAX_MAX_EXECUTION_TIME` | Maximum allowed timeout | `5m` |
| `FLUXBASE_RPC_DEFAULT_MAX_ROWS` | Default max rows returned | `1000` |

## Examples

### User Activity Report

```sql
-- @fluxbase:name user-activity-report
-- @fluxbase:description Generate user activity report for a date range
-- @fluxbase:require-role admin
-- @fluxbase:max-execution-time 2m
-- @fluxbase:input {"start_date": "date", "end_date": "date"}
-- @fluxbase:output {"user_id": "uuid", "email": "text", "login_count": "integer", "last_login": "timestamp"}

SELECT
  u.id as user_id,
  u.email,
  COUNT(l.id) as login_count,
  MAX(l.created_at) as last_login
FROM auth.users u
LEFT JOIN auth.user_logins l ON u.id = l.user_id
  AND l.created_at BETWEEN $start_date AND $end_date
GROUP BY u.id, u.email
ORDER BY login_count DESC;
```

### Order Statistics

```sql
-- @fluxbase:name order-statistics
-- @fluxbase:description Get order statistics by status
-- @fluxbase:public true
-- @fluxbase:allowed-tables orders
-- @fluxbase:input {"status?": "text"}

SELECT
  status,
  COUNT(*) as count,
  SUM(total) as total_amount,
  AVG(total) as avg_amount
FROM orders
WHERE ($status IS NULL OR status = $status)
GROUP BY status;
```

### Search with Pagination

```sql
-- @fluxbase:name search-products
-- @fluxbase:description Search products with full-text search
-- @fluxbase:public true
-- @fluxbase:input {"query": "text", "limit?": "integer", "offset?": "integer"}

SELECT id, name, description, price,
  ts_rank(search_vector, plainto_tsquery('english', $query)) as rank
FROM products
WHERE search_vector @@ plainto_tsquery('english', $query)
ORDER BY rank DESC
LIMIT COALESCE($limit, 20)
OFFSET COALESCE($offset, 0);
```

## Troubleshooting

### Procedure Not Found

**Symptoms:** `404 Procedure not found` error

**Solutions:**
- Check procedure is enabled (`enabled: true`)
- Verify namespace in request matches procedure namespace
- For public endpoints, ensure `@fluxbase:public true` is set
- Run sync to reload procedures from filesystem

### Permission Denied

**Symptoms:** `403 Forbidden` error

**Solutions:**
- Check user has required role (`@fluxbase:require-role`)
- Verify authentication token is valid
- For public procedures, ensure `@fluxbase:public true`

### Execution Timeout

**Symptoms:** `timeout` status on execution

**Solutions:**
- Increase `@fluxbase:max-execution-time` in procedure
- Optimize SQL query (add indexes, reduce data)
- Consider using async execution for long-running queries
- Check `FLUXBASE_RPC_MAX_MAX_EXECUTION_TIME` config limit

### Invalid Parameters

**Symptoms:** Parameter validation errors

**Solutions:**
- Verify input matches defined `@fluxbase:input` schema
- Check parameter types (uuid, integer, etc.)
- Ensure required parameters are provided (non-`?` fields)

## Next Steps

- [Configuration Reference](/docs/reference/configuration) - All RPC configuration options
- [Authentication](/docs/guides/authentication) - Configure user authentication
- [Row-Level Security](/docs/guides/row-level-security) - Secure data access
- [Background Jobs](/docs/guides/jobs) - Long-running tasks with progress tracking
