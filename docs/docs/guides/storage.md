# File Storage

Fluxbase provides a flexible file storage system that supports both local filesystem and S3-compatible storage (including MinIO, AWS S3, Wasabi, DigitalOcean Spaces, and more).

## Overview

The storage service provides:

- **Multiple Storage Providers**: Local filesystem or S3-compatible storage
- **Bucket Management**: Create, delete, and list buckets
- **File Operations**: Upload, download, delete, list files
- **Metadata Support**: Attach custom metadata to files
- **Signed URLs**: Generate temporary access URLs (S3 only)
- **Range Requests**: Support for partial content downloads
- **Copy and Move**: Efficient file operations

## Configuration

Storage is configured via environment variables or the configuration file:

```yaml
storage:
  provider: "local" # or "s3"
  local_path: "./storage"
  max_upload_size: 10485760 # 10MB in bytes

  # S3 Configuration (when provider: "s3")
  s3_endpoint: "s3.amazonaws.com" # or MinIO endpoint
  s3_access_key: "your-access-key"
  s3_secret_key: "your-secret-key"
  s3_region: "us-east-1"
  s3_bucket: "default-bucket"
```

### Environment Variables

```bash
# Storage Provider
FLUXBASE_STORAGE_PROVIDER=local  # or s3

# Local Storage
FLUXBASE_STORAGE_LOCAL_PATH=./storage
FLUXBASE_STORAGE_MAX_UPLOAD_SIZE=10485760

# S3 Storage
FLUXBASE_STORAGE_S3_ENDPOINT=s3.amazonaws.com
FLUXBASE_STORAGE_S3_ACCESS_KEY=your-access-key
FLUXBASE_STORAGE_S3_SECRET_KEY=your-secret-key
FLUXBASE_STORAGE_S3_REGION=us-east-1
FLUXBASE_STORAGE_S3_BUCKET=default-bucket
```

## Local vs S3 Storage

### Local Filesystem Storage

**Pros:**

- Simple setup, no external dependencies
- Fast for development
- No cloud costs
- Works offline

**Cons:**

- Not scalable across multiple servers
- No built-in CDN
- Manual backup required
- Limited to single server

**Use Cases:**

- Development and testing
- Single-server deployments
- Small-scale applications
- Internal tools

### S3-Compatible Storage

**Pros:**

- Highly scalable
- Distributed/redundant storage
- CDN integration available
- Automatic backups
- Works with multiple servers

**Cons:**

- Requires external service
- Network latency
- Cloud costs
- More complex setup

**Use Cases:**

- Production deployments
- Multi-server architectures
- Large file storage
- Global distribution

## API Reference

### Bucket Management

#### Create Bucket

```http
POST /api/v1/storage/buckets/:bucket
```

**Example:**

```bash
curl -X POST http://localhost:8080/api/v1/storage/buckets/my-bucket
```

**Response:**

```json
{
  "bucket": "my-bucket",
  "message": "bucket created successfully"
}
```

#### List Buckets

```http
GET /api/v1/storage/buckets
```

**Example:**

```bash
curl http://localhost:8080/api/v1/storage/buckets
```

**Response:**

```json
{
  "buckets": ["bucket1", "bucket2", "bucket3"]
}
```

#### Delete Bucket

```http
DELETE /api/v1/storage/buckets/:bucket
```

**Example:**

```bash
curl -X DELETE http://localhost:8080/api/v1/storage/buckets/my-bucket
```

**Note:** Bucket must be empty before deletion.

### File Operations

#### Upload File

```http
POST /api/v1/storage/:bucket/:key
Content-Type: multipart/form-data
```

**Example:**

```bash
curl -X POST http://localhost:8080/api/v1/storage/my-bucket/documents/report.pdf \
  -F "file=@/path/to/report.pdf" \
  -F "x-meta-author=John Doe" \
  -F "x-meta-department=Engineering"
```

**Response:**

```json
{
  "key": "documents/report.pdf",
  "bucket": "my-bucket",
  "size": 1024576,
  "content_type": "application/pdf",
  "etag": "abc123...",
  "last_modified": "2025-10-26T12:00:00Z",
  "metadata": {
    "author": "John Doe",
    "department": "Engineering"
  }
}
```

#### Download File

```http
GET /api/v1/storage/:bucket/:key
```

**Example:**

```bash
# Direct download
curl http://localhost:8080/api/v1/storage/my-bucket/documents/report.pdf \
  -o report.pdf

# Download with filename
curl http://localhost:8080/api/v1/storage/my-bucket/documents/report.pdf?download=true \
  -o report.pdf

# Range request (partial download)
curl http://localhost:8080/api/v1/storage/my-bucket/documents/report.pdf \
  -H "Range: bytes=0-1023"
```

#### Delete File

