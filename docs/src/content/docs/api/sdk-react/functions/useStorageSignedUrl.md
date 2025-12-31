---
editUrl: false
next: false
prev: false
title: "useStorageSignedUrl"
---

> **useStorageSignedUrl**(`bucket`, `path`, `expiresIn`?): `UseQueryResult`\<`null` \| `string`, `Error`\>

Hook to create a signed URL

:::caution[Deprecated]
Use useStorageSignedUrlWithOptions for more control including transforms
:::

## Parameters

| Parameter | Type |
| ------ | ------ |
| `bucket` | `string` |
| `path` | `null` \| `string` |
| `expiresIn`? | `number` |

## Returns

`UseQueryResult`\<`null` \| `string`, `Error`\>
