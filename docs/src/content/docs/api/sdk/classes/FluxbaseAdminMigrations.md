---
editUrl: false
next: false
prev: false
title: "FluxbaseAdminMigrations"
---

Admin Migrations manager for database migration operations
Provides create, update, delete, apply, rollback, and smart sync operations

## Constructors

### new FluxbaseAdminMigrations()

> **new FluxbaseAdminMigrations**(`fetch`): [`FluxbaseAdminMigrations`](/api/sdk/classes/fluxbaseadminmigrations/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |

#### Returns

[`FluxbaseAdminMigrations`](/api/sdk/classes/fluxbaseadminmigrations/)

## Methods

### apply()

> **apply**(`name`, `namespace`): `Promise`\<`object`\>

Apply a specific migration

#### Parameters

| Parameter | Type | Default value | Description |
| ------ | ------ | ------ | ------ |
| `name` | `string` | `undefined` | Migration name |
| `namespace` | `string` | `'default'` | Migration namespace (default: 'default') |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with result message

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `object` |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.migrations.apply('001_create_users', 'myapp')
if (data) {
  console.log(data.message) // "Migration applied successfully"
}
```

***

### applyPending()

> **applyPending**(`namespace`): `Promise`\<`object`\>

Apply all pending migrations in order

#### Parameters

| Parameter | Type | Default value | Description |
| ------ | ------ | ------ | ------ |
| `namespace` | `string` | `'default'` | Migration namespace (default: 'default') |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with applied/failed counts

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `object` |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.migrations.applyPending('myapp')
if (data) {
  console.log(`Applied: ${data.applied.length}, Failed: ${data.failed.length}`)
}
```

***

### create()

> **create**(`request`): `Promise`\<`object`\>

Create a new migration

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `request` | [`CreateMigrationRequest`](/api/sdk/interfaces/createmigrationrequest/) | Migration configuration |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with created migration

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`Migration`](/api/sdk/interfaces/migration/) |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.migrations.create({
  namespace: 'myapp',
  name: '001_create_users',
  up_sql: 'CREATE TABLE app.users (id UUID PRIMARY KEY, email TEXT)',
  down_sql: 'DROP TABLE app.users',
  description: 'Create users table'
})
```

***

### delete()

> **delete**(`name`, `namespace`): `Promise`\<`object`\>

Delete a migration (only if status is pending)

#### Parameters

| Parameter | Type | Default value | Description |
| ------ | ------ | ------ | ------ |
| `name` | `string` | `undefined` | Migration name |
| `namespace` | `string` | `'default'` | Migration namespace (default: 'default') |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple

| Name | Type |
| ------ | ------ |
| `data` | `null` |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.migrations.delete('001_create_users', 'myapp')
```

***

### get()

> **get**(`name`, `namespace`): `Promise`\<`object`\>

Get details of a specific migration

#### Parameters

| Parameter | Type | Default value | Description |
| ------ | ------ | ------ | ------ |
| `name` | `string` | `undefined` | Migration name |
| `namespace` | `string` | `'default'` | Migration namespace (default: 'default') |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with migration details

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`Migration`](/api/sdk/interfaces/migration/) |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.migrations.get('001_create_users', 'myapp')
```

***

### getExecutions()

> **getExecutions**(`name`, `namespace`, `limit`): `Promise`\<`object`\>

Get execution history for a migration

#### Parameters

| Parameter | Type | Default value | Description |
| ------ | ------ | ------ | ------ |
| `name` | `string` | `undefined` | Migration name |
| `namespace` | `string` | `'default'` | Migration namespace (default: 'default') |
| `limit` | `number` | `50` | Maximum number of executions to return (default: 50, max: 100) |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with execution records

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`MigrationExecution`](/api/sdk/interfaces/migrationexecution/)[] |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.migrations.getExecutions(
  '001_create_users',
  'myapp',
  10
)
if (data) {
  data.forEach(exec => {
    console.log(`${exec.executed_at}: ${exec.action} - ${exec.status}`)
  })
}
```

***

### list()

> **list**(`namespace`, `status`?): `Promise`\<`object`\>

