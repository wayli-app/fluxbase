package io.github.nimbleflux.fluxbase.graphql

import io.github.nimbleflux.fluxbase.FluxbaseResponse
import io.github.nimbleflux.fluxbase.core.FluxbaseHttpClient
import io.github.nimbleflux.fluxbase.fluxbaseResponse
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put

/**
 * A GraphQL error. Port of `GraphQLError` from `sdk/src/types.ts`.
 */
@Serializable
data class GraphQLError(
    val message: String,
    val locations: List<GraphQLErrorLocation>? = null,
    val path: List<JsonElement>? = null,
)

@Serializable
data class GraphQLErrorLocation(
    val line: Int,
    val column: Int,
)

/**
 * A GraphQL response. Port of `GraphQLResponse<T>` from `sdk/src/types.ts`.
 */
@Serializable
data class GraphQLResponse<T>(
    val data: T? = null,
    val errors: List<GraphQLError>? = null,
)

/**
 * GraphQL module — port of `FluxbaseGraphQL` from `sdk/src/graphql.ts`.
 *
 * All requests go to `POST /api/v1/graphql`.
 *
 * Usage:
 * ```
 * val (result, _) = client.graphql.query<MyData>("{ trips { id title } }")
 * ```
 */
class FluxbaseGraphQL(@PublishedApi internal val http: FluxbaseHttpClient) {

    /**
     * Execute a GraphQL query. POSTs `{query, variables}` to `/api/v1/graphql`.
     * Port of `query()` / `execute()` in `graphql.ts`.
     */
    suspend inline fun <reified T> query(
        query: String,
        variables: Map<String, Any?>? = null,
        headers: Map<String, String> = emptyMap(),
    ): FluxbaseResponse<GraphQLResponse<T>> = fluxbaseResponse {
        val body = buildJsonObject {
            put("query", query)
            if (variables != null) {
                put("variables", buildJsonObject {
                    variables.forEach { (k, v) ->
                        when (v) {
                            null -> put(k, kotlinx.serialization.json.JsonNull)
                            is String -> put(k, v)
                            is Number -> put(k, v)
                            is Boolean -> put(k, v)
                            else -> put(k, v.toString())
                        }
                    }
                })
            }
        }
        http.post("/api/v1/graphql", body, headers)
    }

    /** Alias for [query] — semantically marks the operation as a mutation. */
    suspend inline fun <reified T> mutation(
        mutation: String,
        variables: Map<String, Any?>? = null,
        headers: Map<String, String> = emptyMap(),
    ): FluxbaseResponse<GraphQLResponse<T>> = query(mutation, variables, headers)
}
