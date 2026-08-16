package io.github.nimbleflux.fluxbase.rpc

import io.github.nimbleflux.fluxbase.FluxbaseError
import io.github.nimbleflux.fluxbase.FluxbaseResponse
import io.github.nimbleflux.fluxbase.core.FluxbaseHttpClient
import io.github.nimbleflux.fluxbase.fluxbaseResponse
import kotlinx.coroutines.delay
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put

/**
 * A summary of an available RPC procedure.
 * Port of `RPCProcedureSummary` from `sdk/src/types.ts:4051`.
 */
@Serializable
data class RpcProcedureSummary(
    val name: String,
    val namespace: String = "default",
    val description: String? = null,
    @SerialName("return_type") val returnType: String? = null,
)

/**
 * The response from invoking an RPC procedure.
 * Port of `RPCInvokeResponse` from `sdk/src/types.ts:4116`.
 */
@Serializable
data class RpcInvokeResponse(
    @SerialName("execution_id") val executionId: String? = null,
    val status: String = "completed",
    val result: JsonElement? = null,
    @SerialName("rows_returned") val rowsReturned: Long? = null,
    @SerialName("duration_ms") val durationMs: Long? = null,
    val error: String? = null,
)

/**
 * The status of an async RPC execution.
 * Port of `RPCExecution` from `sdk/src/types.ts:4093`.
 */
@Serializable
data class RpcExecution(
    val id: String,
    val status: String,
    val result: JsonElement? = null,
    val error: String? = null,
    @SerialName("started_at") val startedAt: String? = null,
    @SerialName("completed_at") val completedAt: String? = null,
    @SerialName("duration_ms") val durationMs: Long? = null,
)

/** A log entry from an RPC execution. */
@Serializable
data class RpcExecutionLog(
    val message: String,
    val level: String? = null,
    val timestamp: String? = null,
)

/** Options for [FluxbaseRpc.invoke]. Port of `RPCInvokeOptions` from `rpc.ts:15`. */
data class RpcInvokeOptions(
    val namespace: String = "default",
    val async: Boolean = false,
    val timeout: Long? = null,
)

/** Options for [FluxbaseRpc.waitForCompletion]. */
data class WaitForCompletionOptions(
    val maxWaitMs: Long = 30_000,
    val initialIntervalMs: Long = 500,
    val maxIntervalMs: Long = 5_000,
    val onProgress: ((RpcExecution) -> Unit)? = null,
)

/**
 * RPC (Remote Procedure Call) module — port of `FluxbaseRPC` from `sdk/src/rpc.ts`.
 *
 * Invokes Fluxbase's namespaced SQL procedures via
 * `POST /api/v1/rpc/{namespace}/{name}`. Procedures are namespaced (default
 * namespace is `"default"`; Wayli uses `"wayli"`).
 *
 * Usage:
 * ```
 * // Synchronous invoke
 * val (data, error) = client.rpc.invoke("get-trip-summary", mapOf("trip_id" to "abc"), RpcInvokeOptions(namespace = "wayli"))
 *
 * // Async invoke + poll
 * val (started, _) = client.rpc.invoke("long-report", async = true)
 * val (final, _) = client.rpc.waitForCompletion(started!!.executionId!!)
 * ```
 */
class FluxbaseRpc(@PublishedApi internal val http: FluxbaseHttpClient) {

    /**
     * List available RPC procedures. GETs `/api/v1/rpc/procedures`.
     * Port of `list()` in `rpc.ts:69`.
     */
    suspend fun list(namespace: String? = null): FluxbaseResponse<List<RpcProcedureSummary>> = fluxbaseResponse {
        val query = namespace?.let { "?namespace=${encode(it)}" } ?: ""
        val response: RpcListResponse = http.get("/api/v1/rpc/procedures$query")
        response.procedures
    }

    @Serializable
    private data class RpcListResponse(
        val procedures: List<RpcProcedureSummary> = emptyList(),
        val count: Int = 0,
    )

    /**
     * Invoke an RPC procedure. POSTs `/api/v1/rpc/{namespace}/{name}`.
     * Port of `invoke()` in `rpc.ts:111`.
     *
     * @param name the procedure name.
     * @param params parameters to pass to the procedure.
     * @param options namespace (default "default"), async, timeout.
     */
    suspend fun invoke(
        name: String,
        params: Map<String, Any?>? = null,
        options: RpcInvokeOptions = RpcInvokeOptions(),
    ): FluxbaseResponse<RpcInvokeResponse> = fluxbaseResponse {
        val body = buildJsonObject {
            if (params != null) {
                put("params", buildJsonObject {
                    params.forEach { (k, v) ->
                        when (v) {
                            null -> put(k, kotlinx.serialization.json.JsonNull)
                            is String -> put(k, v)
                            is Number -> put(k, v)
                            is Boolean -> put(k, v)
                            else -> put(k, v.toString())
                        }
                    }
                })
            }
            put("async", options.async)
        }
        http.post("/api/v1/rpc/${encode(options.namespace)}/${encode(name)}", body)
    }

    /**
     * Get the status of an async RPC execution. GETs `/api/v1/rpc/executions/{id}`.
     * Port of `getStatus()` in `rpc.ts:148`.
     */
    suspend fun getStatus(executionId: String): FluxbaseResponse<RpcExecution> = fluxbaseResponse {
        http.get("/api/v1/rpc/executions/${encode(executionId)}")
    }

    /**
     * Get execution logs. GETs `/api/v1/rpc/executions/{id}/logs`.
     * Port of `getLogs()` in `rpc.ts:177`.
     */
    suspend fun getLogs(executionId: String, afterLine: Int? = null): FluxbaseResponse<List<RpcExecutionLog>> = fluxbaseResponse {
        val query = afterLine?.let { "?after=$it" } ?: ""
        val response: RpcLogsResponse = http.get("/api/v1/rpc/executions/${encode(executionId)}/logs$query")
        response.logs
    }

    @Serializable
    private data class RpcLogsResponse(
        val logs: List<RpcExecutionLog> = emptyList(),
        val count: Int = 0,
    )

    /**
     * Poll for execution completion with exponential backoff.
     * Port of `waitForCompletion()` in `rpc.ts:212`.
     *
     * Returns when the execution reaches a terminal state (completed/failed/
     * cancelled/timeout) or when [WaitForCompletionOptions.maxWaitMs] is exceeded.
     */
    suspend fun waitForCompletion(
        executionId: String,
        options: WaitForCompletionOptions = WaitForCompletionOptions(),
    ): FluxbaseResponse<RpcExecution> {
        val startTime = System.currentTimeMillis()
        var interval = options.initialIntervalMs

        while (System.currentTimeMillis() - startTime < options.maxWaitMs) {
            val result = getStatus(executionId)
            val err = result.error
            if (err != null) return FluxbaseResponse.Error(err)
            val execution = result.data ?: return FluxbaseResponse.Error(
                FluxbaseError(message = "Execution not found"),
            )

            options.onProgress?.invoke(execution)

            if (execution.status in TERMINAL_STATES) {
                return FluxbaseResponse.Success(execution)
            }

            delay(interval)
            interval = (interval * 1.5).toLong().coerceAtMost(options.maxIntervalMs)
        }

        return FluxbaseResponse.Error(FluxbaseError(message = "Timeout waiting for execution to complete"))
    }

    private companion object {
        val TERMINAL_STATES = setOf("completed", "failed", "cancelled", "timeout")
    }

    @PublishedApi
    internal fun encode(s: String): String = java.net.URLEncoder.encode(s, "UTF-8")
}
