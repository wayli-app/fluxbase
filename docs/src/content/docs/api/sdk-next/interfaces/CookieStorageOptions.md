---
editUrl: false
next: false
prev: false
title: "CookieStorageOptions"
---

Fluxbase Next.js SDK

Server/client adapters and SSR cookie storage for Fluxbase, built on the
core `@nimbleflux/fluxbase-sdk`.

This is a scaffold: it provides the SSR-auth foundation (cookie storage,
server client factory, client provider). Full React Query hooks (like the
React SDK) are a follow-on.

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="httponly"></a> `httpOnly?` | `boolean` | httpOnly so the cookie is not readable from client JS. **Default** `true` |
| <a id="maxage"></a> `maxAge?` | `number` | Max age in seconds. **Default** `undefined (session cookie)` |
| <a id="path"></a> `path?` | `string` | Cookie path. **Default** `"/"` |
| <a id="samesite"></a> `sameSite?` | `"lax"` \| `"strict"` \| `"none"` | SameSite attribute. **Default** `"lax"` |
| <a id="secure"></a> `secure?` | `boolean` | Require HTTPS. **Default** `process.env.NODE_ENV === 'production'` |
