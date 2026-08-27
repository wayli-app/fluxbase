---
title: "list"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.rpc](../../)/[FluxbaseRpc](../)/[list](./)

# list

[jvm]\
suspend fun [list](./)(namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null): [FluxbaseResponse](../../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[RpcProcedureSummary](../../-rpc-procedure-summary/)&gt;&gt;

List available RPC procedures. GETs `/api/v1/rpc/procedures`. Port of `list()` in `rpc.ts:69`.
