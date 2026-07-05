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

### Constructor

> **new FluxbaseAI**(`fetch`, `wsBaseUrl`): `FluxbaseAI`

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | \{ `delete`: (`path`) => `Promise`\<`void`\>; `get`: \<`T`\>(`path`) => `Promise`\<`T`\>; `patch`: \<`T`\>(`path`, `body?`) => `Promise`\<`T`\>; \} |
| `fetch.delete` | (`path`) => `Promise`\<`void`\> |
| `fetch.get` | \<`T`\>(`path`) => `Promise`\<`T`\> |
| `fetch.patch` | \<`T`\>(`path`, `body?`) => `Promise`\<`T`\> |
| `wsBaseUrl` | `string` |

#### Returns

`FluxbaseAI`

## Methods

### createChat()

> **createChat**(`options`): [`FluxbaseAIChat`](/api/sdk/classes/fluxbaseaichat/)

Create a new AI chat connection

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `options` | `Omit`\<[`AIChatOptions`](/api/sdk/interfaces/aichatoptions/), `"wsUrl"` \| `"_lookupChatbot"`\> | Chat connection options |

#### Returns

[`FluxbaseAIChat`](/api/sdk/classes/fluxbaseaichat/)

FluxbaseAIChat instance

***

### deleteConversation()

> **deleteConversation**(`id`): `Promise`\<\{ `error`: `Error` \| `null`; \}\>

Delete a conversation

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Conversation ID |

#### Returns

`Promise`\<\{ `error`: `Error` \| `null`; \}\>

Promise resolving to { error } (null on success)

#### Example

```typescript
const { error } = await ai.deleteConversation('conv-uuid-123')
if (!error) {
  console.log('Conversation deleted')
}
```

***

### getChatbot()

> **getChatbot**(`id`): `Promise`\<\{ `data`: [`AIChatbotSummary`](/api/sdk/interfaces/aichatbotsummary/) \| `null`; `error`: `Error` \| `null`; \}\>

Get details of a specific chatbot

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Chatbot ID |

#### Returns

`Promise`\<\{ `data`: [`AIChatbotSummary`](/api/sdk/interfaces/aichatbotsummary/) \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with chatbot details

***

### getConversation()

> **getConversation**(`id`): `Promise`\<\{ `data`: [`AIUserConversationDetail`](/api/sdk/interfaces/aiuserconversationdetail/) \| `null`; `error`: `Error` \| `null`; \}\>

Get a single conversation with all messages

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Conversation ID |

#### Returns

`Promise`\<\{ `data`: [`AIUserConversationDetail`](/api/sdk/interfaces/aiuserconversationdetail/) \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with conversation detail

#### Example

```typescript
const { data, error } = await ai.getConversation('conv-uuid-123')
if (data) {
  console.log(`Title: ${data.title}`)
  console.log(`Messages: ${data.messages.length}`)
}
```

***

### getUsage()

> **getUsage**(`chatbotId`): `Promise`\<\{ `data`: [`AIDailyQuotaSnapshot`](/api/sdk/interfaces/aidailyquotasnapshot/) \| `null`; `error`: `Error` \| `null`; \}\>

Get the current user's daily quota for a chatbot (Ask 2).

Returns requests used/limit and tokens used/limit, plus the RFC3339
timestamp of when the counters reset. Best-effort: counters are in-memory
and reset on server restart; per-instance in multi-replica deployments.

For per-turn updates without an extra round-trip, read `daily_quota` from
the `onDone` callback's third argument.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `chatbotId` | `string` | Chatbot ID |

#### Returns

`Promise`\<\{ `data`: [`AIDailyQuotaSnapshot`](/api/sdk/interfaces/aidailyquotasnapshot/) \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } with the quota snapshot

#### Example

```typescript
const { data, error } = await ai.getUsage('chatbot-uuid')
if (data) {
  console.log(`${data.requests.limit - data.requests.used} requests left today`)
  console.log(`${data.tokens.limit - data.tokens.used} tokens left today`)
}
```

***

### listChatbots()

> **listChatbots**(`namespace?`): `Promise`\<\{ `data`: [`AIChatbotSummary`](/api/sdk/interfaces/aichatbotsummary/)[] \| `null`; `error`: `Error` \| `null`; \}\>

List available chatbots (public, enabled)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `namespace?` | `string` |

#### Returns

`Promise`\<\{ `data`: [`AIChatbotSummary`](/api/sdk/interfaces/aichatbotsummary/)[] \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with array of chatbot summaries

***

### listConversations()

> **listConversations**(`options?`): `Promise`\<\{ `data`: [`ListConversationsResult`](/api/sdk/interfaces/listconversationsresult/) \| `null`; `error`: `Error` \| `null`; \}\>

List the authenticated user's conversations

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `options?` | [`ListConversationsOptions`](/api/sdk/interfaces/listconversationsoptions/) | Optional filters and pagination |

#### Returns

`Promise`\<\{ `data`: [`ListConversationsResult`](/api/sdk/interfaces/listconversationsresult/) \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with conversations

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

### lookupChatbot()

> **lookupChatbot**(`name`): `Promise`\<\{ `data`: [`AIChatbotLookupResponse`](/api/sdk/interfaces/aichatbotlookupresponse/) \| `null`; `error`: `Error` \| `null`; \}\>

Lookup a chatbot by name with smart namespace resolution

Resolution logic:
1. If exactly one chatbot with this name exists -> returns it
2. If multiple exist -> tries "default" namespace first
3. If multiple exist and none in "default" -> returns ambiguous=true with namespaces list

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `name` | `string` | Chatbot name |

#### Returns

`Promise`\<\{ `data`: [`AIChatbotLookupResponse`](/api/sdk/interfaces/aichatbotlookupresponse/) \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with lookup result

#### Example

```typescript
// Lookup chatbot by name
const { data, error } = await ai.lookupChatbot('sql-assistant')
if (data?.chatbot) {
  console.log(`Found in namespace: ${data.chatbot.namespace}`)
} else if (data?.ambiguous) {
  console.log(`Chatbot exists in: ${data.namespaces?.join(', ')}`)
}
```

***

### updateConversation()

> **updateConversation**(`id`, `updates`): `Promise`\<\{ `data`: [`AIUserConversationDetail`](/api/sdk/interfaces/aiuserconversationdetail/) \| `null`; `error`: `Error` \| `null`; \}\>

Update a conversation (currently supports title update only)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Conversation ID |
| `updates` | [`UpdateConversationOptions`](/api/sdk/interfaces/updateconversationoptions/) | Fields to update |

#### Returns

`Promise`\<\{ `data`: [`AIUserConversationDetail`](/api/sdk/interfaces/aiuserconversationdetail/) \| `null`; `error`: `Error` \| `null`; \}\>

Promise resolving to { data, error } tuple with updated conversation

#### Example

```typescript
const { data, error } = await ai.updateConversation('conv-uuid-123', {
  title: 'My custom conversation title'
})
```
