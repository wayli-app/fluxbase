---
editUrl: false
next: false
prev: false
title: "SettingsClient"
---

Public Settings Client

Provides read-only access to public settings for non-admin users.
Access is controlled by RLS policies on the app.settings table.

## Example

```typescript
const client = new FluxbaseClient(url, userToken)

// Get single public setting
const betaEnabled = await client.settings.get('features.beta_enabled')
console.log(betaEnabled) // { enabled: true }

// Get multiple public settings
const values = await client.settings.getMany([
  'features.beta_enabled',
  'features.dark_mode',
  'public.app_version'
])
console.log(values)
// {
//   'features.beta_enabled': { enabled: true },
//   'features.dark_mode': { enabled: false },
//   'public.app_version': '1.0.0'
// }
```

## Constructors

### Constructor

> **new SettingsClient**(`fetch`): `SettingsClient`

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |

#### Returns

`SettingsClient`

## Methods

### deleteSecret()

> **deleteSecret**(`key`): `Promise`\<`void`\>

Delete a user secret setting

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `key` | `string` | Secret key to delete |

#### Returns

`Promise`\<`void`\>

Promise<void>

#### Example

```typescript
await client.settings.deleteSecret('openai_api_key')
```

***

### deleteSetting()

> **deleteSetting**(`key`): `Promise`\<`void`\>

Delete a user setting

Removes the user's own setting, reverting to system default (if any).

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `key` | `string` | Setting key to delete |

#### Returns

`Promise`\<`void`\>

Promise<void>

#### Example

```typescript
// Remove user's theme preference, revert to system default
await client.settings.deleteSetting('theme')
```

***

### get()

> **get**(`key`): `Promise`\<`any`\>

Get a single setting's value

Returns only the value field of the setting.
Access is controlled by RLS policies - will return 403 if the user
doesn't have permission to read the setting.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `key` | `string` | Setting key (e.g., 'features.beta_enabled') |

#### Returns

`Promise`\<`any`\>

Promise resolving to the setting's value

#### Throws

Error if setting doesn't exist or user lacks permission

#### Example

```typescript
// Get public setting (any user)
const value = await client.settings.get('features.beta_enabled')
console.log(value) // { enabled: true }

// Get restricted setting (requires permission)
try {
  const secret = await client.settings.get('internal.api_key')
} catch (error) {
  console.error('Access denied:', error)
}
```

***

### getMany()

> **getMany**(`keys?`, `options?`): `Promise`\<`Record`\<`string`, `any`\>\>

Get multiple settings' values by keys, or all settings under a namespace.

Fetches multiple settings in a single request, avoiding a separate
per-key request (each of which would 404 when the key is unset).
Only returns settings the user has permission to read based on RLS policies.
Settings the user can't access — or that don't exist — are omitted from the
result (no error thrown, no existence leak).

#### Parameters

| Parameter | Type | Default value | Description |
| ------ | ------ | ------ | ------ |
| `keys` | `string`[] | `[]` | Array of setting keys to fetch (omit/empty to use prefix only) |
| `options?` | \{ `prefix?`: `string`; \} | `undefined` | Optional. `prefix` fetches every visible key under a namespace (e.g. `'wayli.'`). The prefix must end with a `.`. When both `keys` and `prefix` are given, returns the intersection. |
| `options.prefix?` | `string` | `undefined` | - |

#### Returns

`Promise`\<`Record`\<`string`, `any`\>\>

Promise resolving to object mapping keys to values

#### Example

```typescript
// Fetch specific keys
const values = await client.settings.getMany([
  'features.beta_enabled',  // public - will be returned
  'features.dark_mode',      // public - will be returned
  'internal.api_key'         // secret - will be omitted
])

// Fetch an entire namespace in one call
const wayli = await client.settings.getMany([], { prefix: 'wayli.' })
```

***

### getSecret()

> **getSecret**(`key`): `Promise`\<`SecretSettingMetadata`\>

Get metadata for a user secret setting (never returns the value)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `key` | `string` | Secret key |

#### Returns

`Promise`\<`SecretSettingMetadata`\>

Promise resolving to SecretSettingMetadata

#### Example

```typescript
const metadata = await client.settings.getSecret('openai_api_key')
console.log(metadata.key, metadata.updated_at)
// Note: The actual secret value is never returned
```

***

### getSetting()

> **getSetting**(`key`): `Promise`\<[`UserSettingWithSource`](/api/sdk/interfaces/usersettingwithsource/)\>

Get a setting with user -> system fallback

First checks for a user-specific setting, then falls back to system default.
Returns both the value and the source ("user" or "system").

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `key` | `string` | Setting key (e.g., 'theme', 'notifications.email') |

#### Returns

