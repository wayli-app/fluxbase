---
title: "Getting Started with Fluxbase SDK"
---

The Fluxbase TypeScript SDK provides a type-safe, developer-friendly interface to interact with your Fluxbase backend. It supports database operations, authentication, real-time subscriptions, file storage, and more.

## Installation

### For JavaScript/TypeScript Projects

```bash
npm install @fluxbase/sdk
# or
yarn add @fluxbase/sdk
# or
pnpm add @fluxbase/sdk
```

### For React Projects

If you're building a React application, also install the React hooks package:

```bash
npm install @fluxbase/sdk @fluxbase/sdk-react @tanstack/react-query
```

## Quick Start

### 1. Initialize the Client

Create a Fluxbase client instance by providing your backend URL:

```typescript
import { createClient } from "@fluxbase/sdk";

const client = createClient(
  "http://localhost:8080", // Your Fluxbase backend URL
  "your-anon-key" // Your anon key (generate with ./scripts/generate-keys.sh)
);
```

### 2. Query Your Database

```typescript
// Fetch all users
const { data: users, error } = await client.from("users").select("*").execute();

if (error) {
  console.error("Error fetching users:", error);
} else {
  console.log("Users:", users);
}
```

### 3. Insert Data

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

### 4. Filter and Query

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

### 5. Update Data

```typescript
// Update user by ID
const { data } = await client
  .from("users")
  .eq("id", 123)
  .update({ age: 31 })
  .execute();
```

### 6. Delete Data

```typescript
// Delete user by ID
await client.from("users").eq("id", 123).delete().execute();
```

## Using with React

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

## Configuration Options

### Client Parameters

The `createClient` function accepts two parameters:

```typescript
createClient(
  url: string,      // Backend URL (required)
  apiKey: string    // API key or anon key (required)
)
```

**Parameters:**

- `url`: Your Fluxbase backend URL
- `apiKey`: Your API key (anon key for client-side, service role key for server-side)

### Environment Variables

For production applications, use environment variables:

```typescript
const client = createClient(
  process.env.NEXT_PUBLIC_FLUXBASE_URL || "http://localhost:8080",
  process.env.NEXT_PUBLIC_FLUXBASE_ANON_KEY || "your-anon-key"
);
```

## Next Steps

- [Database Operations](./database.md) - Learn about queries, filters, aggregations, and batch operations
- [React Hooks](./react-hooks.md) - Deep dive into React integration
- [API Reference](../../api/sdk/) - Complete API documentation

## Examples

Check out the `/example` directory in the Fluxbase repository for complete working examples:

- Vanilla JavaScript/TypeScript usage
- React application with hooks
- Next.js integration
- Authentication flows
- Real-time subscriptions
- File uploads and storage

## Support

- GitHub Issues: [github.com/fluxbase-eu/fluxbase/issues](https://github.com/fluxbase-eu/fluxbase/issues)
- Documentation: [https://fluxbase.eu](https://fluxbase.eu)
