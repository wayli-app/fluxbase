package io.github.nimbleflux.fluxbase

import io.github.nimbleflux.fluxbase.auth.FluxbaseAuth
import io.github.nimbleflux.fluxbase.core.FluxbaseHttpClient
import io.github.nimbleflux.fluxbase.core.test.RecordingHttp
import kotlinx.coroutines.test.runTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNotNull
import kotlin.test.assertTrue

/**
 * Unit tests for [FluxbaseClient] — the top-level entry point.
 * Mirrors the TS SDK's `client.test.ts` patterns.
 */
class FluxbaseClientTest {

    @Test
    fun `createClient sets up http with apikey and anon key headers`() {
        val recording = RecordingHttp()
        val client = FluxbaseClient.create(
            url = "http://localhost:8080",
            key = "test-anon-key",
            transport = recording,
        )

        assertEquals("test-anon-key", client.http.defaultHeaders["apikey"])
        // Authorization should be set to the anon key initially
        assertEquals("Bearer test-anon-key", client.http.defaultHeaders["Authorization"])
    }

    @Test
    fun `createClient exposes auth module`() {
        val client = FluxbaseClient.create(
            url = "http://localhost:8080",
            key = "test-anon-key",
            transport = RecordingHttp(),
        )

        assertNotNull(client.auth)
        // auth module should be initialized (no session yet, but module exists)
        assertEquals(null, client.auth.currentSession)
    }

    @Test
    fun `baseUrl strips trailing slash`() {
        val client = FluxbaseClient.create(
            url = "http://localhost:8080/",
            key = "test-anon-key",
            transport = RecordingHttp(),
        )

        assertEquals("http://localhost:8080", client.http.baseUrl)
    }

    @Test
    fun `createClient accepts custom headers`() {
        val client = FluxbaseClient.create(
            url = "http://localhost:8080",
            key = "test-anon-key",
            transport = RecordingHttp(),
            options = FluxbaseClientOptions(headers = mapOf("X-Custom" to "value")),
        )

        assertEquals("value", client.http.defaultHeaders["X-Custom"])
    }

    @Test
    fun `setTenant sets X-FB-Tenant header`() {
        val client = FluxbaseClient.create(
            url = "http://localhost:8080",
            key = "test-anon-key",
            transport = RecordingHttp(),
        )

        client.setTenant("tenant-123")

        assertEquals("tenant-123", client.http.defaultHeaders["X-FB-Tenant"])
    }

    @Test
    fun `auth and http share the same transport`() = runTest {
        val recording = RecordingHttp(
            mockResponseBody = """
                {"user":{"id":"1","email":"a@b.c","created_at":"2024-01-01T00:00:00Z"},
                 "access_token":"tok","refresh_token":"ref","expires_in":3600}
            """.trimIndent(),
        )
        val client = FluxbaseClient.create(
            url = "http://localhost:8080",
            key = "anon-key",
            transport = recording,
        )

        client.auth.signIn("a@b.c", "pw")

        // After sign-in, the http client should have the auth token set
        assertEquals("Bearer tok", client.http.defaultHeaders["Authorization"])
    }
}
