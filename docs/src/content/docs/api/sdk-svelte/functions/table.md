---
editUrl: false
next: false
prev: false
title: "table"
---

> **table**\<`T`\>(`tableName`, `buildQuery?`, `options?`): `CreateQueryResult`\<`T`[], `Error`\>

Reactive table read. Read with `$table(...)`.

## Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `T` | `any` |

## Parameters

| Parameter | Type |
| ------ | ------ |
| `tableName` | `string` |
| `buildQuery?` | (`query`) => [`QueryBuilder`](/api/sdk-svelte/interfaces/querybuilder/)\<`T`\> |
| `options?` | [`FluxbaseQueryOptions`](/api/sdk-svelte/interfaces/fluxbasequeryoptions/)\<`T`\> |

## Returns

`CreateQueryResult`\<`T`[], `Error`\>
