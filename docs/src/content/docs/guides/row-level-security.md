---
title: "Row Level Security (RLS)"
description: Implement PostgreSQL Row Level Security in Fluxbase for fine-grained access control. Build secure multi-tenant applications with automatic database-level enforcement.
---

Row Level Security (RLS) is PostgreSQL's feature for controlling which rows users can access in database tables. Fluxbase provides seamless RLS integration for building secure multi-tenant applications.

## What is RLS?

RLS enables fine-grained access control at the row level. Instead of granting permissions to entire tables, you define policies that determine which rows each user can see, insert, update, or delete.

**Benefits:**

- Automatic enforcement at database level
- No application-level filtering code needed
- Defense in depth security
- Perfect for multi-tenant SaaS applications
- Version-controlled policies via migrations

**Without RLS:**

```typescript
// Must remember to filter by user
const tasks = await client
  .from("tasks")
  .select("*")
  .eq("user_id", currentUser.id); // Easy to forget!
```

**With RLS:**

```typescript
// Database automatically enforces access
const tasks = await client.from("tasks").select("*");
// Only returns current user's tasks
```

## How Fluxbase Implements RLS

Fluxbase sets PostgreSQL session variables from JWT tokens:

- `app.user_id` - Authenticated user's UUID
- `app.role` - User's role (`anon`, `authenticated`, `admin`)

These variables are used in RLS policy conditions.

## Default Permissions

Fluxbase sets up a comprehensive permission system out of the box. Understanding these defaults helps you build secure applications.

### Role Hierarchy

```mermaid
graph TD
    subgraph "Application Roles"
        A[Request] --> B{Has JWT?}
        B -->|No| C[anon]
        B -->|Yes| D{Valid token?}
        D -->|No| C
        D -->|Yes| E{User Role?}
        E -->|Regular User| F[authenticated]
        E -->|Tenant Admin| G[tenant_admin]
        E -->|Instance Admin| H[instance_admin]
    end

    subgraph "Database Roles"
        C --> I[anon role]
        F --> J[authenticated role]
        G --> J
        H --> K[service_role]
    end

    subgraph "Backend Only"
        L[service_role] --> K
        M[tenant_service] --> N[tenant_service role]
    end

    subgraph "RLS Behavior"
        I -->|"Minimal access"| O[RLS Policies Applied]
        J -->|"User context set"| O
        N -->|"Tenant context enforced"| O
        K -->|"BYPASSRLS"| P[Full Access - ALL Tenants]
    end

    style C fill:#ffcccc
    style F fill:#ccffcc
    style G fill:#99ff99
    style H fill:#66ff66
    style K fill:#ccccff
    style L fill:#ccccff
```

**Application roles map to database roles:**

| Application Role | Database Role    | BYPASSRLS | Access Scope                   |
| ---------------- | ---------------- | --------- | ------------------------------ |
| `anon`           | `anon`           | No        | Public data only               |
| `authenticated`  | `authenticated`  | No        | Own data + public data         |
| `tenant_admin`   | `authenticated`  | No        | Data in assigned tenants       |
| `instance_admin` | `service_role`   | **Yes**   | ALL data across ALL tenants    |
| `service_role`   | `service_role`   | **Yes**   | ALL data across ALL tenants    |
| `tenant_service` | `tenant_service` | No        | Data in current tenant context |

:::note[Instance Admin Bypasses RLS]
The `instance_admin` role maps to `service_role` which has PostgreSQL's `BYPASSRLS` privilege. This means instance admins can see ALL data across ALL tenants - use this role carefully and only for platform administrators who need full visibility.
:::

:::note[Tenant Admin Respects RLS]
The `tenant_admin` role maps to `authenticated` and respects RLS policies. Tenant admins can only access data in tenants they are assigned to via `platform.tenant_admin_assignments`. This provides secure tenant isolation.
:::

:::note[Tenant Service Role]
The `tenant_service` role is used for multi-tenant applications. It enforces RLS policies with tenant context, ensuring operations are automatically scoped to the tenant associated with the service key. This provides secure tenant isolation without bypassing RLS.
:::

### Permission Matrix

This table shows the default table-level permissions for each schema. Actual row access is further controlled by RLS policies.

| Schema         | anon            | authenticated | tenant_admin     | instance_admin |
| -------------- | --------------- | ------------- | ---------------- | -------------- |
| **auth**       | None            | Own data      | Tenant users     | ALL            |
| **app**        | Public settings | Own data      | Tenant settings  | ALL            |
| **storage**    | Public buckets  | Own objects   | Tenant objects   | ALL            |
| **functions**  | None            | Own functions | Tenant functions | ALL            |
| **realtime**   | None            | Subscriptions | Tenant config    | ALL            |
| **platform**   | None            | Own tenant    | Assigned tenants | ALL            |
| **jobs**       | None            | Own jobs      | Tenant jobs      | ALL            |
| **migrations** | None            | None          | None             | ALL            |
| **public**     | None            | Via RLS       | Via RLS          | ALL            |

