---
editUrl: false
next: false
prev: false
title: "FluxbaseClient"
---

Main Fluxbase client class

## Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `Database` | `any` |
| `_SchemaName` *extends* `string` & keyof `Database` | `any` |

## Constructors

### new FluxbaseClient()

> **new FluxbaseClient**\<`Database`, `_SchemaName`\>(`fluxbaseUrl`, `fluxbaseKey`, `options`?): [`FluxbaseClient`](/api/sdk/classes/fluxbaseclient/)\<`Database`, `_SchemaName`\>

Create a new Fluxbase client instance

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `fluxbaseUrl` | `string` | The URL of your Fluxbase instance |
| `fluxbaseKey` | `string` | The anon key (JWT token with "anon" role). Generate using scripts/generate-keys.sh |
| `options`? | [`FluxbaseClientOptions`](/api/sdk/interfaces/fluxbaseclientoptions/) | Additional client configuration options |

#### Returns

[`FluxbaseClient`](/api/sdk/classes/fluxbaseclient/)\<`Database`, `_SchemaName`\>

#### Example

```typescript
const client = new FluxbaseClient(
  'http://localhost:8080',
  'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...',  // Anon JWT token
  { timeout: 30000 }
)
```

## Advanced

### http

#### Get Signature

> **get** **http**(): [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/)

Get the internal HTTP client

Use this for advanced scenarios like making custom API calls or admin operations.

##### Example

```typescript
// Make a custom API call
const data = await client.http.get('/api/custom-endpoint')
```

##### Returns

[`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/)

The internal FluxbaseFetch instance

## Authentication

### getAuthToken()

> **getAuthToken**(): `null` \| `string`

Get the current authentication token

#### Returns

`null` \| `string`

The current JWT access token, or null if not authenticated

***

### setAuthToken()

> **setAuthToken**(`token`): `void`

Set a new authentication token

This updates both the HTTP client and realtime connection with the new token.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `token` | `null` \| `string` | The JWT access token to set, or null to clear authentication |

#### Returns

`void`

## Database

### from()

> **from**\<`T`\>(`table`): [`QueryBuilder`](/api/sdk/classes/querybuilder/)\<`T`\>

Create a query builder for a database table

#### Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `T` | `any` |

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `table` | `string` | The table name (can include schema, e.g., 'public.users') |

#### Returns

[`QueryBuilder`](/api/sdk/classes/querybuilder/)\<`T`\>

A query builder instance for constructing and executing queries

#### Example

```typescript
// Simple select
const { data } = await client.from('users').select('*').execute()

// With filters
const { data } = await client.from('products')
  .select('id, name, price')
  .gt('price', 100)
  .eq('category', 'electronics')
  .execute()

// Insert
await client.from('users').insert({ name: 'John', email: 'john@example.com' }).execute()
```

***

### schema()

> **schema**(`schemaName`): [`SchemaQueryBuilder`](/api/sdk/classes/schemaquerybuilder/)

Access a specific database schema

Use this to query tables in non-public schemas.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `schemaName` | `string` | The schema name (e.g., 'jobs', 'analytics') |

#### Returns

[`SchemaQueryBuilder`](/api/sdk/classes/schemaquerybuilder/)

A schema query builder for constructing queries on that schema

#### Example

```typescript
// Query the jobs.execution_logs table
const { data } = await client
  .schema('jobs')
  .from('execution_logs')
  .select('*')
  .eq('job_id', jobId)
  .execute()

// Insert into a custom schema table
await client
  .schema('analytics')
  .from('events')
  .insert({ event_type: 'click', data: {} })
  .execute()
```

## Other

### admin

> **admin**: [`FluxbaseAdmin`](/api/sdk/classes/fluxbaseadmin/)

Admin module for instance management (requires admin authentication)

***

### ai

> **ai**: [`FluxbaseAI`](/api/sdk/classes/fluxbaseai/)

AI module for chatbots and conversation history

***

### auth

> **auth**: [`FluxbaseAuth`](/api/sdk/classes/fluxbaseauth/)

Authentication module for user management

***

### functions

> **functions**: [`FluxbaseFunctions`](/api/sdk/classes/fluxbasefunctions/)

Functions module for invoking and managing edge functions

***

### jobs

> **jobs**: [`FluxbaseJobs`](/api/sdk/classes/fluxbasejobs/)

Jobs module for submitting and monitoring background jobs

***

### management

> **management**: [`FluxbaseManagement`](/api/sdk/classes/fluxbasemanagement/)

Management module for API keys, webhooks, and invitations

***

### realtime

> **realtime**: [`FluxbaseRealtime`](/api/sdk/classes/fluxbaserealtime/)

Realtime module for WebSocket subscriptions

***

### settings

> **settings**: [`SettingsClient`](/api/sdk/classes/settingsclient/)

Settings module for reading public application settings (respects RLS policies)

***

### storage

> **storage**: [`FluxbaseStorage`](/api/sdk/classes/fluxbasestorage/)

Storage module for file operations

## RPC

### rpc

> **rpc**: `CallableRPC`

RPC module for calling PostgreSQL functions - Supabase compatible

Can be called directly (Supabase-style) or access methods like invoke(), list(), getStatus()

#### Example

```typescript
// Supabase-style direct call (uses 'default' namespace)
const { data, error } = await client.rpc('get_user_orders', { user_id: '123' })

// With full options
const { data, error } = await client.rpc.invoke('get_user_orders', { user_id: '123' }, {
  namespace: 'custom',
  async: true
})

// List available procedures
const { data: procedures } = await client.rpc.list()
```

## Realtime

### channel()

> **channel**(`name`, `config`?): [`RealtimeChannel`](/api/sdk/classes/realtimechannel/)

Create or get a realtime channel (Supabase-compatible)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `name` | `string` | Channel name |
| `config`? | [`RealtimeChannelConfig`](/api/sdk/interfaces/realtimechannelconfig/) | Optional channel configuration |

#### Returns

[`RealtimeChannel`](/api/sdk/classes/realtimechannel/)

RealtimeChannel instance

#### Example

```typescript
const channel = client.channel('room-1', {
  broadcast: { self: true },
  presence: { key: 'user-123' }
})
  .on('broadcast', { event: 'message' }, (payload) => {
    console.log('Message:', payload)
  })
  .subscribe()
```

***

### removeChannel()

> **removeChannel**(`channel`): `Promise`\<`"error"` \| `"ok"`\>

Remove a realtime channel (Supabase-compatible)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `channel` | [`RealtimeChannel`](/api/sdk/classes/realtimechannel/) | The channel to remove |

#### Returns

`Promise`\<`"error"` \| `"ok"`\>

Promise resolving to status

#### Example

```typescript
const channel = client.channel('room-1')
await client.removeChannel(channel)
```
