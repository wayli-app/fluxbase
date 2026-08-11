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
 * before invoking the client method. For the binary path ([requestBytes]/`getBytes`),
 * set [mockResponseBytes] — otherwise the text body is UTF-8 encoded as a fallback.
 * To make a call throw, set [mockError].
 */
class RecordingHttp(
    var mockResponseBody: String = "[]",
    var mockError: Throwable? = null,
    var mockStatus: Int = 200,
    var mockHeaders: Map<String, String> = emptyMap(),
    var mockResponseBytes: ByteArray? = null,
) : HttpTransport {

    var lastPath: String? = null
        private set
    var lastMethod: String? = null
        private set
    var lastBody: Any? = null
        private set
    var lastHeaders: Map<String, String> = emptyMap()
        private set

    /**
     * FIFO queue of canned responses. When non-empty, the next request pops the
     * head instead of using the scalar [mockResponseBody]/[mockError]/etc. — lets
     * tests script a sequence (e.g. 401 then 200 for the 401-retry path).
     */
    private val queuedResponses = ArrayDeque<QueuedResponse>()

    private class QueuedResponse(
        val body: String = "[]",
        val status: Int = 200,
        val headers: Map<String, String> = emptyMap(),
        val error: Throwable? = null,
    )

    /** Enqueue the next response (used in order across successive requests). */
    fun queueResponse(
        body: String = "[]",
        status: Int = 200,
        headers: Map<String, String> = emptyMap(),
        error: Throwable? = null,
    ) {
        queuedResponses.addLast(QueuedResponse(body, status, headers, error))
    }

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

        queuedResponses.removeFirstOrNull()?.let { q ->
            q.error?.let { throw it }
            return HttpResponse(q.body, q.status, q.headers)
        }

        mockError?.let { throw it }
        return HttpResponse(mockResponseBody, mockStatus, mockHeaders)
    }

    override suspend fun requestBytes(
        method: HttpMethod,
        path: String,
        headers: Map<String, String>,
    ): ByteArray {
        lastMethod = method.name
        lastPath = path
        lastBody = null
        lastHeaders = headers

        queuedResponses.removeFirstOrNull()?.let { q ->
            q.error?.let { throw it }
            return q.body.toByteArray(Charsets.UTF_8)
        }

        mockError?.let { throw it }
        // Prefer an explicit binary payload; fall back to UTF-8 of the text body.
        return mockResponseBytes ?: mockResponseBody.toByteArray(Charsets.UTF_8)
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
        mockResponseBytes = null
        queuedResponses.clear()
    }
}
