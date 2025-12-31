---
editUrl: false
next: false
prev: false
title: "AppSettingsManager"
---

Application Settings Manager

Manages high-level application settings with a structured API.
Provides type-safe access to authentication, features, email, and security settings.

## Example

```typescript
const settings = client.admin.settings.app

// Get all app settings
const appSettings = await settings.get()
console.log(appSettings.authentication.enable_signup)

// Update specific settings
const updated = await settings.update({
  authentication: {
    enable_signup: true,
    password_min_length: 12
  }
})

// Reset to defaults
await settings.reset()
```

## Constructors

### new AppSettingsManager()

> **new AppSettingsManager**(`fetch`): [`AppSettingsManager`](/api/sdk/classes/appsettingsmanager/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |

#### Returns

[`AppSettingsManager`](/api/sdk/classes/appsettingsmanager/)

## Methods

### configureMailgun()

> **configureMailgun**(`apiKey`, `domain`, `options`?): `Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Configure Mailgun email provider

Convenience method to set up Mailgun email delivery.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `apiKey` | `string` | Mailgun API key |
| `domain` | `string` | Mailgun domain |
| `options`? | `object` | Optional EU region flag and email addresses |
| `options.eu_region`? | `boolean` | - |
| `options.from_address`? | `string` | - |
| `options.from_name`? | `string` | - |
| `options.reply_to_address`? | `string` | - |

#### Returns

`Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Promise resolving to AppSettings

#### Example

```typescript
await client.admin.settings.app.configureMailgun('key-xxx', 'mg.yourapp.com', {
  eu_region: false,
  from_address: 'noreply@yourapp.com',
  from_name: 'Your App'
})
```

***

### configureSendGrid()

> **configureSendGrid**(`apiKey`, `options`?): `Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Configure SendGrid email provider

Convenience method to set up SendGrid email delivery.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `apiKey` | `string` | SendGrid API key |
| `options`? | `object` | Optional from address, name, and reply-to |
| `options.from_address`? | `string` | - |
| `options.from_name`? | `string` | - |
| `options.reply_to_address`? | `string` | - |

#### Returns

`Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Promise resolving to AppSettings

#### Example

```typescript
await client.admin.settings.app.configureSendGrid('SG.xxx', {
  from_address: 'noreply@yourapp.com',
  from_name: 'Your App'
})
```

***

### configureSES()

> **configureSES**(`accessKeyId`, `secretAccessKey`, `region`, `options`?): `Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Configure AWS SES email provider

Convenience method to set up AWS SES email delivery.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `accessKeyId` | `string` | AWS access key ID |
| `secretAccessKey` | `string` | AWS secret access key |
| `region` | `string` | AWS region (e.g., 'us-east-1') |
| `options`? | `object` | Optional email addresses |
| `options.from_address`? | `string` | - |
| `options.from_name`? | `string` | - |
| `options.reply_to_address`? | `string` | - |

#### Returns

`Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Promise resolving to AppSettings

#### Example

```typescript
await client.admin.settings.app.configureSES(
  'AKIAIOSFODNN7EXAMPLE',
  'wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY',
  'us-east-1',
  {
    from_address: 'noreply@yourapp.com',
    from_name: 'Your App'
  }
)
```

***

### configureSMTP()

> **configureSMTP**(`config`): `Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Configure SMTP email provider

Convenience method to set up SMTP email delivery.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `config` | `object` | SMTP configuration |
| `config.from_address`? | `string` | - |
| `config.from_name`? | `string` | - |
| `config.host` | `string` | - |
| `config.password` | `string` | - |
| `config.port` | `number` | - |
| `config.reply_to_address`? | `string` | - |
| `config.use_tls` | `boolean` | - |
| `config.username` | `string` | - |

#### Returns

`Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Promise resolving to AppSettings

#### Example

```typescript
await client.admin.settings.app.configureSMTP({
  host: 'smtp.gmail.com',
  port: 587,
  username: 'your-email@gmail.com',
  password: 'your-app-password',
  use_tls: true,
  from_address: 'noreply@yourapp.com',
  from_name: 'Your App'
})
```

***

### deleteSecretSetting()

> **deleteSecretSetting**(`key`): `Promise`\<`void`\>

Delete a system secret setting

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `key` | `string` | Secret key to delete |

#### Returns

`Promise`\<`void`\>

Promise<void>

#### Example

```typescript
await client.admin.settings.app.deleteSecretSetting('stripe_api_key')
```

***

### deleteSetting()

> **deleteSetting**(`key`): `Promise`\<`void`\>

Delete a custom setting

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `key` | `string` | Setting key to delete |

#### Returns

`Promise`\<`void`\>

Promise<void>

#### Example

```typescript
await client.admin.settings.app.deleteSetting('billing.tiers')
```

***

### disableSignup()

> **disableSignup**(): `Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Disable user signup

Convenience method to disable user registration.

#### Returns

`Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Promise resolving to AppSettings

#### Example

```typescript
await client.admin.settings.app.disableSignup()
```

***

### enableSignup()

> **enableSignup**(): `Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Enable user signup

Convenience method to enable user registration.

#### Returns

`Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Promise resolving to AppSettings

#### Example

```typescript
await client.admin.settings.app.enableSignup()
```

***

### get()

> **get**(): `Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Get all application settings

Returns structured settings for authentication, features, email, and security.

#### Returns

`Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Promise resolving to AppSettings

#### Example

```typescript
const settings = await client.admin.settings.app.get()

console.log('Signup enabled:', settings.authentication.enable_signup)
console.log('Realtime enabled:', settings.features.enable_realtime)
console.log('Email provider:', settings.email.provider)
```

***

### getSecretSetting()

> **getSecretSetting**(`key`): `Promise`\<`SecretSettingMetadata`\>

Get metadata for a system secret setting (never returns the value)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `key` | `string` | Secret key |

#### Returns

`Promise`\<`SecretSettingMetadata`\>

Promise resolving to SecretSettingMetadata

#### Example

```typescript
const metadata = await client.admin.settings.app.getSecretSetting('stripe_api_key')
console.log(metadata.key, metadata.updated_at)
// Note: metadata.value is never included
```

***

### getSetting()

> **getSetting**(`key`): `Promise`\<`any`\>

Get a specific custom setting's value only (without metadata)

Convenience method that returns just the value field instead of the full CustomSetting object.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `key` | `string` | Setting key (e.g., 'billing.tiers', 'features.beta_enabled') |

#### Returns

`Promise`\<`any`\>

Promise resolving to the setting's value

#### Example

```typescript
const tiers = await client.admin.settings.app.getSetting('billing.tiers')
console.log(tiers) // { free: 1000, pro: 10000, enterprise: 100000 }
```

***

### getSettings()

> **getSettings**(`keys`): `Promise`\<`Record`\<`string`, `any`\>\>

Get multiple custom settings' values by keys

Fetches multiple settings in a single request and returns only their values.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `keys` | `string`[] | Array of setting keys to fetch |

#### Returns

`Promise`\<`Record`\<`string`, `any`\>\>

Promise resolving to object mapping keys to values

#### Example

```typescript
const values = await client.admin.settings.app.getSettings([
  'billing.tiers',
  'features.beta_enabled'
])
console.log(values)
// {
//   'billing.tiers': { free: 1000, pro: 10000 },
//   'features.beta_enabled': { enabled: true }
// }
```

***

### getUserSecretValue()

> **getUserSecretValue**(`userId`, `key`): `Promise`\<`string`\>

Get the decrypted value of a user's secret setting

This is a privileged operation that requires service_role.
Use this to retrieve user-specific secrets when running as a service
(e.g., in edge functions or background jobs).

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `userId` | `string` | The user ID whose secret to retrieve |
| `key` | `string` | Secret key |

#### Returns

`Promise`\<`string`\>

Promise resolving to the decrypted secret value

#### Example

```typescript
// In an edge function, get a user's API key for validation
const apiKey = await fluxbaseService.admin.settings.app.getUserSecretValue(
  userId,
  'owntracks_api_key'
)
if (apiKey !== providedKey) {
  throw new Error('Invalid API key')
}
```

***

### listSecretSettings()

> **listSecretSettings**(): `Promise`\<`SecretSettingMetadata`[]\>

List all system secret settings (metadata only, never includes values)

#### Returns

`Promise`\<`SecretSettingMetadata`[]\>

Promise resolving to array of SecretSettingMetadata

#### Example

```typescript
const secrets = await client.admin.settings.app.listSecretSettings()
secrets.forEach(s => console.log(s.key, s.description))
```

***

### listSettings()

> **listSettings**(): `Promise`\<`CustomSetting`[]\>

List all custom settings

#### Returns

`Promise`\<`CustomSetting`[]\>

Promise resolving to array of CustomSetting objects

#### Example

```typescript
const settings = await client.admin.settings.app.listSettings()
settings.forEach(s => console.log(s.key, s.value))
```

***

### reset()

> **reset**(): `Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Reset all application settings to defaults

This will delete all custom settings and return to default values.

#### Returns

`Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Promise resolving to AppSettings - Default settings

#### Example

```typescript
const defaults = await client.admin.settings.app.reset()
console.log('Settings reset to defaults:', defaults)
```

***

### setEmailEnabled()

> **setEmailEnabled**(`enabled`): `Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Enable or disable email functionality

Convenience method to toggle email system on/off.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `enabled` | `boolean` | Whether to enable email |

#### Returns

`Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Promise resolving to AppSettings

#### Example

```typescript
await client.admin.settings.app.setEmailEnabled(true)
```

***

### setEmailVerificationRequired()

> **setEmailVerificationRequired**(`required`): `Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Enable or disable email verification requirement

Convenience method to require email verification for new signups.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `required` | `boolean` | Whether to require email verification |

#### Returns

`Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Promise resolving to AppSettings

#### Example

```typescript
await client.admin.settings.app.setEmailVerificationRequired(true)
```

***

### setFeature()

> **setFeature**(`feature`, `enabled`): `Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Enable or disable a feature

Convenience method to toggle feature flags.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `feature` | `"functions"` \| `"realtime"` \| `"storage"` | Feature name ('realtime' | 'storage' | 'functions') |
| `enabled` | `boolean` | Whether to enable or disable the feature |

#### Returns

`Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Promise resolving to AppSettings

#### Example

```typescript
// Enable realtime
await client.admin.settings.app.setFeature('realtime', true)

// Disable storage
await client.admin.settings.app.setFeature('storage', false)
```

***

### setPasswordComplexity()

> **setPasswordComplexity**(`requirements`): `Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Configure password complexity requirements

Convenience method to set password validation rules.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `requirements` | `object` | Password complexity requirements |
| `requirements.min_length`? | `number` | - |
| `requirements.require_lowercase`? | `boolean` | - |
| `requirements.require_number`? | `boolean` | - |
| `requirements.require_special`? | `boolean` | - |
| `requirements.require_uppercase`? | `boolean` | - |

#### Returns

`Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Promise resolving to AppSettings

#### Example

```typescript
await client.admin.settings.app.setPasswordComplexity({
  min_length: 12,
  require_uppercase: true,
  require_lowercase: true,
  require_number: true,
  require_special: true
})
```

***

### setPasswordMinLength()

> **setPasswordMinLength**(`length`): `Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Update password minimum length

Convenience method to set password requirements.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `length` | `number` | Minimum password length (8-128 characters) |

#### Returns

`Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Promise resolving to AppSettings

#### Example

```typescript
await client.admin.settings.app.setPasswordMinLength(12)
```

***

### setRateLimiting()

> **setRateLimiting**(`enabled`): `Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Enable or disable global rate limiting

Convenience method to toggle global rate limiting.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `enabled` | `boolean` | Whether to enable rate limiting |

#### Returns

`Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Promise resolving to AppSettings

#### Example

```typescript
await client.admin.settings.app.setRateLimiting(true)
```

***

### setSecretSetting()

> **setSecretSetting**(`key`, `value`, `options`?): `Promise`\<`SecretSettingMetadata`\>

Set a system-level secret setting (encrypted)

Creates or updates an encrypted system secret. The value is encrypted server-side
and can only be accessed by edge functions, background jobs, or custom handlers.
The SDK never returns the decrypted value.

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
await client.admin.settings.app.setSecretSetting('stripe_api_key', 'sk-live-xxx', {
  description: 'Stripe API key for payment processing'
})
```

***

### setSessionSettings()

> **setSessionSettings**(`timeoutMinutes`, `maxSessionsPerUser`): `Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Configure session settings

Convenience method to set session timeout and limits.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `timeoutMinutes` | `number` | Session timeout in minutes (0 for no timeout) |
| `maxSessionsPerUser` | `number` | Maximum concurrent sessions per user (0 for unlimited) |

#### Returns

`Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Promise resolving to AppSettings

#### Example

```typescript
// 30 minute sessions, max 3 devices per user
await client.admin.settings.app.setSessionSettings(30, 3)
```

***

### setSetting()

> **setSetting**(`key`, `value`, `options`?): `Promise`\<`CustomSetting`\>

Set or create a custom setting

Creates a new custom setting or updates an existing one.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `key` | `string` | Setting key |
| `value` | `any` | Setting value (any JSON-serializable value) |
| `options`? | `object` | Optional configuration (description, is_public, is_secret, etc.) |
| `options.description`? | `string` | - |
| `options.is_public`? | `boolean` | - |
| `options.is_secret`? | `boolean` | - |
| `options.value_type`? | `string` | - |

#### Returns

`Promise`\<`CustomSetting`\>

Promise resolving to CustomSetting

#### Example

```typescript
await client.admin.settings.app.setSetting('billing.tiers', {
  free: 1000,
  pro: 10000,
  enterprise: 100000
}, {
  description: 'API quotas per billing tier',
  is_public: false
})
```

***

### update()

> **update**(`request`): `Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Update application settings

Supports partial updates - only provide the fields you want to change.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `request` | [`UpdateAppSettingsRequest`](/api/sdk/interfaces/updateappsettingsrequest/) | Settings to update (partial update supported) |

#### Returns

`Promise`\<[`AppSettings`](/api/sdk/interfaces/appsettings/)\>

Promise resolving to AppSettings - Updated settings

#### Example

```typescript
// Update authentication settings
const updated = await client.admin.settings.app.update({
  authentication: {
    enable_signup: true,
    password_min_length: 12
  }
})

// Update multiple categories
await client.admin.settings.app.update({
  authentication: { enable_signup: false },
  features: { enable_realtime: true },
  security: { enable_global_rate_limit: true }
})
```
