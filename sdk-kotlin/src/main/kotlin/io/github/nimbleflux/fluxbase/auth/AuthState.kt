package io.github.nimbleflux.fluxbase.auth

/**
 * Auth state change events. Port of `AuthChangeEvent` from
 * `sdk/src/types.ts:2374`.
 *
 * Emitted via `client.auth.authStateChanges` (a Kotlin `Flow`) whenever the
 * session changes. The TS SDK uses `onAuthStateChange(callback)`; in Kotlin we
 * expose the same events as a `Flow` for coroutine-native consumption.
 */
enum class AuthChangeEvent {
    SIGNED_IN,
    SIGNED_OUT,
    TOKEN_REFRESHED,
    USER_UPDATED,
    PASSWORD_RECOVERY,
    MFA_CHALLENGE_VERIFIED,
}

/**
 * A single auth state change event: the [event] type plus the current session
 * (null after SIGNED_OUT).
 */
data class AuthState(
    val event: AuthChangeEvent,
    val session: AuthSession?,
)
