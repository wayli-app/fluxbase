//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.rpc](../index.md)/[FluxbaseRpc](index.md)

# FluxbaseRpc

[jvm]\
class [FluxbaseRpc](index.md)(http: [FluxbaseHttpClient](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md))

RPC (Remote Procedure Call) module — port of `FluxbaseRPC` from `sdk/src/rpc.ts`.

Invokes Fluxbase's namespaced SQL procedures via `POST /api/v1/rpc/{namespace}/{name}`. Procedures are namespaced (default namespace is `"default"`; Wayli uses `"wayli"`).

Usage:

```kotlin
// Synchronous invoke
val (data, error) = client.rpc.invoke("get-trip-summary", mapOf("trip_id" to "abc"), RpcInvokeOptions(namespace = "wayli"))

// Async invoke + poll
val (started, _) = client.rpc.invoke("long-report", async = true)
val (final, _) = client.rpc.waitForCompletion(started!!.executionId!!)
```

## Constructors

| | |
|---|---|
| [FluxbaseRpc](-fluxbase-rpc.md) | [jvm]<br>constructor(http: [FluxbaseHttpClient](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md)) |

## Functions

| Name | Summary |
|---|---|
| [getLogs](get-logs.md) | [jvm]<br>suspend fun [getLogs](get-logs.md)(executionId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), afterLine: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[RpcExecutionLog](../-rpc-execution-log/index.md)&gt;&gt;<br>Get execution logs. GETs `/api/v1/rpc/executions/{id}/logs`. Port of `getLogs()` in `rpc.ts:177`. |
| [getStatus](get-status.md) | [jvm]<br>suspend fun [getStatus](get-status.md)(executionId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[RpcExecution](../-rpc-execution/index.md)&gt;<br>Get the status of an async RPC execution. GETs `/api/v1/rpc/executions/{id}`. Port of `getStatus()` in `rpc.ts:148`. |
| [invoke](invoke.md) | [jvm]<br>suspend fun [invoke](invoke.md)(name: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), params: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?&gt;? = null, options: [RpcInvokeOptions](../-rpc-invoke-options/index.md) = RpcInvokeOptions()): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[RpcInvokeResponse](../-rpc-invoke-response/index.md)&gt;<br>Invoke an RPC procedure. POSTs `/api/v1/rpc/{namespace}/{name}`. Port of `invoke()` in `rpc.ts:111`. |
| [list](list.md) | [jvm]<br>suspend fun [list](list.md)(namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[RpcProcedureSummary](../-rpc-procedure-summary/index.md)&gt;&gt;<br>List available RPC procedures. GETs `/api/v1/rpc/procedures`. Port of `list()` in `rpc.ts:69`. |
| [waitForCompletion](wait-for-completion.md) | [jvm]<br>suspend fun [waitForCompletion](wait-for-completion.md)(executionId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), options: [WaitForCompletionOptions](../-wait-for-completion-options/index.md) = WaitForCompletionOptions()): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[RpcExecution](../-rpc-execution/index.md)&gt;<br>Poll for execution completion with exponential backoff. Port of `waitForCompletion()` in `rpc.ts:212`. |
