---
title: "File Storage"
description: Configure file storage in Fluxbase with local filesystem or S3-compatible backends. Upload, download, and manage files with signed URLs and access policies.
---

Fluxbase provides file storage supporting local filesystem or S3-compatible storage (MinIO, AWS S3, Wasabi, DigitalOcean Spaces, etc.).

## Features

- Local filesystem or S3-compatible storage
- Bucket management
- File upload, download, delete, list operations
- Custom metadata support
- Signed URLs for temporary access (S3 only)
- Range requests for partial downloads
- Copy and move operations

## Configuration

```yaml
storage:
  provider: "local" # or "s3"
  local_path: "./storage"
  max_upload_size: 2147483648 # 2GB (default)

  # S3 Configuration (when provider: "s3")
  s3_endpoint: "s3.amazonaws.com"
  s3_access_key: "your-access-key"
  s3_secret_key: "your-secret-key"
  s3_region: "us-east-1"
  s3_bucket: "default-bucket"
```

### Environment Variables

```bash
FLUXBASE_STORAGE_PROVIDER=local  # or s3
FLUXBASE_STORAGE_LOCAL_PATH=./storage
FLUXBASE_STORAGE_MAX_UPLOAD_SIZE=2147483648

# S3 Configuration
FLUXBASE_STORAGE_S3_ENDPOINT=s3.amazonaws.com
FLUXBASE_STORAGE_S3_ACCESS_KEY=your-access-key
FLUXBASE_STORAGE_S3_SECRET_KEY=your-secret-key
FLUXBASE_STORAGE_S3_REGION=us-east-1
```

## Provider Comparison

**Local Storage:**

- Simple setup, no external dependencies
- Best for development and single-server deployments
- Not scalable across multiple servers

**S3-Compatible:**

- Highly scalable and distributed
- Best for production with multiple servers
- Requires external service (AWS S3, MinIO, etc.)

### Architecture Comparison

#### Local Storage Architecture

```mermaid
graph TB
    A[Client App 1] -->|Upload/Download| B[Fluxbase Server]
    C[Client App 2] -->|Upload/Download| B
    B -->|Read/Write| D[Local Filesystem<br/>/storage]

    E[Load Balancer] -.->|Cannot scale| F[Multiple Instances]
    F -.->|No shared filesystem| D

    style B fill:#3178c6,color:#fff
    style D fill:#f39c12,color:#fff
    style E fill:#e74c3c,color:#fff,stroke-dasharray: 5 5
    style F fill:#e74c3c,color:#fff,stroke-dasharray: 5 5
```

**Limitations:**

- Single server only - files stored locally cannot be accessed by multiple Fluxbase instances
- No horizontal scaling possible
- Server failure means data loss (unless backups exist)

#### S3-Compatible Storage Architecture (MinIO/S3)

```mermaid
graph TB
    A[Client 1] -->|API Request| LB[Load Balancer]
    B[Client 2] -->|API Request| LB
    C[Client 3] -->|API Request| LB

    LB --> FB1[Fluxbase Instance 1]
    LB --> FB2[Fluxbase Instance 2]
    LB --> FB3[Fluxbase Instance 3]

    FB1 -->|S3 API| S3[MinIO / S3 Cluster]
    FB2 -->|S3 API| S3
    FB3 -->|S3 API| S3

    S3 -->|Distributed| S3A[Storage Node 1]
    S3 -->|Distributed| S3B[Storage Node 2]
    S3 -->|Distributed| S3C[Storage Node 3]

    style LB fill:#ff6b6b,color:#fff
    style FB1 fill:#3178c6,color:#fff
    style FB2 fill:#3178c6,color:#fff
    style FB3 fill:#3178c6,color:#fff
    style S3 fill:#c92a2a,color:#fff
    style S3A fill:#5c940d,color:#fff
    style S3B fill:#5c940d,color:#fff
    style S3C fill:#5c940d,color:#fff
```

**Benefits:**

- Multiple Fluxbase instances can access the same storage
- Horizontally scalable - add more instances as needed
- High availability - storage cluster handles redundancy
- No single point of failure

**Use Cases:**

- **Local Storage**: Development, testing, single-server deployments
- **MinIO**: Self-hosted production with horizontal scaling needs
- **AWS S3/DigitalOcean Spaces**: Cloud production with managed infrastructure

## Installation

```bash
npm install @nimbleflux/fluxbase-sdk
```

## Basic Usage

