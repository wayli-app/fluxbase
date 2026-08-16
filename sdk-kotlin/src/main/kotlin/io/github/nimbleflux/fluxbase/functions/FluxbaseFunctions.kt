package io.github.nimbleflux.fluxbase.functions

import io.github.nimbleflux.fluxbase.FluxbaseError
import io.github.nimbleflux.fluxbase.FluxbaseResponse
import io.github.nimbleflux.fluxbase.core.FluxbaseHttpClient
import io.github.nimbleflux.fluxbase.core.HttpMethod
import io.github.nimbleflux.fluxbase.fluxbaseResponse
import kotlinx.serialization.json.JsonElement

/**
 * Edge Functions module — port of `FluxbaseFunctions` from `sdk/src/functions.ts`.
 *
 * Invokes Fluxbase edge functions via `POST /api/v1/functions/{name}/invoke`.
 *
 * Usage:
 * ```
 * val result = client.functions.invoke<MyData>("my-fn", body = mapOf("key" to "value"), namespace = "wayli")
 * ```
 */
class FluxbaseFunctions(@PublishedApi internal val http: FluxbaseHttpClient) {

    /**
     * Invoke an edge function. Port of `invoke()` in `functions.ts:77`.
     *
     * @param name the function name.
     * @param body the request body (any JSON-serializable value, or null for GET/DELETE).
     * @param method the HTTP method (default POST).
     * @param namespace optional Fluxbase namespace (added as `?namespace=` query param).
     * @param headers per-request headers.
     */
    suspend inline fun <reified T> invoke(
        name: String,
        body: Any? = null,
        method: String = "POST",
        namespace: String? = null,
        headers: Map<String, String> = emptyMap(),
    ): FluxbaseResponse<T> = fluxbaseResponse {
        val endpoint = if (namespace != null) {
            "/api/v1/functions/$name/invoke?namespace=${encode(namespace)}"
        } else {
            "/api/v1/functions/$name/invoke"
        }
        val httpMethod = when (method.uppercase()) {
            "GET" -> HttpMethod.GET
            "DELETE" -> HttpMethod.DELETE
            "PUT" -> HttpMethod.PUT
            "PATCH" -> HttpMethod.PATCH
            else -> HttpMethod.POST
        }
        when (httpMethod) {
            HttpMethod.GET -> http.get<T>(endpoint, headers)
            HttpMethod.DELETE -> { http.delete(endpoint, headers); Unit as T }
            HttpMethod.PUT -> http.put<T>(endpoint, body, headers)
            HttpMethod.PATCH -> http.patch<T>(endpoint, body, headers)
            HttpMethod.POST -> http.post<T>(endpoint, body, headers)
            HttpMethod.HEAD -> http.get<T>(endpoint, headers)
        }
    }

    /** Invoke a function returning raw JSON (for untyped responses). */
    suspend fun invokeJson(
        name: String,
        body: Any? = null,
        method: String = "POST",
        namespace: String? = null,
    ): FluxbaseResponse<JsonElement> = invoke(name, body, method, namespace)

    @PublishedApi
    internal fun encode(s: String): String = java.net.URLEncoder.encode(s, "UTF-8")
}
