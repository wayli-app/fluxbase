---
title: "FluxbaseClientOptions"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase](../)/[FluxbaseClientOptions](./)

# FluxbaseClientOptions

data class [FluxbaseClientOptions](./)(val autoRefresh: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = true, val persist: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = true, val storage: [StorageAdapter](../../iogithubnimblefluxfluxbaseauth/-storage-adapter/) = MemoryStorage(), val headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyMap(), val timeout: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html), val debug: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false)

Options for constructing a [FluxbaseClient](../-fluxbase-client/). Port of `FluxbaseClientOptions` from `sdk/src/types.ts:35`.

#### Parameters

jvm

| | |
|---|---|
| autoRefresh | whether to automatically refresh the JWT before expiry (default true). |
| persist | whether to persist the session to the [storage](storage/) adapter (default true). |
| storage | custom session persistence (default: [MemoryStorage](../../iogithubnimblefluxfluxbaseauth/-memory-storage/)). |
| headers | additional default headers on every request. |
| timeout | request timeout in milliseconds (default 30000). |
| debug | enable verbose logging (default false). |

## Constructors

| | |
|---|---|
| [FluxbaseClientOptions](-fluxbase-client-options/) | [jvm]<br>constructor(autoRefresh: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = true, persist: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = true, storage: [StorageAdapter](../../iogithubnimblefluxfluxbaseauth/-storage-adapter/) = MemoryStorage(), headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyMap(), timeout: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html), debug: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false) |

## Properties

| Name | Summary |
|---|---|
| [autoRefresh](auto-refresh/) | [jvm]<br>val [autoRefresh](auto-refresh/): [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = true |
| [debug](debug/) | [jvm]<br>val [debug](debug/): [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false |
| [headers](headers/) | [jvm]<br>val [headers](headers/): [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; |
| [persist](persist/) | [jvm]<br>val [persist](persist/): [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = true |
| [storage](storage/) | [jvm]<br>val [storage](storage/): [StorageAdapter](../../iogithubnimblefluxfluxbaseauth/-storage-adapter/) |
| [timeout](timeout/) | [jvm]<br>val [timeout](timeout/): [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html) |
