---
editUrl: false
next: false
prev: false
title: "VectorSearchOptions"
---

Options for vector search via the convenience endpoint

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `column` | `string` | Vector column to search |
| `filters?` | [`QueryFilter`](/api/sdk/interfaces/queryfilter/)[] | Additional filters to apply |
| `match_count?` | `number` | Maximum number of results |
| `match_threshold?` | `number` | Minimum similarity threshold (0-1 for cosine, varies for others) |
| `metric?` | [`VectorMetric`](/api/sdk/type-aliases/vectormetric/) | Distance metric to use |
| `query?` | `string` | Text query to search for (will be auto-embedded) |
| `select?` | `string` | Columns to select (default: all) |
| `table` | `string` | Table to search in |
| `vector?` | `number`[] | Direct vector input (alternative to text query) |