List migrations in a namespace

#### Parameters

| Parameter | Type | Default value | Description |
| ------ | ------ | ------ | ------ |
| `namespace` | `string` | `'default'` | Migration namespace (default: 'default') |
| `status`? | `"pending"` \| `"failed"` \| `"applied"` \| `"rolled_back"` | `undefined` | Filter by status: 'pending', 'applied', 'failed', 'rolled_back' |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with migrations array

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`Migration`](/api/sdk/interfaces/migration/)[] |
| `error` | `null` \| `Error` |

#### Example

```typescript
// List all migrations
const { data, error } = await client.admin.migrations.list('myapp')

// List only pending migrations
const { data, error } = await client.admin.migrations.list('myapp', 'pending')
```

***

### register()

> **register**(`migration`): `object`

Register a migration locally for smart sync

Call this method to register migrations in your application code.
When you call sync(), only new or changed migrations will be sent to the server.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `migration` | [`CreateMigrationRequest`](/api/sdk/interfaces/createmigrationrequest/) | Migration definition |

#### Returns

`object`

tuple (always succeeds unless validation fails)

| Name | Type |
| ------ | ------ |
| `error` | `null` \| `Error` |

#### Example

```typescript
// In your app initialization
const { error: err1 } = client.admin.migrations.register({
  name: '001_create_users_table',
  namespace: 'myapp',
  up_sql: 'CREATE TABLE app.users (...)',
  down_sql: 'DROP TABLE app.users',
  description: 'Initial users table'
})

const { error: err2 } = client.admin.migrations.register({
  name: '002_add_posts_table',
  namespace: 'myapp',
  up_sql: 'CREATE TABLE app.posts (...)',
  down_sql: 'DROP TABLE app.posts'
})

// Sync all registered migrations
await client.admin.migrations.sync()
```

***

### rollback()

> **rollback**(`name`, `namespace`): `Promise`\<`object`\>

Rollback a specific migration

#### Parameters

| Parameter | Type | Default value | Description |
| ------ | ------ | ------ | ------ |
| `name` | `string` | `undefined` | Migration name |
| `namespace` | `string` | `'default'` | Migration namespace (default: 'default') |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with result message

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `object` |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.migrations.rollback('001_create_users', 'myapp')
```

***

### sync()

> **sync**(`options`): `Promise`\<`object`\>

Smart sync all registered migrations

Automatically determines which migrations need to be created or updated by:
1. Fetching existing migrations from the server
2. Comparing content hashes to detect changes
3. Only sending new or changed migrations

After successful sync, can optionally auto-apply new migrations and refresh
the server's schema cache.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `options` | `Partial`\<[`SyncMigrationsOptions`](/api/sdk/interfaces/syncmigrationsoptions/)\> | Sync options |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with sync results

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`SyncMigrationsResult`](/api/sdk/interfaces/syncmigrationsresult/) |
| `error` | `null` \| `Error` |

#### Example

```typescript
// Basic sync (idempotent - safe to call on every app startup)
const { data, error } = await client.admin.migrations.sync()
if (data) {
  console.log(`Created: ${data.summary.created}, Updated: ${data.summary.updated}`)
}

// Sync with auto-apply (applies new migrations automatically)
const { data, error } = await client.admin.migrations.sync({
  auto_apply: true
})

// Dry run to preview changes without applying
const { data, error } = await client.admin.migrations.sync({
  dry_run: true
})
```

***

### update()

> **update**(`name`, `updates`, `namespace`): `Promise`\<`object`\>

Update a migration (only if status is pending)

#### Parameters

| Parameter | Type | Default value | Description |
| ------ | ------ | ------ | ------ |
| `name` | `string` | `undefined` | Migration name |
| `updates` | [`UpdateMigrationRequest`](/api/sdk/interfaces/updatemigrationrequest/) | `undefined` | Fields to update |
| `namespace` | `string` | `'default'` | Migration namespace (default: 'default') |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with updated migration

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`Migration`](/api/sdk/interfaces/migration/) |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.migrations.update(
  '001_create_users',
  { description: 'Updated description' },
  'myapp'
)
```
