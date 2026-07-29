---
editUrl: false
next: false
prev: false
title: "NextCookies"
---

The subset of Next.js's `ReadonlyRequestCookies` / `ResponseCookies` API
this adapter depends on. Kept structural to avoid importing `next` at
runtime from this framework-agnostic shim.

## Methods

### delete()

> **delete**(`name`): `void`

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `name` | `string` |

#### Returns

`void`

***

### get()

> **get**(`name`): \{ `value?`: `string`; \} \| `undefined`

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `name` | `string` |

#### Returns

\{ `value?`: `string`; \} \| `undefined`

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
