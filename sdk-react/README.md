# @nimbleflux/fluxbase-sdk-react

React hooks for Fluxbase - Backend as a Service.

[![npm version](https://img.shields.io/npm/v/@nimbleflux/fluxbase-sdk-react.svg)](https://www.npmjs.com/package/@nimbleflux/fluxbase-sdk-react)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Features

- **React Hooks** - Idiomatic React hooks for all Fluxbase features
- **TanStack Query** - Built on React Query for optimal data fetching
- **Type-safe** - Full TypeScript support
- **Auto-refetch** - Smart cache invalidation and refetching
- **Optimistic Updates** - Instant UI updates
- **SSR Support** - Works with Next.js and other SSR frameworks

## Installation

```bash
# npm
npm install @nimbleflux/fluxbase-sdk @nimbleflux/fluxbase-sdk-react @tanstack/react-query

# pnpm
pnpm add @nimbleflux/fluxbase-sdk @nimbleflux/fluxbase-sdk-react @tanstack/react-query
```

## Quick Start

```tsx
import { createClient } from "@nimbleflux/fluxbase-sdk";
import { FluxbaseProvider, useAuth, useTable } from "@nimbleflux/fluxbase-sdk-react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

// Create clients
const fluxbaseClient = createClient({ url: "http://localhost:8080" });
const queryClient = new QueryClient();

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <FluxbaseProvider client={fluxbaseClient}>
        <YourApp />
      </FluxbaseProvider>
    </QueryClientProvider>
  );
}

function YourApp() {
  const { user, signIn, signOut } = useAuth();
  const { data: products } = useTable("products", (q) =>
    q.select("*").eq("active", true).order("created_at", { ascending: false })
  );

  return (
    <div>
      {user ? (
        <>
          <p>Welcome {user.email}</p>
          <button onClick={signOut}>Sign Out</button>
        </>
      ) : (
        <button
          onClick={() =>
            signIn({ email: "user@example.com", password: "pass" })
          }
        >
          Sign In
        </button>
      )}

      <h2>Products</h2>
      {products?.map((product) => (
        <div key={product.id}>{product.name}</div>
      ))}
    </div>
  );
}
```

## Available Hooks

### Authentication

- `useAuth()` - Complete auth state and methods
- `useUser()` - Current user data
- `useSession()` - Current session
- `useSignIn()` - Sign in mutation
- `useSignUp()` - Sign up mutation
- `useSignOut()` - Sign out mutation
- `useUpdateUser()` - Update user profile

### Database

- `useTable()` - Query table with filters and ordering
- `useFluxbaseQuery()` - Custom query hook
- `useInsert()` - Insert rows
- `useUpdate()` - Update rows
- `useUpsert()` - Insert or update
- `useDelete()` - Delete rows
- `useFluxbaseMutation()` - Generic mutation hook

### Realtime

- `useRealtime()` - Subscribe to database changes
- `useTableSubscription()` - Auto-refetch on changes
- `useTableInserts()` - Listen to inserts
- `useTableUpdates()` - Listen to updates
- `useTableDeletes()` - Listen to deletes

### Storage

- `useStorageUpload()` - Upload files
- `useStorageList()` - List files in bucket
- `useStorageDownload()` - Download files
- `useStorageDelete()` - Delete files
- `useStorageSignedUrl()` - Generate signed URLs
- `useStoragePublicUrl()` - Get public URLs

### RPC (PostgreSQL Functions)

- `useRPC()` - Call PostgreSQL function (query)
- `useRPCMutation()` - Call PostgreSQL function (mutation)
- `useRPCBatch()` - Call multiple functions in parallel

### Admin (Management & Operations)

- `useAdminAuth()` - Admin authentication state and login/logout
- `useUsers()` - User management with pagination and CRUD
- `useAPIKeys()` - API key creation and management
- `useWebhooks()` - Webhook configuration and delivery monitoring
- `useAppSettings()` - Application-wide settings management
- `useSystemSettings()` - System key-value settings storage

📚 **[Complete Admin Hooks Guide](./README-ADMIN.md)** - Comprehensive admin dashboard documentation

## Documentation

📚 **[React SDK Reference](../../docs/src/content/docs/api/sdk-react/)**

### Core Guides

- **[SDK Reference](../../docs/src/content/docs/sdk/)** - Core SDK usage and configuration
- **[Admin Hooks Guide](./README-ADMIN.md)** - Comprehensive admin dashboard documentation
- **[Vector Search](../../docs/src/content/docs/guides/vector-search.md)** - Semantic search

### API Reference

- **[React Hooks API](../../docs/src/content/docs/api/sdk-react/)** - Auto-generated from source code
- **[Core SDK API](../../docs/src/content/docs/api/sdk/)** - Core TypeScript SDK reference

## TypeScript Support

All hooks are fully typed. Define your table schemas for complete type safety:

```typescript
interface Product {
  id: string
  name: string
  price: number
  category: string
}

function ProductList() {
  const { data } = useTable<Product>('products', (q) => q.select('*'))
  // data is typed as Product[] | undefined
  return <div>{data?.[0]?.name}</div>
}
```

## Examples

See `sdk-react/examples/` for a working example app.

## Contributing

Contributions are welcome! Please see the project repository for contribution guidelines.

## License

MIT © Fluxbase

## Links

- [Documentation](../../docs/src/content/docs/sdk/)
- [API Reference](../../docs/src/content/docs/api/sdk-react/)
- [Core SDK](../sdk/)
- [GitHub](https://github.com/nimbleflux/fluxbase)
- [Issues](https://github.com/nimbleflux/fluxbase/issues)
