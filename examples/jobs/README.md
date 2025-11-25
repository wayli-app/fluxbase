# Fluxbase Jobs Examples

This directory contains example job functions demonstrating the capabilities of the Fluxbase Jobs system.

## Overview

Background jobs in Fluxbase allow you to run long-running tasks asynchronously with:
- **Progress tracking** - Real-time progress updates
- **Retry logic** - Automatic retries on failure
- **Permissions** - Role-based access control
- **User context** - Access to user identity and permissions
- **Scheduled execution** - Cron-based scheduling
- **Environment variables** - Access to server configuration
- **Database access** - Full RLS-aware database queries

## Example Jobs

### 1. send-report.ts
**Admin-only report generation**

Demonstrates:
- `@fluxbase:require-role admin` - Only admins can trigger this job
- Accessing user context (`context.user`)
- Using environment variables (`Deno.env.get()`)
- Progress reporting (`Fluxbase.reportProgress()`)
- Database queries with RLS

**Submit this job:**
```typescript
// Admin user required
const { data, error } = await client.jobs.submit('send-report', {
  report_type: 'monthly'
})

if (data) {
  console.log('Job ID:', data.id)
  console.log('Status:', data.status)
}
```

### 2. process-user-data.ts
**User-specific data processing**

Demonstrates:
- No special permissions (any authenticated user)
- User context automatically limits access to user's own data
- Processing arrays of items
- Error handling and retry logic
- Per-item progress updates

**Submit this job:**
```typescript
// Any authenticated user
const { data, error } = await client.jobs.submit('process-user-data', {
  items: [
    { id: 1, data: 'item1' },
    { id: 2, data: 'item2' },
    { id: 3, data: 'item3' }
  ]
})

if (data) {
  console.log('Job submitted:', data.id)

  // Poll for status
  const checkStatus = async () => {
    const { data: job } = await client.jobs.get(data.id)
    if (job) {
      console.log(`Progress: ${job.progress_percent}% - ${job.progress_message}`)
      if (job.status === 'running' || job.status === 'pending') {
        setTimeout(checkStatus, 2000) // Check every 2 seconds
      } else {
        console.log('Job completed:', job.result)
      }
    }
  }
  checkStatus()
}
```

### 3. cleanup-old-data.ts
**Scheduled cleanup job**

Demonstrates:
- `@fluxbase:schedule 0 2 * * *` - Runs daily at 2 AM
- Admin-only access
- Batch database operations
- Can also be triggered manually

**Submit manually:**
```typescript
// Admin user required
const { data, error } = await client.jobs.submit('cleanup-old-data', {
  retention_days: 60 // Override default 30 days
})
```

## Annotations

Job functions support JSDoc-style annotations:

- `@fluxbase:require-role <role>` - Require specific role (admin, user, etc.)
- `@fluxbase:timeout <seconds>` - Maximum execution time
- `@fluxbase:max-retries <count>` - Number of retry attempts
- `@fluxbase:schedule <cron>` - Cron expression for scheduled execution
- `@fluxbase:description <text>` - Job description

## Available APIs in Job Functions

### Job Context
```typescript
const context = Fluxbase.getJobContext()
// {
//   job_id: "uuid",
//   job_name: "my-job",
//   namespace: "default",
//   retry_count: 0,
//   payload: { ... },
//   user: {
//     id: "uuid",
//     email: "user@example.com",
//     role: "user"
//   } | null
// }
```

### Progress Reporting
```typescript
// Update progress (0-100) with optional message
await Fluxbase.reportProgress(50, "Processing item 5 of 10")
```

### Database Access
```typescript
// Full Supabase-compatible database client with RLS
const { data, error } = await Fluxbase.database()
  .from('app.my_table')
  .select('*')
  .eq('user_id', context.user.id)

// Queries automatically use user context for RLS
```

### Environment Variables
```typescript
// Only FLUXBASE_* variables are available
const apiUrl = Deno.env.get('FLUXBASE_API_URL')
const appName = Deno.env.get('FLUXBASE_APP_NAME')

// Sensitive variables are blocked for security
```

### Logging
```typescript
console.log('Info message')
console.error('Error message')
console.warn('Warning message')

// Logs are captured and stored with the job
```

## Deployment

### 1. Using SDK (Recommended)
```typescript
import { createClient } from '@fluxbase/sdk'

const client = createClient(process.env.FLUXBASE_URL!, {
  serviceKey: process.env.FLUXBASE_SERVICE_KEY!
})

// Read job file
const code = await Deno.readTextFile('./send-report.ts')

// Create job function
const { data, error } = await client.admin.jobs.create({
  name: 'send-report',
  namespace: 'default',
  code,
  enabled: true,
  timeout_seconds: 600
})
```

### 2. Using Filesystem Auto-Load
```bash
# Configure in .env
FLUXBASE_JOBS_ENABLED=true
FLUXBASE_JOBS_DIR=./jobs
FLUXBASE_JOBS_AUTO_LOAD_ON_BOOT=true

# Place job files in ./jobs directory
mkdir -p jobs
cp examples/jobs/*.ts jobs/

# Restart server - jobs will be automatically loaded
```

### 3. Using Sync API
```typescript
// Sync all jobs from filesystem to database
const { data, error } = await client.admin.jobs.sync('default')

if (data) {
  console.log(`Created: ${data.summary.created}`)
  console.log(`Updated: ${data.summary.updated}`)
  console.log(`Deleted: ${data.summary.deleted}`)
}
```

## Monitoring Jobs

### Get Job Status
```typescript
const { data: job } = await client.jobs.get(jobId)

console.log('Status:', job.status)
console.log('Progress:', job.progress_percent + '%')
console.log('Message:', job.progress_message)
console.log('Result:', job.result)
console.log('Logs:', job.logs)
```

