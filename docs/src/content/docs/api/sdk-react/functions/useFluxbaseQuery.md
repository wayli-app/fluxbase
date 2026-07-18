---
editUrl: false
next: false
prev: false
title: "useFluxbaseQuery"
---

> **useFluxbaseQuery**\<`T`\>(`buildQuery`, `options?`): `UseQueryResult`\<`NoInfer`\<`T`[]\>, `Error`\>

Hook to execute a database query

## Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `T` | `any` |

## Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `buildQuery` | (`client`) => `QueryBuilder`\<`T`\> | Function that builds and returns the query |
| `options?` | `UseFluxbaseQueryOptions`\<`T`\> | React Query options IMPORTANT: You must provide a stable `queryKey` in options for proper caching. Without a custom queryKey, each render may create a new cache entry. |

## Returns

`UseQueryResult`\<`NoInfer`\<`T`[]\>, `Error`\>

## Example

```tsx
// Always provide a queryKey for stable caching
useFluxbaseQuery(
  (client) => client.from('users').select('*'),
  { queryKey: ['users', 'all'] }
)
```
