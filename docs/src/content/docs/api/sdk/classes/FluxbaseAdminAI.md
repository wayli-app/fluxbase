---
editUrl: false
next: false
prev: false
title: "FluxbaseAdminAI"
---

Admin AI manager for managing AI chatbots and providers
Provides create, update, delete, sync, and monitoring operations

## Constructors

### new FluxbaseAdminAI()

> **new FluxbaseAdminAI**(`fetch`): [`FluxbaseAdminAI`](/api/sdk/classes/fluxbaseadminai/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |

#### Returns

[`FluxbaseAdminAI`](/api/sdk/classes/fluxbaseadminai/)

## Methods

### addDocument()

> **addDocument**(`knowledgeBaseId`, `request`): `Promise`\<`object`\>

Add a document to a knowledge base

Document will be chunked and embedded asynchronously.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `knowledgeBaseId` | `string` | Knowledge base ID |
| `request` | `AddDocumentRequest` | Document content and metadata |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with document ID

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `AddDocumentResponse` |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.ai.addDocument('kb-uuid', {
  title: 'Getting Started Guide',
  content: 'This is the content of the document...',
  metadata: { category: 'guides' },
})
if (data) {
  console.log('Document ID:', data.document_id)
}
```

***

### createKnowledgeBase()

> **createKnowledgeBase**(`request`): `Promise`\<`object`\>

Create a new knowledge base

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `request` | `CreateKnowledgeBaseRequest` | Knowledge base configuration |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with created knowledge base

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `KnowledgeBase` |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.ai.createKnowledgeBase({
  name: 'product-docs',
  description: 'Product documentation',
  chunk_size: 512,
  chunk_overlap: 50,
})
```

***

### createProvider()

> **createProvider**(`request`): `Promise`\<`object`\>

Create a new AI provider

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `request` | [`CreateAIProviderRequest`](/api/sdk/interfaces/createaiproviderrequest/) | Provider configuration |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with created provider

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`AIProvider`](/api/sdk/interfaces/aiprovider/) |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.ai.createProvider({
  name: 'openai-main',
  display_name: 'OpenAI (Main)',
  provider_type: 'openai',
  is_default: true,
  config: {
    api_key: 'sk-...',
    model: 'gpt-4-turbo',
  }
})
```

***

### deleteChatbot()

> **deleteChatbot**(`id`): `Promise`\<`object`\>

Delete a chatbot

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Chatbot ID |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple

| Name | Type |
| ------ | ------ |
| `data` | `null` |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.ai.deleteChatbot('uuid')
```

***

### deleteDocument()

> **deleteDocument**(`knowledgeBaseId`, `documentId`): `Promise`\<`object`\>

Delete a document from a knowledge base

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `knowledgeBaseId` | `string` | Knowledge base ID |
| `documentId` | `string` | Document ID |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple

| Name | Type |
| ------ | ------ |
| `data` | `null` |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.ai.deleteDocument('kb-uuid', 'doc-uuid')
```

***

### deleteKnowledgeBase()

> **deleteKnowledgeBase**(`id`): `Promise`\<`object`\>

Delete a knowledge base

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Knowledge base ID |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple

| Name | Type |
| ------ | ------ |
| `data` | `null` |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.ai.deleteKnowledgeBase('uuid')
```

***

### deleteProvider()

> **deleteProvider**(`id`): `Promise`\<`object`\>

Delete a provider

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Provider ID |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple

| Name | Type |
| ------ | ------ |
| `data` | `null` |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.ai.deleteProvider('uuid')
```

***

### getChatbot()

> **getChatbot**(`id`): `Promise`\<`object`\>

Get details of a specific chatbot

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Chatbot ID |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with chatbot details

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`AIChatbot`](/api/sdk/interfaces/aichatbot/) |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.ai.getChatbot('uuid')
if (data) {
  console.log('Chatbot:', data.name)
}
```

***

### getDocument()

> **getDocument**(`knowledgeBaseId`, `documentId`): `Promise`\<`object`\>

Get a specific document

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `knowledgeBaseId` | `string` | Knowledge base ID |
| `documentId` | `string` | Document ID |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with document details

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `KnowledgeBaseDocument` |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.ai.getDocument('kb-uuid', 'doc-uuid')
```

***

### getKnowledgeBase()

> **getKnowledgeBase**(`id`): `Promise`\<`object`\>

Get a specific knowledge base

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Knowledge base ID |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with knowledge base details

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `KnowledgeBase` |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.ai.getKnowledgeBase('uuid')
if (data) {
  console.log('Knowledge base:', data.name)
}
```

