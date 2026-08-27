---
title: "RpcInvokeResponse"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.rpc](../)/[RpcInvokeResponse](./)

# RpcInvokeResponse

[jvm]\
@Serializable

data class [RpcInvokeResponse](./)(val executionId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val status: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;completed&quot;, val result: JsonElement? = null, val rowsReturned: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? = null, val durationMs: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? = null, val error: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null)

The response from invoking an RPC procedure. Port of `RPCInvokeResponse` from `sdk/src/types.ts:4116`.

## Constructors

| | |
|---|---|
| [RpcInvokeResponse](-rpc-invoke-response/) | [jvm]<br>constructor(executionId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, status: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;completed&quot;, result: JsonElement? = null, rowsReturned: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? = null, durationMs: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? = null, error: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null) |

## Properties

| Name | Summary |
|---|---|
| [durationMs](duration-ms/) | [jvm]<br>@SerialName(value = &quot;duration_ms&quot;)<br>val [durationMs](duration-ms/): [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? = null |
| [error](error/) | [jvm]<br>val [error](error/): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null |
| [executionId](execution-id/) | [jvm]<br>@SerialName(value = &quot;execution_id&quot;)<br>val [executionId](execution-id/): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null |
| [result](result/) | [jvm]<br>val [result](result/): JsonElement? = null |
| [rowsReturned](rows-returned/) | [jvm]<br>@SerialName(value = &quot;rows_returned&quot;)<br>val [rowsReturned](rows-returned/): [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? = null |
| [status](status/) | [jvm]<br>val [status](status/): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
