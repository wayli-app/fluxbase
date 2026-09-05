package io.github.nimbleflux.fluxbase.auth

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement

/**
 * A Fluxbase user account. Port of `User` from `sdk/src/types.ts:102`.
 */
@Serializable
data class User(
    val id: String,
    val email: String,
    @SerialName("email_verified") val emailVerified: Boolean = false,
    val role: String = "user",
    val metadata: JsonElement? = null,
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("updated_at") val updatedAt: String = "",
)

/**
 * An authenticated session. Port of `AuthSession` from `sdk/src/types.ts:94`.
 *
 * `expires_at` is computed client-side as `now + expires_in * 1000` (ms epoch),
 * matching the TS SDK (`auth.ts:305`).
 */
@Serializable
data class AuthSession(
    val user: User,
    @SerialName("access_token") val accessToken: String,
    @SerialName("refresh_token") val refreshToken: String,
    @SerialName("expires_in") val expiresIn: Long,
    /** Milliseconds since epoch. Computed client-side on sign-in/refresh. */
    @SerialName("expires_at") val expiresAt: Long? = null,
)

/**
 * Raw auth response from the server (signin/signup/refresh).
 * Port of `AuthResponse` from `sdk/src/types.ts:152`.
 * The client converts this to [AuthSession] by computing `expires_at`.
 */
@Serializable
internal data class AuthResponse(
    // Nullable: some server versions omit `user` from the /auth/refresh
    // response — the caller then carries the signed-in user forward.
    val user: User? = null,
    @SerialName("access_token") val accessToken: String,
    @SerialName("refresh_token") val refreshToken: String,
    @SerialName("expires_in") val expiresIn: Long,
)

/**
 * Returned by `signIn` when 2FA is required.
 * Port of `SignInWith2FAResponse` from `sdk/src/types.ts:229`.
 */
@Serializable
data class SignInWith2FaResponse(
    @SerialName("requires_2fa") val requires2fa: Boolean,
    @SerialName("user_id") val userId: String,
    val message: String = "",
)

// ---- 2FA types (port of `sdk/src/types.ts:162-233`) ----

@Serializable
data class Factor(
    val id: String,
    val type: String = "totp",
    val status: String = "verified",
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("updated_at") val updatedAt: String = "",
    @SerialName("friendly_name") val friendlyName: String? = null,
)

@Serializable
data class TotpSetup(
    @SerialName("qr_code") val qrCode: String,
    val secret: String,
    val uri: String,
)

@Serializable
data class TwoFactorSetupResponse(
    val id: String,
    val type: String = "totp",
    val totp: TotpSetup,
)

@Serializable
data class TwoFactorEnableResponse(
    val success: Boolean,
    @SerialName("backup_codes") val backupCodes: List<String> = emptyList(),
    val message: String = "",
)

@Serializable
data class TwoFactorLoginResponse(
    @SerialName("access_token") val accessToken: String,
    @SerialName("refresh_token") val refreshToken: String,
    val user: User,
    @SerialName("token_type") val tokenType: String? = null,
    @SerialName("expires_in") val expiresIn: Long? = null,
)

@Serializable
data class TwoFactorStatusResponse(
    val all: List<Factor> = emptyList(),
    val totp: List<Factor> = emptyList(),
)

@Serializable
data class TwoFactorDisableResponse(val id: String)
