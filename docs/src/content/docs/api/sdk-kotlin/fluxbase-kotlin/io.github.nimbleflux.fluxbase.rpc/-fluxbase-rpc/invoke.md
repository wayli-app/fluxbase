---
title: "invoke"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.rpc](../index.md)/[FluxbaseRpc](index.md)/[invoke](invoke.md)

# invoke

[jvm]\
suspend fun [invoke](invoke.md)(name: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), params: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?&gt;? = null, options: [RpcInvokeOptions](../-rpc-invoke-options/index.md) = RpcInvokeOptions()): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[RpcInvokeResponse](../-rpc-invoke-response/index.md)&gt;

Invoke an RPC procedure. POSTs `/api/v1/rpc/{namespace}/{name}`. Port of `invoke()` in `rpc.ts:111`.

#### Parameters

jvm

| | |
|---|---|
| name | the procedure name. |
| params | parameters to pass to the procedure. |
| options | namespace (default &quot;default&quot;), async, timeout. |
