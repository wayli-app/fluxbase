# @fluxbase/sdk

Official TypeScript/JavaScript SDK for Fluxbase - Backend as a Service.

[![npm version](https://img.shields.io/npm/v/@fluxbase/sdk.svg)](https://www.npmjs.com/package/@fluxbase/sdk)
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
npm install @fluxbase/sdk
# or
yarn add @fluxbase/sdk
# or
pnpm add @fluxbase/sdk
```

## Quick Start

```typescript
import { createClient } from '@fluxbase/sdk'

// Create a client
const client = createClient({
  url: 'http://localhost:8080',
  auth: {
    autoRefresh: true,
    persist: true,
  },
})

// Authentication
await client.auth.signUp({
  email: 'user@example.com',
  password: 'secure-password',
})

// Query data
const { data } = await client
  .from('products')
  .select('*')
  .eq('category', 'electronics')
  .gte('price', 100)
  .execute()

// Aggregations
const stats = await client
  .from('products')
  .count('*')
  .groupBy('category')
  .execute()

// Realtime subscriptions
client.realtime
  .channel('table:public.products')
  .on('INSERT', (payload) => console.log('New:', payload.new_record))
  .subscribe()

// File upload
await client.storage
  .from('avatars')
  .upload('user-123.png', file)
```

## Documentation

📚 **[Complete Documentation](../../docs/docs/sdks/getting-started.md)**

### Core Guides
- **[Getting Started](../../docs/docs/sdks/getting-started.md)** - Installation, configuration, and basic usage
- **[Database Operations](../../docs/docs/sdks/database.md)** - Queries, filters, aggregations, batch operations, and RPC
- **[React Hooks](../../docs/docs/sdks/react-hooks.md)** - React integration with `@fluxbase/sdk-react`

### API Reference
- **[TypeScript API Docs](../../docs/static/api/sdk/)** - Auto-generated from source code

## Browser & Node.js Support

- **Browsers**: All modern browsers with ES6+ and Fetch API
- **Node.js**: v18+ (native fetch) or v16+ with `cross-fetch` polyfill

## TypeScript Support

Fully typed with TypeScript. Define your schemas for complete type safety:

```typescript
interface Product {
  id: number
  name: string
  price: number
  category: string
}

const { data } = await client.from<Product>('products').select('*').execute()
// data is typed as Product[]
```

## Examples

Check out working examples in the [`/example`](../example/) directory:
- Vanilla JavaScript/TypeScript
- React with hooks
- Next.js integration
- Vue 3 integration

## React Integration

For React applications, use [`@fluxbase/sdk-react`](../sdk-react/) for hooks and automatic state management:

```bash
npm install @fluxbase/sdk @fluxbase/sdk-react @tanstack/react-query
```

See the **[React Hooks Guide](../../docs/docs/sdks/react-hooks.md)** for details.

## Contributing

Contributions are welcome! Please read our [Contributing Guide](../../CONTRIBUTING.md) for details.

## License

MIT © Fluxbase

## Links

- [Documentation](../../docs/docs/sdks/getting-started.md)
- [API Reference](../../docs/static/api/sdk/)
- [GitHub](https://github.com/wayli-app/fluxbase)
- [Issues](https://github.com/wayli-app/fluxbase/issues)
