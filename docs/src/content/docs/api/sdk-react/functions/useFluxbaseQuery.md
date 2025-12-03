---
editUrl: false
next: false
prev: false
title: "useFluxbaseQuery"
---

> **useFluxbaseQuery**\<`T`\>(`buildQuery`, `options`?): `UseQueryResult`\<`T`[], `Error`\>

Hook to execute a database query

## Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `T` | `any` |

## Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `buildQuery` | (`client`) => `QueryBuilder`\<`T`\> | Function that builds and returns the query |
| `options`? | `UseFluxbaseQueryOptions`\<`T`\> | React Query options |

## Returns

`UseQueryResult`\<`T`[], `Error`\>
