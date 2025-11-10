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
CREATE INDEX idx_todos_user_id ON todos(user_id);
CREATE INDEX idx_todos_completed ON todos(completed);

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

## Step 3: Initialize Client and Sign Up

Set up the Fluxbase client and create a user account:

```typescript
import { createClient } from "@fluxbase/sdk";

// Initialize client
const client = createClient({
  url: "http://localhost:8080",
  auth: {
    autoRefresh: true,
    persist: true,
  },
});

// Sign up a new user
const { user, error: signUpError } = await client.auth.signUp({
  email: "user@example.com",
  password: "SecurePassword123",
});

if (signUpError) {
  console.error("Sign up failed:", signUpError);
} else {
  console.log("Signed up as:", user?.email);
}
```

The SDK automatically manages authentication tokens for you.

## Step 4: Create Todos

Define the Todo type and create some todos:

```typescript
interface Todo {
  id?: string;
  user_id?: string;
  title: string;
  description?: string;
  completed?: boolean;
  priority: "low" | "medium" | "high";
  due_date?: string;
  created_at?: string;
  updated_at?: string;
}

// Create your first todo
const { data: newTodo, error: createError } = await client
  .from<Todo>("todos")
  .insert({
    title: "Learn Fluxbase",
    description: "Complete the quick start tutorial",
    priority: "high",
    due_date: "2024-11-01T18:00:00Z",
  })
  .execute();

if (createError) {
  console.error("Create failed:", createError);
} else {
  console.log("Created todo:", newTodo);
}

// Create more todos
await client
  .from<Todo>("todos")
  .insert([
    { title: "Build an app with Fluxbase", priority: "medium" },
    { title: "Deploy to production", priority: "low" },
  ])
  .execute();
```

## Step 5: Query Todos

Query todos with various filters:

```typescript
// Get all todos
const { data: todos, error } = await client
  .from<Todo>("todos")
  .select("*")
  .execute();

console.log("All todos:", todos);

// Get incomplete todos only
const { data: incomplete } = await client
  .from<Todo>("todos")
  .select("*")
  .eq("completed", false)
  .execute();

// Get high priority todos
const { data: highPriority } = await client
  .from<Todo>("todos")
  .select("*")
  .eq("priority", "high")
  .execute();

// Get latest 5 todos, ordered by creation date
const { data: recent } = await client
  .from<Todo>("todos")
  .select("*")
  .order("created_at", { ascending: false })
  .limit(5)
  .execute();

// Complex query: incomplete high-priority todos, ordered by due date
const { data: urgent } = await client
  .from<Todo>("todos")
  .select("*")
  .eq("completed", false)
  .eq("priority", "high")
  .order("due_date", { ascending: true })
  .execute();

// Select specific fields only
const { data: simplified } = await client
  .from<Todo>("todos")
  .select("id, title, completed")
  .execute();
```

## Step 6: Update Todos

Update existing todos:

```typescript
// Mark a todo as completed
const { data: updated, error: updateError } = await client
  .from<Todo>("todos")
  .update({ completed: true })
  .eq("id", "123e4567-e89b-12d3-a456-426614174000")
  .execute();

if (updateError) {
  console.error("Update failed:", updateError);
} else {
  console.log("Updated todo:", updated);
}

// Update multiple fields
await client
  .from<Todo>("todos")
  .update({
    title: "Learn Fluxbase (Updated)",
    priority: "medium",
  })
  .eq("id", "123e4567-e89b-12d3-a456-426614174000")
  .execute();
```

## Step 7: Delete Todos

Delete todos from the database:

```typescript
// Delete a specific todo
const { error: deleteError } = await client
  .from<Todo>("todos")
  .delete()
  .eq("id", "123e4567-e89b-12d3-a456-426614174000")
  .execute();

if (deleteError) {
  console.error("Delete failed:", deleteError);
} else {
  console.log("Todo deleted successfully");
}

// Delete all completed todos
await client.from<Todo>("todos").delete().eq("completed", true).execute();
```

## Step 8: Real-time Subscriptions

Subscribe to todo changes via WebSocket to receive real-time updates:

```typescript
// Subscribe to real-time updates on the todos table
const channel = client.realtime.channel("table:public.todos");

channel
  .on("INSERT", (payload) => {
    console.log("New todo created:", payload.new_record);
  })
  .on("UPDATE", (payload) => {
    console.log("Todo updated:", payload.new_record);
    console.log("Previous state:", payload.old_record);
  })
  .on("DELETE", (payload) => {
    console.log("Todo deleted:", payload.old_record);
  })
  .subscribe();

console.log("Subscribed to real-time updates!");

// Now any changes to the todos table will trigger the callbacks above
// Try creating, updating, or deleting a todo to see real-time updates
```

