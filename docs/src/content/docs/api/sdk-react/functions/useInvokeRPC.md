---
editUrl: false
next: false
prev: false
title: "useInvokeRPC"
---

> **useInvokeRPC**(): `UseMutationResult`\<`RPCInvokeResponse`\<`unknown`\> \| `null`, `Error`, \{ `name`: `string`; `options?`: `RPCInvokeOptions`; `payload?`: `Record`\<`string`, `unknown`\>; \}, `unknown`\>

Hook to invoke an RPC procedure

## Returns

`UseMutationResult`\<`RPCInvokeResponse`\<`unknown`\> \| `null`, `Error`, \{ `name`: `string`; `options?`: `RPCInvokeOptions`; `payload?`: `Record`\<`string`, `unknown`\>; \}, `unknown`\>

## Example

```tsx
const { mutateAsync, data } = useInvokeRPC()
await mutateAsync({ name: 'get-user-orders', payload: { user_id: '123' } })
```
