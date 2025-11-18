# Row Level Security (RLS) Guide

## Overview

Fluxbase supports PostgreSQL Row Level Security (RLS) to enforce fine-grained access control at the database level. RLS ensures that users can only access data they're authorized to see, making it ideal for multi-tenant applications.

## How It Works

1. **Session Variables**: When a request is authenticated, the RLS middleware sets PostgreSQL session variables (`app.user_id` and `app.role`)
2. **RLS Policies**: PostgreSQL policies use these session variables to filter rows automatically
3. **Transparent Enforcement**: The application code doesn't need to worry about access control - PostgreSQL handles it

## Configuration

RLS is always enabled in Fluxbase as a core security feature and cannot be disabled. This ensures multi-tenant data isolation and defense-in-depth security.

For operations that need to bypass RLS (such as administrative tasks), use service keys which have elevated privileges.

## Helper Functions

Fluxbase provides several helper functions for writing RLS policies:

### `auth.uid()` -> UUID

Returns the authenticated user's ID, or NULL for anonymous users. This is a Supabase-compatible alias for `auth.current_user_id()`.

### `auth.jwt()` -> JSONB

Returns JWT claims as JSONB, including `user_metadata` and `app_metadata`. This is a Supabase-compatible function. Use the `->` operator to extract JSONB values or `->>` to extract text values.

Example usage:

```sql
-- Extract a custom role from app_metadata
(auth.jwt() -> 'app_metadata' ->> 'custom_role')

-- Check if user has a specific permission
(auth.jwt() -> 'app_metadata' -> 'permissions' ? 'can_delete')
```

### `auth.role()` -> TEXT

Returns the current user's role (`anon`, `authenticated`, `admin`, etc.). This is a Supabase-compatible alias for `auth.current_user_role()`.

### `auth.current_user_id()` -> UUID

Returns the authenticated user's ID, or NULL for anonymous users.

### `auth.current_user_role()` -> TEXT

Returns the current user's role (`anon`, `authenticated`, `admin`, etc.)

### `auth.is_authenticated()` -> BOOLEAN

Returns TRUE if a user is logged in.

### `auth.is_admin()` -> BOOLEAN

Returns TRUE if the current user is an admin.

### `auth.enable_rls(table_name, schema_name)`

Helper function to enable RLS on a table.

### `auth.disable_rls(table_name, schema_name)`

Helper function to disable RLS on a table.

### `storage.foldername(name)` -> TEXT[]

Supabase-compatible function that extracts folder path components from a storage object name/path. Returns an array of folder names. Use `[1]` to get the first folder, `[2]` for the second, etc.

**Note**: The `storage.objects` table has both `path` (Fluxbase) and `name` (Supabase) columns that are synchronized. You can use either in your policies.

Example usage:

```sql
-- Supabase-style: using 'name' column
CREATE POLICY "Users upload own trip images" ON storage.objects
    FOR INSERT
    WITH CHECK (
        bucket_id = 'trip-images'
        AND (auth.uid())::text = (storage.foldername(name))[1]
    );

-- Fluxbase-style: using 'path' column (equivalent)
CREATE POLICY user_folder_upload ON storage.objects
    FOR INSERT
    WITH CHECK (
        bucket_id = 'avatars'
        AND (storage.foldername(path))[1] = auth.uid()::text
    );

-- Allow uploads to a specific folder
CREATE POLICY private_folder ON storage.objects
    FOR INSERT
    WITH CHECK (
        bucket_id = 'documents'
        AND (storage.foldername(name))[1] = 'private'
    );
```

## Example: Multi-Tenant Tasks Table

```sql
-- Create a multi-tenant table
CREATE TABLE public.tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id),
    title TEXT NOT NULL,
    is_public BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Enable RLS
ALTER TABLE public.tasks ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.tasks FORCE ROW LEVEL SECURITY;

-- Policy: Users can see their own tasks
CREATE POLICY tasks_select_own ON public.tasks
    FOR SELECT
    USING (user_id = auth.uid());

-- Policy: Anyone can see public tasks
CREATE POLICY tasks_select_public ON public.tasks
    FOR SELECT
    USING (is_public = TRUE);

-- Policy: Users can only insert their own tasks
CREATE POLICY tasks_insert_own ON public.tasks
    FOR INSERT
    WITH CHECK (
        auth.is_authenticated()
        AND user_id = auth.uid()
    );

-- Policy: Users can update their own tasks
CREATE POLICY tasks_update_own ON public.tasks
    FOR UPDATE
    USING (user_id = auth.uid())
    WITH CHECK (user_id = auth.uid());

-- Policy: Admins can see/update/delete all tasks
CREATE POLICY tasks_admin_all ON public.tasks
    FOR ALL
    USING (auth.is_admin());

-- Grant permissions to roles
GRANT SELECT, INSERT, UPDATE, DELETE ON public.tasks TO authenticated;
GRANT SELECT ON public.tasks TO anon;
```

