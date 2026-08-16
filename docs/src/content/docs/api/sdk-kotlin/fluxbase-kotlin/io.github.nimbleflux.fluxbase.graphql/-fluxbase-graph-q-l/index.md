---
title: "FluxbaseGraphQL"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.graphql](../index.md)/[FluxbaseGraphQL](index.md)

# FluxbaseGraphQL

[jvm]\
class [FluxbaseGraphQL](index.md)(http: [FluxbaseHttpClient](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md))

GraphQL module — port of `FluxbaseGraphQL` from `sdk/src/graphql.ts`.

All requests go to `POST /api/v1/graphql`.

Usage:

```kotlin
val (result, _) = client.graphql.query<MyData>("{ trips { id title } }")
```

## Constructors

| | |
|---|---|
| [FluxbaseGraphQL](-fluxbase-graph-q-l.md) | [jvm]<br>constructor(http: [FluxbaseHttpClient](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md)) |

## Functions

| Name | Summary |
|---|---|
| [mutation](mutation.md) | [jvm]<br>inline suspend fun &lt;[T](mutation.md)&gt; [mutation](mutation.md)(mutation: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), variables: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?&gt;? = null, headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyMap()): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[GraphQLResponse](../-graph-q-l-response/index.md)&lt;[T](mutation.md)&gt;&gt;<br>Alias for [query](query.md) — semantically marks the operation as a mutation. |
| [query](query.md) | [jvm]<br>inline suspend fun &lt;[T](query.md)&gt; [query](query.md)(query: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), variables: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?&gt;? = null, headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyMap()): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[GraphQLResponse](../-graph-q-l-response/index.md)&lt;[T](query.md)&gt;&gt;<br>Execute a GraphQL query. POSTs `{query, variables}` to `/api/v1/graphql`. Port of `query()` / `execute()` in `graphql.ts`. |
