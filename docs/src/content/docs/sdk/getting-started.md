---
title: Getting Started
description: Install the Fluxbase TypeScript SDK and make your first query in minutes.
---

# TypeScript SDK Getting Started

## Installation

```bash
npm install @nimbleflux/fluxbase-sdk
# or
bun add @nimbleflux/fluxbase-sdk
# or
yarn add @nimbleflux/fluxbase-sdk
```

## Quick Start

```typescript
import { createClient } from "@nimbleflux/fluxbase-sdk";

const client = createClient(
  "https://your-fluxbase-instance.com",
  "your-api-key", // client key (publishable) or service key (secret)
);

// Query a table
const { data, error } = await client
  .from("products")
  .select("id, name, price")
  .gte("price", 10)
  .order("created_at", { ascending: false })
  .limit(20);

if (error) {
  console.error("Query failed:", error);
} else {
  console.log("Products:", data);
}
```

## Authentication

### Sign Up / Sign In

```typescript
// Create a new user
const { data: signUpData, error: signUpError } = await client.auth.signUp({
  email: "user@example.com",
  password: "secure-password",
});

// Sign in
const { session, error } = await client.auth.signIn({
  email: "user@example.com",
  password: "secure-password",
});

// Session is automatically stored and sent with subsequent requests
```

### OAuth

```typescript
// Get OAuth URL for Google
const { data, error } = await client.auth.getOAuthUrl({
  provider: "google",
  redirectUrl: "http://localhost:3000/callback",
});

window.location.href = data.url;
```

## Data Operations

### Insert

```typescript
const { data, error } = await client
  .from("products")
  .insert({ name: "Widget", price: 29.99 })
  .select();
```

### Update

```typescript
const { data, error } = await client
  .from("products")
  .update({ price: 24.99 })
  .eq("id", productId)
  .select();
```

### Delete

```typescript
const { error } = await client
  .from("products")
  .delete()
  .eq("id", productId);
```

## Realtime

```typescript
const channel = client
  .realtime
  .channel("products-changes")
  .on("postgres_changes", { table: "products" }, (payload) => {
    console.log("Change:", payload);
  })
  .subscribe();
```

## Storage

```typescript
// Upload a file
const { error } = await client.storage
  .from("documents")
  .upload("report.pdf", fileBlob);

// Download a file
const { data, error } = await client.storage
  .from("documents")
  .download("report.pdf");
```

## Error Handling

All SDK methods return a `{ data, error }` result type. Check for errors explicitly:

```typescript
const { data, error } = await client.from("users").select("*");

if (error) {
  // Handle error — error is always an Error object with status/code
  console.error(error.message);
  return;
}

// Use data — TypeScript knows it's defined here
console.log(data);
```

## Next Steps

- [Advanced Features](../advanced-features) — Aggregations, vector search, batch operations
- [Admin SDK](../admin) — Admin operations (users, client keys, webhooks)
- [Management SDK](../management) — Client keys, webhooks, invitations
- [Settings SDK](../settings) — App and system settings
- [DDL SDK](../ddl) — Schema and table management
