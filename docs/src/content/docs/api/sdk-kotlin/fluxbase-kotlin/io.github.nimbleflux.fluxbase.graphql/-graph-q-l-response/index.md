---
title: "GraphQLResponse"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.graphql](../)/[GraphQLResponse](./)

# GraphQLResponse

[jvm]\
@Serializable

data class [GraphQLResponse](./)&lt;[T](./)&gt;(val data: [T](./)? = null, val errors: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[GraphQLError](../-graph-q-l-error/)&gt;? = null)

A GraphQL response. Port of `GraphQLResponse<T>` from `sdk/src/types.ts`.

## Constructors

| | |
|---|---|
| [GraphQLResponse](-graph-q-l-response/) | [jvm]<br>constructor(data: [T](./)? = null, errors: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[GraphQLError](../-graph-q-l-error/)&gt;? = null) |

## Properties

| Name | Summary |
|---|---|
| [data](data/) | [jvm]<br>val [data](data/): [T](./)? = null |
| [errors](errors/) | [jvm]<br>val [errors](errors/): [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[GraphQLError](../-graph-q-l-error/)&gt;? = null |
