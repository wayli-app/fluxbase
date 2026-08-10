package io.github.nimbleflux.fluxbase.core

/**
 * HTTP method enum used by the SDK's client layer.
 * (Distinct from Ktor's `HttpMethod` to avoid naming clashes in the transport.)
 */
enum class HttpMethod {
    GET, POST, PUT, PATCH, DELETE, HEAD
}

/**
 * A raw HTTP response: the body as text plus the status code and response headers.
 *
 * The transport returns text (not a deserialized object) because generic
 * deserialization requires reified type parameters, which an interface can't
 * provide. [FluxbaseHttpClient] wraps the transport and adds reified typed
 * methods (`inline fun <reified T>`) that deserialize this body via kotlinx.serialization.
 */
data class HttpResponse(
    val body: String,
    val status: Int,
    val headers: Map<String, String>,
)

/**
 * SPI for the actual HTTP I/O. The SDK's [FluxbaseHttpClient] delegates all wire
 * calls to this interface, which has a production implementation (Ktor-backed,
 * for JVM/Android) and is the seam used by tests:
 * [io.github.nimbleflux.fluxbase.core.test.RecordingHttp] is a fake that records
 * requests instead of sending them.
 *
 * This mirrors the TS SDK's separation where `FluxbaseFetch` wraps `global.fetch()`
 * and tests inject a fake object with the same method shape.
 */
fun interface HttpTransport {
    /**
     * Perform an HTTP [method] request to [path] (relative to the base URL).
     * [body] is a pre-serialized value (will be JSON-encoded by the transport)
     * or null for GET/DELETE/HEAD. [headers] are per-request overrides merged
     * on top of the client defaults. Returns the raw response body as text.
     */
    suspend fun request(
        method: HttpMethod,
        path: String,
        body: Any?,
        headers: Map<String, String>,
    ): HttpResponse
}

/**
 * A Fluxbase API error. Mirrors `FluxbaseError` from `sdk/src/types.ts:235`.
 *
 * Thrown by the transport when a response is not ok (status >= 400). Higher
 * layers (query builder, auth) catch this and convert it into `FluxbaseResponse.Error`.
 */
class FluxbaseException(
    val status: Int,
    val code: String? = null,
    val details: Any? = null,
    message: String,
) : RuntimeException(message)
