---
title: "Package-level declarations"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../)/[io.github.nimbleflux.fluxbase](./)

# Package-level declarations

## Types

| Name | Summary |
|---|---|
| [FluxbaseClient](-fluxbase-client/) | [jvm]<br>class [FluxbaseClient](-fluxbase-client/)<br>The top-level Fluxbase client — port of `FluxbaseClient` from `sdk/src/client.ts`. |
| [FluxbaseClientOptions](-fluxbase-client-options/) | [jvm]<br>data class [FluxbaseClientOptions](-fluxbase-client-options/)(val autoRefresh: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = true, val persist: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = true, val storage: [StorageAdapter](../iogithubnimblefluxfluxbaseauth/-storage-adapter/) = MemoryStorage(), val headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyMap(), val timeout: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html), val debug: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false)<br>Options for constructing a [FluxbaseClient](-fluxbase-client/). Port of `FluxbaseClientOptions` from `sdk/src/types.ts:35`. |
| [FluxbaseError](-fluxbase-error/) | [jvm]<br>data class [FluxbaseError](-fluxbase-error/)(val status: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null, val code: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val details: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)? = null, val message: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)) : [RuntimeException](https://docs.oracle.com/javase/8/docs/api/java/lang/RuntimeException.html)<br>An error from a Fluxbase API call. Port of `FluxbaseError` from `sdk/src/types.ts:235`. |
| [FluxbaseResponse](-fluxbase-response/) | [jvm]<br>sealed interface [FluxbaseResponse](-fluxbase-response/)&lt;out [T](-fluxbase-response/)&gt;<br>The result type for all Fluxbase SDK operations — the Kotlin equivalent of the TS SDK's `{ data: T; error: null } | { data: null; error: Error }` union (`sdk/src/types.ts:3904`). |

## Functions

| Name | Summary |
|---|---|
| [createFluxbaseClient](create-fluxbase-client/) | [jvm]<br>fun [createFluxbaseClient](create-fluxbase-client/)(url: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, options: [FluxbaseClientOptions](-fluxbase-client-options/) = FluxbaseClientOptions(), transport: [HttpTransport](../iogithubnimblefluxfluxbasecore/-http-transport/)? = null): [FluxbaseClient](-fluxbase-client/)<br>Top-level factory function — Kotlin-idiomatic equivalent of the TS `createClient(url, key, options)`. Delegates to [FluxbaseClient.create](-fluxbase-client/-companion/create/). |
| [fluxbaseResponse](fluxbase-response/) | [jvm]<br>suspend fun &lt;[T](fluxbase-response/)&gt; [fluxbaseResponse](fluxbase-response/)(block: suspend () -&gt; [T](fluxbase-response/)): [FluxbaseResponse](-fluxbase-response/)&lt;[T](fluxbase-response/)&gt;<br>Wraps a suspending block, catching exceptions and converting them to [FluxbaseResponse.Error](-fluxbase-response/-error/). The TS SDK uses `wrapAsync` for the same purpose (`sdk/src/utils/error-handling.ts`). |
| [from](from/) | [jvm]<br>inline fun &lt;[T](from/)&gt; [FluxbaseClient](-fluxbase-client/).[from](from/)(table: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), schema: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null): [QueryBuilder](../iogithubnimblefluxfluxbasepostgrest/-query-builder/)&lt;[T](from/)&gt;<br>Start a PostgREST query against [table](from/). Uses a reified type parameter so the kotlinx.serialization serializer is resolved at compile time. |
| [getOrNull](get-or-null/) | [jvm]<br>fun &lt;[T](get-or-null/)&gt; [FluxbaseResponse](-fluxbase-response/)&lt;[T](get-or-null/)&gt;.[getOrNull](get-or-null/)(): [T](get-or-null/)?<br>Returns the data on success, or null on error. Equivalent to `result.data` in the TS SDK. |
| [getOrThrow](get-or-throw/) | [jvm]<br>fun &lt;[T](get-or-throw/)&gt; [FluxbaseResponse](-fluxbase-response/)&lt;[T](get-or-throw/)&gt;.[getOrThrow](get-or-throw/)(): [T](get-or-throw/)<br>Returns the data on success, or throws the error on failure. |
