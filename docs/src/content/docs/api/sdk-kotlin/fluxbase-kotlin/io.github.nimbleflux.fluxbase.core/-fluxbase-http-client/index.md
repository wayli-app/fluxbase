---
title: "FluxbaseHttpClient"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.core](../)/[FluxbaseHttpClient](./)

# FluxbaseHttpClient

class [FluxbaseHttpClient](./)(baseUrl: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), transport: [HttpTransport](../-http-transport/), json: Json = defaultJson)

The shared HTTP client that every Fluxbase module (auth, postgrest, realtime, storage, jobs, …) uses to talk to the Fluxbase server. This is the Kotlin port of `FluxbaseFetch` (`sdk/src/fetch.ts`).

Responsibilities:

- 
   Hold the base URL (trailing slash stripped) and default headers.
- 
   Manage the auth token: `setAuthToken(token)` sets `Authorization: Bearer`;     `setAuthToken(null)` restores the anon key fallback (does NOT delete the     header — matches TS `fetch.ts:93-102`).
- 
   Provide convenience methods (`get`, `post`, `put`, `patch`, `delete`) that     return reified typed results via kotlinx.serialization.
- 
   Delegate the actual I/O to an [HttpTransport](../-http-transport/) SPI, which makes the client     trivially testable with a recording fake.

401 auto-refresh-retry (port of `fetch.ts`'s single-retry-after-refresh): When [setRefreshTokenCallback](set-refresh-token-callback/) has been wired (the client wires it to `auth.refreshSession()`), any request that fails with HTTP 401 triggers a single token refresh, then the request is retried exactly once with the new token. Concurrent 401s are deduped via refreshMutex so only one refresh fires even when many requests fail simultaneously. A second 401 (refresh didn't help) is propagated to the caller — there is no retry loop.

#### Parameters

jvm

| | |
|---|---|
| baseUrl | the Fluxbase server URL (trailing slash stripped). |
| transport | the I/O SPI. If null, a Ktor-backed transport is used at runtime; tests inject an io.github.nimbleflux.fluxbase.core.test.RecordingHttp. |
| json | the JSON decoder used for typed [get](get/)/[post](post/) responses. |

## Constructors

| | |
|---|---|
| [FluxbaseHttpClient](-fluxbase-http-client/) | [jvm]<br>constructor(baseUrl: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), transport: [HttpTransport](../-http-transport/), json: Json = defaultJson) |

## Types

| Name | Summary |
|---|---|
| [Companion](-companion/) | [jvm]<br>object [Companion](-companion/) |

## Properties

| Name | Summary |
|---|---|
| [baseUrl](base-url/) | [jvm]<br>val [baseUrl](base-url/): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)<br>The base URL with any trailing slash removed. |
| [defaultHeaders](default-headers/) | [jvm]<br>val [defaultHeaders](default-headers/): [MutableMap](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-mutable-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt;<br>Default headers applied to every request. `Content-Type: application/json` is always present; `Authorization` is managed via [setAuthToken](set-auth-token/). |

## Functions

| Name | Summary |
|---|---|
| [delete](delete/) | [jvm]<br>suspend fun [delete](delete/)(path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyMap())<br>DELETE [path](delete/). No deserialization (TS returns void). |
| [get](get/) | [jvm]<br>inline suspend fun &lt;[T](get/)&gt; [get](get/)(path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyMap()): [T](get/)<br>GET [path](get/), deserialize the JSON body to [T](get/). |
| [getBytes](get-bytes/) | [jvm]<br>suspend fun [getBytes](get-bytes/)(path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyMap()): [ByteArray](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-byte-array/index.html)<br>GET [path](get-bytes/) and return the response body as raw bytes — the binary-safe path (no charset decode). Used by storage downloads and any other non-text payload. Mirrors `getBlob` in `sdk/src/fetch.ts`. |
| [getWithHeaders](get-with-headers/) | [jvm]<br>suspend fun [getWithHeaders](get-with-headers/)(path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyMap()): [HttpResponse](../-http-response/)<br>GET that also returns response headers — used by the query builder to parse `Content-Range` for count queries. Mirrors `getWithHeaders` in `fetch.ts`. |
| [patch](patch/) | [jvm]<br>inline suspend fun &lt;[T](patch/)&gt; [patch](patch/)(path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), body: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)? = null, headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyMap()): [T](patch/)<br>PATCH [body](patch/) to [path](patch/), deserialize the JSON response to [T](patch/). |
| [post](post/) | [jvm]<br>inline suspend fun &lt;[T](post/)&gt; [post](post/)(path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), body: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)? = null, headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyMap()): [T](post/)<br>POST [body](post/) to [path](post/), deserialize the JSON response to [T](post/). |
| [postWithHeaders](post-with-headers/) | [jvm]<br>suspend fun [postWithHeaders](post-with-headers/)(path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), body: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)? = null, headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyMap()): [HttpResponse](../-http-response/)<br>POST that also returns response headers. Mirrors `postWithHeaders` in `fetch.ts`. |
| [put](put/) | [jvm]<br>inline suspend fun &lt;[T](put/)&gt; [put](put/)(path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), body: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)? = null, headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyMap()): [T](put/)<br>PUT [body](put/) to [path](put/), deserialize the JSON response to [T](put/). |
| [removeHeader](remove-header/) | [jvm]<br>fun [removeHeader](remove-header/)(name: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html))<br>Remove a custom header. |
| [setAnonKey](set-anon-key/) | [jvm]<br>fun [setAnonKey](set-anon-key/)(key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html))<br>Set the anon key used as the Authorization fallback on sign-out. |
| [setAuthToken](set-auth-token/) | [jvm]<br>fun [setAuthToken](set-auth-token/)(token: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)?)<br>Update the authorization token. |
| [setHeader](set-header/) | [jvm]<br>fun [setHeader](set-header/)(name: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html))<br>Set a custom header on all subsequent requests (e.g. `X-FB-Tenant`). |
| [setRefreshTokenCallback](set-refresh-token-callback/) | [jvm]<br>fun [setRefreshTokenCallback](set-refresh-token-callback/)(callback: suspend () -&gt; [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)?)<br>Register the callback that refreshes the access token. On a 401 the client invokes this once (deduped across concurrent requests), applies the returned token via [setAuthToken](set-auth-token/), and retries the original request a single time. Mirrors TS `setRefreshTokenCallback` in `fetch.ts`. |
