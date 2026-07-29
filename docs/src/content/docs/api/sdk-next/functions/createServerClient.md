---
editUrl: false
next: false
prev: false
title: "createServerClient"
---

> **createServerClient**(`url`, `options?`): [`FluxbaseClient`](/api/sdk-next/interfaces/fluxbaseclient/)

Create a Fluxbase client for use in Next.js server code. Reads and writes the
auth session via httpOnly cookies.

Pass `cookies()` from `next/headers` explicitly, or omit it and call this
inside a request scope where `next/headers` is available.

## Parameters

| Parameter | Type |
| ------ | ------ |
| `url` | `string` |
| `options` | [`CreateServerClientOptions`](/api/sdk-next/interfaces/createserverclientoptions/) |

## Returns

[`FluxbaseClient`](/api/sdk-next/interfaces/fluxbaseclient/)
