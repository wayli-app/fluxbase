---
editUrl: false
next: false
prev: false
title: "FluxbaseAdmin"
---

Admin client for managing Fluxbase instance

## Constructors

### new FluxbaseAdmin()

> **new FluxbaseAdmin**(`fetch`): [`FluxbaseAdmin`](/api/sdk/classes/fluxbaseadmin/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |

#### Returns

[`FluxbaseAdmin`](/api/sdk/classes/fluxbaseadmin/)

## Properties

| Property | Modifier | Type | Description |
| ------ | ------ | ------ | ------ |
| `ai` | `public` | [`FluxbaseAdminAI`](/api/sdk/classes/fluxbaseadminai/) | AI manager for chatbot and provider management (create, update, delete, sync) |
| `ddl` | `public` | [`DDLManager`](/api/sdk/classes/ddlmanager/) | DDL manager for database schema and table operations |
| `emailTemplates` | `public` | [`EmailTemplateManager`](/api/sdk/classes/emailtemplatemanager/) | Email template manager for customizing authentication and notification emails |
| `functions` | `public` | [`FluxbaseAdminFunctions`](/api/sdk/classes/fluxbaseadminfunctions/) | Functions manager for edge function management (create, update, delete, sync) |
| `impersonation` | `public` | [`ImpersonationManager`](/api/sdk/classes/impersonationmanager/) | Impersonation manager for user impersonation and audit trail |
| `jobs` | `public` | [`FluxbaseAdminJobs`](/api/sdk/classes/fluxbaseadminjobs/) | Jobs manager for background job management (create, update, delete, sync, monitoring) |
| `management` | `public` | [`FluxbaseManagement`](/api/sdk/classes/fluxbasemanagement/) | Management namespace for API keys, webhooks, and invitations |
| `migrations` | `public` | [`FluxbaseAdminMigrations`](/api/sdk/classes/fluxbaseadminmigrations/) | Migrations manager for database migration operations (create, apply, rollback, sync) |
| `oauth` | `public` | [`FluxbaseOAuth`](/api/sdk/classes/fluxbaseoauth/) | OAuth configuration manager for provider and auth settings |
| `rpc` | `public` | [`FluxbaseAdminRPC`](/api/sdk/classes/fluxbaseadminrpc/) | RPC manager for procedure management (create, update, delete, sync, execution monitoring) |
| `settings` | `public` | [`FluxbaseSettings`](/api/sdk/classes/fluxbasesettings/) | Settings manager for system and application settings |

## Methods

### clearToken()

> **clearToken**(): `void`

Clear admin token

#### Returns

`void`

***

### deleteUser()

> **deleteUser**(`userId`, `type`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`DeleteUserResponse`](/api/sdk/interfaces/deleteuserresponse/)\>\>

Delete a user

Permanently deletes a user and all associated data

#### Parameters

| Parameter | Type | Default value | Description |
| ------ | ------ | ------ | ------ |
| `userId` | `string` | `undefined` | User ID to delete |
| `type` | `"app"` \| `"dashboard"` | `"app"` | User type ('app' or 'dashboard') |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`DeleteUserResponse`](/api/sdk/interfaces/deleteuserresponse/)\>\>

Deletion confirmation

#### Example

```typescript
await admin.deleteUser('user-uuid');
console.log('User deleted');
```

***

### getSetupStatus()

> **getSetupStatus**(): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`AdminSetupStatusResponse`](/api/sdk/interfaces/adminsetupstatusresponse/)\>\>

Check if initial admin setup is needed

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`AdminSetupStatusResponse`](/api/sdk/interfaces/adminsetupstatusresponse/)\>\>

Setup status indicating if initial setup is required

#### Example

```typescript
const status = await admin.getSetupStatus();
if (status.needs_setup) {
  console.log('Initial setup required');
}
```

***

### getToken()

> **getToken**(): `null` \| `string`

Get current admin token

#### Returns

`null` \| `string`

***

### getUserById()

> **getUserById**(`userId`, `type`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`EnrichedUser`](/api/sdk/interfaces/enricheduser/)\>\>

Get a user by ID

Fetch a single user's details by their user ID

#### Parameters

| Parameter | Type | Default value | Description |
| ------ | ------ | ------ | ------ |
| `userId` | `string` | `undefined` | User ID to fetch |
| `type` | `"app"` \| `"dashboard"` | `"app"` | User type ('app' or 'dashboard') |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`EnrichedUser`](/api/sdk/interfaces/enricheduser/)\>\>

User details with metadata

#### Example

```typescript
// Get an app user
const user = await admin.getUserById('user-123');

// Get a dashboard user
const dashboardUser = await admin.getUserById('admin-456', 'dashboard');
console.log('User email:', dashboardUser.email);
console.log('Last login:', dashboardUser.last_login_at);
```

***

### inviteUser()

> **inviteUser**(`request`, `type`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`InviteUserResponse`](/api/sdk/interfaces/inviteuserresponse/)\>\>

Invite a new user

Creates a new user and optionally sends an invitation email

#### Parameters

| Parameter | Type | Default value | Description |
| ------ | ------ | ------ | ------ |
| `request` | [`InviteUserRequest`](/api/sdk/interfaces/inviteuserrequest/) | `undefined` | User invitation details |
| `type` | `"app"` \| `"dashboard"` | `"app"` | User type ('app' or 'dashboard') |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`InviteUserResponse`](/api/sdk/interfaces/inviteuserresponse/)\>\>

