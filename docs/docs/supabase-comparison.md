# Fluxbase vs Supabase

Fluxbase provides API-compatible alternatives to Supabase's core features in a single ~80MB container (~40MB binary). If you're evaluating Fluxbase as a Supabase alternative, this guide highlights key differences.

## Quick Comparison

| Feature | Supabase | Fluxbase |
| --- | --- | --- |
| **Deployment** | ~10 containers (~2GB) | 1 binary or container (~80MB) |
| **REST API** | PostgREST | Compatible |
| **Authentication** | GoTrue (JWT) | Compatible |
| **Realtime** | WebSocket | Compatible |
| **Storage** | S3-compatible | Compatible |
| **Edge Functions** | Deno runtime | Deno runtime |
| **Database** | PostgreSQL 15+ | PostgreSQL 15+ |
| **Row-Level Security** | Yes (auth.uid()) | Yes (current_setting()) |
| **Client SDK** | TypeScript/JS | TypeScript/JS (compatible) |
| **Horizontal Scaling** | Yes (read replicas) | Yes (with configuration)* |
| **Hosted Service** | Yes (free tier available) | No |
| **Pricing** | Free/$25+/month | Open source (MIT) |

## SDK Compatibility

The Fluxbase SDK is API-compatible with Supabase. Only the import statement differs:

```typescript
// Supabase
import { createClient } from '@supabase/supabase-js'
const client = createClient('https://project.supabase.co', 'anon-key')

// Fluxbase
import { createClient } from '@fluxbase/sdk'
const client = createClient('http://localhost:8080', 'api-key')

// Everything else is identical
const { data, error } = await client
  .from('users')
  .select('id, name')
  .eq('status', 'active')
```

All query methods (`.select()`, `.insert()`, `.update()`, `.delete()`, `.eq()`, `.order()`, etc.) work identically.

## Authentication

Same JWT-based flow with identical method signatures:

```typescript
// Sign up
const { data, error } = await client.auth.signUp({
  email: 'user@example.com',
  password: 'password123'
})

// Sign in
await client.auth.signInWithPassword({ email, password })

// Get session
const { data: { session } } = await client.auth.getSession()
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
client.channel('changes')
  .on('postgres_changes',
    { event: '*', schema: 'public', table: 'posts' },
    (payload) => console.log(payload)
  )
  .subscribe()

// Fluxbase
client.realtime
  .channel('table:public.posts')
  .on('*', (payload) => console.log(payload))
  .subscribe()
```

## Storage

Identical API for file operations:

```typescript
// Upload
await client.storage.from('avatars').upload('user1.png', file)

// Download
await client.storage.from('avatars').download('user1.png')

// List
await client.storage.from('avatars').list()
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
- You can configure horizontal scaling infrastructure (load balancer, external DB, S3/MinIO)

## When to Choose Supabase

- You want a hosted service with free tier
- You prefer managed infrastructure with automatic scaling
- You want professional support
- You don't want to manage load balancers, databases, or object storage
- You need built-in read replicas without configuration

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

**Fluxbase:** Supports both vertical and horizontal scaling*

**Supabase:** Horizontal scaling with read replicas for high traffic

***Fluxbase horizontal scaling requirements:**
- External PostgreSQL database (not embedded) - stores data + sessions
- S3-compatible storage (MinIO, AWS S3, etc.) instead of local filesystem
- Load balancer with session stickiness for realtime WebSocket connections

**Note**: Sessions are stored in PostgreSQL (shared across instances). Rate limiting and CSRF are per-instance.

See [Deployment: Scaling](/docs/deployment/scaling#horizontal-scaling) for configuration details.

## Resources

**Fluxbase:**
- [Documentation](/)
- [GitHub](https://github.com/wayli-app/fluxbase)
- [API Reference](/docs/api/sdk/classes/FluxbaseClient)

**Supabase:**
- [Documentation](https://supabase.com/docs)
- [GitHub](https://github.com/supabase/supabase)
- [Pricing](https://supabase.com/pricing)
