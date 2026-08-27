---
title: "signOut"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.auth](../../)/[FluxbaseAuth](../)/[signOut](./)

# signOut

[jvm]\
suspend fun [signOut](./)()

Sign out. POSTs to `/api/v1/auth/signout`, clears the session, and restores the anon key on the HTTP client.

Port of `signOut()` in `auth.ts`. Uses postWithHeaders to avoid deserializing the response body (signOut returns no useful data).
