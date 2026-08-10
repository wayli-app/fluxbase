---
title: "signUp"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.auth](../index.md)/[FluxbaseAuth](index.md)/[signUp](sign-up.md)

# signUp

[jvm]\
suspend fun [signUp](sign-up.md)(email: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), password: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[AuthResult](../-auth-result/index.md)&gt;

Sign up with email and password.

POSTs to `/api/v1/auth/signup`. If email confirmation is disabled, the server returns tokens and a session is established. If email confirmation is required, only the user is returned (no session).

Port of `signUp()` in `auth.ts:331-378`.
