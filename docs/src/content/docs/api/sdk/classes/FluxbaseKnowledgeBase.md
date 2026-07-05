---
editUrl: false
next: false
prev: false
title: "FluxbaseKnowledgeBase"
---

FluxbaseKnowledgeBase provides knowledge base management for RAG applications

## Example

```typescript
// List knowledge bases
const { data: kbs } = await client.knowledgeBase.list()

// Create a knowledge base
const { data: kb } = await client.knowledgeBase.create({
  name: 'Product Docs',
  description: 'Product documentation for RAG'
})

// Add a document
const { data } = await client.knowledgeBase.addDocument(kb.id, {
  title: 'Getting Started',
  content: 'Welcome to our product...'
})

// Search the knowledge base
const { data: results } = await client.knowledgeBase.search(kb.id, {
  query: 'How to get started?'
})
```

## Constructors

### Constructor

> **new FluxbaseKnowledgeBase**(`fetch`): `FluxbaseKnowledgeBase`

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |

#### Returns

`FluxbaseKnowledgeBase`

## Methods

### addDocument()

> **addDocument**(`kbId`, `request`): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`AddDocumentResponse`](/api/sdk/interfaces/adddocumentresponse/)\>\>

Add a document with text content

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `kbId` | `string` | Knowledge base ID |
| `request` | [`AddDocumentRequest`](/api/sdk/interfaces/adddocumentrequest/) | Document content and metadata |

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`AddDocumentResponse`](/api/sdk/interfaces/adddocumentresponse/)\>\>

#### Example

```typescript
const { data } = await client.knowledgeBase.addDocument(kbId, {
  title: 'API Reference',
  content: 'The API supports REST and GraphQL...',
  metadata: { category: 'reference' },
  tags: ['api', 'reference']
})
```

***

### create()

> **create**(`request`): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`KnowledgeBase`](/api/sdk/interfaces/knowledgebase/)\>\>

Create a new knowledge base

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `request` | [`CreateKnowledgeBaseRequest`](/api/sdk/interfaces/createknowledgebaserequest/) | Knowledge base configuration |

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`KnowledgeBase`](/api/sdk/interfaces/knowledgebase/)\>\>

#### Example

```typescript
const { data, error } = await client.knowledgeBase.create({
  name: 'Product Docs',
  description: 'Product documentation',
  embedding_model: 'text-embedding-3-small',
  chunk_size: 1000,
  chunk_overlap: 200
})
```

***

### delete()

> **delete**(`id`): `Promise`\<\{ `data`: `boolean`; `error`: `Error` \| `null`; \}\>

Delete a knowledge base

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Knowledge base ID |

#### Returns

`Promise`\<\{ `data`: `boolean`; `error`: `Error` \| `null`; \}\>

***

### deleteDocument()

> **deleteDocument**(`kbId`, `docId`): `Promise`\<\{ `data`: `boolean`; `error`: `Error` \| `null`; \}\>

Delete a document

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `kbId` | `string` | Knowledge base ID |
| `docId` | `string` | Document ID |

#### Returns

`Promise`\<\{ `data`: `boolean`; `error`: `Error` \| `null`; \}\>

***

### deleteDocumentsByFilter()

> **deleteDocumentsByFilter**(`kbId`, `filter`): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`DeleteDocumentsByFilterResponse`](/api/sdk/interfaces/deletedocumentsbyfilterresponse/)\>\>

Delete documents matching a filter (bulk operation)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `kbId` | `string` | Knowledge base ID |
| `filter` | [`DeleteDocumentsByFilterRequest`](/api/sdk/interfaces/deletedocumentsbyfilterrequest/) | Filter criteria (by tags and/or metadata) |

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`DeleteDocumentsByFilterResponse`](/api/sdk/interfaces/deletedocumentsbyfilterresponse/)\>\>

***

### get()

> **get**(`id`): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`KnowledgeBase`](/api/sdk/interfaces/knowledgebase/)\>\>

Get a knowledge base by ID

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Knowledge base ID |

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`KnowledgeBase`](/api/sdk/interfaces/knowledgebase/)\>\>

***

### getDocument()

> **getDocument**(`kbId`, `docId`): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`KnowledgeBaseDocument`](/api/sdk/interfaces/knowledgebasedocument/)\>\>

Get a single document

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `kbId` | `string` | Knowledge base ID |
| `docId` | `string` | Document ID |

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`KnowledgeBaseDocument`](/api/sdk/interfaces/knowledgebasedocument/)\>\>

***

### getEntityRelationships()

> **getEntityRelationships**(`kbId`, `entityId`): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`AIEntityRelationship`](/api/sdk/interfaces/aientityrelationship/)[]\>\>

Get relationships for an entity

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `kbId` | `string` | Knowledge base ID |
| `entityId` | `string` | Entity ID |

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`AIEntityRelationship`](/api/sdk/interfaces/aientityrelationship/)[]\>\>