## Testing RLS

### 1. Sign up a user

```bash
curl -X POST http://localhost:8080/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}'
```

### 2. Create a task (authenticated)

```bash
curl -X POST http://localhost:8080/api/v1/tables/tasks \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"My Task","user_id":"YOUR_USER_ID"}'
```

### 3. Try to access another user's task

Users will only see their own tasks automatically - no need to filter by user_id!

```bash
# This will only return the authenticated user's tasks
curl http://localhost:8080/api/v1/tables/tasks \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

## Common Patterns

### Public + Private Data

```sql
-- Anyone can read public records, only owners can read private
CREATE POLICY table_select ON public.my_table
    FOR SELECT
    USING (
        is_public = TRUE
        OR user_id = auth.uid()
    );
```

### Organization-Level Access

```sql
-- Users can access data from their organization
CREATE POLICY org_access ON public.documents
    FOR SELECT
    USING (
        organization_id IN (
            SELECT organization_id
            FROM auth.user_organizations
            WHERE user_id = auth.uid()
        )
    );
```

### Role-Based Access

```sql
-- Different permissions for different roles
CREATE POLICY role_based ON public.sensitive_data
    FOR SELECT
    USING (
        CASE auth.role()
            WHEN 'admin' THEN TRUE
            WHEN 'manager' THEN department_id = (SELECT department_id FROM auth.users WHERE id = auth.uid())
            WHEN 'user' THEN user_id = auth.uid()
            ELSE FALSE
        END
    );

-- Simple role check
CREATE POLICY admin_only ON public.admin_settings
    FOR ALL
    USING (auth.role() = 'admin');
```

### Custom Claims from JWT

```sql
-- Use custom claims from app_metadata for fine-grained access control
CREATE POLICY team_access ON public.projects
    FOR SELECT
    USING (
        -- Check if user's team (stored in app_metadata) matches project team
        team_id = (auth.jwt() -> 'app_metadata' ->> 'team_id')::UUID
        OR auth.is_admin()
    );

-- Permission-based access using app_metadata
CREATE POLICY permission_based ON public.documents
    FOR DELETE
    USING (
        -- Check if user has 'can_delete' permission in their app_metadata
        (auth.jwt() -> 'app_metadata' -> 'permissions' ? 'can_delete')
        OR auth.is_admin()
    );
```

## PostgreSQL Roles

Fluxbase creates two PostgreSQL roles:

- `anon`: For anonymous/unauthenticated requests
- `authenticated`: For authenticated users

These roles are used to grant appropriate permissions to functions and tables.

## Performance Considerations

1. **Index Foreign Keys**: Always index columns used in RLS policies (like `user_id`)
2. **Simple Policies**: Keep policies simple - complex subqueries can slow down queries
3. **Test at Scale**: Test RLS policies with realistic data volumes

## Security Best Practices

1. **Always Use FORCE ROW LEVEL SECURITY**: This ensures even table owners respect RLS
2. **Test Policies**: Verify policies work as expected with different user roles
3. **Audit Regularly**: Review RLS policies regularly for security gaps
4. **Grant Minimal Permissions**: Only grant necessary permissions to `anon` and `authenticated` roles

## Debugging

To see what user_id is being used in a query:

```sql
SELECT current_setting('app.user_id', true);
SELECT current_setting('app.role', true);
```

To test policies as a specific user (in psql):

```sql
SET app.user_id = 'user-uuid-here';
SET app.role = 'authenticated';
SELECT * FROM public.tasks;  -- Will only show that user's tasks
```

## See Also

- [PostgreSQL RLS Documentation](https://www.postgresql.org/docs/current/ddl-rowsecurity.html)
- Example migration: [010_rls_example_tasks.up.sql](../internal/database/migrations/010_rls_example_tasks.up.sql)
- RLS middleware: [internal/middleware/rls.go](../internal/middleware/rls.go)
