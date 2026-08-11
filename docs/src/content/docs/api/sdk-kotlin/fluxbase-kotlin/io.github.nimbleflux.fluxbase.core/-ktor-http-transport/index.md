---
title: "KtorHttpTransport"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.core](../index.md)/[KtorHttpTransport](index.md)

# KtorHttpTransport

[jvm]\
class [KtorHttpTransport](index.md)(baseUrl: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), json: Json = FluxbaseHttpClient.defaultJson, timeoutMillis: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)) : [HttpTransport](../-http-transport/index.md)

Ktor-backed [HttpTransport](../-http-transport/index.md) — the production HTTP implementation for JVM/Android.

This is the equivalent of the TS SDK's `FluxbaseFetch` class which wraps `global.fetch()`. It handles the actual TCP I/O, JSON serialization of the request body, and status-code-to-exception conversion.

Two response paths:

- 
   [request](request.md) returns the body as text (for JSON APIs).
- 
   [requestBytes](request-bytes.md) returns the body as raw bytes (binary-safe, for storage     downloads and any other non-text payload) — the Kotlin analogue of the     TS `getBlob`. Bytes never pass through a charset decode, so images and     other non-UTF-8 payloads survive intact.

Request bodies:

- 
   [ByteArray](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-byte-array/index.html) bodies are sent raw (binary-safe, used by storage upload).
- 
   Anything else is JSON-encoded via encodeToJsonElement, matching how the     TS SDK passes plain JS objects through `JSON.stringify`.

The full port (S1) will add: 30s timeout, 401 auto-refresh+retry (single shared refresh, deduped across concurrent requests). For the S0 spike this minimal version proves the wire contract end-to-end.

## Constructors

| | |
|---|---|
| [KtorHttpTransport](-ktor-http-transport.md) | [jvm]<br>constructor(baseUrl: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), json: Json = FluxbaseHttpClient.defaultJson, timeoutMillis: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)) |

## Functions

| Name | Summary |
|---|---|
| [request](request.md) | [jvm]<br>open suspend override fun [request](request.md)(method: [HttpMethod](../-http-method/index.md), path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), body: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?, headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt;): [HttpResponse](../-http-response/index.md)<br>Perform an HTTP [method](request.md) request to [path](request.md) (relative to the base URL). [body](request.md) is a pre-serialized value (will be JSON-encoded by the transport) or null for GET/DELETE/HEAD. [headers](request.md) are per-request overrides merged on top of the client defaults. Returns the raw response body as text. |
| [requestBytes](request-bytes.md) | [jvm]<br>open suspend override fun [requestBytes](request-bytes.md)(method: [HttpMethod](../-http-method/index.md), path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt;): [ByteArray](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-byte-array/index.html)<br>Perform an HTTP [method](request-bytes.md) request and return the response body as raw bytes — the binary-safe path used by [FluxbaseHttpClient.getBytes](../-fluxbase-http-client/get-bytes.md) (e.g. storage downloads). Unlike [request](request.md), the body never passes through a text/charset decode, so non-UTF-8 bytes (images, archives) survive intact. Mirrors the TS SDK's `getBlob` in `sdk/src/fetch.ts`. |
