# @nimbleflux/fluxbase-sdk

Official TypeScript/JavaScript SDK for Fluxbase - Backend as a Service.

[![npm version](https://img.shields.io/npm/v/@nimbleflux/fluxbase-sdk.svg)](https://www.npmjs.com/package/@nimbleflux/fluxbase-sdk)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Features

- **Type-safe** - Full TypeScript support with generated types
- **Database Queries** - PostgREST-compatible query builder with filters, ordering, pagination
- **Aggregations** - Count, sum, avg, min, max with GROUP BY support
- **Batch Operations** - Efficient multi-row insert, update, delete
- **Authentication** - JWT-based auth with automatic token refresh
- **Realtime** - WebSocket subscriptions to database changes
- **Storage** - File upload/download with S3 compatibility
- **RPC** - Call PostgreSQL functions directly
- **Lightweight** - Zero dependencies except fetch polyfill

## Installation

```bash
npm install @nimbleflux/fluxbase-sdk
# or
yarn add @nimbleflux/fluxbase-sdk
# or
pnpm add @nimbleflux/fluxbase-sdk
```

## Quick Start

```typescript
import { createClient } from "@nimbleflux/fluxbase-sdk";

// Create a client
const client = createClient({
  url: "http://localhost:8080",
  auth: {
    autoRefresh: true,
    persist: true,
  },
});

// Authentication
await client.auth.signUp({
  email: "user@example.com",
  password: "secure-password",
});

// Query data
const { data } = await client
  .from("products")
  .select("*")
  .eq("category", "electronics")
  .gte("price", 100)
  .execute();

// Aggregations
const stats = await client
  .from("products")
  .count("*")
  .groupBy("category")
  .execute();

// Realtime subscriptions
client.realtime
  .channel("table:public.products")
  .on("INSERT", (payload) => console.log("New:", payload.new_record))
  .subscribe();

// File upload
await client.storage.from("avatars").upload("user-123.png", file);
```

## Documentation

📚 **[SDK Documentation](../../docs/src/content/docs/sdk/)**

### Core Guides

- **[SDK Reference](../../docs/src/content/docs/sdk/)** - Installation, configuration, and usage
- **[Vector Search](../../docs/src/content/docs/guides/vector-search.md)** - Semantic search with pgvector
- **[AI Chatbots](../../docs/src/content/docs/guides/ai-chatbots.md)** - Build AI-powered chatbots
- **[Knowledge Bases](../../docs/src/content/docs/guides/knowledge-bases.md)** - RAG with document ingestion

### API Reference

- **[TypeScript API Docs](../../docs/src/content/docs/api/sdk/)** - Auto-generated from source code

## Browser & Node.js Support

- **Browsers**: All modern browsers with ES6+ and Fetch API
- **Node.js**: v18+ (native fetch) or v16+ with `cross-fetch` polyfill

## TypeScript Support

Fully typed with TypeScript. Define your schemas for complete type safety:

```typescript
interface Product {
  id: number;
  name: string;
  price: number;
  category: string;
}

const { data } = await client.from<Product>("products").select("*").execute();
// data is typed as Product[]
```

## Examples

See `sdk-react/examples/` for a working example app.

## React Integration

For React applications, use [`@nimbleflux/fluxbase-sdk-react`](../sdk-react/) for hooks and automatic state management:

```bash
# npm
npm install @nimbleflux/fluxbase-sdk @nimbleflux/fluxbase-sdk-react @tanstack/react-query

# pnpm
pnpm add @nimbleflux/fluxbase-sdk @nimbleflux/fluxbase-sdk-react @tanstack/react-query
```

See the **[React SDK README](../sdk-react/README.md)** for details.

## Contributing

Contributions are welcome! Please see the project repository for contribution guidelines.

## License

MIT © Fluxbase

## Links

- [Documentation](../../docs/src/content/docs/sdk/)
- [API Reference](../../docs/src/content/docs/api/sdk/)
- [GitHub](https://github.com/nimbleflux/fluxbase)
- [Issues](https://github.com/nimbleflux/fluxbase/issues)