### List User's Jobs
```typescript
const { data: jobs } = await client.jobs.list({
  status: 'running',
  limit: 20
})

jobs?.forEach(job => {
  console.log(`${job.job_name}: ${job.status} (${job.progress_percent}%)`)
})
```

### Admin: View All Jobs
```typescript
const { data: jobs } = await client.admin.jobs.listJobs({
  namespace: 'default',
  status: 'running'
})
```

### Admin: Get Statistics
```typescript
const { data: stats } = await client.admin.jobs.getStats('default')

console.log(`Pending: ${stats.pending}`)
console.log(`Running: ${stats.running}`)
console.log(`Completed: ${stats.completed}`)
console.log(`Failed: ${stats.failed}`)
```

### Admin: List Workers
```typescript
const { data: workers } = await client.admin.jobs.listWorkers()

workers?.forEach(worker => {
  console.log(`Worker ${worker.id}: ${worker.current_jobs} jobs`)
})
```

## Job Operations

### Cancel a Running Job
```typescript
const { error } = await client.jobs.cancel(jobId)
```

### Retry a Failed Job
```typescript
const { data: newJob, error } = await client.jobs.retry(jobId)
console.log('New job ID:', newJob.id)
```

### Admin: Terminate Job Immediately
```typescript
const { error } = await client.admin.jobs.terminate(jobId)
```

## Configuration

Set these environment variables to configure the jobs system:

```bash
# Enable jobs
FLUXBASE_JOBS_ENABLED=true

# Jobs directory for auto-load
FLUXBASE_JOBS_DIR=./jobs

# Auto-load jobs on server boot
FLUXBASE_JOBS_AUTO_LOAD_ON_BOOT=true

# Number of embedded worker threads
FLUXBASE_JOBS_EMBEDDED_WORKER_COUNT=4

# Default timeout (seconds)
FLUXBASE_JOBS_DEFAULT_TIMEOUT=300

# Default max retries
FLUXBASE_JOBS_DEFAULT_MAX_RETRIES=3
```

## Security

### Permissions
- Jobs without `@fluxbase:require-role` can be triggered by any authenticated user
- Admin-only jobs: `@fluxbase:require-role admin`
- Custom roles: `@fluxbase:require-role custom_role`

### User Context
- User's ID, email, and role are stored with the job
- Database queries automatically use user context for RLS
- JWT tokens are never stored (only metadata)

### Environment Variables
- Only `FLUXBASE_*` prefixed variables are accessible
- Sensitive secrets are explicitly blocked:
  - FLUXBASE_AUTH_JWT_SECRET
  - FLUXBASE_DATABASE_PASSWORD
  - FLUXBASE_DATABASE_ADMIN_PASSWORD
  - FLUXBASE_STORAGE_S3_SECRET_KEY
  - FLUXBASE_STORAGE_S3_ACCESS_KEY
  - FLUXBASE_EMAIL_SMTP_PASSWORD
  - FLUXBASE_SECURITY_SETUP_TOKEN

## Testing Jobs Locally

```bash
# Start development server with jobs enabled
make dev

# In another terminal, submit a test job
curl -X POST http://localhost:8000/api/v1/jobs/submit \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "job_name": "process-user-data",
    "payload": {
      "items": [{"id": 1}, {"id": 2}]
    }
  }'

# Check job status
curl http://localhost:8000/api/v1/jobs/$JOB_ID \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

## Common Patterns

### Long-Running Imports
```typescript
async function handler(req: any) {
  const context = Fluxbase.getJobContext()
  const { file_url } = context.payload

  // Download file
  await Fluxbase.reportProgress(10, "Downloading file...")
  const response = await fetch(file_url)
  const data = await response.json()

  // Process in batches
  const batchSize = 100
  let processed = 0

  for (let i = 0; i < data.length; i += batchSize) {
    const batch = data.slice(i, i + batchSize)

    // Insert batch
    await Fluxbase.database()
      .from('app.imports')
      .insert(batch.map(item => ({
        ...item,
        user_id: context.user.id
      })))

    processed += batch.length
    const progress = 10 + Math.floor((processed / data.length) * 90)
    await Fluxbase.reportProgress(progress, `Imported ${processed}/${data.length} records`)
  }

  return { success: true, imported: processed }
}
```

### Parallel Processing
```typescript
async function handler(req: any) {
  const { items } = Fluxbase.getJobContext().payload

  // Process items in parallel
  const results = await Promise.all(
    items.map(async (item, index) => {
      const result = await processItem(item)

      // Update progress
      const progress = Math.floor(((index + 1) / items.length) * 100)
      await Fluxbase.reportProgress(progress, `Processed ${index + 1}/${items.length}`)

      return result
    })
  )

  return { success: true, results }
}
```

### With External API Calls
```typescript
async function handler(req: any) {
  const context = Fluxbase.getJobContext()
  const apiKey = Deno.env.get('FLUXBASE_EXTERNAL_API_KEY')

  await Fluxbase.reportProgress(25, "Calling external API...")

  const response = await fetch('https://api.example.com/data', {
    headers: {
      'Authorization': `Bearer ${apiKey}`,
      'Content-Type': 'application/json'
    }
  })

  const data = await response.json()

  await Fluxbase.reportProgress(50, "Storing results...")

  // Store results
  await Fluxbase.database()
    .from('app.api_results')
    .insert({
      data,
      user_id: context.user.id,
      fetched_at: new Date().toISOString()
    })

  await Fluxbase.reportProgress(100, "Complete")

  return { success: true, records: data.length }
}
```
