---
title: "User Impersonation SDK"
---

The Impersonation SDK allows administrators to view data as different users, anonymous visitors, or with service-level permissions. This is invaluable for debugging issues, testing RLS policies, and providing customer support.

## Overview

User impersonation enables you to:

- Debug user-reported issues by seeing data exactly as they see it
- Test Row Level Security (RLS) policies with different user roles
- Verify anonymous access to public data
- Perform administrative operations with service-level permissions
- Maintain a complete audit trail of all impersonation sessions

All impersonation sessions are logged for security and compliance.

## Basic Usage

```typescript
import { createClient } from '@fluxbase/sdk'

const client = createClient(
  'http://localhost:8080',
  'your-service-role-key'
)

// Login as admin first
await client.admin.login({
  email: 'admin@example.com',
  password: 'password'
})

// Access impersonation manager
const impersonation = client.admin.impersonation
```

## Impersonation Types

### 1. User Impersonation

Impersonate a specific user to see data exactly as they would see it.

```typescript
const { session, target_user, access_token } = await impersonation.impersonateUser({
  target_user_id: 'user-uuid',
  reason: 'Support ticket #1234 - user reports missing data'
})

console.log('Impersonating:', target_user.email)
console.log('Session ID:', session.id)
```

**Use cases:**
- Debugging user-reported data issues
- Verifying RLS policies for specific users
- Customer support investigations

### 2. Anonymous Impersonation

See data as an unauthenticated visitor would see it.

```typescript
await impersonation.impersonateAnon({
  reason: 'Testing public data access for blog posts'
})

// Now all queries will use anonymous permissions
const publicData = await client.from('posts').select('*').execute()
console.log('Public posts:', publicData.data.length)
```

**Use cases:**
- Testing public data access
- Verifying anon-level RLS policies
- Ensuring sensitive data is protected from public access

### 3. Service Role Impersonation

View data with service-level permissions that may bypass RLS policies.

```typescript
await impersonation.impersonateService({
  reason: 'Administrative data cleanup'
})

// Now all queries will use service role permissions
const allRecords = await client.from('sensitive_data').select('*').execute()
```

**Use cases:**
- Administrative queries
- Testing privileged operations
- Bypassing RLS for data management

## API Reference

### impersonateUser()

Start an impersonation session as a specific user.

```typescript
const response = await impersonation.impersonateUser({
  target_user_id: string,
  reason: string
})
```

**Parameters:**
- `target_user_id` (string): UUID of the user to impersonate
- `reason` (string): Required explanation for audit trail

**Returns:**
```typescript
{
  session: {
    id: string
    admin_user_id: string
    target_user_id: string
    impersonation_type: 'user'
    target_role: string
    reason: string
    started_at: string
    ended_at: string | null
    is_active: boolean
    ip_address: string | null
    user_agent: string | null
  },
  target_user: {
    id: string
    email: string
    role: string
  },
  access_token: string
  refresh_token: string
  expires_in: number
}
```

**Errors:**
- `User not found` - Target user doesn't exist
- `Cannot impersonate yourself` - Trying to impersonate own account
- `Unauthorized` - Not logged in as admin

### impersonateAnon()

Start an impersonation session as anonymous user.

```typescript
const response = await impersonation.impersonateAnon({
  reason: string
})
```

**Parameters:**
- `reason` (string): Required explanation for audit trail

**Returns:** Same structure as `impersonateUser()` but with `target_user: null`

### impersonateService()

Start an impersonation session with service role.

```typescript
const response = await impersonation.impersonateService({
  reason: string
})
```

**Parameters:**
- `reason` (string): Required explanation for audit trail

**Returns:** Same structure as `impersonateUser()` but with `target_user: null`

### stop()

End the current impersonation session.

```typescript
const response = await impersonation.stop()
```

**Returns:**
```typescript
{
  success: boolean
  message: string
}
```

### getCurrent()

Get information about the active impersonation session.

```typescript
const current = await impersonation.getCurrent()
```

**Returns:**
```typescript
{
  session: ImpersonationSession | null
  target_user: ImpersonationTargetUser | null
}
```

**Example:**
```typescript
const { session, target_user } = await impersonation.getCurrent()

if (session) {
  console.log('Currently impersonating:', target_user?.email)
  console.log('Reason:', session.reason)
  console.log('Started:', session.started_at)
} else {
  console.log('No active impersonation')
}
```

### listSessions()

List impersonation sessions for audit and compliance.

```typescript
const sessions = await impersonation.listSessions(options?)
```

**Parameters:**
```typescript
{
  limit?: number              // Max sessions to return
  offset?: number             // Pagination offset
  admin_user_id?: string      // Filter by admin
  target_user_id?: string     // Filter by impersonated user
  impersonation_type?: 'user' | 'anon' | 'service'
  is_active?: boolean         // Filter by active status
}
```

**Returns:**
```typescript
{
  sessions: ImpersonationSession[]
  total: number
}
```

