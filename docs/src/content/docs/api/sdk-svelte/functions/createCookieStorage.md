---
editUrl: false
next: false
prev: false
title: "createCookieStorage"
---

> **createCookieStorage**(`cookies`, `options?`): [`StorageAdapter`](/api/sdk-svelte/interfaces/storageadapter/)

Build a `StorageAdapter` backed by SvelteKit's `cookies()`.

Every key the core SDK stores (e.g. `fluxbase.auth.session`) becomes its
own cookie. Values larger than a cookie can hold (~4KB) should be avoided;
the Fluxbase session JSON is well under that limit.

## Parameters

| Parameter | Type |
| ------ | ------ |
| `cookies` | [`SvelteKitCookies`](/api/sdk-svelte/interfaces/sveltekitcookies/) |
| `options` | [`CookieStorageOptions`](/api/sdk-svelte/interfaces/cookiestorageoptions/) |

## Returns

[`StorageAdapter`](/api/sdk-svelte/interfaces/storageadapter/)
