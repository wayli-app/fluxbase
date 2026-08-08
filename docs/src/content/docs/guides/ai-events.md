---
title: "AI Chat WebSocket Events"
description: Reference for every event type emitted on the AI chat WebSocket — client-to-server messages, server-to-client events, and supervisor-mode agent transitions.
---

# AI Chat WebSocket Events

Fluxbase chatbots communicate over a WebSocket at `/ai/ws`. This page is the complete reference for every event type — both directions.

## Connection lifecycle

1. Client opens WebSocket to `/ai/ws` with optional `Authorization: Bearer <token>` header
2. Client sends `start_chat` to begin a conversation with a named chatbot
3. Client sends `message` events for each user turn
4. Server streams back `progress`, `content`, `query_result`, `agent_transition`, and finally `done` events
5. Either side may close the connection at any time

---

## Client → Server messages

### `start_chat`

Begins a new conversation with a chatbot. Returns a `chat_started` event with the conversation ID.

```typescript
{
  type: "start_chat",
  chatbot: "my-assistant",       // required
  namespace: "default",          // optional, defaults to "default"
}
```

### `message`

Sends one user turn. Server streams back events until `done`.

```typescript
{
  type: "message",
  conversation_id: "conv_abc123", // required
  content: "How many users signed up last week?",
  page_context: "dashboard",      // optional — see Page-aware chatbots
}
```

`page_context` is the optional Level-2 page-aware-chatbot signal. When set, the supervisor looks up the matching `PageProfile` from the chatbot's configuration and uses it to bias routing and override per-page config. Missing or unknown values fall back to the chatbot's global config.

### `cancel`

Cancels an in-progress message generation.

```typescript
{
  type: "cancel",
  conversation_id: "conv_abc123",
}
```

---

## Server → Client events

### `chat_started`

Confirms a conversation has started. Emitted in response to `start_chat`.

```typescript
{
  type: "chat_started",
  conversation_id: "conv_abc123",
  chatbot: "my-assistant",
}
```

### `progress`

Status update while the chatbot is working (e.g., "Thinking...", "Executing query...").

```typescript
{
  type: "progress",
  conversation_id: "conv_abc123",
  step: "thinking" | "querying" | "generating" | "executing",
  message: "Executing: count users by week",
}
```

### `content`

Streamed content delta. Concatenate deltas to build the full response.

```typescript
{
  type: "content",
  conversation_id: "conv_abc123",
  delta: "Last week, ",
}
```

In supervisor mode, the supervisor path emits the full final response as a single `content` event. Future versions may stream token-by-token.

### `query_result`

Structured query result, emitted when the chatbot ran SQL. Use this to render data tables alongside the chat.

```typescript
{
  type: "query_result",
  conversation_id: "conv_abc123",
  query: "SELECT date_trunc('week', created_at) AS week, COUNT(*) FROM auth.users GROUP BY week",
  summary: "Query returned 4 row(s)",
  row_count: 4,
  data: [
    { "week": "2026-06-29T00:00:00Z", "count": 142 },
    { "week": "2026-07-06T00:00:00Z", "count": 198 },
    // ...
  ],
}
```

### `tool_result`

Generic tool execution result, emitted for non-SQL MCP tools (`invoke_function`, `rpc_call`, etc.). SQL and `query_table` results use `query_result` instead.

```typescript
{
  type: "tool_result",
  conversation_id: "conv_abc123",
  message: "invoke_function",
  data: [{ "tool": "invoke_function", "result": "..." }],
}
```

### `agent_transition`

**Supervisor mode only.** Emitted when one agent hands off to another. Use this to render the multi-agent routing flow as observable UI.

```typescript
{
  type: "agent_transition",
  conversation_id: "conv_abc123",
  agent: "sql",                    // currently-active agent
  agent_transition: {
    from: "supervisor",            // optional
    to: "sql",                     // optional
    route: ["sql"],                // supervisor's full routing decision
    reason: "",                    // supervisor's stated reason, if any
    page_context: "dashboard",     // echo of client's page_context
  },
  page_context: "dashboard",
}
```

Typical sequence on an investigative turn:

1. `agent_transition` with `route: ["sql"]` (supervisor made routing decision)
2. `agent_transition` with `to: "sql"` (SQL agent starting)
3. One or more `query_result` events
4. `agent_transition` with `to: "verifier"` (verifier checking grounding)
5. `content` (final response)
6. `done`

### `done`

Turn complete. Always emitted (except on `error` or `cancelled`).

```typescript
{
  type: "done",
  conversation_id: "conv_abc123",
  usage: {
    prompt_tokens: 1245,
    completion_tokens: 87,
    total_tokens: 1332,
    cached_tokens: 800,           // subset of prompt_tokens served from cache
  },
  matched_intent_rules: [          // only when intent rules configured
    {
      keyword: "restaurant",
      required_table: "my_places",
    },
  ],
  daily_quota: {                   // only when limits configured
    requests: { used: 12, limit: 500 },
    tokens:   { used: 4521, limit: 100000 },
    resets_at: "2026-07-19T00:00:00Z",
  },
  page_context: "dashboard",       // echo of client's page_context
}
```

### `error`

Recoverable or unrecoverable error.

```typescript
{
  type: "error",
  conversation_id: "conv_abc123",
  code: "RATE_LIMITED" | "DAILY_LIMIT" | "TOKEN_BUDGET" | "PROVIDER_ERROR" | "STREAM_ERROR" | "PROMPT_ERROR" | ...,
  error: "Rate limit exceeded. Please try again later.",
}
```

### `cancelled`

Confirms cancellation of an in-progress turn.

```typescript
{
  type: "cancelled",
  conversation_id: "conv_abc123",
}
```

---

## SDK usage

The TypeScript SDK exposes typed callbacks for each event:

```typescript
import { FluxbaseAIChat } from "@fluxbase/sdk";

const chat = new FluxbaseAIChat({
  wsUrl: "ws://localhost:8080/ai/ws",
  token: "<jwt>",

  onContent: (delta, convId) => {
    process.stdout.write(delta);
  },

  onProgress: (step, message, convId) => {
    console.log(`[${step}] ${message}`);
  },

  onQueryResult: (query, summary, rowCount, data, convId) => {
    console.log(`Query returned ${rowCount} rows`);
  },

  onAgentTransition: (transition, convId) => {
    if (transition.route) {
      console.log(`Supervisor routed to: ${transition.route.join(", ")}`);
    } else if (transition.to) {
      console.log(`→ ${transition.to}`);
    }
  },

  onDone: (usage, convId, extras) => {
    console.log(`Done. Tokens: ${usage?.total_tokens}`);
    if (extras?.daily_quota) {
      console.log(`Remaining: ${extras.daily_quota.tokens.limit - extras.daily_quota.tokens.used} tokens today`);
    }
  },

  onError: (error, code, convId) => {
    console.error(`Error [${code}]: ${error}`);
  },
});

await chat.connect();
const convId = await chat.startChat("my-assistant");
chat.sendMessage(convId, "How many users signed up last week?", {
  pageContext: "dashboard",
});
```

See the [AI Chatbots guide](/guides/ai-chatbots/) for end-to-end examples and the [Multi-Agent Supervisor guide](/guides/ai-agents/) for routing details.
