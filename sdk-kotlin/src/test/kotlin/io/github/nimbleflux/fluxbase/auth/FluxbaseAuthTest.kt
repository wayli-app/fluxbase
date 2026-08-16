package io.github.nimbleflux.fluxbase.auth

import io.github.nimbleflux.fluxbase.core.FluxbaseHttpClient
import io.github.nimbleflux.fluxbase.core.test.RecordingHttp
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.Json
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNotNull
import kotlin.test.assertNull
import kotlin.test.assertTrue

/**
 * Unit tests for [FluxbaseAuth] — porting the TS SDK's `auth.test.ts`.
 *
 * These tests mirror the TS test patterns:
 *   - Inject a [RecordingHttp] fake (Kotlin equivalent of the TS `mockFetch`).
 *   - Set the canned JSON response.
 *   - Call the auth method.
 *   - Assert on the exact path, body, and deserialized result.
 */
class FluxbaseAuthTest {

    private val signInResponseJson = """
        {
            "user": {"id":"1","email":"user@example.com","created_at":"2024-01-01T00:00:00Z"},
            "access_token": "new-access-token",
            "refresh_token": "new-refresh-token",
            "expires_in": 3600
        }
    """.trimIndent()

    private val requires2faJson = """
        {"requires_2fa":true,"user_id":"1","message":"2FA required"}
    """.trimIndent()

    @Test
    fun `signIn posts to auth signin with email and password`() = runTest {
        val recording = RecordingHttp(mockResponseBody = signInResponseJson)
        val http = FluxbaseHttpClient("http://localhost:8080", recording)
        val auth = FluxbaseAuth(http, autoRefresh = false)

        auth.signIn("user@example.com", "password123")

        assertEquals("POST", recording.lastMethod)
        assertEquals("/api/v1/auth/signin", recording.lastPath)
        val body = recording.lastBody as Map<*, *>
        assertEquals("user@example.com", body["email"])
        assertEquals("password123", body["password"])
    }

    @Test
    fun `signIn returns session on success`() = runTest {
        val recording = RecordingHttp(mockResponseBody = signInResponseJson)
        val http = FluxbaseHttpClient("http://localhost:8080", recording)
        val auth = FluxbaseAuth(http, autoRefresh = false)

        val result = auth.signIn("user@example.com", "password123")

        assertNull(result.error)
        val data = result.data!!
        assertEquals("new-access-token", data.session?.accessToken)
        assertEquals("user@example.com", data.user?.email)
    }

    @Test
    fun `signIn sets auth token on http client after success`() = runTest {
        val recording = RecordingHttp(mockResponseBody = signInResponseJson)
        val http = FluxbaseHttpClient("http://localhost:8080", recording)
        val auth = FluxbaseAuth(http, autoRefresh = false)

        auth.signIn("user@example.com", "password123")

        assertEquals("Bearer new-access-token", http.defaultHeaders["Authorization"])
    }

    @Test
    fun `signIn computes expires_at from expires_in`() = runTest {
        val recording = RecordingHttp(mockResponseBody = signInResponseJson)
        val http = FluxbaseHttpClient("http://localhost:8080", recording)
        val auth = FluxbaseAuth(http, autoRefresh = false)

        val result = auth.signIn("user@example.com", "password123")

        val session = result.data!!.session!!
        assertNotNull(session.expiresAt)
        val now = System.currentTimeMillis()
        assertTrue(session.expiresAt!! >= now + 3599_000, "expiresAt should be >= now+3599s")
        assertTrue(session.expiresAt!! <= now + 3601_000, "expiresAt should be <= now+3601s")
    }

    @Test
    fun `signIn returns 2FA challenge when requires_2fa is true`() = runTest {
        val recording = RecordingHttp(mockResponseBody = requires2faJson)
        val http = FluxbaseHttpClient("http://localhost:8080", recording)
        val auth = FluxbaseAuth(http, autoRefresh = false)

        val result = auth.signIn("user@example.com", "password123")

        assertNull(result.error)
        val data = result.data!!
        assertTrue(data.is2faRequired)
        assertEquals("1", data.userId2fa)
        assertNull(data.session)
    }

    @Test
    fun `signInWithPassword is an alias for signIn`() = runTest {
        val recording = RecordingHttp(mockResponseBody = signInResponseJson)
        val http = FluxbaseHttpClient("http://localhost:8080", recording)
        val auth = FluxbaseAuth(http, autoRefresh = false)

        val result = auth.signInWithPassword("user@example.com", "password123")

        assertNull(result.error)
        assertNotNull(result.data?.session)
    }

    @Test
    fun `signUp posts to auth signup`() = runTest {
        val recording = RecordingHttp(mockResponseBody = signInResponseJson)
        val http = FluxbaseHttpClient("http://localhost:8080", recording)
        val auth = FluxbaseAuth(http, autoRefresh = false)

        auth.signUp("newuser@example.com", "password123")

        assertEquals("POST", recording.lastMethod)
        assertEquals("/api/v1/auth/signup", recording.lastPath)
        val body = recording.lastBody as Map<*, *>
        assertEquals("newuser@example.com", body["email"])
    }

