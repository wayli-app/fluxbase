package io.github.nimbleflux.fluxbase.auth

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * An app-login-enabled OAuth provider.
 * Port of `OAuthProviderInfo` from `sdk/src/types.ts:861`.
 */
@Serializable
data class OAuthProviderInfo(
    val provider: String,
    @SerialName("display_name") val displayName: String = "",
    @SerialName("authorize_url") val authorizeUrl: String? = null,
)

/** Options for [FluxbaseAuth.getOAuthUrl]. Port of `OAuthOptions` (types.ts:872). */
data class OAuthOptions(
    /** Post-login redirect URL (where to go after successful login). */
    val redirectTo: String? = null,
    /** OAuth callback URL (where the provider redirects with the code). */
    val redirectUri: String? = null,
    val scopes: List<String> = emptyList(),
)

/**
 * The authorization URL to open in a browser.
 * Port of `OAuthUrlResponse` from `sdk/src/types.ts:878`.
 */
@Serializable
data class OAuthUrlResponse(
    val url: String,
    val provider: String = "",
)

/** Wraps the /oauth/providers list. */
@Serializable
internal data class OAuthProvidersResponse(
    val providers: List<OAuthProviderInfo> = emptyList(),
)
