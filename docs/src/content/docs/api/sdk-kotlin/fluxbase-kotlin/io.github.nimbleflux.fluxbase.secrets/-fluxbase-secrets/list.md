//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.secrets](../index.md)/[FluxbaseSecrets](index.md)/[list](list.md)

# list

[jvm]\
suspend fun [list](list.md)(scope: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[SecretSummary](../-secret-summary/index.md)&gt;&gt;

List all secrets (metadata only). GETs `/api/v1/secrets`.