***

### getProvider()

> **getProvider**(`id`): `Promise`\<`object`\>

Get details of a specific provider

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Provider ID |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with provider details

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`AIProvider`](/api/sdk/interfaces/aiprovider/) |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.ai.getProvider('uuid')
if (data) {
  console.log('Provider:', data.display_name)
}
```

***

### linkKnowledgeBase()

> **linkKnowledgeBase**(`chatbotId`, `request`): `Promise`\<`object`\>

Link a knowledge base to a chatbot

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `chatbotId` | `string` | Chatbot ID |
| `request` | `LinkKnowledgeBaseRequest` | Link configuration |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with link details

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `ChatbotKnowledgeBaseLink` |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.ai.linkKnowledgeBase('chatbot-uuid', {
  knowledge_base_id: 'kb-uuid',
  priority: 1,
  max_chunks: 5,
  similarity_threshold: 0.7,
})
```

***

### listChatbotKnowledgeBases()

> **listChatbotKnowledgeBases**(`chatbotId`): `Promise`\<`object`\>

List knowledge bases linked to a chatbot

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `chatbotId` | `string` | Chatbot ID |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with linked knowledge bases

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `ChatbotKnowledgeBaseLink`[] |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.ai.listChatbotKnowledgeBases('chatbot-uuid')
if (data) {
  console.log('Linked KBs:', data.map(l => l.knowledge_base_id))
}
```

***

### listChatbots()

> **listChatbots**(`namespace`?): `Promise`\<`object`\>

List all chatbots (admin view)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `namespace`? | `string` | Optional namespace filter |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with array of chatbot summaries

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`AIChatbotSummary`](/api/sdk/interfaces/aichatbotsummary/)[] |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.ai.listChatbots()
if (data) {
  console.log('Chatbots:', data.map(c => c.name))
}
```

***

### listDocuments()

> **listDocuments**(`knowledgeBaseId`): `Promise`\<`object`\>

List documents in a knowledge base

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `knowledgeBaseId` | `string` | Knowledge base ID |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with array of documents

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `KnowledgeBaseDocument`[] |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.ai.listDocuments('kb-uuid')
if (data) {
  console.log('Documents:', data.map(d => d.title))
}
```

***

### listKnowledgeBases()

> **listKnowledgeBases**(): `Promise`\<`object`\>

List all knowledge bases

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with array of knowledge base summaries

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `KnowledgeBaseSummary`[] |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.ai.listKnowledgeBases()
if (data) {
  console.log('Knowledge bases:', data.map(kb => kb.name))
}
```

***

### listProviders()

> **listProviders**(): `Promise`\<`object`\>

List all AI providers

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with array of providers

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`AIProvider`](/api/sdk/interfaces/aiprovider/)[] |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.ai.listProviders()
if (data) {
  console.log('Providers:', data.map(p => p.name))
}
```

***

### searchKnowledgeBase()

> **searchKnowledgeBase**(`knowledgeBaseId`, `query`, `options`?): `Promise`\<`object`\>

Search a knowledge base

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `knowledgeBaseId` | `string` | Knowledge base ID |
| `query` | `string` | Search query |
| `options`? | `object` | Search options |
| `options.max_chunks`? | `number` | - |
| `options.threshold`? | `number` | - |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with search results

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `SearchKnowledgeBaseResponse` |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.ai.searchKnowledgeBase('kb-uuid', 'how to reset password', {
  max_chunks: 5,
  threshold: 0.7,
})
if (data) {
  console.log('Results:', data.results.map(r => r.content))
}
```

***

### setDefaultProvider()

> **setDefaultProvider**(`id`): `Promise`\<`object`\>

Set a provider as the default

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Provider ID |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with updated provider

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`AIProvider`](/api/sdk/interfaces/aiprovider/) |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.ai.setDefaultProvider('uuid')
```

***

### sync()

> **sync**(`options`?): `Promise`\<`object`\>

Sync chatbots from filesystem or API payload

Can sync from:
1. Filesystem (if no chatbots provided) - loads from configured chatbots directory
2. API payload (if chatbots array provided) - syncs provided chatbot specifications

Requires service_role or admin authentication.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `options`? | [`SyncChatbotsOptions`](/api/sdk/interfaces/syncchatbotsoptions/) | Sync options including namespace and optional chatbots array |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with sync results

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`SyncChatbotsResult`](/api/sdk/interfaces/syncchatbotsresult/) |
| `error` | `null` \| `Error` |

#### Example

```typescript
// Sync from filesystem
const { data, error } = await client.admin.ai.sync()

