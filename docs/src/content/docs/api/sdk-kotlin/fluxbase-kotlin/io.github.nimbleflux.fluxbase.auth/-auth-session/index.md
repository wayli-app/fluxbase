---
title: "AuthSession"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.auth](../index.md)/[AuthSession](index.md)

# AuthSession

[jvm]\
@Serializable

data class [AuthSession](index.md)(val user: [User](../-user/index.md), val accessToken: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val refreshToken: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val expiresIn: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html), val expiresAt: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? = null)

An authenticated session. Port of `AuthSession` from `sdk/src/types.ts:94`.

`expires_at` is computed client-side as `now + expires_in * 1000` (ms epoch), matching the TS SDK (`auth.ts:305`).

## Constructors

| | |
|---|---|
| [AuthSession](-auth-session.md) | [jvm]<br>constructor(user: [User](../-user/index.md), accessToken: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), refreshToken: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), expiresIn: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html), expiresAt: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? = null) |

## Properties

| Name | Summary |
|---|---|
| [accessToken](access-token.md) | [jvm]<br>@SerialName(value = &quot;access_token&quot;)<br>val [accessToken](access-token.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [expiresAt](expires-at.md) | [jvm]<br>@SerialName(value = &quot;expires_at&quot;)<br>val [expiresAt](expires-at.md): [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? = null<br>Milliseconds since epoch. Computed client-side on sign-in/refresh. |
| [expiresIn](expires-in.md) | [jvm]<br>@SerialName(value = &quot;expires_in&quot;)<br>val [expiresIn](expires-in.md): [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html) |
| [refreshToken](refresh-token.md) | [jvm]<br>@SerialName(value = &quot;refresh_token&quot;)<br>val [refreshToken](refresh-token.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [user](user.md) | [jvm]<br>val [user](user.md): [User](../-user/index.md) |
