---
title: "Tool Integrations"
description: Configure external services (web search via Tavily) that chatbot specialists can call. Supports env/YAML config, admin UI, and API.
---

# Tool Integrations

Fluxbase chatbots can call external services beyond the database — web search via Tavily being the first-class example. This page covers configuration, the chatbot-side opt-in, and the architecture.

Tool integrations are **separate from AI providers**: they don't do chat completions, don't take a model, and aren't selected per-chatbot via `chatbot.provider_id`. They live in their own `ai.tool_integrations` table with their own CRUD API and admin UI.

## What's available

| Integration type | Providers | What it does |
|---|---|---|
| `web_search` | Tavily (v1) | Search the web for current info. Used by the Web Agent specialist. |
| `fetch_url` | Tavily Extract (v1) | Fetch a single URL as clean markdown. Auto-derived from a `web_search` integration. |

Future providers (Brave, Jina, custom) fit into the same schema without restructuring.

## Configuration

Three paths, all equivalent:

### 1. Environment variable (simplest)

```bash
export FLUXBASE_AI_TAVILY_API_KEY="tvly-..."
# Optional:
# export FLUXBASE_AI_TAVILY_DEFAULT_DEPTH="advanced"
# export FLUXBASE_AI_TAVILY_BASE_URL="https://api.tavily.com"
```

When the server starts, it constructs a synthetic `FROM_CONFIG` integration row marked read-only. Chatbots with `@fluxbase:web-search enabled=true` immediately use it.

### 2. YAML config file

```yaml
ai:
  tavily_api_key: "tvly-..."
  tavily_default_depth: "advanced"   # optional: "basic" (default) or "advanced"
  # tavily_base_url: ""              # optional override
```

Same synthetic read-only row as env vars.

### 3. Admin UI

Navigate to **AI → Tool Integrations** in the sidebar. Click **New Integration**, fill in:

- **Name**: human label (e.g., "Tavily prod")
- **Provider**: Tavily (Brave/Jina will appear here when implemented)
- **API Key**: your Tavily key (encrypted at rest with AES-256-GCM via the master `FLUXBASE_ENCRYPTION_KEY`)
- **Default Search Depth**: `basic` (faster, cheaper) or `advanced` (slower, more thorough)
- **Default**: check to make this the default `web_search` integration for the tenant

Click the **Test connection** button in the row's dropdown menu to verify the credentials work before saving. The result is stored on the integration row and surfaced as a status pill in the list.

### 4. API

```bash
# Create
curl -X POST https://your-server/api/v1/admin/ai/integrations \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Tavily prod",
    "integration_type": "web_search",
    "provider": "tavily",
    "config": {"api_key": "tvly-..."},
    "enabled": true,
    "is_default": true
  }'

# Test connection
curl -X POST https://your-server/api/v1/admin/ai/integrations/$ID/test \
  -H "Authorization: Bearer $TOKEN"
```

Full CRUD endpoints listed in the [API reference](/guides/ai-events/).

## Enable for a chatbot

Tool integrations are tenant-level — but the chatbot decides whether to use them. Add the annotation to the chatbot's source:

```typescript
/**
 * Enable web search for this chatbot.
 * @fluxbase:web-search enabled
 *
 * Optional: restrict results to specific domains
 * @fluxbase:web-search-domains wikipedia.org, docs.python.org, https://news.ycombinator.com
 */
export default `You are a helpful assistant.`;
```

The supervisor's router now includes the **Web Agent** as a routable specialist. When the user asks about current info ("what's the latest X", "what time does Y close today"), the supervisor routes to `web` instead of `chat` with an "I don't know" fallback.

## How the Web Agent works

```
User: "What time does the Berlin zoo close today?"
   │
[SUPERVISOR] routes to ["web"]   (KB has nothing, SQL has nothing, current info needed)
   │
[WEB AGENT]
   ├─ web_search("Berlin zoo closing time today")
   │     → 5 results, top hit zooberlin.de
   ├─ fetch_url("https://www.zooberlin.de/")
   │     → full page markdown
   └─ final answer: "Berlin zoo closes at 18:30 today ([source](https://www.zooberlin.de))."
```

The Web Agent emits `agent_thought` events for each step (search query, fetch URL, result summary), so clients can render the thought process in real time. See [Multi-Agent Supervisor](/guides/ai-agents/) for the full event reference.

## Cost

Tavily pricing (as of this writing):

| Plan | Cost |
|---|---|
| Free | 1,000 searches/month |
| Pro | $30/month, 4,000 searches, then $0.01/search |
| Enterprise | Custom |

Per chatbot turn that routes to the Web Agent:

- 1-2 Tavily calls (search + optional fetch) ≈ $0.01-0.03
- 1-3 extra LLM calls for the agent's reasoning loop ≈ $0.005-0.02

Typical chatbot with websearch enabled: $5-20/month additional cost on top of the existing LLM provider bill. Disable per chatbot by removing the annotation.

## Secrets and encryption

API keys in `tool_integrations.config.api_key` are encrypted at the application layer with AES-256-GCM, keyed by the master `FLUXBASE_ENCRYPTION_KEY`. They never appear in plaintext in API responses (masked to `"***masked***"`).

If the encryption key is missing or changed, decryption fails gracefully — the integration's `api_key` becomes unusable but no other data is lost. Document this clearly in your deploy runbook.

## Multi-tenancy

Each tenant can have its own integrations. Tenant scoping is enforced by the same RLS policies used elsewhere — tenant A's integrations are invisible to tenant B. The synthetic `FROM_CONFIG` row (env/YAML config) is instance-wide and read-only.

## See also

- [Multi-Agent Supervisor](/guides/ai-agents/) — where the Web Agent fits in the pipeline
- [AI Chatbots](/guides/ai-chatbots/) — chatbot configuration including the `@fluxbase:web-search` annotation
- [AI events reference](/guides/ai-events/) — WebSocket events emitted by the Web Agent
