//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.auth](../index.md)/[FluxbaseAuth](index.md)/[signIn](sign-in.md)

# signIn

[jvm]\
suspend fun [signIn](sign-in.md)(email: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), password: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[AuthResult](../-auth-result/index.md)&gt;

Sign in with email and password.

POSTs to `/api/v1/auth/signin` with `{email, password}`. If the server returns `requires_2fa: true`, returns an [AuthResult](../-auth-result/index.md) with [AuthResult.is2faRequired](../-auth-result/is2fa-required.md) set — the caller should then call [verify2FA](verify2-f-a.md).

Port of `signIn()` in `auth.ts:265-313`.
