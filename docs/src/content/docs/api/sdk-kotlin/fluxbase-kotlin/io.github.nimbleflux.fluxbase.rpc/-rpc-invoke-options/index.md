//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.rpc](../index.md)/[RpcInvokeOptions](index.md)

# RpcInvokeOptions

[jvm]\
data class [RpcInvokeOptions](index.md)(val namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;default&quot;, val async: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false, val timeout: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? = null)

Options for [FluxbaseRpc.invoke](../-fluxbase-rpc/invoke.md). Port of `RPCInvokeOptions` from `rpc.ts:15`.

## Constructors

| | |
|---|---|
| [RpcInvokeOptions](-rpc-invoke-options.md) | [jvm]<br>constructor(namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;default&quot;, async: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false, timeout: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? = null) |

## Properties

| Name | Summary |
|---|---|
| [async](async.md) | [jvm]<br>val [async](async.md): [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false |
| [namespace](namespace.md) | [jvm]<br>val [namespace](namespace.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [timeout](timeout.md) | [jvm]<br>val [timeout](timeout.md): [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? = null |
