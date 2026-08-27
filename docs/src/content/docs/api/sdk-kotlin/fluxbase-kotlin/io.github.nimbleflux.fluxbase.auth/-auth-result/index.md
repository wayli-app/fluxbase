---
title: "AuthResult"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.auth](../)/[AuthResult](./)

# AuthResult

[jvm]\
data class [AuthResult](./)(val user: [User](../-user/)? = null, val session: [AuthSession](../-auth-session/)? = null, val is2faRequired: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false, val userId2fa: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null)

The result of a sign-in or sign-up call.

On success with a session: [session](session/) is non-null, [user](user/) is set. On 2FA challenge: [is2faRequired](is2fa-required/) is true, [userId2fa](user-id2fa/) holds the user id, [session](session/) is null. On error: the [FluxbaseResponse.Error](../../iogithubnimblefluxfluxbase/-fluxbase-response/-error/) variant is returned.

Port of the TS `AuthResponseData` (`types.ts:3923`) which returns `{ user, session: AuthSession | null }`, combined with the 2FA branch that returns `SignInWith2FAResponse` instead.

## Constructors

| | |
|---|---|
| [AuthResult](-auth-result/) | [jvm]<br>constructor(user: [User](../-user/)? = null, session: [AuthSession](../-auth-session/)? = null, is2faRequired: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false, userId2fa: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null) |

## Properties

| Name | Summary |
|---|---|
| [is2faRequired](is2fa-required/) | [jvm]<br>val [is2faRequired](is2fa-required/): [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false |
| [session](session/) | [jvm]<br>val [session](session/): [AuthSession](../-auth-session/)? = null |
| [user](user/) | [jvm]<br>val [user](user/): [User](../-user/)? = null |
| [userId2fa](user-id2fa/) | [jvm]<br>val [userId2fa](user-id2fa/): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null |
