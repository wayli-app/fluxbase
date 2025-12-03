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
| `auth?` | `object` | Authentication options |
| `auth.autoRefresh?` | `boolean` | Auto-refresh token when it expires **Default** `true` |
| `auth.persist?` | `boolean` | Persist auth state in localStorage **Default** `true` |
| `auth.token?` | `string` | Access token for authentication |
| `debug?` | `boolean` | Enable debug logging **Default** `false` |
| `headers?` | `Record`\<`string`, `string`\> | Global headers to include in all requests |
| `timeout?` | `number` | Request timeout in milliseconds **Default** `30000` |
