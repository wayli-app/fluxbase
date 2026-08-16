package io.github.nimbleflux.fluxbase.functions

import io.github.nimbleflux.fluxbase.FluxbaseClient
import io.github.nimbleflux.fluxbase.FluxbaseClientOptions
import io.github.nimbleflux.fluxbase.core.test.RecordingHttp
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.Serializable
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull

class FluxbaseFunctionsTest {

    @Serializable
    data class FnResult(val ok: Boolean, val data: String? = null)

    private fun client(recording: RecordingHttp): FluxbaseClient =
        FluxbaseClient.create("http://localhost:8080", "anon-key", FluxbaseClientOptions(autoRefresh = false), recording)

    @Test
    fun `invoke posts to functions invoke endpoint`() = runTest {
        val recording = RecordingHttp(mockResponseBody = """{"ok":true}""")
        val c = client(recording)

        c.functions.invoke<FnResult>("my-fn", body = mapOf("key" to "value"))

        assertEquals("POST", recording.lastMethod)
        assertEquals("/api/v1/functions/my-fn/invoke", recording.lastPath)
    }

    @Test
    fun `invoke with namespace adds query param`() = runTest {
        val recording = RecordingHttp(mockResponseBody = """{"ok":true}""")
        val c = client(recording)

        c.functions.invoke<FnResult>("my-fn", namespace = "wayli")

        assertTrue(recording.lastPath!!.contains("namespace=wayli"))
    }

    @Test
    fun `invoke GET uses GET method`() = runTest {
        val recording = RecordingHttp(mockResponseBody = """{"ok":true}""")
        val c = client(recording)

        c.functions.invoke<FnResult>("status", method = "GET")

        assertEquals("GET", recording.lastMethod)
    }

    @Test
    fun `invoke returns deserialized result`() = runTest {
        val recording = RecordingHttp(mockResponseBody = """{"ok":true,"data":"hello"}""")
        val c = client(recording)

        val result = c.functions.invoke<FnResult>("my-fn")

        assertNull(result.error)
        assertEquals(true, result.data?.ok)
        assertEquals("hello", result.data?.data)
    }
}

private fun assertTrue(condition: Boolean) = kotlin.test.assertTrue(condition)
