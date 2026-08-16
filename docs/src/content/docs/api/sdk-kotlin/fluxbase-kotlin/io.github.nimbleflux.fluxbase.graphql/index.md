---
title: "Package-level declarations"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../index.md)/[io.github.nimbleflux.fluxbase.graphql](index.md)

# Package-level declarations

## Types

| Name | Summary |
|---|---|
| [FluxbaseGraphQL](-fluxbase-graph-q-l/index.md) | [jvm]<br>class [FluxbaseGraphQL](-fluxbase-graph-q-l/index.md)(http: [FluxbaseHttpClient](../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md))<br>GraphQL module — port of `FluxbaseGraphQL` from `sdk/src/graphql.ts`. |
| [GraphQLError](-graph-q-l-error/index.md) | [jvm]<br>@Serializable<br>data class [GraphQLError](-graph-q-l-error/index.md)(val message: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val locations: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[GraphQLErrorLocation](-graph-q-l-error-location/index.md)&gt;? = null, val path: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;JsonElement&gt;? = null)<br>A GraphQL error. Port of `GraphQLError` from `sdk/src/types.ts`. |
| [GraphQLErrorLocation](-graph-q-l-error-location/index.md) | [jvm]<br>@Serializable<br>data class [GraphQLErrorLocation](-graph-q-l-error-location/index.md)(val line: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html), val column: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)) |
| [GraphQLResponse](-graph-q-l-response/index.md) | [jvm]<br>@Serializable<br>data class [GraphQLResponse](-graph-q-l-response/index.md)&lt;[T](-graph-q-l-response/index.md)&gt;(val data: [T](-graph-q-l-response/index.md)? = null, val errors: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[GraphQLError](-graph-q-l-error/index.md)&gt;? = null)<br>A GraphQL response. Port of `GraphQLResponse<T>` from `sdk/src/types.ts`. |
