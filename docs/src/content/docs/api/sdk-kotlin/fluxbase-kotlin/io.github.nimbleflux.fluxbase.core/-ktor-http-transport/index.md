//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.core](../index.md)/[KtorHttpTransport](index.md)

# KtorHttpTransport

[jvm]\
class [KtorHttpTransport](index.md)(baseUrl: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), json: Json = FluxbaseHttpClient.defaultJson) : [HttpTransport](../-http-transport/index.md)

Ktor-backed [HttpTransport](../-http-transport/index.md) — the production HTTP implementation for JVM/Android.

This is the equivalent of the TS SDK's `FluxbaseFetch` class which wraps `global.fetch()`. It handles the actual TCP I/O, JSON serialization of the request body, and status-code-to-exception conversion.

The full port (S1) will add: 30s timeout, 401 auto-refresh+retry (single shared refresh, deduped across concurrent requests), `getBlob` for file downloads, and a `beforeRequest` header-mutation hook. For the S0 spike this minimal version proves the wire contract end-to-end.

NOTE: the request body arrives as an arbitrary Kotlin object (typically a Map or List built by the SDK's method callers, mirroring how the TS SDK passes plain JS objects). We serialize it to JSON via encodeBody.

## Constructors

| | |
|---|---|
| [KtorHttpTransport](-ktor-http-transport.md) | [jvm]<br>constructor(baseUrl: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), json: Json = FluxbaseHttpClient.defaultJson) |

## Functions

| Name | Summary |
|---|---|
| [request](request.md) | [jvm]<br>open suspend override fun [request](request.md)(method: [HttpMethod](../-http-method/index.md), path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), body: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?, headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt;): [HttpResponse](../-http-response/index.md)<br>Perform an HTTP [method](request.md) request to [path](request.md) (relative to the base URL). [body](request.md) is a pre-serialized value (will be JSON-encoded by the transport) or null for GET/DELETE/HEAD. [headers](request.md) are per-request overrides merged on top of the client defaults. Returns the raw response body as text. |
