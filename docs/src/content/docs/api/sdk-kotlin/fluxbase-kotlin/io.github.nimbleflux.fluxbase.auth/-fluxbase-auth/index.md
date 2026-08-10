---
title: "FluxbaseAuth"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.auth](../index.md)/[FluxbaseAuth](index.md)

# FluxbaseAuth

class [FluxbaseAuth](index.md)(http: [FluxbaseHttpClient](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md), autoRefresh: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = true, storage: [StorageAdapter](../-storage-adapter/index.md) = MemoryStorage())

Authentication module — port of `FluxbaseAuth` from `sdk/src/auth.ts`.

Manages the user session: sign in/up/out, session persistence, and auth state change events. The session is stored via a [StorageAdapter](../-storage-adapter/index.md) (default: [MemoryStorage](../-memory-storage/index.md) for JVM; Android injects an EncryptedSharedPreferences-backed implementation).

#### Parameters

jvm

| | |
|---|---|
| http | the shared [FluxbaseHttpClient](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md) for making API calls. |
| autoRefresh | whether to automatically refresh the token before expiry (default true; disabled in tests). TS default is true (`auth.ts:55`). |
| storage | the [StorageAdapter](../-storage-adapter/index.md) for session persistence. |

## Constructors

| | |
|---|---|
| [FluxbaseAuth](-fluxbase-auth.md) | [jvm]<br>constructor(http: [FluxbaseHttpClient](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md), autoRefresh: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = true, storage: [StorageAdapter](../-storage-adapter/index.md) = MemoryStorage()) |

## Properties

| Name | Summary |
|---|---|
| [currentSession](current-session.md) | [jvm]<br>var [currentSession](current-session.md): [AuthSession](../-auth-session/index.md)?<br>The current session, or null if not authenticated. |
| [currentUser](current-user.md) | [jvm]<br>val [currentUser](current-user.md): [User](../-user/index.md)?<br>The current user, or null if not authenticated. |

## Functions

| Name | Summary |
|---|---|
| [disable2FA](disable2-f-a.md) | [jvm]<br>suspend fun [disable2FA](disable2-f-a.md)(password: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[TwoFactorDisableResponse](../-two-factor-disable-response/index.md)&gt;<br>POST `/api/v1/auth/2fa/disable` with `{password}` → disables 2FA. |
| [enable2FA](enable2-f-a.md) | [jvm]<br>suspend fun [enable2FA](enable2-f-a.md)(code: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[TwoFactorEnableResponse](../-two-factor-enable-response/index.md)&gt;<br>POST `/api/v1/auth/2fa/enable` with `{code}` → enables 2FA, returns backup codes. |
| [get2FAStatus](get2-f-a-status.md) | [jvm]<br>suspend fun [get2FAStatus](get2-f-a-status.md)(): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[TwoFactorStatusResponse](../-two-factor-status-response/index.md)&gt;<br>GET `/api/v1/auth/2fa/status` → returns enrolled factors. |
| [getAuthConfig](get-auth-config.md) | [jvm]<br>suspend fun [getAuthConfig](get-auth-config.md)(): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[AuthConfig](../-auth-config/index.md)&gt;<br>Get the server's auth configuration (signup enabled, OAuth providers, password rules). GETs `/api/v1/auth/config`. Port of `getAuthConfig()` in `auth.ts`. |
| [getCurrentUser](get-current-user.md) | [jvm]<br>suspend fun [getCurrentUser](get-current-user.md)(): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[User](../-user/index.md)&gt;<br>Get the current user from the server. GETs `/api/v1/auth/user`. Port of `getCurrentUser()` in `auth.ts`. |
| [onAuthStateChange](on-auth-state-change.md) | [jvm]<br>fun [onAuthStateChange](on-auth-state-change.md)(callback: ([AuthState](../-auth-state/index.md)) -&gt; [Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)): () -&gt; [Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)<br>Register a callback for auth state changes. Returns a function to unsubscribe. |
| [refreshSession](refresh-session.md) | [jvm]<br>suspend fun [refreshSession](refresh-session.md)(): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[AuthSession](../-auth-session/index.md)&gt;<br>Refresh the current session using the stored refresh token. POSTs to `/api/v1/auth/refresh` with `{refresh_token}`. On success, updates the session and emits `TOKEN_REFRESHED`. |
| [sendPasswordReset](send-password-reset.md) | [jvm]<br>suspend fun [sendPasswordReset](send-password-reset.md)(email: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)&gt;<br>Send a password reset email. POSTs `/api/v1/auth/password/reset`. Port of `sendPasswordReset()` in `auth.ts`. |
| [setup2FA](setup2-f-a.md) | [jvm]<br>suspend fun [setup2FA](setup2-f-a.md)(): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[TwoFactorSetupResponse](../-two-factor-setup-response/index.md)&gt;<br>POST `/api/v1/auth/2fa/setup` → returns TOTP secret + QR code. |
| [signIn](sign-in.md) | [jvm]<br>suspend fun [signIn](sign-in.md)(email: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), password: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[AuthResult](../-auth-result/index.md)&gt;<br>Sign in with email and password. |
| [signInWithPassword](sign-in-with-password.md) | [jvm]<br>suspend fun [signInWithPassword](sign-in-with-password.md)(email: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), password: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[AuthResult](../-auth-result/index.md)&gt;<br>Alias for [signIn](sign-in.md) — Supabase-compatible method name. |
| [signOut](sign-out.md) | [jvm]<br>suspend fun [signOut](sign-out.md)()<br>Sign out. POSTs to `/api/v1/auth/signout`, clears the session, and restores the anon key on the HTTP client. |
| [signUp](sign-up.md) | [jvm]<br>suspend fun [signUp](sign-up.md)(email: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), password: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[AuthResult](../-auth-result/index.md)&gt;<br>Sign up with email and password. |
| [updateUser](update-user.md) | [jvm]<br>suspend fun [updateUser](update-user.md)(attributes: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)&gt;): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[User](../-user/index.md)&gt;<br>Update the user's attributes. PATCHes `/api/v1/auth/user`. Port of `updateUser()` in `auth.ts`. |
| [verify2FA](verify2-f-a.md) | [jvm]<br>suspend fun [verify2FA](verify2-f-a.md)(userId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), code: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[AuthResult](../-auth-result/index.md)&gt;<br>POST `/api/v1/auth/2fa/verify` with `{user_id, code}` — completes a 2FA login challenge. On success, establishes a session from the returned tokens. |
