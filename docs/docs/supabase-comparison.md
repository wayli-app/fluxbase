# Fluxbase vs Supabase Comparison

**Evaluating Fluxbase as a Supabase alternative**

If you're familiar with Supabase or considering alternatives, this guide compares Fluxbase and Supabase to help you make an informed decision.

## 📊 Feature Comparison

| Feature                 | Supabase                 | Fluxbase                         | Notes                                                   |
| ----------------------- | ------------------------ | -------------------------------- | ------------------------------------------------------- |
| **Deployment**          | 🐳 ~10 containers (~2GB) | 📦 1 binary or container (~80MB) | Fluxbase is 40x smaller than Supabase                   |
| **Containers Required** | 10+ services             | 1                                | Supabase uses microservices, Fluxbase is a monolith     |
| **REST API**            | ✅ PostgREST             | ✅ Compatible                    | Fluxbase is fully compatible with Supabase query API    |
| **Authentication**      | ✅ GoTrue                | ✅ Compatible                    | Both use JWT-based auth with identical SDK methods      |
| **Realtime**            | ✅ WebSocket             | ✅ Compatible                    | Both use PostgreSQL LISTEN/NOTIFY, identical SDK API    |
| **Storage**             | ✅ S3-compatible         | ✅ Compatible                    | Fluxbase has identical file operations API              |
| **Edge Functions**      | ✅ Deno runtime          | ✅ Deno runtime                  | Both use Deno, but different deployment methods         |
| **Database**            | ✅ PostgreSQL 15+        | ✅ PostgreSQL 15+                | Both use same PostgreSQL database engine                |
| **Row-Level Security**  | ✅ Yes                   | ✅ Yes                           | Both support PostgreSQL RLS (different syntax)          |
| **Client SDK**          | ✅ TypeScript/JS         | ✅ TypeScript/JS                 | Fluxbase SDK is API-compatible with Supabase            |
| **Horizontal Scaling**  | ✅ Yes                   | ❌ No                            | Supabase supports read replicas, Fluxbase doesn't       |
| **Self-Hosted**         | ✅ Yes                   | ✅ Yes                           | Both support self-hosting (different complexity)        |
| **Hosted Service**      | ✅ Yes + Free Tier       | ❌ No                            | Only Supabase offers managed hosting with free tier     |
| **Pricing**             | 💰 Free/$25+/month       | 🆓 Open Source                   | Supabase has paid tiers, Fluxbase is free (MIT license) |

## 🎯 Why Choose Fluxbase?

### Single Binary Simplicity

- Deploy as one ~80MB binary
- No microservices orchestration
- Easier to understand and debug
- Lightweight resource footprint

### Full Control

- Deploy anywhere (AWS, GCP, Azure, on-premise)
- No vendor dependencies
- Customize source code as needed
- Own your infrastructure completely

### Cost Predictability

- No usage-based pricing
- No egress fees
- Predictable hosting costs
- Open source (MIT license)

### API Compatibility

- Same REST API patterns
- Similar client SDK interface
- PostgreSQL-native features
- Familiar authentication flows

## 🎯 Why Choose Supabase?

### Hosted Service

- **Free tier** available (500MB database, 1GB bandwidth)
- Managed infrastructure
- Automatic updates and scaling
- Built-in monitoring and analytics

### Horizontal Scalability

- **Read replicas** for scaling reads
- Load balancing across instances
- Better for high-traffic applications
- Database connection pooling

### Mature Ecosystem

- Larger community
- More third-party integrations
- Extensive documentation
- Professional support available

### Self-Hosted Option

- Supabase also offers self-hosting via Docker
- More complex setup (multiple services)
- Community support for self-hosted version

## 💰 Cost Comparison

### Supabase Pricing

- **Free**: 500MB database, 1GB bandwidth, 2GB file storage
- **Pro**: $25/month + usage fees (additional database size, bandwidth, storage)
- **Team**: $599/month + usage
- **Self-hosted**: Free (infrastructure costs only)

### Fluxbase Pricing

- **Open Source**: Free (MIT license)
- **Infrastructure**: Your hosting costs (typically $5-50/month for VPS)
- **No usage fees**: No per-request or bandwidth charges
- **No free hosted tier**: Must self-host

