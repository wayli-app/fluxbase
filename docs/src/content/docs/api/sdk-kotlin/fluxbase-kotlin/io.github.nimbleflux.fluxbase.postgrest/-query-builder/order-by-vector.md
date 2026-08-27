---
title: "orderByVector"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.postgrest](../../)/[QueryBuilder](../)/[orderByVector](./)

# orderByVector

[jvm]\
fun [orderByVector](./)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), vector: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[Double](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-double/index.html)&gt;, metric: [VectorMetric](../../-vector-metric/)): [QueryBuilder](../)&lt;[T](../)&gt;

Order by vector similarity. Adds a vector order clause. Port of `orderByVector()` in `query-builder.ts:500`.
