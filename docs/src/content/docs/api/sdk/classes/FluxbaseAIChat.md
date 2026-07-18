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

### Constructor

> **new FluxbaseAIChat**(`options`): `FluxbaseAIChat`

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `options` | [`AIChatOptions`](/api/sdk/interfaces/aichatoptions/) |

#### Returns

`FluxbaseAIChat`

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

> **sendMessage**(`conversationId`, `content`, `options?`): `void`

Send a message in a conversation

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `conversationId` | `string` | Conversation ID |
| `content` | `string` | Message content |
| `options?` | \{ `pageContext?`: `string`; \} | Optional per-message options |
| `options.pageContext?` | `string` | Page context string for page-aware chatbots. The supervisor looks up the matching PageProfile (if any) and uses it to bias routing and override per-page config. Missing or unknown values fall back to the chatbot's global config. |

#### Returns

`void`

***

### startChat()

> **startChat**(`chatbot`, `namespace?`, `conversationId?`, `impersonateUserId?`): `Promise`\<`string`\>

Start a new chat session with a chatbot

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `chatbot` | `string` | Chatbot name |
| `namespace?` | `string` | Optional namespace. If not provided and a lookup function is available, performs smart resolution: - If exactly one chatbot with this name exists, uses it - If multiple exist, tries "default" namespace - If ambiguous and not in default, throws error with available namespaces If no lookup function, falls back to "default" namespace. |
| `conversationId?` | `string` | Optional conversation ID to resume |
| `impersonateUserId?` | `string` | Optional user ID to impersonate (admin only) |

#### Returns

`Promise`\<`string`\>

Promise resolving to conversation ID
