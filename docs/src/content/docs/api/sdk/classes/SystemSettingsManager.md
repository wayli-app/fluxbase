---
editUrl: false
next: false
prev: false
title: "SystemSettingsManager"
---

System Settings Manager

Manages low-level system settings with key-value storage.
For application-level settings, use AppSettingsManager instead.

## Example

```typescript
const settings = client.admin.settings.system

// List all system settings
const { settings } = await settings.list()

// Get specific setting
const setting = await settings.get('app.auth.enable_signup')

// Update setting
await settings.update('app.auth.enable_signup', {
  value: { value: true },
  description: 'Enable user signup'
})

// Delete setting
await settings.delete('app.auth.enable_signup')
```

## Constructors

### new SystemSettingsManager()

> **new SystemSettingsManager**(`fetch`): [`SystemSettingsManager`](/api/sdk/classes/systemsettingsmanager/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |

#### Returns

[`SystemSettingsManager`](/api/sdk/classes/systemsettingsmanager/)

## Methods

### delete()

> **delete**(`key`): `Promise`\<`void`\>

Delete a system setting

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `key` | `string` | Setting key to delete |

#### Returns

`Promise`\<`void`\>

Promise<void>

#### Example

```typescript
await client.admin.settings.system.delete('app.auth.enable_signup')
```

***

### get()

> **get**(`key`): `Promise`\<[`SystemSetting`](/api/sdk/interfaces/systemsetting/)\>

Get a specific system setting by key

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `key` | `string` | Setting key (e.g., 'app.auth.enable_signup') |

#### Returns

`Promise`\<[`SystemSetting`](/api/sdk/interfaces/systemsetting/)\>

Promise resolving to SystemSetting

#### Example

```typescript
const setting = await client.admin.settings.system.get('app.auth.enable_signup')
console.log(setting.value)
```

***

### list()

> **list**(): `Promise`\<[`ListSystemSettingsResponse`](/api/sdk/interfaces/listsystemsettingsresponse/)\>

List all system settings

#### Returns

`Promise`\<[`ListSystemSettingsResponse`](/api/sdk/interfaces/listsystemsettingsresponse/)\>

Promise resolving to ListSystemSettingsResponse

#### Example

```typescript
const response = await client.admin.settings.system.list()
console.log(response.settings)
```

***

### update()

> **update**(`key`, `request`): `Promise`\<[`SystemSetting`](/api/sdk/interfaces/systemsetting/)\>

Update or create a system setting

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `key` | `string` | Setting key |
| `request` | [`UpdateSystemSettingRequest`](/api/sdk/interfaces/updatesystemsettingrequest/) | Update request with value and optional description |

#### Returns

`Promise`\<[`SystemSetting`](/api/sdk/interfaces/systemsetting/)\>

Promise resolving to SystemSetting

#### Example

```typescript
const updated = await client.admin.settings.system.update('app.auth.enable_signup', {
  value: { value: true },
  description: 'Enable user signup'
})
```
