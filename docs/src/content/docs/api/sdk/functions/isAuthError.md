---
editUrl: false
next: false
prev: false
title: "isAuthError"
---

> **isAuthError**(`response`): `response is Object`

Type guard to check if an auth response is an error

## Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `response` | [`FluxbaseAuthResponse`](/api/sdk/type-aliases/fluxbaseauthresponse/) | The auth response to check |

## Returns

`response is Object`

true if the auth operation failed

## Example

```typescript
const result = await client.auth.signUp(credentials)

if (isAuthError(result)) {
  console.error('Sign up failed:', result.error.message)
  return
}

// TypeScript knows result.data contains user and session
console.log('Welcome,', result.data.user.email)
```