// Sync with provided chatbot code
const { data, error } = await client.admin.ai.sync({
  namespace: 'default',
  chatbots: [{
    name: 'sql-assistant',
    code: myChatbotCode,
  }],
  options: {
    delete_missing: false, // Don't remove chatbots not in this sync
    dry_run: false,        // Preview changes without applying
  }
})

if (data) {
  console.log(`Synced: ${data.summary.created} created, ${data.summary.updated} updated`)
}
```

***

### toggleChatbot()

> **toggleChatbot**(`id`, `enabled`): `Promise`\<`object`\>

Enable or disable a chatbot

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Chatbot ID |
| `enabled` | `boolean` | Whether to enable or disable |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with updated chatbot

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`AIChatbot`](/api/sdk/interfaces/aichatbot/) |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.ai.toggleChatbot('uuid', true)
```

***

### unlinkKnowledgeBase()

> **unlinkKnowledgeBase**(`chatbotId`, `knowledgeBaseId`): `Promise`\<`object`\>

Unlink a knowledge base from a chatbot

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `chatbotId` | `string` | Chatbot ID |
| `knowledgeBaseId` | `string` | Knowledge base ID |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple

| Name | Type |
| ------ | ------ |
| `data` | `null` |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.ai.unlinkKnowledgeBase('chatbot-uuid', 'kb-uuid')
```

***

### updateChatbotKnowledgeBase()

> **updateChatbotKnowledgeBase**(`chatbotId`, `knowledgeBaseId`, `updates`): `Promise`\<`object`\>

Update a chatbot-knowledge base link

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `chatbotId` | `string` | Chatbot ID |
| `knowledgeBaseId` | `string` | Knowledge base ID |
| `updates` | `UpdateChatbotKnowledgeBaseRequest` | Fields to update |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with updated link

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `ChatbotKnowledgeBaseLink` |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.ai.updateChatbotKnowledgeBase(
  'chatbot-uuid',
  'kb-uuid',
  { max_chunks: 10, enabled: true }
)
```

***

### updateKnowledgeBase()

> **updateKnowledgeBase**(`id`, `updates`): `Promise`\<`object`\>

Update an existing knowledge base

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Knowledge base ID |
| `updates` | `UpdateKnowledgeBaseRequest` | Fields to update |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with updated knowledge base

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `KnowledgeBase` |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.ai.updateKnowledgeBase('uuid', {
  description: 'Updated description',
  enabled: true,
})
```

***

### updateProvider()

> **updateProvider**(`id`, `updates`): `Promise`\<`object`\>

Update an existing AI provider

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Provider ID |
| `updates` | [`UpdateAIProviderRequest`](/api/sdk/interfaces/updateaiproviderrequest/) | Fields to update |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with updated provider

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`AIProvider`](/api/sdk/interfaces/aiprovider/) |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await client.admin.ai.updateProvider('uuid', {
  display_name: 'Updated Name',
  config: {
    api_key: 'new-key',
    model: 'gpt-4-turbo',
  },
  enabled: true,
})
```

***

### uploadDocument()

> **uploadDocument**(`knowledgeBaseId`, `file`, `title`?): `Promise`\<`object`\>

Upload a document file to a knowledge base

Supported file types: PDF, TXT, MD, HTML, CSV, DOCX, XLSX, RTF, EPUB, JSON
Maximum file size: 50MB

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `knowledgeBaseId` | `string` | Knowledge base ID |
| `file` | `Blob` \| `File` | File to upload (File or Blob) |
| `title`? | `string` | Optional document title (defaults to filename without extension) |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with upload result

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `UploadDocumentResponse` |
| `error` | `null` \| `Error` |

#### Example

```typescript
// Browser
const fileInput = document.getElementById('file') as HTMLInputElement
const file = fileInput.files?.[0]
if (file) {
  const { data, error } = await client.admin.ai.uploadDocument('kb-uuid', file)
  if (data) {
    console.log('Document ID:', data.document_id)
    console.log('Extracted length:', data.extracted_length)
  }
}

// Node.js (with node-fetch or similar)
import { Blob } from 'buffer'
const content = await fs.readFile('document.pdf')
const blob = new Blob([content], { type: 'application/pdf' })
const { data, error } = await client.admin.ai.uploadDocument('kb-uuid', blob, 'My Document')
```
