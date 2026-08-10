package io.github.nimbleflux.fluxbase.postgrest

import io.github.nimbleflux.fluxbase.FluxbaseError
import io.github.nimbleflux.fluxbase.FluxbaseResponse
import io.github.nimbleflux.fluxbase.core.FluxbaseHttpClient
import io.github.nimbleflux.fluxbase.core.HttpMethod
import io.github.nimbleflux.fluxbase.core.HttpResponse
import io.github.nimbleflux.fluxbase.core.HttpTransport
import io.github.nimbleflux.fluxbase.fluxbaseResponse
import kotlinx.serialization.KSerializer
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.json.Json

/**
 * Result of a PostgREST query. Port of `PostgrestResponse<T>` from
 * `sdk/src/types.ts:257`.
 */
data class PostgrestResponse<T>(
    val data: T?,
    val error: FluxbaseError?,
    val count: Long?,
    val status: Int,
    val statusText: String,
)

/**
 * A chainable PostgREST query builder. Port of `QueryBuilder<T>` from
 * `sdk/src/query-builder.ts`.
 *
 * Builds PostgREST-style query strings against Fluxbase's
 * `/api/v1/tables/{schema}/{table}` endpoint. The builder is immutable — each
 * filter method returns a new [QueryBuilder] (matching the TS `clone()` pattern)
 * so a base query can be reused with different filters.
 *
 * Generic type [T] is resolved at compile time via the reified [from] extension.
 *
 * Usage:
 * ```
 * val result = client.from<Trip>().select().eq("user_id", uid).execute()
 * ```
 */