## Common Use Cases

### 1. Debugging User Issues

```typescript
async function debugUserIssue(userId: string, ticketId: string) {
  // Start impersonation
  const { target_user } = await client.admin.impersonation.impersonateUser({
    target_user_id: userId,
    reason: `Support ticket #${ticketId} - investigating data access issue`
  })

  console.log(`Impersonating: ${target_user.email}`)

  // Query data as the user would see it
  const { data, error } = await client
    .from('user_documents')
    .select('*')
    .execute()

  console.log('User can see documents:', data?.length)

  // Stop impersonation
  await client.admin.impersonation.stop()
}
```

### 2. Testing RLS Policies

```typescript
async function testRLSPolicy() {
  // Test as regular user
  await client.admin.impersonation.impersonateUser({
    target_user_id: 'regular-user-id',
    reason: 'Testing RLS policy for regular users'
  })

  const regularUserData = await client.from('posts').select('*').execute()
  console.log('Regular user sees:', regularUserData.data?.length)

  // Stop and test as admin
  await client.admin.impersonation.stop()

  // Test as service role
  await client.admin.impersonation.impersonateService({
    reason: 'Testing RLS policy bypass with service role'
  })

  const serviceData = await client.from('posts').select('*').execute()
  console.log('Service role sees:', serviceData.data?.length)

  // Cleanup
  await client.admin.impersonation.stop()
}
```

### 3. Verifying Public Access

```typescript
async function verifyPublicAccess() {
  // Impersonate anonymous user
  await client.admin.impersonation.impersonateAnon({
    reason: 'Verifying public blog posts are accessible'
  })

  // Query as anonymous user
  const { data } = await client
    .from('blog_posts')
    .select('*')
    .eq('status', 'published')
    .execute()

  console.log('Public can see posts:', data?.length)

  // Verify private posts are hidden
  const privateQuery = await client
    .from('blog_posts')
    .select('*')
    .eq('status', 'draft')
    .execute()

  if (privateQuery.data?.length === 0) {
    console.log('✓ Private posts correctly hidden from public')
  } else {
    console.error('✗ Security issue: Private posts visible to public!')
  }

  await client.admin.impersonation.stop()
}
```

### 4. Audit Trail Review

```typescript
async function reviewImpersonationActivity() {
  // Get all sessions from last 7 days
  const { sessions, total } = await client.admin.impersonation.listSessions({
    limit: 100,
    offset: 0
  })

  console.log(`Total sessions: ${total}`)

  sessions.forEach(session => {
    const duration = session.ended_at
      ? new Date(session.ended_at).getTime() - new Date(session.started_at).getTime()
      : 'ongoing'

    console.log(`
      Admin: ${session.admin_user_id}
      Type: ${session.impersonation_type}
      Reason: ${session.reason}
      Duration: ${duration}ms
      Started: ${session.started_at}
    `)
  })
}
```

### 5. Customer Support Workflow

```typescript
async function supportWorkflow(userEmail: string, issue: string) {
  try {
    // Find user
    const { users } = await client.admin.listUsers({
      search: userEmail,
      limit: 1
    })

    if (users.length === 0) {
      throw new Error(`User ${userEmail} not found`)
    }

    const user = users[0]

    // Start impersonation
    console.log(`Starting impersonation of ${user.email}`)
    await client.admin.impersonation.impersonateUser({
      target_user_id: user.id,
      reason: `Customer support: ${issue}`
    })

    // Investigate the issue
    // ... perform debugging queries ...

    // Take screenshots or gather data
    const userView = await client.from('dashboard').select('*').execute()
    console.log('User dashboard data:', userView.data)

  } catch (error) {
    console.error('Support workflow error:', error)
  } finally {
    // Always stop impersonation
    await client.admin.impersonation.stop()
    console.log('Impersonation ended')
  }
}
```

### 6. Multi-Scenario Testing

```typescript
async function testMultipleScenarios() {
  const scenarios = [
    {
      type: 'user' as const,
      userId: 'premium-user-id',
      reason: 'Testing premium user features'
    },
    {
      type: 'user' as const,
      userId: 'free-user-id',
      reason: 'Testing free user limitations'
    },
    {
      type: 'anon' as const,
      reason: 'Testing public access'
    }
  ]

  for (const scenario of scenarios) {
    console.log(`\n--- Testing: ${scenario.reason} ---`)

    if (scenario.type === 'user') {
      await client.admin.impersonation.impersonateUser({
        target_user_id: scenario.userId!,
        reason: scenario.reason
      })
    } else {
      await client.admin.impersonation.impersonateAnon({
        reason: scenario.reason
      })
    }

    // Run tests
    const features = await client.from('available_features').select('*').execute()
    console.log('Available features:', features.data?.map(f => f.name))

    // Stop before next scenario
    await client.admin.impersonation.stop()
  }
}
```

### 7. Finding Who Impersonated a User

```typescript
async function findImpersonationHistory(userId: string) {
  const { sessions } = await client.admin.impersonation.listSessions({
    target_user_id: userId
  })

  console.log(`Impersonation history for user ${userId}:`)

  sessions.forEach(session => {
    console.log(`
      Admin: ${session.admin_user_id}
      Reason: ${session.reason}
      Started: ${session.started_at}
      Ended: ${session.ended_at || 'Still active'}
      IP: ${session.ip_address}
    `)
  })
}
```

## Error Handling

```typescript
try {
  await client.admin.impersonation.impersonateUser({
    target_user_id: 'user-id',
    reason: 'Support investigation'
  })

  // Perform operations

} catch (error) {
  if (error.message.includes('not found')) {
    console.error('User does not exist')
  } else if (error.message.includes('yourself')) {
    console.error('Cannot impersonate your own account')
  } else if (error.message.includes('Unauthorized')) {
    console.error('Must be logged in as admin')
  } else {
    console.error('Impersonation error:', error)
  }
} finally {
  // Always stop impersonation in cleanup
  try {
    await client.admin.impersonation.stop()
  } catch {
    // Ignore if no active session
  }
}
```

## Security & Best Practices

### 1. Always Provide Clear Reasons

```typescript
// ✓ Good - Clear and specific
await impersonation.impersonateUser({
  target_user_id: 'user-123',
  reason: 'Support ticket #5678 - user reports missing invoices'
})

