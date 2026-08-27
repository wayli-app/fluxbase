---
title: "sendPasswordReset"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.auth](../../)/[FluxbaseAuth](../)/[sendPasswordReset](./)

# sendPasswordReset

[jvm]\
suspend fun [sendPasswordReset](./)(email: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)&gt;

Send a password reset email. POSTs `/api/v1/auth/password/reset`. Port of `sendPasswordReset()` in `auth.ts`.
