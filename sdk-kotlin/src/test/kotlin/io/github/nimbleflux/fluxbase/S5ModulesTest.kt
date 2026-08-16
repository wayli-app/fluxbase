package io.github.nimbleflux.fluxbase

import io.github.nimbleflux.fluxbase.branching.CreateBranchOptions
import io.github.nimbleflux.fluxbase.core.test.RecordingHttp
import io.github.nimbleflux.fluxbase.tenant.CreateTenantOptions
import io.github.nimbleflux.fluxbase.vector.EmbedRequest
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.Serializable
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull
import kotlin.test.assertTrue

/**
 * Tests for the S5 parity modules: GraphQL, Vector, Branching, Tenant, Management.
 * Each test verifies the correct endpoint path, method, and deserialization.
 */
class S5ModulesTest {

    private fun client(recording: RecordingHttp): FluxbaseClient =
        FluxbaseClient.create(
            url = "http://localhost:8080",
            key = "anon-key",
            transport = recording,
            options = FluxbaseClientOptions(autoRefresh = false),
        )

    // ---- GraphQL ----

    @Serializable
    data class GqlData(val trips: List<Map<String, String>> = emptyList())

    @Test
    fun `graphql query posts to graphql endpoint`() = runTest {
        val recording = RecordingHttp(mockResponseBody = """{"data":{"trips":[]}}""")
        val c = client(recording)

        c.graphql.query<GqlData>("{ trips { id title } }")

        assertEquals("POST", recording.lastMethod)
        assertEquals("/api/v1/graphql", recording.lastPath)
    }

    @Test
    fun `graphql query returns data`() = runTest {
        val recording = RecordingHttp(mockResponseBody = """{"data":{"trips":[{"id":"1"}]}}""")
        val c = client(recording)

        val result = c.graphql.query<GqlData>("{ trips { id } }")

        assertNull(result.error)
        assertEquals(1, result.data?.data?.trips?.size)
    }

    // ---- Vector ----

    @Test
    fun `vector embed posts to vector embed endpoint`() = runTest {
        val recording = RecordingHttp(
            mockResponseBody = """{"embeddings":[[0.1,0.2]],"model":"text-embedding","dimensions":2}""",
        )
        val c = client(recording)

        c.vector.embed(EmbedRequest(text = "hello"))

        assertEquals("POST", recording.lastMethod)
        assertEquals("/api/v1/vector/embed", recording.lastPath)
    }

    @Test
    fun `vector search posts to vector search endpoint`() = runTest {
        val recording = RecordingHttp(
            mockResponseBody = """{"data":[],"distances":[],"model":"test"}""",
        )
        val c = client(recording)

        c.vector.search("documents", "embedding", query = "find similar")

        assertEquals("POST", recording.lastMethod)
        assertEquals("/api/v1/vector/search", recording.lastPath)
    }

    // ---- Branching ----

    @Test
    fun `branching create posts to admin branches`() = runTest {
        val recording = RecordingHttp(
            mockResponseBody = """{"id":"b1","name":"feature","slug":"feature","database_name":"db_feature","status":"creating"}""",
        )
        val c = client(recording)

        c.branching.create("feature", CreateBranchOptions(dataCloneMode = "full_clone"))

        assertEquals("POST", recording.lastMethod)
        assertEquals("/api/v1/admin/branches", recording.lastPath)
    }

    @Test
    fun `branching list gets branches`() = runTest {
        val recording = RecordingHttp(
            mockResponseBody = """{"branches":[],"total":0,"limit":0,"offset":0}""",
        )
        val c = client(recording)

        c.branching.list()

        assertEquals("GET", recording.lastMethod)
        assertTrue(recording.lastPath!!.contains("/api/v1/admin/branches"))
    }

    @Test
    fun `branching get gets branch by id`() = runTest {
        val recording = RecordingHttp(
            mockResponseBody = """{"id":"b1","name":"x","slug":"x","database_name":"db","status":"ready"}""",
        )
        val c = client(recording)

        c.branching.get("b1")

        assertEquals("/api/v1/admin/branches/b1", recording.lastPath)
    }

    // ---- Tenant ----

    @Test
    fun `tenant create posts to admin tenants`() = runTest {
        val recording = RecordingHttp(
            mockResponseBody = """{"id":"t1","slug":"acme","name":"Acme","status":"active"}""",
        )
        val c = client(recording)

        c.tenant.create(CreateTenantOptions(slug = "acme", name = "Acme"))

        assertEquals("POST", recording.lastMethod)
        assertEquals("/api/v1/admin/tenants", recording.lastPath)
    }

    @Test
    fun `tenant listMine gets tenants mine`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "[]")
        val c = client(recording)

        c.tenant.listMine()

        assertEquals("GET", recording.lastMethod)
        assertEquals("/api/v1/admin/tenants/mine", recording.lastPath)
    }

    // ---- Management ----

    @Test
    fun `management clientKeys create posts to client-keys`() = runTest {
        val recording = RecordingHttp(
            mockResponseBody = """{"client_key":{"id":"k1","name":"app","scopes":["read"]},"key":"fbk_secret"}""",
        )
        val c = client(recording)

        c.management.clientKeys.create("app", listOf("read"))

        assertEquals("POST", recording.lastMethod)
        assertEquals("/api/v1/client-keys", recording.lastPath)
    }

    @Test
    fun `management webhooks list gets webhooks`() = runTest {
        val recording = RecordingHttp(mockResponseBody = """{"webhooks":[],"total":0}""")
        val c = client(recording)

        c.management.webhooks.list()

        assertEquals("GET", recording.lastMethod)
        assertEquals("/api/v1/webhooks", recording.lastPath)
    }

    @Test
    fun `management invitations validate gets invitation validate`() = runTest {
        val recording = RecordingHttp(
            mockResponseBody = """{"valid":true}""",
        )
        val c = client(recording)

        c.management.invitations.validate("inv-token-123")

        assertEquals("GET", recording.lastMethod)
        assertEquals("/api/v1/invitations/inv-token-123/validate", recording.lastPath)
    }
}
