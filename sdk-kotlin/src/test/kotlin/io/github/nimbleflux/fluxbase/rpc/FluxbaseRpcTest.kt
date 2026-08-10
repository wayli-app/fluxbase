package io.github.nimbleflux.fluxbase.rpc

import io.github.nimbleflux.fluxbase.FluxbaseClient
import io.github.nimbleflux.fluxbase.FluxbaseClientOptions
import io.github.nimbleflux.fluxbase.core.test.RecordingHttp
import kotlinx.coroutines.test.runTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull
import kotlin.test.assertTrue

class FluxbaseRpcTest {

    private fun client(recording: RecordingHttp): FluxbaseClient =
        FluxbaseClient.create("http://localhost:8080", "anon-key", FluxbaseClientOptions(autoRefresh = false), recording)

    @Test
    fun `invoke posts to namespaced rpc endpoint`() = runTest {
        val recording = RecordingHttp(mockResponseBody = """{"execution_id":"e1","status":"completed","result":{"count":5}}""")
        val c = client(recording)

        c.rpc.invoke("get-trip-summary", mapOf("trip_id" to "abc"), RpcInvokeOptions(namespace = "wayli"))

        assertEquals("POST", recording.lastMethod)
        assertEquals("/api/v1/rpc/wayli/get-trip-summary", recording.lastPath)
    }

    @Test
    fun `invoke uses default namespace when not specified`() = runTest {
        val recording = RecordingHttp(mockResponseBody = """{"status":"completed"}""")
        val c = client(recording)

        c.rpc.invoke("simple-proc")

        assertEquals("/api/v1/rpc/default/simple-proc", recording.lastPath)
    }

    @Test
    fun `invoke returns response with result`() = runTest {
        val recording = RecordingHttp(mockResponseBody = """{"execution_id":"e1","status":"completed","result":{"count":5},"duration_ms":42}""")
        val c = client(recording)

        val result = c.rpc.invoke("count-trips")

        assertNull(result.error)
        assertEquals("e1", result.data?.executionId)
        assertEquals("completed", result.data?.status)
        assertEquals(42, result.data?.durationMs)
    }

    @Test
    fun `list gets procedures`() = runTest {
        val recording = RecordingHttp(mockResponseBody = """{"procedures":[{"name":"get-trips","namespace":"wayli"}],"count":1}""")
        val c = client(recording)

        val result = c.rpc.list(namespace = "wayli")

        assertEquals("GET", recording.lastMethod)
        assertTrue(recording.lastPath!!.contains("namespace=wayli"))
        assertNull(result.error)
        assertEquals(1, result.data!!.size)
        assertEquals("get-trips", result.data!![0].name)
    }

    @Test
    fun `getStatus gets execution by id`() = runTest {
        val recording = RecordingHttp(mockResponseBody = """{"id":"exec-1","status":"running"}""")
        val c = client(recording)

        val result = c.rpc.getStatus("exec-1")

        assertEquals("GET", recording.lastMethod)
        assertEquals("/api/v1/rpc/executions/exec-1", recording.lastPath)
        assertNull(result.error)
        assertEquals("running", result.data?.status)
    }

    @Test
    fun `getLogs returns log entries`() = runTest {
        val recording = RecordingHttp(mockResponseBody = """{"logs":[{"message":"started","level":"info"}],"count":1}""")
        val c = client(recording)

        val result = c.rpc.getLogs("exec-1", afterLine = 5)

        assertTrue(recording.lastPath!!.contains("after=5"))
        assertNull(result.error)
        assertEquals(1, result.data!!.size)
        assertEquals("started", result.data!![0].message)
    }

    @Test
    fun `invoke with async flag sets async in body`() = runTest {
        val recording = RecordingHttp(mockResponseBody = """{"execution_id":"e1","status":"queued"}""")
        val c = client(recording)

        c.rpc.invoke("long-report", options = RpcInvokeOptions(async = true))

        val body = recording.lastBody as JsonObject
        // The body is a JsonObject; check the async field
        assertEquals(true, body["async"]?.toString()?.contains("true"))
    }

    @Test
    fun `waitForCompletion returns on terminal status`() = runTest {
        val recording = RecordingHttp(mockResponseBody = """{"id":"exec-1","status":"completed","result":{"ok":true}}""")
        val c = client(recording)

        val result = c.rpc.waitForCompletion("exec-1", WaitForCompletionOptions(maxWaitMs = 5000, initialIntervalMs = 50))

        assertNull(result.error)
        assertEquals("completed", result.data?.status)
    }
}

private typealias JsonObject = kotlinx.serialization.json.JsonObject