```http
DELETE /api/v1/storage/:bucket/:key
```

**Example:**

```bash
curl -X DELETE http://localhost:8080/api/v1/storage/my-bucket/documents/report.pdf
```

#### Get File Metadata

```http
HEAD /api/v1/storage/:bucket/:key
```

**Example:**

```bash
curl -I http://localhost:8080/api/v1/storage/my-bucket/documents/report.pdf
```

**Response:**

```json
{
  "key": "documents/report.pdf",
  "bucket": "my-bucket",
  "size": 1024576,
  "content_type": "application/pdf",
  "etag": "abc123...",
  "last_modified": "2025-10-26T12:00:00Z",
  "metadata": {
    "author": "John Doe"
  }
}
```

#### List Files

```http
GET /api/v1/storage/:bucket?prefix=&delimiter=&limit=1000
```

**Query Parameters:**

- `prefix`: Filter files by prefix (e.g., "documents/")
- `delimiter`: Directory delimiter (e.g., "/")
- `limit`: Maximum number of files to return (default: 1000)

**Example:**

```bash
# List all files in bucket
curl http://localhost:8080/api/v1/storage/my-bucket

# List files with prefix
curl http://localhost:8080/api/v1/storage/my-bucket?prefix=documents/

# List with limit
curl http://localhost:8080/api/v1/storage/my-bucket?limit=10
```

**Response:**

```json
{
  "bucket": "my-bucket",
  "objects": [
    {
      "key": "documents/report.pdf",
      "size": 1024576,
      "content_type": "application/pdf",
      "last_modified": "2025-10-26T12:00:00Z",
      "etag": "abc123..."
    }
  ],
  "prefixes": ["documents/", "images/"],
  "truncated": false
}
```

### Advanced Features

#### Generate Signed URL (S3 Only)

Generate a temporary URL for secure file access without authentication.

```http
POST /api/v1/storage/:bucket/:key/signed-url
Content-Type: application/json
```

**Request Body:**

```json
{
  "method": "GET",
  "expires_in": 3600
}
```

**Example:**

```bash
curl -X POST http://localhost:8080/api/v1/storage/my-bucket/documents/report.pdf/signed-url \
  -H "Content-Type: application/json" \
  -d '{"method":"GET","expires_in":3600}'
```

**Response:**

```json
{
  "url": "https://s3.amazonaws.com/my-bucket/documents/report.pdf?X-Amz-Algorithm=...",
  "expires_at": "2025-10-26T13:00:00Z"
}
```

**Note:** This feature is only available with S3-compatible storage. Local storage will return `501 Not Implemented`.

## JavaScript/TypeScript Client Examples

### Upload File

```typescript
async function uploadFile(bucket: string, key: string, file: File) {
  const formData = new FormData();
  formData.append("file", file);
  formData.append("x-meta-author", "John Doe");

  const response = await fetch(`/api/v1/storage/${bucket}/${key}`, {
    method: "POST",
    body: formData,
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  return await response.json();
}
```

### Download File

```typescript
async function downloadFile(bucket: string, key: string) {
  const response = await fetch(
    `/api/v1/storage/${bucket}/${key}?download=true`,
    {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    },
  );

  const blob = await response.blob();
  const url = window.URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = key.split("/").pop() || "download";
  a.click();
}
```

### List Files

```typescript
async function listFiles(bucket: string, prefix?: string) {
  const params = new URLSearchParams();
  if (prefix) params.append("prefix", prefix);

  const response = await fetch(`/api/v1/storage/${bucket}?${params}`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  return await response.json();
}
```

## React Example Component

```tsx
import { useState } from "react";

function FileUploader() {
  const [file, setFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);

  const handleUpload = async () => {
    if (!file) return;

    setUploading(true);
    try {
      const formData = new FormData();
      formData.append("file", file);

      const response = await fetch("/api/v1/storage/uploads/" + file.name, {
        method: "POST",
        body: formData,
        headers: {
          Authorization: `Bearer ${localStorage.getItem("token")}`,
        },
      });

      if (response.ok) {
        const data = await response.json();
        console.log("File uploaded:", data);
        alert("File uploaded successfully!");
      } else {
        const error = await response.json();
        alert(`Upload failed: ${error.error}`);
      }
    } catch (error) {
      console.error("Upload error:", error);
      alert("Upload failed");
    } finally {
      setUploading(false);
    }
  };

  return (
    <div>
      <input
        type="file"
        onChange={(e) => setFile(e.target.files?.[0] || null)}
      />
      <button onClick={handleUpload} disabled={!file || uploading}>
        {uploading ? "Uploading..." : "Upload"}
      </button>
    </div>
  );
}
```

## MinIO Setup (For Development)

### Using Docker

