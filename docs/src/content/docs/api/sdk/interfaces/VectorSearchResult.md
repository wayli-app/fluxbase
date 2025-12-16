---
editUrl: false
next: false
prev: false
title: "VectorSearchResult"
---

Result from vector search

## Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `T` | `Record`\<`string`, `unknown`\> |

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `data` | `T`[] | Matched records |
| `distances` | `number`[] | Distance scores for each result |
| `model?` | `string` | Embedding model used (if query text was embedded) |
