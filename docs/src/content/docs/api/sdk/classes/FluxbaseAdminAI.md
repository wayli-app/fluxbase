---
editUrl: false
next: false
prev: false
title: "FluxbaseAdminAI"
---

Admin AI manager for managing AI chatbots and providers
Provides create, update, delete, sync, and monitoring operations

## Constructors

### Constructor

> **new FluxbaseAdminAI**(`fetch`): `FluxbaseAdminAI`

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |

#### Returns

`FluxbaseAdminAI`

## Methods

### clearEmbeddingProvider()

> **clearEmbeddingProvider**(`id`): `Promise`\<\{ `data`: \{ `use_for_embeddings`: `boolean`; \} \| `null`; `error`: `Error` \| `null`; \}\>

Clear the embedding provider assignment for a provider

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Provider ID |

#### Returns

`Promise`\<\{ `data`: \{ `use_for_embeddings`: `boolean`; \} \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple

***

### createProvider()

> **createProvider**(`params`): `Promise`\<\{ `data`: [`AIProvider`](/api/sdk/interfaces/aiprovider/) \| `null`; `error`: `Error` \| `null`; \}\>

Create a new AI provider

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `params` | \{ `config?`: `Record`\<`string`, `unknown`\>; `display_name?`: `string`; `is_default?`: `boolean`; `name`: `string`; `provider_type`: `string`; \} | Provider configuration including name, provider_type, and optional config |
| `params.config?` | `Record`\<`string`, `unknown`\> | - |
| `params.display_name?` | `string` | - |
| `params.is_default?` | `boolean` | - |
| `params.name` | `string` | - |
| `params.provider_type` | `string` | - |

#### Returns

`Promise`\<\{ `data`: [`AIProvider`](/api/sdk/interfaces/aiprovider/) \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with created provider

***

### deleteChatbot()

> **deleteChatbot**(`id`): `Promise`\<\{ `data`: `null`; `error`: `Error` \| `null`; \}\>

Delete a chatbot

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Chatbot ID |

#### Returns

`Promise`\<\{ `data`: `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple

#### Example

```typescript
const { data, error } = await client.admin.ai.deleteChatbot('uuid')
```

***

### deleteProvider()

> **deleteProvider**(`id`): `Promise`\<\{ `data`: `null`; `error`: `Error` \| `null`; \}\>

Delete an AI provider

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Provider ID |

#### Returns

`Promise`\<\{ `data`: `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple

***

### getChatbot()

> **getChatbot**(`id`): `Promise`\<\{ `data`: [`AIChatbot`](/api/sdk/interfaces/aichatbot/) \| `null`; `error`: `Error` \| `null`; \}\>

Get details of a specific chatbot

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Chatbot ID |

#### Returns

`Promise`\<\{ `data`: [`AIChatbot`](/api/sdk/interfaces/aichatbot/) \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with chatbot details

#### Example

```typescript
const { data, error } = await client.admin.ai.getChatbot('uuid')
if (data) {
  console.log('Chatbot:', data.name)
}
```

***

### getProvider()

> **getProvider**(`id`): `Promise`\<\{ `data`: [`AIProvider`](/api/sdk/interfaces/aiprovider/) \| `null`; `error`: `Error` \| `null`; \}\>

Get details of a specific AI provider

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Provider ID |

#### Returns

`Promise`\<\{ `data`: [`AIProvider`](/api/sdk/interfaces/aiprovider/) \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with provider details

***

### getTableDetails()

> **getTableDetails**(`schema`, `table`): `Promise`\<\{ `data`: [`TableDetails`](/api/sdk/interfaces/tabledetails/) \| `null`; `error`: `Error` \| `null`; \}\>

Get detailed table information including columns

Use this to discover available columns before exporting.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `schema` | `string` | Schema name (e.g., 'public') |
| `table` | `string` | Table name |

#### Returns

`Promise`\<\{ `data`: [`TableDetails`](/api/sdk/interfaces/tabledetails/) \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with table details

#### Example

```typescript
const { data, error } = await client.admin.ai.getTableDetails('public', 'users')
if (data) {
  console.log('Columns:', data.columns.map(c => c.name))
  console.log('Primary key:', data.primary_key)
}
```

***

### linkKnowledgeBase()

> **linkKnowledgeBase**(`chatbotId`, `request`): `Promise`\<\{ `data`: [`ChatbotKnowledgeBaseLink`](/api/sdk/interfaces/chatbotknowledgebaselink/) \| `null`; `error`: `Error` \| `null`; \}\>

Link a knowledge base to a chatbot

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `chatbotId` | `string` | Chatbot ID |
| `request` | [`LinkKnowledgeBaseRequest`](/api/sdk/interfaces/linkknowledgebaserequest/) | Link configuration |

#### Returns

`Promise`\<\{ `data`: [`ChatbotKnowledgeBaseLink`](/api/sdk/interfaces/chatbotknowledgebaselink/) \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with link details

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

> **listChatbotKnowledgeBases**(`chatbotId`): `Promise`\<\{ `data`: [`ChatbotKnowledgeBaseLink`](/api/sdk/interfaces/chatbotknowledgebaselink/)[] \| `null`; `error`: `Error` \| `null`; \}\>

List knowledge bases linked to a chatbot

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `chatbotId` | `string` | Chatbot ID |

#### Returns

`Promise`\<\{ `data`: [`ChatbotKnowledgeBaseLink`](/api/sdk/interfaces/chatbotknowledgebaselink/)[] \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with linked knowledge bases

#### Example

```typescript
const { data, error } = await client.admin.ai.listChatbotKnowledgeBases('chatbot-uuid')
if (data) {
  console.log('Linked KBs:', data.map(l => l.knowledge_base_id))
}
```

***

### listChatbots()

> **listChatbots**(`namespace?`): `Promise`\<\{ `data`: [`AIChatbotSummary`](/api/sdk/interfaces/aichatbotsummary/)[] \| `null`; `error`: `Error` \| `null`; \}\>

List all chatbots (admin view)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `namespace?` | `string` | Optional namespace filter |

#### Returns

`Promise`\<\{ `data`: [`AIChatbotSummary`](/api/sdk/interfaces/aichatbotsummary/)[] \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with array of chatbot summaries

#### Example

```typescript
const { data, error } = await client.admin.ai.listChatbots()
if (data) {
  console.log('Chatbots:', data.map(c => c.name))
}
```

***

### listProviders()

> **listProviders**(): `Promise`\<\{ `data`: [`AIProvider`](/api/sdk/interfaces/aiprovider/)[] \| `null`; `error`: `Error` \| `null`; \}\>

List all AI providers

#### Returns

`Promise`\<\{ `data`: [`AIProvider`](/api/sdk/interfaces/aiprovider/)[] \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with array of providers

#### Example

```typescript
const { data, error } = await client.admin.ai.listProviders()
if (data) {
  console.log('Providers:', data.map(p => p.name))
}
```

***

### setDefaultProvider()

> **setDefaultProvider**(`id`): `Promise`\<\{ `data`: [`AIProvider`](/api/sdk/interfaces/aiprovider/) \| `null`; `error`: `Error` \| `null`; \}\>

Set a provider as the default provider

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Provider ID |

#### Returns

`Promise`\<\{ `data`: [`AIProvider`](/api/sdk/interfaces/aiprovider/) \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with updated provider

***

### setEmbeddingProvider()

> **setEmbeddingProvider**(`id`): `Promise`\<\{ `data`: \{ `use_for_embeddings`: `boolean`; \} \| `null`; `error`: `Error` \| `null`; \}\>

Set a provider as the embedding provider

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Provider ID |

#### Returns

`Promise`\<\{ `data`: \{ `use_for_embeddings`: `boolean`; \} \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with updated provider

***

### sync()

> **sync**(`options?`): `Promise`\<\{ `data`: [`SyncChatbotsResult`](/api/sdk/interfaces/syncchatbotsresult/) \| `null`; `error`: `Error` \| `null`; \}\>

Sync chatbots from filesystem or API payload

Can sync from:
1. Filesystem (if no chatbots provided) - loads from configured chatbots directory
2. API payload (if chatbots array provided) - syncs provided chatbot specifications

Requires service_role or admin authentication.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `options?` | [`SyncChatbotsOptions`](/api/sdk/interfaces/syncchatbotsoptions/) | Sync options including namespace and optional chatbots array |

#### Returns

`Promise`\<\{ `data`: [`SyncChatbotsResult`](/api/sdk/interfaces/syncchatbotsresult/) \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with sync results

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

> **toggleChatbot**(`id`, `enabled`): `Promise`\<\{ `data`: [`AIChatbot`](/api/sdk/interfaces/aichatbot/) \| `null`; `error`: `Error` \| `null`; \}\>

Enable or disable a chatbot

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Chatbot ID |
| `enabled` | `boolean` | Whether to enable or disable |

#### Returns

`Promise`\<\{ `data`: [`AIChatbot`](/api/sdk/interfaces/aichatbot/) \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with updated chatbot

#### Example

```typescript
const { data, error } = await client.admin.ai.toggleChatbot('uuid', true)
```

***

### unlinkKnowledgeBase()

> **unlinkKnowledgeBase**(`chatbotId`, `knowledgeBaseId`): `Promise`\<\{ `data`: `null`; `error`: `Error` \| `null`; \}\>

Unlink a knowledge base from a chatbot

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `chatbotId` | `string` | Chatbot ID |
| `knowledgeBaseId` | `string` | Knowledge base ID |

#### Returns

`Promise`\<\{ `data`: `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple

#### Example

```typescript
const { data, error } = await client.admin.ai.unlinkKnowledgeBase('chatbot-uuid', 'kb-uuid')
```

***

### updateChatbotKnowledgeBase()

> **updateChatbotKnowledgeBase**(`chatbotId`, `knowledgeBaseId`, `updates`): `Promise`\<\{ `data`: [`ChatbotKnowledgeBaseLink`](/api/sdk/interfaces/chatbotknowledgebaselink/) \| `null`; `error`: `Error` \| `null`; \}\>

Update a chatbot-knowledge base link

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `chatbotId` | `string` | Chatbot ID |
| `knowledgeBaseId` | `string` | Knowledge base ID |
| `updates` | [`UpdateChatbotKnowledgeBaseRequest`](/api/sdk/interfaces/updatechatbotknowledgebaserequest/) | Fields to update |

#### Returns

`Promise`\<\{ `data`: [`ChatbotKnowledgeBaseLink`](/api/sdk/interfaces/chatbotknowledgebaselink/) \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with updated link

#### Example

```typescript
const { data, error } = await client.admin.ai.updateChatbotKnowledgeBase(
  'chatbot-uuid',
  'kb-uuid',
  { max_chunks: 10, enabled: true }
)
```

***

### updateProvider()

> **updateProvider**(`id`, `updates`): `Promise`\<\{ `data`: [`AIProvider`](/api/sdk/interfaces/aiprovider/) \| `null`; `error`: `Error` \| `null`; \}\>

Update an existing AI provider

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Provider ID |
| `updates` | \{ `config?`: `Record`\<`string`, `unknown`\>; `display_name?`: `string`; `embedding_model?`: `string` \| `null`; `enabled?`: `boolean`; \} | Fields to update |
| `updates.config?` | `Record`\<`string`, `unknown`\> | - |
| `updates.display_name?` | `string` | - |
| `updates.embedding_model?` | `string` \| `null` | - |
| `updates.enabled?` | `boolean` | - |

#### Returns

`Promise`\<\{ `data`: [`AIProvider`](/api/sdk/interfaces/aiprovider/) \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with updated provider
