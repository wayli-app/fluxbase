package io.github.nimbleflux.fluxbase.auth

import io.github.nimbleflux.fluxbase.FluxbaseError
import io.github.nimbleflux.fluxbase.FluxbaseResponse
import io.github.nimbleflux.fluxbase.core.FluxbaseHttpClient
import io.github.nimbleflux.fluxbase.getOrNull
import io.github.nimbleflux.fluxbase.fluxbaseResponse
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.datetime.Clock
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive

/**
 * The result of a sign-in or sign-up call.
 *
 * On success with a session: [session] is non-null, [user] is set.
 * On 2FA challenge: [is2faRequired] is true, [userId2fa] holds the user id, [session] is null.
 * On error: the [FluxbaseResponse.Error] variant is returned.
 *
 * Port of the TS `AuthResponseData` (`types.ts:3923`) which returns
 * `{ user, session: AuthSession | null }`, combined with the 2FA branch
 * that returns `SignInWith2FAResponse` instead.
 */
data class AuthResult(
    val user: User? = null,
    val session: AuthSession? = null,
    val is2faRequired: Boolean = false,
    val userId2fa: String? = null,
)

/**
 * Authentication module — port of `FluxbaseAuth` from `sdk/src/auth.ts`.
 *
 * Manages the user session: sign in/up/out, session persistence, and auth state
 * change events. The session is stored via a [StorageAdapter] (default:
 * [MemoryStorage] for JVM; Android injects an EncryptedSharedPreferences-backed
 * implementation).
 *
 * @param http the shared [FluxbaseHttpClient] for making API calls.
 * @param autoRefresh whether to automatically refresh the token before expiry
 *   (default true; disabled in tests). TS default is true (`auth.ts:55`).
 * @param storage the [StorageAdapter] for session persistence.
 */
