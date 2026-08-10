//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.rpc](../index.md)/[FluxbaseRpc](index.md)/[waitForCompletion](wait-for-completion.md)

# waitForCompletion

[jvm]\
suspend fun [waitForCompletion](wait-for-completion.md)(executionId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), options: [WaitForCompletionOptions](../-wait-for-completion-options/index.md) = WaitForCompletionOptions()): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[RpcExecution](../-rpc-execution/index.md)&gt;

Poll for execution completion with exponential backoff. Port of `waitForCompletion()` in `rpc.ts:212`.

Returns when the execution reaches a terminal state (completed/failed/ cancelled/timeout) or when [WaitForCompletionOptions.maxWaitMs](../-wait-for-completion-options/max-wait-ms.md) is exceeded.
