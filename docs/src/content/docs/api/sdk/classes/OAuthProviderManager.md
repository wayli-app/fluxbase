---
editUrl: false
next: false
prev: false
title: "OAuthProviderManager"
---

OAuth Provider Manager

Manages OAuth provider configurations for third-party authentication.
Supports both built-in providers (Google, GitHub, etc.) and custom OAuth2 providers.

## Example

```typescript
const oauth = client.admin.oauth

// List all OAuth providers
const { providers } = await oauth.listProviders()

// Create a new provider
await oauth.createProvider({
  provider_name: 'github',
  display_name: 'GitHub',
  enabled: true,
  client_id: 'your-client-id',
  client_secret: 'your-client-secret',
  redirect_url: 'https://yourapp.com/auth/callback',
  scopes: ['user:email', 'read:user'],
  is_custom: false
})

// Update a provider
await oauth.updateProvider('provider-id', {
  enabled: false
})

// Delete a provider
await oauth.deleteProvider('provider-id')
```

## Constructors

### new OAuthProviderManager()

> **new OAuthProviderManager**(`fetch`): [`OAuthProviderManager`](/api/sdk/classes/oauthprovidermanager/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |

#### Returns

[`OAuthProviderManager`](/api/sdk/classes/oauthprovidermanager/)

## Methods

### createProvider()

> **createProvider**(`request`): `Promise`\<[`CreateOAuthProviderResponse`](/api/sdk/interfaces/createoauthproviderresponse/)\>

Create a new OAuth provider

Creates a new OAuth provider configuration. For built-in providers (Google, GitHub, etc.),
set `is_custom` to false. For custom OAuth2 providers, set `is_custom` to true and provide
the authorization, token, and user info URLs.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `request` | [`CreateOAuthProviderRequest`](/api/sdk/interfaces/createoauthproviderrequest/) | OAuth provider configuration |

#### Returns

`Promise`\<[`CreateOAuthProviderResponse`](/api/sdk/interfaces/createoauthproviderresponse/)\>

Promise resolving to CreateOAuthProviderResponse

#### Examples

```typescript
// Create GitHub provider
const result = await client.admin.oauth.createProvider({
  provider_name: 'github',
  display_name: 'GitHub',
  enabled: true,
  client_id: process.env.GITHUB_CLIENT_ID,
  client_secret: process.env.GITHUB_CLIENT_SECRET,
  redirect_url: 'https://yourapp.com/auth/callback',
  scopes: ['user:email', 'read:user'],
  is_custom: false
})

console.log('Provider created:', result.id)
```

```typescript
// Create custom OAuth2 provider
await client.admin.oauth.createProvider({
  provider_name: 'custom_sso',
  display_name: 'Custom SSO',
  enabled: true,
  client_id: 'client-id',
  client_secret: 'client-secret',
  redirect_url: 'https://yourapp.com/auth/callback',
  scopes: ['openid', 'profile', 'email'],
  is_custom: true,
  authorization_url: 'https://sso.example.com/oauth/authorize',
  token_url: 'https://sso.example.com/oauth/token',
  user_info_url: 'https://sso.example.com/oauth/userinfo'
})
```

***

### deleteProvider()

> **deleteProvider**(`providerId`): `Promise`\<[`DeleteOAuthProviderResponse`](/api/sdk/interfaces/deleteoauthproviderresponse/)\>

Delete an OAuth provider

Permanently deletes an OAuth provider configuration. This will prevent users from
authenticating with this provider.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `providerId` | `string` | Provider ID (UUID) to delete |

#### Returns

`Promise`\<[`DeleteOAuthProviderResponse`](/api/sdk/interfaces/deleteoauthproviderresponse/)\>

Promise resolving to DeleteOAuthProviderResponse

#### Examples

```typescript
await client.admin.oauth.deleteProvider('provider-id')
console.log('Provider deleted')
```

```typescript
// Safe deletion with confirmation
const provider = await client.admin.oauth.getProvider('provider-id')
const confirmed = await confirm(`Delete ${provider.display_name}?`)

if (confirmed) {
  await client.admin.oauth.deleteProvider('provider-id')
}
```

***

### disableProvider()

> **disableProvider**(`providerId`): `Promise`\<[`UpdateOAuthProviderResponse`](/api/sdk/interfaces/updateoauthproviderresponse/)\>

Disable an OAuth provider

Convenience method to disable a provider.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `providerId` | `string` | Provider ID (UUID) |

#### Returns

`Promise`\<[`UpdateOAuthProviderResponse`](/api/sdk/interfaces/updateoauthproviderresponse/)\>

Promise resolving to UpdateOAuthProviderResponse

#### Example

```typescript
await client.admin.oauth.disableProvider('provider-id')
```

***

### enableProvider()

> **enableProvider**(`providerId`): `Promise`\<[`UpdateOAuthProviderResponse`](/api/sdk/interfaces/updateoauthproviderresponse/)\>

Enable an OAuth provider

Convenience method to enable a provider.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `providerId` | `string` | Provider ID (UUID) |

#### Returns

`Promise`\<[`UpdateOAuthProviderResponse`](/api/sdk/interfaces/updateoauthproviderresponse/)\>

Promise resolving to UpdateOAuthProviderResponse

#### Example

```typescript
await client.admin.oauth.enableProvider('provider-id')
```

***

### getProvider()

> **getProvider**(`providerId`): `Promise`\<[`OAuthProvider`](/api/sdk/interfaces/oauthprovider/)\>

Get a specific OAuth provider by ID

Retrieves detailed configuration for a single OAuth provider.
Note: Client secret is not included in the response.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `providerId` | `string` | Provider ID (UUID) |

#### Returns

`Promise`\<[`OAuthProvider`](/api/sdk/interfaces/oauthprovider/)\>

Promise resolving to OAuthProvider

#### Example

```typescript
const provider = await client.admin.oauth.getProvider('provider-uuid')

console.log('Provider:', provider.display_name)
console.log('Scopes:', provider.scopes.join(', '))
console.log('Redirect URL:', provider.redirect_url)
```

***

### listProviders()

> **listProviders**(): `Promise`\<[`OAuthProvider`](/api/sdk/interfaces/oauthprovider/)[]\>

List all OAuth providers

Retrieves all configured OAuth providers including both enabled and disabled providers.
Note: Client secrets are not included in the response for security reasons.

#### Returns

`Promise`\<[`OAuthProvider`](/api/sdk/interfaces/oauthprovider/)[]\>

Promise resolving to ListOAuthProvidersResponse

#### Example

```typescript
const { providers } = await client.admin.oauth.listProviders()

providers.forEach(provider => {
  console.log(`${provider.display_name}: ${provider.enabled ? 'enabled' : 'disabled'}`)
})
```

***

### updateProvider()

> **updateProvider**(`providerId`, `request`): `Promise`\<[`UpdateOAuthProviderResponse`](/api/sdk/interfaces/updateoauthproviderresponse/)\>

Update an existing OAuth provider

Updates an OAuth provider configuration. All fields are optional - only provided fields
will be updated. To update the client secret, provide a non-empty value.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `providerId` | `string` | Provider ID (UUID) |
| `request` | [`UpdateOAuthProviderRequest`](/api/sdk/interfaces/updateoauthproviderrequest/) | Fields to update |

#### Returns

`Promise`\<[`UpdateOAuthProviderResponse`](/api/sdk/interfaces/updateoauthproviderresponse/)\>

Promise resolving to UpdateOAuthProviderResponse

#### Examples

```typescript
// Disable a provider
await client.admin.oauth.updateProvider('provider-id', {
  enabled: false
})
```

```typescript
// Update scopes and redirect URL
await client.admin.oauth.updateProvider('provider-id', {
  scopes: ['user:email', 'read:user', 'read:org'],
  redirect_url: 'https://newdomain.com/auth/callback'
})
```

```typescript
// Rotate client secret
await client.admin.oauth.updateProvider('provider-id', {
  client_id: 'new-client-id',
  client_secret: 'new-client-secret'
})
```
