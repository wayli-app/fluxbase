---
title: "requestBytes"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.core](../../)/[HttpTransport](../)/[requestBytes](./)

# requestBytes

[jvm]\
abstract suspend fun [requestBytes](./)(method: [HttpMethod](../../-http-method/), path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt;): [ByteArray](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-byte-array/index.html)

Perform an HTTP [method](./) request and return the response body as raw bytes — the binary-safe path used by [FluxbaseHttpClient.getBytes](../../-fluxbase-http-client/get-bytes/) (e.g. storage downloads). Unlike [request](../request/), the body never passes through a text/charset decode, so non-UTF-8 bytes (images, archives) survive intact. Mirrors the TS SDK's `getBlob` in `sdk/src/fetch.ts`.
