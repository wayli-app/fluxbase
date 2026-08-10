package io.github.nimbleflux.fluxbase.auth

import io.github.nimbleflux.fluxbase.core.FluxbaseHttpClient
import io.github.nimbleflux.fluxbase.core.test.RecordingHttp
import kotlinx.coroutines.test.runTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNotNull
import kotlin.test.assertNull
import kotlin.test.assertTrue

/**
 * Tests for the extended auth methods: refreshSession, getCurrentUser,
 * updateUser, getAuthConfig. These mirror the corresponding TS auth.test.ts
 * blocks for these endpoints.
 */
class FluxbaseAuthExtendedTest {

    private val signInJson = """
        {"user":{"id":"1","email":"user@example.com","created_at":"2024-01-01T00:00:00Z"},
         "access_token":"access-1","refresh_token":"refresh-1","expires_in":3600}
    """.trimIndent()

    private val refreshJson = """
        {"user":{"id":"1","email":"user@example.com","created_at":"2024-01-01T00:00:00Z"},
         "access_token":"access-2","refresh_token":"refresh-2","expires_in":3600}
    """.trimIndent()

    @Test
    fun `refreshSession posts to auth refresh with refresh token`() = runTest {
        val recording = RecordingHttp(mockResponseBody = signInJson)
        val http = FluxbaseHttpClient("http://localhost:8080", recording)
        val auth = FluxbaseAuth(http, autoRefresh = false)
        auth.signIn("user@example.com", "pw")

        recording.mockResponseBody = refreshJson
        val result = auth.refreshSession()

        assertEquals("POST", recording.lastMethod)
        assertEquals("/api/v1/auth/refresh", recording.lastPath)
        val body = recording.lastBody as Map<*, *>
        assertEquals("refresh-1", body["refresh_token"])
        assertNull(result.error)
        assertEquals("access-2", auth.currentSession?.accessToken)
    }

    @Test
    fun `refreshSession emits TOKEN_REFRESHED event`() = runTest {
        val recording = RecordingHttp(mockResponseBody = signInJson)
        val http = FluxbaseHttpClient("http://localhost:8080", recording)
        val auth = FluxbaseAuth(http, autoRefresh = false)
        auth.signIn("user@example.com", "pw")

        val events = mutableListOf<AuthState>()
        auth.onAuthStateChange { events.add(it) }

        recording.mockResponseBody = refreshJson
        auth.refreshSession()

        assertTrue(events.any { it.event == AuthChangeEvent.TOKEN_REFRESHED })
    }

    @Test
    fun `getCurrentUser GETs auth user`() = runTest {
        val recording = RecordingHttp(mockResponseBody = signInJson)
        val http = FluxbaseHttpClient("http://localhost:8080", recording)
        val auth = FluxbaseAuth(http, autoRefresh = false)
        auth.signIn("user@example.com", "pw")

        val userJson = """{"id":"1","email":"user@example.com","email_verified":true,"role":"user","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}"""
        recording.mockResponseBody = userJson
        val result = auth.getCurrentUser()

        assertEquals("GET", recording.lastMethod)
        assertEquals("/api/v1/auth/user", recording.lastPath)
        assertNull(result.error)
        assertEquals("user@example.com", result.data?.email)
    }

    @Test
    fun `updateUser PATCHes auth user`() = runTest {
        val recording = RecordingHttp(mockResponseBody = signInJson)
        val http = FluxbaseHttpClient("http://localhost:8080", recording)
        val auth = FluxbaseAuth(http, autoRefresh = false)
        auth.signIn("user@example.com", "pw")

        val userJson = """{"id":"1","email":"new@example.com","email_verified":true,"role":"user","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-02T00:00:00Z"}"""
        recording.mockResponseBody = userJson
        val result = auth.updateUser(mapOf("email" to "new@example.com"))

        assertEquals("PATCH", recording.lastMethod)
        assertEquals("/api/v1/auth/user", recording.lastPath)
        assertNull(result.error)
        assertEquals("new@example.com", result.data?.email)
    }

    @Test
    fun `getAuthConfig GETs auth config`() = runTest {
        val recording = RecordingHttp(mockResponseBody = """
            {"signup_enabled":true,"password_login_enabled":true,"password_min_length":8,
             "oauth_providers":["google","github"],"saml_providers":[]}
        """.trimIndent())
        val http = FluxbaseHttpClient("http://localhost:8080", recording)
        val auth = FluxbaseAuth(http, autoRefresh = false)

        val result = auth.getAuthConfig()

        assertEquals("GET", recording.lastMethod)
        assertEquals("/api/v1/auth/config", recording.lastPath)
        assertNull(result.error)
        val config = result.data!!
        assertEquals(true, config.signupEnabled)
        assertEquals(8, config.passwordMinLength)
    }

    @Test
    fun `sendPasswordReset posts to password reset`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "{}")
        val http = FluxbaseHttpClient("http://localhost:8080", recording)
        val auth = FluxbaseAuth(http, autoRefresh = false)

        auth.sendPasswordReset("user@example.com")

        assertEquals("POST", recording.lastMethod)
        assertEquals("/api/v1/auth/password/reset", recording.lastPath)
        val body = recording.lastBody as Map<*, *>
        assertEquals("user@example.com", body["email"])
    }

    @Test
    fun `refreshSession returns error when not authenticated`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "{}")
        val http = FluxbaseHttpClient("http://localhost:8080", recording)
        val auth = FluxbaseAuth(http, autoRefresh = false)

        val result = auth.refreshSession()

        assertNotNull(result.error)
        assertEquals("No active session", result.error?.message)
    }
}
