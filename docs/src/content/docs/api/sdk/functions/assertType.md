---
editUrl: false
next: false
prev: false
title: "assertType"
---

> **assertType**\<`T`\>(`value`, `validator`, `errorMessage`): `asserts value is T`

Assert that a value is of type T, throwing if validation fails

## Type Parameters

| Type Parameter |
| ------ |
| `T` |

## Parameters

| Parameter | Type | Default value | Description |
| ------ | ------ | ------ | ------ |
| `value` | `unknown` | `undefined` | The value to assert |
| `validator` | (`v`) => `v is T` | `undefined` | A type guard function to validate the value |
| `errorMessage` | `string` | `'Type assertion failed'` | Optional custom error message |

## Returns

`asserts value is T`

## Throws

Error if validation fails

## Example

```typescript
const response = await client.functions.invoke('get-user')
assertType(response.data, isObject, 'Expected user object')
// Now response.data is typed as Record<string, unknown>
```
