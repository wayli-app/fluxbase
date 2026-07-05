---
editUrl: false
next: false
prev: false
title: "useAIChat"
---

> **useAIChat**(`options`): `object`

Hook for AI chatbot streaming chat

## Parameters

| Parameter | Type |
| ------ | ------ |
| `options` | \{ `chatbot`: `string`; `namespace?`: `string`; `onError?`: (`error`) => `void`; `onQueryResult?`: (`result`) => `void`; \} |
| `options.chatbot` | `string` |
| `options.namespace?` | `string` |
| `options.onError?` | (`error`) => `void` |
| `options.onQueryResult?` | (`result`) => `void` |

## Returns

`object`

| Name | Type | Default value |
| ------ | ------ | ------ |
| `cancel()` | () => `void` | - |
| `error` | `Error` \| `null` | `state.error` |
| `isConnected` | `boolean` | `state.isConnected` |
| `isStreaming` | `boolean` | `state.isStreaming` |
| `messages` | [`ChatMessage`](/api/sdk-react/interfaces/chatmessage/)[] | `state.messages` |
| `reset()` | () => `void` | - |
| `sendMessage()` | (`content`) => `Promise`\<`void`\> | - |

## Example

```tsx
const { messages, sendMessage, isStreaming, error } = useAIChat({
  chatbot: 'my-chatbot',
  onQueryResult: (result) => console.log('SQL:', result.query),
})
```
