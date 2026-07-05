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
| <a id="oncontent"></a> `onContent?` | (`delta`, `conversationId`) => `void` | Callback for content chunks (streaming) |
| <a id="ondone"></a> `onDone?` | (`usage`, `conversationId`, `extras?`) => `void` | Callback when message is complete |
| <a id="onerror"></a> `onError?` | (`error`, `code`, `conversationId`) => `void` | Callback for errors |
| <a id="onevent"></a> `onEvent?` | (`event`) => `void` | Callback for all events |
| <a id="onprogress"></a> `onProgress?` | (`step`, `message`, `conversationId`) => `void` | Callback for progress updates |
| <a id="onqueryresult"></a> `onQueryResult?` | (`query`, `summary`, `rowCount`, `data`, `conversationId`) => `void` | Callback for query results |
| <a id="reconnectattempts"></a> `reconnectAttempts?` | `number` | Reconnect attempts (0 = no reconnect) |
| <a id="reconnectdelay"></a> `reconnectDelay?` | `number` | Reconnect delay in ms |
| <a id="token"></a> `token?` | `string` | JWT token for authentication |
| <a id="wsurl"></a> `wsUrl?` | `string` | WebSocket URL (defaults to ws://host/ai/ws) |
