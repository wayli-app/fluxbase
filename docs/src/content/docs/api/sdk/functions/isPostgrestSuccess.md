---
editUrl: false
next: false
prev: false
title: "isPostgrestSuccess"
---

> **isPostgrestSuccess**\<`T`\>(`response`): `response is PostgrestResponse<T> & Object`

Type guard to check if a PostgrestResponse is successful (has data)

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

true if the response has data and no error
