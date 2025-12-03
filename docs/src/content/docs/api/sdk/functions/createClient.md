---
editUrl: false
next: false
prev: false
title: "createClient"
---

> **createClient**\<`Database`, `SchemaName`\>(`fluxbaseUrl`?, `fluxbaseKey`?, `options`?): [`FluxbaseClient`](/api/sdk/classes/fluxbaseclient/)\<`Database`, `SchemaName`\>

Create a new Fluxbase client instance (Supabase-compatible)

This function signature is identical to Supabase's createClient, making migration seamless.

When called without arguments (or with undefined values), the function will attempt to
read from environment variables:
- `FLUXBASE_URL` - The URL of your Fluxbase instance
- `FLUXBASE_ANON_KEY` or `FLUXBASE_JOB_TOKEN` or `FLUXBASE_SERVICE_TOKEN` - The API key/token

This is useful in:
- Server-side environments where env vars are set
- Fluxbase job functions where tokens are automatically provided
- Edge functions with configured environment

## Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `Database` | `any` |
| `SchemaName` *extends* `string` | `any` |

## Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `fluxbaseUrl`? | `string` | The URL of your Fluxbase instance (optional if FLUXBASE_URL env var is set) |
| `fluxbaseKey`? | `string` | The anon key or JWT token (optional if env var is set) |
| `options`? | [`FluxbaseClientOptions`](/api/sdk/interfaces/fluxbaseclientoptions/) | Optional client configuration |

## Returns

[`FluxbaseClient`](/api/sdk/classes/fluxbaseclient/)\<`Database`, `SchemaName`\>

A configured Fluxbase client instance with full TypeScript support

## Example

```typescript
import { createClient } from '@fluxbase/sdk'

// Initialize with anon key (identical to Supabase)
const client = createClient(
  'http://localhost:8080',
  'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...'  // Anon JWT token
)

// With additional options
const client = createClient(
  'http://localhost:8080',
  'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...',
  { timeout: 30000, debug: true }
)

// In a Fluxbase job function (reads from env vars automatically)
const client = createClient()

// With TypeScript database types
const client = createClient<Database>(
  'http://localhost:8080',
  'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...'
)
```
