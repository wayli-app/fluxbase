---
editUrl: false
next: false
prev: false
title: "NuxtCookieEvent"
---

Minimal interface over an h3 `H3Event` (or any cookie bag) that this adapter
needs. Kept structural so it works against Nuxt's event without importing
h3 at runtime.

## Methods

### deleteCookie()

> **deleteCookie**(`name`, `opts?`): `void`

Delete a cookie.

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `name` | `string` |
| `opts?` | \{ `path?`: `string`; \} |
| `opts.path?` | `string` |

#### Returns

`void`

***

### getCookie()

> **getCookie**(`name`): `string` \| `undefined`

Read a cookie value.

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `name` | `string` |

#### Returns

`string` \| `undefined`

***

### setCookie()

> **setCookie**(`name`, `value`, `opts?`): `void`

Set a cookie with options.

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `name` | `string` |
| `value` | `string` |
| `opts?` | \{ `httpOnly?`: `boolean`; `maxAge?`: `number`; `path?`: `string`; `sameSite?`: `"lax"` \| `"strict"` \| `"none"`; `secure?`: `boolean`; \} |
| `opts.httpOnly?` | `boolean` |
| `opts.maxAge?` | `number` |
| `opts.path?` | `string` |
| `opts.sameSite?` | `"lax"` \| `"strict"` \| `"none"` |
| `opts.secure?` | `boolean` |

#### Returns

`void`