***

### getKnowledgeGraph()

> **getKnowledgeGraph**(`kbId`): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`KnowledgeGraphData`](/api/sdk/interfaces/knowledgegraphdata/)\>\>

Get the full knowledge graph for a knowledge base

Returns all entities and relationships for visualization.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `kbId` | `string` | Knowledge base ID |

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`KnowledgeGraphData`](/api/sdk/interfaces/knowledgegraphdata/)\>\>

***

### list()

> **list**(): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`KnowledgeBaseSummary`](/api/sdk/interfaces/knowledgebasesummary/)[]\>\>

List all knowledge bases the user has access to

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`KnowledgeBaseSummary`](/api/sdk/interfaces/knowledgebasesummary/)[]\>\>

#### Example

```typescript
const { data, error } = await client.knowledgeBase.list()
```

***

### listDocuments()

> **listDocuments**(`kbId`): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`KnowledgeBaseDocument`](/api/sdk/interfaces/knowledgebasedocument/)[]\>\>

List documents in a knowledge base

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `kbId` | `string` | Knowledge base ID |

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`KnowledgeBaseDocument`](/api/sdk/interfaces/knowledgebasedocument/)[]\>\>

***

### listEntities()

> **listEntities**(`kbId`, `type?`): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`AIEntity`](/api/sdk/interfaces/aientity/)[]\>\>

List entities in a knowledge base

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `kbId` | `string` | Knowledge base ID |
| `type?` | [`AIEntityType`](/api/sdk/type-aliases/aientitytype/) | Optional entity type filter |

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`AIEntity`](/api/sdk/interfaces/aientity/)[]\>\>

***

### search()

> **search**(`kbId`, `request`): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`SearchKnowledgeBaseResponse`](/api/sdk/interfaces/searchknowledgebaseresponse/)\>\>

Search a knowledge base using semantic similarity

Automatically embeds the query text and returns matching chunks.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `kbId` | `string` | Knowledge base ID |
| `request` | [`SearchKnowledgeBaseRequest`](/api/sdk/interfaces/searchknowledgebaserequest/) | Search parameters |

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`SearchKnowledgeBaseResponse`](/api/sdk/interfaces/searchknowledgebaseresponse/)\>\>

#### Example

```typescript
const { data, error } = await client.knowledgeBase.search(kbId, {
  query: 'How to configure authentication?',
  max_chunks: 5,
  threshold: 0.8
})

data.results.forEach(result => {
  console.log(result.document_title, result.similarity)
  console.log(result.content)
})
```

***

### searchEntities()

> **searchEntities**(`kbId`, `query`): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`AIEntity`](/api/sdk/interfaces/aientity/)[]\>\>

Search entities by name

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `kbId` | `string` | Knowledge base ID |
| `query` | `string` | Search query |

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`AIEntity`](/api/sdk/interfaces/aientity/)[]\>\>

***

### update()

> **update**(`id`, `updates`): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`KnowledgeBase`](/api/sdk/interfaces/knowledgebase/)\>\>

Update a knowledge base

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Knowledge base ID |
| `updates` | [`UpdateKnowledgeBaseRequest`](/api/sdk/interfaces/updateknowledgebaserequest/) | Fields to update |

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`KnowledgeBase`](/api/sdk/interfaces/knowledgebase/)\>\>

***

### updateDocument()

> **updateDocument**(`kbId`, `docId`, `updates`): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`KnowledgeBaseDocument`](/api/sdk/interfaces/knowledgebasedocument/)\>\>

Update a document's metadata

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `kbId` | `string` | Knowledge base ID |
| `docId` | `string` | Document ID |
| `updates` | [`UpdateDocumentRequest`](/api/sdk/interfaces/updatedocumentrequest/) | Fields to update |

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`KnowledgeBaseDocument`](/api/sdk/interfaces/knowledgebasedocument/)\>\>

***

### uploadDocument()

> **uploadDocument**(`kbId`, `file`, `filename`, `metadata?`): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`UploadDocumentResponse`](/api/sdk/interfaces/uploaddocumentresponse/)\>\>

Upload a file as a document

Supports: PDF, TXT, MD, HTML, CSV, DOCX, XLSX, RTF, EPUB, JSON (max 50MB)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `kbId` | `string` | Knowledge base ID |
| `file` | `ArrayBuffer` \| `Blob` \| `File` | File to upload (File, Blob, or ArrayBuffer) |
| `filename` | `string` | Name of the file |
| `metadata?` | `Record`\<`string`, `string`\> | Optional metadata |

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`UploadDocumentResponse`](/api/sdk/interfaces/uploaddocumentresponse/)\>\>

#### Example

```typescript
const file = new File(['content'], 'guide.pdf', { type: 'application/pdf' })
const { data } = await client.knowledgeBase.uploadDocument(kbId, file, 'guide.pdf')
```
