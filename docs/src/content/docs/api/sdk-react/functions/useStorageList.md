---
editUrl: false
next: false
prev: false
title: "useStorageList"
---

> **useStorageList**(`bucket`, `options`?): `UseQueryResult`\<`any`[], `Error`\>

Hook to list files in a bucket

## Parameters

| Parameter | Type |
| ------ | ------ |
| `bucket` | `string` |
| `options`? | `ListOptions` & `Omit`\<`UseQueryOptions`\<`any`[], `Error`, `any`[], readonly `unknown`[]\>, `"queryFn"` \| `"queryKey"`\> |

## Returns

`UseQueryResult`\<`any`[], `Error`\>
