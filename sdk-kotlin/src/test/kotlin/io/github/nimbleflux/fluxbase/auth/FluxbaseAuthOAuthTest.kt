package io.github.nimbleflux.fluxbase.auth

import io.github.nimbleflux.fluxbase.core.FluxbaseHttpClient
import io.github.nimbleflux.fluxbase.auth.FluxbaseAuth.Companion.OAUTH_PROVIDER_KEY
import io.github.nimbleflux.fluxbase.auth.FluxbaseAuth.Companion.OAUTH_REDIRECT_URI_KEY
import io.github.nimbleflux.fluxbase.core.test.RecordingHttp
import kotlinx.coroutines.test.runTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNotNull
import kotlin.test.assertNull
import kotlin.test.assertTrue

/**
 * OAuth method tests — porting the TS `auth.test.ts` OAuth cases:
 * providers list, authorize URL (with stored provider/redirect_uri), and
 * the code-for-session exchange that establishes the session.
 */
class FluxbaseAuthOAuthTest {

    private val providersJson = """
        {"providers": [
            {"provider": "authentik", "display_name": "Authentik SSO",
             "authorize_url": "http://localhost:8080/api/v1/auth/oauth/authentik/authorize"},
            {"provider": "google"}
        ]}
    """.trimIndent()

    private val authorizeJson = """
        {"url": "https://idp.example.com/authorize?state=abc", "provider": "authentik"}
    """.trimIndent()

    private val callbackJson = """
        {"user": {"id": "u1", "email": "sso@example.com"},
         "access_token": "oauth-access", "refresh_token": "oauth-refresh",
         "expires_in": 3600, "is_new_user": false}
    """.trimIndent()

    @Test
    fun `getOAuthProviders lists app-login providers`() = runTest {
        val recording = RecordingHttp(mockResponseBody = providersJson)
        val auth = FluxbaseAuth(FluxbaseHttpClient("http://localhost:8080", recording), autoRefresh = false)

        val result = auth.getOAuthProviders()

        assertNull(result.error)
        assertEquals("/api/v1/auth/oauth/providers", recording.lastPath)
        val providers = result.data!!
        assertEquals(2, providers.size)
        assertEquals("authentik", providers[0].provider)
        assertEquals("Authentik SSO", providers[0].displayName)
        assertEquals("google", providers[1].provider)
        assertEquals("", providers[1].displayName) // absent → default
    }

    @Test
    fun `getOAuthUrl passes redirect_uri and remembers provider`() = runTest {
        val recording = RecordingHttp(mockResponseBody = authorizeJson)
        val storage = MapStorage()
        val auth = FluxbaseAuth(
            FluxbaseHttpClient("http://localhost:8080", recording),
            autoRefresh = false,
            storage = storage,
        )

        val result = auth.getOAuthUrl(
            "authentik",
            OAuthOptions(redirectUri = "wayli://oauth/callback"),
        )

        assertNull(result.error)
        assertTrue(recording.lastPath!!.startsWith("/api/v1/auth/oauth/authentik/authorize?"))
        assertTrue("redirect_uri=wayli%3A%2F%2Foauth%2Fcallback" in recording.lastPath!!)
        assertEquals("https://idp.example.com/authorize?state=abc", result.data!!.url)
        // Remembered for the exchange call.
        assertEquals("authentik", storage.getItem(OAUTH_PROVIDER_KEY))
        assertEquals("wayli://oauth/callback", storage.getItem(OAUTH_REDIRECT_URI_KEY))
    }

    @Test
    fun `exchangeCodeForSession establishes the session and clears stored state`() = runTest {
        val recording = RecordingHttp(mockResponseBody = callbackJson)
        val storage = MapStorage().apply {
            setItem(OAUTH_PROVIDER_KEY, "authentik")
            setItem(OAUTH_REDIRECT_URI_KEY, "wayli://oauth/callback")
        }
        val auth = FluxbaseAuth(
            FluxbaseHttpClient("http://localhost:8080", recording),
            autoRefresh = false,
            storage = storage,
        )

        val result = auth.exchangeCodeForSession("code-123", state = "abc")

        assertNull(result.error)
        val path = recording.lastPath!!
        assertTrue(path.startsWith("/api/v1/auth/oauth/authentik/callback?"))
        assertTrue("code=code-123" in path)
        assertTrue("state=abc" in path)
        assertTrue("redirect_uri=wayli%3A%2F%2Foauth%2Fcallback" in path)

        val session = result.data!!.session!!
        assertEquals("oauth-access", session.accessToken)
        assertEquals("sso@example.com", session.user.email)
        assertNotNull(session.expiresAt)
        assertEquals(session, auth.currentSession)
        // Stored OAuth state cleared after the exchange.
        assertNull(storage.getItem(OAUTH_PROVIDER_KEY))
        assertNull(storage.getItem(OAUTH_REDIRECT_URI_KEY))
    }

    @Test
    fun `exchangeCodeForSession without a prior getOAuthUrl fails`() = runTest {
        val recording = RecordingHttp(mockResponseBody = callbackJson)
        val auth = FluxbaseAuth(FluxbaseHttpClient("http://localhost:8080", recording), autoRefresh = false)

        val result = auth.exchangeCodeForSession("code-123")

        assertNotNull(result.error)
        assertTrue("getOAuthUrl" in result.error!!.message)
        assertNull(auth.currentSession)
    }

    @Test
    fun `auth state change fires SIGNED_IN on exchange`() = runTest {
        val recording = RecordingHttp(mockResponseBody = callbackJson)
        val storage = MapStorage().apply { setItem(OAUTH_PROVIDER_KEY, "authentik") }
        val auth = FluxbaseAuth(
            FluxbaseHttpClient("http://localhost:8080", recording),
            autoRefresh = false,
            storage = storage,
        )
        val events = mutableListOf<AuthChangeEvent>()
        auth.onAuthStateChange { events.add(it.event) }

        auth.exchangeCodeForSession("code-123")

        assertEquals(listOf(AuthChangeEvent.SIGNED_IN), events)
    }

    /** Minimal StorageAdapter backed by a map. */
    private class MapStorage : StorageAdapter {
        private val map = mutableMapOf<String, String>()
        override fun getItem(key: String): String? = map[key]
        override fun setItem(key: String, value: String) { map[key] = value }
        override fun removeItem(key: String) { map.remove(key) }
    }
}
