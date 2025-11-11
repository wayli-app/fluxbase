# File Storage

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
  max_upload_size: 10485760 # 10MB

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
FLUXBASE_STORAGE_MAX_UPLOAD_SIZE=10485760

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
npm install @fluxbase/sdk
```

## Basic Usage

```typescript
import { createClient } from '@fluxbase/sdk'

const client = createClient('http://localhost:8080', 'your-api-key')

// Upload file
const file = document.getElementById('fileInput').files[0]
const { data, error } = await client.storage
  .from('avatars')
  .upload('user1.png', file)

// Download file
const { data: blob } = await client.storage
  .from('avatars')
  .download('user1.png')

// List files
const { data: files } = await client.storage
  .from('avatars')
  .list()

// Delete file
await client.storage
  .from('avatars')
  .remove(['user1.png'])
```

## Bucket Operations

| Method | Purpose | Parameters |
|--------|---------|------------|
| `createBucket()` | Create new bucket | `name`, `options` (public, file_size_limit, allowed_mime_types) |
| `listBuckets()` | List all buckets | None |
| `getBucket()` | Get bucket details | `name` |
| `deleteBucket()` | Delete bucket | `name` |

**Example:**

```typescript
// Create bucket
await client.storage.createBucket('avatars', {
  public: false,
  file_size_limit: 5242880,
  allowed_mime_types: ['image/png', 'image/jpeg']
})

// List/get/delete
const { data: buckets } = await client.storage.listBuckets()
const { data: bucket } = await client.storage.getBucket('avatars')
await client.storage.deleteBucket('avatars')
```

## File Operations

| Method | Purpose | Parameters |
|--------|---------|------------|
| `upload()` | Upload file | `path`, `file`, `options` (contentType, cacheControl, upsert) |
| `download()` | Download file | `path` |
| `list()` | List files | `path`, `options` (limit, offset, sortBy) |
| `remove()` | Delete files | `paths[]` |
| `copy()` | Copy file | `from`, `to` |
| `move()` | Move file | `from`, `to` |

**Example:**

```typescript
// Upload
await client.storage
  .from('avatars')
  .upload('user1.png', file, { upsert: true })

// Download
const { data } = await client.storage
  .from('avatars')
  .download('user1.png')

// List
const { data: files } = await client.storage
  .from('avatars')
  .list('subfolder/', { limit: 100 })

// Delete
await client.storage
  .from('avatars')
  .remove(['file1.png', 'file2.png'])

// Copy/Move
await client.storage.from('avatars').copy('old.png', 'new.png')
await client.storage.from('avatars').move('old.png', 'new.png')
```

## Public vs Private Files

```typescript
// Public bucket (no auth required)
await client.storage.createBucket('public-images', { public: true })
const url = client.storage.from('public-images').getPublicUrl('logo.png')

// Private bucket (requires auth or signed URL)
await client.storage.createBucket('private-docs', { public: false })
```

## Signed URLs (S3 Only)

```typescript
const { data } = await client.storage
  .from('private-docs')
  .createSignedUrl('document.pdf', 3600) // 1 hour expiry
```

## Metadata

```typescript
// Upload with metadata
await client.storage.from('avatars').upload('profile.png', file, {
  metadata: { user_id: '123', description: 'Profile picture' }
})

// Get file info
const { data } = await client.storage.from('avatars').getFileInfo('profile.png')
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

## Error Handling

```typescript
try {
  const { data, error } = await client.storage
    .from('avatars')
    .upload('file.png', file)

  if (error) {
    if (error.message.includes('already exists')) {
      // File exists, use upsert: true or different name
    } else if (error.message.includes('too large')) {
      // File exceeds size limit
    } else {
      // Other error
      console.error('Upload error:', error)
    }
  }
} catch (err) {
  console.error('Network error:', err)
}
```

## REST API

For direct HTTP access without the SDK, see the [Storage API Reference](/docs/api/storage).

## Related Documentation

- [Authentication](/docs/guides/authentication) - Secure file access
- [Row-Level Security](/docs/guides/row-level-security) - File access policies
- [Configuration](/docs/reference/configuration) - All storage options
