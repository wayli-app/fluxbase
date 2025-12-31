---
editUrl: false
next: false
prev: false
title: "hasPostgrestError"
---

> **hasPostgrestError**\<`T`\>(`response`): `response is PostgrestResponse<T> & Object`

Type guard to check if a PostgrestResponse has an error

## Type Parameters

| Type Parameter |
| ------ |
| `T` |

## Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `response` | [`PostgrestResponse`](/api/sdk/interfaces/postgrestresponse/)\<`T`\> | The Postgrest response to check |

## Returns

`response is PostgrestResponse<T> & Object`

true if the response contains an error

## Example

```typescript
const response = await client.from('products').select('*').execute()

if (hasPostgrestError(response)) {
  // TypeScript knows: response.error is PostgrestError
  console.error('Query failed:', response.error.message)
  if (response.error.hint) {
    console.log('Hint:', response.error.hint)
  }
  return
}

// TypeScript knows: response.data is T (not null)
console.log('Found', response.data.length, 'products')
```
