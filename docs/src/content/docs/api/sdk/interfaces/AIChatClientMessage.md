---
editUrl: false
next: false
prev: false
title: "AIChatClientMessage"
---

AI chat message for WebSocket

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="chatbot"></a> `chatbot?` | `string` | - |
| <a id="content"></a> `content?` | `string` | - |
| <a id="conversation_id"></a> `conversation_id?` | `string` | - |
| <a id="impersonate_user_id"></a> `impersonate_user_id?` | `string` | - |
| <a id="namespace"></a> `namespace?` | `string` | - |
| <a id="page_context"></a> `page_context?` | `string` | Optional page context for page-aware chatbots (Level 2 page profiles). Sent per-message; the supervisor looks up the matching PageProfile (if any) and uses it to bias routing and override per-page config. Missing or unknown values fall back to the chatbot's global config. |
| <a id="type"></a> `type` | `"start_chat"` \| `"message"` \| `"cancel"` | - |
