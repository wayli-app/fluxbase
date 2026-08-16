---
title: "enable2FA"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.auth](../index.md)/[FluxbaseAuth](index.md)/[enable2FA](enable2-f-a.md)

# enable2FA

[jvm]\
suspend fun [enable2FA](enable2-f-a.md)(code: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[TwoFactorEnableResponse](../-two-factor-enable-response/index.md)&gt;

POST `/api/v1/auth/2fa/enable` with `{code}` → enables 2FA, returns backup codes.
