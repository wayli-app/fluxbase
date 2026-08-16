---
title: "vectorSearch"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.postgrest](../index.md)/[QueryBuilder](index.md)/[vectorSearch](vector-search.md)

# vectorSearch

[jvm]\
fun [vectorSearch](vector-search.md)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), vector: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[Double](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-double/index.html)&gt;, metric: [VectorMetric](../-vector-metric/index.md) = VectorMetric.COSINE): [QueryBuilder](index.md)&lt;[T](index.md)&gt;

Vector similarity search — combines an order-by-distance + a distance filter. Port of `vectorSearch()` in `query-builder.ts:540`.
