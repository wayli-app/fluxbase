---
editUrl: false
next: false
prev: false
title: "ImpersonationManager"
---

Impersonation Manager

Manages user impersonation for debugging, testing RLS policies, and customer support.
Allows admins to view data as different users, anonymous visitors, or with service role permissions.

All impersonation sessions are logged in the audit trail for security and compliance.

## Example

```typescript
const impersonation = client.admin.impersonation

// Impersonate a specific user
const { session, access_token } = await impersonation.impersonateUser({
  target_user_id: 'user-uuid',
  reason: 'Support ticket #1234'
})

// Impersonate anonymous user
await impersonation.impersonateAnon({
  reason: 'Testing public data access'
})

// Impersonate with service role
await impersonation.impersonateService({
  reason: 'Administrative query'
})

// Stop impersonation
await impersonation.stop()
```

## Constructors

### new ImpersonationManager()

> **new ImpersonationManager**(`fetch`): [`ImpersonationManager`](/api/sdk/classes/impersonationmanager/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |

#### Returns

[`ImpersonationManager`](/api/sdk/classes/impersonationmanager/)

## Methods

### getCurrent()

> **getCurrent**(): `Promise`\<[`GetImpersonationResponse`](/api/sdk/interfaces/getimpersonationresponse/)\>

Get current impersonation session

Retrieves information about the active impersonation session, if any.

#### Returns

`Promise`\<[`GetImpersonationResponse`](/api/sdk/interfaces/getimpersonationresponse/)\>

Promise resolving to current impersonation session or null

#### Example

```typescript
const current = await client.admin.impersonation.getCurrent()

if (current.session) {
  console.log('Currently impersonating:', current.target_user?.email)
  console.log('Reason:', current.session.reason)
  console.log('Started:', current.session.started_at)
} else {
  console.log('No active impersonation')
}
```

***

### impersonateAnon()

> **impersonateAnon**(`request`): `Promise`\<[`StartImpersonationResponse`](/api/sdk/interfaces/startimpersonationresponse/)\>

Impersonate anonymous user

Start an impersonation session as an unauthenticated user. This allows you to see
what data is publicly accessible and test RLS policies for anonymous access.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `request` | [`ImpersonateAnonRequest`](/api/sdk/interfaces/impersonateanonrequest/) | Impersonation request with reason |

#### Returns

`Promise`\<[`StartImpersonationResponse`](/api/sdk/interfaces/startimpersonationresponse/)\>

Promise resolving to impersonation session with access token

#### Example

```typescript
await client.admin.impersonation.impersonateAnon({
  reason: 'Testing public data access for blog posts'
})

// Now all queries will use anonymous permissions
const publicPosts = await client.from('posts').select('*')
console.log('Public posts:', publicPosts.length)
```

***

### impersonateService()

> **impersonateService**(`request`): `Promise`\<[`StartImpersonationResponse`](/api/sdk/interfaces/startimpersonationresponse/)\>

Impersonate with service role

Start an impersonation session with service-level permissions. This provides elevated
access that may bypass RLS policies, useful for administrative operations.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `request` | [`ImpersonateServiceRequest`](/api/sdk/interfaces/impersonateservicerequest/) | Impersonation request with reason |

#### Returns

`Promise`\<[`StartImpersonationResponse`](/api/sdk/interfaces/startimpersonationresponse/)\>

Promise resolving to impersonation session with access token

#### Example

```typescript
await client.admin.impersonation.impersonateService({
  reason: 'Administrative data cleanup'
})

// Now all queries will use service role permissions
const allRecords = await client.from('sensitive_data').select('*')
console.log('All records:', allRecords.length)
```

***

### impersonateUser()

> **impersonateUser**(`request`): `Promise`\<[`StartImpersonationResponse`](/api/sdk/interfaces/startimpersonationresponse/)\>

Impersonate a specific user

Start an impersonation session as a specific user. This allows you to see data
exactly as that user would see it, respecting all RLS policies and permissions.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `request` | [`ImpersonateUserRequest`](/api/sdk/interfaces/impersonateuserrequest/) | Impersonation request with target user ID and reason |

#### Returns

`Promise`\<[`StartImpersonationResponse`](/api/sdk/interfaces/startimpersonationresponse/)\>

Promise resolving to impersonation session with access token

#### Example

```typescript
const result = await client.admin.impersonation.impersonateUser({
  target_user_id: 'user-123',
  reason: 'Support ticket #5678 - user reports missing data'
})

console.log('Impersonating:', result.target_user.email)
console.log('Session ID:', result.session.id)

// Use the access token for subsequent requests
// (typically handled automatically by the SDK)
```

***

### listSessions()

> **listSessions**(`options`): `Promise`\<[`ListImpersonationSessionsResponse`](/api/sdk/interfaces/listimpersonationsessionsresponse/)\>

List impersonation sessions (audit trail)

Retrieves a list of impersonation sessions for audit and compliance purposes.
Can be filtered by admin user, target user, type, and active status.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `options` | [`ListImpersonationSessionsOptions`](/api/sdk/interfaces/listimpersonationsessionsoptions/) | Filter and pagination options |

#### Returns

`Promise`\<[`ListImpersonationSessionsResponse`](/api/sdk/interfaces/listimpersonationsessionsresponse/)\>

Promise resolving to list of impersonation sessions

#### Examples

```typescript
// List all sessions
const { sessions, total } = await client.admin.impersonation.listSessions()
console.log(`Total sessions: ${total}`)

// List active sessions only
const active = await client.admin.impersonation.listSessions({
  is_active: true
})
console.log('Active sessions:', active.sessions.length)

// List sessions for a specific admin
const adminSessions = await client.admin.impersonation.listSessions({
  admin_user_id: 'admin-uuid',
  limit: 50
})

// List user impersonation sessions only
const userSessions = await client.admin.impersonation.listSessions({
  impersonation_type: 'user',
  offset: 0,
  limit: 100
})
```

```typescript
// Audit trail: Find who impersonated a specific user
const userHistory = await client.admin.impersonation.listSessions({
  target_user_id: 'user-uuid'
})

userHistory.sessions.forEach(session => {
  console.log(`Admin ${session.admin_user_id} impersonated user`)
  console.log(`Reason: ${session.reason}`)
  console.log(`Duration: ${session.started_at} - ${session.ended_at}`)
})
```

***

### stop()

> **stop**(): `Promise`\<[`StopImpersonationResponse`](/api/sdk/interfaces/stopimpersonationresponse/)\>

Stop impersonation

Ends the current impersonation session and returns to admin context.
The session is marked as ended in the audit trail.

#### Returns

`Promise`\<[`StopImpersonationResponse`](/api/sdk/interfaces/stopimpersonationresponse/)\>

Promise resolving to stop confirmation

#### Example

```typescript
await client.admin.impersonation.stop()
console.log('Impersonation ended')

// Subsequent queries will use admin permissions
```
