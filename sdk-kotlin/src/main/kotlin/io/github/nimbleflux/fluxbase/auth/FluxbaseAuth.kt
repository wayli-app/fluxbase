package io.github.nimbleflux.fluxbase.auth

import io.github.nimbleflux.fluxbase.FluxbaseError
import io.github.nimbleflux.fluxbase.FluxbaseResponse
import io.github.nimbleflux.fluxbase.core.FluxbaseHttpClient
import io.github.nimbleflux.fluxbase.fluxbaseResponse
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

    init {
        // Restore session from storage (mirrors `auth.ts:172-187`).
        restoreSession()
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
        }
    }

    private fun emitState(event: AuthChangeEvent, session: AuthSession?) {
        stateListeners.forEach { it(AuthState(event, session)) }
    }
}
