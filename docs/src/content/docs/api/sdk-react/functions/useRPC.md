---
editUrl: false
next: false
prev: false
title: "useRPC"
---

> **useRPC**\<`TData`, `TParams`\>(`functionName`, `params`?, `options`?): `UseQueryResult`\<`NoInfer`\<`TData`\>, `Error`\>

Hook to call a PostgreSQL function and cache the result

## Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `TData` | `unknown` |
| `TParams` *extends* `Record`\<`string`, `unknown`\> | `Record`\<`string`, `unknown`\> |

## Parameters

| Parameter | Type |
| ------ | ------ |
| `functionName` | `string` |
| `params`? | `TParams` |
| `options`? | `Omit`\<`UseQueryOptions`\<`TData`, `Error`, `TData`, readonly `unknown`[]\>, `"queryFn"` \| `"queryKey"`\> |

## Returns

`UseQueryResult`\<`NoInfer`\<`TData`\>, `Error`\>

## Example

```tsx
const { data, isLoading, error } = useRPC(
  'calculate_total',
  { order_id: 123 },
  { enabled: !!orderId }
)
```
