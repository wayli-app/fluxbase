---
title: "RpcInvokeOptions"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.rpc](../)/[RpcInvokeOptions](./)

# RpcInvokeOptions

[jvm]\
data class [RpcInvokeOptions](./)(val namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;default&quot;, val async: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false, val timeout: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? = null)

Options for [FluxbaseRpc.invoke](../-fluxbase-rpc/invoke/). Port of `RPCInvokeOptions` from `rpc.ts:15`.

## Constructors

| | |
|---|---|
| [RpcInvokeOptions](-rpc-invoke-options/) | [jvm]<br>constructor(namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;default&quot;, async: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false, timeout: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? = null) |

## Properties

| Name | Summary |
|---|---|
| [async](async/) | [jvm]<br>val [async](async/): [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false |
| [namespace](namespace/) | [jvm]<br>val [namespace](namespace/): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [timeout](timeout/) | [jvm]<br>val [timeout](timeout/): [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? = null |