### Multi-Tenancy Access Matrix

For multi-tenant deployments, access is controlled by tenant context:

| Resource             | anon | authenticated | tenant_admin | tenant_service | instance_admin |
| -------------------- | ---- | ------------- | ------------ | -------------- | -------------- |
| **All tenants**      | None | None          | None         | None           | Full Access    |
| **Assigned tenants** | None | None          | Full CRUD    | Full CRUD      | Full Access    |
| **Own records**      | None | Full CRUD     | Full CRUD    | Full CRUD      | Full Access    |
| **Public data**      | Read | Read          | Read         | Read           | Full Access    |

**Key Points:**

- **instance_admin**: Maps to `service_role` with `BYPASSRLS` - sees ALL data across ALL tenants
- **tenant_admin**: Maps to `authenticated` - can only access data in tenants they're assigned to
- **tenant_service**: Used for tenant-scoped API operations - respects RLS with tenant context
- **authenticated**: Regular users - can only access their own data

### Schema Details

#### auth

Anonymous users have no direct access to auth tables. All authentication operations (signup, signin, password reset) are performed internally using the service role. Authenticated users can view and update their own profile, manage their sessions and client keys. Dashboard admins can view all users and perform administrative actions.

#### app

Application settings with fine-grained access control. Anyone can read public, non-secret settings. Authenticated users can read all non-secret settings. Write access is controlled by the `editable_by` field on each setting.

#### storage

File storage with ownership-based access. Public buckets are readable by anyone. Users own their uploaded files and have full control over them. File sharing is managed through the permissions table. Admins can manage all files.

#### functions

Edge functions management. Only dashboard admins can create, update, or delete edge functions.

#### realtime

Realtime subscription configuration. Authenticated users can view the configuration. Only admins can modify it.

#### platform

Multi-tenancy and platform management tables (tenants, service_keys, tenant_admin_assignments). Instance admins have full access. Tenant admins can manage their tenant's keys. The `tenant_service` role provides automatic tenant isolation for tenant-scoped operations.

#### jobs

Background job processing. Users can view, submit, and cancel their own jobs. Dashboard admins can view all jobs. The service role manages workers and job execution.

#### migrations

Internal migration tracking. Internal declarative-schema state is tracked in `platform.declarative_state`; all user-facing migrations (filesystem and API-managed) are tracked in `platform.migrations` with different namespaces (`filesystem` for local files, custom namespaces for API). Only the service role has access. This schema is not exposed to regular users.

#### public

User-defined tables. **No default access for anon or authenticated users.** Only the service role has access by default. This "closed by default" approach ensures developers must explicitly define RLS policies to grant access, preventing accidental data exposure.

### Helper Functions

Fluxbase provides helper functions for use in RLS policies:

| Function                               | Returns   | Description                      |
| -------------------------------------- | --------- | -------------------------------- |
| `auth.current_user_id()`               | `uuid`    | Current user's ID from JWT       |
| `auth.uid()`                           | `uuid`    | Alias for `current_user_id()`    |
| `auth.current_user_role()`             | `text`    | Current role from JWT            |
| `auth.role()`                          | `text`    | Alias for `current_user_role()`  |
| `auth.is_admin()`                      | `boolean` | Whether current user is admin    |
| `auth.jwt()`                           | `jsonb`   | Full JWT claims                  |
| `auth.is_authenticated()`              | `boolean` | Whether user is authenticated    |
| `auth.has_tenant_access(tenant_id)`    | `boolean` | Check tenant context access      |
| `platform.is_instance_admin(user_id)`  | `boolean` | Check if user is instance admin  |
| `storage.has_tenant_access(tenant_id)` | `boolean` | Check tenant context for storage |

## Enable RLS on Tables

```sql
-- Enable RLS on a table
ALTER TABLE posts ENABLE ROW LEVEL SECURITY;

-- Create policy
CREATE POLICY "Users can view own posts"
ON posts
FOR SELECT
USING (current_setting('app.user_id', true)::uuid = user_id);
```

## Common RLS Patterns

### User-Owned Resources

```sql
-- Users can only see their own posts
CREATE POLICY "select_own_posts"
ON posts FOR SELECT
USING (current_setting('app.user_id', true)::uuid = user_id);

-- Users can only insert their own posts
CREATE POLICY "insert_own_posts"
ON posts FOR INSERT
WITH CHECK (current_setting('app.user_id', true)::uuid = user_id);

-- Users can only update their own posts
CREATE POLICY "update_own_posts"
ON posts FOR UPDATE
USING (current_setting('app.user_id', true)::uuid = user_id);

-- Users can only delete their own posts
CREATE POLICY "delete_own_posts"
ON posts FOR DELETE
USING (current_setting('app.user_id', true)::uuid = user_id);
```

