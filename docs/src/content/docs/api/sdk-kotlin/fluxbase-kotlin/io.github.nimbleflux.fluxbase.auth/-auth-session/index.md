---
title: "AuthSession"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.auth](../)/[AuthSession](./)

# AuthSession

[jvm]\
@Serializable

data class [AuthSession](./)(val user: [User](../-user/), val accessToken: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val refreshToken: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val expiresIn: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html), val expiresAt: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? = null)

An authenticated session. Port of `AuthSession` from `sdk/src/types.ts:94`.

`expires_at` is computed client-side as `now + expires_in * 1000` (ms epoch), matching the TS SDK (`auth.ts:305`).

## Constructors

| | |
|---|---|
| [AuthSession](-auth-session/) | [jvm]<br>constructor(user: [User](../-user/), accessToken: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), refreshToken: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), expiresIn: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html), expiresAt: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? = null) |

## Properties

| Name | Summary |
|---|---|
| [accessToken](access-token/) | [jvm]<br>@SerialName(value = &quot;access_token&quot;)<br>val [accessToken](access-token/): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [expiresAt](expires-at/) | [jvm]<br>@SerialName(value = &quot;expires_at&quot;)<br>val [expiresAt](expires-at/): [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? = null<br>Milliseconds since epoch. Computed client-side on sign-in/refresh. |
| [expiresIn](expires-in/) | [jvm]<br>@SerialName(value = &quot;expires_in&quot;)<br>val [expiresIn](expires-in/): [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html) |
| [refreshToken](refresh-token/) | [jvm]<br>@SerialName(value = &quot;refresh_token&quot;)<br>val [refreshToken](refresh-token/): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [user](user/) | [jvm]<br>val [user](user/): [User](../-user/) |
