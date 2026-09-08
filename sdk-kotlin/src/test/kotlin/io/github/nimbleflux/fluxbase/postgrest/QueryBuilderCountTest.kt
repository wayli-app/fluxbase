package io.github.nimbleflux.fluxbase.postgrest

import io.github.nimbleflux.fluxbase.createFluxbaseClient
import io.github.nimbleflux.fluxbase.core.test.RecordingHttp
import io.github.nimbleflux.fluxbase.from
import kotlinx.serialization.Serializable
import kotlin.test.Test
import kotlinx.coroutines.test.runTest
import kotlin.test.assertEquals
import kotlin.test.assertNull

/**
 * The `?count=exact` total arrives in the Content-Range response header.
 * HTTP/2 connections deliver header names lowercase ("content-range") while
 * HTTP/1.1 canonicalizes ("Content-Range") — the response headers map is
 * case-sensitive, so the lookup must be case-insensitive or the count
 * silently reads as null over TLS (frozen range-dependent UIs downstream).
 */
class QueryBuilderCountTest {

    @Serializable
    private data class Row(val id: Int)

    private fun transport(headerName: String) = RecordingHttp(
        mockResponseBody = """[{"id":1}]""",
        mockHeaders = mapOf(headerName to "0-0/3958"),
    )

    @Test
    fun `count parses from canonical Content-Range header`() = runTest {
        val client = createFluxbaseClient(url = "https://example.com", key = "anon", transport = transport("Content-Range"))
        val result = client.from<Row>("t").select().count().limit(1).execute()
        assertEquals(3958L, result.count)
    }

    @Test
    fun `count parses from lowercase http2 content-range header`() = runTest {
        val client = createFluxbaseClient(url = "https://example.com", key = "anon", transport = transport("content-range"))
        val result = client.from<Row>("t").select().count().limit(1).execute()
        assertEquals(3958L, result.count)
    }

    @Test
    fun `count is null when no Content-Range header is present`() = runTest {
        val client = createFluxbaseClient(url = "https://example.com", key = "anon", transport = RecordingHttp(mockResponseBody = "[]"))
        val result = client.from<Row>("t").select().count().limit(1).execute()
        assertNull(result.count)
    }
}
