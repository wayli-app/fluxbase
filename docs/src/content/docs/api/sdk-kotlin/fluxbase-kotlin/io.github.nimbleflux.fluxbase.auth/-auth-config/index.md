---
title: "AuthConfig"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.auth](../index.md)/[AuthConfig](index.md)

# AuthConfig

[jvm]\
@Serializable

data class [AuthConfig](index.md)(val signupEnabled: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = true, val passwordLoginEnabled: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = true, val passwordMinLength: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html) = 8, val passwordRequireUppercase: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false, val passwordRequireLowercase: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false, val passwordRequireNumber: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false, val passwordRequireSpecial: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false, val oauthProviders: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyList(), val samlProviders: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyList(), val emailConfirmationRequired: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false)

Server-side auth configuration returned by `GET /api/v1/auth/config`. Port of the `getAuthConfig()` response shape used by the TS SDK and by Wayli (`web/src/routes/auth/signin/+page.svelte`).

This is the single call that drives all server-driven auth behavior: whether signup is allowed, which OAuth/SAML providers are available, password rules, and CAPTCHA config.

## Constructors

| | |
|---|---|
| [AuthConfig](-auth-config.md) | [jvm]<br>constructor(signupEnabled: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = true, passwordLoginEnabled: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = true, passwordMinLength: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html) = 8, passwordRequireUppercase: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false, passwordRequireLowercase: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false, passwordRequireNumber: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false, passwordRequireSpecial: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false, oauthProviders: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyList(), samlProviders: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyList(), emailConfirmationRequired: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false) |

## Properties

| Name | Summary |
|---|---|
| [emailConfirmationRequired](email-confirmation-required.md) | [jvm]<br>@SerialName(value = &quot;email_confirmation_required&quot;)<br>val [emailConfirmationRequired](email-confirmation-required.md): [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false |
| [oauthProviders](oauth-providers.md) | [jvm]<br>@SerialName(value = &quot;oauth_providers&quot;)<br>val [oauthProviders](oauth-providers.md): [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; |
| [passwordLoginEnabled](password-login-enabled.md) | [jvm]<br>@SerialName(value = &quot;password_login_enabled&quot;)<br>val [passwordLoginEnabled](password-login-enabled.md): [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = true |
| [passwordMinLength](password-min-length.md) | [jvm]<br>@SerialName(value = &quot;password_min_length&quot;)<br>val [passwordMinLength](password-min-length.md): [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html) = 8 |
| [passwordRequireLowercase](password-require-lowercase.md) | [jvm]<br>@SerialName(value = &quot;password_require_lowercase&quot;)<br>val [passwordRequireLowercase](password-require-lowercase.md): [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false |
| [passwordRequireNumber](password-require-number.md) | [jvm]<br>@SerialName(value = &quot;password_require_number&quot;)<br>val [passwordRequireNumber](password-require-number.md): [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false |
| [passwordRequireSpecial](password-require-special.md) | [jvm]<br>@SerialName(value = &quot;password_require_special&quot;)<br>val [passwordRequireSpecial](password-require-special.md): [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false |
| [passwordRequireUppercase](password-require-uppercase.md) | [jvm]<br>@SerialName(value = &quot;password_require_uppercase&quot;)<br>val [passwordRequireUppercase](password-require-uppercase.md): [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false |
| [samlProviders](saml-providers.md) | [jvm]<br>@SerialName(value = &quot;saml_providers&quot;)<br>val [samlProviders](saml-providers.md): [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; |
| [signupEnabled](signup-enabled.md) | [jvm]<br>@SerialName(value = &quot;signup_enabled&quot;)<br>val [signupEnabled](signup-enabled.md): [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = true |
