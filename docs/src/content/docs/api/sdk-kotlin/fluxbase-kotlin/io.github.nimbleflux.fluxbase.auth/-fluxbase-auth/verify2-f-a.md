---
title: "verify2FA"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.auth](../../)/[FluxbaseAuth](../)/[verify2FA](./)

# verify2FA

[jvm]\
suspend fun [verify2FA](./)(userId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), code: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[AuthResult](../../-auth-result/)&gt;

POST `/api/v1/auth/2fa/verify` with `{user_id, code}` — completes a 2FA login challenge. On success, establishes a session from the returned tokens.
