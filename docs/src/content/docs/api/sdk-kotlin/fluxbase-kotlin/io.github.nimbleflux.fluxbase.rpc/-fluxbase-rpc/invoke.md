---
title: "invoke"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.rpc](../../)/[FluxbaseRpc](../)/[invoke](./)

# invoke

[jvm]\
suspend fun [invoke](./)(name: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), params: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?&gt;? = null, options: [RpcInvokeOptions](../../-rpc-invoke-options/) = RpcInvokeOptions()): [FluxbaseResponse](../../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[RpcInvokeResponse](../../-rpc-invoke-response/)&gt;

Invoke an RPC procedure. POSTs `/api/v1/rpc/{namespace}/{name}`. Port of `invoke()` in `rpc.ts:111`.

#### Parameters

jvm

| | |
|---|---|
| name | the procedure name. |
| params | parameters to pass to the procedure. |
| options | namespace (default &quot;default&quot;), async, timeout. |
