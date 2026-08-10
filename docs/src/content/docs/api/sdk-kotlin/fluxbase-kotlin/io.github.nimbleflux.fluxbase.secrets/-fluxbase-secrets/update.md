//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.secrets](../index.md)/[FluxbaseSecrets](index.md)/[update](update.md)

# update

[jvm]\
suspend fun [update](update.md)(name: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, description: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[SecretSummary](../-secret-summary/index.md)&gt;

Update a secret's value (creates a new version). PUTs `/api/v1/secrets/by-name/{name}`.
