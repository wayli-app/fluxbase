---
title: "getLogs"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.rpc](../index.md)/[FluxbaseRpc](index.md)/[getLogs](get-logs.md)

# getLogs

[jvm]\
suspend fun [getLogs](get-logs.md)(executionId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), afterLine: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[RpcExecutionLog](../-rpc-execution-log/index.md)&gt;&gt;

Get execution logs. GETs `/api/v1/rpc/executions/{id}/logs`. Port of `getLogs()` in `rpc.ts:177`.
