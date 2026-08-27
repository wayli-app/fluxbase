---
title: "Package-level declarations"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../)/[io.github.nimbleflux.fluxbase.graphql](./)

# Package-level declarations

## Types

| Name | Summary |
|---|---|
| [FluxbaseGraphQL](-fluxbase-graph-q-l/) | [jvm]<br>class [FluxbaseGraphQL](-fluxbase-graph-q-l/)(http: [FluxbaseHttpClient](../iogithubnimblefluxfluxbasecore/-fluxbase-http-client/))<br>GraphQL module — port of `FluxbaseGraphQL` from `sdk/src/graphql.ts`. |
| [GraphQLError](-graph-q-l-error/) | [jvm]<br>@Serializable<br>data class [GraphQLError](-graph-q-l-error/)(val message: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val locations: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[GraphQLErrorLocation](-graph-q-l-error-location/)&gt;? = null, val path: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;JsonElement&gt;? = null)<br>A GraphQL error. Port of `GraphQLError` from `sdk/src/types.ts`. |
| [GraphQLErrorLocation](-graph-q-l-error-location/) | [jvm]<br>@Serializable<br>data class [GraphQLErrorLocation](-graph-q-l-error-location/)(val line: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html), val column: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)) |
| [GraphQLResponse](-graph-q-l-response/) | [jvm]<br>@Serializable<br>data class [GraphQLResponse](-graph-q-l-response/)&lt;[T](-graph-q-l-response/)&gt;(val data: [T](-graph-q-l-response/)? = null, val errors: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[GraphQLError](-graph-q-l-error/)&gt;? = null)<br>A GraphQL response. Port of `GraphQLResponse<T>` from `sdk/src/types.ts`. |
