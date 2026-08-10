package io.github.nimbleflux.fluxbase.postgrest

import io.github.nimbleflux.fluxbase.FluxbaseClient
import io.github.nimbleflux.fluxbase.FluxbaseClientOptions
import io.github.nimbleflux.fluxbase.from
import io.github.nimbleflux.fluxbase.core.test.RecordingHttp
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.Serializable
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull
import kotlin.test.assertTrue

/**
 * Unit tests for [QueryBuilder] — porting the TS SDK's `query-builder.test.ts`.
 */
class QueryBuilderTest {

    @Serializable
    data class Product(val id: Long, val name: String, val price: Double)

    private fun client(recording: RecordingHttp): FluxbaseClient =
        FluxbaseClient.create(
            url = "http://localhost:8080",
            key = "anon-key",
            transport = recording,
            options = io.github.nimbleflux.fluxbase.FluxbaseClientOptions(autoRefresh = false),
        )

    @Test
    fun `select builds GET to tables path`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "[]")
        val c = client(recording)

        c.from<Product>("products").select().execute()

        assertEquals("GET", recording.lastMethod)
        assertEquals("/api/v1/tables/products", recording.lastPath)
    }

    @Test
    fun `select with schema builds schema path`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "[]")
        val c = client(recording)

        c.from<Product>("products", schema = "public").select().execute()

        assertEquals("/api/v1/tables/public/products", recording.lastPath)
    }

    @Test
    fun `eq filter adds eq operator`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "[]")
        val c = client(recording)

        c.from<Product>("products").eq("price", 29.99).execute()

        assertTrue(recording.lastPath!!.contains("price=eq.29.99"))
    }

    @Test
    fun `neq filter`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "[]")
        val c = client(recording)

        c.from<Product>("products").neq("status", "deleted").execute()

        assertTrue(recording.lastPath!!.contains("status=neq.deleted"))
    }

    @Test
    fun `gt and lt filters`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "[]")
        val c = client(recording)

        c.from<Product>("products").gt("price", 10).lt("price", 100).execute()

        val path = recording.lastPath!!
        assertTrue(path.contains("price=gt.10"))
        assertTrue(path.contains("price=lt.100"))
    }

    @Test
    fun `gte and lte filters`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "[]")
        val c = client(recording)

        c.from<Product>("products").gte("price", 10).lte("price", 100).execute()

        val path = recording.lastPath!!
        assertTrue(path.contains("price=gte.10"))
        assertTrue(path.contains("price=lte.100"))
    }

    @Test
    fun `in filter`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "[]")
        val c = client(recording)

        c.from<Product>("products").`in`("category", listOf("electronics", "books")).execute()

        assertTrue(recording.lastPath!!.contains("category=in."))
    }

    @Test
    fun `like filter`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "[]")
        val c = client(recording)

        c.from<Product>("products").like("name", "%phone%").execute()

        assertTrue(recording.lastPath!!.contains("name=like."))
    }

    @Test
    fun `order adds order param`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "[]")
        val c = client(recording)

        c.from<Product>("products").order("price").execute()

        assertTrue(recording.lastPath!!.contains("order=price"))
    }

    @Test
    fun `order descending`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "[]")
        val c = client(recording)

        c.from<Product>("products").order("price", ascending = false).execute()

        assertTrue(recording.lastPath!!.contains("order=price.desc"))
    }

    @Test
    fun `limit and offset`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "[]")
        val c = client(recording)

        c.from<Product>("products").limit(10).offset(20).execute()

        val path = recording.lastPath!!
        assertTrue(path.contains("limit=10"))
        assertTrue(path.contains("offset=20"))
    }

    @Test
    fun `range sets limit and offset via clone`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "[]")
        val c = client(recording)

        val base = c.from<Product>("products").order("id")
        base.range(0, 9).execute()

        val path = recording.lastPath!!
        assertTrue(path.contains("offset=0"))
        assertTrue(path.contains("limit=10"))
    }

    @Test
    fun `multiple filters chain`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "[]")
        val c = client(recording)

        c.from<Product>("products").eq("status", "active").gte("price", 10).lte("price", 100).execute()

        val path = recording.lastPath!!
        assertTrue(path.contains("status=eq.active"))
        assertTrue(path.contains("price=gte.10"))
        assertTrue(path.contains("price=lte.100"))
    }

    @Test
    fun `insert sends POST with body`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "[]")
        val c = client(recording)

        c.from<Product>("products").insert(mapOf("name" to "Widget", "price" to 9.99))

        assertEquals("POST", recording.lastMethod)
        assertEquals("/api/v1/tables/products", recording.lastPath)
        val body = recording.lastBody as Map<*, *>
        assertEquals("Widget", body["name"])
    }

    @Test
    fun `update sends PATCH with filters in query string`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "[]")
        val c = client(recording)

        c.from<Product>("products").eq("id", 1).update(mapOf("name" to "Updated"))

        assertEquals("PATCH", recording.lastMethod)
        assertTrue(recording.lastPath!!.contains("id=eq.1"))
    }

    @Test
    fun `delete sends DELETE with filters in query string`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "[]")
        val c = client(recording)

        c.from<Product>("products").eq("id", 1).delete()

        assertEquals("DELETE", recording.lastMethod)
        assertTrue(recording.lastPath!!.contains("id=eq.1"))
    }

    @Test
    fun `execute returns deserialized list`() = runTest {
        val recording = RecordingHttp(mockResponseBody = """[{"id":1,"name":"Widget","price":9.99}]""")
        val c = client(recording)

        val result = c.from<Product>("products").select().execute()

        assertNull(result.error)
        val data = result.data!!
        assertEquals(1, data.size)
        assertEquals("Widget", data[0].name)
    }

    @Test
    fun `single returns one row or error`() = runTest {
        val recording = RecordingHttp(mockResponseBody = """[{"id":1,"name":"Widget","price":9.99}]""")
        val c = client(recording)

        val result = c.from<Product>("products").select().single()

        assertNull(result.error)
        assertEquals("Widget", result.data?.name)
    }

    @Test
    fun `maybeSingle returns null for empty result`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "[]")
        val c = client(recording)

        val result = c.from<Product>("products").select().maybeSingle()

        assertNull(result.error)
        assertNull(result.data)
    }
}
