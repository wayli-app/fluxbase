//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.vector](../index.md)/[FluxbaseVector](index.md)

# FluxbaseVector

[jvm]\
class [FluxbaseVector](index.md)(http: [FluxbaseHttpClient](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md))

Vector embedding and similarity search module — port of `FluxbaseVector` from `sdk/src/vector.ts`.

Usage:

```kotlin
val (result, _) = client.vector.embed(EmbedRequest(text = "hello"))
val (results, _) = client.vector.search("documents", "embedding", query = "find similar")
```

## Constructors

| | |
|---|---|
| [FluxbaseVector](-fluxbase-vector.md) | [jvm]<br>constructor(http: [FluxbaseHttpClient](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md)) |

## Functions

| Name | Summary |
|---|---|
| [embed](embed.md) | [jvm]<br>suspend fun [embed](embed.md)(request: [EmbedRequest](../-embed-request/index.md)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[EmbedResponse](../-embed-response/index.md)&gt;<br>POST `/api/v1/vector/embed` — generate embeddings for text. |
| [search](search.md) | [jvm]<br>suspend fun [search](search.md)(table: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), query: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, vector: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[Double](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-double/index.html)&gt;? = null, metric: [VectorSearchMetric](../-vector-search-metric/index.md) = VectorSearchMetric.COSINE, matchThreshold: [Double](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-double/index.html)? = null, matchCount: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null, select: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[VectorSearchResult](../-vector-search-result/index.md)&gt;<br>POST `/api/v1/vector/search` — similarity search on a vector column. Either [query](search.md) (text, will be embedded server-side) or [vector](search.md) (pre-computed) must be provided. |
