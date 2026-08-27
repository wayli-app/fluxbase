---
title: "getLogs"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.rpc](../../)/[FluxbaseRpc](../)/[getLogs](./)

# getLogs

[jvm]\
suspend fun [getLogs](./)(executionId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), afterLine: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null): [FluxbaseResponse](../../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[RpcExecutionLog](../../-rpc-execution-log/)&gt;&gt;

Get execution logs. GETs `/api/v1/rpc/executions/{id}/logs`. Port of `getLogs()` in `rpc.ts:177`.
