---
title: "TypeScript SDK"
---

The Fluxbase TypeScript SDK provides a type-safe, developer-friendly interface to interact with your Fluxbase backend. It supports database operations, authentication, real-time subscriptions, file storage, and more.

## Overview

Official SDKs for building applications with Fluxbase:

### TypeScript / JavaScript

The official TypeScript SDK works in browsers, Node.js, and any JavaScript environment.

**[@fluxbase/sdk](https://www.npmjs.com/package/@fluxbase/sdk)**

- ✅ Framework-agnostic (works with any JS framework)
- ✅ Full TypeScript support with type inference
- ✅ Database queries with PostgREST-compatible API
- ✅ Authentication, realtime, storage, and RPC
- ✅ Works in browsers and Node.js

### React Hooks

React-specific hooks built on top of TanStack Query for optimal React integration.

**[@fluxbase/sdk-react](https://www.npmjs.com/package/@fluxbase/sdk-react)**

- ✅ Idiomatic React hooks
- ✅ Built on TanStack Query (React Query)
- ✅ Automatic caching and refetching
- ✅ Optimistic updates
- ✅ SSR support (Next.js, Remix)

## Getting Started

### Installation

#### For JavaScript/TypeScript Projects

```bash
npm install @fluxbase/sdk
# or
yarn add @fluxbase/sdk
# or
pnpm add @fluxbase/sdk
```

#### For React Projects

If you're building a React application, also install the React hooks package:

```bash
npm install @fluxbase/sdk @fluxbase/sdk-react @tanstack/react-query
```

### Quick Start

#### 1. Initialize the Client

Create a Fluxbase client instance by providing your backend URL:

```typescript
import { createClient } from "@fluxbase/sdk";

const client = createClient(
  "http://localhost:8080", // Your Fluxbase backend URL
  "your-anon-key" // Your anon key (generate with ./scripts/generate-keys.sh)
);
```

#### 2. Query Your Database

```typescript
// Fetch all users
const { data: users, error } = await client.from("users").select("*").execute();

if (error) {
  console.error("Error fetching users:", error);
} else {
  console.log("Users:", users);
}
```

#### 3. Insert Data

```typescript
// Insert a new user
const { data, error } = await client
  .from("users")
  .insert({
    name: "John Doe",
    email: "john@example.com",
    age: 30,
  })
  .execute();
```

#### 4. Filter and Query

```typescript
// Get users older than 25
const { data } = await client
  .from("users")
  .select("id, name, email, age")
  .gt("age", 25)
  .order("name", { ascending: true })
  .execute();

// Get a specific user by email
const { data: user } = await client
  .from("users")
  .select("*")
  .eq("email", "john@example.com")
  .single()
  .execute();
```

### Using with React

For React applications, use the `@fluxbase/sdk-react` package for easy integration with React Query:

```tsx
import { FluxbaseProvider, useFluxbaseQuery } from "@fluxbase/sdk-react";
import { createClient } from "@fluxbase/sdk";

// Create client
const client = createClient("http://localhost:8080", "your-anon-key");

// Wrap your app
function App() {
  return (
    <FluxbaseProvider client={client}>
      <UsersList />
    </FluxbaseProvider>
  );
}

// Use hooks in components
function UsersList() {
  const {
    data: users,
    isLoading,
    error,
  } = useFluxbaseQuery({
    table: "users",
    select: "*",
    orderBy: { column: "name", ascending: true },
  });

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <ul>
      {users?.map((user) => (
        <li key={user.id}>
          {user.name} - {user.email}
        </li>
      ))}
    </ul>
  );
}
```

### Configuration

For production applications, use environment variables:

```typescript
const client = createClient(
  process.env.NEXT_PUBLIC_FLUXBASE_URL || "http://localhost:8080",
  process.env.NEXT_PUBLIC_FLUXBASE_ANON_KEY || "your-anon-key"
);
```

## Next Steps

- [Database Operations](./database) - Learn about queries, filters, aggregations, and batch operations
- [React Hooks](./react-hooks) - Deep dive into React integration
- [Database Branching](./branching) - Create and manage database branches
- [API Reference](/docs/api/sdk/) - Complete TypeScript API documentation
- [React API Reference](/docs/api/sdk-react/) - React hooks API documentation

## Examples

Check out the `/example` directory in the Fluxbase repository for complete working examples:

- Vanilla JavaScript/TypeScript usage
- React application with hooks
- Next.js integration
- Authentication flows
- Real-time subscriptions
- File uploads and storage

## Need Help?

- [GitHub Discussions](https://github.com/fluxbase-eu/fluxbase/discussions) - Ask questions and share ideas
- [GitHub Issues](https://github.com/fluxbase-eu/fluxbase/issues) - Report bugs and request features
