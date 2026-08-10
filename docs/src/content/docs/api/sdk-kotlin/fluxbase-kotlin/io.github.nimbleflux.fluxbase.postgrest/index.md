//[fluxbase-kotlin](../../index.md)/[io.github.nimbleflux.fluxbase.postgrest](index.md)

# Package-level declarations

## Types

| Name | Summary |
|---|---|
| [CountType](-count-type/index.md) | [jvm]<br>enum [CountType](-count-type/index.md) : [Enum](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-enum/index.html)&lt;[CountType](-count-type/index.md)&gt; <br>Count mode for PostgREST queries. Port of `CountType` from `sdk/src/types.ts:271`. |
| [PostgrestResponse](-postgrest-response/index.md) | [jvm]<br>data class [PostgrestResponse](-postgrest-response/index.md)&lt;[T](-postgrest-response/index.md)&gt;(val data: [T](-postgrest-response/index.md)?, val error: [FluxbaseError](../io.github.nimbleflux.fluxbase/-fluxbase-error/index.md)?, val count: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)?, val status: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html), val statusText: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html))<br>Result of a PostgREST query. Port of `PostgrestResponse<T>` from `sdk/src/types.ts:257`. |
| [QueryBuilder](-query-builder/index.md) | [jvm]<br>class [QueryBuilder](-query-builder/index.md)&lt;[T](-query-builder/index.md)&gt;<br>A chainable PostgREST query builder. Port of `QueryBuilder<T>` from `sdk/src/query-builder.ts`. |
| [VectorMetric](-vector-metric/index.md) | [jvm]<br>enum [VectorMetric](-vector-metric/index.md) : [Enum](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-enum/index.html)&lt;[VectorMetric](-vector-metric/index.md)&gt; <br>Vector similarity metric for pgvector queries. Port of `VectorMetric` from `sdk/src/query-builder.ts:507`. |
