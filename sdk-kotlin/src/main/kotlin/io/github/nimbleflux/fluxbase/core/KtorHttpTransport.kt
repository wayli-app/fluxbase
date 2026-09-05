package io.github.nimbleflux.fluxbase.core

import io.ktor.client.HttpClient
import io.ktor.client.engine.okhttp.OkHttp
import io.ktor.client.plugins.HttpTimeout
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.request.header
import io.ktor.client.request.request
import io.ktor.client.request.setBody
import io.ktor.client.statement.HttpResponse as KtorHttpResponse
import io.ktor.client.statement.bodyAsText
import io.ktor.client.statement.readRawBytes
import io.ktor.http.isSuccess
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.buildJsonObject

/**
 * Ktor-backed [HttpTransport] — the production HTTP implementation for JVM/Android.
 *
 * This is the equivalent of the TS SDK's `FluxbaseFetch` class which wraps
 * `global.fetch()`. It handles the actual TCP I/O, JSON serialization of the
 * request body, and status-code-to-exception conversion.
 *
 * Two response paths:
 *  - [request] returns the body as text (for JSON APIs).
 *  - [requestBytes] returns the body as raw bytes (binary-safe, for storage
 *    downloads and any other non-text payload) — the Kotlin analogue of the
 *    TS `getBlob`. Bytes never pass through a charset decode, so images and
 *    other non-UTF-8 payloads survive intact.
 *
 * Request bodies:
 *  - [ByteArray] bodies are sent raw (binary-safe, used by storage upload).
 *  - Anything else is JSON-encoded via [encodeToJsonElement], matching how the
 *    TS SDK passes plain JS objects through `JSON.stringify`.
 *
 * The full port (S1) will add: 30s timeout, 401 auto-refresh+retry (single
 * shared refresh, deduped across concurrent requests). For the S0 spike this
 * minimal version proves the wire contract end-to-end.
 */
class KtorHttpTransport(
    private val baseUrl: String,
    private val json: Json = FluxbaseHttpClient.defaultJson,
    timeoutMillis: Long = 30_000,
    trustAllCertificates: Boolean = false,
) : HttpTransport {

    private val client = HttpClient(OkHttp) {
        install(ContentNegotiation) { json(this@KtorHttpTransport.json) }
        install(HttpTimeout) {
            requestTimeoutMillis = timeoutMillis
            connectTimeoutMillis = timeoutMillis
            socketTimeoutMillis = timeoutMillis
        }
        if (trustAllCertificates) {
            engine {
                config {
                    sslSocketFactory(TrustAllCertificates.socketFactory, TrustAllCertificates.trustManager)
                    hostnameVerifier(TrustAllCertificates.hostnameVerifier)
                }
            }
        }
        // TODO(S1): connection pool tuning.
    }

    override suspend fun request(
        method: HttpMethod,
        path: String,
        body: Any?,
        headers: Map<String, String>,
    ): HttpResponse {
        val ktorResponse = rawRequest(method, path, body, headers)
        val responseText = ktorResponse.bodyAsText()

        if (!ktorResponse.status.isSuccess()) {
            throw FluxbaseException(
                status = ktorResponse.status.value,
                message = parseErrorMessage(responseText, ktorResponse.status.description),
            )
        }

        return HttpResponse(responseText, ktorResponse.status.value, responseHeadersOf(ktorResponse))
    }

    override suspend fun requestBytes(
        method: HttpMethod,
        path: String,
        headers: Map<String, String>,
    ): ByteArray {
        val ktorResponse = rawRequest(method, path, body = null, headers)
        val bytes = ktorResponse.readRawBytes()

        if (!ktorResponse.status.isSuccess()) {
            // Best-effort error message: decode the (likely textual) error body.
            throw FluxbaseException(
                status = ktorResponse.status.value,
                message = parseErrorMessage(String(bytes, Charsets.UTF_8), ktorResponse.status.description),
            )
        }
        return bytes
    }

    /**
     * Build and send the request, returning the raw Ktor response. Body handling:
     * [ByteArray] → sent verbatim (binary upload); anything else → JSON-encoded.
     * Callers ([request]/[requestBytes]) read the response body in the form they
     * need and apply the status→exception mapping.
     */
    private suspend fun rawRequest(
        method: HttpMethod,
        path: String,
        body: Any?,
        headers: Map<String, String>,
    ): KtorHttpResponse = client.request(baseUrl.trimEnd('/') + path) {
        this.method = when (method) {
            HttpMethod.GET -> io.ktor.http.HttpMethod.Get
            HttpMethod.POST -> io.ktor.http.HttpMethod.Post
            HttpMethod.PUT -> io.ktor.http.HttpMethod.Put
            HttpMethod.PATCH -> io.ktor.http.HttpMethod.Patch
            HttpMethod.DELETE -> io.ktor.http.HttpMethod.Delete
            HttpMethod.HEAD -> io.ktor.http.HttpMethod.Head
        }
        headers.forEach { (k, v) -> header(k, v) }
        when (body) {
            null -> Unit
            is ByteArray -> setBody(body) // binary-safe (e.g. storage upload)
            else -> {
                val jsonElement = encodeToJsonElement(body)
                setBody(json.encodeToString(kotlinx.serialization.json.JsonElement.serializer(), jsonElement))
            }
        }
    }

    /** Parse the `error` field from a JSON error body, falling back to the raw text/status. */
    private fun parseErrorMessage(text: String, statusDescription: String): String = try {
        val parsed = json.parseToJsonElement(text) as? JsonObject
        parsed?.get("error")?.toString()?.trim('"')
            ?: text.ifBlank { statusDescription }
    } catch (_: Exception) {
        text.ifBlank { statusDescription }
    }

    private fun responseHeadersOf(ktorResponse: KtorHttpResponse): Map<String, String> =
        ktorResponse.headers.entries()
            .associate { it.key to it.value.joinToString(",") }

    /**
     * Convert an arbitrary Kotlin object to a [kotlinx.serialization.json.JsonElement].
     * Handles Maps, Lists, primitives, and passes pre-built [JsonElement] bodies
     * through untouched (RPC bodies are built as JsonObjects by the caller —
     * re-wrapping their leaves via `toString()` would corrupt booleans into
     * "false" strings and quote string values, which servers reject). Mirrors
     * how the TS SDK passes plain JS objects through `JSON.stringify`. Note:
     * binary payloads are NOT routed through here — [rawRequest] sends
     * [ByteArray] verbatim.
     */
    internal fun encodeToJsonElement(value: Any?): kotlinx.serialization.json.JsonElement = when (value) {
        null -> kotlinx.serialization.json.JsonNull
        is kotlinx.serialization.json.JsonElement -> value
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
