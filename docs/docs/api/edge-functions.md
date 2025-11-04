# Edge Functions API Reference

Complete API reference for Fluxbase Edge Functions.

## Base URL

```
/api/v1/functions
```

## Authentication

All management endpoints require authentication. Include your access token in the Authorization header:

```
Authorization: Bearer YOUR_ACCESS_TOKEN
```

Function invocation endpoints can be public or authenticated depending on your configuration.

---

## Management Endpoints

### Create Function

Create a new edge function.

**Endpoint:** `POST /api/v1/functions`

**Headers:**
```
Authorization: Bearer YOUR_ACCESS_TOKEN
Content-Type: application/json
```

**Request Body:**

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | string | Yes | - | Unique function name (alphanumeric, hyphens, underscores) |
| `description` | string | No | null | Function description |
| `code` | string | Yes | - | TypeScript/JavaScript code with handler function |
| `enabled` | boolean | No | false | Whether function is enabled for invocation |
| `timeout_seconds` | integer | No | 30 | Maximum execution time (1-300) |
| `memory_limit_mb` | integer | No | 128 | Maximum memory usage (64-1024) |
| `allow_net` | boolean | No | true | Allow network access |
| `allow_env` | boolean | No | true | Allow environment variable access |
| `allow_read` | boolean | No | false | Allow filesystem read |
| `allow_write` | boolean | No | false | Allow filesystem write |
| `cron_schedule` | string | No | null | Cron expression for scheduled execution (future feature) |

**Example Request:**

```json
{
  "name": "send-welcome-email",
  "description": "Sends welcome email to new users",
  "code": "async function handler(req) { const data = JSON.parse(req.body); return { status: 200, body: JSON.stringify({ sent: true }) }; }",
  "enabled": true,
  "timeout_seconds": 30,
  "allow_net": true,
  "allow_env": true
}
```

**Response (201 Created):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "send-welcome-email",
  "description": "Sends welcome email to new users",
  "code": "async function handler(req) { ... }",
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

**Errors:**

- `400 Bad Request` - Invalid request body or missing required fields
- `401 Unauthorized` - Missing or invalid authentication token
- `409 Conflict` - Function with this name already exists
- `500 Internal Server Error` - Server error

---

### List Functions

Get a list of all edge functions.

**Endpoint:** `GET /api/v1/functions`

**Headers:**
```
Authorization: Bearer YOUR_ACCESS_TOKEN
```

**Response (200 OK):**

```json
[
  {
    "id": "uuid",
    "name": "send-welcome-email",
    "description": "Sends welcome email to new users",
    "version": 1,
    "enabled": true,
    "timeout_seconds": 30,
    "memory_limit_mb": 128,
    "created_at": "2024-10-26T10:00:00Z",
    "updated_at": "2024-10-26T10:00:00Z"
  },
  {
    "id": "uuid",
    "name": "process-webhook",
    "description": "Processes incoming webhooks",
    "version": 3,
    "enabled": true,
    "timeout_seconds": 60,
    "memory_limit_mb": 256,
    "created_at": "2024-10-25T08:00:00Z",
    "updated_at": "2024-10-26T09:30:00Z"
  }
]
```

**Note:** This endpoint returns function metadata only, not the code. Use GET `/api/v1/functions/:name` to retrieve the full function including code.

---

### Get Function

Get details of a specific function including its code.

**Endpoint:** `GET /api/v1/functions/:name`

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `name` | string | Function name |

**Headers:**
```
Authorization: Bearer YOUR_ACCESS_TOKEN
```

**Response (200 OK):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "send-welcome-email",
  "description": "Sends welcome email to new users",
  "code": "async function handler(req) { const data = JSON.parse(req.body); return { status: 200, body: JSON.stringify({ sent: true }) }; }",
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

**Errors:**

- `401 Unauthorized` - Missing or invalid authentication token
- `404 Not Found` - Function not found

