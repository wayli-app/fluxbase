---
editUrl: false
next: false
prev: false
title: "AIChatOptions"
---

Chat connection options

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `onContent?` | (`delta`: `string`, `conversationId`: `string`) => `void` | Callback for content chunks (streaming) |
| `onDone?` | (`usage`: `undefined` \| [`AIUsageStats`](/api/sdk/interfaces/aiusagestats/), `conversationId`: `string`) => `void` | Callback when message is complete |
| `onError?` | (`error`: `string`, `code`: `undefined` \| `string`, `conversationId`: `undefined` \| `string`) => `void` | Callback for errors |
| `onEvent?` | (`event`: [`AIChatEvent`](/api/sdk/interfaces/aichatevent/)) => `void` | Callback for all events |
| `onProgress?` | (`step`: `string`, `message`: `string`, `conversationId`: `string`) => `void` | Callback for progress updates |
| `onQueryResult?` | (`query`: `string`, `summary`: `string`, `rowCount`: `number`, `data`: `Record`\<`string`, `any`\>[], `conversationId`: `string`) => `void` | Callback for query results |
| `reconnectAttempts?` | `number` | Reconnect attempts (0 = no reconnect) |
| `reconnectDelay?` | `number` | Reconnect delay in ms |
| `token?` | `string` | JWT token for authentication |
| `wsUrl?` | `string` | WebSocket URL (defaults to ws://host/ai/ws) |
