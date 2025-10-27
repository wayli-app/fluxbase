# Fluxbase SDK Examples

This directory contains practical examples demonstrating how to use the Fluxbase TypeScript SDK.

## Examples

### 1. [Quickstart](./01-quickstart.ts)
Get started quickly with the basics:
- Client initialization
- User authentication (sign up, sign in, sign out)
- Database queries (insert, select, update, delete)
- Realtime subscriptions
- File storage operations
- RPC function calls

### 2. [Database Operations](./02-database.ts)
Comprehensive database operations:
- CRUD operations with full TypeScript support
- Advanced filtering (eq, neq, gt, gte, lt, lte, in, like, is)
- Sorting and pagination
- Aggregations (count, sum, avg, min, max)
- Batch operations (insertMany, updateMany, deleteMany)
- Upsert operations

## Running the Examples

### Prerequisites

1. **Start Fluxbase server**:
```bash
cd /workspace
make run
```

2. **Install dependencies** (if running examples standalone):
```bash
npm install @fluxbase/sdk
```

### Run an Example

```bash
# Using ts-node
npx ts-node examples/01-quickstart.ts

# Or compile and run
npx tsc examples/01-quickstart.ts
node examples/01-quickstart.js
```

### Run Examples in Your Project

```typescript
import { createClient } from '@fluxbase/sdk'

const client = createClient({
  url: process.env.FLUXBASE_URL || 'http://localhost:8080',
  auth: {
    autoRefresh: true,
    persist: true,
  },
})

// Use the client...
```

## TypeScript Support

All examples include full TypeScript types. Define your database schema for complete type safety:

```typescript
interface Product {
  id?: number
  name: string
  price: number
  category: string
  in_stock: boolean
  created_at?: string
}

// Fully typed queries
const { data } = await client
  .from<Product>('products')
  .select('*')
  .eq('category', 'electronics')
  .execute()

// data is typed as Product[]
```

## Error Handling

All SDK methods return a result object with `data` and `error` properties:

```typescript
const { data, error } = await client
  .from('products')
  .select('*')
  .execute()

if (error) {
  console.error('Query failed:', error.message)
  return
}

console.log('Products:', data)
```

## Environment Variables

Create a `.env` file for configuration:

```bash
FLUXBASE_URL=http://localhost:8080
FLUXBASE_ANON_KEY=your-anon-key  # Optional
```

## Additional Resources

- **[SDK Documentation](../../docs/docs/sdks/getting-started.md)** - Complete SDK guide
- **[API Reference](../../docs/static/api/sdk/)** - Auto-generated API docs
- **[Fluxbase Documentation](../../docs/docs/)** - Full platform docs

## Contributing

Have a useful example? Submit a PR! Examples should be:
- **Practical** - Solve real-world problems
- **Well-documented** - Explain what's happening
- **Runnable** - Work out of the box
- **Type-safe** - Demonstrate TypeScript features
