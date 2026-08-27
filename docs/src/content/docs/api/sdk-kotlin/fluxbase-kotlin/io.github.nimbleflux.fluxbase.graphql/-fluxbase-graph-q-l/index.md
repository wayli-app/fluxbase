---
title: "FluxbaseGraphQL"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.graphql](../)/[FluxbaseGraphQL](./)

# FluxbaseGraphQL

[jvm]\
class [FluxbaseGraphQL](./)(http: [FluxbaseHttpClient](../../iogithubnimblefluxfluxbasecore/-fluxbase-http-client/))

GraphQL module — port of `FluxbaseGraphQL` from `sdk/src/graphql.ts`.

All requests go to `POST /api/v1/graphql`.

Usage:

```kotlin
val (result, _) = client.graphql.query<MyData>("{ trips { id title } }")
```

## Constructors

| | |
|---|---|
| [FluxbaseGraphQL](-fluxbase-graph-q-l/) | [jvm]<br>constructor(http: [FluxbaseHttpClient](../../iogithubnimblefluxfluxbasecore/-fluxbase-http-client/)) |

## Functions

| Name | Summary |
|---|---|
| [mutation](mutation/) | [jvm]<br>inline suspend fun &lt;[T](mutation/)&gt; [mutation](mutation/)(mutation: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), variables: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?&gt;? = null, headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyMap()): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[GraphQLResponse](../-graph-q-l-response/)&lt;[T](mutation/)&gt;&gt;<br>Alias for [query](query/) — semantically marks the operation as a mutation. |
| [query](query/) | [jvm]<br>inline suspend fun &lt;[T](query/)&gt; [query](query/)(query: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), variables: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?&gt;? = null, headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyMap()): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[GraphQLResponse](../-graph-q-l-response/)&lt;[T](query/)&gt;&gt;<br>Execute a GraphQL query. POSTs `{query, variables}` to `/api/v1/graphql`. Port of `query()` / `execute()` in `graphql.ts`. |
