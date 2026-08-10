package io.github.nimbleflux.fluxbase.core.test

import io.github.nimbleflux.fluxbase.core.HttpMethod
import io.github.nimbleflux.fluxbase.core.HttpResponse
import io.github.nimbleflux.fluxbase.core.HttpTransport

/**
 * A recording fake [HttpTransport] — the Kotlin equivalent of the TS SDK's
 * `MockFetch` class (see `sdk/src/query-builder.test.ts`). It records the last
 * request (path, method, body, headers) and returns a canned response, letting
 * tests assert on the exact wire shapes the client produces without a real server.
 *
 * Usage:
 * ```
 * val recording = RecordingHttp()
 * val client = FluxbaseHttpClient("http://localhost:8080", recording)
 * client.get<String>("/api/v1/auth/config")
 * assertEquals("/api/v1/auth/config", recording.lastPath)
 * ```
 *
 * To make a call return specific data, set [mockResponseBody] to a JSON string
 * before invoking the client method. To make a call throw, set [mockError].
 */
class RecordingHttp(
    var mockResponseBody: String = "[]",
    var mockError: Throwable? = null,
    var mockStatus: Int = 200,
    var mockHeaders: Map<String, String> = emptyMap(),
) : HttpTransport {

    var lastPath: String? = null
        private set
    var lastMethod: String? = null
        private set
    var lastBody: Any? = null
        private set
    var lastHeaders: Map<String, String> = emptyMap()
        private set

    override suspend fun request(
        method: HttpMethod,
        path: String,
        body: Any?,
        headers: Map<String, String>,
    ): HttpResponse {
        lastMethod = method.name
        lastPath = path
        lastBody = body
        lastHeaders = headers

        mockError?.let { throw it }
        return HttpResponse(mockResponseBody, mockStatus, mockHeaders)
    }

    /** Reset all recorded state and mock returns (call in @BeforeEach). */
    fun reset() {
        lastPath = null
        lastMethod = null
        lastBody = null
        lastHeaders = emptyMap()
        mockResponseBody = "[]"
        mockError = null
        mockStatus = 200
        mockHeaders = emptyMap()
    }
}
