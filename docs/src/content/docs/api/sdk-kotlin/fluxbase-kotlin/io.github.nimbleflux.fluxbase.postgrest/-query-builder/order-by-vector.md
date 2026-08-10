//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.postgrest](../index.md)/[QueryBuilder](index.md)/[orderByVector](order-by-vector.md)

# orderByVector

[jvm]\
fun [orderByVector](order-by-vector.md)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), vector: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[Double](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-double/index.html)&gt;, metric: [VectorMetric](../-vector-metric/index.md)): [QueryBuilder](index.md)&lt;[T](index.md)&gt;

Order by vector similarity. Adds a vector order clause. Port of `orderByVector()` in `query-builder.ts:500`.
