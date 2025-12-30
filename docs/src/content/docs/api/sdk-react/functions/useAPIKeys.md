---
editUrl: false
next: false
prev: false
title: "useAPIKeys"
---

> **useAPIKeys**(`options`): `UseAPIKeysReturn`

Hook for managing client keys

Provides API key list and management functions.

## Parameters

| Parameter | Type |
| ------ | ------ |
| `options` | `UseAPIKeysOptions` |

## Returns

`UseAPIKeysReturn`

## Example

```tsx
function APIKeyManager() {
  const { keys, isLoading, createKey, revokeKey } = useAPIKeys()

  const handleCreate = async () => {
    const { key, keyData } = await createKey({
      name: 'Backend Service',
      description: 'API key for backend',
      expires_at: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000).toISOString()
    })
    alert(`Key created: ${key}`)
  }

  return (
    <div>
      <button onClick={handleCreate}>Create Key</button>
      {keys.map(k => (
        <div key={k.id}>
          {k.name}
          <button onClick={() => revokeKey(k.id)}>Revoke</button>
        </div>
      ))}
    </div>
  )
}
```