class QueryBuilder<T> @PublishedApi internal constructor(
    private val http: FluxbaseHttpClient,
    internal val serializer: KSerializer<T>,
    private val table: String,
    private val schema: String? = null,
    private var selectColumns: String = "*",
    private var filters: MutableList<Filter> = mutableListOf(),
    private var orderClause: String? = null,
    private var limitValue: Int? = null,
    private var offsetValue: Int? = null,
) {
    data class Filter(val column: String, val operator: String, val value: Any?)

    fun select(columns: String = "*"): QueryBuilder<T> = clone().also { it.selectColumns = columns }

    // ---- Filters ----
    fun eq(column: String, value: Any?) = addFilter(column, "eq", value)
    fun neq(column: String, value: Any?) = addFilter(column, "neq", value)
    fun gt(column: String, value: Any?) = addFilter(column, "gt", value)
    fun gte(column: String, value: Any?) = addFilter(column, "gte", value)
    fun lt(column: String, value: Any?) = addFilter(column, "lt", value)
    fun lte(column: String, value: Any?) = addFilter(column, "lte", value)
    fun like(column: String, value: String) = addFilter(column, "like", value)
    fun ilike(column: String, value: String) = addFilter(column, "ilike", value)
    fun `in`(column: String, values: List<Any?>) = addFilter(column, "in", values)
    fun is_(column: String, value: Any?) = addFilter(column, "is", value)

    private fun addFilter(column: String, operator: String, value: Any?): QueryBuilder<T> = clone().also {
        it.filters.add(Filter(column, operator, value))
    }

    fun order(column: String, ascending: Boolean = true): QueryBuilder<T> = clone().also {
        val direction = if (ascending) "asc" else "desc"
        it.orderClause = if (orderClause != null) "$orderClause,$column.$direction" else "$column.$direction"
    }

    fun limit(limit: Int): QueryBuilder<T> = clone().also { it.limitValue = limit }
    fun offset(offset: Int): QueryBuilder<T> = clone().also { it.offsetValue = offset }

    /** Range-based pagination — returns a clone so the base builder can be reused. */
    fun range(from: Int, to: Int): QueryBuilder<T> = clone().also {
        it.offsetValue = from
        it.limitValue = to - from + 1
    }

    /** Limit to 1 row; error if 0 rows (PGRST116). Port of `single()`. */
    suspend fun single(): FluxbaseResponse<T> {
        val withLimit = clone().also { it.limitValue = 1 }
        val response = withLimit.executeList()
        val err = response.error
        if (err != null) return FluxbaseResponse.Error(err)
        val list = response.data
        if (list.isNullOrEmpty()) return FluxbaseResponse.Error(
            FluxbaseError(message = "No rows found", code = "PGRST116"),
        )
        return FluxbaseResponse.Success(list.first())
    }

    /** Limit to 1 row; null if 0 rows (no error). Port of `maybeSingle()`. */
    suspend fun maybeSingle(): FluxbaseResponse<T?> {
        val withLimit = clone().also { it.limitValue = 1 }
        val response = withLimit.executeList()
        val err = response.error
        if (err != null) return FluxbaseResponse.Error(err)
        return FluxbaseResponse.Success(response.data?.firstOrNull())
    }

    // ---- Mutations ----

    /** INSERT. */
    suspend fun insert(values: Map<String, Any?>): PostgrestResponse<T> =
        executeMutation(HttpMethod.POST, values)

    /** UPDATE (requires filters). */
    suspend fun update(values: Map<String, Any?>): PostgrestResponse<T> =
        executeMutation(HttpMethod.PATCH, values)

    /** DELETE (requires filters). */
    suspend fun delete(): PostgrestResponse<T> =
        executeMutation(HttpMethod.DELETE, null)

    // ---- Execution ----

    /** Execute a SELECT query and return the list of rows. */
    suspend fun execute(): FluxbaseResponse<List<T>> = executeList()

    private suspend fun executeList(): FluxbaseResponse<List<T>> = fluxbaseResponse {
        val path = buildTablePath() + buildQueryString()
        val rawResponse = http.getWithHeaders(path)
        val json = FluxbaseHttpClient.defaultJson
        json.decodeFromString(ListSerializer(serializer), rawResponse.body)
    }

    private suspend fun executeMutation(method: HttpMethod, body: Any?): PostgrestResponse<T> {
        val path = buildTablePath() + if (method != HttpMethod.POST) buildQueryString() else ""
        return try {
            val rawResponse: HttpResponse = http.doTransportRequest(method, path, body)
            PostgrestResponse(
                data = null,
                error = null,
                count = null,
                status = rawResponse.status,
                statusText = "",
            )
        } catch (e: io.github.nimbleflux.fluxbase.core.FluxbaseException) {
            PostgrestResponse(
                data = null,
                error = FluxbaseError(message = e.message ?: "Request failed", status = e.status, code = e.code),
                count = null,
                status = e.status,
                statusText = "",
            )
        }
    }

    // ---- Query string building (port of query-builder.ts:1210-1314) ----

    internal fun buildTablePath(): String =
        if (schema != null) "/api/v1/tables/$schema/$table" else "/api/v1/tables/$table"

    internal fun buildQueryString(): String {
        val params = mutableListOf<String>()
        if (selectColumns != "*") params.add("select=${encode(selectColumns)}")
        filters.forEach { f ->
            params.add("${f.column}=${f.operator}.${formatValue(f.value)}")
        }
        orderClause?.let { params.add("order=$it") }
        limitValue?.let { params.add("limit=$it") }
        offsetValue?.let { params.add("offset=$it") }
        return if (params.isEmpty()) "" else "?" + params.joinToString("&")
    }

    private fun formatValue(value: Any?): String = when (value) {
        null -> "null"
        is Boolean -> if (value) "true" else "false"
        is List<*> -> "(" + value.joinToString(",") { it.toString() } + ")"
        else -> encode(value.toString())
    }

    private fun encode(s: String): String = java.net.URLEncoder.encode(s, "UTF-8")

    private fun clone(): QueryBuilder<T> = QueryBuilder(
        http = http,
        serializer = serializer,
        table = table,
        schema = schema,
        selectColumns = selectColumns,
        filters = filters.toMutableList(),
        orderClause = orderClause,
        limitValue = limitValue,
        offsetValue = offsetValue,
    )
}

/**
 * Internal: expose the transport for mutation requests (PATCH/DELETE need raw
 * access to set the query string on non-GET methods).
 */
@PublishedApi
internal suspend fun FluxbaseHttpClient.doTransportRequest(
    method: HttpMethod,
    path: String,
    body: Any?,
): HttpResponse = transport.request(method, path, body, defaultHeaders.toMap())