class FluxbaseAuth(
    private val http: FluxbaseHttpClient,
    private val autoRefresh: Boolean = true,
    private val storage: StorageAdapter = MemoryStorage(),
) {
    private val json: Json = FluxbaseHttpClient.defaultJson

    /** The current session, or null if not authenticated. */
    var currentSession: AuthSession? = null
        private set

    /** The current user, or null if not authenticated. */
    val currentUser: User?
        get() = currentSession?.user

    private val stateListeners = mutableListOf<(AuthState) -> Unit>()

    companion object {
        /** Stored while an OAuth flow is in flight (TS: auth.ts:62-63). */
        internal const val OAUTH_PROVIDER_KEY = "fluxbase.auth.oauth_provider"
        internal const val OAUTH_REDIRECT_URI_KEY = "fluxbase.auth.oauth_redirect_uri"

        /** Refresh this long before the access token expires. */
        private const val REFRESH_LEAD_MS = 60_000L

        /** Cap the scheduler's sleep so session changes are picked up. */
        private const val MAX_REFRESH_POLL_MS = 10 * 60_000L
    }

    // ---- Proactive token refresh (TS autoRefresh parity) ----
    // Declared BEFORE the init block: restore-time refresh launches on this
    // scope, and properties initialize in declaration order.

    private val refreshScope = kotlinx.coroutines.CoroutineScope(
        kotlinx.coroutines.SupervisorJob() + kotlinx.coroutines.Dispatchers.Default,
    )
    private val refreshMutex = kotlinx.coroutines.sync.Mutex()
    @Volatile private var autoRefreshRunning = false

    init {
        // Restore session from storage (mirrors `auth.ts:172-187`).
        restoreSession()
        if (autoRefresh) startAutoRefresh()
    }

    private fun startAutoRefresh() {
        if (autoRefreshRunning || currentSession == null) return
        autoRefreshRunning = true
        refreshScope.launch {
            // One scheduler for the client's lifetime — a null session just
            // idles the poll until the next sign-in reuses the loop. (Kept
            // free of break/continue per detekt's LoopWithTooManyJumpStatements.)
            while (kotlinx.coroutines.currentCoroutineContext().isActive) {
                val expiresAt = currentSession?.expiresAt
                val sleepMs = if (expiresAt != null) {
                    val refreshAt = expiresAt - REFRESH_LEAD_MS
                    val now = Clock.System.now().toEpochMilliseconds()
                    (refreshAt - now).coerceIn(1_000, MAX_REFRESH_POLL_MS)
                } else {
                    MAX_REFRESH_POLL_MS
                }
                kotlinx.coroutines.delay(sleepMs)
                val due = currentSession?.expiresAt
                if (due != null && Clock.System.now().toEpochMilliseconds() >= due - REFRESH_LEAD_MS) {
                    // Failure is non-fatal: the reactive 401-retry path in
                    // FluxbaseHttpClient still covers the next request.
                    refreshSession().getOrNull()
                }
            }
        }
    }

    // ---- Sign in / sign up / sign out ----

    /**
     * Sign in with email and password.
     *
     * POSTs to `/api/v1/auth/signin` with `{email, password}`.
     * If the server returns `requires_2fa: true`, returns an [AuthResult] with
     * [AuthResult.is2faRequired] set — the caller should then call [verify2FA].
     *
     * Port of `signIn()` in `auth.ts:265-313`.
     */
    suspend fun signIn(email: String, password: String): FluxbaseResponse<AuthResult> =
        fluxbaseResponse {
            val body = mapOf("email" to email, "password" to password)
            val responseText = http.postWithHeaders("/api/v1/auth/signin", body).body

            // Check for 2FA challenge before parsing as AuthResponse.
            val parsed = json.parseToJsonElement(responseText).jsonObject
            if (parsed["requires_2fa"]?.jsonPrimitive?.contentOrNull == "true") {
                val twoFa = json.decodeFromString(SignInWith2FaResponse.serializer(), responseText)
                return@fluxbaseResponse AuthResult(
                    is2faRequired = true,
                    userId2fa = twoFa.userId,
                )
            }

            // Normal sign-in: parse as AuthResponse and build session.
            val authResponse = json.decodeFromString(AuthResponse.serializer(), responseText)
            val session = AuthSession(
                user = authResponse.user,
                accessToken = authResponse.accessToken,
                refreshToken = authResponse.refreshToken,
                expiresIn = authResponse.expiresIn,
                expiresAt = Clock.System.now().toEpochMilliseconds() + authResponse.expiresIn * 1000,
            )
            setSessionInternal(session, AuthChangeEvent.SIGNED_IN)
            AuthResult(user = session.user, session = session)
        }

    /** Alias for [signIn] — Supabase-compatible method name. */
    suspend fun signInWithPassword(email: String, password: String): FluxbaseResponse<AuthResult> =
        signIn(email, password)

    // ---- OAuth ----

    /**
     * List the app-login-enabled OAuth providers.
     * GETs `/api/v1/auth/oauth/providers`. Port of `getOAuthProviders()` (auth.ts:913).
     */
    suspend fun getOAuthProviders(): FluxbaseResponse<List<OAuthProviderInfo>> = fluxbaseResponse {
        val responseText = http.getWithHeaders("/api/v1/auth/oauth/providers").body
        json.decodeFromString(OAuthProvidersResponse.serializer(), responseText).providers
    }

    /**
     * Get the authorization URL for [provider] to open in a system browser.
     * GETs `/api/v1/auth/oauth/{provider}/authorize`. Port of `getOAuthUrl()` (auth.ts:923).
     *
     * The provider and [OAuthOptions.redirectUri] are remembered (storage) for
     * the matching [exchangeCodeForSession] call.
     */
    suspend fun getOAuthUrl(
        provider: String,
        options: OAuthOptions = OAuthOptions(),
    ): FluxbaseResponse<OAuthUrlResponse> = fluxbaseResponse {
        val params = buildList {
            options.redirectTo?.let { add("redirect_to=${encode(it)}") }
            options.redirectUri?.let { add("redirect_uri=${encode(it)}") }
            if (options.scopes.isNotEmpty()) {
                add("scopes=${encode(options.scopes.joinToString(","))}")
            }
        }
        val query = if (params.isEmpty()) "" else "?" + params.joinToString("&")
        val responseText = http.getWithHeaders("/api/v1/auth/oauth/$provider/authorize$query").body

        // Remember for exchangeCodeForSession (mirrors the TS storage keys).
        storage.setItem(OAUTH_PROVIDER_KEY, provider)
        options.redirectUri?.let { storage.setItem(OAUTH_REDIRECT_URI_KEY, it) }

        json.decodeFromString(OAuthUrlResponse.serializer(), responseText)
    }

    /**
     * Exchange the OAuth authorization code (from the deep-link/callback)
     * for a session and establish it. GETs
     * `/api/v1/auth/oauth/{provider}/callback?code&state&redirect_uri`.
     * Port of `exchangeCodeForSession()` (auth.ts:955).
     *
     * Requires a preceding [getOAuthUrl] call (for the stored provider).
     */
    suspend fun exchangeCodeForSession(code: String, state: String? = null): FluxbaseResponse<AuthResult> =
        fluxbaseResponse {
            val provider = storage.getItem(OAUTH_PROVIDER_KEY)
                ?: throw FluxbaseAuthException("No OAuth provider found. Call getOAuthUrl first.")
            val redirectUri = storage.getItem(OAUTH_REDIRECT_URI_KEY)

            val params = buildList {
                add("code=$code")
                state?.let { add("state=$it") }
                redirectUri?.let { add("redirect_uri=${encode(it)}") }
            }
            val responseText = http.getWithHeaders("/api/v1/auth/oauth/$provider/callback?${params.joinToString("&")}").body

            storage.removeItem(OAUTH_PROVIDER_KEY)
            storage.removeItem(OAUTH_REDIRECT_URI_KEY)

            val authResponse = json.decodeFromString(AuthResponse.serializer(), responseText)
            val session = AuthSession(
                user = authResponse.user,
                accessToken = authResponse.accessToken,
                refreshToken = authResponse.refreshToken,
                expiresIn = authResponse.expiresIn,
                expiresAt = Clock.System.now().toEpochMilliseconds() + authResponse.expiresIn * 1000,
            )
            setSessionInternal(session, AuthChangeEvent.SIGNED_IN)
            AuthResult(user = session.user, session = session)
        }

    private fun encode(s: String): String = java.net.URLEncoder.encode(s, "UTF-8")

    /**
     * Sign up with email and password.
     *
     * POSTs to `/api/v1/auth/signup`. If email confirmation is disabled, the
     * server returns tokens and a session is established. If email confirmation
     * is required, only the user is returned (no session).
     *
     * Port of `signUp()` in `auth.ts:331-378`.
     */
    suspend fun signUp(email: String, password: String): FluxbaseResponse<AuthResult> =
        fluxbaseResponse {
            val body = mapOf("email" to email, "password" to password)
            val responseText = http.postWithHeaders("/api/v1/auth/signup", body).body
            val parsed = json.parseToJsonElement(responseText).jsonObject

            // If tokens are present, email confirmation is disabled → establish session.
            if (parsed["access_token"] != null && parsed["refresh_token"] != null) {
                val authResponse = json.decodeFromString(AuthResponse.serializer(), responseText)
                val session = AuthSession(
                    user = authResponse.user,
                    accessToken = authResponse.accessToken,
                    refreshToken = authResponse.refreshToken,
                    expiresIn = authResponse.expiresIn,
                    expiresAt = Clock.System.now().toEpochMilliseconds() + authResponse.expiresIn * 1000,
                )
                setSessionInternal(session, AuthChangeEvent.SIGNED_IN)
                AuthResult(user = session.user, session = session)
            } else {
                // Email confirmation required — return user without session.
                // The response wraps the user in a "user" field: {"user": {...}}.
                val user = if (parsed["user"] != null) {
                    json.decodeFromJsonElement(User.serializer(), parsed["user"]!!)
                } else {
                    json.decodeFromString(User.serializer(), responseText)
                }
                AuthResult(user = user, session = null)
            }
        }

    /**
     * Sign out. POSTs to `/api/v1/auth/signout`, clears the session, and restores
     * the anon key on the HTTP client.
     *
     * Port of `signOut()` in `auth.ts`. Uses postWithHeaders to avoid deserializing
     * the response body (signOut returns no useful data).
     */
    suspend fun signOut() {
        http.postWithHeaders("/api/v1/auth/signout")
        clearSession()
    }

    // ---- Session management ----

    /**
     * Register a callback for auth state changes. Returns a function to unsubscribe.
     *
     * Kotlin-native equivalent of the TS `onAuthStateChange(callback)`. For
     * coroutine-native consumption, wrap this in a callbackFlow.
     */
    fun onAuthStateChange(callback: (AuthState) -> Unit): () -> Unit {
        stateListeners.add(callback)
        return { stateListeners.remove(callback) }
    }

    // ---- Session refresh / user management (port of auth.ts) ----

    /**
     * Refresh the current session using the stored refresh token.
     * POSTs to `/api/v1/auth/refresh` with `{refresh_token}`.
     * On success, updates the session and emits `TOKEN_REFRESHED`.
     *
     * Port of `refreshSession()` in `auth.ts`.
     */
    suspend fun refreshSession(): FluxbaseResponse<AuthSession> =
        fluxbaseResponse {
            val requestedRefreshToken = currentSession?.refreshToken
                ?: return@fluxbaseResponse throw FluxbaseError(message = "No active session")
            // Single-flight: every caller (the autoRefresh scheduler, the
            // restore-time refresh, and the HTTP client's reactive 401-retry)
            // funnels through this lock. Concurrent refreshes with the same
            // token otherwise race the server's rotation — the loser's
            // persisted refresh token stops matching the stored hash and the
            // whole session bricks ("possible token theft" server-side).
            refreshMutex.withLock {
                // A sibling refreshed while we waited for the lock: its tokens
                // are already current — return them instead of refreshing
                // again with the now-rotated token.
                currentSession?.let { live ->
                    if (live.refreshToken != requestedRefreshToken) {
                        return@fluxbaseResponse live
                    }
                }
                val body = mapOf("refresh_token" to requestedRefreshToken)
                // Bypass the 401-retry path so a refresh whose own token is expired
                // can't recurse into another refresh.
                val responseText = http.postWithoutRetry("/api/v1/auth/refresh", body).body
                val authResponse = json.decodeFromString(AuthResponse.serializer(), responseText)
                val session = AuthSession(
                    user = authResponse.user,
                    accessToken = authResponse.accessToken,
                    refreshToken = authResponse.refreshToken,
                    expiresIn = authResponse.expiresIn,
                    expiresAt = Clock.System.now().toEpochMilliseconds() + authResponse.expiresIn * 1000,
                )
                setSessionInternal(session, AuthChangeEvent.TOKEN_REFRESHED)
                session
            }
        }

    /**
     * Get the current user from the server. GETs `/api/v1/auth/user`.
     * Port of `getCurrentUser()` in `auth.ts`.
     */
    suspend fun getCurrentUser(): FluxbaseResponse<User> =
        fluxbaseResponse {
            http.get("/api/v1/auth/user")
        }

    /**
     * Update the user's attributes. PATCHes `/api/v1/auth/user`.
     * Port of `updateUser()` in `auth.ts`.
     */
    suspend fun updateUser(attributes: Map<String, Any>): FluxbaseResponse<User> =
        fluxbaseResponse {
            http.patch("/api/v1/auth/user", attributes)
        }

    /**
     * Get the server's auth configuration (signup enabled, OAuth providers,
     * password rules). GETs `/api/v1/auth/config`.
     * Port of `getAuthConfig()` in `auth.ts`.
     */
    suspend fun getAuthConfig(): FluxbaseResponse<AuthConfig> =
        fluxbaseResponse {
            http.get("/api/v1/auth/config")
        }

    /**
     * Send a password reset email. POSTs `/api/v1/auth/password/reset`.
     * Port of `sendPasswordReset()` in `auth.ts`.
     */
    suspend fun sendPasswordReset(email: String): FluxbaseResponse<Unit> =
        fluxbaseResponse {
            http.postWithHeaders("/api/v1/auth/password/reset", mapOf("email" to email))
            Unit
        }

    // ---- 2FA (port of `auth.ts:640-744`) ----

    /** POST `/api/v1/auth/2fa/setup` → returns TOTP secret + QR code. */
    suspend fun setup2FA(): FluxbaseResponse<TwoFactorSetupResponse> =
        fluxbaseResponse {
            http.post("/api/v1/auth/2fa/setup")
        }

    /** POST `/api/v1/auth/2fa/enable` with `{code}` → enables 2FA, returns backup codes. */
    suspend fun enable2FA(code: String): FluxbaseResponse<TwoFactorEnableResponse> =
        fluxbaseResponse {
            http.post("/api/v1/auth/2fa/enable", mapOf("code" to code))
        }

    /**
     * POST `/api/v1/auth/2fa/verify` with `{user_id, code}` — completes a 2FA login
     * challenge. On success, establishes a session from the returned tokens.
     */
    suspend fun verify2FA(userId: String, code: String): FluxbaseResponse<AuthResult> =
        fluxbaseResponse {
            val responseText = http.postWithHeaders(
                "/api/v1/auth/2fa/verify",
                mapOf("user_id" to userId, "code" to code),
            ).body
            val twoFa = json.decodeFromString(TwoFactorLoginResponse.serializer(), responseText)
            val expiresIn = twoFa.expiresIn ?: 3600
            val session = AuthSession(
                user = twoFa.user,
                accessToken = twoFa.accessToken,
                refreshToken = twoFa.refreshToken,
                expiresIn = expiresIn,
                expiresAt = Clock.System.now().toEpochMilliseconds() + expiresIn * 1000,
            )
            setSessionInternal(session, AuthChangeEvent.MFA_CHALLENGE_VERIFIED)
            AuthResult(user = session.user, session = session)
        }

    /** POST `/api/v1/auth/2fa/disable` with `{password}` → disables 2FA. */
    suspend fun disable2FA(password: String): FluxbaseResponse<TwoFactorDisableResponse> =
        fluxbaseResponse {
            http.post("/api/v1/auth/2fa/disable", mapOf("password" to password))
        }

    /** GET `/api/v1/auth/2fa/status` → returns enrolled factors. */
    suspend fun get2FAStatus(): FluxbaseResponse<TwoFactorStatusResponse> =
        fluxbaseResponse {
            http.get("/api/v1/auth/2fa/status")
        }

    // ---- Internal session lifecycle ----

    private fun setSessionInternal(session: AuthSession, event: AuthChangeEvent) {
        currentSession = session
        http.setAuthToken(session.accessToken)
        saveSession(session)
        emitState(event, session)
        if (autoRefresh) startAutoRefresh()
    }

    private fun clearSession() {
        currentSession = null
        http.setAuthToken(null) // Restores anon key
        storage.removeItem(AUTH_STORAGE_KEY)
        emitState(AuthChangeEvent.SIGNED_OUT, null)
    }

    private fun saveSession(session: AuthSession) {
        storage.setItem(
            AUTH_STORAGE_KEY,
            json.encodeToString(AuthSession.serializer(), session),
        )
    }

    private fun restoreSession() {
        val stored = storage.getItem(AUTH_STORAGE_KEY) ?: return
        runCatching {
            val session = json.decodeFromString(AuthSession.serializer(), stored)
            currentSession = session
            http.setAuthToken(session.accessToken)
            // A restored access token is usually already stale (15-minute
            // lifetime vs. days between app launches) — refresh it up front
            // so the first API calls don't race the 401-retry path.
            if (autoRefresh && session.expiresAt != null &&
                Clock.System.now().toEpochMilliseconds() >= session.expiresAt!! - REFRESH_LEAD_MS
            ) {
                refreshScope.launch { refreshSession().getOrNull() }
            }
        }
    }

    private fun emitState(event: AuthChangeEvent, session: AuthSession?) {
        stateListeners.forEach { it(AuthState(event, session)) }
    }
}
