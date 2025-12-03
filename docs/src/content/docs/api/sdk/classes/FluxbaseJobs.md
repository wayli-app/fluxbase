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

### new FluxbaseJobs()

> **new FluxbaseJobs**(`fetch`): [`FluxbaseJobs`](/api/sdk/classes/fluxbasejobs/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |

#### Returns

[`FluxbaseJobs`](/api/sdk/classes/fluxbasejobs/)

## Methods

### cancel()

> **cancel**(`jobId`): `Promise`\<`object`\>

Cancel a pending or running job

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `jobId` | `string` | Job ID to cancel |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple

| Name | Type |
| ------ | ------ |
| `data` | `null` |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { error } = await client.jobs.cancel('550e8400-e29b-41d4-a716-446655440000')

if (!error) {
  console.log('Job cancelled successfully')
}
```

***

### get()

> **get**(`jobId`): `Promise`\<`object`\>

Get status and details of a specific job

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `jobId` | `string` | Job ID |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with job details

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `Job` |
| `error` | `null` \| `Error` |

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

### list()

> **list**(`filters`?): `Promise`\<`object`\>

List jobs submitted by the current user

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `filters`? | `object` | Optional filters (status, namespace, limit, offset) |
| `filters.includeResult`? | `boolean` | - |
| `filters.limit`? | `number` | - |
| `filters.namespace`? | `string` | - |
| `filters.offset`? | `number` | - |
| `filters.status`? | `string` | - |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with array of jobs

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `Job`[] |
| `error` | `null` \| `Error` |

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

> **retry**(`jobId`): `Promise`\<`object`\>

Retry a failed job

Creates a new job execution with the same parameters

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `jobId` | `string` | Job ID to retry |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with new job

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `Job` |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data: newJob, error } = await client.jobs.retry('550e8400-e29b-41d4-a716-446655440000')

if (newJob) {
  console.log('Job retried, new ID:', newJob.id)
}
```

***

### submit()

> **submit**(`jobName`, `payload`?, `options`?): `Promise`\<`object`\>

Submit a new job for execution

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `jobName` | `string` | Name of the job function to execute |
| `payload`? | `any` | Job input data |
| `options`? | `object` | Additional options (priority, namespace, scheduled time) |
| `options.namespace`? | `string` | - |
| `options.priority`? | `number` | - |
| `options.scheduled`? | `string` | - |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with submitted job details

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `Job` |
| `error` | `null` \| `Error` |

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
```
