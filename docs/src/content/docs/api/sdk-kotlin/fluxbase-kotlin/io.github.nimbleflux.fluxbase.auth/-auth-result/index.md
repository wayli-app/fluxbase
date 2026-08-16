---
title: "AuthResult"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.auth](../index.md)/[AuthResult](index.md)

# AuthResult

[jvm]\
data class [AuthResult](index.md)(val user: [User](../-user/index.md)? = null, val session: [AuthSession](../-auth-session/index.md)? = null, val is2faRequired: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false, val userId2fa: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null)

The result of a sign-in or sign-up call.

On success with a session: [session](session.md) is non-null, [user](user.md) is set. On 2FA challenge: [is2faRequired](is2fa-required.md) is true, [userId2fa](user-id2fa.md) holds the user id, [session](session.md) is null. On error: the [FluxbaseResponse.Error](../../io.github.nimbleflux.fluxbase/-fluxbase-response/-error/index.md) variant is returned.

Port of the TS `AuthResponseData` (`types.ts:3923`) which returns `{ user, session: AuthSession | null }`, combined with the 2FA branch that returns `SignInWith2FAResponse` instead.

## Constructors

| | |
|---|---|
| [AuthResult](-auth-result.md) | [jvm]<br>constructor(user: [User](../-user/index.md)? = null, session: [AuthSession](../-auth-session/index.md)? = null, is2faRequired: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false, userId2fa: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null) |

## Properties

| Name | Summary |
|---|---|
| [is2faRequired](is2fa-required.md) | [jvm]<br>val [is2faRequired](is2fa-required.md): [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false |
| [session](session.md) | [jvm]<br>val [session](session.md): [AuthSession](../-auth-session/index.md)? = null |
| [user](user.md) | [jvm]<br>val [user](user.md): [User](../-user/index.md)? = null |
| [userId2fa](user-id2fa.md) | [jvm]<br>val [userId2fa](user-id2fa.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null |
