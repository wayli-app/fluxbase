---
editUrl: false
next: false
prev: false
title: "useTable"
---

> **useTable**\<`T`\>(`table`, `buildQuery`?, `options`?): `UseQueryResult`\<`T`[], `Error`\>

Hook for table queries with a simpler API

## Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `T` | `any` |

## Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `table` | `string` | Table name |
| `buildQuery`? | (`query`) => `QueryBuilder`\<`T`\> | Function to build the query |
| `options`? | `UseFluxbaseQueryOptions`\<`T`\> | - |

## Returns

`UseQueryResult`\<`T`[], `Error`\>
