---
title: "setup2FA"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.auth](../../)/[FluxbaseAuth](../)/[setup2FA](./)

# setup2FA

[jvm]\
suspend fun [setup2FA](./)(): [FluxbaseResponse](../../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[TwoFactorSetupResponse](../../-two-factor-setup-response/)&gt;

POST `/api/v1/auth/2fa/setup` → returns TOTP secret + QR code.