## Step 9: Run Your Application

Save all the code above in `app.ts` and run it:

```bash
# Install TypeScript and ts-node if needed
npm install -D typescript tsx

# Run the application
npx tsx app.ts
```

You should see your todos and real-time updates as you make changes!

## Step 10: Explore the Admin UI

Open http://localhost:8080/admin in your browser.

The Admin UI provides:

1. **Tables Browser** - View and edit todos directly
2. **API Explorer** - Test API endpoints with a Postman-like interface
3. **Realtime Dashboard** - Monitor WebSocket connections
4. **Storage Browser** - Manage file uploads
5. **Authentication** - Manage users and sessions
6. **System Monitoring** - View logs and metrics

## Next Steps

Congratulations! You've built a complete todo application with Fluxbase. Here's what to explore next:

### Add File Attachments

```typescript
// Create storage bucket
const { data: bucket, error: bucketError } = await client.storage.createBucket({
  name: "todo-attachments",
  public: false,
});

// Upload a file
const file = new File(["content"], "document.pdf", { type: "application/pdf" });
const { data: uploadedFile, error: uploadError } = await client.storage
  .from("todo-attachments")
  .upload("todos/todo-123/document.pdf", file);

if (uploadError) {
  console.error("Upload failed:", uploadError);
} else {
  console.log("File uploaded:", uploadedFile);
}
```

### Add Row-Level Security

```sql
-- Enable RLS
ALTER TABLE todos ENABLE ROW LEVEL SECURITY;

-- Policy: Users can only see their own todos
CREATE POLICY todos_select_policy ON todos
  FOR SELECT
  USING (user_id = current_setting('app.user_id', true)::uuid);

-- Policy: Users can only insert their own todos
CREATE POLICY todos_insert_policy ON todos
  FOR INSERT
  WITH CHECK (user_id = current_setting('app.user_id', true)::uuid);

-- Policy: Users can only update their own todos
CREATE POLICY todos_update_policy ON todos
  FOR UPDATE
  USING (user_id = current_setting('app.user_id', true)::uuid);

-- Policy: Users can only delete their own todos
CREATE POLICY todos_delete_policy ON todos
  FOR DELETE
  USING (user_id = current_setting('app.user_id', true)::uuid);
```

### Add Aggregations

```typescript
// Count total todos
const { data: totalCount } = await client
  .from("todos")
  .select("*", { count: "exact", head: true })
  .execute();

console.log("Total todos:", totalCount);

// For more complex aggregations, use RPC functions (see below)
```

### Create Custom RPC Functions

```sql
CREATE OR REPLACE FUNCTION get_todo_stats(user_uuid UUID)
RETURNS JSON AS $$
DECLARE
  result JSON;
BEGIN
  SELECT json_build_object(
    'total', COUNT(*),
    'completed', COUNT(*) FILTER (WHERE completed = true),
    'incomplete', COUNT(*) FILTER (WHERE completed = false),
    'high_priority', COUNT(*) FILTER (WHERE priority = 'high'),
    'overdue', COUNT(*) FILTER (WHERE due_date < NOW() AND completed = false)
  )
  INTO result
  FROM todos
  WHERE user_id = user_uuid;

  RETURN result;
END;
$$ LANGUAGE plpgsql;
```

Call it via RPC using the SDK:

```typescript
interface TodoStats {
  total: number;
  completed: number;
  incomplete: number;
  high_priority: number;
  overdue: number;
}

const { data: stats, error } = await client.rpc<TodoStats>("get_todo_stats", {
  user_uuid: user?.id,
});

if (error) {
  console.error("RPC call failed:", error);
} else {
  console.log("Todo statistics:", stats);
}
```

## Learn More

- [Database Operations](../guides/typescript-sdk/database.md) - Master the query builder
- [Authentication](../guides/authentication.md) - Secure your application
- [Realtime](../guides/realtime.md) - Build collaborative features
- [Storage](../guides/storage.md) - Handle file uploads
- [React Hooks](../guides/typescript-sdk/react-hooks.md) - Build React applications
- [TypeScript SDK API Reference](../api/sdk/) - Complete SDK API documentation

## Example Applications

Check out complete example applications in the [Fluxbase repository](https://github.com/wayli-app/fluxbase/tree/main/examples):

- Todo App (this tutorial)
- Blog Platform
- Chat Application
- E-commerce Store
- Social Media Feed

## Need Help?

- **GitHub Issues**: [github.com/wayli-app/fluxbase/issues](https://github.com/wayli-app/fluxbase/issues)
- **GitHub Discussions**: [github.com/wayli-app/fluxbase/discussions](https://github.com/wayli-app/fluxbase/discussions)
- **Discord**: [discord.gg/fluxbase](https://discord.gg/fluxbase)
