---
editUrl: false
next: false
prev: false
title: "AIChatServerMessage"
---

AI chat server message

## Properties

| Property | Type |
| ------ | ------ |
| `chatbot?` | `string` |
| `code?` | `string` |
| `conversation_id?` | `string` |
| `data?` | `Record`\<`string`, `any`\>[] |
| `delta?` | `string` |
| `error?` | `string` |
| `message?` | `string` |
| `message_id?` | `string` |
| `query?` | `string` |
| `row_count?` | `number` |
| `step?` | `string` |
| `summary?` | `string` |
| `type` | `"error"` \| `"cancelled"` \| `"chat_started"` \| `"progress"` \| `"content"` \| `"query_result"` \| `"done"` |
| `usage?` | [`AIUsageStats`](/api/sdk/interfaces/aiusagestats/) |
