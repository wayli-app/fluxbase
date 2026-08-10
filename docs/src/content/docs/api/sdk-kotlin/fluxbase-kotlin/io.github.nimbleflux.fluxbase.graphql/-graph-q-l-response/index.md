//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.graphql](../index.md)/[GraphQLResponse](index.md)

# GraphQLResponse

[jvm]\
@Serializable

data class [GraphQLResponse](index.md)&lt;[T](index.md)&gt;(val data: [T](index.md)? = null, val errors: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[GraphQLError](../-graph-q-l-error/index.md)&gt;? = null)

A GraphQL response. Port of `GraphQLResponse<T>` from `sdk/src/types.ts`.

## Constructors

| | |
|---|---|
| [GraphQLResponse](-graph-q-l-response.md) | [jvm]<br>constructor(data: [T](index.md)? = null, errors: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[GraphQLError](../-graph-q-l-error/index.md)&gt;? = null) |

## Properties

| Name | Summary |
|---|---|
| [data](data.md) | [jvm]<br>val [data](data.md): [T](index.md)? = null |
| [errors](errors.md) | [jvm]<br>val [errors](errors.md): [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[GraphQLError](../-graph-q-l-error/index.md)&gt;? = null |
