---
editUrl: false
next: false
prev: false
title: "FluxbaseBranching"
---

Branching client for database branch management

Database branches allow you to create isolated copies of your database
for development, testing, and preview environments.

## Constructors

### new FluxbaseBranching()

> **new FluxbaseBranching**(`fetch`): [`FluxbaseBranching`](/api/sdk/classes/fluxbasebranching/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |

#### Returns

[`FluxbaseBranching`](/api/sdk/classes/fluxbasebranching/)

## Methods

### create()

> **create**(`name`, `options`?): `Promise`\<`object`\>

Create a new database branch

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `name` | `string` | Branch name (will be converted to a slug) |
| `options`? | [`CreateBranchOptions`](/api/sdk/interfaces/createbranchoptions/) | Branch creation options |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with created branch

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`Branch`](/api/sdk/interfaces/branch/) |
| `error` | `null` \| `Error` |

#### Example

```typescript
// Create a simple branch
const { data, error } = await client.branching.create('feature/add-auth')

// Create with options
const { data } = await client.branching.create('feature/add-auth', {
  dataCloneMode: 'schema_only',  // Don't clone data
  expiresIn: '7d',               // Auto-delete after 7 days
  type: 'persistent'             // Won't auto-delete on PR merge
})

// Create a PR preview branch
const { data } = await client.branching.create('pr-123', {
  type: 'preview',
  githubPRNumber: 123,
  githubRepo: 'owner/repo',
  expiresIn: '7d'
})

// Clone with full data (for debugging)
const { data } = await client.branching.create('debug-issue-456', {
  dataCloneMode: 'full_clone'
})
```

***

### delete()

> **delete**(`idOrSlug`): `Promise`\<`object`\>

Delete a database branch

This permanently deletes the branch database and all its data.
Cannot delete the main branch.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `idOrSlug` | `string` | Branch ID (UUID) or slug |

#### Returns

`Promise`\<`object`\>

Promise resolving to { error } (null on success)

| Name | Type |
| ------ | ------ |
| `error` | `null` \| `Error` |

#### Example

```typescript
// Delete a branch
const { error } = await client.branching.delete('feature/add-auth')

if (error) {
  console.error('Failed to delete branch:', error.message)
}
```

***

### exists()

> **exists**(`idOrSlug`): `Promise`\<`boolean`\>

Check if a branch exists

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `idOrSlug` | `string` | Branch ID (UUID) or slug |

#### Returns

`Promise`\<`boolean`\>

Promise resolving to true if branch exists, false otherwise

#### Example

```typescript
const exists = await client.branching.exists('feature/add-auth')

if (!exists) {
  await client.branching.create('feature/add-auth')
}
```

***

### get()

> **get**(`idOrSlug`): `Promise`\<`object`\>

Get a specific branch by ID or slug

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `idOrSlug` | `string` | Branch ID (UUID) or slug |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with branch details

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`Branch`](/api/sdk/interfaces/branch/) |
| `error` | `null` \| `Error` |

#### Example

```typescript
// Get by slug
const { data, error } = await client.branching.get('feature/add-auth')

// Get by ID
const { data } = await client.branching.get('123e4567-e89b-12d3-a456-426614174000')
```

***

### getActivity()

> **getActivity**(`idOrSlug`, `limit`): `Promise`\<`object`\>

Get activity log for a branch

#### Parameters

| Parameter | Type | Default value | Description |
| ------ | ------ | ------ | ------ |
| `idOrSlug` | `string` | `undefined` | Branch ID (UUID) or slug |
| `limit` | `number` | `50` | Maximum number of entries to return (default: 50, max: 100) |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with activity entries

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`BranchActivity`](/api/sdk/interfaces/branchactivity/)[] |
| `error` | `null` \| `Error` |

#### Example

```typescript
// Get recent activity
const { data, error } = await client.branching.getActivity('feature/add-auth')

if (data) {
  for (const entry of data) {
    console.log(`${entry.action}: ${entry.status}`)
  }
}

// Get more entries
const { data } = await client.branching.getActivity('feature/add-auth', 100)
```

***

### getPoolStats()

> **getPoolStats**(): `Promise`\<`object`\>

Get connection pool statistics for all branches

This is useful for monitoring and debugging branch connections.

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with pool stats

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`BranchPoolStats`](/api/sdk/interfaces/branchpoolstats/)[] |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.branching.getPoolStats()

if (data) {
  for (const pool of data) {
    console.log(`${pool.slug}: ${pool.active_connections} active`)
  }
}
```

***

### list()

> **list**(`options`?): `Promise`\<`object`\>

List all database branches

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `options`? | [`ListBranchesOptions`](/api/sdk/interfaces/listbranchesoptions/) | Filter and pagination options |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with branches list

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`ListBranchesResponse`](/api/sdk/interfaces/listbranchesresponse/) |
| `error` | `null` \| `Error` |

#### Example

```typescript
// List all branches
const { data, error } = await client.branching.list()

// Filter by status
const { data } = await client.branching.list({ status: 'ready' })

// Filter by type
const { data } = await client.branching.list({ type: 'preview' })

// Only show my branches
const { data } = await client.branching.list({ mine: true })

// Pagination
const { data } = await client.branching.list({ limit: 10, offset: 20 })
```

***

### reset()

> **reset**(`idOrSlug`): `Promise`\<`object`\>

Reset a branch to its parent state

This drops and recreates the branch database, resetting all data
to match the parent branch. Cannot reset the main branch.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `idOrSlug` | `string` | Branch ID (UUID) or slug |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with reset branch

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`Branch`](/api/sdk/interfaces/branch/) |
| `error` | `null` \| `Error` |

#### Example

```typescript
// Reset a branch
const { data, error } = await client.branching.reset('feature/add-auth')

if (data) {
  console.log('Branch reset, status:', data.status)
}
```

***

### waitForReady()

> **waitForReady**(`idOrSlug`, `options`?): `Promise`\<`object`\>

Wait for a branch to be ready

Polls the branch status until it reaches 'ready' or an error state.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `idOrSlug` | `string` | Branch ID (UUID) or slug |
| `options`? | `object` | Polling options |
| `options.pollInterval`? | `number` | Poll interval in milliseconds (default: 1000) |
| `options.timeout`? | `number` | Timeout in milliseconds (default: 30000) |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with ready branch

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`Branch`](/api/sdk/interfaces/branch/) |
| `error` | `null` \| `Error` |

#### Example

```typescript
// Create branch and wait for it to be ready
const { data: branch } = await client.branching.create('feature/add-auth')

const { data: ready, error } = await client.branching.waitForReady(branch!.slug, {
  timeout: 60000,     // 60 seconds
  pollInterval: 1000  // Check every second
})

if (ready) {
  console.log('Branch is ready!')
}
```
