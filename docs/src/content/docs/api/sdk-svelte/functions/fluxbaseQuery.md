---
editUrl: false
next: false
prev: false
title: "fluxbaseQuery"
---

> **fluxbaseQuery**\<`T`\>(`buildQuery`, `options?`): `CreateQueryResult`\<`T`[], `Error`\>

Reactive Fluxbase query. Read with `$fluxbaseQuery(...)`.

## Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `T` | `any` |

## Parameters

| Parameter | Type |
| ------ | ------ |
| `buildQuery` | (`client`) => [`QueryBuilder`](/api/sdk-svelte/interfaces/querybuilder/)\<`T`\> |
| `options?` | [`FluxbaseQueryOptions`](/api/sdk-svelte/interfaces/fluxbasequeryoptions/)\<`T`\> |

## Returns

`CreateQueryResult`\<`T`[], `Error`\>
