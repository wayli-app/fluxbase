//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.secrets](../index.md)/[FluxbaseSecrets](index.md)

# FluxbaseSecrets

[jvm]\
class [FluxbaseSecrets](index.md)(http: [FluxbaseHttpClient](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md))

Encrypted Secrets module — port of `SecretsManager` from `sdk/src/secrets.ts`.

Fluxbase's secrets are encrypted at rest with `FLUXBASE_ENCRYPTION_KEY` and injected into function/job runtime environments. Values are never returned by the API — only metadata.

Usage:

```kotlin
client.secrets.set("my-api-key", "secret-value", scope = "namespace", namespace = "wayli")
val (secrets, _) = client.secrets.list(namespace = "wayli")
```

## Constructors

| | |
|---|---|
| [FluxbaseSecrets](-fluxbase-secrets.md) | [jvm]<br>constructor(http: [FluxbaseHttpClient](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md)) |

## Functions

| Name | Summary |
|---|---|
| [delete](delete.md) | [jvm]<br>suspend fun [delete](delete.md)(name: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)&gt;<br>Delete a secret by name. DELETEs `/api/v1/secrets/by-name/{name}`. |
| [get](get.md) | [jvm]<br>suspend fun [get](get.md)(name: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[SecretSummary](../-secret-summary/index.md)&gt;<br>Get a secret's metadata by name. GETs `/api/v1/secrets/by-name/{name}`. |
| [list](list.md) | [jvm]<br>suspend fun [list](list.md)(scope: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[SecretSummary](../-secret-summary/index.md)&gt;&gt;<br>List all secrets (metadata only). GETs `/api/v1/secrets`. |
| [set](set.md) | [jvm]<br>suspend fun [set](set.md)(name: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), scope: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;global&quot;, namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, description: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[SecretSummary](../-secret-summary/index.md)&gt;<br>Create a secret. POSTs `/api/v1/secrets`. |
| [update](update.md) | [jvm]<br>suspend fun [update](update.md)(name: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, description: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[SecretSummary](../-secret-summary/index.md)&gt;<br>Update a secret's value (creates a new version). PUTs `/api/v1/secrets/by-name/{name}`. |
