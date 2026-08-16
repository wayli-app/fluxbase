package io.github.nimbleflux.fluxbase.vector

import io.github.nimbleflux.fluxbase.FluxbaseResponse
import io.github.nimbleflux.fluxbase.core.FluxbaseHttpClient
import io.github.nimbleflux.fluxbase.fluxbaseResponse
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put

/** Vector similarity metric. */
enum class VectorSearchMetric(val value: String) {
    L2("l2"),
    COSINE("cosine"),
    INNER_PRODUCT("inner_product"),
}

@Serializable
data class EmbedRequest(
    val text: String? = null,
    val texts: List<String>? = null,
    val model: String? = null,
    val provider: String? = null,
)

@Serializable
data class EmbedResponse(
    val embeddings: List<List<Double>>,
    val model: String,
    val dimensions: Int,
)

@Serializable
data class VectorSearchResult(
    val data: List<JsonElement>,
    val distances: List<Double>,
    val model: String? = null,
)

/**
 * Vector embedding and similarity search module — port of `FluxbaseVector`
 * from `sdk/src/vector.ts`.
 *
 * Usage:
 * ```
 * val (result, _) = client.vector.embed(EmbedRequest(text = "hello"))
 * val (results, _) = client.vector.search("documents", "embedding", query = "find similar")
 * ```
 */
class FluxbaseVector(@PublishedApi internal val http: FluxbaseHttpClient) {

    /** POST `/api/v1/vector/embed` — generate embeddings for text. */
    suspend fun embed(request: EmbedRequest): FluxbaseResponse<EmbedResponse> = fluxbaseResponse {
        val body = buildJsonObject {
            request.text?.let { put("text", it) }
            request.texts?.let { texts ->
                put("texts", kotlinx.serialization.json.JsonArray(texts.map { kotlinx.serialization.json.JsonPrimitive(it) }))
            }
            request.model?.let { put("model", it) }
            request.provider?.let { put("provider", it) }
        }
        http.post("/api/v1/vector/embed", body)
    }

    /**
     * POST `/api/v1/vector/search` — similarity search on a vector column.
     * Either [query] (text, will be embedded server-side) or [vector] (pre-computed)
     * must be provided.
     */
    suspend fun search(
        table: String,
        column: String,
        query: String? = null,
        vector: List<Double>? = null,
        metric: VectorSearchMetric = VectorSearchMetric.COSINE,
        matchThreshold: Double? = null,
        matchCount: Int? = null,
        select: String? = null,
    ): FluxbaseResponse<VectorSearchResult> = fluxbaseResponse {
        val body = buildJsonObject {
            put("table", table)
            put("column", column)
            query?.let { put("query", it) }
            vector?.let { vec ->
                put("vector", kotlinx.serialization.json.JsonArray(vec.map { kotlinx.serialization.json.JsonPrimitive(it) }))
            }
            put("metric", metric.value)
            matchThreshold?.let { put("match_threshold", it) }
            matchCount?.let { put("match_count", it) }
            select?.let { put("select", it) }
        }
        http.post("/api/v1/vector/search", body)
    }
}
