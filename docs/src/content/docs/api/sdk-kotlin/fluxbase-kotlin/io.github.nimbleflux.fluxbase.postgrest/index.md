//[fluxbase-kotlin](../../index.md)/[io.github.nimbleflux.fluxbase.postgrest](index.md)

# Package-level declarations

## Types

| Name | Summary |
|---|---|
| [PostgrestResponse](-postgrest-response/index.md) | [jvm]<br>data class [PostgrestResponse](-postgrest-response/index.md)&lt;[T](-postgrest-response/index.md)&gt;(val data: [T](-postgrest-response/index.md)?, val error: [FluxbaseError](../io.github.nimbleflux.fluxbase/-fluxbase-error/index.md)?, val count: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)?, val status: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html), val statusText: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html))<br>Result of a PostgREST query. Port of `PostgrestResponse<T>` from `sdk/src/types.ts:257`. |
| [QueryBuilder](-query-builder/index.md) | [jvm]<br>class [QueryBuilder](-query-builder/index.md)&lt;[T](-query-builder/index.md)&gt;<br>A chainable PostgREST query builder. Port of `QueryBuilder<T>` from `sdk/src/query-builder.ts`. |
