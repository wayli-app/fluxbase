---
title: "get"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.secrets](../../)/[FluxbaseSecrets](../)/[get](./)

# get

[jvm]\
suspend fun [get](./)(name: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null): [FluxbaseResponse](../../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[SecretSummary](../../-secret-summary/)&gt;

Get a secret's metadata by name. GETs `/api/v1/secrets/by-name/{name}`.