```typescript
import { createClient } from "@nimbleflux/fluxbase-sdk";

const client = createClient("http://localhost:8080", "your-anon-key");

// Upload file
const file = document.getElementById("fileInput").files[0];
const { data, error } = await client.storage
  .from("avatars")
  .upload("user1.png", file);

// Download file
const { data: blob } = await client.storage
  .from("avatars")
  .download("user1.png");

// List files
const { data: files } = await client.storage.from("avatars").list();

// Delete file
await client.storage.from("avatars").remove(["user1.png"]);
```

## Bucket Operations

| Method           | Purpose            | Parameters |
| ---------------- | ------------------ | ---------- |
| `createBucket()` | Create new bucket  | `name`     |
| `listBuckets()`  | List all buckets   | None       |
| `getBucket()`    | Get bucket details | `name`     |
| `deleteBucket()` | Delete bucket      | `name`     |

**Example:**

```typescript
await client.storage.createBucket("avatars");

// List/get/delete
const { data: buckets } = await client.storage.listBuckets();
const { data: bucket } = await client.storage.getBucket("avatars");
await client.storage.deleteBucket("avatars");
```

## File Operations

| Method       | Purpose       | Parameters                                                    |
| ------------ | ------------- | ------------------------------------------------------------- |
| `upload()`   | Upload file   | `path`, `file`, `options` (contentType, cacheControl, upsert) |
| `download()` | Download file | `path`                                                        |
| `list()`     | List files    | `path`, `options` (limit, offset, sortBy)                     |
| `remove()`   | Delete files  | `paths[]`                                                     |
| `copy()`     | Copy file     | `from`, `to`                                                  |
| `move()`     | Move file     | `from`, `to`                                                  |

**Example:**

```typescript
// Upload
await client.storage
  .from("avatars")
  .upload("user1.png", file, { upsert: true });

// Download
const { data } = await client.storage.from("avatars").download("user1.png");

// List
const { data: files } = await client.storage
  .from("avatars")
  .list("subfolder/", { limit: 100 });

// Delete
await client.storage.from("avatars").remove(["file1.png", "file2.png"]);

// Copy/Move
await client.storage.from("avatars").copy("old.png", "new.png");
await client.storage.from("avatars").move("old.png", "new.png");
```

## Upload Progress Tracking

Track upload progress by providing an `onUploadProgress` callback in the upload options:

### Vanilla SDK

```typescript
import { createClient } from "@nimbleflux/fluxbase-sdk";

const client = createClient("http://localhost:8080", "your-anon-key");
const file = document.getElementById("fileInput").files[0];

// Upload with progress tracking
const { data, error } = await client.storage
  .from("avatars")
  .upload("user1.png", file, {
    onUploadProgress: (progress) => {
      console.log(`Upload progress: ${progress.percentage}%`);
      console.log(`Loaded: ${progress.loaded} / ${progress.total} bytes`);

      // Update UI progress bar
      const progressBar = document.getElementById("progress");
      progressBar.value = progress.percentage;
    },
  });
```

### React Hook (with automatic state management)

```tsx
import { useStorageUploadWithProgress } from "@nimbleflux/fluxbase-sdk-react";

function UploadComponent() {
  const { upload, progress, reset } = useStorageUploadWithProgress("avatars");

  const handleUpload = (file: File) => {
    upload.mutate({
      path: "user-avatar.png",
      file: file,
    });
  };

  return (
    <div>
      <input type="file" onChange={(e) => handleUpload(e.target.files[0])} />

      {progress && (
        <div>
          <progress value={progress.percentage} max="100" />
          <p>{progress.percentage}% uploaded</p>
          <p>
            {progress.loaded} / {progress.total} bytes
          </p>
        </div>
      )}

      {upload.isSuccess && <p>Upload complete!</p>}
      {upload.isError && <p>Upload failed: {upload.error?.message}</p>}
    </div>
  );
}
```

### React Hook (with custom callback)

```tsx
import { useState } from "react";
import { useStorageUpload } from "@nimbleflux/fluxbase-sdk-react";

function UploadComponent() {
  const upload = useStorageUpload("avatars");
  const [uploadProgress, setUploadProgress] = useState(0);

  const handleUpload = (file: File) => {
    upload.mutate({
      path: "user-avatar.png",
      file: file,
      options: {
        onUploadProgress: (progress) => {
          setUploadProgress(progress.percentage);
        },
      },
    });
  };

  return (
    <div>
      <input type="file" onChange={(e) => handleUpload(e.target.files[0])} />

      {uploadProgress > 0 && uploadProgress < 100 && (
        <div>
          <progress value={uploadProgress} max="100" />
          <p>{uploadProgress}% uploaded</p>
        </div>
      )}

      {upload.isSuccess && <p>Upload complete!</p>}
    </div>
  );
}
```

