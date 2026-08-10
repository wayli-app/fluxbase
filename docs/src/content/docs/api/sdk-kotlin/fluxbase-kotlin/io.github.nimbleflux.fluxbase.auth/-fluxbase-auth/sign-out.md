//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.auth](../index.md)/[FluxbaseAuth](index.md)/[signOut](sign-out.md)

# signOut

[jvm]\
suspend fun [signOut](sign-out.md)()

Sign out. POSTs to `/api/v1/auth/signout`, clears the session, and restores the anon key on the HTTP client.

Port of `signOut()` in `auth.ts`. Uses postWithHeaders to avoid deserializing the response body (signOut returns no useful data).
