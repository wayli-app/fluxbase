---
editUrl: false
next: false
prev: false
title: "FluxbaseFunctions"
---

Edge Functions client for invoking serverless functions
API-compatible with Supabase Functions

For admin operations (create, update, delete, sync), use client.admin.functions

## Constructors

### new FluxbaseFunctions()

> **new FluxbaseFunctions**(`fetch`): [`FluxbaseFunctions`](/api/sdk/classes/fluxbasefunctions/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |

#### Returns

[`FluxbaseFunctions`](/api/sdk/classes/fluxbasefunctions/)

## Methods

### get()

> **get**(`name`): `Promise`\<`object`\>

Get details of a specific edge function

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `name` | `string` | Function name |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with function metadata

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`EdgeFunction`](/api/sdk/interfaces/edgefunction/) |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.functions.get('my-function')
if (data) {
  console.log('Function version:', data.version)
}
```

***

### invoke()

> **invoke**\<`T`\>(`functionName`, `options`?): `Promise`\<`object`\>

Invoke an edge function

This method is fully compatible with Supabase's functions.invoke() API.

#### Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `T` | `unknown` |

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `functionName` | `string` | The name of the function to invoke |
| `options`? | [`FunctionInvokeOptions`](/api/sdk/interfaces/functioninvokeoptions/) | Invocation options including body, headers, HTTP method, and namespace |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `T` |
| `error` | `null` \| `Error` |

#### Example

```typescript
// Simple invocation (uses first matching function by namespace alphabetically)
const { data, error } = await client.functions.invoke('hello', {
  body: { name: 'World' }
})

// Invoke a specific namespace's function
const { data, error } = await client.functions.invoke('hello', {
  body: { name: 'World' },
  namespace: 'my-app'
})

// With GET method
const { data, error } = await client.functions.invoke('get-data', {
  method: 'GET'
})

// With custom headers
const { data, error } = await client.functions.invoke('api-proxy', {
  body: { query: 'search' },
  headers: { 'Authorization': 'Bearer token' },
  method: 'POST'
})
```

***

### list()

> **list**(): `Promise`\<`object`\>

List all public edge functions

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with array of public functions

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`EdgeFunction`](/api/sdk/interfaces/edgefunction/)[] |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.functions.list()
if (data) {
  console.log('Functions:', data.map(f => f.name))
}
```