### Public Read, Authenticated Write

```sql
-- Anyone can view posts
CREATE POLICY "public_read_posts"
ON posts FOR SELECT
USING (true);

-- Only authenticated users can insert
CREATE POLICY "authenticated_insert_posts"
ON posts FOR INSERT
WITH CHECK (current_setting('app.role', true) = 'authenticated');
```

### Role-Based Access

```sql
-- Admins can see all posts
-- Users can only see their own posts
CREATE POLICY "role_based_posts"
ON posts FOR SELECT
USING (
  current_setting('app.role', true) = 'admin'
  OR current_setting('app.user_id', true)::uuid = user_id
);
```

### Team/Organization Access

```sql
-- Users can see posts from their organization
CREATE POLICY "org_posts"
ON posts FOR SELECT
USING (
  organization_id IN (
    SELECT organization_id
    FROM user_organizations
    WHERE user_id = current_setting('app.user_id', true)::uuid
  )
);
```

### Combination Policies

```sql
-- Posts visible if:
-- 1. User owns the post, OR
-- 2. Post is published and public
CREATE POLICY "select_posts"
ON posts FOR SELECT
USING (
  current_setting('app.user_id', true)::uuid = user_id
  OR (published = true AND visibility = 'public')
);
```

### Multi-Tenant Isolation

Fluxbase uses a combination of session variables and helper functions for tenant isolation:

```sql
-- Service role bypasses all RLS (instance_admin)
-- Tenant users can only access their tenant's data
-- Regular users can access their own data
CREATE POLICY "tenant_settings_select"
ON platform.tenant_settings FOR SELECT TO PUBLIC
USING (
    CURRENT_USER = 'service_role'::name                    -- Instance admin bypass
    OR CURRENT_USER = 'tenant_service'::name               -- Tenant service with context
    OR auth.has_tenant_access(tenant_id)                   -- Tenant admin check
);

-- Tenant admin assignment check
CREATE POLICY "tenant_admin_access"
ON platform.tenants FOR SELECT TO PUBLIC
USING (
    CURRENT_USER = 'service_role'::name                    -- Instance admin bypass
    OR EXISTS (
        SELECT 1 FROM platform.tenant_admin_assignments
        WHERE tenant_id = tenants.id
        AND user_id = auth.uid()
        AND is_active = true
    )
);
```

### How Tenant Context Works

When a request comes in with tenant context, Fluxbase sets the `app.current_tenant_id` session variable:

```sql
-- The auth.has_tenant_access() function checks tenant context
CREATE OR REPLACE FUNCTION auth.has_tenant_access(resource_tenant_id uuid)
RETURNS boolean
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path = public AS $$
    SELECT CASE
        WHEN current_setting('app.current_tenant_id', TRUE) = '' THEN
            resource_tenant_id IS NULL
        ELSE
            resource_tenant_id::text = current_setting('app.current_tenant_id', TRUE)
    END;
$$;
```

**Tenant context sources (in order of precedence):**

1. `X-FB-Tenant` header - Explicit tenant override
2. JWT `tenant_id` claim - From authentication
3. Default tenant - Fallback for single-tenant deployments

## Helper Functions

Create helper functions for cleaner policies:

```sql
-- Get current user ID
CREATE FUNCTION auth_user_id()
RETURNS uuid
LANGUAGE sql STABLE
AS $$
  SELECT current_setting('app.user_id', true)::uuid;
$$;

-- Get current user role
CREATE FUNCTION auth_role()
RETURNS text
LANGUAGE sql STABLE
AS $$
  SELECT current_setting('app.role', true);
$$;

-- Check if user is admin
CREATE FUNCTION is_admin()
RETURNS boolean
LANGUAGE sql STABLE
AS $$
  SELECT current_setting('app.role', true) = 'admin';
$$;
```

Use in policies:

```sql
CREATE POLICY "select_own_posts"
ON posts FOR SELECT
USING (auth_user_id() = user_id);

CREATE POLICY "admin_full_access"
ON posts FOR ALL
USING (is_admin());
```

## Multiple Policies

You can create multiple policies for the same operation. They are combined with OR logic:

```sql
-- Policy 1: Users see own posts
CREATE POLICY "own_posts"
ON posts FOR SELECT
USING (auth_user_id() = user_id);

-- Policy 2: Users see published posts
CREATE POLICY "published_posts"
ON posts FOR SELECT
USING (published = true);

-- Result: Users see their own posts OR published posts
```

