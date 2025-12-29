---
title: HTTP API Reference
description: Complete HTTP API documentation for Fluxbase REST endpoints
---
The Fluxbase HTTP API provides RESTful endpoints for authentication, storage, and database operations. All endpoints are prefixed with `/api/v1/`.

## Base URL

```
http://localhost:8080/api/v1
```

## Authentication

Most endpoints require authentication via JWT bearer tokens. Include the token in the `Authorization` header:

```bash
curl -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  http://localhost:8080/api/v1/auth/user
```

## API Categories

### Authentication

Endpoints for user registration, login, and session management.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/auth/signup` | Register a new user |
| `POST` | `/auth/signin` | Sign in with email/password |
| `POST` | `/auth/signout` | Sign out current session |
| `POST` | `/auth/refresh` | Refresh access token |
| `GET` | `/auth/user` | Get current user |
| `PATCH` | `/auth/user` | Update current user |
| `POST` | `/auth/magiclink` | Request magic link |
| `GET` | `/auth/magiclink/verify` | Verify magic link token |

### Storage

Endpoints for file storage operations.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/storage/buckets` | List all buckets |
| `POST` | `/storage/buckets/{bucket}` | Create bucket |
| `DELETE` | `/storage/buckets/{bucket}` | Delete bucket |
| `GET` | `/storage/{bucket}` | List files in bucket |
| `POST` | `/storage/{bucket}/{key}` | Upload file |
| `GET` | `/storage/{bucket}/{key}` | Download file |
| `HEAD` | `/storage/{bucket}/{key}` | Get file metadata |
| `DELETE` | `/storage/{bucket}/{key}` | Delete file |
| `POST` | `/storage/{bucket}/{key}/signed-url` | Generate signed URL |

### GraphQL

A full GraphQL API auto-generated from your database schema.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/graphql` | Execute GraphQL queries and mutations |

See the [GraphQL API documentation](/docs/api/http/graphql) for complete details on queries, mutations, filtering, and SDK usage.

### Database Tables

Auto-generated CRUD endpoints for your PostgreSQL tables. Endpoints are pluralized automatically.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/tables/{table}` | List records with filtering |
| `POST` | `/tables/{table}` | Create record(s) |
| `PATCH` | `/tables/{table}` | Batch update records |
| `DELETE` | `/tables/{table}` | Batch delete records |
| `GET` | `/tables/{table}/{id}` | Get record by ID |
| `PUT` | `/tables/{table}/{id}` | Replace record |
| `PATCH` | `/tables/{table}/{id}` | Update record |
| `DELETE` | `/tables/{table}/{id}` | Delete record |

## Query Parameters

Table endpoints support PostgREST-compatible query parameters:

| Parameter | Description | Example |
|-----------|-------------|---------|
| `select` | Columns to return | `?select=id,name,email` |
| `order` | Sort order | `?order=created_at.desc` |
| `limit` | Max results | `?limit=10` |
| `offset` | Pagination offset | `?offset=20` |
| `{column}.{op}` | Column filter | `?name.eq=John&age.gt=18` |

### Filter Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `eq` | Equal | `?status.eq=active` |
| `neq` | Not equal | `?status.neq=deleted` |
| `gt` | Greater than | `?age.gt=18` |
| `gte` | Greater than or equal | `?age.gte=18` |
| `lt` | Less than | `?price.lt=100` |
| `lte` | Less than or equal | `?price.lte=100` |
| `like` | Pattern match | `?name.like=John%` |
| `ilike` | Case-insensitive pattern | `?name.ilike=john%` |
| `in` | In list | `?status.in=(active,pending)` |
| `is` | Is null/not null | `?deleted_at.is.null` |

## OpenAPI Specification

A live OpenAPI 3.0 specification is available at:

```
GET /openapi.json
```

This specification is generated dynamically based on your database schema and includes all available endpoints with their request/response schemas.

## Error Responses

All errors follow a consistent format:

```json
{
  "error": "Error message description"
}
```

Common HTTP status codes:

| Code | Description |
|------|-------------|
| `200` | Success |
| `201` | Created |
| `204` | No content (successful delete) |
| `400` | Bad request |
| `401` | Unauthorized |
| `403` | Forbidden |
| `404` | Not found |
| `409` | Conflict |
| `500` | Internal server error |
