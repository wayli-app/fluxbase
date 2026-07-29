---
editUrl: false
next: false
prev: false
title: "FluxbaseAdminJobs"
---

Admin Jobs manager for managing background job functions
Provides create, update, delete, sync, and monitoring operations

## Constructors

### Constructor

> **new FluxbaseAdminJobs**(`fetch`): `FluxbaseAdminJobs`

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |

#### Returns

`FluxbaseAdminJobs`

## Methods

### cancel()

> **cancel**(`jobId`): `Promise`\<\{ `data`: `null`; `error`: `Error` \| `null`; \}\>

Cancel a running or pending job

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `jobId` | `string` | Job ID |

#### Returns

`Promise`\<\{ `data`: `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple

#### Example

```typescript
const { data, error } = await client.admin.jobs.cancel('550e8400-e29b-41d4-a716-446655440000')
```

***

### create()

> **create**(`params`): `Promise`\<\{ `data`: `JobFunction` \| `null`; `error`: `Error` \| `null`; \}\>

Create a new job function

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `params` | \{ `code`: `string`; `enabled?`: `boolean`; `name`: `string`; `namespace?`: `string`; `timeout_seconds?`: `number`; \} | Job function configuration |
| `params.code` | `string` | - |
| `params.enabled?` | `boolean` | - |
| `params.name` | `string` | - |
| `params.namespace?` | `string` | - |
| `params.timeout_seconds?` | `number` | - |

#### Returns

`Promise`\<\{ `data`: `JobFunction` \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with created job function

***

### delete()

> **delete**(`namespace`, `name`): `Promise`\<\{ `data`: `null`; `error`: `Error` \| `null`; \}\>

Delete a job function

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `namespace` | `string` | Namespace |
| `name` | `string` | Job function name |

#### Returns

`Promise`\<\{ `data`: `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple

#### Example

```typescript
const { data, error } = await client.admin.jobs.delete('default', 'process-data')
```

***

### get()

> **get**(`namespace`, `name`): `Promise`\<\{ `data`: `JobFunction` \| `null`; `error`: `Error` \| `null`; \}\>

Get details of a specific job function

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `namespace` | `string` | Namespace |
| `name` | `string` | Job function name |

#### Returns

`Promise`\<\{ `data`: `JobFunction` \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with job function metadata

#### Example

```typescript
const { data, error } = await client.admin.jobs.get('default', 'process-data')
if (data) {
  console.log('Job function version:', data.version)
}
```

***

### getJob()

> **getJob**(`jobId`): `Promise`\<\{ `data`: [`Job`](/api/sdk/interfaces/job/) \| `null`; `error`: `Error` \| `null`; \}\>

Get details of a specific job (execution)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `jobId` | `string` | Job ID |

#### Returns

`Promise`\<\{ `data`: [`Job`](/api/sdk/interfaces/job/) \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with job details

#### Example

```typescript
const { data, error } = await client.admin.jobs.getJob('550e8400-e29b-41d4-a716-446655440000')
if (data) {
  console.log(`Job ${data.job_name}: ${data.status}`)
}
```

***

### getStats()

> **getStats**(`namespace?`): `Promise`\<\{ `data`: `JobStats` \| `null`; `error`: `Error` \| `null`; \}\>

Get job statistics

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `namespace?` | `string` | Optional namespace filter |

#### Returns

`Promise`\<\{ `data`: `JobStats` \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with job stats

#### Example

```typescript
const { data, error } = await client.admin.jobs.getStats('default')
if (data) {
  console.log(`Pending: ${data.pending}, Running: ${data.running}`)
}
```

***

### list()

> **list**(`namespace?`): `Promise`\<\{ `data`: `JobFunction`[] \| `null`; `error`: `Error` \| `null`; \}\>

List all job functions (admin view)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `namespace?` | `string` | Optional namespace filter |

#### Returns

`Promise`\<\{ `data`: `JobFunction`[] \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with array of job functions

#### Example

```typescript
const { data, error } = await client.admin.jobs.list('default')
if (data) {
  console.log('Job functions:', data.map(f => f.name))
}
```

***

### listJobs()

> **listJobs**(`filters?`): `Promise`\<\{ `data`: [`Job`](/api/sdk/interfaces/job/)[] \| `null`; `error`: `Error` \| `null`; \}\>

List all jobs (executions) across all namespaces (admin view)

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
const { data, error } = await client.admin.jobs.listJobs({
  status: 'running',
  namespace: 'default',
  limit: 50
})
if (data) {
  data.forEach(job => {
    console.log(`${job.job_name}: ${job.status}`)
  })
}
```

***

### listNamespaces()

> **listNamespaces**(): `Promise`\<\{ `data`: `string`[] \| `null`; `error`: `Error` \| `null`; \}\>

List all namespaces that have job functions

#### Returns

`Promise`\<\{ `data`: `string`[] \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with array of namespace strings

#### Example

```typescript
const { data, error } = await client.admin.jobs.listNamespaces()
if (data) {
  console.log('Available namespaces:', data)
}
```

***

### listWorkers()

> **listWorkers**(): `Promise`\<\{ `data`: `JobWorker`[] \| `null`; `error`: `Error` \| `null`; \}\>

List active workers

#### Returns

`Promise`\<\{ `data`: `JobWorker`[] \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with array of workers

#### Example

```typescript
const { data, error } = await client.admin.jobs.listWorkers()
if (data) {
  data.forEach(worker => {
    console.log(`Worker ${worker.id}: ${worker.current_jobs} jobs`)
  })
}
```

***

### retry()

> **retry**(`jobId`): `Promise`\<\{ `data`: [`Job`](/api/sdk/interfaces/job/) \| `null`; `error`: `Error` \| `null`; \}\>

Retry a failed job

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `jobId` | `string` | Job ID |

#### Returns

`Promise`\<\{ `data`: [`Job`](/api/sdk/interfaces/job/) \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with new job

#### Example

```typescript
const { data, error } = await client.admin.jobs.retry('550e8400-e29b-41d4-a716-446655440000')
```

***

### sync()

> **sync**(`options`): `Promise`\<\{ `data`: `SyncJobsResult` \| `null`; `error`: `Error` \| `null`; \}\>

Sync multiple job functions to a namespace

Can sync from:
1. Filesystem (if no jobs provided) - loads from configured jobs directory
2. API payload (if jobs array provided) - syncs provided job specifications

Requires service_role or admin authentication.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `options` | `string` \| `SyncJobsOptions` | Sync options including namespace and optional jobs array |

#### Returns

`Promise`\<\{ `data`: `SyncJobsResult` \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with sync results

#### Example

```typescript
// Sync from filesystem
const { data, error } = await client.admin.jobs.sync({ namespace: 'default' })

// Sync with pre-bundled code (client-side bundling)
const bundled = await FluxbaseAdminJobs.bundleCode({ code: myJobCode })
const { data, error } = await client.admin.jobs.sync({
  namespace: 'default',
  functions: [{
    name: 'my-job',
    code: bundled.code,
    is_pre_bundled: true,
    original_code: myJobCode,
  }],
  options: {
    delete_missing: true, // Remove jobs not in this sync
    dry_run: false,       // Preview changes without applying
  }
})

if (data) {
  console.log(`Synced: ${data.summary.created} created, ${data.summary.updated} updated`)
}
```

***

### syncWithBundling()

> **syncWithBundling**(`options`, `bundleOptions?`): `Promise`\<\{ `data`: `SyncJobsResult` \| `null`; `error`: `Error` \| `null`; \}\>

Sync job functions with automatic client-side bundling

This is a convenience method that bundles all job code using esbuild
before sending to the server. Requires esbuild as a peer dependency.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `options` | `SyncJobsOptions` | Sync options including namespace and jobs array |
| `bundleOptions?` | `Partial`\<[`BundleOptions`](/api/sdk/interfaces/bundleoptions/)\> | Optional bundling configuration |

#### Returns

`Promise`\<\{ `data`: `SyncJobsResult` \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with sync results

#### Example

```typescript
const { data, error } = await client.admin.jobs.syncWithBundling({
  namespace: 'default',
  functions: [
    { name: 'process-data', code: processDataCode },
    { name: 'send-email', code: sendEmailCode },
  ],
  options: { delete_missing: true }
})
```

***

### terminate()

> **terminate**(`jobId`): `Promise`\<\{ `data`: `null`; `error`: `Error` \| `null`; \}\>

Terminate a running job immediately

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `jobId` | `string` | Job ID |

#### Returns

`Promise`\<\{ `data`: `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple

#### Example

```typescript
const { data, error } = await client.admin.jobs.terminate('550e8400-e29b-41d4-a716-446655440000')
```

***

### update()

> **update**(`namespace`, `name`, `updates`): `Promise`\<\{ `data`: `JobFunction` \| `null`; `error`: `Error` \| `null`; \}\>

Update an existing job function

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `namespace` | `string` | Namespace |
| `name` | `string` | Job function name |
| `updates` | \{ `code?`: `string`; `enabled?`: `boolean`; `timeout_seconds?`: `number`; \} | Fields to update |
| `updates.code?` | `string` | - |
| `updates.enabled?` | `boolean` | - |
| `updates.timeout_seconds?` | `number` | - |

#### Returns

`Promise`\<\{ `data`: `JobFunction` \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with updated job function

***

### bundleCode()

> `static` **bundleCode**(`options`): `Promise`\<[`BundleResult`](/api/sdk/interfaces/bundleresult/)\>

Bundle job code using esbuild (client-side)

Transforms and bundles TypeScript/JavaScript code into a single file
that can be executed by the Fluxbase jobs runtime.

Requires esbuild as a peer dependency.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `options` | [`BundleOptions`](/api/sdk/interfaces/bundleoptions/) | Bundle options including source code |

#### Returns

`Promise`\<[`BundleResult`](/api/sdk/interfaces/bundleresult/)\>

Promise resolving to bundled code

#### Throws

Error if esbuild is not available

#### Example

```typescript
const bundled = await FluxbaseAdminJobs.bundleCode({
  code: `
    import { helper } from './utils'
    export async function handler(req) {
      return helper(req.payload)
    }
  `,
  minify: true,
})

// Use bundled code in sync
await client.admin.jobs.sync({
  namespace: 'default',
  functions: [{
    name: 'my-job',
    code: bundled.code,
    is_pre_bundled: true,
  }]
})
```