## 🔄 API Compatibility

Fluxbase is designed to be compatible with Supabase's API patterns, making evaluation easier.

### Database Queries

✅ **Identical API** - Only the import statement differs

```typescript
// Supabase
import { createClient } from "@supabase/supabase-js";
const client = createClient(
  "https://your-project.supabase.co",
  "your-anon-key"
);

// Fluxbase
import { createClient } from "@fluxbase/sdk";
const client = createClient("http://localhost:8080", "your-api-key");

// Everything else is identical
const { data, error } = await client
  .from("users")
  .select("id, name, email")
  .eq("status", "active")
  .order("created_at", { ascending: false });
```

**Query methods**: `.select()`, `.insert()`, `.update()`, `.delete()`, `.eq()`, `.neq()`, `.gt()`, `.gte()`, `.lt()`, `.lte()`, `.like()`, `.ilike()`, `.is()`, `.in()`, `.order()`, `.limit()`, `.range()` - all work identically.

**Recent improvements**: Fluxbase now supports awaiting queries directly without calling `.execute()`:

```typescript
// Both of these work identically in Fluxbase (just like Supabase)
const { data } = await client.from('users').select('*')
const { data } = await client.from('users').select('*').execute()
```

### Authentication

✅ **Identical API** - Same method signatures and responses

```typescript
// Sign up - identical in both
const { data, error } = await client.auth.signUp({
  email: "user@example.com",
  password: "password123",
});

// Sign in - identical in both
const { data, error } = await client.auth.signInWithPassword({
  email: "user@example.com",
  password: "password123",
});

// Sign out - identical in both
await client.auth.signOut();

// Get session - identical in both
const {
  data: { session },
} = await client.auth.getSession();
```

**Auth methods**: `.signUp()`, `.signInWithPassword()`, `.signInWithOAuth()`, `.signOut()`, `.getSession()`, `.getUser()`, `.resetPasswordForEmail()` - all work identically.

**Recent improvements**: Fluxbase now supports auth state change listeners:

```typescript
// Listen to auth state changes - identical in both
const { data: { subscription } } = client.auth.onAuthStateChange((event, session) => {
  console.log('Auth event:', event, session)
  // Events: SIGNED_IN, SIGNED_OUT, TOKEN_REFRESHED, USER_UPDATED, etc.
})

// Unsubscribe when done
subscription.unsubscribe()
```

**Admin API improvements**: Fluxbase now includes `admin.getUserById()` for easier user management:

```typescript
// Fluxbase - new method
const user = await client.admin.getUserById('user-id-123')

// Previously required filtering (still works)
const { users } = await client.admin.listUsers()
const user = users.find(u => u.id === 'user-id-123')
```

### Realtime Subscriptions

✅ **Compatible API** - Similar channel and subscription patterns with minor syntax differences

```typescript
// Supabase
import { createClient } from "@supabase/supabase-js";
const client = createClient(
  "https://your-project.supabase.co",
  "your-anon-key"
);

const subscription = client
  .channel("table-changes")
  .on(
    "postgres_changes",
    { event: "*", schema: "public", table: "posts" },
    (payload) => console.log(payload)
  )
  .subscribe();

// Fluxbase - Similar pattern, different event syntax
import { createClient } from "@fluxbase/sdk";
const client = createClient("http://localhost:8080", "your-api-key");

const channel = client.realtime
  .channel("table:public.posts")
  .on("INSERT", (payload) => console.log("New:", payload.new_record))
  .on("UPDATE", (payload) => console.log("Updated:", payload.new_record))
  .on("DELETE", (payload) => console.log("Deleted:", payload.old_record))
  .subscribe();

// Or use wildcard for all events
const channel = client.realtime
  .channel("table:public.posts")
  .on("*", (payload) => console.log(payload))
  .subscribe();

// Unsubscribe - similar in both
channel.unsubscribe(); // Fluxbase
await client.removeChannel(subscription); // Supabase
```

**Key differences:**

- Supabase uses `postgres_changes` event with filter object, Fluxbase uses direct event names (`INSERT`, `UPDATE`, `DELETE`, `*`)
- Channel naming: Supabase uses custom names, Fluxbase uses `table:schema.table` format
- Payload structure is similar (both provide `new_record`, `old_record`)

