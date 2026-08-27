---
title: "KtorHttpTransport"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.core](../)/[KtorHttpTransport](./)

# KtorHttpTransport

[jvm]\
class [KtorHttpTransport](./)(baseUrl: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), json: Json = FluxbaseHttpClient.defaultJson, timeoutMillis: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)) : [HttpTransport](../-http-transport/)

Ktor-backed [HttpTransport](../-http-transport/) — the production HTTP implementation for JVM/Android.

This is the equivalent of the TS SDK's `FluxbaseFetch` class which wraps `global.fetch()`. It handles the actual TCP I/O, JSON serialization of the request body, and status-code-to-exception conversion.

Two response paths:

- 
   [request](request/) returns the body as text (for JSON APIs).
- 
   [requestBytes](request-bytes/) returns the body as raw bytes (binary-safe, for storage     downloads and any other non-text payload) — the Kotlin analogue of the     TS `getBlob`. Bytes never pass through a charset decode, so images and     other non-UTF-8 payloads survive intact.

Request bodies:

- 
   [ByteArray](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-byte-array/index.html) bodies are sent raw (binary-safe, used by storage upload).
- 
   Anything else is JSON-encoded via encodeToJsonElement, matching how the     TS SDK passes plain JS objects through `JSON.stringify`.

The full port (S1) will add: 30s timeout, 401 auto-refresh+retry (single shared refresh, deduped across concurrent requests). For the S0 spike this minimal version proves the wire contract end-to-end.

## Constructors

| | |
|---|---|
| [KtorHttpTransport](-ktor-http-transport/) | [jvm]<br>constructor(baseUrl: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), json: Json = FluxbaseHttpClient.defaultJson, timeoutMillis: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)) |

## Functions

| Name | Summary |
|---|---|
| [request](request/) | [jvm]<br>open suspend override fun [request](request/)(method: [HttpMethod](../-http-method/), path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), body: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?, headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt;): [HttpResponse](../-http-response/)<br>Perform an HTTP [method](request/) request to [path](request/) (relative to the base URL). [body](request/) is a pre-serialized value (will be JSON-encoded by the transport) or null for GET/DELETE/HEAD. [headers](request/) are per-request overrides merged on top of the client defaults. Returns the raw response body as text. |
| [requestBytes](request-bytes/) | [jvm]<br>open suspend override fun [requestBytes](request-bytes/)(method: [HttpMethod](../-http-method/), path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt;): [ByteArray](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-byte-array/index.html)<br>Perform an HTTP [method](request-bytes/) request and return the response body as raw bytes — the binary-safe path used by [FluxbaseHttpClient.getBytes](../-fluxbase-http-client/get-bytes/) (e.g. storage downloads). Unlike [request](request/), the body never passes through a text/charset decode, so non-UTF-8 bytes (images, archives) survive intact. Mirrors the TS SDK's `getBlob` in `sdk/src/fetch.ts`. |
