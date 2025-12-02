# Fluxbase API Cookbook

**Complete code examples for common use cases**

This cookbook provides copy-paste ready examples for all major Fluxbase features. Each example is production-ready and follows best practices.

## 📚 Table of Contents

1. [Getting Started](#getting-started)
2. [Authentication](#authentication)
3. [Database Operations](#database-operations)
4. [Query Operators](#query-operators)
5. [Aggregations](#aggregations)
6. [Row-Level Security (RLS)](#row-level-security-rls)
7. [Edge Functions](#edge-functions)
8. [Realtime Subscriptions](#realtime-subscriptions)
9. [Storage](#storage)
10. [Advanced Patterns](#advanced-patterns)

---

## Getting Started

### Installation

```bash
# Install the TypeScript SDK
npm install @fluxbase/client

# Or use the REST API directly (no SDK needed)
curl https://your-project.fluxbase.io/api/v1/tables/users
```

### Initialize Client

```typescript
import { createClient } from "@fluxbase/client";

const fluxbase = createClient(
  "https://your-project.fluxbase.io",
  "your-anon-key", // For anonymous access
);

// For service role access, use the service role key instead
// const fluxbase = createClient(
//   'https://your-project.fluxbase.io',
//   'your-service-role-key'
// )

// With authentication
const { data, error } = await fluxbase.auth.signIn({
  email: "user@example.com",
  password: "password123",
});

// Client now has authenticated session
```

---

## Authentication

### 1. Sign Up New User

```typescript
const { data, error } = await fluxbase.auth.signUp({
  email: "newuser@example.com",
  password: "SecurePass123!",
  metadata: {
    firstName: "John",
    lastName: "Doe",
  },
});

if (error) {
  console.error("Sign up failed:", error.message);
} else {
  console.log("User created:", data.user.id);
  console.log("Access token:", data.session.access_token);
  // Email verification sent automatically
}
```

### 2. Sign In

```typescript
const { data, error } = await fluxbase.auth.signIn({
  email: "user@example.com",
  password: "SecurePass123!",
});

if (error) {
  console.error("Sign in failed:", error.message);
} else {
  // Session automatically stored in client
  console.log("Signed in:", data.user.email);
}
```

### 3. Sign Out

```typescript
const { error } = await fluxbase.auth.signOut();

if (!error) {
  console.log("Signed out successfully");
}
```

### 4. Get Current User

```typescript
const { data, error } = await fluxbase.auth.getUser();

if (data) {
  console.log("Current user:", data.email);
  console.log("User ID:", data.id);
  console.log("Metadata:", data.user_metadata);
} else {
  console.log("No user signed in");
}
```

### 5. Update User Metadata

```typescript
const { data, error } = await fluxbase.auth.updateUser({
  metadata: {
    displayName: "John D.",
    avatar: "https://example.com/avatar.jpg",
    preferences: {
      theme: "dark",
      language: "en",
    },
  },
});
```

### 6. Refresh Session

```typescript
const { data, error } = await fluxbase.auth.refreshSession();

if (data) {
  console.log("New access token:", data.access_token);
}
```

### 7. Password Reset

```typescript
// Request reset email
const { error } = await fluxbase.auth.resetPasswordForEmail("user@example.com");

// In password reset form (after clicking email link)
const { error } = await fluxbase.auth.updatePassword({
  newPassword: "NewSecurePass123!",
});
```

### 8. Magic Link Authentication

```typescript
const { error } = await fluxbase.auth.signInWithOtp({
  email: "user@example.com",
});

// User clicks link in email, automatically signed in
```

---

## Database Operations

### 1. Insert Single Row

```typescript
const { data, error } = await fluxbase
  .from("posts")
  .insert({
    title: "My First Post",
    content: "Hello, World!",
    published: false,
  })
  .select(); // Return inserted data

console.log("Created post:", data[0]);
```

### 2. Insert Multiple Rows

```typescript
const { data, error } = await fluxbase
  .from("posts")
  .insert([
    { title: "Post 1", content: "Content 1" },
    { title: "Post 2", content: "Content 2" },
    { title: "Post 3", content: "Content 3" },
  ])
  .select();

console.log(`Created ${data.length} posts`);
```

### 3. Select All

```typescript
const { data, error } = await fluxbase.from("posts").select("*");

console.log("All posts:", data);
```

### 4. Select Specific Columns

```typescript
const { data, error } = await fluxbase
  .from("posts")
  .select("id, title, created_at");

// Returns only specified columns
```

### 5. Select with Related Data (Joins)

```typescript
const { data, error } = await fluxbase.from("posts").select(`
    id,
    title,
    author:users!user_id (
      id,
      email,
      user_metadata
    ),
    comments (
      id,
      content,
      created_at
    )
  `);

console.log("Post with author and comments:", data[0]);
```

### 6. Update Row

```typescript
const { data, error } = await fluxbase
  .from("posts")
  .update({ published: true })
  .eq("id", "post-uuid")
  .select();

console.log("Updated post:", data[0]);
```

### 7. Update Multiple Rows

```typescript
const { data, error } = await fluxbase
  .from("posts")
  .update({ category: "archived" })
  .lt("created_at", "2024-01-01")
  .select();

console.log(`Archived ${data.length} posts`);
```

### 8. Upsert (Insert or Update)

```typescript
const { data, error } = await fluxbase
  .from("user_settings")
  .upsert(
    {
      user_id: "user-uuid",
      theme: "dark",
      language: "en",
    },
    {
      onConflict: "user_id", // Unique constraint column
    },
  )
  .select();
```

### 9. Delete Row

```typescript
const { error } = await fluxbase.from("posts").delete().eq("id", "post-uuid");
```

### 10. Delete Multiple Rows

```typescript
const { data, error } = await fluxbase
  .from("posts")
  .delete()
  .eq("published", false)
  .lt("created_at", "2023-01-01")
  .select();

console.log(`Deleted ${data.length} unpublished posts`);
```

---

## Query Operators

### 1. Equal (eq)

```typescript
const { data } = await fluxbase
  .from("posts")
  .select("*")
  .eq("author_id", "user-uuid");
```

### 2. Not Equal (neq)

```typescript
const { data } = await fluxbase
  .from("posts")
  .select("*")
  .neq("status", "draft");
```

### 3. Greater Than (gt, gte)

```typescript
const { data } = await fluxbase
  .from("posts")
  .select("*")
  .gte("view_count", 1000); // >= 1000 views
```

### 4. Less Than (lt, lte)

```typescript
const { data } = await fluxbase
  .from("posts")
  .select("*")
  .lt("created_at", "2024-01-01"); // Before 2024
```

### 5. Pattern Matching (like, ilike)

```typescript
// Case-sensitive
const { data } = await fluxbase
  .from("posts")
  .select("*")
  .like("title", "%Tutorial%");

// Case-insensitive
const { data } = await fluxbase
  .from("posts")
  .select("*")
  .ilike("title", "%tutorial%");
```

### 6. In List (in)

```typescript
const { data } = await fluxbase
  .from("posts")
  .select("*")
  .in("category", ["tech", "science", "programming"]);
```

### 7. IS NULL / IS NOT NULL

```typescript
// Find posts without tags
const { data } = await fluxbase.from("posts").select("*").is("tags", null);

// Find posts with tags
const { data } = await fluxbase
  .from("posts")
  .select("*")
  .not("tags", "is", null);
```

### 8. Full-Text Search

```typescript
const { data } = await fluxbase
  .from("posts")
  .select("*")
  .textSearch("title,content", "javascript tutorial");
```

### 9. Range Queries

```typescript
const { data } = await fluxbase
  .from("posts")
  .select("*")
  .gte("created_at", "2024-01-01")
  .lte("created_at", "2024-12-31");
```

### 10. Complex Filters (AND/OR)

```typescript
// AND (multiple filters are AND by default)
const { data } = await fluxbase
  .from("posts")
  .select("*")
  .eq("published", true)
  .gte("view_count", 100);

// OR (using .or())
const { data } = await fluxbase
  .from("posts")
  .select("*")
  .or("published.eq.true,featured.eq.true");
```

---

## Aggregations

### 1. Count Rows

```typescript
const { data, count } = await fluxbase
  .from("posts")
  .select("*", { count: "exact" })
  .eq("published", true);

console.log(`Total published posts: ${count}`);
```

### 2. Sum

```typescript
const { data } = await fluxbase.from("orders").select("total").sum("total");

console.log("Total revenue:", data[0].sum);
```

### 3. Average

```typescript
const { data } = await fluxbase.from("products").select("price").avg("price");

console.log("Average price:", data[0].avg);
```

### 4. Min / Max

```typescript
const { data } = await fluxbase
  .from("products")
  .select("price")
  .min("price")
  .max("price");

console.log("Price range:", data[0].min, "-", data[0].max);
```

### 5. Group By

```typescript
const { data } = await fluxbase
  .from("posts")
  .select("category, count:id")
  .groupBy("category");

// Returns: [{ category: 'tech', count: 45 }, ...]
```

### 6. Group By with Aggregations

```typescript
const { data } = await fluxbase
  .from("orders")
  .select("user_id, total:amount.sum(), count:id")
  .groupBy("user_id")
  .order("total", { ascending: false });

// Top customers by revenue
```

---

## Row-Level Security (RLS)

### 1. Enable RLS on Table

```sql
-- Run in PostgreSQL
ALTER TABLE posts ENABLE ROW LEVEL SECURITY;
```

### 2. User Can Only See Their Own Data

```sql
-- RLS Policy
CREATE POLICY "Users can view own posts"
  ON posts
  FOR SELECT
  USING (user_id::text = current_setting('app.user_id', true));
```

```typescript
// Client code (automatically filtered by RLS)
const { data } = await fluxbase.from("posts").select("*");

// Only returns posts where user_id matches authenticated user
```

### 3. User Can Only Insert Own Data

```sql
CREATE POLICY "Users can insert own posts"
  ON posts
  FOR INSERT
  WITH CHECK (user_id::text = current_setting('app.user_id', true));
```

```typescript
const { data, error } = await fluxbase.from("posts").insert({
  title: "My Post",
  content: "Content",
  user_id: fluxbase.auth.user().id, // Must match authenticated user
});
```

### 4. User Can Only Update Own Data

```sql
CREATE POLICY "Users can update own posts"
  ON posts
  FOR UPDATE
  USING (user_id::text = current_setting('app.user_id', true))
  WITH CHECK (user_id::text = current_setting('app.user_id', true));
```

### 5. Admin Role Access

```sql
CREATE POLICY "Admins can see all posts"
  ON posts
  FOR ALL
  USING (
    current_setting('app.role', true) = 'admin'
  );
```

### 6. Public Read, Authenticated Write

```sql
-- Anyone can read published posts
CREATE POLICY "Public can read published posts"
  ON posts
  FOR SELECT
  USING (published = true);

-- Only authenticated users can write
CREATE POLICY "Authenticated users can insert"
  ON posts
  FOR INSERT
  TO authenticated
  WITH CHECK (true);
```

---

## Edge Functions

### 1. Create Function

```typescript
const { data, error } = await fluxbase.functions.create({
  name: "hello-world",
  code: `
    function handler(request) {
      return {
        status: 200,
        body: JSON.stringify({ message: 'Hello, World!' })
      }
    }
  `,
  permissions: ["--allow-net"],
});
```

### 2. Invoke Function (HTTP)

```typescript
const { data, error } = await fluxbase.functions.invoke("hello-world", {
  body: { name: "John" },
});

console.log(data); // { message: 'Hello, World!' }
```

### 3. Function with Database Access

```typescript
const functionCode = `
async function handler(request) {
  // Access authenticated user ID
  const userId = request.user_id

  if (!userId) {
    return {
      status: 401,
      body: JSON.stringify({ error: 'Unauthorized' })
    }
  }

  // Query database (automatically filtered by RLS)
  const response = await fetch('http://localhost:8080/api/v1/tables/posts', {
    headers: {
      'Authorization': request.headers.authorization
    }
  })

  const posts = await response.json()

  return {
    status: 200,
    body: JSON.stringify({
      user_id: userId,
      post_count: posts.length
    })
  }
}
`;

await fluxbase.functions.create({
  name: "user-stats",
  code: functionCode,
  permissions: ["--allow-net"],
});
```

### 4. Function with External API

```typescript
const functionCode = `
async function handler(request) {
  const { city } = JSON.parse(request.body)

  // Call external API
  const response = await fetch(\`https://api.weather.com/v1/current?city=\${city}\`)
  const weather = await response.json()

  return {
    status: 200,
    body: JSON.stringify(weather)
  }
}
`;

await fluxbase.functions.create({
  name: "get-weather",
  code: functionCode,
  permissions: ["--allow-net=api.weather.com"],
});
```

### 5. Scheduled Function (Cron)

```typescript
await fluxbase.functions.create({
  name: "daily-cleanup",
  code: `
    async function handler(request) {
      // Delete old records
      const response = await fetch('http://localhost:8080/api/v1/tables/logs', {
        method: 'DELETE',
        headers: {
          'Content-Type': 'application/json',
          'apikey': request.env.SERVICE_ROLE_KEY
        },
        body: JSON.stringify({
          filter: { created_at: { lt: '2024-01-01' } }
        })
      })

      return {
        status: 200,
        body: JSON.stringify({ message: 'Cleanup complete' })
      }
    }
  `,
  schedule: "0 2 * * *", // Daily at 2 AM
  permissions: ["--allow-net"],
});
```

---

## Realtime Subscriptions

### 1. Subscribe to Table Changes

```typescript
const subscription = fluxbase
  .from("posts")
  .on("*", (payload) => {
    console.log("Change detected:", payload);
  })
  .subscribe();

// payload = {
//   type: 'INSERT' | 'UPDATE' | 'DELETE',
//   table: 'posts',
//   record: { ... },
//   old_record: { ... }  // For UPDATE/DELETE
// }
```

### 2. Subscribe to INSERT Only

```typescript
const subscription = fluxbase
  .from("posts")
  .on("INSERT", (payload) => {
    console.log("New post:", payload.record);
  })
  .subscribe();
```

### 3. Subscribe to UPDATE Only

```typescript
const subscription = fluxbase
  .from("posts")
  .on("UPDATE", (payload) => {
    console.log("Post updated:", payload.record);
    console.log("Old values:", payload.old_record);
  })
  .subscribe();
```

### 4. Subscribe to DELETE Only

```typescript
const subscription = fluxbase
  .from("posts")
  .on("DELETE", (payload) => {
    console.log("Post deleted:", payload.old_record.id);
  })
  .subscribe();
```

### 5. Subscribe with Filters

```typescript
const subscription = fluxbase
  .from("posts")
  .on("INSERT", (payload) => {
    console.log("New published post:", payload.record);
  })
  .filter("published", "eq", true)
  .subscribe();
```

### 6. Multiple Subscriptions

```typescript
const channel = fluxbase.channel("my-channel");

channel
  .on("posts", "INSERT", (payload) => {
    console.log("New post:", payload);
  })
  .on("comments", "INSERT", (payload) => {
    console.log("New comment:", payload);
  })
  .subscribe();
```

### 7. Broadcast Messages

```typescript
const channel = fluxbase.channel("room-1");

// Send message
channel.send({
  type: "broadcast",
  event: "message",
  payload: { text: "Hello!", user: "John" },
});

// Receive messages
channel.on("broadcast", { event: "message" }, (payload) => {
  console.log("New message:", payload.text);
});

channel.subscribe();
```

### 8. Presence Tracking

```typescript
const channel = fluxbase.channel("room-1");

// Track user presence
channel.track({
  user_id: "user-123",
  username: "John Doe",
  status: "online",
});

// Listen to presence changes
channel.on("presence", { event: "join" }, (user) => {
  console.log("User joined:", user);
});

channel.on("presence", { event: "leave" }, (user) => {
  console.log("User left:", user);
});

channel.subscribe();
```

### 9. Unsubscribe

```typescript
// Unsubscribe specific subscription
subscription.unsubscribe();

// Remove all subscriptions
fluxbase.removeAllChannels();
```

---

## Storage

### 1. Create Bucket

```typescript
const { data, error } = await fluxbase.storage.createBucket({
  name: "avatars",
  public: true, // Public read access
});
```

### 2. Upload File

```typescript
const file = document.querySelector('input[type="file"]').files[0];

const { data, error } = await fluxbase.storage
  .from("avatars")
  .upload("user-123/avatar.jpg", file);

console.log("File uploaded:", data.path);
```

### 3. Upload with Custom Path

```typescript
const { data, error } = await fluxbase.storage
  .from("documents")
  .upload(`${userId}/reports/2024-Q1.pdf`, file);
```

### 4. Upload with Metadata

```typescript
const { data, error } = await fluxbase.storage
  .from("avatars")
  .upload("avatar.jpg", file, {
    metadata: {
      uploadedBy: userId,
      description: "User profile picture",
    },
    contentType: "image/jpeg",
  });
```

### 5. Download File

```typescript
const { data, error } = await fluxbase.storage
  .from("avatars")
  .download("user-123/avatar.jpg");

// data is a Blob
const url = URL.createObjectURL(data);
```

### 6. Get Public URL

```typescript
const { publicURL } = fluxbase.storage
  .from("avatars")
  .getPublicUrl("user-123/avatar.jpg");

// Use in <img src={publicURL} />
```

### 7. Create Signed URL (Private Files)

```typescript
const { data, error } = await fluxbase.storage
  .from("documents")
  .createSignedUrl("private/contract.pdf", 3600); // Expires in 1 hour

console.log("Temporary URL:", data.signedURL);
```

### 8. List Files

```typescript
const { data, error } = await fluxbase.storage
  .from("avatars")
  .list("user-123/", {
    limit: 100,
    offset: 0,
    sortBy: { column: "created_at", order: "desc" },
  });

console.log("Files:", data);
```

### 9. Delete File

```typescript
const { error } = await fluxbase.storage
  .from("avatars")
  .remove(["user-123/avatar.jpg"]);
```

### 10. Move File

```typescript
const { data, error } = await fluxbase.storage
  .from("avatars")
  .move("old-path/avatar.jpg", "new-path/avatar.jpg");
```

### 11. Copy File

```typescript
const { data, error } = await fluxbase.storage
  .from("avatars")
  .copy("original/avatar.jpg", "thumbnails/avatar-small.jpg");
```

---

## Advanced Patterns

### 1. Pagination

```typescript
const PAGE_SIZE = 20;

async function getPosts(page: number) {
  const { data, error, count } = await fluxbase
    .from("posts")
    .select("*", { count: "exact" })
    .order("created_at", { ascending: false })
    .range(page * PAGE_SIZE, (page + 1) * PAGE_SIZE - 1);

  return {
    posts: data,
    totalPages: Math.ceil(count / PAGE_SIZE),
    currentPage: page,
  };
}

// Usage
const page1 = await getPosts(0);
const page2 = await getPosts(1);
```

### 2. Cursor-Based Pagination

```typescript
async function getPostsAfter(cursor?: string) {
  let query = fluxbase
    .from("posts")
    .select("*")
    .order("created_at", { ascending: false })
    .limit(20);

  if (cursor) {
    query = query.gt("created_at", cursor);
  }

  const { data } = await query;

  return {
    posts: data,
    nextCursor: data.length > 0 ? data[data.length - 1].created_at : null,
  };
}

// Usage
const page1 = await getPostsAfter();
const page2 = await getPostsAfter(page1.nextCursor);
```

### 3. Optimistic Updates

```typescript
async function likePost(postId: string) {
  // Optimistically update UI
  const optimisticPost = { ...currentPost, likes: currentPost.likes + 1 };
  setPost(optimisticPost);

  try {
    // Actual update
    const { data, error } = await fluxbase
      .from("posts")
      .update({ likes: optimisticPost.likes })
      .eq("id", postId)
      .select();

    if (error) throw error;

    // Update with server data
    setPost(data[0]);
  } catch (error) {
    // Rollback on error
    setPost(currentPost);
    console.error("Failed to like post:", error);
  }
}
```

### 4. Batch Operations

```typescript
async function batchInsert(records: any[]) {
  const BATCH_SIZE = 100;
  const results = [];

  for (let i = 0; i < records.length; i += BATCH_SIZE) {
    const batch = records.slice(i, i + BATCH_SIZE);

    const { data, error } = await fluxbase.from("posts").insert(batch).select();

    if (error) {
      console.error(`Batch ${i / BATCH_SIZE} failed:`, error);
      continue;
    }

    results.push(...data);
  }

  return results;
}
```

### 5. Transactions (Multiple Operations)

```typescript
async function transferCredits(
  fromUserId: string,
  toUserId: string,
  amount: number,
) {
  // Start transaction using RPC function
  const { data, error } = await fluxbase.rpc("transfer_credits", {
    from_user: fromUserId,
    to_user: toUserId,
    amount: amount,
  });

  return { success: !error, error };
}

// PostgreSQL function
/*
CREATE OR REPLACE FUNCTION transfer_credits(
  from_user UUID,
  to_user UUID,
  amount INTEGER
) RETURNS BOOLEAN AS $$
BEGIN
  UPDATE users SET credits = credits - amount WHERE id = from_user;
  UPDATE users SET credits = credits + amount WHERE id = to_user;
  RETURN TRUE;
EXCEPTION WHEN OTHERS THEN
  RETURN FALSE;
END;
$$ LANGUAGE plpgsql;
*/
```

### 6. Caching with React Query

```typescript
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";

function usePosts() {
  return useQuery({
    queryKey: ["posts"],
    queryFn: async () => {
      const { data, error } = await fluxbase
        .from("posts")
        .select("*")
        .order("created_at", { ascending: false });

      if (error) throw error;
      return data;
    },
    staleTime: 60000, // Cache for 1 minute
  });
}

function useCreatePost() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (newPost) => {
      const { data, error } = await fluxbase
        .from("posts")
        .insert(newPost)
        .select();

      if (error) throw error;
      return data[0];
    },
    onSuccess: () => {
      // Invalidate and refetch
      queryClient.invalidateQueries({ queryKey: ["posts"] });
    },
  });
}
```

### 7. Error Handling Pattern

```typescript
async function safeQuery<T>(
  queryFn: () => Promise<{ data: T | null; error: any }>,
): Promise<T> {
  const { data, error } = await queryFn();

  if (error) {
    // Log error to monitoring service
    console.error("Query failed:", error);

    // Throw custom error
    throw new DatabaseError(error.message, error.code);
  }

  if (!data) {
    throw new NotFoundError("No data returned");
  }

  return data;
}

// Usage
try {
  const posts = await safeQuery(() => fluxbase.from("posts").select("*"));
} catch (error) {
  if (error instanceof NotFoundError) {
    // Handle not found
  } else if (error instanceof DatabaseError) {
    // Handle database error
  }
}
```

### 8. Infinite Scroll

```typescript
import { useInfiniteQuery } from "@tanstack/react-query";

function useInfinitePosts() {
  return useInfiniteQuery({
    queryKey: ["posts", "infinite"],
    queryFn: async ({ pageParam = 0 }) => {
      const { data, error } = await fluxbase
        .from("posts")
        .select("*")
        .order("created_at", { ascending: false })
        .range(pageParam * 20, (pageParam + 1) * 20 - 1);

      if (error) throw error;
      return {
        posts: data,
        nextPage: data.length === 20 ? pageParam + 1 : undefined,
      };
    },
    getNextPageParam: (lastPage) => lastPage.nextPage,
  });
}

// Usage in component
const { data, fetchNextPage, hasNextPage, isFetchingNextPage } =
  useInfinitePosts();
```

### 9. Debounced Search

```typescript
import { useState, useEffect } from "react";
import { useDebounce } from "use-debounce";

function useSearch(searchTerm: string) {
  const [debouncedTerm] = useDebounce(searchTerm, 500);

  return useQuery({
    queryKey: ["search", debouncedTerm],
    queryFn: async () => {
      if (!debouncedTerm) return [];

      const { data, error } = await fluxbase
        .from("posts")
        .select("*")
        .ilike("title", `%${debouncedTerm}%`)
        .limit(10);

      if (error) throw error;
      return data;
    },
    enabled: debouncedTerm.length > 0,
  });
}
```

### 10. Real-time Collaboration

```typescript
function useCollaborativeDocument(docId: string) {
  const [doc, setDoc] = useState(null);
  const [users, setUsers] = useState([]);

  useEffect(() => {
    // Load document
    fluxbase
      .from("documents")
      .select("*")
      .eq("id", docId)
      .single()
      .then(({ data }) => setDoc(data));

    // Subscribe to changes
    const docSubscription = fluxbase
      .from("documents")
      .on("UPDATE", (payload) => {
        if (payload.record.id === docId) {
          setDoc(payload.record);
        }
      })
      .eq("id", docId)
      .subscribe();

    // Track presence
    const channel = fluxbase.channel(`doc:${docId}`);

    channel.track({
      user_id: fluxbase.auth.user().id,
      username: fluxbase.auth.user().email,
      cursor: null,
    });

    channel.on("presence", { event: "sync" }, () => {
      const presenceState = channel.presenceState();
      setUsers(Object.values(presenceState));
    });

    channel.subscribe();

    return () => {
      docSubscription.unsubscribe();
      channel.unsubscribe();
    };
  }, [docId]);

  return { doc, users };
}
```

---

## 🎯 Best Practices

### 1. Always Handle Errors

```typescript
const { data, error } = await fluxbase.from("posts").select("*");

if (error) {
  console.error("Error:", error.message);
  // Show user-friendly message
  return;
}

// Use data safely
```

### 2. Use TypeScript Types

```typescript
interface Post {
  id: string;
  title: string;
  content: string;
  created_at: string;
}

const { data, error } = await fluxbase.from<Post>("posts").select("*");

// data is typed as Post[]
```

### 3. Leverage RLS for Security

```typescript
// Don't do this (client-side filtering)
const { data } = await fluxbase
  .from("posts")
  .select("*")
  .eq("user_id", currentUser.id);

// Do this (RLS automatically filters)
const { data } = await fluxbase.from("posts").select("*");
// RLS policy ensures user only sees their own posts
```

### 4. Use Indexes for Performance

```sql
-- Add indexes for frequently queried columns
CREATE INDEX idx_posts_user_id ON posts(user_id);
CREATE INDEX idx_posts_created_at ON posts(created_at DESC);
CREATE INDEX idx_posts_published ON posts(published) WHERE published = true;
```

### 5. Batch Operations When Possible

```typescript
// Inefficient (N queries)
for (const post of posts) {
  await fluxbase.from("posts").insert(post);
}

// Efficient (1 query)
await fluxbase.from("posts").insert(posts);
```

---

## 📚 Additional Resources

- [API Reference](https://docs.fluxbase.io/api)
- [SDK Documentation](https://docs.fluxbase.io/sdk)
- [Example Applications](https://github.com/fluxbase/examples)
- [Community Discord](https://discord.gg/fluxbase)

---

**Last Updated**: 2025-10-30
**Status**: Complete ✅
**Examples**: 60+ production-ready code snippets