    @Test
    fun `signUp returns user and session when email confirmation disabled`() = runTest {
        val recording = RecordingHttp(mockResponseBody = signInResponseJson)
        val http = FluxbaseHttpClient("http://localhost:8080", recording)
        val auth = FluxbaseAuth(http, autoRefresh = false)

        val result = auth.signUp("newuser@example.com", "password123")

        assertNull(result.error)
        assertNotNull(result.data?.session)
    }

    @Test
    fun `signUp returns user with null session when email confirmation required`() = runTest {
        val emailConfirmJson = """
            {"user":{"id":"2","email":"new@example.com","created_at":"2024-01-01T00:00:00Z"}}
        """.trimIndent()
        val recording = RecordingHttp(mockResponseBody = emailConfirmJson)
        val http = FluxbaseHttpClient("http://localhost:8080", recording)
        val auth = FluxbaseAuth(http, autoRefresh = false)

        val result = auth.signUp("new@example.com", "password123")

        assertNull(result.error)
        val data = result.data!!
        assertNotNull(data.user)
        assertNull(data.session)
    }

    @Test
    fun `signOut posts to auth signout and clears session`() = runTest {
        val recording = RecordingHttp(mockResponseBody = signInResponseJson)
        val http = FluxbaseHttpClient("http://localhost:8080", recording)
        http.setAnonKey("anon-key-123")
        val auth = FluxbaseAuth(http, autoRefresh = false)
        auth.signIn("user@example.com", "password123")
        assertEquals("Bearer new-access-token", http.defaultHeaders["Authorization"])

        recording.mockResponseBody = "[]"
        auth.signOut()

        assertEquals("POST", recording.lastMethod)
        assertEquals("/api/v1/auth/signout", recording.lastPath)
        // Sign-out restores anon key
        assertEquals("Bearer anon-key-123", http.defaultHeaders["Authorization"])
        assertNull(auth.currentSession)
    }

    @Test
    fun `getSession returns null when not authenticated`() {
        val recording = RecordingHttp(mockResponseBody = signInResponseJson)
        val http = FluxbaseHttpClient("http://localhost:8080", recording)
        val auth = FluxbaseAuth(http, autoRefresh = false)

        assertNull(auth.currentSession)
        assertNull(auth.currentUser)
    }

    @Test
    fun `getSession returns session after signIn`() = runTest {
        val recording = RecordingHttp(mockResponseBody = signInResponseJson)
        val http = FluxbaseHttpClient("http://localhost:8080", recording)
        val auth = FluxbaseAuth(http, autoRefresh = false)
        auth.signIn("user@example.com", "password123")

        assertEquals("new-access-token", auth.currentSession?.accessToken)
        assertEquals("user@example.com", auth.currentUser?.email)
    }

    @Test
    fun `session is persisted to storage adapter`() = runTest {
        val storage = MemoryStorage()
        val recording = RecordingHttp(mockResponseBody = signInResponseJson)
        val http = FluxbaseHttpClient("http://localhost:8080", recording)
        val auth = FluxbaseAuth(http, autoRefresh = false, storage = storage)

        auth.signIn("user@example.com", "password123")

        val stored = storage.getItem(AUTH_STORAGE_KEY)
        assertNotNull(stored)
        assertTrue(stored.contains("new-access-token"))
    }

    @Test
    fun `session is restored from storage on construction`() = runTest {
        val storage = MemoryStorage()
        val preAuthJson = Json.encodeToString(
            AuthSession.serializer(),
            AuthSession(
                user = User(id = "1", email = "persisted@example.com"),
                accessToken = "persisted-token",
                refreshToken = "persisted-refresh",
                expiresIn = 3600,
                expiresAt = System.currentTimeMillis() + 3_600_000,
            ),
        )
        storage.setItem(AUTH_STORAGE_KEY, preAuthJson)

        val recording = RecordingHttp(mockResponseBody = "[]")
        val http = FluxbaseHttpClient("http://localhost:8080", recording)
        val auth = FluxbaseAuth(http, autoRefresh = false, storage = storage)

        assertEquals("persisted-token", auth.currentSession?.accessToken)
        assertEquals("persisted@example.com", auth.currentUser?.email)
    }

    @Test
    fun `onAuthStateChange fires SIGNED_IN on signIn`() = runTest {
        val recording = RecordingHttp(mockResponseBody = signInResponseJson)
        val http = FluxbaseHttpClient("http://localhost:8080", recording)
        val auth = FluxbaseAuth(http, autoRefresh = false)

        val events = mutableListOf<AuthState>()
        auth.onAuthStateChange { events.add(it) }

        auth.signIn("user@example.com", "password123")

        assertTrue(events.any { it.event == AuthChangeEvent.SIGNED_IN })
        assertNotNull(events.first { it.event == AuthChangeEvent.SIGNED_IN }.session)
    }

    @Test
    fun `onAuthStateChange fires SIGNED_OUT on signOut`() = runTest {
        val recording = RecordingHttp(mockResponseBody = signInResponseJson)
        val http = FluxbaseHttpClient("http://localhost:8080", recording)
        val auth = FluxbaseAuth(http, autoRefresh = false)
        auth.signIn("user@example.com", "password123")

        val events = mutableListOf<AuthState>()
        auth.onAuthStateChange { events.add(it) }

        recording.mockResponseBody = "[]"
        auth.signOut()

        assertTrue(events.any { it.event == AuthChangeEvent.SIGNED_OUT })
        assertNull(events.first { it.event == AuthChangeEvent.SIGNED_OUT }.session)
    }
}
