---
title: "Package-level declarations"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../)/[io.github.nimbleflux.fluxbase.postgrest](./)

# Package-level declarations

## Types

| Name | Summary |
|---|---|
| [CountType](-count-type/) | [jvm]<br>enum [CountType](-count-type/) : [Enum](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-enum/index.html)&lt;[CountType](-count-type/)&gt; <br>Count mode for PostgREST queries. Port of `CountType` from `sdk/src/types.ts:271`. |
| [PostgrestResponse](-postgrest-response/) | [jvm]<br>data class [PostgrestResponse](-postgrest-response/)&lt;[T](-postgrest-response/)&gt;(val data: [T](-postgrest-response/)?, val error: [FluxbaseError](../iogithubnimblefluxfluxbase/-fluxbase-error/)?, val count: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)?, val status: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html), val statusText: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html))<br>Result of a PostgREST query. Port of `PostgrestResponse<T>` from `sdk/src/types.ts:257`. |
| [QueryBuilder](-query-builder/) | [jvm]<br>class [QueryBuilder](-query-builder/)&lt;[T](-query-builder/)&gt;<br>A chainable PostgREST query builder. Port of `QueryBuilder<T>` from `sdk/src/query-builder.ts`. |
| [VectorMetric](-vector-metric/) | [jvm]<br>enum [VectorMetric](-vector-metric/) : [Enum](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-enum/index.html)&lt;[VectorMetric](-vector-metric/)&gt; <br>Vector similarity metric for pgvector queries. Port of `VectorMetric` from `sdk/src/query-builder.ts:507`. |
