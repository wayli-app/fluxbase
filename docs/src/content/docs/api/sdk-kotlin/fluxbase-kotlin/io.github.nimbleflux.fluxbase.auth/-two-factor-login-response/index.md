//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.auth](../index.md)/[TwoFactorLoginResponse](index.md)

# TwoFactorLoginResponse

[jvm]\
@Serializable

data class [TwoFactorLoginResponse](index.md)(val accessToken: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val refreshToken: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val user: [User](../-user/index.md), val tokenType: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val expiresIn: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? = null)

## Constructors

| | |
|---|---|
| [TwoFactorLoginResponse](-two-factor-login-response.md) | [jvm]<br>constructor(accessToken: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), refreshToken: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), user: [User](../-user/index.md), tokenType: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, expiresIn: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? = null) |

## Properties

| Name | Summary |
|---|---|
| [accessToken](access-token.md) | [jvm]<br>@SerialName(value = &quot;access_token&quot;)<br>val [accessToken](access-token.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [expiresIn](expires-in.md) | [jvm]<br>@SerialName(value = &quot;expires_in&quot;)<br>val [expiresIn](expires-in.md): [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? = null |
| [refreshToken](refresh-token.md) | [jvm]<br>@SerialName(value = &quot;refresh_token&quot;)<br>val [refreshToken](refresh-token.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [tokenType](token-type.md) | [jvm]<br>@SerialName(value = &quot;token_type&quot;)<br>val [tokenType](token-type.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null |
| [user](user.md) | [jvm]<br>val [user](user.md): [User](../-user/index.md) |
