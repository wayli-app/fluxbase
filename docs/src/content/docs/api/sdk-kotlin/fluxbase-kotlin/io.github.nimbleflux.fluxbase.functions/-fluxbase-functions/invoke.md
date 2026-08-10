---
title: "invoke"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.functions](../index.md)/[FluxbaseFunctions](index.md)/[invoke](invoke.md)

# invoke

[jvm]\
inline suspend fun &lt;[T](invoke.md)&gt; [invoke](invoke.md)(name: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), body: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)? = null, method: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;POST&quot;, namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyMap()): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[T](invoke.md)&gt;

Invoke an edge function. Port of `invoke()` in `functions.ts:77`.

#### Parameters

jvm

| | |
|---|---|
| name | the function name. |
| body | the request body (any JSON-serializable value, or null for GET/DELETE). |
| method | the HTTP method (default POST). |
| namespace | optional Fluxbase namespace (added as `?namespace=` query param). |
| headers | per-request headers. |