// ✗ Bad - Vague reason
await impersonation.impersonateUser({
  target_user_id: 'user-123',
  reason: 'testing'
})
```

### 2. Stop Impersonation When Done

Always stop impersonation sessions to:
- Clear the audit trail properly
- Avoid confusion
- Prevent accidental data modifications

```typescript
// Use try/finally to ensure cleanup
try {
  await impersonation.impersonateUser({
    target_user_id: 'user-123',
    reason: 'Debugging data access'
  })

  // Do work...

} finally {
  await impersonation.stop()
}
```

### 3. Review Audit Logs Regularly

```typescript
// Monitor impersonation usage
async function monitorUsage() {
  const { sessions } = await client.admin.impersonation.listSessions({
    is_active: false,
    limit: 50
  })

  // Check for suspicious patterns
  const byAdmin = new Map()
  sessions.forEach(s => {
    byAdmin.set(s.admin_user_id, (byAdmin.get(s.admin_user_id) || 0) + 1)
  })

  byAdmin.forEach((count, admin) => {
    if (count > 20) {
      console.warn(`Admin ${admin} has ${count} impersonation sessions`)
    }
  })
}
```

### 4. Limit Impersonation Duration

```typescript
async function timedImpersonation(userId: string, maxMinutes: number = 15) {
  await client.admin.impersonation.impersonateUser({
    target_user_id: userId,
    reason: 'Timed support session'
  })

  // Auto-stop after timeout
  setTimeout(async () => {
    await client.admin.impersonation.stop()
    console.log('Impersonation auto-ended after timeout')
  }, maxMinutes * 60 * 1000)
}
```

### 5. Prevent Self-Impersonation

The SDK automatically prevents admins from impersonating themselves:

```typescript
// This will throw an error
await client.admin.impersonation.impersonateUser({
  target_user_id: currentAdmin.id, // Your own ID
  reason: 'Testing'
})
// Error: Cannot impersonate yourself
```

## Integration with RLS

When impersonating, all database queries respect Row Level Security policies:

```sql
-- RLS policy example
CREATE POLICY "Users can only see their own data"
ON user_documents
FOR SELECT
USING (user_id = current_setting('app.user_id')::uuid);
```

When you impersonate a user, the `app.user_id` session variable is set to their ID, so the RLS policy works correctly.

## Type Definitions

```typescript
interface ImpersonationSession {
  id: string
  admin_user_id: string
  target_user_id: string | null
  impersonation_type: 'user' | 'anon' | 'service'
  target_role: string
  reason: string
  started_at: string
  ended_at: string | null
  is_active: boolean
  ip_address: string | null
  user_agent: string | null
}

interface ImpersonationTargetUser {
  id: string
  email: string
  role: string
}

interface StartImpersonationResponse {
  session: ImpersonationSession
  target_user: ImpersonationTargetUser | null
  access_token: string
  refresh_token: string
  expires_in: number
}

interface StopImpersonationResponse {
  success: boolean
  message: string
}

interface ListImpersonationSessionsResponse {
  sessions: ImpersonationSession[]
  total: number
}
```

## Related Resources

- [User Management SDK](/docs/sdk/admin#user-management) - Manage users
- [Authentication Guide](/docs/guides/authentication) - Learn about authentication
- [Row Level Security](/docs/guides/row-level-security) - Configure RLS policies
- [Admin Guide](/docs/guides/admin/user-impersonation) - Dashboard impersonation guide
