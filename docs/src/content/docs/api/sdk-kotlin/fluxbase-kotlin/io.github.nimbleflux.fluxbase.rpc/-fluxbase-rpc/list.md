---
title: "list"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.rpc](../index.md)/[FluxbaseRpc](index.md)/[list](list.md)

# list

[jvm]\
suspend fun [list](list.md)(namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[RpcProcedureSummary](../-rpc-procedure-summary/index.md)&gt;&gt;

List available RPC procedures. GETs `/api/v1/rpc/procedures`. Port of `list()` in `rpc.ts:69`.
