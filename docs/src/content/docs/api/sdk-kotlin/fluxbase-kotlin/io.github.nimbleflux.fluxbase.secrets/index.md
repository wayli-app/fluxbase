---
title: "Package-level declarations"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../)/[io.github.nimbleflux.fluxbase.secrets](./)

# Package-level declarations

## Types

| Name | Summary |
|---|---|
| [FluxbaseSecrets](-fluxbase-secrets/) | [jvm]<br>class [FluxbaseSecrets](-fluxbase-secrets/)(http: [FluxbaseHttpClient](../iogithubnimblefluxfluxbasecore/-fluxbase-http-client/))<br>Encrypted Secrets module — port of `SecretsManager` from `sdk/src/secrets.ts`. |
| [SecretSummary](-secret-summary/) | [jvm]<br>@Serializable<br>data class [SecretSummary](-secret-summary/)(val id: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val name: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val scope: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;global&quot;, val namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val description: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val currentVersion: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html) = 1, val createdAt: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;&quot;, val updatedAt: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;&quot;)<br>A secret summary (metadata only — values are never returned by the API). Port of `SecretSummary` from `sdk/src/secrets.ts`. |
