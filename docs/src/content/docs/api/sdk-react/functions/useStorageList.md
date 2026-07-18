---
editUrl: false
next: false
prev: false
title: "useStorageList"
---

> **useStorageList**(`bucket`, `options?`): `UseQueryResult`\<`NoInfer`\<`any`[]\>, `Error`\>

Hook to list files in a bucket

## Parameters

| Parameter | Type |
| ------ | ------ |
| `bucket` | `string` |
| `options?` | `ListOptions` & `Omit`\<`UseQueryOptions`\<`any`[], `Error`, `any`[], readonly `unknown`[]\>, `"queryKey"` \| `"queryFn"`\> |

## Returns

`UseQueryResult`\<`NoInfer`\<`any`[]\>, `Error`\>
