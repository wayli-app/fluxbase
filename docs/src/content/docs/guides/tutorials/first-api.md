---
title: "Build Your First API"
description: "Complete guide to creating a REST API with Fluxbase"
---

This tutorial walks you through building a complete REST API with Fluxbase, from creating your first table to querying data with authentication.

## What You'll Build

A simple task management API with:
- A `tasks` table
- CRUD operations via REST API
- User authentication
- Row-Level Security (RLS) so users only see their own tasks

## Prerequisites

- Fluxbase running locally (see [Quick Start](/getting-started/quick-start/))
- Admin account created
- Node.js installed (for the TypeScript SDK examples)

## Step 1: Create the Tasks Table

### Via the Dashboard

1. Open the admin dashboard at `http://localhost:8080/admin`
2. Navigate to **Tables** in the sidebar
3. Click **Create Table**
4. Enter the table details:
   - **Name**: `tasks`
   - **Schema**: `public`

5. Add the following columns:

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| id | uuid | No | gen_random_uuid() |
| user_id | uuid | No | auth.uid() |
| title | text | No | - |
| description | text | Yes | - |
| completed | boolean | No | false |
| created_at | timestamptz | No | now() |

6. Click **Create Table**

### Via SQL

Alternatively, run this SQL in the dashboard's SQL editor:

```sql
CREATE TABLE public.tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL DEFAULT auth.uid(),
    title TEXT NOT NULL,
    description TEXT,
    completed BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Create an index for efficient user queries
CREATE INDEX idx_tasks_user_id ON public.tasks(user_id);
```

## Step 2: Generate a Client Key

Client keys authenticate your application with Fluxbase.

1. In the dashboard, go to **Settings** > **Client Keys**
2. Click **Create Key**
3. Enter:
   - **Name**: `My App`
   - **Type**: `Public` (for frontend apps) or `Service` (for backend apps)
4. Copy the generated key - you'll need it in the next step

:::tip[Key Types]
- **Public keys** (`anon`): For frontend apps. Users must authenticate.
- **Service keys** (`service_role`): For backend apps. Bypass RLS - use carefully!
:::

## Step 3: Set Up Your Project

Create a new Node.js project:

```bash
mkdir my-fluxbase-app
cd my-fluxbase-app
npm init -y
npm install @fluxbase/sdk
```

Create a file `index.ts`:

```typescript
import { createClient } from '@fluxbase/sdk'

// Replace with your server URL and client key
const fluxbase = createClient(
  'http://localhost:8080',
  'your-client-key-here'
)

async function main() {
  // Test the connection
  const { data, error } = await fluxbase.from('tasks').select('*')

  if (error) {
    console.error('Error:', error.message)
    return
  }

  console.log('Tasks:', data)
}

main()
```

Run it:

```bash
npx tsx index.ts
```

You should see an empty array `[]` since we haven't added any tasks yet.

## Step 4: Enable Row-Level Security

RLS ensures users can only access their own data.

1. In the dashboard, go to **Tables** > **tasks**
2. Click the **RLS Policies** tab
3. First, enable RLS on the table:

```sql
ALTER TABLE public.tasks ENABLE ROW LEVEL SECURITY;
```

4. Create policies for each operation:

```sql
-- Users can view their own tasks
CREATE POLICY "Users can view own tasks"
ON public.tasks FOR SELECT
USING (auth.uid() = user_id);

-- Users can create their own tasks
CREATE POLICY "Users can create own tasks"
ON public.tasks FOR INSERT
WITH CHECK (auth.uid() = user_id);

-- Users can update their own tasks
CREATE POLICY "Users can update own tasks"
ON public.tasks FOR UPDATE
USING (auth.uid() = user_id);

-- Users can delete their own tasks
CREATE POLICY "Users can delete own tasks"
ON public.tasks FOR DELETE
USING (auth.uid() = user_id);
```

## Step 5: Enable User Authentication

Enable signups in the dashboard:

1. Go to **Settings** > **Authentication**
2. Enable **Allow Signups**
3. Save changes

## Step 6: Create a User

Update your `index.ts` to sign up a user:

```typescript
import { createClient } from '@fluxbase/sdk'

const fluxbase = createClient(
  'http://localhost:8080',
  'your-client-key-here'
)

async function main() {
  // Sign up a new user
  const { data: authData, error: authError } = await fluxbase.auth.signUp({
    email: 'user@example.com',
    password: 'securepassword123'
  })

  if (authError) {
    console.error('Auth error:', authError.message)
    return
  }

  console.log('User created:', authData.user?.email)
  console.log('Access token:', authData.access_token)
}

main()
```

