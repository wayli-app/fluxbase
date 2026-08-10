---
title: "setup2FA"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.auth](../index.md)/[FluxbaseAuth](index.md)/[setup2FA](setup2-f-a.md)

# setup2FA

[jvm]\
suspend fun [setup2FA](setup2-f-a.md)(): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[TwoFactorSetupResponse](../-two-factor-setup-response/index.md)&gt;

POST `/api/v1/auth/2fa/setup` → returns TOTP secret + QR code.
