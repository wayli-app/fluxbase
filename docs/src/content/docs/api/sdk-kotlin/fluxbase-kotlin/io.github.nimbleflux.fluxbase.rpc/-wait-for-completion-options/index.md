//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.rpc](../index.md)/[WaitForCompletionOptions](index.md)

# WaitForCompletionOptions

[jvm]\
data class [WaitForCompletionOptions](index.md)(val maxWaitMs: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html), val initialIntervalMs: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html) = 500, val maxIntervalMs: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html), val onProgress: ([RpcExecution](../-rpc-execution/index.md)) -&gt; [Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)? = null)

Options for [FluxbaseRpc.waitForCompletion](../-fluxbase-rpc/wait-for-completion.md).

## Constructors

| | |
|---|---|
| [WaitForCompletionOptions](-wait-for-completion-options.md) | [jvm]<br>constructor(maxWaitMs: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html), initialIntervalMs: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html) = 500, maxIntervalMs: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html), onProgress: ([RpcExecution](../-rpc-execution/index.md)) -&gt; [Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)? = null) |

## Properties

| Name | Summary |
|---|---|
| [initialIntervalMs](initial-interval-ms.md) | [jvm]<br>val [initialIntervalMs](initial-interval-ms.md): [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html) = 500 |
| [maxIntervalMs](max-interval-ms.md) | [jvm]<br>val [maxIntervalMs](max-interval-ms.md): [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html) |
| [maxWaitMs](max-wait-ms.md) | [jvm]<br>val [maxWaitMs](max-wait-ms.md): [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html) |
| [onProgress](on-progress.md) | [jvm]<br>val [onProgress](on-progress.md): ([RpcExecution](../-rpc-execution/index.md)) -&gt; [Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)? = null |
