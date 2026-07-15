---
title: React SDK Getting Started
description: React hooks for Fluxbase — provider setup, queries, mutations, and auth.
---

# React SDK Getting Started

## Installation

```bash
npm install @nimbleflux/fluxbase-sdk-react @nimbleflux/fluxbase-sdk @tanstack/react-query
# or
bun add @nimbleflux/fluxbase-sdk-react @nimbleflux/fluxbase-sdk @tanstack/react-query
```

## Provider Setup

Wrap your app with the `FluxbaseProvider` and `QueryClientProvider`:

```tsx
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { FluxbaseProvider, createClient } from "@nimbleflux/fluxbase-sdk-react";

const client = createClient({
  url: "https://your-fluxbase-instance.com",
  apiKey: "your-client-key",
});

const queryClient = new QueryClient();

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <FluxbaseProvider client={client}>
        <YourApp />
      </FluxbaseProvider>
    </QueryClientProvider>
  );
}
```

## Data Queries

### useTable — Query a Table

```tsx
import { useTable } from "@nimbleflux/fluxbase-sdk-react";

function ProductList() {
  const { data, isLoading, error } = useTable("products", {
    select: "id, name, price",
    gte: { price: 10 },
    order: { column: "created_at", ascending: false },
    limit: 20,
  });

  if (isLoading) return <p>Loading...</p>;
  if (error) return <p>Error: {error.message}</p>;

  return (
    <ul>
      {data?.map((product) => (
        <li key={product.id}>{product.name} — ${product.price}</li>
      ))}
    </ul>
  );
}
```

### useInsert, useUpdate, useDelete — Mutations

```tsx
import { useInsert, useUpdate, useDelete } from "@nimbleflux/fluxbase-sdk-react";

function ProductActions() {
  const insert = useInsert("products");
  const update = useUpdate("products");
  const deleteMutation = useDelete("products");

  const handleCreate = async () => {
    await insert.mutateAsync({ name: "Widget", price: 29.99 });
  };

  const handlePriceUpdate = async (id: string) => {
    await update.mutateAsync({ id, values: { price: 24.99 } });
  };

  const handleDelete = async (id: string) => {
    await deleteMutation.mutateAsync({ id });
  };

  return <div>{/* your UI */}</div>;
}
```

## Authentication

### useUser, useSignIn, useSignOut

```tsx
import { useUser, useSignIn, useSignOut } from "@nimbleflux/fluxbase-sdk-react";

function AuthBar() {
  const { data: user, isLoading } = useUser();
  const signIn = useSignIn();
  const signOut = useSignOut();

  if (isLoading) return null;

  if (!user) {
    return <button onClick={() => signIn.mutateAsync({ email: "...", password: "..." })}>Sign In</button>;
  }

  return (
    <div>
      <span>{user.email}</span>
      <button onClick={() => signOut.mutateAsync()}>Sign Out</button>
    </div>
  );
}
```

## Realtime

### useTableSubscription

```tsx
import { useTableSubscription } from "@nimbleflux/fluxbase-sdk-react";

function LiveProductList() {
  const { data, isLoading } = useTable("products");
  useTableSubscription("products", (payload) => {
    console.log("Realtime update:", payload);
  });

  if (isLoading) return <p>Loading...</p>;

  return <ul>{data?.map((p) => <li key={p.id}>{p.name}</li>)}</ul>;
}
```

## Storage

### useStorageUpload, useStorageList

```tsx
import { useStorageUpload, useStorageList, useStorageDownload } from "@nimbleflux/fluxbase-sdk-react";

function FileManager() {
  const { data: files } = useStorageList("documents");
  const upload = useStorageUpload("documents");
  const download = useStorageDownload("documents");

  const handleUpload = async (file: File) => {
    await upload.mutateAsync({ path: file.name, file });
  };

  return (
    <div>
      <input type="file" onChange={(e) => e.target.files?.[0] && handleUpload(e.target.files[0])} />
      {files?.map((f) => <div key={f.name}>{f.name}</div>)}
    </div>
  );
}
```

## Available Hooks

### Data
- `useTable`, `useInsert`, `useUpdate`, `useUpsert`, `useDelete` — CRUD operations
- `useGraphQLQuery`, `useGraphQLMutation` — GraphQL queries

### Auth
- `useUser`, `useSession`, `useSignIn`, `useSignUp`, `useSignOut`, `useUpdateUser`
- `useSAMLProviders`, `useSignInWithSAML` — SAML SSO

### Realtime
- `useTableSubscription`, `useTableInserts`, `useTableUpdates`, `useTableDeletes`

### Storage
- `useStorageList`, `useStorageUpload`, `useStorageDownload`, `useStorageDelete`
- `useStoragePublicUrl`, `useStorageSignedUrl`, `useStorageBuckets`

### Functions & Jobs
- `useInvokeFunction`, `useFunctions` — Edge functions
- `useSubmitJob`, `useJobStatus`, `useJobs`, `useCancelJob`, `useRetryJob` — Background jobs

### AI & Knowledge Bases
- `useChatbots`, `useConversations`, `useAIChat` — AI chatbots
- `useKnowledgeBases`, `useKBDocuments`, `useKBSearch` — Knowledge bases

### Admin
- `useAdminAuth`, `useUsers`, `useClientKeys`, `useServiceKeys`
- `useSecrets`, `useMigrations`, `useSchemas`, `useTables` (DDL)
- `useImpersonateUser`, `useStopImpersonation`
- `useWebhooks`, `useAppSettings`, `useSystemSettings`

### Branching
- `useBranches`, `useCreateBranch`, `useDeleteBranch`, `useResetBranch`

## Error Handling

All hooks use TanStack Query — errors are available on the `error` field:

```tsx
const { data, error, isLoading } = useTable("products");

if (error) {
  // error is an Error instance
  console.error(error.message);
}
```
