---
title: "vectorSearch"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.postgrest](../../)/[QueryBuilder](../)/[vectorSearch](./)

# vectorSearch

[jvm]\
fun [vectorSearch](./)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), vector: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[Double](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-double/index.html)&gt;, metric: [VectorMetric](../../-vector-metric/) = VectorMetric.COSINE): [QueryBuilder](../)&lt;[T](../)&gt;

Vector similarity search — combines an order-by-distance + a distance filter. Port of `vectorSearch()` in `query-builder.ts:540`.