Created user and invitation details

#### Example

```typescript
const response = await admin.inviteUser({
  email: 'newuser@example.com',
  role: 'user',
  send_email: true
});

console.log('User invited:', response.user.email);
console.log('Invitation link:', response.invitation_link);
```

***

### listUsers()

> **listUsers**(`options`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`ListUsersResponse`](/api/sdk/interfaces/listusersresponse/)\>\>

List all users

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `options` | [`ListUsersOptions`](/api/sdk/interfaces/listusersoptions/) | Filter and pagination options |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`ListUsersResponse`](/api/sdk/interfaces/listusersresponse/)\>\>

List of users with metadata

#### Example

```typescript
// List all users
const { users, total } = await admin.listUsers();

// List with filters
const result = await admin.listUsers({
  exclude_admins: true,
  search: 'john',
  limit: 50,
  type: 'app'
});
```

***

### login()

> **login**(`request`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`AdminAuthResponse`](/api/sdk/interfaces/adminauthresponse/)\>\>

Admin login

Authenticate as an admin user

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `request` | [`AdminLoginRequest`](/api/sdk/interfaces/adminloginrequest/) | Login request containing email and password |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`AdminAuthResponse`](/api/sdk/interfaces/adminauthresponse/)\>\>

Authentication response with tokens

#### Example

```typescript
const response = await admin.login({
  email: 'admin@example.com',
  password: 'password123'
});

// Token is automatically set in the client
console.log('Logged in as:', response.user.email);
```

***

### logout()

> **logout**(): `Promise`\<[`VoidResponse`](/api/sdk/type-aliases/voidresponse/)\>

Admin logout

Invalidates the current admin session

#### Returns

`Promise`\<[`VoidResponse`](/api/sdk/type-aliases/voidresponse/)\>

#### Example

```typescript
await admin.logout();
localStorage.removeItem('admin_token');
```

***

### me()

> **me**(): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`AdminMeResponse`](/api/sdk/interfaces/adminmeresponse/)\>\>

Get current admin user information

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`AdminMeResponse`](/api/sdk/interfaces/adminmeresponse/)\>\>

Current admin user details

#### Example

```typescript
const { user } = await admin.me();
console.log('Logged in as:', user.email);
console.log('Role:', user.role);
```

***

### refreshToken()

> **refreshToken**(`request`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`AdminRefreshResponse`](/api/sdk/interfaces/adminrefreshresponse/)\>\>

Refresh admin access token

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `request` | [`AdminRefreshRequest`](/api/sdk/interfaces/adminrefreshrequest/) | Refresh request containing the refresh token |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`AdminRefreshResponse`](/api/sdk/interfaces/adminrefreshresponse/)\>\>

New access and refresh tokens

#### Example

```typescript
const refreshToken = localStorage.getItem('admin_refresh_token');
const response = await admin.refreshToken({ refresh_token: refreshToken });

// Update stored tokens
localStorage.setItem('admin_token', response.access_token);
localStorage.setItem('admin_refresh_token', response.refresh_token);
```

***

### resetUserPassword()

> **resetUserPassword**(`userId`, `type`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`ResetUserPasswordResponse`](/api/sdk/interfaces/resetuserpasswordresponse/)\>\>

Reset user password

Generates a new password for the user and optionally sends it via email

#### Parameters

| Parameter | Type | Default value | Description |
| ------ | ------ | ------ | ------ |
| `userId` | `string` | `undefined` | User ID |
| `type` | `"app"` \| `"dashboard"` | `"app"` | User type ('app' or 'dashboard') |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`ResetUserPasswordResponse`](/api/sdk/interfaces/resetuserpasswordresponse/)\>\>

Reset confirmation message

#### Example

```typescript
const response = await admin.resetUserPassword('user-uuid');
console.log(response.message);
```

***

### setToken()

> **setToken**(`token`): `void`

Set admin authentication token

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `token` | `string` |

#### Returns

`void`

***

### setup()

> **setup**(`request`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`AdminAuthResponse`](/api/sdk/interfaces/adminauthresponse/)\>\>

Perform initial admin setup

Creates the first admin user and completes initial setup.
This endpoint can only be called once.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `request` | [`AdminSetupRequest`](/api/sdk/interfaces/adminsetuprequest/) | Setup request containing email, password, and name |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`AdminAuthResponse`](/api/sdk/interfaces/adminauthresponse/)\>\>

Authentication response with tokens

#### Example

```typescript
const response = await admin.setup({
  email: 'admin@example.com',
  password: 'SecurePassword123!',
  name: 'Admin User'
});

// Store tokens
localStorage.setItem('admin_token', response.access_token);
```

***

### updateUserRole()

> **updateUserRole**(`userId`, `role`, `type`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`EnrichedUser`](/api/sdk/interfaces/enricheduser/)\>\>

Update user role

Changes a user's role

#### Parameters

| Parameter | Type | Default value | Description |
| ------ | ------ | ------ | ------ |
| `userId` | `string` | `undefined` | User ID |
| `role` | `string` | `undefined` | New role |
| `type` | `"app"` \| `"dashboard"` | `"app"` | User type ('app' or 'dashboard') |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`EnrichedUser`](/api/sdk/interfaces/enricheduser/)\>\>

Updated user

#### Example

```typescript
const user = await admin.updateUserRole('user-uuid', 'admin');
console.log('User role updated:', user.role);
```
