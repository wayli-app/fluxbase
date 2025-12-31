---
title: "MCP Resources Reference"
description: "Complete reference of MCP resources for schema, functions, storage, and RPC"
---

MCP resources provide read-only access to Fluxbase metadata and configuration.

## Schema Resources

### Database Schema Overview

**URI:** `fluxbase://schema/tables`

**Scope Required:** `read:tables`

Returns a complete overview of all database tables and their schemas.

```json
{
  "method": "resources/read",
  "params": {
    "uri": "fluxbase://schema/tables"
  }
}
```

**Response:**

```json
{
  "contents": [
    {
      "uri": "fluxbase://schema/tables",
      "mimeType": "application/json",
      "text": "{\"tables\":[{\"schema\":\"public\",\"name\":\"users\",\"columns\":[...]}]}"
    }
  ]
}
```

**Notes:**
- Internal schemas (auth, storage, functions, jobs) are hidden from non-admin users
- Admin users see all schemas

### Table Details

**URI Template:** `fluxbase://schema/tables/{schema}/{table}`

**Scope Required:** `read:tables`

Returns detailed information about a specific table.

```json
{
  "method": "resources/read",
  "params": {
    "uri": "fluxbase://schema/tables/public/users"
  }
}
```

**Response includes:**
- Column names, types, nullability, and defaults
- Primary key information
- Foreign key relationships
- Index definitions
- Column positions

## Function Resources

### Edge Functions List

**URI:** `fluxbase://functions`

**Scope Required:** `execute:functions`

Returns a list of available edge functions.

```json
{
  "method": "resources/read",
  "params": {
    "uri": "fluxbase://functions"
  }
}
```

**Response includes:**
- Function name and namespace
- Enabled status
- Description
- Rate limits
- Concurrency settings

## Storage Resources

### Storage Buckets

**URI:** `fluxbase://storage/buckets`

**Scope Required:** `read:storage`

Returns a list of available storage buckets.

```json
{
  "method": "resources/read",
  "params": {
    "uri": "fluxbase://storage/buckets"
  }
}
```

**Response includes:**
- Bucket name
- Public/private status
- File size limits
- Allowed MIME types

## RPC Resources

### RPC Procedures

**URI:** `fluxbase://rpc`

**Scope Required:** `execute:rpc`

Returns a list of available RPC procedures.

```json
{
  "method": "resources/read",
  "params": {
    "uri": "fluxbase://rpc"
  }
}
```

**Response includes:**
- Procedure name and namespace
- Enabled status
- Input/output schemas
- Maximum execution time

## Listing Resources

To get a list of all available resources:

```json
{
  "method": "resources/list",
  "params": {}
}
```

**Response:**

```json
{
  "resources": [
    {
      "uri": "fluxbase://schema/tables",
      "name": "Database Schema",
      "description": "Complete database schema information",
      "mimeType": "application/json"
    },
    {
      "uri": "fluxbase://functions",
      "name": "Edge Functions",
      "description": "Available edge functions",
      "mimeType": "application/json"
    }
  ]
}
```

## Resource Templates

Some resources use URI templates with placeholders:

```json
{
  "method": "resources/templates",
  "params": {}
}
```

**Response:**

```json
{
  "resourceTemplates": [
    {
      "uriTemplate": "fluxbase://schema/tables/{schema}/{table}",
      "name": "Table Details",
      "description": "Detailed information about a specific table"
    }
  ]
}
```

## Access Control

Resources respect the same scope-based access control as tools:

| Resource | Required Scope |
|----------|---------------|
| `fluxbase://schema/tables` | `read:tables` |
| `fluxbase://schema/tables/{schema}/{table}` | `read:tables` |
| `fluxbase://functions` | `execute:functions` |
| `fluxbase://storage/buckets` | `read:storage` |
| `fluxbase://rpc` | `execute:rpc` |

## Error Responses

Resource read errors return standard JSON-RPC errors:

```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32002,
    "message": "Resource not found: fluxbase://invalid"
  },
  "id": 1
}
```
