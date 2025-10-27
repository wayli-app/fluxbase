---
sidebar_position: 1
---

# REST API Reference

Complete HTTP API reference for Fluxbase's PostgREST-compatible REST endpoints.

## Base URL

```
http://localhost:8080/api
```

## Authentication

Include JWT token in the Authorization header:

```http
Authorization: Bearer YOUR_ACCESS_TOKEN
```

Get a token via [Authentication API](./authentication.md).

## Table Endpoints

### List Records

```http
GET /tables/{table_name}
```

**Query Parameters:**

| Parameter | Description | Example |
|-----------|-------------|---------|
| `select` | Columns to return | `id,name,email` |
| `limit` | Max records | `10` |
| `offset` | Skip records | `20` |
| `order` | Sort order | `created_at.desc` |

**Example:**

```bash
curl "http://localhost:8080/api/tables/users?select=id,name&limit=10"
```

### Get Single Record

```http
GET /tables/{table_name}?{filter}&single=true
```

**Example:**

```bash
curl "http://localhost:8080/api/tables/users?id=eq.123&single=true"
```

### Create Record

```http
POST /tables/{table_name}
Content-Type: application/json

{
  "field1": "value1",
  "field2": "value2"
}
```

**Example:**

```bash
curl -X POST http://localhost:8080/api/tables/users \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer TOKEN" \
  -d '{"name": "John", "email": "john@example.com"}'
```

### Update Records

```http
PATCH /tables/{table_name}?{filter}
Content-Type: application/json

{
  "field": "new_value"
}
```

**Example:**

```bash
curl -X PATCH "http://localhost:8080/api/tables/users?id=eq.123" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer TOKEN" \
  -d '{"name": "John Updated"}'
```

### Delete Records

```http
DELETE /tables/{table_name}?{filter}
```

**Example:**

```bash
curl -X DELETE "http://localhost:8080/api/tables/users?id=eq.123" \
  -H "Authorization: Bearer TOKEN"
```

## Filter Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `eq` | Equal | `status=eq.active` |
| `neq` | Not equal | `status=neq.inactive` |
| `gt` | Greater than | `age=gt.18` |
| `gte` | Greater than or equal | `price=gte.100` |
| `lt` | Less than | `stock=lt.10` |
| `lte` | Less than or equal | `price=lte.500` |
| `like` | Pattern match | `name=like.*john*` |
| `ilike` | Case-insensitive pattern | `email=ilike.*@gmail.com` |
| `in` | In list | `status=in.(active,pending)` |
| `is` | Is null/not null | `deleted_at=is.null` |

## Aggregation Endpoints

### Count

```http
GET /aggregate/{table}/count?column={column}
```

**Example:**

```bash
# Count all records
curl "http://localhost:8080/api/aggregate/users/count?column=*"

# Count by group
curl "http://localhost:8080/api/aggregate/orders/count?column=*&group_by=status"
```

### Sum

```http
GET /aggregate/{table}/sum?column={column}
```

**Example:**

```bash
curl "http://localhost:8080/api/aggregate/orders/sum?column=total"
```

### Average

```http
GET /aggregate/{table}/avg?column={column}
```

### Min/Max

```http
GET /aggregate/{table}/min?column={column}
GET /aggregate/{table}/max?column={column}
```

## RPC (Function Calls)

```http
POST /rpc/{function_name}
Content-Type: application/json

{
  "param1": "value1",
  "param2": "value2"
}
```

**Example:**

```bash
curl -X POST http://localhost:8080/api/rpc/calculate_discount \
  -H "Content-Type: application/json" \
  -d '{"product_id": 123, "coupon": "SAVE20"}'
```

## Batch Operations

### Batch Insert

```http
POST /tables/{table}/batch
Content-Type: application/json

[
  {"field": "value1"},
  {"field": "value2"}
]
```

### Batch Update

```http
PATCH /tables/{table}/batch?{filter}
Content-Type: application/json

{"field": "new_value"}
```

### Batch Delete

```http
DELETE /tables/{table}/batch?{filter}
```

## Response Format

### Success Response

```json
{
  "data": [...],
  "count": 100
}
```

### Error Response

```json
{
  "error": {
    "code": "23505",
    "message": "duplicate key value violates unique constraint",
    "details": "Key (email)=(user@example.com) already exists."
  }
}
```

## Status Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 201 | Created |
| 204 | No Content (delete success) |
| 400 | Bad Request |
| 401 | Unauthorized |
| 403 | Forbidden |
| 404 | Not Found |
| 409 | Conflict (duplicate) |
| 500 | Server Error |

## Headers

### Request Headers

```http
Authorization: Bearer {token}
Content-Type: application/json
Prefer: return=representation
```

### Response Headers

```http
Content-Type: application/json
Content-Range: 0-9/100
X-Request-ID: abc123
```

## See Also

- [Authentication API](./authentication.md) - User authentication endpoints
- [Storage API](./storage.md) - File storage endpoints
- [Realtime API](./realtime.md) - WebSocket protocol
- [SDK Documentation](../guides/typescript-sdk/database.md) - TypeScript SDK for REST API