Run it to create the user.

## Step 7: Create and Query Tasks

Now let's build a complete example:

```typescript
import { createClient } from '@fluxbase/sdk'

const fluxbase = createClient(
  'http://localhost:8080',
  'your-client-key-here'
)

async function main() {
  // Sign in
  const { data: auth, error: authError } = await fluxbase.auth.signIn({
    email: 'user@example.com',
    password: 'securepassword123'
  })

  if (authError) {
    console.error('Login failed:', authError.message)
    return
  }

  console.log('Logged in as:', auth.user?.email)

  // Create a task
  const { data: newTask, error: createError } = await fluxbase
    .from('tasks')
    .insert({
      title: 'Learn Fluxbase',
      description: 'Complete the first API tutorial'
    })
    .select()
    .single()

  if (createError) {
    console.error('Create error:', createError.message)
    return
  }

  console.log('Created task:', newTask)

  // List all tasks
  const { data: tasks, error: listError } = await fluxbase
    .from('tasks')
    .select('*')
    .order('created_at', { ascending: false })

  if (listError) {
    console.error('List error:', listError.message)
    return
  }

  console.log('All tasks:', tasks)

  // Update a task
  const { data: updated, error: updateError } = await fluxbase
    .from('tasks')
    .update({ completed: true })
    .eq('id', newTask.id)
    .select()
    .single()

  if (updateError) {
    console.error('Update error:', updateError.message)
    return
  }

  console.log('Updated task:', updated)

  // Delete a task
  const { error: deleteError } = await fluxbase
    .from('tasks')
    .delete()
    .eq('id', newTask.id)

  if (deleteError) {
    console.error('Delete error:', deleteError.message)
  } else {
    console.log('Task deleted')
  }
}

main()
```

## Step 8: Test via REST API

You can also use the REST API directly:

```bash
# Get an access token
TOKEN=$(curl -s -X POST http://localhost:8080/auth/v1/token \
  -H "Content-Type: application/json" \
  -H "X-Client-Key: your-client-key-here" \
  -d '{"email":"user@example.com","password":"securepassword123","grant_type":"password"}' \
  | jq -r '.access_token')

# Create a task
curl -X POST http://localhost:8080/api/v1/tables/public/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"My first task","description":"Created via REST API"}'

# List tasks
curl http://localhost:8080/api/v1/tables/public/tasks \
  -H "Authorization: Bearer $TOKEN"

# Filter tasks
curl "http://localhost:8080/api/v1/tables/public/tasks?completed=eq.false" \
  -H "Authorization: Bearer $TOKEN"
```

## Query Operators

The REST API supports PostgREST-compatible operators:

| Operator | Description | Example |
|----------|-------------|---------|
| `eq` | Equal | `?status=eq.active` |
| `neq` | Not equal | `?status=neq.deleted` |
| `gt` | Greater than | `?created_at=gt.2024-01-01` |
| `gte` | Greater or equal | `?priority=gte.5` |
| `lt` | Less than | `?priority=lt.3` |
| `lte` | Less or equal | `?priority=lte.10` |
| `like` | Pattern match | `?title=like.*urgent*` |
| `ilike` | Case-insensitive pattern | `?title=ilike.*URGENT*` |
| `in` | In list | `?status=in.(active,pending)` |
| `is` | Is null/true/false | `?deleted_at=is.null` |

## SDK Query Examples

```typescript
// Filter by column value
const { data } = await fluxbase
  .from('tasks')
  .select('*')
  .eq('completed', false)

// Multiple filters (AND)
const { data } = await fluxbase
  .from('tasks')
  .select('*')
  .eq('completed', false)
  .gte('priority', 5)

// Order and limit
const { data } = await fluxbase
  .from('tasks')
  .select('*')
  .order('created_at', { ascending: false })
  .limit(10)

// Select specific columns
const { data } = await fluxbase
  .from('tasks')
  .select('id, title, completed')

// Count rows
const { count } = await fluxbase
  .from('tasks')
  .select('*', { count: 'exact', head: true })
```

## Next Steps

Congratulations! You've built your first Fluxbase API. Here's what to explore next:

- [Authentication Guide](/guides/authentication/) - Add OAuth, magic links, and 2FA
- [Row-Level Security](/guides/row-level-security/) - Advanced RLS patterns
- [Edge Functions](/guides/edge-functions/) - Add serverless business logic
- [Realtime Subscriptions](/guides/realtime/) - Get live updates via WebSocket
- [TypeScript SDK Guide](/guides/typescript-sdk/) - Complete SDK reference
