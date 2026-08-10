//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.auth](../index.md)/[FluxbaseAuth](index.md)/[disable2FA](disable2-f-a.md)

# disable2FA

[jvm]\
suspend fun [disable2FA](disable2-f-a.md)(password: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[TwoFactorDisableResponse](../-two-factor-disable-response/index.md)&gt;

POST `/api/v1/auth/2fa/disable` with `{password}` → disables 2FA.