---

### Update Function

Update an existing function. Version is automatically incremented.

**Endpoint:** `PUT /api/v1/functions/:name`

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `name` | string | Function name |

**Headers:**
```
Authorization: Bearer YOUR_ACCESS_TOKEN
Content-Type: application/json
```

**Request Body:**

All fields are optional. Only include fields you want to update.

| Field | Type | Description |
|-------|------|-------------|
| `description` | string | Function description |
| `code` | string | Updated function code |
| `enabled` | boolean | Enable/disable function |
| `timeout_seconds` | integer | Maximum execution time |
| `memory_limit_mb` | integer | Maximum memory usage |
| `allow_net` | boolean | Allow network access |
| `allow_env` | boolean | Allow environment variables |
| `allow_read` | boolean | Allow filesystem read |
| `allow_write` | boolean | Allow filesystem write |
| `cron_schedule` | string | Cron schedule |

**Example Request:**

```json
{
  "code": "async function handler(req) { /* updated code */ }",
  "enabled": true,
  "timeout_seconds": 60
}
```

**Response (200 OK):**

Returns the updated function with incremented version:

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "send-welcome-email",
  "description": "Sends welcome email to new users",
  "code": "async function handler(req) { /* updated code */ }",
  "version": 2,
  "enabled": true,
  "timeout_seconds": 60,
  "memory_limit_mb": 128,
  "allow_net": true,
  "allow_env": true,
  "allow_read": false,
  "allow_write": false,
  "cron_schedule": null,
  "created_at": "2024-10-26T10:00:00Z",
  "updated_at": "2024-10-26T10:15:00Z",
  "created_by": "user-uuid"
}
```

**Errors:**

- `400 Bad Request` - Invalid request body
- `401 Unauthorized` - Missing or invalid authentication token
- `404 Not Found` - Function not found
- `500 Internal Server Error` - Server error

---

### Delete Function

Delete an edge function permanently.

**Endpoint:** `DELETE /api/v1/functions/:name`

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `name` | string | Function name |

**Headers:**
```
Authorization: Bearer YOUR_ACCESS_TOKEN
```

**Response (204 No Content)**

No response body. The function and all its execution history are permanently deleted.

**Errors:**

- `401 Unauthorized` - Missing or invalid authentication token
- `404 Not Found` - Function not found

---

## Invocation Endpoints

### Invoke Function

Execute an edge function with a payload.

**Endpoint:** `POST /api/v1/functions/:name/invoke`

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `name` | string | Function name |

**Headers:**
```
Content-Type: application/json
Authorization: Bearer YOUR_ACCESS_TOKEN (optional, for authenticated functions)
```

**Request Body:**

Any valid JSON. This will be passed to your function's `handler` as `request.body`.

**Example Request:**

```json
{
  "email": "user@example.com",
  "name": "Alice",
  "action": "send_welcome"
}
```

**Response:**

The response depends entirely on what your function returns. The HTTP status code and body are determined by your function's return value.

**Example Response (200 OK):**

```json
{
  "sent": true,
  "message_id": "msg_123456",
  "timestamp": "2024-10-26T10:00:00Z"
}
```

**Errors:**

- `404 Not Found` - Function not found or disabled
- `408 Request Timeout` - Function exceeded timeout limit
- `500 Internal Server Error` - Function threw an uncaught error
- `503 Service Unavailable` - Function runtime unavailable

**Error Response Example:**

```json
{
  "error": "Function execution failed",
  "message": "ReferenceError: someVariable is not defined",
  "function": "send-welcome-email",
  "execution_id": "exec-uuid"
}
```

---

### Get Execution History

Retrieve execution logs for a function.

**Endpoint:** `GET /api/v1/functions/:name/executions`

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `name` | string | Function name |

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | integer | 50 | Maximum number of executions to return (1-1000) |
| `offset` | integer | 0 | Number of executions to skip for pagination |
| `status` | string | - | Filter by status: `success`, `error`, `timeout` |

**Headers:**
```
Authorization: Bearer YOUR_ACCESS_TOKEN
```

**Example Request:**

```
GET /api/v1/functions/send-welcome-email/executions?limit=10&status=error
```

**Response (200 OK):**

```json
[
  {
    "id": "exec-uuid-1",
    "function_id": "function-uuid",
    "trigger_type": "http",
    "status": "success",
    "status_code": 200,
    "duration_ms": 125,
    "result": "{\"sent\":true,\"message_id\":\"msg_123456\"}",
    "logs": "console.log: Sending email to user@example.com\nconsole.log: Email sent successfully",
    "error_message": null,
    "executed_at": "2024-10-26T10:00:00Z",
    "completed_at": "2024-10-26T10:00:01Z"
  },
  {
    "id": "exec-uuid-2",
    "function_id": "function-uuid",
    "trigger_type": "http",
    "status": "error",
    "status_code": 500,
    "duration_ms": 50,
    "result": null,
    "logs": "console.error: Failed to connect to email service",
    "error_message": "Error: ECONNREFUSED",
    "executed_at": "2024-10-26T09:55:00Z",
    "completed_at": "2024-10-26T09:55:01Z"
  }
]
```

**Execution Status:**

- `success` - Function completed successfully
- `error` - Function threw an error
- `timeout` - Function exceeded timeout limit

**Errors:**

- `401 Unauthorized` - Missing or invalid authentication token
- `404 Not Found` - Function not found

---

## Request/Response Schema

### Function Request Object

The object passed to your `handler` function:

```typescript
interface FunctionRequest {
  method: string;           // HTTP method (GET, POST, PUT, DELETE, etc.)
  url: string;             // Full request URL
  headers: Record<string, string>; // Request headers as key-value pairs
  body: string;            // Raw request body as string
  user_id?: string;        // Authenticated user ID (if auth token provided)
}
```

### Function Response Object

The object your `handler` function should return:

```typescript
interface FunctionResponse {
  status: number;          // HTTP status code (200, 400, 500, etc.)
  headers?: Record<string, string>; // Response headers (optional)
  body: string;            // Response body as string
}
```

---

## Rate Limits

| Endpoint | Rate Limit |
|----------|------------|
| Management (Create/Update/Delete) | 60 requests/minute |
| List Functions | 300 requests/minute |
| Invoke Function | 600 requests/minute per function |
| Execution History | 300 requests/minute |

Rate limits are per user/token. Exceeding limits returns:

```json
{
  "error": "Rate limit exceeded",
  "retry_after": 30
}
```

**Status Code:** 429 Too Many Requests

---

## WebSocket Support (Future)

Real-time function invocation via WebSocket is planned for a future release:

```javascript
const ws = new WebSocket('wss://your-instance.com/api/v1/functions/stream');

ws.send(JSON.stringify({
  function: 'my-function',
  payload: { key: 'value' }
}));

ws.onmessage = (event) => {
  console.log('Function result:', event.data);
};
```

---

## Pagination

For endpoints that return lists (executions), use `limit` and `offset`:

```bash
# Get first 50 executions
GET /api/v1/functions/my-function/executions?limit=50&offset=0

# Get next 50 executions
GET /api/v1/functions/my-function/executions?limit=50&offset=50
```

---

## Filtering

### Execution History Filters

- `status` - Filter by execution status (`success`, `error`, `timeout`)
- `trigger_type` - Filter by trigger type (`http`, `cron`, `trigger`)

**Example:**

```bash
GET /api/v1/functions/my-function/executions?status=error&limit=20
```

---

## Next Steps

- [Edge Functions Guide](/guides/edge-functions) - Learn how to write functions
- [Authentication](/guides/authentication) - Secure your functions
- [Database Access](/guides/database) - Query data from functions
