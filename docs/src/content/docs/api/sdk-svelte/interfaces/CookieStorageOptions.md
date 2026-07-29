---
editUrl: false
next: false
prev: false
title: "CookieStorageOptions"
---

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="httponly"></a> `httpOnly?` | `boolean` | Mark the cookie httpOnly so it is not readable from client JS. **Default** `true` |
| <a id="maxage"></a> `maxAge?` | `number` | Max age in seconds. **Default** `undefined (session cookie)` |
| <a id="path"></a> `path?` | `string` | Cookie path. **Default** `"/"` |
| <a id="samesite"></a> `sameSite?` | `"lax"` \| `"strict"` \| `"none"` | SameSite attribute. **Default** `"lax"` |
| <a id="secure"></a> `secure?` | `boolean` | Require HTTPS. Defaults to true in production. **Default** `process.env.NODE_ENV === 'production'` |
