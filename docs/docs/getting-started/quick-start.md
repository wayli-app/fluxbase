---
sidebar_position: 2
---

# Quick Start

Build your first application with Fluxbase in 10 minutes. This tutorial walks you through creating a simple todo list application.

## What We'll Build

A todo list application with:

- ✅ REST API for CRUD operations
- ✅ Real-time updates via WebSockets
- ✅ User authentication
- ✅ File attachments (storage)

## Prerequisites

- Fluxbase installed and running ([Installation Guide](./installation.md))
- PostgreSQL database set up
- Node.js 16+ and npm/yarn
- Basic knowledge of SQL and TypeScript

## Step 1: Create the Database Schema

Connect to your PostgreSQL database:

```bash
psql postgres://fluxbase:password@localhost:5432/fluxbase
```

Create the todos table:

```sql
CREATE TABLE todos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  description TEXT,
  completed BOOLEAN DEFAULT false,
  priority TEXT CHECK (priority IN ('low', 'medium', 'high')) DEFAULT 'medium',
  due_date TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create index for faster queries
CREATE INDEX IF NOT EXISTS idx_todos_user_id ON todos(user_id);
CREATE INDEX IF NOT EXISTS idx_todos_completed ON todos(completed);

-- Enable realtime for this table
SELECT enable_realtime('todos');
```

Exit psql:

```sql
\q
```

## Step 2: Install the TypeScript SDK

Install the Fluxbase SDK in your project:

```bash
npm install @fluxbase/sdk
# or
yarn add @fluxbase/sdk
```

Create a new file `app.ts` to get started.

## Step 3: Generate Anon Key

Fluxbase uses anon keys (JWT tokens) for client initialization, just like Supabase. Generate one first:

```bash
./scripts/generate-keys.sh
# Select option 3: "Generate Anon Key"
# Copy the generated JWT token
```

This creates a JWT token with the "anon" role that respects Row-Level Security policies.

## Step 4: Initialize Client and Sign Up

Set up the Fluxbase client and create a user account:

```typescript
import { createClient } from "@fluxbase/sdk";

// Initialize client (identical to Supabase)
const client = createClient(
  "http://localhost:8080",
  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." // Your anon key from step 3
);

// Sign up a new user
const { data, error: signUpError } = await client.auth.signUp({
  email: "user@example.com",
  password: "SecurePassword123",
});

if (signUpError) {
  console.error("Sign up failed:", signUpError);
} else {
  console.log("Signed up as:", data.user.email);
}
```

The SDK automatically manages authentication tokens for you. After sign-up or sign-in, the user's JWT token replaces the anon key for authenticated requests.

## Step 5: CRUD Operations

```typescript
interface Todo {
  title: string;
  priority: "low" | "medium" | "high";
  completed?: boolean;
}

// Create
const { data: newTodo } = await client
  .from<Todo>("todos")
  .insert({ title: "Learn Fluxbase", priority: "high" })
  .execute();

// Read (query)
const { data: todos } = await client
  .from<Todo>("todos")
  .select("*")
  .eq("completed", false)
  .order("created_at", { ascending: false })
  .execute();

// Update
await client
  .from<Todo>("todos")
  .update({ completed: true })
  .eq("id", newTodo?.id)
  .execute();

// Delete
await client.from<Todo>("todos").delete().eq("id", newTodo?.id).execute();
```

## Step 6: Real-time Subscriptions

```typescript
const channel = client.realtime.channel("table:public.todos");

channel
  .on("INSERT", (payload) => console.log("Created:", payload.new_record))
  .on("UPDATE", (payload) => console.log("Updated:", payload.new_record))
  .on("DELETE", (payload) => console.log("Deleted:", payload.old_record))
  .subscribe();
```

## Step 7: Run Your Application

```bash
npm install -D typescript tsx
npx tsx app.ts
```

## Step 8: Explore Admin UI

Open http://localhost:8080/admin for:

- Tables Browser - View/edit data
- API Explorer - Test endpoints
- Realtime Dashboard - Monitor WebSockets
- Storage Browser - Manage files
- Authentication - Manage users
- System Monitoring - View logs/metrics

## Next Steps

**Add file attachments:**

```typescript
await client.storage.createBucket({ name: "todo-attachments" });
await client.storage.from("todo-attachments").upload("file.pdf", file);
```

**Add Row-Level Security:**

```sql
ALTER TABLE todos ENABLE ROW LEVEL SECURITY;

CREATE POLICY todos_select_policy ON todos
  FOR SELECT
  USING (user_id = current_setting('app.user_id', true)::uuid);
```

**Custom RPC functions:**

```typescript
const { data } = await client.rpc("get_todo_stats", { user_uuid: user?.id });
```

## Learn More

- [Database Operations](../guides/typescript-sdk/database.md)
- [Authentication](../guides/authentication.md)
- [Realtime](../guides/realtime.md)
- [Storage](../guides/storage.md)
- [TypeScript SDK](../api/sdk/)
