package io.github.nimbleflux.fluxbase.core

import kotlinx.serialization.json.Json

/**
 * The shared HTTP client that every Fluxbase module (auth, postgrest, realtime,
 * storage, jobs, …) uses to talk to the Fluxbase server. This is the Kotlin port
 * of `FluxbaseFetch` (`sdk/src/fetch.ts`).
 *
 * Responsibilities:
 *   - Hold the base URL (trailing slash stripped) and default headers.
 *   - Manage the auth token: `setAuthToken(token)` sets `Authorization: Bearer`;
 *     `setAuthToken(null)` restores the anon key fallback (does NOT delete the
 *     header — matches TS `fetch.ts:93-102`).
 *   - Provide convenience methods (`get`, `post`, `put`, `patch`, `delete`) that
 *     return reified typed results via kotlinx.serialization.
 *   - Delegate the actual I/O to an [HttpTransport] SPI, which makes the client
 *     trivially testable with a recording fake.
 *
 * NOTE on the 401 auto-refresh: the TS `FluxbaseFetch` does a single retry after
 * refreshing the token on 401. That logic will be wired in S1 (the transport
 * reports the status; the client decides whether to refresh+retry). For the S0
 * spike, requests are single-shot.
 *
 * @param baseUrl the Fluxbase server URL (trailing slash stripped).
 * @param transport the I/O SPI. If null, a Ktor-backed transport is used at runtime;
 *   tests inject an [io.github.nimbleflux.fluxbase.core.test.RecordingHttp].
 * @param json the JSON decoder used for typed [get]/[post] responses.
 */
class FluxbaseHttpClient(
    baseUrl: String,
    internal val transport: HttpTransport,
    private val json: Json = defaultJson,
) {
    /** The base URL with any trailing slash removed. */
    val baseUrl: String = baseUrl.trimEnd('/')

    /**
     * Default headers applied to every request. `Content-Type: application/json`
     * is always present; `Authorization` is managed via [setAuthToken].
     */
    val defaultHeaders: MutableMap<String, String> = mutableMapOf(
        "Content-Type" to "application/json",
    )

    private var anonKey: String? = null

    /** Set the anon key used as the Authorization fallback on sign-out. */
    fun setAnonKey(key: String) {
        anonKey = key
        defaultHeaders["Authorization"] = "Bearer $key"
    }

    /**
     * Update the authorization token.
     * - With a non-null token: sets `Authorization: Bearer <token>`.
     * - With null: restores the anon key if one is set (does NOT remove the header).
     *   This is how sign-out falls back to anonymous access — see `fetch.ts:93-102`.
     */
    fun setAuthToken(token: String?) {
        when {
            token != null -> defaultHeaders["Authorization"] = "Bearer $token"
            anonKey != null -> defaultHeaders["Authorization"] = "Bearer $anonKey"
            else -> defaultHeaders.remove("Authorization")
        }
    }

    /** Set a custom header on all subsequent requests (e.g. `X-FB-Tenant`). */
    fun setHeader(name: String, value: String) {
        defaultHeaders[name] = value
    }

    /** Remove a custom header. */
    fun removeHeader(name: String) {
        defaultHeaders.remove(name)
    }

    // ---- Convenience methods (typed via reified inline) ----

    /** GET [path], deserialize the JSON body to [T]. */
    suspend inline fun <reified T> get(path: String, headers: Map<String, String> = emptyMap()): T =
        decode(doRequest(HttpMethod.GET, path, null, mergedHeaders(headers)).body)

    /** POST [body] to [path], deserialize the JSON response to [T]. */
    suspend inline fun <reified T> post(path: String, body: Any? = null, headers: Map<String, String> = emptyMap()): T =
        decode(doRequest(HttpMethod.POST, path, body, mergedHeaders(headers)).body)

    /** PUT [body] to [path], deserialize the JSON response to [T]. */
    suspend inline fun <reified T> put(path: String, body: Any? = null, headers: Map<String, String> = emptyMap()): T =
        decode(doRequest(HttpMethod.PUT, path, body, mergedHeaders(headers)).body)

    /** PATCH [body] to [path], deserialize the JSON response to [T]. */
    suspend inline fun <reified T> patch(path: String, body: Any? = null, headers: Map<String, String> = emptyMap()): T =
        decode(doRequest(HttpMethod.PATCH, path, body, mergedHeaders(headers)).body)

    /** DELETE [path]. No deserialization (TS returns void). */
    suspend fun delete(path: String, headers: Map<String, String> = emptyMap()) {
        transport.request(HttpMethod.DELETE, path, null, mergedHeaders(headers))
    }

    /**
     * GET that also returns response headers — used by the query builder to parse
     * `Content-Range` for count queries. Mirrors `getWithHeaders` in `fetch.ts`.
     */
    suspend fun getWithHeaders(path: String, headers: Map<String, String> = emptyMap()): HttpResponse =
        transport.request(HttpMethod.GET, path, null, mergedHeaders(headers))

    /**
     * POST that also returns response headers. Mirrors `postWithHeaders` in `fetch.ts`.
     */
    suspend fun postWithHeaders(path: String, body: Any? = null, headers: Map<String, String> = emptyMap()): HttpResponse =
        transport.request(HttpMethod.POST, path, body, mergedHeaders(headers))

    // ---- Internal helpers (exposed for inline functions) ----

    @PublishedApi
    internal suspend fun doRequest(
        method: HttpMethod,
        path: String,
        body: Any?,
        headers: Map<String, String>,
    ): HttpResponse = transport.request(method, path, body, headers)

    @PublishedApi
    internal fun mergedHeaders(perRequest: Map<String, String>): Map<String, String> =
        defaultHeaders + perRequest

    @PublishedApi
    internal val jsonInstance: Json = json

    @PublishedApi
    internal inline fun <reified T> decode(body: String): T {
        if (body.isBlank() && Unit is T) {
            @Suppress("UNCHECKED_CAST")
            return Unit as T
        }
        return jsonInstance.decodeFromString(body)
    }

    companion object {
        /** Shared JSON config — lenient to tolerate Fluxbase's varied response shapes. */
        val defaultJson: Json = Json {
            ignoreUnknownKeys = true
            isLenient = true
            encodeDefaults = false
            explicitNulls = false
        }
    }
}
