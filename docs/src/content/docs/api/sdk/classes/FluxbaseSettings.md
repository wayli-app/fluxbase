---
editUrl: false
next: false
prev: false
title: "FluxbaseSettings"
---

Settings Manager

Provides access to system-level and application-level settings.
AppSettingsManager now handles both structured framework settings and custom key-value settings.

## Example

```typescript
const settings = client.admin.settings

// Access system settings
const systemSettings = await settings.system.list()

// Access app settings (structured)
const appSettings = await settings.app.get()
await settings.app.enableSignup()

// Access custom settings (key-value)
await settings.app.setSetting('billing.tiers', { free: 1000, pro: 10000 })
const tiers = await settings.app.getSetting('billing.tiers')
```

## Constructors

### new FluxbaseSettings()

> **new FluxbaseSettings**(`fetch`): [`FluxbaseSettings`](/api/sdk/classes/fluxbasesettings/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |

#### Returns

[`FluxbaseSettings`](/api/sdk/classes/fluxbasesettings/)

## Properties

| Property | Modifier | Type |
| ------ | ------ | ------ |
| `app` | `public` | [`AppSettingsManager`](/api/sdk/classes/appsettingsmanager/) |
| `system` | `public` | [`SystemSettingsManager`](/api/sdk/classes/systemsettingsmanager/) |
