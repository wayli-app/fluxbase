---
title: "FluxbaseVector"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.vector](../)/[FluxbaseVector](./)

# FluxbaseVector

[jvm]\
class [FluxbaseVector](./)(http: [FluxbaseHttpClient](../../iogithubnimblefluxfluxbasecore/-fluxbase-http-client/))

Vector embedding and similarity search module — port of `FluxbaseVector` from `sdk/src/vector.ts`.

Usage:

```kotlin
val (result, _) = client.vector.embed(EmbedRequest(text = "hello"))
val (results, _) = client.vector.search("documents", "embedding", query = "find similar")
```

## Constructors

| | |
|---|---|
| [FluxbaseVector](-fluxbase-vector/) | [jvm]<br>constructor(http: [FluxbaseHttpClient](../../iogithubnimblefluxfluxbasecore/-fluxbase-http-client/)) |

## Functions

| Name | Summary |
|---|---|
| [embed](embed/) | [jvm]<br>suspend fun [embed](embed/)(request: [EmbedRequest](../-embed-request/)): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[EmbedResponse](../-embed-response/)&gt;<br>POST `/api/v1/vector/embed` — generate embeddings for text. |
| [search](search/) | [jvm]<br>suspend fun [search](search/)(table: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), query: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, vector: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[Double](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-double/index.html)&gt;? = null, metric: [VectorSearchMetric](../-vector-search-metric/) = VectorSearchMetric.COSINE, matchThreshold: [Double](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-double/index.html)? = null, matchCount: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null, select: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[VectorSearchResult](../-vector-search-result/)&gt;<br>POST `/api/v1/vector/search` — similarity search on a vector column. Either [query](search/) (text, will be embedded server-side) or [vector](search/) (pre-computed) must be provided. |
