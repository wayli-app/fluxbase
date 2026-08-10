---
title: "verify2FA"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.auth](../index.md)/[FluxbaseAuth](index.md)/[verify2FA](verify2-f-a.md)

# verify2FA

[jvm]\
suspend fun [verify2FA](verify2-f-a.md)(userId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), code: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[AuthResult](../-auth-result/index.md)&gt;

POST `/api/v1/auth/2fa/verify` with `{user_id, code}` — completes a 2FA login challenge. On success, establishes a session from the returned tokens.
