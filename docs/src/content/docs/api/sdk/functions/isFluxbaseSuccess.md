---
editUrl: false
next: false
prev: false
title: "isFluxbaseSuccess"
---

> **isFluxbaseSuccess**\<`T`\>(`response`): `response is Object`

Type guard to check if a FluxbaseResponse is a success response

## Type Parameters

| Type Parameter |
| ------ |
| `T` |

## Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `response` | [`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<`T`\> | The response to check |

## Returns

`response is Object`

true if the response is successful (data is not null, error is null)

## Example

```typescript
const result = await client.from('users').select('*').execute()

if (isFluxbaseSuccess(result)) {
  // TypeScript knows: result.data is T, result.error is null
  result.data.forEach(user => console.log(user.name))
}
```
