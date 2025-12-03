---
editUrl: false
next: false
prev: false
title: "FluxbaseAdminJobs"
---

Admin Jobs manager for managing background job functions
Provides create, update, delete, sync, and monitoring operations

## Constructors

### new FluxbaseAdminJobs()

> **new FluxbaseAdminJobs**(`fetch`): [`FluxbaseAdminJobs`](/api/sdk/classes/fluxbaseadminjobs/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |

#### Returns

[`FluxbaseAdminJobs`](/api/sdk/classes/fluxbaseadminjobs/)

## Methods

### cancel()

> **cancel**(`jobId`): `Promise`\<`object`\>

Cancel a running or pending job

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `jobId` | `string` | Job ID |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple

| Name | Type |
| ------ | ------ |
| `data` | `null` |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.jobs.cancel('550e8400-e29b-41d4-a716-446655440000')
```

***

### create()

> **create**(`request`): `Promise`\<`object`\>

Create a new job function

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `request` | `CreateJobFunctionRequest` | Job function configuration and code |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with created job function metadata

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `JobFunction` |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.jobs.create({
  name: 'process-data',
  code: 'export async function handler(req) { return { success: true } }',
  enabled: true,
  timeout_seconds: 300
})
```

***

### delete()

> **delete**(`namespace`, `name`): `Promise`\<`object`\>

Delete a job function

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `namespace` | `string` | Namespace |
| `name` | `string` | Job function name |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple

| Name | Type |
| ------ | ------ |
| `data` | `null` |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.jobs.delete('default', 'process-data')
```

***

### get()

> **get**(`namespace`, `name`): `Promise`\<`object`\>

Get details of a specific job function

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `namespace` | `string` | Namespace |
| `name` | `string` | Job function name |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with job function metadata

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `JobFunction` |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.jobs.get('default', 'process-data')
if (data) {
  console.log('Job function version:', data.version)
}
```

***

### getJob()

> **getJob**(`jobId`): `Promise`\<`object`\>

Get details of a specific job (execution)

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
const { data, error } = await client.admin.jobs.getJob('550e8400-e29b-41d4-a716-446655440000')
if (data) {
  console.log(`Job ${data.job_name}: ${data.status}`)
}
```

***

### getStats()

> **getStats**(`namespace`?): `Promise`\<`object`\>

Get job statistics

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `namespace`? | `string` | Optional namespace filter |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with job stats

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `JobStats` |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.jobs.getStats('default')
if (data) {
  console.log(`Pending: ${data.pending}, Running: ${data.running}`)
}
```

***

### list()

> **list**(`namespace`?): `Promise`\<`object`\>

List all job functions (admin view)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `namespace`? | `string` | Optional namespace filter |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with array of job functions

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `JobFunction`[] |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.jobs.list('default')
if (data) {
  console.log('Job functions:', data.map(f => f.name))
}
```

***

### listJobs()

> **listJobs**(`filters`?): `Promise`\<`object`\>

List all jobs (executions) across all namespaces (admin view)

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

> **listNamespaces**(): `Promise`\<`object`\>

List all namespaces that have job functions

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with array of namespace strings

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `string`[] |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.jobs.listNamespaces()
if (data) {
  console.log('Available namespaces:', data)
}
```

***

### listWorkers()

> **listWorkers**(): `Promise`\<`object`\>

List active workers

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with array of workers

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `JobWorker`[] |
| `error` | `null` \| `Error` |

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

> **retry**(`jobId`): `Promise`\<`object`\>

Retry a failed job

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `jobId` | `string` | Job ID |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with new job

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `Job` |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.jobs.retry('550e8400-e29b-41d4-a716-446655440000')
```

***

### sync()

> **sync**(`options`): `Promise`\<`object`\>

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

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with sync results

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `SyncJobsResult` |
| `error` | `null` \| `Error` |

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

> **syncWithBundling**(`options`, `bundleOptions`?): `Promise`\<`object`\>

Sync job functions with automatic client-side bundling

This is a convenience method that bundles all job code using esbuild
before sending to the server. Requires esbuild as a peer dependency.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `options` | `SyncJobsOptions` | Sync options including namespace and jobs array |
| `bundleOptions`? | `Partial`\<[`BundleOptions`](/api/sdk/interfaces/bundleoptions/)\> | Optional bundling configuration |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with sync results

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `SyncJobsResult` |
| `error` | `null` \| `Error` |

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

> **terminate**(`jobId`): `Promise`\<`object`\>

Terminate a running job immediately

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `jobId` | `string` | Job ID |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple

| Name | Type |
| ------ | ------ |
| `data` | `null` |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.jobs.terminate('550e8400-e29b-41d4-a716-446655440000')
```

***

### update()

> **update**(`namespace`, `name`, `updates`): `Promise`\<`object`\>

Update an existing job function

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `namespace` | `string` | Namespace |
| `name` | `string` | Job function name |
| `updates` | `UpdateJobFunctionRequest` | Fields to update |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with updated job function metadata

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `JobFunction` |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.jobs.update('default', 'process-data', {
  enabled: false,
  timeout_seconds: 600
})
```

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
