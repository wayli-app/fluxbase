---
title: "refreshSession"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.auth](../../)/[FluxbaseAuth](../)/[refreshSession](./)

# refreshSession

[jvm]\
suspend fun [refreshSession](./)(): [FluxbaseResponse](../../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[AuthSession](../../-auth-session/)&gt;

Refresh the current session using the stored refresh token. POSTs to `/api/v1/auth/refresh` with `{refresh_token}`. On success, updates the session and emits `TOKEN_REFRESHED`.

Port of `refreshSession()` in `auth.ts`.
