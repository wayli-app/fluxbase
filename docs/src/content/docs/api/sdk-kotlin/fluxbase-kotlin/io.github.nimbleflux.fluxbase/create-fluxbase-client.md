---
title: "createFluxbaseClient"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase](../)/[createFluxbaseClient](./)

# createFluxbaseClient

[jvm]\
fun [createFluxbaseClient](./)(url: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, options: [FluxbaseClientOptions](../-fluxbase-client-options/) = FluxbaseClientOptions(), transport: [HttpTransport](../../iogithubnimblefluxfluxbasecore/-http-transport/)? = null): [FluxbaseClient](../-fluxbase-client/)

Top-level factory function — Kotlin-idiomatic equivalent of the TS `createClient(url, key, options)`. Delegates to [FluxbaseClient.create](../-fluxbase-client/-companion/create/).
