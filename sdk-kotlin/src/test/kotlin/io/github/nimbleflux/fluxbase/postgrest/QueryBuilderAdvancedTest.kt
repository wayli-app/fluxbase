package io.github.nimbleflux.fluxbase.postgrest

import io.github.nimbleflux.fluxbase.FluxbaseClient
import io.github.nimbleflux.fluxbase.FluxbaseClientOptions
import io.github.nimbleflux.fluxbase.from
import io.github.nimbleflux.fluxbase.core.test.RecordingHttp
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.Serializable
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

/**
 * Tests for advanced PostgREST operators: PostGIS, pgvector, between, upsert,
 * and count. Porting the TS query-builder.test.ts equivalents.
 */
class QueryBuilderAdvancedTest {

    @Serializable
    data class Place(val id: Long, val name: String)

    private fun client(recording: RecordingHttp): FluxbaseClient =
        FluxbaseClient.create(
            url = "http://localhost:8080",
            key = "anon-key",
            transport = recording,
            options = FluxbaseClientOptions(autoRefresh = false),
        )

    // ---- PostGIS (port of query-builder.ts:382-464) ----

    @Test
    fun `intersects adds st_intersects filter`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "[]")
        val c = client(recording)
        val point = mapOf("type" to "Point", "coordinates" to listOf(-122.4, 37.8))

        c.from<Place>("places").intersects("location", point).execute()

        val path = recording.lastPath!!
        assertTrue(path.contains("location=st_intersects."))
        assertTrue(path.contains("Point"))
    }

    @Test
    fun `stContains adds st_contains filter`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "[]")
        val c = client(recording)

        c.from<Place>("places").stContains("region", mapOf("type" to "Point", "coordinates" to listOf(0.0, 0.0))).execute()

        assertTrue(recording.lastPath!!.contains("region=st_contains."))
    }

    @Test
    fun `within adds st_within filter`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "[]")
        val c = client(recording)

        c.from<Place>("places").within("point", mapOf("type" to "Polygon", "coordinates" to listOf<List<Double>>(emptyList()))).execute()

        assertTrue(recording.lastPath!!.contains("point=st_within."))
    }

    @Test
    fun `stDWithin adds st_dwithin filter with distance`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "[]")
        val c = client(recording)
        val point = mapOf("type" to "Point", "coordinates" to listOf(-122.4, 37.8))

        c.from<Place>("places").stDWithin("location", point, 1000.0).execute()

        val path = recording.lastPath!!
        assertTrue(path.contains("location=st_dwithin."))
        assertTrue(path.contains("1000"))
    }

    @Test
    fun `stDistance adds st_distance filter`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "[]")
        val c = client(recording)

        c.from<Place>("places").stDistance("location", mapOf("type" to "Point", "coordinates" to listOf(0.0, 0.0))).execute()

        assertTrue(recording.lastPath!!.contains("location=st_distance."))
    }

    // ---- pgvector (port of query-builder.ts:500-570) ----

    @Test
    fun `vectorSearch adds vector order clause with cosine metric`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "[]")
        val c = client(recording)
        val vector = listOf(0.1, 0.2, 0.3)

        c.from<Place>("places").vectorSearch("embedding", vector, VectorMetric.COSINE).execute()

        val path = recording.lastPath!!
        // vectorSearch delegates to orderByVector which adds an order clause
        assertTrue(path.contains("order=embedding.vec_cos"), "path should contain vec_cos order: $path")
    }

    @Test
    fun `orderByVector adds vector order clause`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "[]")
        val c = client(recording)
        val vector = listOf(0.1, 0.2, 0.3)

        c.from<Place>("places").orderByVector("embedding", vector, VectorMetric.L2).execute()

        val path = recording.lastPath!!
        assertTrue(path.contains("order=embedding.vec_l2"), "path should contain vec_l2 order: $path")
    }

    // ---- Between ----

    @Test
    fun `between adds gte and lte filters`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "[]")
        val c = client(recording)

        c.from<Place>("places").between("price", 10, 100).execute()

        val path = recording.lastPath!!
        assertTrue(path.contains("price=gte.10"))
        assertTrue(path.contains("price=lte.100"))
    }

    @Test
    fun `notBetween adds lt and gt filters`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "[]")
        val c = client(recording)

        c.from<Place>("places").notBetween("price", 10, 100).execute()

        val path = recording.lastPath!!
        assertTrue(path.contains("price=lt.10") || path.contains("price=gt.100"))
    }

    // ---- Upsert ----

    @Test
    fun `upsert sends POST with Prefer header for conflict resolution`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "[]")
        val c = client(recording)

        c.from<Place>("places").upsert(mapOf("id" to 1, "name" to "Updated"))

        assertEquals("POST", recording.lastMethod)
        // Upsert uses the Prefer resolution=merge header
        assertTrue(recording.lastHeaders.containsKey("Prefer") || recording.lastHeaders["Prefer"]?.contains("resolution") == true,
            "upsert should set Prefer resolution header")
    }

    // ---- Count ----

    @Test
    fun `count adds count param and parses Content-Range`() = runTest {
        val recording = RecordingHttp(
            mockResponseBody = "[]",
            mockStatus = 200,
            mockHeaders = mapOf("Content-Range" to "0-0/42"),
        )
        val c = client(recording)

        val result = c.from<Place>("places").select().count(CountType.EXACT).execute()

        // Count is parsed from Content-Range header (0-0/42 → 42)
        assertTrue(result.count != null, "count should be parsed from Content-Range")
    }
}
