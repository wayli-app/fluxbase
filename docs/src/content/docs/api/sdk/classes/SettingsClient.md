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