`Promise`\<[`UserSettingWithSource`](/api/sdk/interfaces/usersettingwithsource/)\>

Promise resolving to UserSettingWithSource with value and source

#### Throws

Error if setting doesn't exist in either user or system

#### Example

```typescript
// Get theme with fallback to system default
const { value, source } = await client.settings.getSetting('theme')
console.log(value)   // { mode: 'dark' }
console.log(source)  // 'system' (from system default)

// After user sets their own theme
const { value, source } = await client.settings.getSetting('theme')
console.log(source)  // 'user' (user's own setting)
```

***

### getSystemSetting()

> **getSystemSetting**(`key`): `Promise`\<\{ `key`: `string`; `value`: `Record`\<`string`, `unknown`\>; \}\>

Get a system-level setting (no user override)

Returns the system default for this key, ignoring any user-specific value.
Useful for reading default configurations.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `key` | `string` | Setting key |

#### Returns

`Promise`\<\{ `key`: `string`; `value`: `Record`\<`string`, `unknown`\>; \}\>

Promise resolving to the setting value

#### Throws

Error if system setting doesn't exist

#### Example

```typescript
// Get system default theme
const { value } = await client.settings.getSystemSetting('theme')
console.log('System default theme:', value)
```

***

### getUserSetting()

> **getUserSetting**(`key`): `Promise`\<[`UserSetting`](/api/sdk/interfaces/usersetting/)\>

Get only the user's own setting (no fallback to system)

Returns the user's own setting for this key, or throws if not found.
Use this when you specifically want to check if the user has set a value.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `key` | `string` | Setting key |

#### Returns

`Promise`\<[`UserSetting`](/api/sdk/interfaces/usersetting/)\>

Promise resolving to UserSetting

#### Throws

Error if user has no setting with this key

#### Example

```typescript
// Check if user has set their own theme
try {
  const setting = await client.settings.getUserSetting('theme')
  console.log('User theme:', setting.value)
} catch (e) {
  console.log('User has not set a theme')
}
```

***

### listSecrets()

> **listSecrets**(): `Promise`\<`SecretSettingMetadata`[]\>

List all user's secret settings (metadata only, never includes values)

#### Returns

`Promise`\<`SecretSettingMetadata`[]\>

Promise resolving to array of SecretSettingMetadata

#### Example

```typescript
const secrets = await client.settings.listSecrets()
secrets.forEach(s => console.log(s.key, s.description))
```

***

### listSettings()

> **listSettings**(): `Promise`\<[`UserSetting`](/api/sdk/interfaces/usersetting/)[]\>

List all user's own settings

Returns all non-encrypted settings the current user has set.
Does not include system defaults.

#### Returns

`Promise`\<[`UserSetting`](/api/sdk/interfaces/usersetting/)[]\>

Promise resolving to array of UserSetting

#### Example

```typescript
const settings = await client.settings.listSettings()
settings.forEach(s => console.log(s.key, s.value))
```

***

### setSecret()

> **setSecret**(`key`, `value`, `options?`): `Promise`\<`SecretSettingMetadata`\>

Set a user secret setting (encrypted)

Creates or updates an encrypted secret that belongs to the current user.
The value is encrypted server-side with a user-specific key and can only be
accessed by edge functions, background jobs, or custom handlers running on
behalf of this user. Even admins cannot see the decrypted value.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `key` | `string` | Secret key |
| `value` | `string` | Secret value (will be encrypted server-side) |
| `options?` | \{ `description?`: `string`; \} | Optional description |
| `options.description?` | `string` | - |

#### Returns

`Promise`\<`SecretSettingMetadata`\>

Promise resolving to SecretSettingMetadata (never includes the value)

#### Example

```typescript
// Store user's API key for a third-party service
await client.settings.setSecret('openai_api_key', 'sk-abc123', {
  description: 'My OpenAI API key'
})
```

***

### setSetting()

> **setSetting**(`key`, `value`, `options?`): `Promise`\<[`UserSetting`](/api/sdk/interfaces/usersetting/)\>

Set a user setting (create or update)

Creates or updates a non-encrypted user setting.
This value will override any system default when using getSetting().

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `key` | `string` | Setting key |
| `value` | `Record`\<`string`, `unknown`\> | Setting value (any JSON-serializable object) |
| `options?` | \{ `description?`: `string`; \} | Optional description |
| `options.description?` | `string` | - |

#### Returns

`Promise`\<[`UserSetting`](/api/sdk/interfaces/usersetting/)\>

Promise resolving to UserSetting

#### Example

```typescript
// Set user's theme preference
await client.settings.setSetting('theme', { mode: 'dark', accent: 'blue' })

// Set with description
await client.settings.setSetting('notifications', { email: true, push: false }, {
  description: 'User notification preferences'
})
```
