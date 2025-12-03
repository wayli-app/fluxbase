---
editUrl: false
next: false
prev: false
title: "useAppSettings"
---

> **useAppSettings**(`options`): `UseAppSettingsReturn`

Hook for managing application settings

## Parameters

| Parameter | Type |
| ------ | ------ |
| `options` | `UseAppSettingsOptions` |

## Returns

`UseAppSettingsReturn`

## Example

```tsx
function SettingsPanel() {
  const { settings, isLoading, updateSettings } = useAppSettings({ autoFetch: true })

  const handleToggleFeature = async (feature: string, enabled: boolean) => {
    await updateSettings({
      features: { ...settings?.features, [feature]: enabled }
    })
  }

  return <div>...</div>
}
```
