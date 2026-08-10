---
title: "Package-level declarations"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../index.md)/[io.github.nimbleflux.fluxbase.core](index.md)

# Package-level declarations

## Types

| Name | Summary |
|---|---|
| [FluxbaseException](-fluxbase-exception/index.md) | [jvm]<br>class [FluxbaseException](-fluxbase-exception/index.md)(val status: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html), val code: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val details: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)? = null, message: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)) : [RuntimeException](https://docs.oracle.com/javase/8/docs/api/java/lang/RuntimeException.html)<br>A Fluxbase API error. Mirrors `FluxbaseError` from `sdk/src/types.ts:235`. |
| [FluxbaseHttpClient](-fluxbase-http-client/index.md) | [jvm]<br>class [FluxbaseHttpClient](-fluxbase-http-client/index.md)(baseUrl: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), transport: [HttpTransport](-http-transport/index.md), json: Json = defaultJson)<br>The shared HTTP client that every Fluxbase module (auth, postgrest, realtime, storage, jobs, …) uses to talk to the Fluxbase server. This is the Kotlin port of `FluxbaseFetch` (`sdk/src/fetch.ts`). |
| [HttpMethod](-http-method/index.md) | [jvm]<br>enum [HttpMethod](-http-method/index.md) : [Enum](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-enum/index.html)&lt;[HttpMethod](-http-method/index.md)&gt; <br>HTTP method enum used by the SDK's client layer. (Distinct from Ktor's `HttpMethod` to avoid naming clashes in the transport.) |
| [HttpResponse](-http-response/index.md) | [jvm]<br>data class [HttpResponse](-http-response/index.md)(val body: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val status: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html), val headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt;)<br>A raw HTTP response: the body as text plus the status code and response headers. |
| [HttpTransport](-http-transport/index.md) | [jvm]<br>fun interface [HttpTransport](-http-transport/index.md)<br>SPI for the actual HTTP I/O. The SDK's [FluxbaseHttpClient](-fluxbase-http-client/index.md) delegates all wire calls to this interface, which has a production implementation (Ktor-backed, for JVM/Android) and is the seam used by tests: io.github.nimbleflux.fluxbase.core.test.RecordingHttp is a fake that records requests instead of sending them. |
| [KtorHttpTransport](-ktor-http-transport/index.md) | [jvm]<br>class [KtorHttpTransport](-ktor-http-transport/index.md)(baseUrl: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), json: Json = FluxbaseHttpClient.defaultJson) : [HttpTransport](-http-transport/index.md)<br>Ktor-backed [HttpTransport](-http-transport/index.md) — the production HTTP implementation for JVM/Android. |
