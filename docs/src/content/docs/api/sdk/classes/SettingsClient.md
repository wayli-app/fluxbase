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

### new SettingsClient()

> **new SettingsClient**(`fetch`): [`SettingsClient`](/api/sdk/classes/settingsclient/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |

#### Returns

[`SettingsClient`](/api/sdk/classes/settingsclient/)

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

> **getMany**(`keys`): `Promise`\<`Record`\<`string`, `any`\>\>

Get multiple settings' values by keys

Fetches multiple settings in a single request.
Only returns settings the user has permission to read based on RLS policies.
Settings the user can't access will be omitted from the result (no error thrown).

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `keys` | `string`[] | Array of setting keys to fetch |

#### Returns

`Promise`\<`Record`\<`string`, `any`\>\>

Promise resolving to object mapping keys to values

#### Example

```typescript
const values = await client.settings.getMany([
  'features.beta_enabled',  // public - will be returned
  'features.dark_mode',      // public - will be returned
  'internal.api_key'         // secret - will be omitted
])
console.log(values)
// {
//   'features.beta_enabled': { enabled: true },
//   'features.dark_mode': { enabled: false }
//   // 'internal.api_key' is omitted (no error)
// }
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

### setSecret()

> **setSecret**(`key`, `value`, `options`?): `Promise`\<`SecretSettingMetadata`\>

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
| `options`? | `object` | Optional description |
| `options.description`? | `string` | - |

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
