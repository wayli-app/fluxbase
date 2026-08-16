---
title: "getWithHeaders"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.core](../index.md)/[FluxbaseHttpClient](index.md)/[getWithHeaders](get-with-headers.md)

# getWithHeaders

[jvm]\
suspend fun [getWithHeaders](get-with-headers.md)(path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyMap()): [HttpResponse](../-http-response/index.md)

GET that also returns response headers — used by the query builder to parse `Content-Range` for count queries. Mirrors `getWithHeaders` in `fetch.ts`.
