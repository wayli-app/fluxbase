# API Cookbook

Production-ready code examples for common Fluxbase use cases.

## Table of Contents

1. [Getting Started](#getting-started)
2. [Authentication](#authentication)
3. [Database Operations](#database-operations)
4. [Query Operators](#query-operators)
5. [Realtime Subscriptions](#realtime-subscriptions)
6. [Storage](#storage)
7. [Row-Level Security](#row-level-security)
8. [Edge Functions](#edge-functions)

## Getting Started

### Installation

```bash
npm install @fluxbase/sdk
```

### Initialize Client

```typescript
import { createClient } from "@fluxbase/sdk";

const client = createClient("http://localhost:8080", "your-anon-key");
```

## Authentication

### Sign Up

```typescript
const { user, session } = await client.auth.signUp({
  email: "user@example.com",
  password: "SecurePass123",
  metadata: { name: "John Doe" },
});
```

### Sign In

```typescript
const { user, session } = await client.auth.signIn({
  email: "user@example.com",
  password: "SecurePass123",
});
```

### Sign Out

```typescript
await client.auth.signOut();
```

### Get Current User

```typescript
const user = await client.auth.getCurrentUser();
if (user) {
  console.log("Logged in as:", user.email);
}
```

### Password Reset

```typescript
// Request reset
await client.auth.resetPassword({ email: "user@example.com" });

// User receives email with token, then:
await client.auth.confirmPasswordReset({
  token: "reset-token",
  password: "NewPassword123",
});
```

## Database Operations

### Select All Rows

```typescript
const { data, error } = await client.from("posts").select("*");
```

### Select Specific Columns

```typescript
const { data } = await client.from("posts").select("id, title, author_id");
```

### Select with Joins

```typescript
const { data } = await client
  .from("posts")
  .select("id, title, author(name, email)");
```

### Insert Single Row

```typescript
const { data } = await client.from("posts").insert({
  title: "My Post",
  content: "Post content",
  published: true,
});
```

### Insert Multiple Rows

```typescript
const { data } = await client.from("posts").insert([
  { title: "Post 1", content: "Content 1" },
  { title: "Post 2", content: "Content 2" },
]);
```

### Update Rows

```typescript
const { data } = await client
  .from("posts")
  .update({ published: true })
  .eq("author_id", userId);
```

### Delete Rows

```typescript
const { data } = await client.from("posts").delete().eq("id", postId);
```

### Upsert (Insert or Update)

```typescript
const { data } = await client.from("posts").upsert({
  id: "existing-id",
  title: "Updated Title",
});
```

## Query Operators

### Equality

```typescript
// Equal
await client.from("posts").select("*").eq("status", "published");

// Not equal
await client.from("posts").select("*").neq("status", "draft");
```

### Comparison

```typescript
// Greater than
await client.from("posts").select("*").gt("views", 1000);

// Greater than or equal
await client.from("posts").select("*").gte("views", 100);

// Less than
await client.from("posts").select("*").lt("views", 50);

// Less than or equal
await client.from("posts").select("*").lte("views", 10);
```

### Pattern Matching

```typescript
// Like (case-sensitive)
await client.from("posts").select("*").like("title", "%tutorial%");

// ILike (case-insensitive)
await client.from("posts").select("*").ilike("title", "%TUTORIAL%");
```

### IN Operator

```typescript
await client.from("posts").select("*").in("status", ["published", "featured"]);
```

### IS NULL / IS NOT NULL

```typescript
// Null
await client.from("posts").select("*").is("deleted_at", null);

// Not null
await client.from("posts").select("*").not("deleted_at", "is", null);
```

### Ordering

```typescript
// Ascending
await client.from("posts").select("*").order("created_at", { ascending: true });

// Descending
await client.from("posts").select("*").order("views", { ascending: false });

// Multiple columns
await client
  .from("posts")
  .select("*")
  .order("featured", { ascending: false })
  .order("created_at", { ascending: false });
```

### Pagination

```typescript
// Limit
await client.from("posts").select("*").limit(10);

// Offset
await client.from("posts").select("*").limit(10).offset(20);

// Range
await client.from("posts").select("*").range(0, 9); // First 10 rows
```

### Combining Filters

```typescript
const { data } = await client
  .from("posts")
  .select("*")
  .eq("published", true)
  .gte("views", 100)
  .order("created_at", { ascending: false })
  .limit(20);
```

## Realtime Subscriptions

### Subscribe to Table Changes

```typescript
const channel = client.realtime
  .channel("table:public.posts")
  .on("INSERT", (payload) => {
    console.log("New post:", payload.new_record);
  })
  .on("UPDATE", (payload) => {
    console.log("Updated post:", payload.new_record);
  })
  .on("DELETE", (payload) => {
    console.log("Deleted post:", payload.old_record);
  })
  .subscribe();

// Cleanup
channel.unsubscribe();
```

### Subscribe to All Events

```typescript
const channel = client.realtime
  .channel("table:public.posts")
  .on("*", (payload) => {
    console.log("Event:", payload.type, payload);
  })
  .subscribe();
```

### React Hook

```typescript
import { useEffect, useState } from "react";

function usePosts() {
  const [posts, setPosts] = useState([]);

  useEffect(() => {
    // Initial load
    client
      .from("posts")
      .select("*")
      .then(({ data }) => setPosts(data));

    // Subscribe to changes
    const channel = client.realtime
      .channel("table:public.posts")
      .on("INSERT", ({ new_record }) => {
        setPosts((prev) => [...prev, new_record]);
      })
      .on("UPDATE", ({ new_record }) => {
        setPosts((prev) =>
          prev.map((p) => (p.id === new_record.id ? new_record : p)),
        );
      })
      .on("DELETE", ({ old_record }) => {
        setPosts((prev) => prev.filter((p) => p.id !== old_record.id));
      })
      .subscribe();

    return () => channel.unsubscribe();
  }, []);

  return posts;
}
```

## Storage

### Upload File

```typescript
const file = document.getElementById("fileInput").files[0];

const { data, error } = await client.storage
  .from("avatars")
  .upload("user-123.png", file);
```

### Download File

```typescript
const { data } = await client.storage.from("avatars").download("user-123.png");

// Create download link
const url = URL.createObjectURL(data);
const a = document.createElement("a");
a.href = url;
a.download = "avatar.png";
a.click();
```

### List Files

```typescript
const { data: files } = await client.storage.from("avatars").list();

files.forEach((file) => {
  console.log(file.name, file.size);
});
```

### Delete File

```typescript
await client.storage.from("avatars").remove(["user-123.png"]);
```

### Get Public URL

```typescript
const url = client.storage.from("public-bucket").getPublicUrl("logo.png");
```

### Create Signed URL (Private Files)

```typescript
const { data } = await client.storage
  .from("private-docs")
  .createSignedUrl("document.pdf", 3600); // 1 hour expiry

console.log("Temporary URL:", data.signedUrl);
```

## Row-Level Security

### Enable RLS on Table

```sql
ALTER TABLE posts ENABLE ROW LEVEL SECURITY;
```

### User Can Only See Own Posts

```sql
CREATE POLICY "Users see own posts"
ON posts FOR SELECT
USING (current_setting('app.user_id', true)::uuid = user_id);
```

### User Can Only Insert Own Posts

```sql
CREATE POLICY "Users insert own posts"
ON posts FOR INSERT
WITH CHECK (current_setting('app.user_id', true)::uuid = user_id);
```

### Public Read, Auth Write

```sql
-- Anyone can read
CREATE POLICY "Public read"
ON posts FOR SELECT
USING (true);

-- Only authenticated can write
CREATE POLICY "Auth write"
ON posts FOR INSERT
WITH CHECK (current_setting('app.role', true) = 'authenticated');
```

### Admin Access

```sql
CREATE POLICY "Admin full access"
ON posts FOR ALL
USING (current_setting('app.role', true) = 'admin');
```

## Edge Functions

### Create Function

```typescript
await client.functions.create({
  name: "send-email",
  code: `
    async function handler(req) {
      const { to, subject, body } = JSON.parse(req.body || '{}')

      // Send email logic here

      return {
        status: 200,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ success: true })
      }
    }
  `,
  enabled: true,
});
```

### Invoke Function

```typescript
const result = await client.functions.invoke("send-email", {
  to: "user@example.com",
  subject: "Hello",
  body: "Welcome to Fluxbase!",
});
```

### Function with Database Access

```typescript
await client.functions.create({
  name: "get-stats",
  code: `
    async function handler(req) {
      // Access database via client
      const dbUrl = Deno.env.get('DATABASE_URL')

      // Your logic here

      return {
        status: 200,
        body: JSON.stringify({ stats: 'data' })
      }
    }
  `,
});
```

## Common Patterns

### Todo App

```typescript
// Create todos table first
/*
CREATE TABLE todos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES auth.users(id),
  title TEXT NOT NULL,
  completed BOOLEAN DEFAULT false,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

ALTER TABLE todos ENABLE ROW LEVEL SECURITY;

CREATE POLICY "Users see own todos"
ON todos FOR ALL
USING (current_setting('app.user_id', true)::uuid = user_id);
*/

// Get todos
const { data: todos } = await client
  .from("todos")
  .select("*")
  .order("created_at", { ascending: false });

// Add todo
await client.from("todos").insert({
  title: "Buy groceries",
  user_id: currentUser.id,
});

// Toggle completed
await client.from("todos").update({ completed: true }).eq("id", todoId);

// Delete todo
await client.from("todos").delete().eq("id", todoId);
```

### Blog Posts with Comments

```typescript
// Create tables
/*
CREATE TABLE posts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  author_id UUID REFERENCES auth.users(id),
  title TEXT NOT NULL,
  content TEXT,
  published BOOLEAN DEFAULT false,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE comments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  post_id UUID REFERENCES posts(id) ON DELETE CASCADE,
  author_id UUID REFERENCES auth.users(id),
  content TEXT NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW()
);
*/

// Get posts with comments
const { data: posts } = await client
  .from("posts")
  .select(
    `
    id,
    title,
    content,
    author(name, email),
    comments(id, content, author(name))
  `,
  )
  .eq("published", true)
  .order("created_at", { ascending: false });
```

### User Profiles

```typescript
// Create profiles table
/*
CREATE TABLE profiles (
  id UUID PRIMARY KEY REFERENCES auth.users(id),
  username TEXT UNIQUE,
  avatar_url TEXT,
  bio TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Auto-create profile on user signup
CREATE OR REPLACE FUNCTION handle_new_user()
RETURNS TRIGGER AS $$
BEGIN
  INSERT INTO public.profiles (id, username)
  VALUES (NEW.id, NEW.email);
  RETURN NEW;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

CREATE TRIGGER on_auth_user_created
  AFTER INSERT ON auth.users
  FOR EACH ROW EXECUTE FUNCTION handle_new_user();
*/

// Get user profile
const { data: profile } = await client
  .from("profiles")
  .select("*")
  .eq("id", userId)
  .single();

// Update profile
await client
  .from("profiles")
  .update({ bio: "My new bio", avatar_url: "https://..." })
  .eq("id", userId);
```

## Error Handling

```typescript
try {
  const { data, error } = await client.from("posts").select("*");

  if (error) {
    console.error("Query error:", error.message);
    return;
  }

  console.log("Posts:", data);
} catch (err) {
  console.error("Network error:", err);
}
```

## Related Documentation

- [Authentication Guide](/docs/guides/authentication)
- [Database Queries](/docs/guides/typescript-sdk/database)
- [Realtime](/docs/guides/realtime)
- [Storage](/docs/guides/storage)
- [Row-Level Security](/docs/guides/row-level-security)
- [Edge Functions](/docs/guides/edge-functions)
