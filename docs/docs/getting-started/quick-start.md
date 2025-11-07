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
- curl or Postman for API testing
- Basic knowledge of SQL and REST APIs

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

## Step 2: Create a User Account

Sign up a new user:

```bash
curl -X POST http://localhost:8080/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePassword123"
  }'
```

Response:

```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "role": "authenticated"
  },
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2024-10-27T10:15:00Z"
}
```

Save the `access_token` - you'll need it for authenticated requests.

## Step 3: Create Todos

Create your first todo:

```bash
curl -X POST http://localhost:8080/api/v1/tables/todos \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "title": "Learn Fluxbase",
    "description": "Complete the quick start tutorial",
    "priority": "high",
    "due_date": "2024-11-01T18:00:00Z"
  }'
```

Response:

```json
[
  {
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "title": "Learn Fluxbase",
    "description": "Complete the quick start tutorial",
    "completed": false,
    "priority": "high",
    "due_date": "2024-11-01T18:00:00Z",
    "created_at": "2024-10-27T10:00:00Z",
    "updated_at": "2024-10-27T10:00:00Z"
  }
]
```

Create more todos:

```bash
# Create second todo
curl -X POST http://localhost:8080/api/v1/tables/todos \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "title": "Build an app with Fluxbase",
    "priority": "medium"
  }'

# Create third todo
curl -X POST http://localhost:8080/api/v1/tables/todos \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "title": "Deploy to production",
    "priority": "low"
  }'
```

## Step 4: Query Todos

### Get All Todos

```bash
curl http://localhost:8080/api/v1/tables/todos \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

### Filter by Completed Status

```bash
# Get incomplete todos
curl "http://localhost:8080/api/v1/tables/todos?completed=eq.false" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

### Filter by Priority

```bash
# Get high priority todos
curl "http://localhost:8080/api/v1/tables/todos?priority=eq.high" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

### Order and Limit

```bash
# Get latest 5 todos, ordered by creation date
curl "http://localhost:8080/api/v1/tables/todos?order=created_at.desc&limit=5" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

### Complex Query

```bash
# Get incomplete high-priority todos, ordered by due date
curl "http://localhost:8080/api/v1/tables/todos?completed=eq.false&priority=eq.high&order=due_date.asc" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

### Select Specific Fields

```bash
# Only get id, title, and completed status
curl "http://localhost:8080/api/v1/tables/todos?select=id,title,completed" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

## Step 5: Update Todos

Mark a todo as completed:

```bash
curl -X PATCH http://localhost:8080/api/v1/tables/todos \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Prefer: return=representation" \
  -d '{
    "id": "eq.123e4567-e89b-12d3-a456-426614174000",
    "completed": true,
    "updated_at": "2024-10-27T11:00:00Z"
  }'
```

Update multiple fields:

```bash
curl -X PATCH http://localhost:8080/api/v1/tables/todos \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "id": "eq.123e4567-e89b-12d3-a456-426614174000",
    "title": "Learn Fluxbase (Updated)",
    "priority": "medium",
    "updated_at": "2024-10-27T11:00:00Z"
  }'
```

## Step 6: Delete Todos

Delete a specific todo:

```bash
curl -X DELETE "http://localhost:8080/api/v1/tables/todos?id=eq.123e4567-e89b-12d3-a456-426614174000" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

Delete all completed todos:

```bash
curl -X DELETE "http://localhost:8080/api/v1/tables/todos?completed=eq.true" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

## Step 7: Real-time Subscriptions

Subscribe to todo changes via WebSocket.

Create a file `realtime-test.html`:

```html
<!DOCTYPE html>
<html>
  <head>
    <title>Fluxbase Realtime Test</title>
  </head>
  <body>
    <h1>Todo Updates (Real-time)</h1>
    <div id="updates"></div>

    <script>
      const token = "YOUR_ACCESS_TOKEN";
      const ws = new WebSocket(`ws://localhost:8080/realtime?token=${token}`);

      ws.onopen = () => {
        console.log("Connected to Fluxbase realtime");

        // Subscribe to todos table
        ws.send(
          JSON.stringify({
            type: "subscribe",
            channel: "table:public.todos",
          }),
        );
      };

      ws.onmessage = (event) => {
        const message = JSON.parse(event.data);
        console.log("Received:", message);

        if (message.type === "broadcast") {
          const div = document.getElementById("updates");
          const p = document.createElement("p");
          p.textContent = `${message.payload.type}: ${JSON.stringify(message.payload.record)}`;
          div.insertBefore(p, div.firstChild);
        }
      };

      ws.onerror = (error) => {
        console.error("WebSocket error:", error);
      };
    </script>
  </body>
</html>
```

Open this HTML file in your browser, then create/update/delete todos via curl. You'll see real-time updates!

## Step 8: Use the TypeScript SDK

Install the SDK:

```bash
npm install @fluxbase/sdk
```

Create `app.ts`:

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

interface Todo {
  id: string;
  title: string;
  description?: string;
  completed: boolean;
  priority: "low" | "medium" | "high";
  due_date?: string;
  created_at: string;
  updated_at: string;
}

async function main() {
  // Sign in
  const { user, error: authError } = await client.auth.signIn({
    email: "user@example.com",
    password: "SecurePassword123",
  });

  if (authError) {
    console.error("Auth failed:", authError);
    return;
  }

  console.log("Signed in as:", user?.email);

  // Get all todos
  const { data: todos, error } = await client
    .from<Todo>("todos")
    .select("*")
    .order("created_at", { ascending: false })
    .execute();

  if (error) {
    console.error("Query failed:", error);
    return;
  }

  console.log("Todos:", todos);

  // Create a new todo
  const { data: newTodo, error: createError } = await client
    .from<Todo>("todos")
    .insert({
      title: "Todo from TypeScript",
      description: "Created using the SDK",
      priority: "medium",
    })
    .execute();

  if (createError) {
    console.error("Create failed:", createError);
    return;
  }

  console.log("Created todo:", newTodo);

  // Subscribe to real-time updates
  client.realtime
    .channel("table:public.todos")
    .on("INSERT", (payload) => {
      console.log("New todo:", payload.new_record);
    })
    .on("UPDATE", (payload) => {
      console.log("Updated todo:", payload.new_record);
    })
    .on("DELETE", (payload) => {
      console.log("Deleted todo:", payload.old_record);
    })
    .subscribe();

  console.log("Subscribed to real-time updates");
}

main();
```

Run it:

```bash
npx tsx app.ts
```

## Step 9: Explore the Admin UI

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

```bash
# Create storage bucket
curl -X POST http://localhost:8080/api/v1/storage/buckets \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{"name": "todo-attachments", "public": false}'

# Upload a file
curl -X POST http://localhost:8080/api/v1/storage/buckets/todo-attachments/files \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -F "file=@document.pdf" \
  -F "path=todos/todo-123/document.pdf"
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

```bash
# Count total todos
curl "http://localhost:8080/api/aggregate/todos/count?column=*" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"

# Count by priority
curl "http://localhost:8080/api/aggregate/todos/count?column=*&group_by=priority" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"

# Count completed vs incomplete
curl "http://localhost:8080/api/aggregate/todos/count?column=*&group_by=completed" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
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

Call it via RPC:

```bash
curl -X POST http://localhost:8080/api/rpc/get_todo_stats \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{"user_uuid": "550e8400-e29b-41d4-a716-446655440000"}'
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
