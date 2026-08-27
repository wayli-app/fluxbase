---
title: "enable2FA"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.auth](../../)/[FluxbaseAuth](../)/[enable2FA](./)

# enable2FA

[jvm]\
suspend fun [enable2FA](./)(code: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[TwoFactorEnableResponse](../../-two-factor-enable-response/)&gt;

POST `/api/v1/auth/2fa/enable` with `{code}` → enables 2FA, returns backup codes.
