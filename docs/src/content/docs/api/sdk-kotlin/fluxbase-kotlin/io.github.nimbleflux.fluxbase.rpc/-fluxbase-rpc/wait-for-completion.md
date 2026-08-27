---
title: "waitForCompletion"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.rpc](../../)/[FluxbaseRpc](../)/[waitForCompletion](./)

# waitForCompletion

[jvm]\
suspend fun [waitForCompletion](./)(executionId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), options: [WaitForCompletionOptions](../../-wait-for-completion-options/) = WaitForCompletionOptions()): [FluxbaseResponse](../../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[RpcExecution](../../-rpc-execution/)&gt;

Poll for execution completion with exponential backoff. Port of `waitForCompletion()` in `rpc.ts:212`.

Returns when the execution reaches a terminal state (completed/failed/ cancelled/timeout) or when [WaitForCompletionOptions.maxWaitMs](../../-wait-for-completion-options/max-wait-ms/) is exceeded.
