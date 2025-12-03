---
editUrl: false
next: false
prev: false
title: "FluxbaseOAuth"
---

OAuth Configuration Manager

Root manager providing access to OAuth provider and authentication settings management.

## Example

```typescript
const oauth = client.admin.oauth

// Manage OAuth providers
const providers = await oauth.providers.listProviders()

// Manage auth settings
const settings = await oauth.authSettings.get()
```

## Constructors

### new FluxbaseOAuth()

> **new FluxbaseOAuth**(`fetch`): [`FluxbaseOAuth`](/api/sdk/classes/fluxbaseoauth/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |

#### Returns

[`FluxbaseOAuth`](/api/sdk/classes/fluxbaseoauth/)

## Properties

| Property | Modifier | Type |
| ------ | ------ | ------ |
| `authSettings` | `public` | [`AuthSettingsManager`](/api/sdk/classes/authsettingsmanager/) |
| `providers` | `public` | [`OAuthProviderManager`](/api/sdk/classes/oauthprovidermanager/) |
