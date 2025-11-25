# Quick Start Guide: Fluxbase Jobs

This guide will help you get started with Fluxbase jobs in under 5 minutes.

## Prerequisites

- Fluxbase server running (see main README)
- Docker (for local development)
- Node.js or Deno installed

## Step 1: Start Fluxbase with Jobs Enabled

```bash
# Clone the repository (if not already)
git clone https://github.com/wayli-app/fluxbase.git
cd fluxbase

# Configure environment for jobs (optional - these are defaults)
cat >> .env.local << EOF
FLUXBASE_JOBS_DIR=./examples/jobs
FLUXBASE_JOBS_AUTO_LOAD_ON_BOOT=true
FLUXBASE_JOBS_EMBEDDED_WORKER_COUNT=4
FLUXBASE_FEATURES_JOBS_ENABLED=true
EOF

# Start development server
make dev
```

The server will automatically load all job functions from `./examples/jobs/` on startup.

## Step 2: Verify Jobs are Loaded

```bash
# Get admin token (after initial setup)
export ADMIN_TOKEN="your_admin_token_here"

# List loaded job functions
curl http://localhost:8000/api/v1/admin/jobs/functions \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

You should see the example jobs: `send-report`, `process-user-data`, `cleanup-old-data`, `bulk-export`.

## Step 3: Submit Your First Job

### Using curl:

```bash
# First, authenticate as a user
LOGIN_RESPONSE=$(curl -X POST http://localhost:8000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }')

# Extract access token
export ACCESS_TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.access_token')

# Submit a job
JOB_RESPONSE=$(curl -X POST http://localhost:8000/api/v1/jobs/submit \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "job_name": "process-user-data",
    "payload": {
      "items": [
        {"id": 1, "data": "item1"},
        {"id": 2, "data": "item2"},
        {"id": 3, "data": "item3"}
      ]
    }
  }')

# Extract job ID
export JOB_ID=$(echo $JOB_RESPONSE | jq -r '.id')
echo "Job ID: $JOB_ID"

# Check job status
curl http://localhost:8000/api/v1/jobs/$JOB_ID \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq
```

### Using the SDK:

```typescript
import { createClient } from "@fluxbase/sdk";

const client = createClient("http://localhost:8000", {
  apiKey: process.env.FLUXBASE_ANON_KEY!,
});

// Login
await client.auth.login({
  email: "user@example.com",
  password: "password123",
});

// Submit job
const { data: job, error } = await client.jobs.submit("process-user-data", {
  items: [
    { id: 1, data: "item1" },
    { id: 2, data: "item2" },
    { id: 3, data: "item3" },
  ],
});

console.log("Job ID:", job?.id);
console.log("Status:", job?.status);

// Check status
const { data: status } = await client.jobs.get(job!.id);
console.log("Progress:", status?.progress_percent + "%");
console.log("Message:", status?.progress_message);
```

## Step 4: Monitor Job Progress

### Using curl (polling):

```bash
# Poll every 2 seconds
while true; do
  curl http://localhost:8000/api/v1/jobs/$JOB_ID \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -s | jq '{status, progress_percent, progress_message, result}'
  sleep 2
done
```

### Using SDK:

```typescript
async function waitForJob(jobId: string) {
  let completed = false;

  while (!completed) {
    const { data: job } = await client.jobs.get(jobId);

    console.log(`${job?.progress_percent}%: ${job?.progress_message}`);

    if (job?.status === "completed") {
      console.log("Result:", job.result);
      completed = true;
    } else if (job?.status === "failed") {
      console.error("Error:", job.error);
      completed = true;
    } else if (job?.status === "cancelled") {
      console.log("Job was cancelled");
      completed = true;
    }

    if (!completed) {
      await new Promise((resolve) => setTimeout(resolve, 2000));
    }
  }
}

await waitForJob(job!.id);
```

## Step 5: Try an Admin Job

Admin jobs require the `admin` role. Here's how to submit an admin-only job:

```bash
# Authenticate as admin
ADMIN_LOGIN=$(curl -X POST http://localhost:8000/api/v1/admin/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "admin_password"
  }')

export ADMIN_TOKEN=$(echo $ADMIN_LOGIN | jq -r '.access_token')

# Submit admin job (send-report)
curl -X POST http://localhost:8000/api/v1/jobs/submit \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "job_name": "send-report",
    "payload": {
      "report_type": "monthly"
    }
  }' | jq
