---
title: "TwoFactorLoginResponse"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.auth](../)/[TwoFactorLoginResponse](./)

# TwoFactorLoginResponse

[jvm]\
@Serializable

data class [TwoFactorLoginResponse](./)(val accessToken: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val refreshToken: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val user: [User](../-user/), val tokenType: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val expiresIn: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? = null)

## Constructors

| | |
|---|---|
| [TwoFactorLoginResponse](-two-factor-login-response/) | [jvm]<br>constructor(accessToken: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), refreshToken: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), user: [User](../-user/), tokenType: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, expiresIn: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? = null) |

## Properties

| Name | Summary |
|---|---|
| [accessToken](access-token/) | [jvm]<br>@SerialName(value = &quot;access_token&quot;)<br>val [accessToken](access-token/): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [expiresIn](expires-in/) | [jvm]<br>@SerialName(value = &quot;expires_in&quot;)<br>val [expiresIn](expires-in/): [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? = null |
| [refreshToken](refresh-token/) | [jvm]<br>@SerialName(value = &quot;refresh_token&quot;)<br>val [refreshToken](refresh-token/): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [tokenType](token-type/) | [jvm]<br>@SerialName(value = &quot;token_type&quot;)<br>val [tokenType](token-type/): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null |
| [user](user/) | [jvm]<br>val [user](user/): [User](../-user/) |
