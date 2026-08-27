---
title: "FluxbaseFunctions"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.functions](../)/[FluxbaseFunctions](./)

# FluxbaseFunctions

[jvm]\
class [FluxbaseFunctions](./)(http: [FluxbaseHttpClient](../../iogithubnimblefluxfluxbasecore/-fluxbase-http-client/))

Edge Functions module — port of `FluxbaseFunctions` from `sdk/src/functions.ts`.

Invokes Fluxbase edge functions via `POST /api/v1/functions/{name}/invoke`.

Usage:

```kotlin
val result = client.functions.invoke<MyData>("my-fn", body = mapOf("key" to "value"), namespace = "wayli")
```

## Constructors

| | |
|---|---|
| [FluxbaseFunctions](-fluxbase-functions/) | [jvm]<br>constructor(http: [FluxbaseHttpClient](../../iogithubnimblefluxfluxbasecore/-fluxbase-http-client/)) |

## Functions

| Name | Summary |
|---|---|
| [invoke](invoke/) | [jvm]<br>inline suspend fun &lt;[T](invoke/)&gt; [invoke](invoke/)(name: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), body: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)? = null, method: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;POST&quot;, namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyMap()): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[T](invoke/)&gt;<br>Invoke an edge function. Port of `invoke()` in `functions.ts:77`. |
| [invokeJson](invoke-json/) | [jvm]<br>suspend fun [invokeJson](invoke-json/)(name: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), body: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)? = null, method: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;POST&quot;, namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;JsonElement&gt;<br>Invoke a function returning raw JSON (for untyped responses). |
