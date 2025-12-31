---
title: "MCP Tools Reference"
description: "Complete reference of MCP tools for database, storage, functions, and jobs"
---

The MCP server provides 12 tools for interacting with Fluxbase.

## Database Tools

### query_table

Query a database table with filters, ordering, and pagination.

**Scope Required:** `read:tables`

```json
{
  "name": "query_table",
  "arguments": {
    "table": "public.users",
    "select": "id,email,created_at",
    "filter": {
      "is_active": "eq.true",
      "age": "gt.18"
    },
    "order": "created_at.desc",
    "limit": 100,
    "offset": 0
  }
}
```

**Filter Operators:**
- `eq` - Equals
- `neq` - Not equals
- `gt` - Greater than
- `gte` - Greater than or equal
- `lt` - Less than
- `lte` - Less than or equal
- `like` - Pattern match (case-sensitive)
- `ilike` - Pattern match (case-insensitive)
- `is` - IS NULL/NOT NULL
- `in` - In array

### insert_record

Insert a new record into a table.

**Scope Required:** `write:tables`

```json
{
  "name": "insert_record",
  "arguments": {
    "table": "public.products",
    "data": {
      "name": "Widget",
      "price": 29.99,
      "category": "electronics"
    },
    "returning": "id,created_at"
  }
}
```

### update_record

Update records matching a filter.

**Scope Required:** `write:tables`

```json
{
  "name": "update_record",
  "arguments": {
    "table": "public.products",
    "data": {
      "price": 24.99
    },
    "filter": {
      "id": "eq.123"
    },
    "returning": "id,price,updated_at"
  }
}
```

**Note:** The `filter` parameter is required to prevent accidental bulk updates.

### delete_record

Delete records matching a filter.

**Scope Required:** `write:tables`

```json
{
  "name": "delete_record",
  "arguments": {
    "table": "public.products",
    "filter": {
      "id": "eq.123"
    },
    "returning": "id"
  }
}
```

**Note:** The `filter` parameter is required to prevent accidental bulk deletes.

## Storage Tools

### list_objects

List objects in a storage bucket.

**Scope Required:** `read:storage`

```json
{
  "name": "list_objects",
  "arguments": {
    "bucket": "uploads",
    "prefix": "images/",
    "limit": 100,
    "start_after": "images/file099.jpg"
  }
}
```

### download_object

Download a file from storage.

**Scope Required:** `read:storage`

```json
{
  "name": "download_object",
  "arguments": {
    "bucket": "uploads",
    "key": "documents/report.pdf"
  }
}
```

**Notes:**
- Text files are returned as-is
- Binary files are returned as base64
- Maximum file size: 10MB

### upload_object

Upload a file to storage.

**Scope Required:** `write:storage`

```json
{
  "name": "upload_object",
  "arguments": {
    "bucket": "uploads",
    "key": "documents/report.txt",
    "content": "Report content here...",
    "content_type": "text/plain",
    "encoding": "text"
  }
}
```

**Encoding Options:**
- `text` - Plain text content
- `base64` - Base64-encoded binary content

### delete_object

Delete a file from storage.

**Scope Required:** `write:storage`

```json
{
  "name": "delete_object",
  "arguments": {
    "bucket": "uploads",
    "key": "documents/old-report.pdf"
  }
}
```

## Function Tools

### invoke_function

Invoke an edge function.

**Scope Required:** `execute:functions`

```json
{
  "name": "invoke_function",
  "arguments": {
    "name": "send-email",
    "namespace": "default",
    "method": "POST",
    "body": {
      "to": "user@example.com",
      "subject": "Hello",
      "body": "Message content"
    },
    "headers": {
      "X-Custom-Header": "value"
    }
  }
}
```

**Response:**

```json
{
  "status": 200,
  "headers": {"content-type": "application/json"},
  "body": {"success": true},
  "execution_id": "exec-123"
}
```

### invoke_rpc

Execute an RPC procedure.

**Scope Required:** `execute:rpc`

```json
{
  "name": "invoke_rpc",
  "arguments": {
    "name": "calculate_total",
    "namespace": "default",
    "params": {
      "order_id": "ord-123"
    }
  }
}
```

**Response:**

```json
{
  "data": [{"total": 149.99}],
  "rows_returned": 1,
  "duration_ms": 15
}
```

## Job Tools

### submit_job

Submit a background job.

**Scope Required:** `execute:jobs`

```json
{
  "name": "submit_job",
  "arguments": {
    "job_name": "process-report",
    "namespace": "default",
    "payload": {
      "report_id": "rpt-123",
      "format": "pdf"
    },
    "priority": 5,
    "scheduled_at": "2024-01-15T10:00:00Z"
  }
}
```

**Response:**

```json
{
  "job_id": "job-456",
  "created_at": "2024-01-15T09:00:00Z"
}
```

### get_job_status

Get the status of a background job.

**Scope Required:** `execute:jobs`

```json
{
  "name": "get_job_status",
  "arguments": {
    "job_id": "job-456"
  }
}
```

**Response:**

```json
{
  "status": "completed",
  "attempts": 1,
  "started_at": "2024-01-15T10:00:00Z",
  "completed_at": "2024-01-15T10:00:15Z",
  "result": {"output": "Report generated"},
  "progress": 100
}
```

## Vector Search Tools

### search_vectors

Perform semantic similarity search.

**Scope Required:** `read:vectors`

```json
{
  "name": "search_vectors",
  "arguments": {
    "query": "How do I reset my password?",
    "chatbot_id": "cb-123",
    "knowledge_bases": ["kb-docs", "kb-faq"],
    "limit": 5,
    "threshold": 0.7,
    "tags": ["support"]
  }
}
```

**Response:**

```json
{
  "results": [
    {
      "content": "To reset your password, click...",
      "similarity": 0.92,
      "metadata": {"source": "faq.md"}
    }
  ]
}
```

## Error Handling

All tools return errors in a consistent format:

```json
{
  "content": [
    {
      "type": "text",
      "text": "Error message here"
    }
  ],
  "isError": true
}
```

Common error codes:
- `-32002` - Resource not found
- `-32003` - Tool not found
- `-32004` - Tool execution error
- `-32005` - Unauthorized
- `-32006` - Forbidden
- `-32007` - Rate limited
