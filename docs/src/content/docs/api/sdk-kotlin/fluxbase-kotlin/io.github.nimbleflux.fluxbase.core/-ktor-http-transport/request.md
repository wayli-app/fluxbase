//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.core](../index.md)/[KtorHttpTransport](index.md)/[request](request.md)

# request

[jvm]\
open suspend override fun [request](request.md)(method: [HttpMethod](../-http-method/index.md), path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), body: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?, headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt;): [HttpResponse](../-http-response/index.md)

Perform an HTTP [method](request.md) request to [path](request.md) (relative to the base URL). [body](request.md) is a pre-serialized value (will be JSON-encoded by the transport) or null for GET/DELETE/HEAD. [headers](request.md) are per-request overrides merged on top of the client defaults. Returns the raw response body as text.
