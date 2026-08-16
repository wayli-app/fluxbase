package io.github.nimbleflux.fluxbase.settings

import io.github.nimbleflux.fluxbase.FluxbaseClient
import io.github.nimbleflux.fluxbase.FluxbaseClientOptions
import io.github.nimbleflux.fluxbase.core.test.RecordingHttp
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull
import kotlin.test.assertTrue

class FluxbaseSettingsTest {

    private fun client(recording: RecordingHttp): FluxbaseClient =
        FluxbaseClient.create("http://localhost:8080", "anon-key", FluxbaseClientOptions(autoRefresh = false), recording)

    @Test
    fun `setSetting PUTs value and description to user setting path`() = runTest {
        val body =
            """{"id":"set-1","key":"wayli.pexels_rate_limit","value":{"limit":100},"description":"Pexels rate limit"}"""
        val recording = RecordingHttp(mockResponseBody = body)
        val c = client(recording)

        val result = c.settings.setSetting(
            "wayli.pexels_rate_limit",
            mapOf("limit" to 100),
            description = "Pexels rate limit",
        )

        assertEquals("PUT", recording.lastMethod)
        assertTrue(recording.lastPath!!.contains("/api/v1/settings/user/wayli.pexels_rate_limit"))
        assertNull(result.error)
        assertEquals("set-1", result.data?.id)
        assertEquals("wayli.pexels_rate_limit", result.data?.key)
        assertEquals(100, result.data?.value?.jsonObject?.get("limit")?.jsonPrimitive?.content?.toInt())
    }

    @Test
    fun `setSetting works without a description`() = runTest {
        val recording = RecordingHttp(mockResponseBody = """{"id":"s1","key":"k","value":{"a":1}}""")
        val c = client(recording)

        val result = c.settings.setSetting("k", mapOf("a" to 1))

        assertEquals("PUT", recording.lastMethod)
        assertNull(result.error)
    }

    @Test
    fun `deleteSetting DELETEs user setting`() = runTest {
        val recording = RecordingHttp()
        val c = client(recording)

        c.settings.deleteSetting("wayli.pexels_rate_limit")

        assertEquals("DELETE", recording.lastMethod)
        assertTrue(recording.lastPath!!.contains("/api/v1/settings/user/wayli.pexels_rate_limit"))
    }

    @Test
    fun `listSettings GETs user list endpoint and parses entries`() = runTest {
        val recording = RecordingHttp(
            mockResponseBody = """[{"id":"s1","key":"theme","value":{"mode":"dark"}},{"id":"s2","key":"notif","value":{"email":true}}]""",
        )
        val c = client(recording)

        val result = c.settings.listSettings()

        assertEquals("GET", recording.lastMethod)
        assertEquals("/api/v1/settings/user/list", recording.lastPath)
        assertNull(result.error)
        assertEquals(2, result.data!!.size)
        assertEquals("theme", result.data!![0].key)
    }

    @Test
    fun `get fetches a setting by key`() = runTest {
        val recording = RecordingHttp(mockResponseBody = """{"value":{"enabled":true}}""")
        val c = client(recording)

        val result = c.settings.get("features.beta_enabled")

        assertEquals("GET", recording.lastMethod)
        assertTrue(recording.lastPath!!.contains("/api/v1/settings/features.beta_enabled"))
        assertNull(result.error)
    }
}
