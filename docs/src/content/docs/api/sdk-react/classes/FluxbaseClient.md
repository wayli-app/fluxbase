---
editUrl: false
next: false
prev: false
title: "FluxbaseClient"
---

Main Fluxbase client class

## Type Parameters

| Type Parameter                                      | Default type |
| --------------------------------------------------- | ------------ |
| `Database`                                          | `any`        |
| `_SchemaName` _extends_ `string` & keyof `Database` | `any`        |

## Constructors

### new FluxbaseClient()

> **new FluxbaseClient**\<`Database`, `_SchemaName`\>(`fluxbaseUrl`, `fluxbaseKey`, `options`?): [`FluxbaseClient`](/api/sdk-react/classes/fluxbaseclient/)\<`Database`, `_SchemaName`\>

Create a new Fluxbase client instance

#### Parameters

| Parameter     | Type                    | Description                                                                        |
| ------------- | ----------------------- | ---------------------------------------------------------------------------------- |
| `fluxbaseUrl` | `string`                | The URL of your Fluxbase instance                                                  |
| `fluxbaseKey` | `string`                | The anon key (JWT token with "anon" role). Generate using scripts/generate-keys.sh |
| `options`?    | `FluxbaseClientOptions` | Additional client configuration options                                            |

#### Returns

[`FluxbaseClient`](/api/sdk-react/classes/fluxbaseclient/)\<`Database`, `_SchemaName`\>

#### Example

```typescript
const client = new FluxbaseClient(
  "http://localhost:8080",
  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...", // Anon JWT token
  { timeout: 30000 },
);
```

## Advanced

### http

#### Get Signature

> **get** **http**(): `FluxbaseFetch`

Get the internal HTTP client

Use this for advanced scenarios like making custom API calls or admin operations.

##### Example

```typescript
// Make a custom API call
const data = await client.http.get("/api/custom-endpoint");
```

##### Returns

`FluxbaseFetch`

The internal FluxbaseFetch instance

## Authentication

### getAuthToken()

> **getAuthToken**(): `null` \| `string`

Get the current authentication token

#### Returns

`null` \| `string`

The current JWT access token, or null if not authenticated

---

### setAuthToken()

> **setAuthToken**(`token`): `void`

Set a new authentication token

This updates both the HTTP client and realtime connection with the new token.

#### Parameters

| Parameter | Type               | Description                                                  |
| --------- | ------------------ | ------------------------------------------------------------ |
| `token`   | `null` \| `string` | The JWT access token to set, or null to clear authentication |

#### Returns

`void`

## Database

### from()

> **from**\<`T`\>(`table`): `QueryBuilder`\<`T`\>

Create a query builder for a database table

#### Type Parameters

| Type Parameter | Default type |
| -------------- | ------------ |
| `T`            | `any`        |

#### Parameters

| Parameter | Type     | Description                                               |
| --------- | -------- | --------------------------------------------------------- |
| `table`   | `string` | The table name (can include schema, e.g., 'public.users') |

#### Returns

`QueryBuilder`\<`T`\>

A query builder instance for constructing and executing queries

#### Example

```typescript
// Simple select
const { data } = await client.from("users").select("*").execute();

// With filters
const { data } = await client
  .from("products")
  .select("id, name, price")
  .gt("price", 100)
  .eq("category", "electronics")
  .execute();

// Insert
await client
  .from("users")
  .insert({ name: "John", email: "john@example.com" })
  .execute();
```

---

### schema()

> **schema**(`schemaName`): `SchemaQueryBuilder`

Access a specific database schema

Use this to query tables in non-public schemas.

#### Parameters

| Parameter    | Type     | Description                                 |
| ------------ | -------- | ------------------------------------------- |
| `schemaName` | `string` | The schema name (e.g., 'jobs', 'analytics') |

#### Returns

`SchemaQueryBuilder`

A schema query builder for constructing queries on that schema

#### Example

```typescript
// Query the logging.entries table
const { data } = await client
  .schema("logging")
  .from("entries")
  .select("*")
  .eq("execution_id", executionId)
  .execute();

// Insert into a custom schema table
await client
  .schema("analytics")
  .from("events")
  .insert({ event_type: "click", data: {} })
  .execute();
```

## Other

### admin

> **admin**: `FluxbaseAdmin`

Admin module for instance management (requires admin authentication)

---

### ai

> **ai**: `FluxbaseAI`

AI module for chatbots and conversation history

---

### auth

> **auth**: `FluxbaseAuth`

Authentication module for user management

---

### functions

> **functions**: `FluxbaseFunctions`

Functions module for invoking and managing edge functions

---

### jobs

> **jobs**: `FluxbaseJobs`

Jobs module for submitting and monitoring background jobs

---

### management

> **management**: `FluxbaseManagement`

Management module for client keys, webhooks, and invitations

---

### realtime

> **realtime**: `FluxbaseRealtime`

Realtime module for WebSocket subscriptions

---

### settings

> **settings**: `SettingsClient`

Settings module for reading public application settings (respects RLS policies)

---

### storage

> **storage**: `FluxbaseStorage`

Storage module for file operations

## RPC

### rpc

> **rpc**: `CallableRPC`

RPC module for calling PostgreSQL functions - Supabase compatible

Can be called directly (Supabase-style) or access methods like invoke(), list(), getStatus()

#### Example

```typescript
// Supabase-style direct call (uses 'default' namespace)
const { data, error } = await client.rpc("get_user_orders", { user_id: "123" });

// With full options
const { data, error } = await client.rpc.invoke(
  "get_user_orders",
  { user_id: "123" },
  {
    namespace: "custom",
    async: true,
  },
);

// List available procedures
const { data: procedures } = await client.rpc.list();
```

## Realtime

### channel()

> **channel**(`name`, `config`?): `RealtimeChannel`

Create or get a realtime channel (Supabase-compatible)

#### Parameters

| Parameter | Type                    | Description                    |
| --------- | ----------------------- | ------------------------------ |
| `name`    | `string`                | Channel name                   |
| `config`? | `RealtimeChannelConfig` | Optional channel configuration |

#### Returns

`RealtimeChannel`

RealtimeChannel instance

#### Example

```typescript
const channel = client
  .channel("room-1", {
    broadcast: { self: true },
    presence: { key: "user-123" },
  })
  .on("broadcast", { event: "message" }, (payload) => {
    console.log("Message:", payload);
  })
  .subscribe();
```

---

### removeChannel()

> **removeChannel**(`channel`): `Promise`\<`"error"` \| `"ok"`\>

Remove a realtime channel (Supabase-compatible)

#### Parameters

| Parameter | Type              | Description           |
| --------- | ----------------- | --------------------- |
| `channel` | `RealtimeChannel` | The channel to remove |

#### Returns

`Promise`\<`"error"` \| `"ok"`\>

Promise resolving to status

#### Example

```typescript
const channel = client.channel("room-1");
await client.removeChannel(channel);
```

## Vector Search

### vector

> **vector**: `FluxbaseVector`

Vector search module for pgvector similarity search

Provides convenience methods for vector similarity search:

- `embed()` - Generate embeddings from text
- `search()` - Search for similar vectors with auto-embedding

#### Example

```typescript
// Search with automatic embedding
const { data } = await client.vector.search({
  table: "documents",
  column: "embedding",
  query: "How to use TypeScript?",
  match_count: 10,
});

// Generate embeddings
const { data } = await client.vector.embed({ text: "Hello world" });
```

Note: For more control, use the QueryBuilder methods:

- `vectorSearch()` - Filter and order by vector similarity
- `orderByVector()` - Order results by vector distance
