package io.github.nimbleflux.fluxbase.core

import io.ktor.client.HttpClient
import io.ktor.client.engine.okhttp.OkHttp
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.request.header
import io.ktor.client.request.request
import io.ktor.client.request.setBody
import io.ktor.client.statement.bodyAsText
import io.ktor.http.isSuccess
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.buildJsonObject

/**
 * Ktor-backed [HttpTransport] — the production HTTP implementation for JVM/Android.
 *
 * This is the equivalent of the TS SDK's `FluxbaseFetch` class which wraps
 * `global.fetch()`. It handles the actual TCP I/O, JSON serialization of the
 * request body, and status-code-to-exception conversion.
 *
 * The full port (S1) will add: 30s timeout, 401 auto-refresh+retry (single
 * shared refresh, deduped across concurrent requests), `getBlob` for file
 * downloads, and a `beforeRequest` header-mutation hook. For the S0 spike this
 * minimal version proves the wire contract end-to-end.
 *
 * NOTE: the request body arrives as an arbitrary Kotlin object (typically a Map
 * or List built by the SDK's method callers, mirroring how the TS SDK passes
 * plain JS objects). We serialize it to JSON via [encodeBody].
 */
class KtorHttpTransport(
    private val baseUrl: String,
    private val json: Json = FluxbaseHttpClient.defaultJson,
) : HttpTransport {

    private val client: HttpClient = HttpClient(OkHttp) {
        install(ContentNegotiation) { json(this@KtorHttpTransport.json) }
        // TODO(S1): configure timeout (30s default), connection pool.
    }

    override suspend fun request(
        method: HttpMethod,
        path: String,
        body: Any?,
        headers: Map<String, String>,
    ): HttpResponse {
        val ktorResponse = client.request(baseUrl.trimEnd('/') + path) {
            this.method = when (method) {
                HttpMethod.GET -> io.ktor.http.HttpMethod.Get
                HttpMethod.POST -> io.ktor.http.HttpMethod.Post
                HttpMethod.PUT -> io.ktor.http.HttpMethod.Put
                HttpMethod.PATCH -> io.ktor.http.HttpMethod.Patch
                HttpMethod.DELETE -> io.ktor.http.HttpMethod.Delete
                HttpMethod.HEAD -> io.ktor.http.HttpMethod.Head
            }
            headers.forEach { (k, v) -> header(k, v) }
            if (body != null) {
            val jsonElement = encodeToJsonElement(body)
            setBody(json.encodeToString(kotlinx.serialization.json.JsonElement.serializer(), jsonElement))
        }
        }

        val responseText = ktorResponse.bodyAsText()

        if (!ktorResponse.status.isSuccess()) {
            val message = try {
                val parsed = json.parseToJsonElement(responseText) as? kotlinx.serialization.json.JsonObject
                parsed?.get("error")?.toString()?.trim('"')
                    ?: responseText.ifBlank { ktorResponse.status.description }
            } catch (_: Exception) {
                responseText.ifBlank { ktorResponse.status.description }
            }
            throw FluxbaseException(status = ktorResponse.status.value, message = message)
        }

        val responseHeaders = ktorResponse.headers.entries()
            .associate { it.key to it.value.joinToString(",") }
        return HttpResponse(responseText, ktorResponse.status.value, responseHeaders)
    }

    /**
     * Convert an arbitrary Kotlin object to a [kotlinx.serialization.json.JsonElement].
     * Handles Maps, Lists, primitives. @Serializable types are handled by the
     * `else` branch via reflection-free `encodeToJsonElement` extension. Mirrors
     * how the TS SDK passes plain objects through `JSON.stringify`.
     */
    @Suppress("UNCHECKED_CAST")
    private fun encodeToJsonElement(value: Any?): kotlinx.serialization.json.JsonElement = when (value) {
        null -> kotlinx.serialization.json.JsonNull
        is String -> kotlinx.serialization.json.JsonPrimitive(value)
        is Number -> kotlinx.serialization.json.JsonPrimitive(value)
        is Boolean -> kotlinx.serialization.json.JsonPrimitive(value)
        is Map<*, *> -> buildJsonObject {
            value.forEach { (k, v) -> put(k.toString(), encodeToJsonElement(v)) }
        }
        is List<*> -> kotlinx.serialization.json.buildJsonArray {
            value.forEach { add(encodeToJsonElement(it)) }
        }
        else -> kotlinx.serialization.json.JsonPrimitive(value.toString())
    }
}
