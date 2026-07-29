---
editUrl: false
next: false
prev: false
title: "StorageAdapter"
---

Storage adapter for persisting auth state.

Implements the same subset of the Web `Storage` interface used by the SDK
(`getItem`, `setItem`, `removeItem`). This lets callers swap the default
`localStorage`/in-memory storage for a custom store — most commonly a
cookie-backed adapter for SSR frameworks (Next.js, SvelteKit, Nuxt) where
the session must be read from an httpOnly cookie on the server.

## Example

```typescript
import type { StorageAdapter } from '@nimbleflux/fluxbase-sdk'

const memoryAdapter: StorageAdapter = {
  getItem: (k) => myMap.get(k) ?? null,
  setItem: (k, v) => myMap.set(k, v),
  removeItem: (k) => myMap.delete(k),
}
```

## Methods

### getItem()

> **getItem**(`key`): `string` \| `null`

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `key` | `string` |

#### Returns

`string` \| `null`

***

### removeItem()

> **removeItem**(`key`): `void`

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `key` | `string` |

#### Returns

`void`

***

### setItem()

> **setItem**(`key`, `value`): `void`

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `key` | `string` |
| `value` | `string` |

#### Returns

`void`
