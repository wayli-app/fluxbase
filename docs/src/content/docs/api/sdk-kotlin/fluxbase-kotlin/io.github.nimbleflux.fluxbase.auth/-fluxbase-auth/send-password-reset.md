//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.auth](../index.md)/[FluxbaseAuth](index.md)/[sendPasswordReset](send-password-reset.md)

# sendPasswordReset

[jvm]\
suspend fun [sendPasswordReset](send-password-reset.md)(email: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)&gt;

Send a password reset email. POSTs `/api/v1/auth/password/reset`. Port of `sendPasswordReset()` in `auth.ts`.