```bash
# Start MinIO server
docker run -p 9000:9000 -p 9001:9001 \
  -e "MINIO_ROOT_USER=minioadmin" \
  -e "MINIO_ROOT_PASSWORD=minioadmin" \
  minio/minio server /data --console-address ":9001"

# Access MinIO Console at http://localhost:9001
# Use credentials: minioadmin / minioadmin
```

### Configure Fluxbase to Use MinIO

```yaml
storage:
  provider: "s3"
  s3_endpoint: "http://localhost:9000"
  s3_access_key: "minioadmin"
  s3_secret_key: "minioadmin"
  s3_region: "us-east-1"
```

## Best Practices

### File Organization

Organize files using a hierarchical structure:

```
user-uploads/
  ├── avatars/
  │   ├── user-123.jpg
  │   └── user-456.jpg
  ├── documents/
  │   ├── reports/
  │   │   ├── 2025/
  │   │   │   ├── q1-report.pdf
  │   │   │   └── q2-report.pdf
  │   └── contracts/
  └── images/
      └── products/
```

### Metadata Usage

Use metadata to store additional file information:

```bash
curl -X POST http://localhost:8080/api/v1/storage/files/document.pdf \
  -F "file=@document.pdf" \
  -F "x-meta-author=John Doe" \
  -F "x-meta-department=Sales" \
  -F "x-meta-project=Q4-2025" \
  -F "x-meta-confidential=true"
```

### Security

1. **Authentication**: Always require authentication for sensitive files
2. **Access Control**: Implement per-bucket or per-file permissions
3. **Signed URLs**: Use signed URLs for temporary access instead of making files public
4. **File Validation**: Validate file types and sizes on upload
5. **Virus Scanning**: Consider integrating virus scanning for user uploads

### Performance

1. **CDN Integration**: Use a CDN for frequently accessed files
2. **Caching**: Implement proper cache headers for static content
3. **Range Requests**: Support range requests for large files
4. **Compression**: Compress files before storage when appropriate
5. **Batch Operations**: Use batch operations for multiple files

## Error Handling

### Common Error Codes

- `400 Bad Request`: Missing required parameters
- `401 Unauthorized`: Invalid or missing authentication
- `404 Not Found`: Bucket or file doesn't exist
- `409 Conflict`: Bucket already exists or bucket not empty
- `413 Payload Too Large`: File exceeds max upload size
- `500 Internal Server Error`: Server-side storage error
- `501 Not Implemented`: Feature not supported (e.g., signed URLs on local storage)

### Example Error Response

```json
{
  "error": "file exceeds maximum upload size of 10MB"
}
```

## Testing

### Unit Tests

The storage service includes comprehensive unit tests:

```bash
# Run storage unit tests
go test ./internal/storage -v

# Run API integration tests
go test ./internal/api -run TestStorageAPI -v
```

### Manual Testing

```bash
# Create bucket
curl -X POST http://localhost:8080/api/v1/storage/buckets/test

# Upload file
curl -X POST http://localhost:8080/api/v1/storage/test/sample.txt \
  -F "file=@sample.txt"

# List files
curl http://localhost:8080/api/v1/storage/test

# Download file
curl http://localhost:8080/api/v1/storage/test/sample.txt

# Delete file
curl -X DELETE http://localhost:8080/api/v1/storage/test/sample.txt

# Delete bucket
curl -X DELETE http://localhost:8080/api/v1/storage/buckets/test
```

## Troubleshooting

### Local Storage Issues

**Problem**: Permission denied errors

**Solution**: Ensure the storage directory has correct permissions:

```bash
chmod 755 ./storage
```

**Problem**: Disk space errors

**Solution**: Check available disk space:

```bash
df -h
```

### S3 Storage Issues

**Problem**: Connection refused to S3 endpoint

**Solution**: Verify endpoint configuration and network connectivity:

```bash
curl -v http://your-s3-endpoint:9000/
```

**Problem**: Access denied errors

**Solution**: Verify access key and secret key are correct, and have proper permissions.

**Problem**: Bucket doesn't exist

**Solution**: Create the bucket first using the API or S3 console.

## Migration

### Migrating from Local to S3

1. Update configuration to use S3
2. Upload existing files to S3 bucket
3. Update database references if storing file paths
4. Test thoroughly before removing local files

### Example Migration Script

```bash
#!/bin/bash
# Migrate files from local storage to S3

BUCKET="my-bucket"
LOCAL_PATH="./storage"

# Upload all files
find "$LOCAL_PATH" -type f | while read file; do
  # Remove local path prefix
  key="${file#$LOCAL_PATH/}"

  # Upload to S3
  aws s3 cp "$file" "s3://$BUCKET/$key"
done
```

## Related Documentation

- [Authentication](./authentication.md) - Securing storage endpoints
- [API Reference](../api/storage.md) - Complete API documentation
