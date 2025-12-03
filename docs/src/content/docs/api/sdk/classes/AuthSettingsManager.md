---
editUrl: false
next: false
prev: false
title: "AuthSettingsManager"
---

Authentication Settings Manager

Manages global authentication settings including password requirements, session timeouts,
and signup configuration.

## Example

```typescript
const authSettings = client.admin.authSettings

// Get current settings
const settings = await authSettings.get()

// Update settings
await authSettings.update({
  password_min_length: 12,
  password_require_uppercase: true,
  session_timeout_minutes: 120
})
```

## Constructors

### new AuthSettingsManager()

> **new AuthSettingsManager**(`fetch`): [`AuthSettingsManager`](/api/sdk/classes/authsettingsmanager/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |

#### Returns

[`AuthSettingsManager`](/api/sdk/classes/authsettingsmanager/)

## Methods

### get()

> **get**(): `Promise`\<[`AuthSettings`](/api/sdk/interfaces/authsettings/)\>

Get current authentication settings

Retrieves all authentication configuration settings.

#### Returns

`Promise`\<[`AuthSettings`](/api/sdk/interfaces/authsettings/)\>

Promise resolving to AuthSettings

#### Example

```typescript
const settings = await client.admin.authSettings.get()

console.log('Password min length:', settings.password_min_length)
console.log('Signup enabled:', settings.enable_signup)
console.log('Session timeout:', settings.session_timeout_minutes, 'minutes')
```

***

### update()

> **update**(`request`): `Promise`\<[`UpdateAuthSettingsResponse`](/api/sdk/interfaces/updateauthsettingsresponse/)\>

Update authentication settings

Updates one or more authentication settings. All fields are optional - only provided
fields will be updated.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `request` | [`UpdateAuthSettingsRequest`](/api/sdk/interfaces/updateauthsettingsrequest/) | Settings to update |

#### Returns

`Promise`\<[`UpdateAuthSettingsResponse`](/api/sdk/interfaces/updateauthsettingsresponse/)\>

Promise resolving to UpdateAuthSettingsResponse

#### Examples

```typescript
// Strengthen password requirements
await client.admin.authSettings.update({
  password_min_length: 16,
  password_require_uppercase: true,
  password_require_lowercase: true,
  password_require_number: true,
  password_require_special: true
})
```

```typescript
// Extend session timeout
await client.admin.authSettings.update({
  session_timeout_minutes: 240,
  max_sessions_per_user: 10
})
```

```typescript
// Disable email verification during development
await client.admin.authSettings.update({
  require_email_verification: false
})
```
