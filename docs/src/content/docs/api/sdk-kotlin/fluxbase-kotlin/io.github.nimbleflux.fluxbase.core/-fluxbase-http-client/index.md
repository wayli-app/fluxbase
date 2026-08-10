---
title: "FluxbaseHttpClient"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.core](../index.md)/[FluxbaseHttpClient](index.md)

# FluxbaseHttpClient

class [FluxbaseHttpClient](index.md)(baseUrl: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), transport: [HttpTransport](../-http-transport/index.md), json: Json = defaultJson)

The shared HTTP client that every Fluxbase module (auth, postgrest, realtime, storage, jobs, …) uses to talk to the Fluxbase server. This is the Kotlin port of `FluxbaseFetch` (`sdk/src/fetch.ts`).

Responsibilities:

- 
   Hold the base URL (trailing slash stripped) and default headers.
- 
   Manage the auth token: `setAuthToken(token)` sets `Authorization: Bearer`;     `setAuthToken(null)` restores the anon key fallback (does NOT delete the     header — matches TS `fetch.ts:93-102`).
- 
   Provide convenience methods (`get`, `post`, `put`, `patch`, `delete`) that     return reified typed results via kotlinx.serialization.
- 
   Delegate the actual I/O to an [HttpTransport](../-http-transport/index.md) SPI, which makes the client     trivially testable with a recording fake.

NOTE on the 401 auto-refresh: the TS `FluxbaseFetch` does a single retry after refreshing the token on 401. That logic will be wired in S1 (the transport reports the status; the client decides whether to refresh+retry). For the S0 spike, requests are single-shot.

#### Parameters

jvm

| | |
|---|---|
| baseUrl | the Fluxbase server URL (trailing slash stripped). |
| transport | the I/O SPI. If null, a Ktor-backed transport is used at runtime; tests inject an io.github.nimbleflux.fluxbase.core.test.RecordingHttp. |
| json | the JSON decoder used for typed [get](get.md)/[post](post.md) responses. |

## Constructors

| | |
|---|---|
| [FluxbaseHttpClient](-fluxbase-http-client.md) | [jvm]<br>constructor(baseUrl: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), transport: [HttpTransport](../-http-transport/index.md), json: Json = defaultJson) |

## Types

| Name | Summary |
|---|---|
| [Companion](-companion/index.md) | [jvm]<br>object [Companion](-companion/index.md) |

## Properties

| Name | Summary |
|---|---|
| [baseUrl](base-url.md) | [jvm]<br>val [baseUrl](base-url.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)<br>The base URL with any trailing slash removed. |
| [defaultHeaders](default-headers.md) | [jvm]<br>val [defaultHeaders](default-headers.md): [MutableMap](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-mutable-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt;<br>Default headers applied to every request. `Content-Type: application/json` is always present; `Authorization` is managed via [setAuthToken](set-auth-token.md). |

## Functions

| Name | Summary |
|---|---|
| [delete](delete.md) | [jvm]<br>suspend fun [delete](delete.md)(path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyMap())<br>DELETE [path](delete.md). No deserialization (TS returns void). |
| [get](get.md) | [jvm]<br>inline suspend fun &lt;[T](get.md)&gt; [get](get.md)(path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyMap()): [T](get.md)<br>GET [path](get.md), deserialize the JSON body to [T](get.md). |
| [getWithHeaders](get-with-headers.md) | [jvm]<br>suspend fun [getWithHeaders](get-with-headers.md)(path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyMap()): [HttpResponse](../-http-response/index.md)<br>GET that also returns response headers — used by the query builder to parse `Content-Range` for count queries. Mirrors `getWithHeaders` in `fetch.ts`. |
| [patch](patch.md) | [jvm]<br>inline suspend fun &lt;[T](patch.md)&gt; [patch](patch.md)(path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), body: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)? = null, headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyMap()): [T](patch.md)<br>PATCH [body](patch.md) to [path](patch.md), deserialize the JSON response to [T](patch.md). |
| [post](post.md) | [jvm]<br>inline suspend fun &lt;[T](post.md)&gt; [post](post.md)(path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), body: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)? = null, headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyMap()): [T](post.md)<br>POST [body](post.md) to [path](post.md), deserialize the JSON response to [T](post.md). |
| [postWithHeaders](post-with-headers.md) | [jvm]<br>suspend fun [postWithHeaders](post-with-headers.md)(path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), body: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)? = null, headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyMap()): [HttpResponse](../-http-response/index.md)<br>POST that also returns response headers. Mirrors `postWithHeaders` in `fetch.ts`. |
| [put](put.md) | [jvm]<br>inline suspend fun &lt;[T](put.md)&gt; [put](put.md)(path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), body: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)? = null, headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyMap()): [T](put.md)<br>PUT [body](put.md) to [path](put.md), deserialize the JSON response to [T](put.md). |
| [removeHeader](remove-header.md) | [jvm]<br>fun [removeHeader](remove-header.md)(name: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html))<br>Remove a custom header. |
| [setAnonKey](set-anon-key.md) | [jvm]<br>fun [setAnonKey](set-anon-key.md)(key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html))<br>Set the anon key used as the Authorization fallback on sign-out. |
| [setAuthToken](set-auth-token.md) | [jvm]<br>fun [setAuthToken](set-auth-token.md)(token: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)?)<br>Update the authorization token. |
| [setHeader](set-header.md) | [jvm]<br>fun [setHeader](set-header.md)(name: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html))<br>Set a custom header on all subsequent requests (e.g. `X-FB-Tenant`). |
