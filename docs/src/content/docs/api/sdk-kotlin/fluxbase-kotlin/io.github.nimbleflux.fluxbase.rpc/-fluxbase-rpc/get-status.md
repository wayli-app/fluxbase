---
title: "getStatus"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.rpc](../index.md)/[FluxbaseRpc](index.md)/[getStatus](get-status.md)

# getStatus

[jvm]\
suspend fun [getStatus](get-status.md)(executionId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[RpcExecution](../-rpc-execution/index.md)&gt;

Get the status of an async RPC execution. GETs `/api/v1/rpc/executions/{id}`. Port of `getStatus()` in `rpc.ts:148`.
