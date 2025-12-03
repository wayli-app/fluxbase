---
editUrl: false
next: false
prev: false
title: "useRPCBatch"
---

> **useRPCBatch**\<`TData`\>(`calls`, `options`?): `UseQueryResult`\<`TData`[], `Error`\>

Hook to call multiple RPC functions in parallel

## Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `TData` | `unknown` |

## Parameters

| Parameter | Type |
| ------ | ------ |
| `calls` | `object`[] |
| `options`? | `Omit`\<`UseQueryOptions`\<`TData`[], `Error`, `TData`[], readonly `unknown`[]\>, `"queryFn"` \| `"queryKey"`\> |

## Returns

`UseQueryResult`\<`TData`[], `Error`\>

## Example

```tsx
const { data, isLoading } = useRPCBatch([
  { name: 'get_user_stats', params: { user_id: 123 } },
  { name: 'get_recent_orders', params: { limit: 10 } },
])
```
