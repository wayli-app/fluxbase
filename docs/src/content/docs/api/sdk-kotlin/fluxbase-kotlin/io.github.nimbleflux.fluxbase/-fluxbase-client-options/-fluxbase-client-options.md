---
title: "FluxbaseClientOptions"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase](../../)/[FluxbaseClientOptions](../)/[FluxbaseClientOptions](./)

# FluxbaseClientOptions

[jvm]\
constructor(autoRefresh: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = true, persist: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = true, storage: [StorageAdapter](../../../iogithubnimblefluxfluxbaseauth/-storage-adapter/) = MemoryStorage(), headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyMap(), timeout: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html), debug: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false)

#### Parameters

jvm

| | |
|---|---|
| autoRefresh | whether to automatically refresh the JWT before expiry (default true). |
| persist | whether to persist the session to the storage adapter (default true). |
| storage | custom session persistence (default: [MemoryStorage](../../../iogithubnimblefluxfluxbaseauth/-memory-storage/)). |
| headers | additional default headers on every request. |
| timeout | request timeout in milliseconds (default 30000). |
| debug | enable verbose logging (default false). |
