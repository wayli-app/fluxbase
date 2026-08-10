//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.graphql](../index.md)/[GraphQLError](index.md)

# GraphQLError

[jvm]\
@Serializable

data class [GraphQLError](index.md)(val message: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val locations: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[GraphQLErrorLocation](../-graph-q-l-error-location/index.md)&gt;? = null, val path: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;JsonElement&gt;? = null)

A GraphQL error. Port of `GraphQLError` from `sdk/src/types.ts`.

## Constructors

| | |
|---|---|
| [GraphQLError](-graph-q-l-error.md) | [jvm]<br>constructor(message: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), locations: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[GraphQLErrorLocation](../-graph-q-l-error-location/index.md)&gt;? = null, path: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;JsonElement&gt;? = null) |

## Properties

| Name | Summary |
|---|---|
| [locations](locations.md) | [jvm]<br>val [locations](locations.md): [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[GraphQLErrorLocation](../-graph-q-l-error-location/index.md)&gt;? = null |
| [message](message.md) | [jvm]<br>val [message](message.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [path](path.md) | [jvm]<br>val [path](path.md): [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;JsonElement&gt;? = null |
