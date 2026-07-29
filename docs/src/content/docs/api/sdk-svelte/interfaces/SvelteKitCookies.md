---
editUrl: false
next: false
prev: false
title: "SvelteKitCookies"
---

The subset of SvelteKit's `Cookies` API this adapter depends on.
Kept structural so it works against the real type without importing kit
at runtime (the adapter ships in a browser/edge bundle too).

## Methods

### delete()

> **delete**(`name`, `opts?`): `void`

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `name` | `string` |
| `opts?` | \{ `path?`: `string`; \} |
| `opts.path?` | `string` |

#### Returns

`void`

***

### get()

> **get**(`name`): `string` \| `undefined`

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `name` | `string` |

#### Returns

`string` \| `undefined`

***

### set()

> **set**(`name`, `value`, `opts?`): `void`

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `name` | `string` |
| `value` | `string` |
| `opts?` | \{ `expires?`: `Date`; `httpOnly?`: `boolean`; `maxAge?`: `number`; `path?`: `string`; `sameSite?`: `"lax"` \| `"strict"` \| `"none"`; `secure?`: `boolean`; \} |
| `opts.expires?` | `Date` |
| `opts.httpOnly?` | `boolean` |
| `opts.maxAge?` | `number` |
| `opts.path?` | `string` |
| `opts.sameSite?` | `"lax"` \| `"strict"` \| `"none"` |
| `opts.secure?` | `boolean` |

#### Returns

`void`
