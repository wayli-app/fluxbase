//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.core](../index.md)/[FluxbaseHttpClient](index.md)/[setAuthToken](set-auth-token.md)

# setAuthToken

[jvm]\
fun [setAuthToken](set-auth-token.md)(token: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)?)

Update the authorization token.

- 
   With a non-null token: sets `Authorization: Bearer <token>`.
- 
   With null: restores the anon key if one is set (does NOT remove the header). This is how sign-out falls back to anonymous access — see `fetch.ts:93-102`.
