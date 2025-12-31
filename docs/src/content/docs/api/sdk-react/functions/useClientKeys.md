---
editUrl: false
next: false
prev: false
title: "useClientKeys"
---

> **useClientKeys**(`options`): `UseClientKeysReturn`

Hook for managing client keys

Provides client key list and management functions.

## Parameters

| Parameter | Type |
| ------ | ------ |
| `options` | `UseClientKeysOptions` |

## Returns

`UseClientKeysReturn`

## Example

```tsx
function ClientKeyManager() {
  const { keys, isLoading, createKey, revokeKey } = useClientKeys()

  const handleCreate = async () => {
    const { key, keyData } = await createKey({
      name: 'Backend Service',
      description: 'Client key for backend',
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
