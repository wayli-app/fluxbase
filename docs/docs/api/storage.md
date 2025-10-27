---
sidebar_position: 3
---

# Storage API Reference

HTTP API reference for file storage operations in Fluxbase.

## Base URL

```
http://localhost:8080/api/v1/storage
```

## Authentication

```http
Authorization: Bearer YOUR_ACCESS_TOKEN
```

## Bucket Management

### List Buckets

```http
GET /buckets
```

**Response:**

```json
{
  "buckets": [
    {
      "name": "avatars",
      "public": false,
      "created_at": "2024-10-27T10:00:00Z"
    }
  ]
}
```

### Create Bucket

```http
POST /buckets
Content-Type: application/json

{
  "name": "bucket-name",
  "public": false
}
```

### Delete Bucket

```http
DELETE /buckets/{bucket_name}
```

## File Operations

### Upload File

```http
POST /buckets/{bucket}/files
Content-Type: multipart/form-data

file: (binary)
path: path/to/file.jpg
```

**Example (curl):**

```bash
curl -X POST http://localhost:8080/api/v1/storage/buckets/avatars/files \
  -H "Authorization: Bearer TOKEN" \
  -F "file=@avatar.jpg" \
  -F "path=users/123/avatar.jpg"
```

**Response:**

```json
{
  "id": "abc123",
  "path": "users/123/avatar.jpg",
  "bucket": "avatars",
  "size": 102400,
  "content_type": "image/jpeg",
  "url": "/storage/buckets/avatars/files/users/123/avatar.jpg",
  "created_at": "2024-10-27T10:00:00Z"
}
```

### Download File

```http
GET /buckets/{bucket}/files/{path}
```

**Example:**

```bash
curl http://localhost:8080/api/v1/storage/buckets/avatars/files/users/123/avatar.jpg \
  -H "Authorization: Bearer TOKEN" \
  -o avatar.jpg
```

### List Files

```http
GET /buckets/{bucket}/files?prefix={path}&limit={n}
```

**Query Parameters:**

| Parameter | Description | Example |
|-----------|-------------|---------|
| `prefix` | Filter by path prefix | `users/123/` |
| `limit` | Max files to return | `100` |
| `offset` | Skip files | `20` |

**Response:**

```json
{
  "files": [
    {
      "id": "abc123",
      "path": "users/123/avatar.jpg",
      "size": 102400,
      "content_type": "image/jpeg",
      "metadata": {},
      "created_at": "2024-10-27T10:00:00Z"
    }
  ]
}
```

### Delete File

```http
DELETE /buckets/{bucket}/files/{path}
```

**Example:**

```bash
curl -X DELETE \
  "http://localhost:8080/api/v1/storage/buckets/avatars/files/users/123/avatar.jpg" \
  -H "Authorization: Bearer TOKEN"
```

### Copy File

```http
POST /buckets/{bucket}/files/{path}/copy
Content-Type: application/json

{
  "destination_bucket": "backups",
  "destination_path": "archive/avatar.jpg"
}
```

### Move File

```http
POST /buckets/{bucket}/files/{path}/move
Content-Type: application/json

{
  "destination_bucket": "backups",
  "destination_path": "archive/avatar.jpg"
}
```

## Signed URLs (S3 Only)

### Generate Signed URL

```http
POST /buckets/{bucket}/files/{path}/signed-url
Content-Type: application/json

{
  "expiry_seconds": 3600
}
```

**Response:**

```json
{
  "url": "https://s3.amazonaws.com/bucket/path?X-Amz-Signature=...",
  "expires_at": "2024-10-27T11:00:00Z"
}
```

**Example:**

```bash
curl -X POST \
  "http://localhost:8080/api/v1/storage/buckets/avatars/files/user.jpg/signed-url" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer TOKEN" \
  -d '{"expiry_seconds": 3600}'
```

## Metadata

### Get File Metadata

```http
HEAD /buckets/{bucket}/files/{path}
```

**Response Headers:**

```http
Content-Length: 102400
Content-Type: image/jpeg
Last-Modified: Mon, 27 Oct 2024 10:00:00 GMT
ETag: "abc123"
X-Custom-Metadata: value
```

### Update Metadata

```http
PATCH /buckets/{bucket}/files/{path}
Content-Type: application/json

{
  "metadata": {
    "description": "User avatar",
    "uploaded_by": "admin"
  }
}
```

## Response Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 201 | File uploaded |
| 204 | File deleted |
| 400 | Bad request (invalid file) |
| 401 | Unauthorized |
| 403 | Forbidden (access denied) |
| 404 | File not found |
| 409 | Conflict (file exists) |
| 413 | File too large |
| 500 | Server error |

## Error Response

```json
{
  "error": {
    "code": "FILE_TOO_LARGE",
    "message": "File size exceeds maximum allowed size of 10MB"
  }
}
```

## Configuration

See [Configuration Reference](../reference/configuration.md#storage) for storage configuration options.

## SDK Usage

See [Storage SDK Documentation](../guides/typescript-sdk/getting-started.md#storage) for TypeScript SDK examples.

## See Also

- [Storage Guide](../guides/storage.md) - Complete storage documentation
- [Configuration](../reference/configuration.md) - Storage configuration
- [Authentication API](./authentication.md) - Get access tokens
