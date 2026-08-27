---
title: "FluxbaseAuth"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.auth](../../)/[FluxbaseAuth](../)/[FluxbaseAuth](./)

# FluxbaseAuth

[jvm]\
constructor(http: [FluxbaseHttpClient](../../../iogithubnimblefluxfluxbasecore/-fluxbase-http-client/), autoRefresh: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = true, storage: [StorageAdapter](../../-storage-adapter/) = MemoryStorage())

#### Parameters

jvm

| | |
|---|---|
| http | the shared [FluxbaseHttpClient](../../../iogithubnimblefluxfluxbasecore/-fluxbase-http-client/) for making API calls. |
| autoRefresh | whether to automatically refresh the token before expiry (default true; disabled in tests). TS default is true (`auth.ts:55`). |
| storage | the [StorageAdapter](../../-storage-adapter/) for session persistence. |
