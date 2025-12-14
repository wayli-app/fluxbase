---
editUrl: false
next: false
prev: false
title: "FluxbaseAIChat"
---

AI Chat client for WebSocket-based chat with AI chatbots

## Example

```typescript
const chat = new FluxbaseAIChat({
  wsUrl: 'ws://localhost:8080/ai/ws',
  token: 'my-jwt-token',
  onContent: (delta, convId) => {
    process.stdout.write(delta)
  },
  onProgress: (step, message) => {
    console.log(`[${step}] ${message}`)
  },
  onQueryResult: (query, summary, rowCount, data) => {
    console.log(`Query: ${query}`)
    console.log(`Result: ${summary} (${rowCount} rows)`)
  },
  onDone: (usage) => {
    console.log(`\nTokens: ${usage?.total_tokens}`)
  },
  onError: (error, code) => {
    console.error(`Error: ${error} (${code})`)
  },
})

await chat.connect()
const convId = await chat.startChat('sql-assistant')
await chat.sendMessage(convId, 'Show me the top 10 users by order count')
```

## Constructors

### new FluxbaseAIChat()

> **new FluxbaseAIChat**(`options`): [`FluxbaseAIChat`](/api/sdk/classes/fluxbaseaichat/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `options` | [`AIChatOptions`](/api/sdk/interfaces/aichatoptions/) |

#### Returns

[`FluxbaseAIChat`](/api/sdk/classes/fluxbaseaichat/)

## Methods

### cancel()

> **cancel**(`conversationId`): `void`

Cancel an ongoing message generation

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `conversationId` | `string` | Conversation ID |

#### Returns

`void`

***

### connect()

> **connect**(): `Promise`\<`void`\>

Connect to the AI chat WebSocket

#### Returns

`Promise`\<`void`\>

Promise that resolves when connected

***

### disconnect()

> **disconnect**(): `void`

Disconnect from the AI chat WebSocket

#### Returns

`void`

***

### getAccumulatedContent()

> **getAccumulatedContent**(`conversationId`): `string`

Get the full accumulated response content for a conversation

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `conversationId` | `string` | Conversation ID |

#### Returns

`string`

Accumulated content string

***

### isConnected()

> **isConnected**(): `boolean`

Check if connected

#### Returns

`boolean`

***

### sendMessage()

> **sendMessage**(`conversationId`, `content`): `void`

Send a message in a conversation

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `conversationId` | `string` | Conversation ID |
| `content` | `string` | Message content |

#### Returns

`void`

***

### startChat()

> **startChat**(`chatbot`, `namespace`?, `conversationId`?, `impersonateUserId`?): `Promise`\<`string`\>

Start a new chat session with a chatbot

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `chatbot` | `string` | Chatbot name |
| `namespace`? | `string` | Optional namespace (defaults to 'default') |
| `conversationId`? | `string` | Optional conversation ID to resume |
| `impersonateUserId`? | `string` | Optional user ID to impersonate (admin only) |

#### Returns

`Promise`\<`string`\>

Promise resolving to conversation ID
