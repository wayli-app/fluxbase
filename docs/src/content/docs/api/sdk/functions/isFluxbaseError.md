---
editUrl: false
next: false
prev: false
title: "isFluxbaseError"
---

> **isFluxbaseError**\<`T`\>(`response`): `response is Object`

Type guard to check if a FluxbaseResponse is an error response

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

true if the response is an error (data is null, error is not null)

## Example

```typescript
const result = await client.auth.signIn(credentials)

if (isFluxbaseError(result)) {
  // TypeScript knows: result.error is Error, result.data is null
  console.error('Sign in failed:', result.error.message)
  return
}

// TypeScript knows: result.data is T, result.error is null
console.log('Signed in as:', result.data.user.email)
```
