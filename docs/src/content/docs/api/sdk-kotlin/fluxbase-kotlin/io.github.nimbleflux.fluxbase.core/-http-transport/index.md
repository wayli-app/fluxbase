---
title: "HttpTransport"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.core](../index.md)/[HttpTransport](index.md)

# HttpTransport

interface [HttpTransport](index.md)

SPI for the actual HTTP I/O. The SDK's [FluxbaseHttpClient](../-fluxbase-http-client/index.md) delegates all wire calls to this interface, which has a production implementation (Ktor-backed, for JVM/Android) and is the seam used by tests: io.github.nimbleflux.fluxbase.core.test.RecordingHttp is a fake that records requests instead of sending them.

This mirrors the TS SDK's separation where `FluxbaseFetch` wraps `global.fetch()` and tests inject a fake object with the same method shape.

#### Inheritors

| |
|---|
| [KtorHttpTransport](../-ktor-http-transport/index.md) |

## Functions

| Name | Summary |
|---|---|
| [request](request.md) | [jvm]<br>abstract suspend fun [request](request.md)(method: [HttpMethod](../-http-method/index.md), path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), body: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?, headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt;): [HttpResponse](../-http-response/index.md)<br>Perform an HTTP [method](request.md) request to [path](request.md) (relative to the base URL). [body](request.md) is a pre-serialized value (will be JSON-encoded by the transport) or null for GET/DELETE/HEAD. [headers](request.md) are per-request overrides merged on top of the client defaults. Returns the raw response body as text. |
| [requestBytes](request-bytes.md) | [jvm]<br>abstract suspend fun [requestBytes](request-bytes.md)(method: [HttpMethod](../-http-method/index.md), path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt;): [ByteArray](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-byte-array/index.html)<br>Perform an HTTP [method](request-bytes.md) request and return the response body as raw bytes — the binary-safe path used by [FluxbaseHttpClient.getBytes](../-fluxbase-http-client/get-bytes.md) (e.g. storage downloads). Unlike [request](request.md), the body never passes through a text/charset decode, so non-UTF-8 bytes (images, archives) survive intact. Mirrors the TS SDK's `getBlob` in `sdk/src/fetch.ts`. |
