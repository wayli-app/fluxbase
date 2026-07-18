---
editUrl: false
next: false
prev: false
title: "AIChatServerMessage"
---

AI chat server message

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="agent"></a> `agent?` | `string` | Currently-active specialist agent name, on agent_transition events. |
| <a id="agent_transition"></a> `agent_transition?` | `AIAgentTransition` | Agent transition payload, present on agent_transition events (supervisor mode). |
| <a id="chatbot"></a> `chatbot?` | `string` | - |
| <a id="code"></a> `code?` | `string` | - |
| <a id="conversation_id"></a> `conversation_id?` | `string` | - |
| <a id="daily_quota"></a> `daily_quota?` | [`AIDailyQuotaSnapshot`](/api/sdk/interfaces/aidailyquotasnapshot/) | Per-user daily quota snapshot at turn end (Ask 2). Omitted when no limits configured. |
| <a id="data"></a> `data?` | `Record`\<`string`, `unknown`\>[] | - |
| <a id="delta"></a> `delta?` | `string` | - |
| <a id="error"></a> `error?` | `string` | - |
| <a id="matched_intent_rules"></a> `matched_intent_rules?` | [`AIMatchedIntentRule`](/api/sdk/interfaces/aimatchedintentrule/)[] | Intent rules that fired for this turn (Ask 5). Empty when none match. |
| <a id="message"></a> `message?` | `string` | - |
| <a id="message_id"></a> `message_id?` | `string` | - |
| <a id="page_context"></a> `page_context?` | `string` | Echo of the client's page_context, on agent_transition events. |
| <a id="query"></a> `query?` | `string` | - |
| <a id="row_count"></a> `row_count?` | `number` | - |
| <a id="step"></a> `step?` | `string` | - |
| <a id="summary"></a> `summary?` | `string` | - |
| <a id="type"></a> `type` | `"error"` \| `"cancelled"` \| `"chat_started"` \| `"progress"` \| `"content"` \| `"query_result"` \| `"tool_result"` \| `"agent_transition"` \| `"done"` | - |
| <a id="usage"></a> `usage?` | [`AIUsageStats`](/api/sdk/interfaces/aiusagestats/) | - |
