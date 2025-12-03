---
editUrl: false
next: false
prev: false
title: "useSystemSettings"
---

> **useSystemSettings**(`options`): `UseSystemSettingsReturn`

Hook for managing system settings (key-value storage)

## Parameters

| Parameter | Type |
| ------ | ------ |
| `options` | `UseSystemSettingsOptions` |

## Returns

`UseSystemSettingsReturn`

## Example

```tsx
function SystemSettings() {
  const { settings, isLoading, updateSetting } = useSystemSettings({ autoFetch: true })

  const handleUpdateSetting = async (key: string, value: any) => {
    await updateSetting(key, { value })
  }

  return <div>...</div>
}
```
