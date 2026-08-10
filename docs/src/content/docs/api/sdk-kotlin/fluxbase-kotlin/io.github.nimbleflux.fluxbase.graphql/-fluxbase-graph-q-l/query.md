//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.graphql](../index.md)/[FluxbaseGraphQL](index.md)/[query](query.md)

# query

[jvm]\
inline suspend fun &lt;[T](query.md)&gt; [query](query.md)(query: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), variables: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?&gt;? = null, headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyMap()): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[GraphQLResponse](../-graph-q-l-response/index.md)&lt;[T](query.md)&gt;&gt;

Execute a GraphQL query. POSTs `{query, variables}` to `/api/v1/graphql`. Port of `query()` / `execute()` in `graphql.ts`.
