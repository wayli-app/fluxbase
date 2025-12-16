---
editUrl: false
next: false
prev: false
title: "FluxbaseVector"
---

FluxbaseVector provides vector search functionality using pgvector

## Example

```typescript
// Embed text and search
const { data: results } = await client.vector.search({
  table: 'documents',
  column: 'embedding',
  query: 'How to use TypeScript?',
  match_count: 10
})

// Embed text directly
const { data: embedding } = await client.vector.embed({ text: 'Hello world' })
```

## Constructors

### new FluxbaseVector()

> **new FluxbaseVector**(`fetch`): [`FluxbaseVector`](/api/sdk/classes/fluxbasevector/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |

#### Returns

[`FluxbaseVector`](/api/sdk/classes/fluxbasevector/)

## Methods

### embed()

> **embed**(`request`): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`EmbedResponse`](/api/sdk/interfaces/embedresponse/)\>\>

Generate embeddings for text

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `request` | [`EmbedRequest`](/api/sdk/interfaces/embedrequest/) |

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`EmbedResponse`](/api/sdk/interfaces/embedresponse/)\>\>

#### Example

```typescript
// Single text
const { data } = await client.vector.embed({
  text: 'Hello world'
})
console.log(data.embeddings[0]) // [0.1, 0.2, ...]

// Multiple texts
const { data } = await client.vector.embed({
  texts: ['Hello', 'World'],
  model: 'text-embedding-3-small'
})
```

***

### search()

> **search**\<`T`\>(`options`): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`VectorSearchResult`](/api/sdk/interfaces/vectorsearchresult/)\<`T`\>\>\>

Search for similar vectors with automatic text embedding

This is a convenience method that:
1. Embeds the query text automatically (if `query` is provided)
2. Performs vector similarity search
3. Returns results with distance scores

#### Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `T` | `Record`\<`string`, `unknown`\> |

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `options` | [`VectorSearchOptions`](/api/sdk/interfaces/vectorsearchoptions/) |

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`VectorSearchResult`](/api/sdk/interfaces/vectorsearchresult/)\<`T`\>\>\>

#### Example

```typescript
// Search with text query (auto-embedded)
const { data } = await client.vector.search({
  table: 'documents',
  column: 'embedding',
  query: 'How to use TypeScript?',
  match_count: 10,
  match_threshold: 0.8
})

// Search with pre-computed vector
const { data } = await client.vector.search({
  table: 'documents',
  column: 'embedding',
  vector: [0.1, 0.2, ...],
  metric: 'cosine',
  match_count: 10
})

// With additional filters
const { data } = await client.vector.search({
  table: 'documents',
  column: 'embedding',
  query: 'TypeScript tutorial',
  filters: [
    { column: 'status', operator: 'eq', value: 'published' }
  ],
  match_count: 10
})
```
