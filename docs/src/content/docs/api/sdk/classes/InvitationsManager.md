---
editUrl: false
next: false
prev: false
title: "InvitationsManager"
---

Invitations management client

Provides methods for creating and managing user invitations.
Invitations allow admins to invite new users to join the dashboard.

## Example

```typescript
const client = createClient({ url: 'http://localhost:8080' })
await client.admin.login({ email: 'admin@example.com', password: 'password' })

// Create an invitation
const invitation = await client.management.invitations.create({
  email: 'newuser@example.com',
  role: 'dashboard_user'
})

console.log('Invite link:', invitation.invite_link)
```

## Constructors

### new InvitationsManager()

> **new InvitationsManager**(`fetch`): [`InvitationsManager`](/api/sdk/classes/invitationsmanager/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |

#### Returns

[`InvitationsManager`](/api/sdk/classes/invitationsmanager/)

## Methods

### accept()

> **accept**(`token`, `request`): `Promise`\<[`AcceptInvitationResponse`](/api/sdk/interfaces/acceptinvitationresponse/)\>

Accept an invitation and create a new user (public endpoint)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `token` | `string` | Invitation token |
| `request` | [`AcceptInvitationRequest`](/api/sdk/interfaces/acceptinvitationrequest/) | User details (password and name) |

#### Returns

`Promise`\<[`AcceptInvitationResponse`](/api/sdk/interfaces/acceptinvitationresponse/)\>

Created user with authentication tokens

#### Example

```typescript
const response = await client.management.invitations.accept('invitation-token', {
  password: 'SecurePassword123!',
  name: 'John Doe'
})

// Store tokens
localStorage.setItem('access_token', response.access_token)
console.log('Welcome:', response.user.name)
```

***

### create()

> **create**(`request`): `Promise`\<[`CreateInvitationResponse`](/api/sdk/interfaces/createinvitationresponse/)\>

Create a new invitation (admin only)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `request` | [`CreateInvitationRequest`](/api/sdk/interfaces/createinvitationrequest/) | Invitation details |

#### Returns

`Promise`\<[`CreateInvitationResponse`](/api/sdk/interfaces/createinvitationresponse/)\>

Created invitation with invite link

#### Example

```typescript
const invitation = await client.management.invitations.create({
  email: 'newuser@example.com',
  role: 'dashboard_user',
  expiry_duration: 604800 // 7 days in seconds
})

// Share the invite link
console.log('Send this link to the user:', invitation.invite_link)
```

***

### list()

> **list**(`options`): `Promise`\<[`ListInvitationsResponse`](/api/sdk/interfaces/listinvitationsresponse/)\>

List all invitations (admin only)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `options` | [`ListInvitationsOptions`](/api/sdk/interfaces/listinvitationsoptions/) | Filter options |

#### Returns

`Promise`\<[`ListInvitationsResponse`](/api/sdk/interfaces/listinvitationsresponse/)\>

List of invitations

#### Example

```typescript
// List pending invitations only
const { invitations } = await client.management.invitations.list({
  include_accepted: false,
  include_expired: false
})

// List all invitations including accepted and expired
const all = await client.management.invitations.list({
  include_accepted: true,
  include_expired: true
})
```

***

### revoke()

> **revoke**(`token`): `Promise`\<[`RevokeInvitationResponse`](/api/sdk/interfaces/revokeinvitationresponse/)\>

Revoke an invitation (admin only)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `token` | `string` | Invitation token |

#### Returns

`Promise`\<[`RevokeInvitationResponse`](/api/sdk/interfaces/revokeinvitationresponse/)\>

Revocation confirmation

#### Example

```typescript
await client.management.invitations.revoke('invitation-token')
console.log('Invitation revoked')
```

***

### validate()

> **validate**(`token`): `Promise`\<[`ValidateInvitationResponse`](/api/sdk/interfaces/validateinvitationresponse/)\>

Validate an invitation token (public endpoint)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `token` | `string` | Invitation token |

#### Returns

`Promise`\<[`ValidateInvitationResponse`](/api/sdk/interfaces/validateinvitationresponse/)\>

Validation result with invitation details

#### Example

```typescript
const result = await client.management.invitations.validate('invitation-token')

if (result.valid) {
  console.log('Valid invitation for:', result.invitation?.email)
} else {
  console.error('Invalid:', result.error)
}
```
