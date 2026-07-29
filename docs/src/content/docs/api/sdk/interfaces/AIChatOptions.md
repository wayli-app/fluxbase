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
| <a id="onagentthought"></a> `onAgentThought?` | (`thought`, `conversationId`) => `void` | Callback for agent thought events (supervisor mode only). Fires for each piece of agent reasoning: routing plan, streamed thought chunk, tool call decision, or tool result summary. Use this to render a "thought process" stream alongside the final response. Suppressed server-side when the chatbot has @fluxbase:show-reasoning false — only reasoning chunks are gated, tool_call/tool_result/plan events always fire so users see actions. |
| <a id="onagenttransition"></a> `onAgentTransition?` | (`transition`, `conversationId`) => `void` | Callback for agent transition events (supervisor mode only). Fires when the supervisor routes to a specialist agent, when one agent hands off to another, and when the synthesizer/verifier engage. Use this to render the multi-agent flow as observable UI. |
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