## Testing RLS Policies

### Test as Specific User

```sql
-- Set session variables manually
SET LOCAL app.user_id = '123e4567-e89b-12d3-a456-426614174000';
SET LOCAL app.role = 'authenticated';

-- Test query
SELECT * FROM posts;
-- Should only return posts accessible to this user
```

### Test via SDK

```typescript
import { createClient } from "@nimbleflux/fluxbase-sdk";

const client = createClient("http://localhost:8080", "user-api-key");

// Queries automatically use authenticated user's context
const posts = await client.from("posts").select("*");

// Should only return posts user has access to
console.log(posts.data);
```

### Test Service Role (Bypass RLS)

```typescript
const adminClient = createClient("http://localhost:8080", {
  serviceKey: process.env.SERVICE_KEY,
});

// Service key bypasses RLS - returns ALL posts
const allPosts = await adminClient.from("posts").select("*");
```

## Performance Considerations

**Index on Policy Columns:**

```sql
-- If policies filter by user_id, index it
CREATE INDEX IF NOT EXISTS idx_posts_user_id ON posts(user_id);

-- If policies check organization_id
CREATE INDEX IF NOT EXISTS idx_posts_org_id ON posts(organization_id);
```

**Avoid Complex Subqueries:**

```sql
-- Slow: Subquery runs for each row
CREATE POLICY "slow_policy"
ON posts FOR SELECT
USING (
  user_id IN (
    SELECT user_id FROM teams
    WHERE team_id = (SELECT team_id FROM user_teams WHERE user_id = auth_user_id())
  )
);

-- Better: Join or simplified logic
-- Consider denormalizing team_id onto posts table
```

**Use STABLE Functions:**
Mark helper functions as STABLE (not VOLATILE) to allow caching:

```sql
CREATE FUNCTION auth_user_id()
RETURNS uuid
LANGUAGE sql STABLE  -- STABLE, not VOLATILE
AS $$ ... $$;
```

## Security Best Practices

**Always Enable RLS:**

```sql
-- Enable on all user-accessible tables
ALTER TABLE posts ENABLE ROW LEVEL SECURITY;
ALTER TABLE comments ENABLE ROW LEVEL SECURITY;
ALTER TABLE profiles ENABLE ROW LEVEL SECURITY;
```

**Default Deny:**

```sql
-- No policies = no access (except for service role)
-- This is secure by default
```

**Explicit Policies:**

```sql
-- Be explicit about what each policy allows
-- Don't use overly permissive policies like:
-- USING (true)  -- Allows everything!
```

**Test Thoroughly:**

- Test as different users
- Test unauthorized access attempts
- Test with anonymous users
- Test with service role

**Audit Policies:**

```sql
-- List all policies for a table
SELECT * FROM pg_policies WHERE tablename = 'posts';
```

## Debugging RLS

### Check if RLS is Enabled

```sql
SELECT tablename, rowsecurity
FROM pg_tables
WHERE schemaname = 'public';
```

### View Policies

```sql
SELECT *
FROM pg_policies
WHERE tablename = 'posts';
```

### Check Session Variables

```sql
SELECT current_setting('app.user_id', true);
SELECT current_setting('app.role', true);
```

### Explain Query with RLS

```sql
SET LOCAL app.user_id = 'some-uuid';
EXPLAIN ANALYZE
SELECT * FROM posts;
-- Shows how RLS policies affect query plan
```

## Bypassing RLS (Admin Operations)

Use service keys to bypass RLS for administrative operations:

```typescript
const adminClient = createClient("http://localhost:8080", {
  serviceKey: process.env.FLUXBASE_SERVICE_ROLE_KEY,
});

// Bypasses all RLS policies
const allUsers = await adminClient.from("users").select("*");
```

**Security:**

- Never expose service keys in client code
- Use only in backend services
- Store in secure secrets management

## Common Issues

**Policy Not Applied:**

- Verify RLS is enabled: `ALTER TABLE ... ENABLE ROW LEVEL SECURITY`
- Check policy exists: `SELECT * FROM pg_policies WHERE tablename = 'table_name'`
- Ensure session variables are set

**Empty Results:**

- Policy may be too restrictive
- Check `current_setting('app.user_id')` is set correctly
- Test with simpler policy first

**Performance Issues:**

- Add indexes on columns used in policies
- Avoid complex subqueries in policies
- Use `EXPLAIN ANALYZE` to identify bottlenecks

**Service Role Still Affected:**

- Service keys should bypass RLS
- Verify you're using `serviceKey` not `apiKey` in SDK

## Related Documentation

- [Authentication](/guides/authentication) - JWT tokens and roles
- [SDK Reference](/api/sdk/readme/) - Table management and querying
- [Security](/security/overview) - Overall security best practices
