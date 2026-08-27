---
title: "WaitForCompletionOptions"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.rpc](../)/[WaitForCompletionOptions](./)

# WaitForCompletionOptions

[jvm]\
data class [WaitForCompletionOptions](./)(val maxWaitMs: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html), val initialIntervalMs: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html) = 500, val maxIntervalMs: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html), val onProgress: ([RpcExecution](../-rpc-execution/)) -&gt; [Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)? = null)

Options for [FluxbaseRpc.waitForCompletion](../-fluxbase-rpc/wait-for-completion/).

## Constructors

| | |
|---|---|
| [WaitForCompletionOptions](-wait-for-completion-options/) | [jvm]<br>constructor(maxWaitMs: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html), initialIntervalMs: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html) = 500, maxIntervalMs: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html), onProgress: ([RpcExecution](../-rpc-execution/)) -&gt; [Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)? = null) |

## Properties

| Name | Summary |
|---|---|
| [initialIntervalMs](initial-interval-ms/) | [jvm]<br>val [initialIntervalMs](initial-interval-ms/): [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html) = 500 |
| [maxIntervalMs](max-interval-ms/) | [jvm]<br>val [maxIntervalMs](max-interval-ms/): [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html) |
| [maxWaitMs](max-wait-ms/) | [jvm]<br>val [maxWaitMs](max-wait-ms/): [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html) |
| [onProgress](on-progress/) | [jvm]<br>val [onProgress](on-progress/): ([RpcExecution](../-rpc-execution/)) -&gt; [Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)? = null |