```

**Note**: Regular users will get a 403 Forbidden error if they try to submit this job.

## Step 6: Create Your Own Job

1. Create a new file in your jobs directory:

```bash
cat > ./jobs/my-first-job.ts << 'EOF'
/**
 * My First Job
 *
 * @fluxbase:timeout 300
 * @fluxbase:description My first custom job function
 */

export async function handler(req: any) {
  const context = Fluxbase.getJobContext()

  console.log('Hello from my first job!')
  console.log('User:', context.user?.email)
  console.log('Payload:', context.payload)

  await Fluxbase.reportProgress(25, 'Step 1: Starting...')
  await new Promise(resolve => setTimeout(resolve, 1000))

  await Fluxbase.reportProgress(50, 'Step 2: Processing...')
  await new Promise(resolve => setTimeout(resolve, 1000))

  await Fluxbase.reportProgress(75, 'Step 3: Finishing...')
  await new Promise(resolve => setTimeout(resolve, 1000))

  await Fluxbase.reportProgress(100, 'Complete!')

  return {
    success: true,
    message: 'Job completed successfully!',
    user: context.user?.email
  }
}
EOF
```

2. Sync the new job to the database:

```bash
# Using SDK
const { data } = await client.admin.jobs.sync('default')
console.log('Created:', data?.summary.created)

# Or restart the server (if auto_load_on_boot is enabled)
make restart
```

3. Submit your job:

```bash
curl -X POST http://localhost:8000/api/v1/jobs/submit \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "job_name": "my-first-job",
    "payload": {
      "message": "Hello, World!"
    }
  }' | jq
```

## Common Commands

### List all jobs

```bash
curl http://localhost:8000/api/v1/jobs \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq
```

### List running jobs

```bash
curl http://localhost:8000/api/v1/jobs?status=running \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq
```

### Cancel a job

```bash
curl -X POST http://localhost:8000/api/v1/jobs/$JOB_ID/cancel \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq
```

### Retry a failed job

```bash
curl -X POST http://localhost:8000/api/v1/jobs/$JOB_ID/retry \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq
```

### Admin: View statistics

```bash
curl http://localhost:8000/api/v1/admin/jobs/stats \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq
```

### Admin: List all job functions

```bash
curl http://localhost:8000/api/v1/admin/jobs/functions \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq
```

### Admin: View all jobs (all users)

```bash
curl http://localhost:8000/api/v1/admin/jobs/queue \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq
```

### Admin: List workers

```bash
curl http://localhost:8000/api/v1/admin/jobs/workers \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq
```

## Troubleshooting

### Jobs not loading at startup

- Check `FLUXBASE_FEATURES_JOBS_ENABLED=true` (enabled by default)
- Check `FLUXBASE_JOBS_DIR` points to the correct directory
- Check `FLUXBASE_JOBS_AUTO_LOAD_ON_BOOT=true` (enabled by default)
- Check server logs for error messages

### Job stuck in "pending" status

- Check that workers are running: `curl http://localhost:8000/api/v1/admin/jobs/workers`
- Check `FLUXBASE_JOBS_EMBEDDED_WORKER_COUNT > 0`
- Check server logs for worker errors

### Permission denied errors

- Regular users cannot submit jobs with `@fluxbase:require-role admin`
- Check the user's role matches the required role
- Use admin token for admin-only jobs

### Job fails immediately

- Check job logs: `curl http://localhost:8000/api/v1/jobs/$JOB_ID | jq .logs`
- Check for syntax errors in your job code
- Check database permissions (RLS policies)

### Environment variables not available

- Only `FLUXBASE_*` prefixed variables are available
- Check the variable is set in your `.env.local`
- Restart the server after changing environment variables

## Next Steps

- Read the full [README.md](./README.md) for detailed documentation
- Explore the example jobs to see different patterns
- Check out [client-example.ts](./client-example.ts) for SDK usage
- Learn about [job permissions](../../JOBS_PERMISSIONS_IMPLEMENTATION.md)

## Resources

- [Fluxbase Documentation](../../docs)
- [API Reference](../../docs/api)
- [Jobs Configuration](../../internal/config)
- [Issue Tracker](https://github.com/wayli-app/fluxbase/issues)
