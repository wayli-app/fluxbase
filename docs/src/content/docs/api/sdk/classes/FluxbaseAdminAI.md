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

### createIntegration()

> **createIntegration**(`request`): `Promise`\<\{ `data`: [`ToolIntegration`](/api/sdk/interfaces/toolintegration/) \| `null`; `error`: `Error` \| `null`; \}\>

Create a new tool integration. api_key in config is encrypted at the
storage layer; never appears in plaintext in API responses.

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `request` | [`CreateToolIntegrationRequest`](/api/sdk/interfaces/createtoolintegrationrequest/) |

#### Returns

`Promise`\<\{ `data`: [`ToolIntegration`](/api/sdk/interfaces/toolintegration/) \| `null`; `error`: `Error` \| `null`; \}\>

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

### deleteIntegration()

> **deleteIntegration**(`id`): `Promise`\<\{ `error`: `Error` \| `null`; \}\>

Delete a tool integration. Refuses to delete read-only integrations
configured via env/YAML.

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `id` | `string` |

#### Returns

`Promise`\<\{ `error`: `Error` \| `null`; \}\>

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

### getIntegration()

> **getIntegration**(`id`): `Promise`\<\{ `data`: [`ToolIntegration`](/api/sdk/interfaces/toolintegration/) \| `null`; `error`: `Error` \| `null`; \}\>

Get a single tool integration by ID.

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `id` | `string` |

#### Returns

`Promise`\<\{ `data`: [`ToolIntegration`](/api/sdk/interfaces/toolintegration/) \| `null`; `error`: `Error` \| `null`; \}\>

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

### listIntegrations()

> **listIntegrations**(`params?`): `Promise`\<\{ `data`: [`ToolIntegration`](/api/sdk/interfaces/toolintegration/)[] \| `null`; `error`: `Error` \| `null`; \}\>

List tool integrations, optionally filtered by integration_type.

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `params?` | \{ `integration_type?`: [`IntegrationType`](/api/sdk/type-aliases/integrationtype/); \} |
| `params.integration_type?` | [`IntegrationType`](/api/sdk/type-aliases/integrationtype/) |

#### Returns

`Promise`\<\{ `data`: [`ToolIntegration`](/api/sdk/interfaces/toolintegration/)[] \| `null`; `error`: `Error` \| `null`; \}\>

#### Example

```ts
const { data, error } = await client.admin.ai.listIntegrations({ integration_type: "web_search" })
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

### setDefaultIntegration()

> **setDefaultIntegration**(`id`): `Promise`\<\{ `error`: `Error` \| `null`; \}\>

Mark an integration as the default for its integration_type within
the current tenant. The server clears any prior default in the same
transaction.

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `id` | `string` |

#### Returns

`Promise`\<\{ `error`: `Error` \| `null`; \}\>

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

### testIntegration()

> **testIntegration**(`id`): `Promise`\<\{ `data`: [`TestToolIntegrationResult`](/api/sdk/interfaces/testtoolintegrationresult/) \| `null`; `error`: `Error` \| `null`; \}\>

Test an integration by running a real "hello world" call against
its provider. Stores last_tested_at + last_test_status + error
on the integration row so the admin UI can display health.

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `id` | `string` |

#### Returns

`Promise`\<\{ `data`: [`TestToolIntegrationResult`](/api/sdk/interfaces/testtoolintegrationresult/) \| `null`; `error`: `Error` \| `null`; \}\>

#### Example

```ts
const { data, error } = await client.admin.ai.testIntegration(integrationId)
if (data?.status === "ok") { console.log("Tavily credentials verified") }
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

### updateChatbot()

> **updateChatbot**(`id`, `updates`): `Promise`\<\{ `data`: [`AIChatbot`](/api/sdk/interfaces/aichatbot/) \| `null`; `error`: `Error` \| `null`; \}\>

Update a chatbot's configuration (partial update).

Only the provided fields are changed; omitted fields are left untouched.
For limit/budget fields, 0 means "unlimited". Changes take effect on the
next request (limits are read fresh from the database per message).

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Chatbot ID |
| `updates` | \{ `allow_unauthenticated?`: `boolean`; `conversation_ttl_hours?`: `number`; `daily_request_limit?`: `number`; `daily_token_budget?`: `number`; `description?`: `string`; `enabled?`: `boolean`; `is_public?`: `boolean`; `max_conversation_turns?`: `number`; `max_tokens?`: `number`; `persist_conversations?`: `boolean`; `provider_id?`: `string` \| `null`; `rate_limit_per_minute?`: `number`; `temperature?`: `number`; \} | Fields to update (all optional) |
| `updates.allow_unauthenticated?` | `boolean` | - |
| `updates.conversation_ttl_hours?` | `number` | - |
| `updates.daily_request_limit?` | `number` | - |
| `updates.daily_token_budget?` | `number` | - |
| `updates.description?` | `string` | - |
| `updates.enabled?` | `boolean` | - |
| `updates.is_public?` | `boolean` | - |
| `updates.max_conversation_turns?` | `number` | - |
| `updates.max_tokens?` | `number` | - |
| `updates.persist_conversations?` | `boolean` | - |
| `updates.provider_id?` | `string` \| `null` | - |
| `updates.rate_limit_per_minute?` | `number` | - |
| `updates.temperature?` | `number` | - |

#### Returns

`Promise`\<\{ `data`: [`AIChatbot`](/api/sdk/interfaces/aichatbot/) \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with the updated chatbot

#### Example

```typescript
const { data, error } = await client.admin.ai.updateChatbot('uuid', {
  daily_request_limit: 500,   // 0 = unlimited
  daily_token_budget: 200000, // 0 = unlimited
})
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

### updateIntegration()

> **updateIntegration**(`id`, `request`): `Promise`\<\{ `data`: [`ToolIntegration`](/api/sdk/interfaces/toolintegration/) \| `null`; `error`: `Error` \| `null`; \}\>

Update an existing tool integration. Passing config.api_key =
"***masked***" preserves the existing encrypted value (useful when
the admin changes only the name and not the API key).

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `id` | `string` |
| `request` | [`UpdateToolIntegrationRequest`](/api/sdk/interfaces/updatetoolintegrationrequest/) |

#### Returns

`Promise`\<\{ `data`: [`ToolIntegration`](/api/sdk/interfaces/toolintegration/) \| `null`; `error`: `Error` \| `null`; \}\>

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
