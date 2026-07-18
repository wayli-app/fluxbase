---
editUrl: false
next: false
prev: false
title: "AIChatEvent"
---

Chat event data

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="agent"></a> `agent?` | `string` | Currently-active specialist agent name, on agent_transition events. |
| <a id="agenttransition"></a> `agentTransition?` | `AIAgentTransition` | Agent transition payload, present on agent_transition events (supervisor mode). |
| <a id="chatbot"></a> `chatbot?` | `string` | - |
| <a id="code"></a> `code?` | `string` | - |
| <a id="conversationid"></a> `conversationId?` | `string` | - |
| <a id="data"></a> `data?` | `Record`\<`string`, `any`\>[] | - |
| <a id="delta"></a> `delta?` | `string` | - |
| <a id="error"></a> `error?` | `string` | - |
| <a id="message"></a> `message?` | `string` | - |
| <a id="pagecontext"></a> `pageContext?` | `string` | Echo of the client's page_context, on agent_transition events. |
| <a id="query"></a> `query?` | `string` | - |
| <a id="rowcount"></a> `rowCount?` | `number` | - |
| <a id="step"></a> `step?` | `string` | - |
| <a id="summary"></a> `summary?` | `string` | - |
| <a id="type"></a> `type` | [`AIChatEventType`](/api/sdk/type-aliases/aichateventtype/) | - |
| <a id="usage"></a> `usage?` | [`AIUsageStats`](/api/sdk/interfaces/aiusagestats/) | - |
