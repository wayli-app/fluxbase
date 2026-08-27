---
title: "getStatus"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.rpc](../../)/[FluxbaseRpc](../)/[getStatus](./)

# getStatus

[jvm]\
suspend fun [getStatus](./)(executionId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[RpcExecution](../../-rpc-execution/)&gt;

Get the status of an async RPC execution. GETs `/api/v1/rpc/executions/{id}`. Port of `getStatus()` in `rpc.ts:148`.
