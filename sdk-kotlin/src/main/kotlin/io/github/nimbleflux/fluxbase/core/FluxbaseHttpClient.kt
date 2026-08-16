package io.github.nimbleflux.fluxbase.core

import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
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
 * 401 auto-refresh-retry (port of `fetch.ts`'s single-retry-after-refresh):
 *   When [setRefreshTokenCallback] has been wired (the client wires it to
 *   `auth.refreshSession()`), any request that fails with HTTP 401 triggers a
 *   single token refresh, then the request is retried exactly once with the new
 *   token. Concurrent 401s are deduped via [refreshMutex] so only one refresh
 *   fires even when many requests fail simultaneously. A second 401 (refresh
 *   didn't help) is propagated to the caller — there is no retry loop.
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

    // ---- 401 auto-refresh-retry ----

    /** Serializes token refreshes so concurrent 401s trigger a single refresh. */
    private val refreshMutex = Mutex()

    /**
     * Invoked on a 401 to obtain a fresh access token (or null if refresh failed).
     * Wired by [io.github.nimbleflux.fluxbase.FluxbaseClient] to `auth.refreshSession()`.
     */
    private var tokenRefreshCallback: (suspend () -> String?)? = null

    /**
     * Register the callback that refreshes the access token. On a 401 the client
     * invokes this once (deduped across concurrent requests), applies the returned
     * token via [setAuthToken], and retries the original request a single time.
     * Mirrors TS `setRefreshTokenCallback` in `fetch.ts`.
     */
    fun setRefreshTokenCallback(callback: suspend () -> String?) {
        tokenRefreshCallback = callback
    }

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
        decode(doRequest(HttpMethod.GET, path, null, headers).body)

    /** POST [body] to [path], deserialize the JSON response to [T]. */
    suspend inline fun <reified T> post(path: String, body: Any? = null, headers: Map<String, String> = emptyMap()): T =
        decode(doRequest(HttpMethod.POST, path, body, headers).body)

    /** PUT [body] to [path], deserialize the JSON response to [T]. */
    suspend inline fun <reified T> put(path: String, body: Any? = null, headers: Map<String, String> = emptyMap()): T =
        decode(doRequest(HttpMethod.PUT, path, body, headers).body)

    /** PATCH [body] to [path], deserialize the JSON response to [T]. */
    suspend inline fun <reified T> patch(path: String, body: Any? = null, headers: Map<String, String> = emptyMap()): T =
        decode(doRequest(HttpMethod.PATCH, path, body, headers).body)

    /** DELETE [path]. No deserialization (TS returns void). */
    suspend fun delete(path: String, headers: Map<String, String> = emptyMap()) {
        doRequest(HttpMethod.DELETE, path, null, headers)
    }

    /**
     * GET that also returns response headers — used by the query builder to parse
     * `Content-Range` for count queries. Mirrors `getWithHeaders` in `fetch.ts`.
     */
    suspend fun getWithHeaders(path: String, headers: Map<String, String> = emptyMap()): HttpResponse =
        doRequest(HttpMethod.GET, path, null, headers)

    /**
     * POST that also returns response headers. Mirrors `postWithHeaders` in `fetch.ts`.
     */
    suspend fun postWithHeaders(path: String, body: Any? = null, headers: Map<String, String> = emptyMap()): HttpResponse =
        doRequest(HttpMethod.POST, path, body, headers)

    /**
     * Internal: POST that bypasses the 401-retry path. Used by
     * [io.github.nimbleflux.fluxbase.auth.FluxbaseAuth.refreshSession] so the
     * token-refresh call can't recurse into itself when its token is also expired.
     */
    internal suspend fun postWithoutRetry(path: String, body: Any?, headers: Map<String, String> = emptyMap()): HttpResponse =
        transport.request(HttpMethod.POST, path, body, mergedHeaders(headers))

    /**
     * GET [path] and return the response body as raw bytes — the binary-safe path
     * (no charset decode). Used by storage downloads and any other non-text payload.
     * Mirrors `getBlob` in `sdk/src/fetch.ts`.
     */
    suspend fun getBytes(path: String, headers: Map<String, String> = emptyMap()): ByteArray {
        val callback = tokenRefreshCallback
        val merged = mergedHeaders(headers)
        return try {
            transport.requestBytes(HttpMethod.GET, path, merged)
        } catch (e: FluxbaseException) {
            if (e.status != 401 || callback == null) throw e
            refreshTokenIfStale(merged["Authorization"])
            transport.requestBytes(HttpMethod.GET, path, mergedHeaders(headers))
        }
    }

    // ---- Internal helpers (exposed for inline functions) ----

    /**
     * The retry-capable core. Merges [perRequestHeaders] on top of the defaults,
     * performs the request, and — on a 401 with a refresh callback wired — refreshes
     * the token once and retries. [perRequestHeaders] (not pre-merged) is taken so
     * the retry can re-merge against the refreshed defaults.
     */
    @PublishedApi
    internal suspend fun doRequest(
        method: HttpMethod,
        path: String,
        body: Any?,
        perRequestHeaders: Map<String, String>,
    ): HttpResponse {
        val callback = tokenRefreshCallback
        return try {
            transport.request(method, path, body, mergedHeaders(perRequestHeaders))
        } catch (e: FluxbaseException) {
            if (e.status != 401 || callback == null) throw e
            val failedToken = mergedHeaders(perRequestHeaders)["Authorization"]
            refreshTokenIfStale(failedToken)
            transport.request(method, path, body, mergedHeaders(perRequestHeaders))
        }
    }

    /**
     * Refresh-once dedupe. If the auth token hasn't already been rotated by a
     * concurrent retry (compare [failedToken] to the current default), invoke the
     * refresh callback and apply the new token. Otherwise a sibling already
     * refreshed — reuse its token and skip the refresh call.
     */
    private suspend fun refreshTokenIfStale(failedToken: String?) {
        val callback = tokenRefreshCallback ?: return
        refreshMutex.withLock {
            if (defaultHeaders["Authorization"] == failedToken) {
                callback()?.let { setAuthToken(it) }
            }
        }
    }

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
