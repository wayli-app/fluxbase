---
editUrl: false
next: false
prev: false
title: "SchemaQueryBuilder"
---

## Constructors

### new SchemaQueryBuilder()

> **new SchemaQueryBuilder**(`fetch`, `schemaName`): [`SchemaQueryBuilder`](/api/sdk/classes/schemaquerybuilder/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |
| `schemaName` | `string` |

#### Returns

[`SchemaQueryBuilder`](/api/sdk/classes/schemaquerybuilder/)

## Methods

### from()

> **from**\<`T`\>(`table`): [`QueryBuilder`](/api/sdk/classes/querybuilder/)\<`T`\>

Create a query builder for a table in this schema

#### Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `T` | `unknown` |

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `table` | `string` | The table name (without schema prefix) |

#### Returns

[`QueryBuilder`](/api/sdk/classes/querybuilder/)\<`T`\>

A query builder instance for constructing and executing queries