### Progress Object

The `onUploadProgress` callback receives an object with the following properties:

```typescript
interface UploadProgress {
  loaded: number; // Number of bytes uploaded so far
  total: number; // Total number of bytes to upload
  percentage: number; // Upload percentage (0-100)
}
```

### Notes

- Progress tracking uses `XMLHttpRequest` instead of `fetch()` for better progress events
- Progress tracking is optional and backward-compatible
- When no progress callback is provided, the standard `fetch()` API is used
- Progress updates may not be perfectly linear depending on network conditions
- The progress callback is called multiple times during the upload

## Public vs Private Files

```typescript
// Public bucket (no auth required)
await client.storage.createBucket("public-images");
const url = client.storage.from("public-images").getPublicUrl("logo.svg");

// Private bucket (requires auth or signed URL)
await client.storage.createBucket("private-docs");
```

## Signed URLs (S3 Only)

```typescript
const { data } = await client.storage
  .from("private-docs")
  .createSignedUrl("document.pdf", { expiresIn: 3600 }); // 1 hour expiry
```

## Metadata

```typescript
await client.storage.from("avatars").upload("profile.png", file, {
  metadata: { user_id: "123", description: "Profile picture" },
});
```

## S3 Provider Setup

### AWS S3

```yaml
storage:
  provider: "s3"
  s3_endpoint: "s3.amazonaws.com"
  s3_access_key: "AKIAIOSFODNN7EXAMPLE"
  s3_secret_key: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
  s3_region: "us-east-1"
  s3_bucket: "my-app-storage"
```

### MinIO (Self-Hosted)

```yaml
storage:
  provider: "s3"
  s3_endpoint: "localhost:9000"
  s3_access_key: "minioadmin"
  s3_secret_key: "minioadmin"
  s3_region: "us-east-1"
  s3_bucket: "fluxbase"
  s3_use_ssl: false # for development
```

Start MinIO with Docker:

```bash
docker run -d \
  -p 9000:9000 \
  -p 9001:9001 \
  --name minio \
  -e "MINIO_ROOT_USER=minioadmin" \
  -e "MINIO_ROOT_PASSWORD=minioadmin" \
  -v ./minio-data:/data \
  minio/minio server /data --console-address ":9001"
```

### DigitalOcean Spaces

```yaml
storage:
  provider: "s3"
  s3_endpoint: "nyc3.digitaloceanspaces.com"
  s3_access_key: "your-spaces-key"
  s3_secret_key: "your-spaces-secret"
  s3_region: "us-east-1"
  s3_bucket: "my-space"
```

## Best Practices

**File Naming:**

- Use consistent naming conventions
- Avoid special characters
- Use lowercase for better compatibility
- Include file extensions

**Security:**

- Keep buckets private by default
- Use signed URLs for temporary access
- Validate file types before upload
- Set file size limits
- Never expose S3 credentials in client code

**Performance:**

- Use appropriate file size limits
- Implement client-side compression for large files
- Use CDN for public files
- Cache control headers for static assets

**Organization:**

- Use path prefixes to organize files (e.g., `users/123/avatar.png`)
- Separate buckets by access level
- Use metadata for searchability

## S3 Credential Management

**SECURITY WARNING:** Never hardcode S3 credentials in configuration files or commit them to version control. Always use environment variables or secret management systems.

### Environment Variables (Recommended for Development)

```yaml
# fluxbase.yaml - DO NOT hardcode credentials here
storage:
  provider: "s3"
  s3_endpoint: "${FLUXBASE_STORAGE_S3_ENDPOINT}"
  s3_access_key: "${FLUXBASE_STORAGE_S3_ACCESS_KEY}"
  s3_secret_key: "${FLUXBASE_STORAGE_S3_SECRET_KEY}"
  s3_region: "${FLUXBASE_STORAGE_S3_REGION}"
  s3_bucket: "${FLUXBASE_STORAGE_S3_BUCKET}"
```

Set environment variables:

```bash
# .env file (NEVER commit this)
export FLUXBASE_STORAGE_S3_ENDPOINT="s3.amazonaws.com"
export FLUXBASE_STORAGE_S3_ACCESS_KEY="AKIAIOSFODNN7EXAMPLE"
export FLUXBASE_STORAGE_S3_SECRET_KEY="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
export FLUXBASE_STORAGE_S3_REGION="us-east-1"
export FLUXBASE_STORAGE_S3_BUCKET="my-app-storage"

# Load and run
source .env
fluxbase server
```

