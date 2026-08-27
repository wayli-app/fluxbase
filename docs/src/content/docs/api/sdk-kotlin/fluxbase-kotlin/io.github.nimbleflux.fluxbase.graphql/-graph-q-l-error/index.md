---
title: "GraphQLError"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.graphql](../)/[GraphQLError](./)

# GraphQLError

[jvm]\
@Serializable

data class [GraphQLError](./)(val message: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val locations: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[GraphQLErrorLocation](../-graph-q-l-error-location/)&gt;? = null, val path: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;JsonElement&gt;? = null)

A GraphQL error. Port of `GraphQLError` from `sdk/src/types.ts`.

## Constructors

| | |
|---|---|
| [GraphQLError](-graph-q-l-error/) | [jvm]<br>constructor(message: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), locations: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[GraphQLErrorLocation](../-graph-q-l-error-location/)&gt;? = null, path: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;JsonElement&gt;? = null) |

## Properties

| Name | Summary |
|---|---|
| [locations](locations/) | [jvm]<br>val [locations](locations/): [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[GraphQLErrorLocation](../-graph-q-l-error-location/)&gt;? = null |
| [message](message/) | [jvm]<br>val [message](message/): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [path](path/) | [jvm]<br>val [path](path/): [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;JsonElement&gt;? = null |
