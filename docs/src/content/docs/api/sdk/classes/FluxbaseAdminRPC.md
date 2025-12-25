---
editUrl: false
next: false
prev: false
title: "FluxbaseAdminRPC"
---

Admin RPC manager for managing RPC procedures
Provides sync, CRUD, and execution monitoring operations

## Constructors

### new FluxbaseAdminRPC()

> **new FluxbaseAdminRPC**(`fetch`): [`FluxbaseAdminRPC`](/api/sdk/classes/fluxbaseadminrpc/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |

#### Returns

[`FluxbaseAdminRPC`](/api/sdk/classes/fluxbaseadminrpc/)

## Methods

### cancelExecution()

> **cancelExecution**(`executionId`): `Promise`\<`object`\>

Cancel a running execution

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `executionId` | `string` | Execution ID |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with updated execution

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`RPCExecution`](/api/sdk/interfaces/rpcexecution/) |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.rpc.cancelExecution('execution-uuid')
```

***

### delete()

> **delete**(`namespace`, `name`): `Promise`\<`object`\>

Delete an RPC procedure

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `namespace` | `string` | Procedure namespace |
| `name` | `string` | Procedure name |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple

| Name | Type |
| ------ | ------ |
| `data` | `null` |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.rpc.delete('default', 'get-user-orders')
```

***

### get()

> **get**(`namespace`, `name`): `Promise`\<`object`\>

Get details of a specific RPC procedure

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `namespace` | `string` | Procedure namespace |
| `name` | `string` | Procedure name |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with procedure details

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`RPCProcedure`](/api/sdk/interfaces/rpcprocedure/) |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.rpc.get('default', 'get-user-orders')
if (data) {
  console.log('Procedure:', data.name)
  console.log('SQL:', data.sql_query)
}
```

***

### getExecution()

> **getExecution**(`executionId`): `Promise`\<`object`\>

Get details of a specific execution

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `executionId` | `string` | Execution ID |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with execution details

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`RPCExecution`](/api/sdk/interfaces/rpcexecution/) |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.rpc.getExecution('execution-uuid')
if (data) {
  console.log('Status:', data.status)
  console.log('Duration:', data.duration_ms, 'ms')
}
```

***

### getExecutionLogs()

> **getExecutionLogs**(`executionId`, `afterLine`?): `Promise`\<`object`\>

Get execution logs for a specific execution

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `executionId` | `string` | Execution ID |
| `afterLine`? | `number` | Optional line number to get logs after (for polling) |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with execution logs

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`ExecutionLog`](/api/sdk/interfaces/executionlog/)[] |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.rpc.getExecutionLogs('execution-uuid')
if (data) {
  for (const log of data) {
    console.log(`[${log.level}] ${log.message}`)
  }
}
```

***

### list()

> **list**(`namespace`?): `Promise`\<`object`\>

List all RPC procedures (admin view)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `namespace`? | `string` | Optional namespace filter |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with array of procedure summaries

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`RPCProcedureSummary`](/api/sdk/interfaces/rpcproceduresummary/)[] |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.rpc.list()
if (data) {
  console.log('Procedures:', data.map(p => p.name))
}
```

***

### listExecutions()

> **listExecutions**(`filters`?): `Promise`\<`object`\>

List RPC executions with optional filters

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `filters`? | [`RPCExecutionFilters`](/api/sdk/interfaces/rpcexecutionfilters/) | Optional filters for namespace, procedure, status, user |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with array of executions

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`RPCExecution`](/api/sdk/interfaces/rpcexecution/)[] |
| `error` | `null` \| `Error` |

#### Example

```typescript
// List all executions
const { data, error } = await client.admin.rpc.listExecutions()

// List failed executions for a specific procedure
const { data, error } = await client.admin.rpc.listExecutions({
  namespace: 'default',
  procedure: 'get-user-orders',
  status: 'failed',
})
```

***

### listNamespaces()

> **listNamespaces**(): `Promise`\<`object`\>

List all namespaces

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with array of namespace names

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `string`[] |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.rpc.listNamespaces()
if (data) {
  console.log('Namespaces:', data)
}
```

***

### sync()

> **sync**(`options`?): `Promise`\<`object`\>

Sync RPC procedures from filesystem or API payload

Can sync from:
1. Filesystem (if no procedures provided) - loads from configured procedures directory
2. API payload (if procedures array provided) - syncs provided procedure specifications

Requires service_role or admin authentication.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `options`? | [`SyncRPCOptions`](/api/sdk/interfaces/syncrpcoptions/) | Sync options including namespace and optional procedures array |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with sync results

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`SyncRPCResult`](/api/sdk/interfaces/syncrpcresult/) |
| `error` | `null` \| `Error` |

#### Example

```typescript
// Sync from filesystem
const { data, error } = await client.admin.rpc.sync()

// Sync with provided procedure code
const { data, error } = await client.admin.rpc.sync({
  namespace: 'default',
  procedures: [{
    name: 'get-user-orders',
    code: myProcedureSQL,
  }],
  options: {
    delete_missing: false, // Don't remove procedures not in this sync
    dry_run: false,        // Preview changes without applying
  }
})

if (data) {
  console.log(`Synced: ${data.summary.created} created, ${data.summary.updated} updated`)
}
```

***

### toggle()

> **toggle**(`namespace`, `name`, `enabled`): `Promise`\<`object`\>

Enable or disable an RPC procedure

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `namespace` | `string` | Procedure namespace |
| `name` | `string` | Procedure name |
| `enabled` | `boolean` | Whether to enable or disable |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with updated procedure

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`RPCProcedure`](/api/sdk/interfaces/rpcprocedure/) |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.rpc.toggle('default', 'get-user-orders', true)
```

***

### update()

> **update**(`namespace`, `name`, `updates`): `Promise`\<`object`\>

Update an RPC procedure

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `namespace` | `string` | Procedure namespace |
| `name` | `string` | Procedure name |
| `updates` | [`UpdateRPCProcedureRequest`](/api/sdk/interfaces/updaterpcprocedurerequest/) | Fields to update |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with updated procedure

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`RPCProcedure`](/api/sdk/interfaces/rpcprocedure/) |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.rpc.update('default', 'get-user-orders', {
  enabled: false,
  max_execution_time_seconds: 60,
})
```
