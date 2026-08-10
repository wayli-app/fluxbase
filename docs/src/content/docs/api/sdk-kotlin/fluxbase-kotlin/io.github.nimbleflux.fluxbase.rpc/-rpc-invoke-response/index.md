//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.rpc](../index.md)/[RpcInvokeResponse](index.md)

# RpcInvokeResponse

[jvm]\
@Serializable

data class [RpcInvokeResponse](index.md)(val executionId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val status: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;completed&quot;, val result: JsonElement? = null, val rowsReturned: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? = null, val durationMs: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? = null, val error: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null)

The response from invoking an RPC procedure. Port of `RPCInvokeResponse` from `sdk/src/types.ts:4116`.

## Constructors

| | |
|---|---|
| [RpcInvokeResponse](-rpc-invoke-response.md) | [jvm]<br>constructor(executionId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, status: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;completed&quot;, result: JsonElement? = null, rowsReturned: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? = null, durationMs: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? = null, error: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null) |

## Properties

| Name | Summary |
|---|---|
| [durationMs](duration-ms.md) | [jvm]<br>@SerialName(value = &quot;duration_ms&quot;)<br>val [durationMs](duration-ms.md): [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? = null |
| [error](error.md) | [jvm]<br>val [error](error.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null |
| [executionId](execution-id.md) | [jvm]<br>@SerialName(value = &quot;execution_id&quot;)<br>val [executionId](execution-id.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null |
| [result](result.md) | [jvm]<br>val [result](result.md): JsonElement? = null |
| [rowsReturned](rows-returned.md) | [jvm]<br>@SerialName(value = &quot;rows_returned&quot;)<br>val [rowsReturned](rows-returned.md): [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? = null |
| [status](status.md) | [jvm]<br>val [status](status.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
