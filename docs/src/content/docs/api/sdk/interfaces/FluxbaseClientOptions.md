---
editUrl: false
next: false
prev: false
title: "FluxbaseClientOptions"
---

Client configuration options (Supabase-compatible)
These options are passed as the third parameter to createClient()

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="auth"></a> `auth?` | `object` | Authentication options |
| `auth.autoRefresh?` | `boolean` | Auto-refresh token when it expires **Default** `true` |
| `auth.persist?` | `boolean` | Persist auth state in localStorage **Default** `true` |
| `auth.storage?` | [`StorageAdapter`](/api/sdk/interfaces/storageadapter/) | Custom storage adapter for persisting the auth session. When provided, this adapter is always used (over the default localStorage/in-memory choice) — useful for SSR frameworks that need to back the session with an httpOnly cookie. When omitted, the SDK falls back to its default behavior (localStorage in browsers, in-memory in Node/SSR). |
| `auth.token?` | `string` | Access token for authentication |
| <a id="debug"></a> `debug?` | `boolean` | Enable debug logging **Default** `false` |
| <a id="headers"></a> `headers?` | `Record`\<`string`, `string`\> | Global headers to include in all requests |
| <a id="timeout"></a> `timeout?` | `number` | Request timeout in milliseconds **Default** `30000` |
