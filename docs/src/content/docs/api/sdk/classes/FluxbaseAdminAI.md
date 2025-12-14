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
