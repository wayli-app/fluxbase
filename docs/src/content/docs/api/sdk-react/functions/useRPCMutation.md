---
editUrl: false
next: false
prev: false
title: "useRPCMutation"
---

> **useRPCMutation**\<`TData`, `TParams`\>(`functionName`, `options`?): `UseMutationResult`\<`TData`, `Error`, `TParams`, `unknown`\>

Hook to create a mutation for calling PostgreSQL functions
Useful for functions that modify data

## Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `TData` | `unknown` |
| `TParams` *extends* `Record`\<`string`, `unknown`\> | `Record`\<`string`, `unknown`\> |

## Parameters

| Parameter | Type |
| ------ | ------ |
| `functionName` | `string` |
| `options`? | `Omit`\<`UseMutationOptions`\<`TData`, `Error`, `TParams`, `unknown`\>, `"mutationFn"`\> |

## Returns

`UseMutationResult`\<`TData`, `Error`, `TParams`, `unknown`\>

## Example

```tsx
const createOrder = useRPCMutation('create_order')

const handleSubmit = async () => {
  await createOrder.mutateAsync({
    user_id: 123,
    items: [{ product_id: 1, quantity: 2 }]
  })
}
```
