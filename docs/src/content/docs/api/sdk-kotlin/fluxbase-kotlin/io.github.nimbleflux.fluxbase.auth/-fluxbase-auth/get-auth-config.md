---
title: "getAuthConfig"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.auth](../index.md)/[FluxbaseAuth](index.md)/[getAuthConfig](get-auth-config.md)

# getAuthConfig

[jvm]\
suspend fun [getAuthConfig](get-auth-config.md)(): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[AuthConfig](../-auth-config/index.md)&gt;

Get the server's auth configuration (signup enabled, OAuth providers, password rules). GETs `/api/v1/auth/config`. Port of `getAuthConfig()` in `auth.ts`.
