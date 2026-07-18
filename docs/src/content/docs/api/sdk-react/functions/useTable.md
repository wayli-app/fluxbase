---
editUrl: false
next: false
prev: false
title: "useTable"
---

> **useTable**\<`T`\>(`table`, `buildQuery?`, `options?`): `UseQueryResult`\<`NoInfer`\<`T`[]\>, `Error`\>

Hook for table queries with a simpler API

## Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `T` | `any` |

## Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `table` | `string` | Table name |
| `buildQuery?` | (`query`) => `QueryBuilder`\<`T`\> | Optional function to build the query (e.g., add filters) |
| `options?` | `UseFluxbaseQueryOptions`\<`T`\> | Query options including a stable queryKey NOTE: When using buildQuery with filters, provide a custom queryKey that includes the filter values to ensure proper caching. |

## Returns

`UseQueryResult`\<`NoInfer`\<`T`[]\>, `Error`\>

## Example

```tsx
// Simple query - queryKey is auto-generated from table name
useTable('users')

// With filters - provide queryKey including filter values
useTable('users',
  (q) => q.eq('status', 'active'),
  { queryKey: ['users', 'active'] }
)
```
