---
title: "disable2FA"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.auth](../../)/[FluxbaseAuth](../)/[disable2FA](./)

# disable2FA

[jvm]\
suspend fun [disable2FA](./)(password: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[TwoFactorDisableResponse](../../-two-factor-disable-response/)&gt;

POST `/api/v1/auth/2fa/disable` with `{password}` → disables 2FA.
