---
title: "FluxbaseSecrets"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.secrets](../)/[FluxbaseSecrets](./)

# FluxbaseSecrets

[jvm]\
class [FluxbaseSecrets](./)(http: [FluxbaseHttpClient](../../iogithubnimblefluxfluxbasecore/-fluxbase-http-client/))

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
| [FluxbaseSecrets](-fluxbase-secrets/) | [jvm]<br>constructor(http: [FluxbaseHttpClient](../../iogithubnimblefluxfluxbasecore/-fluxbase-http-client/)) |

## Functions

| Name | Summary |
|---|---|
| [delete](delete/) | [jvm]<br>suspend fun [delete](delete/)(name: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)&gt;<br>Delete a secret by name. DELETEs `/api/v1/secrets/by-name/{name}`. |
| [get](get/) | [jvm]<br>suspend fun [get](get/)(name: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[SecretSummary](../-secret-summary/)&gt;<br>Get a secret's metadata by name. GETs `/api/v1/secrets/by-name/{name}`. |
| [list](list/) | [jvm]<br>suspend fun [list](list/)(scope: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[SecretSummary](../-secret-summary/)&gt;&gt;<br>List all secrets (metadata only). GETs `/api/v1/secrets`. |
| [set](set/) | [jvm]<br>suspend fun [set](set/)(name: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), scope: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;global&quot;, namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, description: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[SecretSummary](../-secret-summary/)&gt;<br>Create a secret. POSTs `/api/v1/secrets`. |
| [update](update/) | [jvm]<br>suspend fun [update](update/)(name: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, description: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[SecretSummary](../-secret-summary/)&gt;<br>Update a secret's value (creates a new version). PUTs `/api/v1/secrets/by-name/{name}`. |