### Secret Management Systems (Recommended for Production)

:::note
Fluxbase does **not** support inline secret templating (e.g. `{{ vault ... }}`) in `fluxbase.yaml`. Inject credentials via environment variables (loaded from `.env`), your orchestrator's secret mechanism (Docker Secrets, Kubernetes Secrets), or IAM roles.
:::

#### Docker Secrets

```yaml
# docker-compose.yml
services:
  fluxbase:
    environment:
      - FLUXBASE_STORAGE_S3_ENDPOINT=file:/run/secrets/s3_endpoint
      - FLUXBASE_STORAGE_S3_ACCESS_KEY=file:/run/secrets/s3_access_key
      - FLUXBASE_STORAGE_S3_SECRET_KEY=file:/run/secrets/s3_secret_key
    secrets:
      - s3_endpoint
      - s3_access_key
      - s3_secret_key

secrets:
  s3_endpoint:
    external: true
  s3_access_key:
    external: true
  s3_secret_key:
    external: true
```

### IAM Roles (Best for AWS Deployments)

For applications running on AWS EC2, ECS, or Lambda, use IAM roles instead of credentials:

```yaml
# fluxbase.yaml
storage:
  provider: "s3"
  s3_endpoint: "s3.amazonaws.com"
  s3_region: "us-east-1"
  s3_bucket: "my-app-storage"
  # No access_key/secret_key needed - uses IAM role from EC2/ECS
```

Attach an IAM policy to your EC2 instance or ECS task:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject",
        "s3:DeleteObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::my-app-storage",
        "arn:aws:s3:::my-app-storage/*"
      ]
    }
  ]
}
```

### Credential Rotation

Regularly rotate S3 credentials:

1. **Generate new credentials** in AWS IAM or S3 provider
2. **Update environment variables** or secret management system
3. **Restart Fluxbase** to load new credentials
4. **Revoke old credentials** after confirming new ones work

Automation tools like AWS Secrets Manager or HashiCorp Vault can automate this process.

## Malware Scanning

Fluxbase does not include built-in malware scanning. For untrusted uploads, scan files in an edge function or job (e.g. calling ClamAV or a scanning API) before serving them.

## Error Handling

```typescript
try {
  const { data, error } = await client.storage
    .from("avatars")
    .upload("file.png", file);

  if (error) {
    if (error.message.includes("already exists")) {
      // File exists, use upsert: true or different name
    } else if (error.message.includes("too large")) {
      // File exceeds size limit
    } else {
      // Other error
      console.error("Upload error:", error);
    }
  }
} catch (err) {
  console.error("Network error:", err);
}
```

## REST API

All storage endpoints are under `/api/v1/storage`. Bucket and object operations require authentication and the `storage:read` / `storage:write` scopes.

### Uploads

| Mode | Method & Path | When to use |
|------|---------------|-------------|
| Simple | `POST /storage/{bucket}/{key}` | Typical uploads (request body is the file) |
| Multipart | `POST /storage/{bucket}/multipart` | S3-style multipart uploads |
| Streaming | `POST /storage/{bucket}/stream/{key}` | Large streamed request bodies |
| Resumable (chunked) | `POST /storage/{bucket}/chunked/init` → `PUT /storage/{bucket}/chunked/{uploadId}/{chunkIndex}` → `POST /storage/{bucket}/chunked/{uploadId}/complete` | Very large files; resumable across dropped connections |

Resumable uploads also expose `GET /storage/{bucket}/chunked/{uploadId}/status` (progress) and `DELETE /storage/{bucket}/chunked/{uploadId}` (abort).

### Downloads

`GET` or `HEAD` `/storage/{bucket}/{key}` downloads an object or fetches its metadata. `GET /storage/object` downloads via a signed-URL token (public, no auth header required).

### Signed URLs

`POST /storage/{bucket}/sign/{key}` generates a time-limited signed URL for a private object, with optional image-transform parameters.

### File Sharing

Share an object with another user and manage grants:

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/storage/{bucket}/{key}/share` | Share a file with a user (read/write) |
| `GET` | `/storage/{bucket}/{key}/shares` | List a file's shares |
| `DELETE` | `/storage/{bucket}/{key}/share/{user_id}` | Revoke a share |

For SDK usage, see the [Storage SDK reference](/api/sdk/classes/fluxbasestorage/).

## Related Documentation

- [Authentication](/guides/authentication) - Secure file access
- [Row-Level Security](/guides/row-level-security) - File access policies
- [Configuration](/reference/configuration) - All storage options
