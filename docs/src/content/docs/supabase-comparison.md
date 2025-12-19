---
title: "Fluxbase vs Supabase"
---

Fluxbase provides API-compatible alternatives to Supabase's core features in a single ~110MB container (~70MB binary). If you're evaluating Fluxbase as a Supabase alternative, this guide highlights key differences.

## Quick Comparison

| Feature                | Supabase                     | Fluxbase                      |
| ---------------------- | ---------------------------- | ----------------------------- |
| **Deployment**         | ~13 containers (~2.5GB)      | 1 binary or container (~110MB) |
| **REST API**           | PostgREST                    | ✅ Built-in                   |
| **Authentication**     | GoTrue (JWT)                 | ✅ Built-in                   |
| **Realtime**           | WebSocket                    | WebSocket                     |
| **Storage**            | S3 or local                  | S3 or local                   |
| **AI Chatbots**        | ❌ No                        | ✅ Built-in                   |
| **Edge Functions**     | Deno runtime                 | Deno runtime                  |
| **Database**           | PostgreSQL 15+               | PostgreSQL 15+                |
| **Row-Level Security** | ✅ Yes                       | ✅ Yes                        |
| **Client SDK**         | TypeScript/JS                | TypeScript/JS                 |
| **Horizontal Scaling** | ✅ Yes (read replicas)       | ✅ Yes (distributed backends) |
| **Hosted Service**     | ✅ Yes (free tier available) | ❌ No(t yet?)                 |
| **Pricing**            | Free/$25+/month              | Open source (ELv2)            |

## SDK Compatibility

The Fluxbase SDK is API-compatible with Supabase. Only the import statement differs:

```typescript
// Supabase
import { createClient } from "@supabase/supabase-js";
const client = createClient("https://project.supabase.co", "anon-key");

// Fluxbase
import { createClient } from "@fluxbase/sdk";
const client = createClient("http://localhost:8080", "api-key");

// Everything else is identical
const { data, error } = await client
  .from("users")
  .select("id, name")
  .eq("status", "active");
```

All query methods (`.select()`, `.insert()`, `.update()`, `.delete()`, `.eq()`, `.order()`, etc.) work identically.

## Authentication

Same JWT-based flow with identical method signatures:

```typescript
// Sign up
const { data, error } = await client.auth.signUp({
  email: "user@example.com",
  password: "password123",
});

// Sign in
await client.auth.signInWithPassword({ email, password });

// Get session
const {
  data: { session },
} = await client.auth.getSession();
```

Both support email/password, magic links, OAuth providers, and session management.

**RLS syntax difference:**

```sql
-- Supabase
USING (auth.uid() = user_id)

-- Fluxbase
USING (current_setting('app.user_id', true)::uuid = user_id)
```

## Realtime

Similar patterns with different syntax:

```typescript
// Supabase
client
  .channel("changes")
  .on(
    "postgres_changes",
    { event: "*", schema: "public", table: "posts" },
    (payload) => console.log(payload)
  )
  .subscribe();

// Fluxbase
client.realtime
  .channel("table:public.posts")
  .on("*", (payload) => console.log(payload))
  .subscribe();
```

## Storage

Identical API for file operations:

```typescript
// Upload
await client.storage.from("avatars").upload("user1.png", file);

// Download
await client.storage.from("avatars").download("user1.png");

// List
await client.storage.from("avatars").list();
```

## Edge Functions

Both use Deno runtime with different deployment approaches:

**Supabase:** CLI-based deployment with `serve()` function

**Fluxbase:** File-based (GitOps), API-based, or dashboard deployment with `handler()` function

Function code requires minor adaptation when switching platforms.

## When to Choose Fluxbase

- You want simple deployment (single binary or container)
- You prefer self-hosting with full control
- You need predictable costs (no usage fees)
- You want to customize backend code
- You want horizontal scaling with just PostgreSQL (no Redis required)
- You need AI chatbots with database access built-in

## When to Choose Supabase

- You want a hosted service with free tier
- You prefer fully managed infrastructure
- You want professional support
- You don't want to manage any infrastructure
- You need built-in read replicas and global edge functions

## Deployment

**Fluxbase:** Single binary or container

**Supabase:** Multiple services via docker-compose

## Migration Considerations

Switching between platforms requires:

- **Database:** Standard PostgreSQL migration (pg_dump/restore)
- **RLS policies:** Update syntax (`auth.uid()` ↔ `current_setting()`)
- **SDK code:** Change import statement only
- **Edge functions:** Adapt function signature
- **Testing:** Verify behavior in new environment

Not a one-click migration, but API compatibility minimizes code changes.

## Scaling

**Fluxbase:** Supports both vertical and horizontal scaling with distributed state backends

**Supabase:** Horizontal scaling with read replicas for high traffic

**Fluxbase horizontal scaling features:**

- **Distributed rate limiting** - Shared counters across all instances (via PostgreSQL or Redis/Dragonfly)
- **Cross-instance broadcasts** - Pub/sub for realtime application events
- **Scheduler leader election** - Prevents duplicate cron job execution
- **Stateless authentication** - Nonces stored in PostgreSQL for multi-instance auth flows

**Requirements:**

- External PostgreSQL database (stores data, sessions, and distributed state)
- S3-compatible storage (MinIO, AWS S3, etc.) instead of local filesystem
- Load balancer with session stickiness for WebSocket connections

**Configuration:**

```bash
# Enable distributed state (PostgreSQL backend, no extra dependencies)
FLUXBASE_SCALING_BACKEND=postgres
FLUXBASE_SCALING_ENABLE_SCHEDULER_LEADER_ELECTION=true

# Or use Dragonfly for high-scale (1000+ req/s)
FLUXBASE_SCALING_BACKEND=redis
FLUXBASE_SCALING_REDIS_URL=redis://dragonfly:6379
```

See [Deployment: Scaling](/docs/deployment/scaling#horizontal-scaling) for full configuration details.

## Resources

**Fluxbase:**

- [Documentation](/)
- [GitHub](https://github.com/fluxbase-eu/fluxbase)
- [API Reference](/docs/api/sdk/classes/FluxbaseClient)

**Supabase:**

- [Documentation](https://supabase.com/docs)
- [GitHub](https://github.com/supabase/supabase)
- [Pricing](https://supabase.com/pricing)
