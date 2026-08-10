//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase](../index.md)/[FluxbaseClientOptions](index.md)

# FluxbaseClientOptions

data class [FluxbaseClientOptions](index.md)(val autoRefresh: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = true, val persist: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = true, val storage: [StorageAdapter](../../io.github.nimbleflux.fluxbase.auth/-storage-adapter/index.md) = MemoryStorage(), val headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyMap(), val timeout: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html), val debug: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false)

Options for constructing a [FluxbaseClient](../-fluxbase-client/index.md). Port of `FluxbaseClientOptions` from `sdk/src/types.ts:35`.

#### Parameters

jvm

| | |
|---|---|
| autoRefresh | whether to automatically refresh the JWT before expiry (default true). |
| persist | whether to persist the session to the [storage](storage.md) adapter (default true). |
| storage | custom session persistence (default: [MemoryStorage](../../io.github.nimbleflux.fluxbase.auth/-memory-storage/index.md)). |
| headers | additional default headers on every request. |
| timeout | request timeout in milliseconds (default 30000). |
| debug | enable verbose logging (default false). |

## Constructors

| | |
|---|---|
| [FluxbaseClientOptions](-fluxbase-client-options.md) | [jvm]<br>constructor(autoRefresh: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = true, persist: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = true, storage: [StorageAdapter](../../io.github.nimbleflux.fluxbase.auth/-storage-adapter/index.md) = MemoryStorage(), headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyMap(), timeout: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html), debug: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false) |

## Properties

| Name | Summary |
|---|---|
| [autoRefresh](auto-refresh.md) | [jvm]<br>val [autoRefresh](auto-refresh.md): [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = true |
| [debug](debug.md) | [jvm]<br>val [debug](debug.md): [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false |
| [headers](headers.md) | [jvm]<br>val [headers](headers.md): [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; |
| [persist](persist.md) | [jvm]<br>val [persist](persist.md): [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = true |
| [storage](storage.md) | [jvm]<br>val [storage](storage.md): [StorageAdapter](../../io.github.nimbleflux.fluxbase.auth/-storage-adapter/index.md) |
| [timeout](timeout.md) | [jvm]<br>val [timeout](timeout.md): [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html) |
