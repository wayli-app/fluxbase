package io.github.nimbleflux.fluxbase.core

import io.github.nimbleflux.fluxbase.core.test.RecordingHttp
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.Serializable
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull
import kotlin.test.assertTrue

/**
 * Unit tests for [FluxbaseHttpClient] — the shared HTTP layer that every module
 * (auth, postgrest, realtime, storage, …) depends on.
 *
 * Mirrors the approach of the TS SDK's `fetch.test.ts`: we use a recording fake
 * (see [RecordingHttp]) instead of a real HTTP server, asserting on the exact
 * path, method, headers, and body the client sends.
 *
 * The TS SDK's contract (from `sdk/src/fetch.ts`):
 *   - baseUrl has trailing slash stripped.
 *   - default headers include `Content-Type: application/json`.
 *   - `setAuthToken(null)` restores the anon key (does NOT delete the header).
 *   - `setAuthToken(token)` sets `Authorization: Bearer <token>`.
 *   - POST/GET/etc build URL as `baseUrl + path`.
 */
class FluxbaseHttpTest {

    @Serializable
    data class Dummy(val value: String)

    @Test
    fun `baseUrl strips trailing slash`() {
        val client = FluxbaseHttpClient("http://localhost:8080/", RecordingHttp())
        assertEquals("http://localhost:8080", client.baseUrl)
    }

    @Test
    fun `GET sends request to baseUrl plus path`() = runTest {
        val recording = RecordingHttp(mockResponseBody = """{"value":"ok"}""")
        val client = FluxbaseHttpClient("http://localhost:8080", recording)

        client.get<Dummy>("/api/v1/auth/config")

        assertEquals("GET", recording.lastMethod)
        assertEquals("/api/v1/auth/config", recording.lastPath)
    }

    @Test
    fun `GET deserializes typed response`() = runTest {
        val recording = RecordingHttp(mockResponseBody = """{"value":"ok"}""")
        val client = FluxbaseHttpClient("http://localhost:8080", recording)

        val result = client.get<Dummy>("/api/v1/auth/config")

        assertEquals("ok", result.value)
    }

    @Test
    fun `POST sends JSON body and method`() = runTest {
        val recording = RecordingHttp(mockResponseBody = """{"value":"ok"}""")
        val client = FluxbaseHttpClient("http://localhost:8080", recording)

        client.post<Dummy>("/api/v1/auth/signin", mapOf("email" to "a@b.c", "password" to "x"))

        assertEquals("POST", recording.lastMethod)
        assertEquals("/api/v1/auth/signin", recording.lastPath)
        assertEquals(mapOf("email" to "a@b.c", "password" to "x"), recording.lastBody)
    }

    @Test
    fun `setAuthToken sets Bearer header`() {
        val client = FluxbaseHttpClient("http://localhost:8080", RecordingHttp())
        client.setAuthToken("my-jwt")
        assertEquals("Bearer my-jwt", client.defaultHeaders["Authorization"])
    }

    @Test
    fun `setAuthToken null restores anon key`() {
        val client = FluxbaseHttpClient("http://localhost:8080", RecordingHttp())
        client.setAnonKey("anon-key-123")
        client.setAuthToken("my-jwt")
        client.setAuthToken(null)

        // When null, restore to anon key — NOT delete the header.
        assertEquals("Bearer anon-key-123", client.defaultHeaders["Authorization"])
    }

    @Test
    fun `setAnonKey is stored for fallback`() {
        val client = FluxbaseHttpClient("http://localhost:8080", RecordingHttp())
        client.setAnonKey("anon-key")
        // No token set yet → Authorization should fall back to anon key
        assertEquals("Bearer anon-key", client.defaultHeaders["Authorization"])
    }

    @Test
    fun `default headers include Content-Type json`() {
        val client = FluxbaseHttpClient("http://localhost:8080", RecordingHttp())
        assertEquals("application/json", client.defaultHeaders["Content-Type"])
    }

    @Test
    fun `setHeader adds custom header`() {
        val client = FluxbaseHttpClient("http://localhost:8080", RecordingHttp())
        client.setHeader("X-FB-Tenant", "tenant-1")
        assertEquals("tenant-1", client.defaultHeaders["X-FB-Tenant"])
    }

    @Test
    fun `removeHeader removes custom header`() {
        val client = FluxbaseHttpClient("http://localhost:8080", RecordingHttp())
        client.setHeader("X-FB-Tenant", "tenant-1")
        client.removeHeader("X-FB-Tenant")
        assertNull(client.defaultHeaders["X-FB-Tenant"])
    }

    @Test
    fun `per-request headers override defaults`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "[]")
        val client = FluxbaseHttpClient("http://localhost:8080", recording)

        client.get<List<String>>("/path", mapOf("Content-Type" to "text/plain", "X-Custom" to "yes"))

        assertEquals("text/plain", recording.lastHeaders["Content-Type"])
        assertEquals("yes", recording.lastHeaders["X-Custom"])
    }

    @Test
    fun `transport error propagates`() = runTest {
        val recording = RecordingHttp(mockError = FluxbaseException(401, message = "Unauthorized"))
        val client = FluxbaseHttpClient("http://localhost:8080", recording)

        val thrown = kotlin.test.assertFailsWith<FluxbaseException> {
            client.get<Dummy>("/api/v1/auth/user")
        }
        assertEquals(401, thrown.status)
    }
}
