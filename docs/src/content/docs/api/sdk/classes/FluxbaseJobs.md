---
editUrl: false
next: false
prev: false
title: "FluxbaseJobs"
---

Jobs client for submitting and monitoring background jobs

For admin operations (create job functions, manage workers, view all jobs),
use client.admin.jobs

## Constructors

### Constructor

> **new FluxbaseJobs**(`fetch`, `isServiceRole?`): `FluxbaseJobs`

#### Parameters

| Parameter | Type | Default value |
| ------ | ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) | `undefined` |
| `isServiceRole` | `boolean` | `false` |

#### Returns

`FluxbaseJobs`

## Methods

### cancel()

> **cancel**(`jobId`): `Promise`\<\{ `data`: `null`; `error`: `Error` \| `null`; \}\>

Cancel a pending or running job

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `jobId` | `string` | Job ID to cancel |

#### Returns

`Promise`\<\{ `data`: `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple

#### Example

```typescript
const { error } = await client.jobs.cancel('550e8400-e29b-41d4-a716-446655440000')

if (!error) {
  console.log('Job cancelled successfully')
}
```

***

### get()

> **get**(`jobId`): `Promise`\<\{ `data`: [`Job`](/api/sdk/interfaces/job/) \| `null`; `error`: `Error` \| `null`; \}\>

Get status and details of a specific job

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `jobId` | `string` | Job ID |

#### Returns

`Promise`\<\{ `data`: [`Job`](/api/sdk/interfaces/job/) \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with job details

#### Example

```typescript
const { data: job, error } = await client.jobs.get('550e8400-e29b-41d4-a716-446655440000')

if (job) {
  console.log('Status:', job.status)
  console.log('Progress:', job.progress_percent + '%')
  console.log('Result:', job.result)
  console.log('Logs:', job.logs)
}
```

***

### getLogs()

> **getLogs**(`jobId`, `afterLine?`): `Promise`\<\{ `data`: [`ExecutionLog`](/api/sdk/interfaces/executionlog/)[] \| `null`; `error`: `Error` \| `null`; \}\>

Get execution logs for a job

Returns logs for the specified job. Only returns logs for jobs
owned by the authenticated user (unless using service_role).

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `jobId` | `string` | Job ID |
| `afterLine?` | `number` | Optional line number to get logs after (for polling/streaming) |

#### Returns

`Promise`\<\{ `data`: [`ExecutionLog`](/api/sdk/interfaces/executionlog/)[] \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with execution logs

#### Example

```typescript
// Get all logs for a job
const { data: logs, error } = await client.jobs.getLogs('550e8400-e29b-41d4-a716-446655440000')

if (logs) {
  for (const log of logs) {
    console.log(`[${log.level}] ${log.message}`)
  }
}

// Backfill + stream pattern
const { data: logs } = await client.jobs.getLogs(jobId)
let lastLine = Math.max(...(logs?.map(l => l.line_number) ?? []), 0)

const channel = client.realtime
  .executionLogs(jobId, 'job')
  .onLog((log) => {
    if (log.line_number > lastLine) {
      displayLog(log)
      lastLine = log.line_number
    }
  })
  .subscribe()
```

***

### list()

> **list**(`filters?`): `Promise`\<\{ `data`: [`Job`](/api/sdk/interfaces/job/)[] \| `null`; `error`: `Error` \| `null`; \}\>

List jobs submitted by the current user

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `filters?` | \{ `includeResult?`: `boolean`; `limit?`: `number`; `namespace?`: `string`; `offset?`: `number`; `status?`: `string`; \} | Optional filters (status, namespace, limit, offset) |
| `filters.includeResult?` | `boolean` | - |
| `filters.limit?` | `number` | - |
| `filters.namespace?` | `string` | - |
| `filters.offset?` | `number` | - |
| `filters.status?` | `string` | - |

#### Returns

`Promise`\<\{ `data`: [`Job`](/api/sdk/interfaces/job/)[] \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with array of jobs

#### Example

```typescript
// List all your jobs
const { data: jobs, error } = await client.jobs.list()

// Filter by status
const { data: running } = await client.jobs.list({
  status: 'running'
})

// Paginate
const { data: page } = await client.jobs.list({
  limit: 20,
  offset: 40
})
```

***

### retry()

> **retry**(`jobId`): `Promise`\<\{ `data`: [`Job`](/api/sdk/interfaces/job/) \| `null`; `error`: `Error` \| `null`; \}\>

Retry a failed job

Creates a new job execution with the same parameters

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `jobId` | `string` | Job ID to retry |

#### Returns

`Promise`\<\{ `data`: [`Job`](/api/sdk/interfaces/job/) \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with new job

#### Example

```typescript
const { data: newJob, error } = await client.jobs.retry('550e8400-e29b-41d4-a716-446655440000')

if (newJob) {
  console.log('Job retried, new ID:', newJob.id)
}
```

***

### submit()

> **submit**(`jobName`, `payload?`, `options?`): `Promise`\<\{ `data`: [`Job`](/api/sdk/interfaces/job/) \| `null`; `error`: `Error` \| `null`; \}\>

Submit a new job for execution

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `jobName` | `string` | Name of the job function to execute |
| `payload?` | `unknown` | Job input data |
| `options?` | \{ `namespace?`: `string`; `onBehalfOf?`: `OnBehalfOf`; `priority?`: `number`; `scheduled?`: `string`; \} | Additional options (priority, namespace, scheduled time, onBehalfOf) |
| `options.namespace?` | `string` | - |
| `options.onBehalfOf?` | `OnBehalfOf` | Submit job on behalf of another user (service_role only). The job will be created with the specified user's identity, allowing them to see the job and its logs via RLS. If not provided, the current user's identity and role from user_profiles will be automatically included. |
| `options.priority?` | `number` | - |
| `options.scheduled?` | `string` | - |

#### Returns

`Promise`\<\{ `data`: [`Job`](/api/sdk/interfaces/job/) \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with submitted job details

#### Example

```typescript
// Submit a simple job
const { data, error } = await client.jobs.submit('send-email', {
  to: 'user@example.com',
  subject: 'Hello',
  body: 'Welcome!'
})

if (data) {
  console.log('Job submitted:', data.id)
  console.log('Status:', data.status)
}

// Submit with priority
const { data } = await client.jobs.submit('high-priority-task', payload, {
  priority: 10
})

// Schedule for later
const { data } = await client.jobs.submit('scheduled-task', payload, {
  scheduled: '2025-01-01T00:00:00Z'
})

// Submit on behalf of a user (service_role only)
const { data } = await serviceClient.jobs.submit('user-task', payload, {
  onBehalfOf: {
    user_id: 'user-uuid',
    user_email: 'user@example.com'
  }
})
```
