---
title: "SignInWith2FaResponse"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.auth](../index.md)/[SignInWith2FaResponse](index.md)

# SignInWith2FaResponse

[jvm]\
@Serializable

data class [SignInWith2FaResponse](index.md)(val requires2fa: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html), val userId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val message: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;&quot;)

Returned by `signIn` when 2FA is required. Port of `SignInWith2FAResponse` from `sdk/src/types.ts:229`.

## Constructors

| | |
|---|---|
| [SignInWith2FaResponse](-sign-in-with2-fa-response.md) | [jvm]<br>constructor(requires2fa: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html), userId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), message: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;&quot;) |

## Properties

| Name | Summary |
|---|---|
| [message](message.md) | [jvm]<br>val [message](message.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [requires2fa](requires2fa.md) | [jvm]<br>@SerialName(value = &quot;requires_2fa&quot;)<br>val [requires2fa](requires2fa.md): [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) |
| [userId](user-id.md) | [jvm]<br>@SerialName(value = &quot;user_id&quot;)<br>val [userId](user-id.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
