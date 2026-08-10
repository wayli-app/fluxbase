---
title: "refreshSession"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.auth](../index.md)/[FluxbaseAuth](index.md)/[refreshSession](refresh-session.md)

# refreshSession

[jvm]\
suspend fun [refreshSession](refresh-session.md)(): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[AuthSession](../-auth-session/index.md)&gt;

Refresh the current session using the stored refresh token. POSTs to `/api/v1/auth/refresh` with `{refresh_token}`. On success, updates the session and emits `TOKEN_REFRESHED`.

Port of `refreshSession()` in `auth.ts`.
