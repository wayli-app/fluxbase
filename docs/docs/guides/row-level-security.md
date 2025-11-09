---
title: Row Level Security (RLS)
sidebar_position: 8
---

# Row Level Security (RLS)

Row Level Security (RLS) is PostgreSQL's powerful feature that allows you to control which rows users can access in database tables. Fluxbase provides seamless RLS integration, making it easy to build secure multi-tenant applications where users can only see and modify their own data.

## Table of Contents

- [What is RLS?](#what-is-rls)
- [How Fluxbase Implements RLS](#how-fluxbase-implements-rls)
- [Configuration](#configuration)
- [Helper Functions](#helper-functions)
- [Creating RLS Policies](#creating-rls-policies)
- [Common Patterns](#common-patterns)
- [Testing RLS Policies](#testing-rls-policies)
- [Performance Considerations](#performance-considerations)
- [Security Best Practices](#security-best-practices)
- [Debugging](#debugging)
- [Advanced Topics](#advanced-topics)

---

## What is RLS?

Row Level Security is a PostgreSQL security feature that enables fine-grained access control at the row level. Instead of granting permissions to entire tables, RLS allows you to define policies that determine which rows each user can see, insert, update, or delete.

### Key Benefits

1. **Automatic Enforcement**: PostgreSQL enforces access control at the database level
2. **No Application Code Changes**: Your API doesn't need filtering logic
3. **Defense in Depth**: Even if application code has bugs, the database protects data
4. **Multi-Tenancy Made Easy**: Perfect for SaaS applications with isolated customer data
5. **Auditable**: Policies are defined in SQL and version-controlled with migrations

### Without RLS vs. With RLS

**Without RLS** (application-level filtering):

```typescript
// Application code must remember to filter by user_id
const tasks = await client
  .from("tasks")
  .select("*")
  .eq("user_id", currentUser.id) // Easy to forget!
  .execute();
```

**With RLS** (database-level enforcement):

```typescript
// No filtering needed - database automatically enforces access
const tasks = await client.from("tasks").select("*").execute(); // Only returns current user's tasks
```

---

## How Fluxbase Implements RLS

Fluxbase's RLS implementation consists of three components working together:

### 1. Authentication Middleware

When a user authenticates, Fluxbase's auth middleware extracts the user ID and role from the JWT token and stores them in the Fiber context.

### 2. RLS Middleware

Before each database query, the RLS middleware sets PostgreSQL session variables:

- `app.user_id`: The authenticated user's UUID
- `app.role`: The user's role (`anon`, `authenticated`, `admin`, etc.)

```go
// Example: Setting RLS context
SELECT set_config('app.user_id', '123e4567-e89b-12d3-a456-426614174000', true);
SELECT set_config('app.role', 'authenticated', true);
```

### 3. RLS Policies

Your database policies use these session variables to filter rows:

```sql
CREATE POLICY user_tasks ON tasks
  FOR SELECT
  USING (user_id = auth.current_user_id());
```

**Flow Diagram:**

```
User Request
    ↓
Auth Middleware (extracts JWT)
    ↓
RLS Middleware (sets app.user_id, app.role)
    ↓
Database Query
    ↓
PostgreSQL RLS (filters rows based on policies)
    ↓
Response (only authorized rows)
```

---

## Configuration

### Enabling/Disabling RLS

RLS is **enabled by default**. To disable it (not recommended for production):

**Via `fluxbase.yaml`:**

```yaml
auth:
  enable_rls: false
```

**Via Environment Variable:**

```bash
FLUXBASE_AUTH_ENABLE_RLS=false
```

**⚠️ Warning:** Disabling RLS removes automatic row-level access control. Only disable for development/testing or if you have alternative security measures.

### Per-Table Configuration

Even with RLS enabled globally, you control which tables use RLS:

```sql
-- Enable RLS on a table
ALTER TABLE public.my_table ENABLE ROW LEVEL SECURITY;

-- Force RLS even for table owners
ALTER TABLE public.my_table FORCE ROW LEVEL SECURITY;

-- Disable RLS on a table
ALTER TABLE public.my_table DISABLE ROW LEVEL SECURITY;
```

---

## Helper Functions

Fluxbase provides SQL helper functions for writing RLS policies:

### `auth.current_user_id()` → UUID

Returns the authenticated user's ID from the session variable `app.user_id`.

```sql
CREATE POLICY user_data ON my_table
  FOR SELECT
  USING (user_id = auth.current_user_id());
```

**Returns:**

- User's UUID if authenticated
- `NULL` if not authenticated (anonymous request)

### `auth.current_user_role()` → TEXT

Returns the current user's role from the session variable `app.role`.

```sql
CREATE POLICY admin_access ON sensitive_data
  FOR ALL
  USING (auth.current_user_role() = 'admin');
```

**Common Roles:**

- `anon`: Unauthenticated/anonymous users
- `authenticated`: Logged-in users
- `admin`: Admin users
- `dashboard_admin`: Dashboard administrators
- `service_role`: Service accounts (for backend services)

### `auth.is_authenticated()` → BOOLEAN

Returns `TRUE` if a user is logged in (i.e., `auth.current_user_id()` is not NULL).

```sql
CREATE POLICY authenticated_only ON premium_features
  FOR SELECT
  USING (auth.is_authenticated());
```

### `auth.is_admin()` → BOOLEAN

Returns `TRUE` if the current user has the `admin` role.

```sql
CREATE POLICY admin_full_access ON all_data
  FOR ALL
  USING (auth.is_admin());
```

---

## Creating RLS Policies

### Policy Basics

An RLS policy has four components:

1. **Command**: Which SQL operation (SELECT, INSERT, UPDATE, DELETE, or ALL)
2. **USING Expression**: Which rows the user can access
3. **WITH CHECK Expression**: Which rows the user can create/modify (for INSERT/UPDATE)
4. **Roles**: Which PostgreSQL roles the policy applies to (optional)

### Basic Policy Syntax

```sql
CREATE POLICY policy_name ON table_name
  FOR command
  [TO role_name]
  USING (expression)       -- Rows user can access
  [WITH CHECK (expression)]; -- Rows user can create/modify
```

### Example: Simple User Isolation

```sql
-- Create a tasks table
CREATE TABLE public.tasks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES auth.users(id),
  title TEXT NOT NULL,
  completed BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Enable RLS
ALTER TABLE public.tasks ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.tasks FORCE ROW LEVEL SECURITY;

-- Policy: Users can only see their own tasks
CREATE POLICY tasks_select ON public.tasks
  FOR SELECT
  USING (user_id = auth.current_user_id());

-- Policy: Users can only insert their own tasks
CREATE POLICY tasks_insert ON public.tasks
  FOR INSERT
  WITH CHECK (
    auth.is_authenticated()
    AND user_id = auth.current_user_id()
  );

-- Policy: Users can only update their own tasks
CREATE POLICY tasks_update ON public.tasks
  FOR UPDATE
  USING (user_id = auth.current_user_id())
  WITH CHECK (user_id = auth.current_user_id());

-- Policy: Users can only delete their own tasks
CREATE POLICY tasks_delete ON public.tasks
  FOR DELETE
  USING (user_id = auth.current_user_id());

-- Grant table permissions to roles
GRANT SELECT, INSERT, UPDATE, DELETE ON public.tasks TO authenticated;
GRANT SELECT ON public.tasks TO anon;
```

### USING vs. WITH CHECK

- **USING**: Determines which existing rows the user can access
- **WITH CHECK**: Determines which new/modified rows the user can create

**Example:**

```sql
-- User can read all tasks, but can only create/modify their own
CREATE POLICY tasks_policy ON tasks
  FOR ALL
  USING (TRUE)  -- Can read any row
  WITH CHECK (user_id = auth.current_user_id());  -- Can only create own rows
```

---

## Common Patterns

### Pattern 1: Public + Private Data

Allow everyone to see public content, but only owners can see private content.

```sql
CREATE TABLE public.posts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES auth.users(id),
  title TEXT NOT NULL,
  content TEXT,
  is_public BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

ALTER TABLE public.posts ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.posts FORCE ROW LEVEL SECURITY;

-- Policy: Anyone can see public posts, owners can see their private posts
CREATE POLICY posts_select ON public.posts
  FOR SELECT
  USING (
    is_public = TRUE
    OR user_id = auth.current_user_id()
  );

-- Policy: Only authenticated users can insert their own posts
CREATE POLICY posts_insert ON public.posts
  FOR INSERT
  WITH CHECK (
    auth.is_authenticated()
    AND user_id = auth.current_user_id()
  );

-- Policy: Only owners can update their posts
CREATE POLICY posts_update ON public.posts
  FOR UPDATE
  USING (user_id = auth.current_user_id())
  WITH CHECK (user_id = auth.current_user_id());

-- Policy: Only owners can delete their posts
CREATE POLICY posts_delete ON public.posts
  FOR DELETE
  USING (user_id = auth.current_user_id());

-- Grant permissions
GRANT SELECT, INSERT, UPDATE, DELETE ON public.posts TO authenticated;
GRANT SELECT ON public.posts TO anon;  -- Anonymous users can read public posts
```

### Pattern 2: Organization/Team-Based Access

Users can access data belonging to their organization or team.

```sql
CREATE TABLE public.documents (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL,
  title TEXT NOT NULL,
  content TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE public.user_organizations (
  user_id UUID REFERENCES auth.users(id),
  organization_id UUID,
  role TEXT NOT NULL, -- 'owner', 'admin', 'member'
  PRIMARY KEY (user_id, organization_id)
);

ALTER TABLE public.documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.documents FORCE ROW LEVEL SECURITY;

-- Policy: Users can see documents from their organizations
CREATE POLICY documents_select ON public.documents
  FOR SELECT
  USING (
    organization_id IN (
      SELECT organization_id
      FROM public.user_organizations
      WHERE user_id = auth.current_user_id()
    )
  );

-- Policy: Only org admins/owners can insert documents
CREATE POLICY documents_insert ON public.documents
  FOR INSERT
  WITH CHECK (
    organization_id IN (
      SELECT organization_id
      FROM public.user_organizations
      WHERE user_id = auth.current_user_id()
        AND role IN ('owner', 'admin')
    )
  );

GRANT SELECT, INSERT, UPDATE, DELETE ON public.documents TO authenticated;
GRANT SELECT ON public.user_organizations TO authenticated;
```

### Pattern 3: Role-Based Access Control (RBAC)

Different permissions based on user roles.

```sql
CREATE TABLE public.sensitive_data (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES auth.users(id),
  department_id UUID,
  data TEXT,
  security_level INT DEFAULT 1
);

ALTER TABLE public.sensitive_data ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.sensitive_data FORCE ROW LEVEL SECURITY;

-- Policy: Access based on role and security level
CREATE POLICY rbac_access ON public.sensitive_data
  FOR SELECT
  USING (
    CASE auth.current_user_role()
      -- Admins can see everything
      WHEN 'admin' THEN TRUE
      -- Managers can see their department's data up to level 3
      WHEN 'manager' THEN
        department_id = (
          SELECT department_id FROM auth.users
          WHERE id = auth.current_user_id()
        )
        AND security_level <= 3
      -- Regular users can only see their own data at level 1
      WHEN 'authenticated' THEN
        user_id = auth.current_user_id()
        AND security_level = 1
      -- Anonymous users see nothing
      ELSE FALSE
    END
  );

GRANT SELECT ON public.sensitive_data TO authenticated, anon;
```

### Pattern 4: Time-Based Access

Restrict access based on time periods.

```sql
CREATE TABLE public.scheduled_content (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title TEXT NOT NULL,
  content TEXT,
  publish_at TIMESTAMPTZ NOT NULL,
  unpublish_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

ALTER TABLE public.scheduled_content ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.scheduled_content FORCE ROW LEVEL SECURITY;

-- Policy: Content is visible only during its scheduled time
CREATE POLICY scheduled_access ON public.scheduled_content
  FOR SELECT
  USING (
    NOW() >= publish_at
    AND (unpublish_at IS NULL OR NOW() < unpublish_at)
  );

GRANT SELECT ON public.scheduled_content TO authenticated, anon;
```

### Pattern 5: Cascading Permissions

Inherit permissions from related records.

```sql
CREATE TABLE public.projects (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_id UUID REFERENCES auth.users(id),
  name TEXT NOT NULL
);

CREATE TABLE public.project_members (
  project_id UUID REFERENCES public.projects(id),
  user_id UUID REFERENCES auth.users(id),
  role TEXT NOT NULL,
  PRIMARY KEY (project_id, user_id)
);

CREATE TABLE public.project_files (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID REFERENCES public.projects(id),
  filename TEXT NOT NULL,
  content BYTEA
);

-- Enable RLS on all tables
ALTER TABLE public.projects ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.project_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.project_files ENABLE ROW LEVEL SECURITY;

ALTER TABLE public.projects FORCE ROW LEVEL SECURITY;
ALTER TABLE public.project_members FORCE ROW LEVEL SECURITY;
ALTER TABLE public.project_files FORCE ROW LEVEL SECURITY;

-- Project access
CREATE POLICY projects_access ON public.projects
  FOR SELECT
  USING (
    owner_id = auth.current_user_id()
    OR id IN (
      SELECT project_id FROM public.project_members
      WHERE user_id = auth.current_user_id()
    )
  );

-- Project members access
CREATE POLICY members_access ON public.project_members
  FOR SELECT
  USING (
    project_id IN (
      SELECT id FROM public.projects
      WHERE owner_id = auth.current_user_id()
        OR id IN (
          SELECT project_id FROM public.project_members
          WHERE user_id = auth.current_user_id()
        )
    )
  );

-- Files inherit project permissions
CREATE POLICY files_access ON public.project_files
  FOR SELECT
  USING (
    project_id IN (
      SELECT id FROM public.projects
      WHERE owner_id = auth.current_user_id()
        OR id IN (
          SELECT project_id FROM public.project_members
          WHERE user_id = auth.current_user_id()
        )
    )
  );

GRANT SELECT ON public.projects, public.project_members, public.project_files TO authenticated;
```

---

## Testing RLS Policies

### 1. Unit Testing in Migrations

Create up/down migrations for your RLS policies:

```sql
-- migrations/010_user_tasks_rls.up.sql
BEGIN;

CREATE TABLE IF NOT EXISTS public.tasks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES auth.users(id),
  title TEXT NOT NULL,
  completed BOOLEAN DEFAULT FALSE
);

ALTER TABLE public.tasks ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.tasks FORCE ROW LEVEL SECURITY;

CREATE POLICY tasks_isolation ON public.tasks
  FOR ALL
  USING (user_id = auth.current_user_id())
  WITH CHECK (user_id = auth.current_user_id());

GRANT ALL ON public.tasks TO authenticated;

COMMIT;
```

```sql
-- migrations/010_user_tasks_rls.down.sql
BEGIN;

DROP POLICY IF EXISTS tasks_isolation ON public.tasks;
ALTER TABLE public.tasks DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS public.tasks;

COMMIT;
```

### 2. Manual Testing with psql

```bash
# Connect to your database
psql $DATABASE_URL

# Simulate a specific user
SET app.user_id = '123e4567-e89b-12d3-a456-426614174000';
SET app.role = 'authenticated';

# Test queries
SELECT * FROM public.tasks;  -- Should only show this user's tasks

# Reset to test as another user
SET app.user_id = '987fcdeb-51a2-43f1-9876-543210fedcba';
SELECT * FROM public.tasks;  -- Should show different tasks

# Test as anonymous
RESET app.user_id;
SET app.role = 'anon';
SELECT * FROM public.tasks;  -- Should follow anonymous policies
```

### 3. API Testing

```typescript
import { createClient } from "@fluxbase/sdk";

const client = createClient({ url: "http://localhost:8080" });

// Create test users
const { user: user1 } = await client.auth.signUp({
  email: "user1@example.com",
  password: "password123",
});

const { user: user2 } = await client.auth.signUp({
  email: "user2@example.com",
  password: "password123",
});

// User 1 creates a task
await client.auth.signIn({
  email: "user1@example.com",
  password: "password123",
});

await client
  .from("tasks")
  .insert({
    user_id: user1.id,
    title: "User 1 Task",
  })
  .execute();

// User 2 should NOT see User 1's task
await client.auth.signIn({
  email: "user2@example.com",
  password: "password123",
});

const { data } = await client.from("tasks").select("*").execute();
console.assert(data.length === 0, "User 2 should not see User 1 tasks");
console.log("✅ RLS test passed");
```

### 4. Automated E2E Tests

Fluxbase includes comprehensive RLS E2E tests. See [`test/e2e/rls_test.go`](../../../test/e2e/rls_test.go) for examples.

```go
func TestRLSUserCanAccessOwnData(t *testing.T) {
    tc := setupRLSTest(t)
    defer tc.Close()

    // Create two users
    user1ID, token1 := tc.CreateTestUser("user1@example.com", "password123")
    user2ID, token2 := tc.CreateTestUser("user2@example.com", "password123")

    // User 1 creates a task
    tc.NewRequest("POST", "/api/v1/tables/tasks").
        WithAuth(token1).
        WithBody(map[string]interface{}{
            "user_id": user1ID,
            "title":   "User 1 Task",
        }).
        Send().
        AssertStatus(201)

    // User 2 queries tasks - should only see their own (none)
    resp := tc.NewRequest("GET", "/api/v1/tables/tasks").
        WithAuth(token2).
        Send().
        AssertStatus(200)

    var tasks []map[string]interface{}
    resp.JSON(&tasks)
    require.Len(t, tasks, 0, "User 2 should not see User 1's tasks")
}
```

---

## Performance Considerations

### 1. Index Foreign Keys

**Always** index columns used in RLS policies, especially `user_id`:

```sql
CREATE INDEX idx_tasks_user_id ON public.tasks(user_id);
CREATE INDEX idx_documents_organization_id ON public.documents(organization_id);
```

**Without index:**

```
Seq Scan on tasks (cost=0.00..1000.00 rows=10000 width=100)
  Filter: (user_id = current_setting('app.user_id')::uuid)
```

**With index:**

```
Index Scan using idx_tasks_user_id on tasks (cost=0.15..8.20 rows=5 width=100)
  Index Cond: (user_id = current_setting('app.user_id')::uuid)
```

### 2. Keep Policies Simple

Complex policies with subqueries can be slow. Optimize by:

**Bad (slow subquery):**

```sql
CREATE POLICY complex_access ON documents
  FOR SELECT
  USING (
    user_id IN (
      SELECT u.id FROM users u
      JOIN organizations o ON u.org_id = o.id
      WHERE o.id = (SELECT org_id FROM user_settings WHERE user_id = auth.current_user_id())
    )
  );
```

**Good (simple check with indexed column):**

```sql
-- Add org_id directly to documents table
ALTER TABLE documents ADD COLUMN org_id UUID;
CREATE INDEX idx_documents_org_id ON documents(org_id);

CREATE POLICY simple_access ON documents
  FOR SELECT
  USING (
    org_id IN (
      SELECT org_id FROM user_organizations
      WHERE user_id = auth.current_user_id()
    )
  );
```

### 3. Use Security Definer Functions

For complex permission logic, use `SECURITY DEFINER` functions:

```sql
-- Create a function that checks permissions
CREATE OR REPLACE FUNCTION auth.can_access_document(doc_id UUID)
RETURNS BOOLEAN
SECURITY DEFINER
SET search_path = public, auth
AS $$
BEGIN
  RETURN EXISTS (
    SELECT 1 FROM documents d
    JOIN project_members pm ON d.project_id = pm.project_id
    WHERE d.id = doc_id
      AND pm.user_id = auth.current_user_id()
      AND pm.role IN ('owner', 'admin', 'member')
  );
END;
$$ LANGUAGE plpgsql STABLE;

-- Use function in policy (faster than inline subquery)
CREATE POLICY document_access ON documents
  FOR SELECT
  USING (auth.can_access_document(id));
```

### 4. Benchmark with Realistic Data

Test policies with production-like data volumes:

```sql
-- Generate test data
INSERT INTO tasks (user_id, title)
SELECT
  (SELECT id FROM auth.users ORDER BY RANDOM() LIMIT 1),
  'Task ' || generate_series
FROM generate_series(1, 100000);

-- Benchmark query
EXPLAIN ANALYZE
SELECT * FROM tasks
WHERE completed = FALSE;
```

### 5. Consider Partition Tables

For very large tables, partition by tenant/org:

```sql
CREATE TABLE tasks (
  id UUID DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  title TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW()
) PARTITION BY LIST (user_id);

-- Create partitions per user or organization
CREATE TABLE tasks_user_1 PARTITION OF tasks
  FOR VALUES IN ('123e4567-e89b-12d3-a456-426614174000');
```

---

## Security Best Practices

### 1. Always Use FORCE ROW LEVEL SECURITY

```sql
-- ✅ GOOD: Even table owners can't bypass RLS
ALTER TABLE public.sensitive_data ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.sensitive_data FORCE ROW LEVEL SECURITY;

-- ❌ BAD: Table owners can bypass RLS
ALTER TABLE public.sensitive_data ENABLE ROW LEVEL SECURITY;
```

### 2. Grant Minimal Permissions

```sql
-- ✅ GOOD: Only grant necessary permissions
GRANT SELECT, INSERT ON public.tasks TO authenticated;
GRANT SELECT ON public.tasks TO anon;

-- ❌ BAD: Don't grant ALL to everyone
GRANT ALL ON public.tasks TO authenticated, anon;
```

### 3. Validate WITH CHECK Expressions

```sql
-- ✅ GOOD: Prevent privilege escalation
CREATE POLICY tasks_insert ON tasks
  FOR INSERT
  WITH CHECK (
    auth.is_authenticated()
    AND user_id = auth.current_user_id()
    AND created_at >= NOW() - INTERVAL '1 minute'  -- Prevent backdating
  );

-- ❌ BAD: No validation allows users to create tasks for others
CREATE POLICY tasks_insert ON tasks
  FOR INSERT
  WITH CHECK (TRUE);
```

### 4. Test All Permission Combinations

Test matrix:

| User Type             | SELECT     | INSERT  | UPDATE  | DELETE  |
| --------------------- | ---------- | ------- | ------- | ------- |
| Anonymous             | ✓ (public) | ✗       | ✗       | ✗       |
| Authenticated (own)   | ✓          | ✓       | ✓       | ✓       |
| Authenticated (other) | ✗          | ✗       | ✗       | ✗       |
| Admin                 | ✓ (all)    | ✓ (all) | ✓ (all) | ✓ (all) |

### 5. Audit Policies Regularly

```sql
-- List all policies on a table
SELECT
  schemaname,
  tablename,
  policyname,
  permissive,
  roles,
  cmd,
  qual,
  with_check
FROM pg_policies
WHERE tablename = 'tasks';
```

### 6. Use Separate Policies for Each Operation

```sql
-- ✅ GOOD: Separate policies are clearer and easier to maintain
CREATE POLICY tasks_select ON tasks FOR SELECT USING (...);
CREATE POLICY tasks_insert ON tasks FOR INSERT WITH CHECK (...);
CREATE POLICY tasks_update ON tasks FOR UPDATE USING (...) WITH CHECK (...);
CREATE POLICY tasks_delete ON tasks FOR DELETE USING (...);

-- ❌ BAD: Single policy for all operations is harder to understand
CREATE POLICY tasks_all ON tasks FOR ALL USING (...) WITH CHECK (...);
```

### 7. Document Your Policies

```sql
COMMENT ON POLICY tasks_select ON public.tasks IS
  'Users can only see their own tasks. Admins can see all tasks.';

COMMENT ON TABLE public.tasks IS
  'Task management with user isolation via RLS';
```

---

## Debugging

### Check Session Variables

```sql
-- See current RLS context
SELECT
  current_setting('app.user_id', TRUE) AS user_id,
  current_setting('app.role', TRUE) AS role,
  current_user AS pg_user;
```

### View Active Policies

```sql
-- List all policies on a table
\d+ public.tasks

-- Or query directly
SELECT * FROM pg_policies WHERE tablename = 'tasks';
```

### Test Specific User Context

```sql
BEGIN;

-- Set user context
SET LOCAL app.user_id = '123e4567-e89b-12d3-a456-426614174000';
SET LOCAL app.role = 'authenticated';

-- Run test query
SELECT * FROM tasks;

ROLLBACK;  -- Don't commit test data
```

### Explain Query Plans

```sql
-- See if RLS is causing performance issues
EXPLAIN (ANALYZE, BUFFERS)
SELECT * FROM tasks WHERE completed = FALSE;
```

### Enable PostgreSQL Query Logging

```yaml
# fluxbase.yaml
database:
  log_queries: true
  log_level: debug
```

```bash
# Or via environment
FLUXBASE_DATABASE_LOG_QUERIES=true
```

### Common Issues

**Issue: "Permission denied for table"**

```sql
-- Solution: Grant permissions to role
GRANT SELECT ON public.tasks TO authenticated;
```

**Issue: "RLS policy prevents access"**

```sql
-- Solution: Check if policy is too restrictive
SELECT * FROM pg_policies WHERE tablename = 'tasks';
-- Review USING and WITH CHECK expressions
```

**Issue: "Infinite recursion in RLS policy"**

```sql
-- Problem: Policy references itself
CREATE POLICY bad_policy ON tasks
  FOR SELECT
  USING (id IN (SELECT id FROM tasks));  -- ❌ Circular reference

-- Solution: Use simpler expression
CREATE POLICY good_policy ON tasks
  FOR SELECT
  USING (user_id = auth.current_user_id());
```

---

## Advanced Topics

### Bypassing RLS for Admin Operations

```sql
-- Create admin function that bypasses RLS
CREATE OR REPLACE FUNCTION admin.delete_all_user_data(target_user_id UUID)
RETURNS VOID
SECURITY DEFINER  -- Runs with function owner's permissions
SET search_path = public, auth
AS $$
BEGIN
  -- This function bypasses RLS because SECURITY DEFINER
  DELETE FROM tasks WHERE user_id = target_user_id;
  DELETE FROM documents WHERE user_id = target_user_id;
  -- Add other cleanup...
END;
$$ LANGUAGE plpgsql;

-- Only admins can execute this function
REVOKE EXECUTE ON FUNCTION admin.delete_all_user_data FROM PUBLIC;
GRANT EXECUTE ON FUNCTION admin.delete_all_user_data TO dashboard_admin;
```

### Multi-Tenancy with Schema Isolation

For strict tenant isolation, consider schema-per-tenant:

```sql
-- Create tenant schema
CREATE SCHEMA tenant_acme;

-- Create tables in tenant schema
CREATE TABLE tenant_acme.tasks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title TEXT NOT NULL
);

-- Set search path for tenant
SET search_path = tenant_acme, public;
```

### Combining RLS with Column-Level Security

```sql
-- Hide sensitive columns
CREATE TABLE users_extended (
  user_id UUID PRIMARY KEY,
  email TEXT,
  ssn TEXT,  -- Sensitive
  salary INT  -- Sensitive
);

ALTER TABLE users_extended ENABLE ROW LEVEL SECURITY;
ALTER TABLE users_extended FORCE ROW LEVEL SECURITY;

-- Row policy
CREATE POLICY users_access ON users_extended
  FOR SELECT
  USING (user_id = auth.current_user_id() OR auth.is_admin());

-- Column permissions (only admins see sensitive columns)
REVOKE ALL ON users_extended FROM authenticated;
GRANT SELECT (user_id, email) ON users_extended TO authenticated;
GRANT SELECT ON users_extended TO admin;
```

### Performance: Materialized Views

For expensive policies, use materialized views:

```sql
-- Create materialized view with pre-computed access
CREATE MATERIALIZED VIEW user_accessible_documents AS
SELECT d.*, uo.user_id
FROM documents d
JOIN user_organizations uo ON d.org_id = uo.org_id;

CREATE INDEX idx_user_docs ON user_accessible_documents(user_id);

-- Simple RLS policy on materialized view
ALTER TABLE user_accessible_documents ENABLE ROW LEVEL SECURITY;
CREATE POLICY user_docs_access ON user_accessible_documents
  FOR SELECT
  USING (user_id = auth.current_user_id());

-- Refresh periodically
REFRESH MATERIALIZED VIEW user_accessible_documents;
```

---

## Migration Examples

### Creating RLS-Enabled Table

```sql
-- migrations/010_create_tasks.up.sql
BEGIN;

CREATE TABLE IF NOT EXISTS public.tasks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  description TEXT,
  completed BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for RLS policy
CREATE INDEX idx_tasks_user_id ON public.tasks(user_id);

-- Enable RLS
ALTER TABLE public.tasks ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.tasks FORCE ROW LEVEL SECURITY;

-- Policies
CREATE POLICY tasks_user_isolation ON public.tasks
  FOR ALL
  USING (user_id = auth.current_user_id())
  WITH CHECK (user_id = auth.current_user_id());

CREATE POLICY tasks_admin_all ON public.tasks
  FOR ALL
  USING (auth.is_admin());

-- Permissions
GRANT SELECT, INSERT, UPDATE, DELETE ON public.tasks TO authenticated;

COMMENT ON TABLE public.tasks IS 'User tasks with RLS isolation';

COMMIT;
```

```sql
-- migrations/010_create_tasks.down.sql
BEGIN;

DROP POLICY IF EXISTS tasks_admin_all ON public.tasks;
DROP POLICY IF EXISTS tasks_user_isolation ON public.tasks;
ALTER TABLE public.tasks DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS public.tasks;

COMMIT;
```

---

## Further Reading

- [PostgreSQL RLS Documentation](https://www.postgresql.org/docs/current/ddl-rowsecurity.html)
- [Fluxbase Middleware: rls.go](../../internal/middleware/rls.go)
- [Example Tests: rls_test.go](../../test/e2e/rls_test.go)
- [Database Schema: 001_fluxbase_schema.up.sql](../../internal/database/migrations/001_fluxbase_schema.up.sql)

---

## Summary

Row Level Security is a powerful tool for building secure, multi-tenant applications. Key takeaways:

- ✅ **Enable RLS on all tables with sensitive data**
- ✅ **Use `FORCE ROW LEVEL SECURITY` to prevent bypass**
- ✅ **Index columns used in policies (especially user_id)**
- ✅ **Keep policies simple for better performance**
- ✅ **Test policies thoroughly with different user contexts**
- ✅ **Grant minimal necessary permissions**
- ✅ **Document your policies and audit them regularly**

With Fluxbase's RLS integration, you get automatic, database-level data isolation with minimal application code changes. Start with simple user isolation policies and expand to more complex organizational or role-based access as needed.