### File Storage

✅ **Identical API** - Same upload/download methods

```typescript
// Upload - identical in both
const { data, error } = await client.storage
  .from("avatars")
  .upload("user1.png", file);

// Download - identical in both
const { data, error } = await client.storage
  .from("avatars")
  .download("user1.png");

// List files - identical in both
const { data, error } = await client.storage.from("avatars").list();

// Delete - identical in both
const { data, error } = await client.storage
  .from("avatars")
  .remove(["user1.png"]);
```

**Storage methods**: `.upload()`, `.download()`, `.list()`, `.remove()`, `.createSignedUrl()`, `.getPublicUrl()` - all work identically.

### Edge Functions

⚠️ **Same Runtime, Similar Deployment** - Both use Deno and support file-based deployment

**Supabase** - CLI-based file deployment:

```typescript
// functions/hello/index.ts
import { serve } from "https://deno.land/std@0.168.0/http/server.ts";

serve(async (req) => {
  const { name } = await req.json();
  return new Response(JSON.stringify({ message: `Hello ${name}!` }), {
    headers: { "Content-Type": "application/json" },
  });
});
```

```bash
# Deploy via CLI
supabase functions deploy hello
```

**Fluxbase** - Multiple deployment options:

#### 1. File-Based Deployment (Recommended for Production)

GitOps-friendly, ideal for Docker/Kubernetes:

```bash
# Create function file
mkdir -p ./functions
cat > ./functions/hello.ts << 'EOF'
async function handler(req) {
  const { name } = JSON.parse(req.body || '{}');
  return {
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ message: `Hello ${name}!` })
  };
}
EOF
```

**Docker deployment:**

```yaml
# docker-compose.yml
services:
  fluxbase:
    image: ghcr.io/wayli-app/fluxbase:latest:latest
    volumes:
      - ./functions:/app/functions
    environment:
      FLUXBASE_FUNCTIONS_ENABLED: "true"
      FLUXBASE_FUNCTIONS_DIR: /app/functions
```

**Kubernetes deployment:**

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: fluxbase-functions
spec:
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 1Gi
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: fluxbase
spec:
  template:
    spec:
      containers:
        - name: fluxbase
          image: ghcr.io/wayli-app/fluxbase:latest:latest
          env:
            - name: FLUXBASE_FUNCTIONS_ENABLED
              value: "true"
            - name: FLUXBASE_FUNCTIONS_DIR
              value: /app/functions
          volumeMounts:
            - name: functions
              mountPath: /app/functions
      volumes:
        - name: functions
          persistentVolumeClaim:
            claimName: fluxbase-functions
```

**Reload functions (admin-only):**

```bash
curl -X POST http://localhost:8080/api/v1/admin/functions/reload \
  -H "Authorization: Bearer ADMIN_TOKEN"
```

#### 2. API-Based Deployment

Dynamic creation via REST API:

```bash
curl -X POST http://localhost:8080/api/v1/functions \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "name": "hello",
    "code": "async function handler(req) { ... }",
    "enabled": true
  }'
```

#### 3. Admin Dashboard

Visual editor with Monaco syntax highlighting:

- Navigate to Functions section
- Click "New Function"
- Write code in browser editor
- Save (stores in database and syncs to filesystem)

**Key Similarities & Differences**:

- ✅ **Runtime**: Both use Deno (secure, fast, TypeScript-first)
- ✅ **File-based deployment**: Both support storing functions as `.ts` files
- ✅ **Version control**: Both work with Git (functions are files)
- ✅ **Volume mounting**: Both support Docker/Kubernetes volume mounting
- ⚠️ **Function signature**: Supabase uses `serve()`, Fluxbase uses `handler(req)`
- ⚠️ **Deployment tools**: Supabase uses CLI, Fluxbase uses API/Dashboard/volume mount
- ⚠️ **Hot reload**: Fluxbase requires manual reload API call, Supabase auto-deploys
- ❌ **Not directly portable**: Function code needs minor adaptation when switching platforms

**Migration tip**: Converting from Supabase to Fluxbase requires changing the function structure:

```typescript
// Supabase
serve(async (req) => {
  const data = await req.json();
  return new Response(JSON.stringify(result), {
    headers: { "Content-Type": "application/json" },
  });
});

