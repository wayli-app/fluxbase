---
editUrl: false
next: false
prev: false
title: "FluxbaseAI"
---

Fluxbase AI client for listing chatbots and managing conversations

## Example

```typescript
const ai = new FluxbaseAI(fetchClient, 'ws://localhost:8080')

// List available chatbots
const { data, error } = await ai.listChatbots()

// Create a chat connection
const chat = ai.createChat({
  token: 'my-jwt-token',
  onContent: (delta) => process.stdout.write(delta),
})

await chat.connect()
const convId = await chat.startChat('sql-assistant')
chat.sendMessage(convId, 'Show me recent orders')
```

## Constructors

### new FluxbaseAI()

> **new FluxbaseAI**(`fetch`, `wsBaseUrl`): [`FluxbaseAI`](/api/sdk/classes/fluxbaseai/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | `object` |
| `fetch.delete` | (`path`) => `Promise`\<`void`\> |
| `fetch.get` | \<`T`\>(`path`) => `Promise`\<`T`\> |
| `fetch.patch` | \<`T`\>(`path`, `body`?) => `Promise`\<`T`\> |
| `wsBaseUrl` | `string` |

#### Returns

[`FluxbaseAI`](/api/sdk/classes/fluxbaseai/)

## Methods

### createChat()

> **createChat**(`options`): [`FluxbaseAIChat`](/api/sdk/classes/fluxbaseaichat/)

Create a new AI chat connection

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `options` | `Omit`\<[`AIChatOptions`](/api/sdk/interfaces/aichatoptions/), `"wsUrl"`\> | Chat connection options |

#### Returns

[`FluxbaseAIChat`](/api/sdk/classes/fluxbaseaichat/)

FluxbaseAIChat instance

***

### deleteConversation()

> **deleteConversation**(`id`): `Promise`\<`object`\>

Delete a conversation

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Conversation ID |

#### Returns

`Promise`\<`object`\>

Promise resolving to { error } (null on success)

| Name | Type |
| ------ | ------ |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { error } = await ai.deleteConversation('conv-uuid-123')
if (!error) {
  console.log('Conversation deleted')
}
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
| `data` | `null` \| [`AIChatbotSummary`](/api/sdk/interfaces/aichatbotsummary/) |
| `error` | `null` \| `Error` |

***

### getConversation()

> **getConversation**(`id`): `Promise`\<`object`\>

Get a single conversation with all messages

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Conversation ID |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with conversation detail

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`AIUserConversationDetail`](/api/sdk/interfaces/aiuserconversationdetail/) |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await ai.getConversation('conv-uuid-123')
if (data) {
  console.log(`Title: ${data.title}`)
  console.log(`Messages: ${data.messages.length}`)
}
```

***

### listChatbots()

> **listChatbots**(): `Promise`\<`object`\>

List available chatbots (public, enabled)

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with array of chatbot summaries

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`AIChatbotSummary`](/api/sdk/interfaces/aichatbotsummary/)[] |
| `error` | `null` \| `Error` |

***

### listConversations()

> **listConversations**(`options`?): `Promise`\<`object`\>

List the authenticated user's conversations

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `options`? | [`ListConversationsOptions`](/api/sdk/interfaces/listconversationsoptions/) | Optional filters and pagination |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with conversations

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`ListConversationsResult`](/api/sdk/interfaces/listconversationsresult/) |
| `error` | `null` \| `Error` |

#### Example

```typescript
// List all conversations
const { data, error } = await ai.listConversations()

// Filter by chatbot
const { data, error } = await ai.listConversations({ chatbot: 'sql-assistant' })

// With pagination
const { data, error } = await ai.listConversations({ limit: 20, offset: 0 })
```

***

### updateConversation()

> **updateConversation**(`id`, `updates`): `Promise`\<`object`\>

Update a conversation (currently supports title update only)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Conversation ID |
| `updates` | [`UpdateConversationOptions`](/api/sdk/interfaces/updateconversationoptions/) | Fields to update |

#### Returns

`Promise`\<`object`\>

Promise resolving to { data, error } tuple with updated conversation

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`AIUserConversationDetail`](/api/sdk/interfaces/aiuserconversationdetail/) |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await ai.updateConversation('conv-uuid-123', {
  title: 'My custom conversation title'
})
```
