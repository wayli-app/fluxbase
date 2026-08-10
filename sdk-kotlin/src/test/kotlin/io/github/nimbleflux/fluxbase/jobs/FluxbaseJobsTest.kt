package io.github.nimbleflux.fluxbase.jobs

import io.github.nimbleflux.fluxbase.FluxbaseClient
import io.github.nimbleflux.fluxbase.FluxbaseClientOptions
import io.github.nimbleflux.fluxbase.core.test.RecordingHttp
import kotlinx.coroutines.test.runTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull
import kotlin.test.assertTrue

class FluxbaseJobsTest {

    private val jobJson = """{"id":"job-1","job_name":"trip-detection","status":"queued","namespace":"wayli"}"""

    private fun client(recording: RecordingHttp): FluxbaseClient =
        FluxbaseClient.create("http://localhost:8080", "anon-key", FluxbaseClientOptions(autoRefresh = false), recording)

    @Test
    fun `submit posts to jobs submit endpoint`() = runTest {
        val recording = RecordingHttp(mockResponseBody = jobJson)
        val c = client(recording)

        c.jobs.submit("trip-detection", mapOf("date" to "2024-01-01"), SubmitJobOptions(namespace = "wayli"))

        assertEquals("POST", recording.lastMethod)
        assertEquals("/api/v1/jobs/submit", recording.lastPath)
        val body = recording.lastBody as Map<*, *>
        assertEquals("trip-detection", body["job_name"])
        assertEquals("wayli", body["namespace"])
    }

    @Test
    fun `submit returns job`() = runTest {
        val recording = RecordingHttp(mockResponseBody = jobJson)
        val c = client(recording)

        val result = c.jobs.submit("trip-detection")

        assertNull(result.error)
        assertEquals("job-1", result.data?.id)
        assertEquals("trip-detection", result.data?.jobName)
    }

    @Test
    fun `get gets job by id`() = runTest {
        val recording = RecordingHttp(mockResponseBody = jobJson)
        val c = client(recording)

        c.jobs.get("job-1")

        assertEquals("GET", recording.lastMethod)
        assertEquals("/api/v1/jobs/job-1", recording.lastPath)
    }

    @Test
    fun `cancel posts to jobs cancel`() = runTest {
        val recording = RecordingHttp(mockResponseBody = jobJson)
        val c = client(recording)

        c.jobs.cancel("job-1")

        assertEquals("POST", recording.lastMethod)
        assertTrue(recording.lastPath!!.contains("/api/v1/jobs/job-1/cancel"))
    }

    @Test
    fun `list with filters`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "[$jobJson]")
        val c = client(recording)

        c.jobs.list(status = "queued", namespace = "wayli", limit = 10)

        val path = recording.lastPath!!
        assertTrue(path.contains("status=queued"))
        assertTrue(path.contains("namespace=wayli"))
        assertTrue(path.contains("limit=10"))
    }
}