// Fluxbase equivalent
async function handler(req) {
  const data = JSON.parse(req.body || "{}");
  return {
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(result),
  };
}
```

## 🔐 Authentication Differences

Both use JWT tokens and support:

- Email/password
- Magic links
- OAuth providers (Google, GitHub, etc.)
- Session management
- Row-Level Security (RLS)

**Key Difference**: RLS policy syntax

**Supabase** uses `auth.uid()`:

```sql
CREATE POLICY "Users can view own data"
ON users
FOR SELECT
USING (auth.uid() = id);
```

**Fluxbase** uses `current_setting()`:

```sql
CREATE POLICY "Users can view own data"
ON users
FOR SELECT
USING (current_setting('app.user_id', true)::uuid = id);
```

## ⚖️ Trade-offs

### Choose Fluxbase If:

- ✅ You want the simplest possible deployment (single binary)
- ✅ You prefer self-hosting with full control
- ✅ You want predictable, low costs (no usage fees)
- ✅ You need to customize the backend code
- ✅ Your traffic is moderate (doesn't require horizontal scaling)
- ✅ You're comfortable managing your own infrastructure

### Choose Supabase If:

- ✅ You want a **free hosted tier** to start
- ✅ You need **horizontal scalability** for high traffic
- ✅ You prefer managed infrastructure
- ✅ You want professional support options
- ✅ You need a mature ecosystem with many integrations
- ✅ You're building a high-scale application
- ✅ You don't want to manage infrastructure

## 🚀 Deployment

### Fluxbase Deployment

```bash
# Single binary deployment
./fluxbase --config fluxbase.yaml

# Or with Docker
docker run -d -p 8080:8080 \
  -v $(pwd)/fluxbase.yaml:/etc/fluxbase/fluxbase.yaml \
  fluxbase/fluxbase:latest
```

### Supabase Self-Hosted Deployment

Supabase requires multiple services (Kong, PostgREST, GoTrue, Realtime, Storage, etc.):

```bash
# Clone Supabase
git clone https://github.com/supabase/supabase
cd supabase/docker

# Start all services
docker-compose up -d
```

More complex setup but more features and scalability.

## 📚 Additional Resources

### Fluxbase

- [Documentation](/)
- [GitHub Repository](https://github.com/wayli-app/fluxbase)
- [API Reference](/docs/api/sdk/classes/FluxbaseClient)

### Supabase

- [Supabase Documentation](https://supabase.com/docs)
- [Supabase GitHub](https://github.com/supabase/supabase)
- [Pricing](https://supabase.com/pricing)

## ❓ FAQ

### Can I switch from Supabase to Fluxbase later?

The API compatibility makes code changes minimal, but consider:

- **Database migration**: Standard PostgreSQL tools (pg_dump/restore)
- **RLS policies**: Need syntax updates (`auth.uid()` → `current_setting()`)
- **Edge Functions**: Same Deno runtime, minimal changes
- **Not a one-click migration**: Requires planning and testing

### Does Fluxbase scale like Supabase?

**No.** Fluxbase is a single binary designed for vertical scaling (more CPU/RAM on one server), not horizontal scaling (multiple servers). Supabase supports read replicas and horizontal scaling, making it better for very high-traffic applications.

### Which is faster?

Both use PostgreSQL, so database performance is similar. Fluxbase may have slightly lower latency (fewer network hops), but Supabase can scale to handle more concurrent users.

### Can I use both?

Yes! You could:

- Develop locally with Fluxbase
- Deploy to Supabase for production (if you need hosted service)
- Or vice versa (prototype on Supabase free tier, self-host Fluxbase for production)

## 🎯 Conclusion

**Fluxbase** is ideal for developers who want maximum simplicity, full control, and predictable costs for moderate-scale applications.

**Supabase** is ideal for teams who want a managed service with a free tier, horizontal scalability, and don't want to manage infrastructure.

Both are excellent choices depending on your needs, team size, and scale requirements.
