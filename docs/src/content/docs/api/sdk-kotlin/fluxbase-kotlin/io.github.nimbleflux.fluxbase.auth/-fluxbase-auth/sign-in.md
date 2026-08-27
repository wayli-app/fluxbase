---
title: "signIn"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.auth](../../)/[FluxbaseAuth](../)/[signIn](./)

# signIn

[jvm]\
suspend fun [signIn](./)(email: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), password: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[AuthResult](../../-auth-result/)&gt;

Sign in with email and password.

POSTs to `/api/v1/auth/signin` with `{email, password}`. If the server returns `requires_2fa: true`, returns an [AuthResult](../../-auth-result/) with [AuthResult.is2faRequired](../../-auth-result/is2fa-required/) set — the caller should then call [verify2FA](../verify2-f-a/).

Port of `signIn()` in `auth.ts:265-313`.
