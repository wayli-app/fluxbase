package io.github.nimbleflux.fluxbase.auth

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * Server-side auth configuration returned by `GET /api/v1/auth/config`.
 * Port of the `getAuthConfig()` response shape used by the TS SDK and by Wayli
 * (`web/src/routes/auth/signin/+page.svelte`).
 *
 * This is the single call that drives all server-driven auth behavior:
 * whether signup is allowed, which OAuth/SAML providers are available, password
 * rules, and CAPTCHA config.
 */
@Serializable
data class AuthConfig(
    @SerialName("signup_enabled") val signupEnabled: Boolean = true,
    @SerialName("password_login_enabled") val passwordLoginEnabled: Boolean = true,
    @SerialName("password_min_length") val passwordMinLength: Int = 8,
    @SerialName("password_require_uppercase") val passwordRequireUppercase: Boolean = false,
    @SerialName("password_require_lowercase") val passwordRequireLowercase: Boolean = false,
    @SerialName("password_require_number") val passwordRequireNumber: Boolean = false,
    @SerialName("password_require_special") val passwordRequireSpecial: Boolean = false,
    @SerialName("oauth_providers") val oauthProviders: List<String> = emptyList(),
    @SerialName("saml_providers") val samlProviders: List<String> = emptyList(),
    @SerialName("email_confirmation_required") val emailConfirmationRequired: Boolean = false,
    /**
     * Publishable anon key for this instance (the same key the web app
     * exposes to browsers). Lets native clients connect without manual key
     * entry. Null on servers that don't publish it.
     */
    @SerialName("anon_key") val anonKey: String? = null,
)
