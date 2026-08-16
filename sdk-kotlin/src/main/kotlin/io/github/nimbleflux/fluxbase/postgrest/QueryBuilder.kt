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
    private var countMode: String? = null,
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

    // ---- PostGIS operators (port of query-builder.ts:382-464) ----

    /** PostGIS ST_Intersects — filter geometries that intersect [geojson]. */
    fun intersects(column: String, geojson: Any?) = addFilter(column, "st_intersects", geojson)

    /** PostGIS ST_Contains — filter geometries that contain [geojson]. */
    fun stContains(column: String, geojson: Any?) = addFilter(column, "st_contains", geojson)

    /** PostGIS ST_Within — filter geometries within [geojson]. */
    fun within(column: String, geojson: Any?) = addFilter(column, "st_within", geojson)

    /** PostGIS ST_Touches. */
    fun touches(column: String, geojson: Any?) = addFilter(column, "st_touches", geojson)

    /** PostGIS ST_Crosses. */
    fun crosses(column: String, geojson: Any?) = addFilter(column, "st_crosses", geojson)

    /** PostGIS ST_Overlaps. */
    fun stOverlaps(column: String, geojson: Any?) = addFilter(column, "st_overlaps", geojson)

    /** PostGIS ST_DWithin — within [distanceMeters] of [geojson]. */
    fun stDWithin(column: String, geojson: Any?, distanceMeters: Double) =
        clone().also {
            // st_dwithin takes a geometry + distance: column=st_dwithin.{geojson},{distance}
            it.filters.add(Filter(column, "st_dwithin", listOf(geojson, distanceMeters)))
        }

    /** PostGIS ST_Distance — returns distance (used for ordering). */
    fun stDistance(column: String, geojson: Any?) = addFilter(column, "st_distance", geojson)

    // ---- pgvector (port of query-builder.ts:500-570) ----

    /**
     * Order by vector similarity. Adds a vector order clause.
     * Port of `orderByVector()` in `query-builder.ts:500`.
     */
    fun orderByVector(column: String, vector: List<Double>, metric: VectorMetric): QueryBuilder<T> = clone().also {
        val vecStr = "[" + vector.joinToString(",") { it.toString() } + "]"
        val clause = "$column.${metric.operator}.$vecStr"
        it.orderClause = if (orderClause != null) "$orderClause,$clause" else clause
    }

    /**
     * Vector similarity search — combines an order-by-distance + a distance filter.
     * Port of `vectorSearch()` in `query-builder.ts:540`.
     */
    fun vectorSearch(column: String, vector: List<Double>, metric: VectorMetric = VectorMetric.COSINE): QueryBuilder<T> {
        return orderByVector(column, vector, metric)
    }

    // ---- Between (port of query-builder.ts between/notBetween) ----

    /** Filter values between [min] and [max] (inclusive). Adds gte + lte filters. */
    fun between(column: String, min: Any?, max: Any?): QueryBuilder<T> = clone().also {
        it.filters.add(Filter(column, "gte", min))
        it.filters.add(Filter(column, "lte", max))
    }

    /** Filter values NOT between [min] and [max]. Adds lt + gt filters. */
    fun notBetween(column: String, min: Any?, max: Any?): QueryBuilder<T> = clone().also {
        it.filters.add(Filter(column, "lt", min))
        it.filters.add(Filter(column, "gt", max))
    }

    /** Request a count of total matching rows. */
    fun count(countType: CountType = CountType.EXACT): QueryBuilder<T> = clone().also {
        it.countMode = countType.value
    }

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

    // ---- Execution ----

    // ---- Mutations ----

    /** INSERT. */
    suspend fun insert(values: Map<String, Any?>): PostgrestResponse<T> =
        executeMutation(HttpMethod.POST, values, extraHeaders = emptyMap())

    /**
     * UPSERT (INSERT with conflict resolution).
     * Port of `upsert()` in `query-builder.ts:102`. Adds the `Prefer:
     * resolution=merge-duplicates` header to a POST.
     */
    suspend fun upsert(values: Map<String, Any?>): PostgrestResponse<T> =
        executeMutation(
            HttpMethod.POST,
            values,
            extraHeaders = mapOf("Prefer" to "resolution=merge-duplicates"),
        )

    /** UPDATE (requires filters). */
    suspend fun update(values: Map<String, Any?>): PostgrestResponse<T> =
        executeMutation(HttpMethod.PATCH, values)

    /** DELETE (requires filters). */
    suspend fun delete(): PostgrestResponse<T> =
        executeMutation(HttpMethod.DELETE, null)

    // ---- Execution ----

    /** Execute a SELECT query and return the list of rows (with count if requested). */
    suspend fun execute(): PostgrestResponse<List<T>> = executeList()

    private suspend fun executeList(): PostgrestResponse<List<T>> {
        val path = buildTablePath() + buildQueryString()
        return try {
            val rawResponse = http.getWithHeaders(path)
            val json = FluxbaseHttpClient.defaultJson
            val data = if (rawResponse.body.isBlank()) {
                emptyList()
            } else {
                json.decodeFromString(ListSerializer(serializer), rawResponse.body)
            }
            val parsedCount = parseCountFromRange(rawResponse.headers["Content-Range"])
            PostgrestResponse(
                data = data,
                error = null,
                count = parsedCount,
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

    private fun parseCountFromRange(contentRange: String?): Long? {
        if (contentRange == null) return null
        // Content-Range format: "0-99/5000" → total is 5000
        val regex = Regex("""/(\d+)$""")
        return regex.find(contentRange)?.groupValues?.get(1)?.toLongOrNull()
    }

    private suspend fun executeMutation(
        method: HttpMethod,
        body: Any?,
        extraHeaders: Map<String, String> = emptyMap(),
    ): PostgrestResponse<T> {
        val path = buildTablePath() + if (method != HttpMethod.POST) buildQueryString() else ""
        return try {
            val rawResponse: HttpResponse = http.doTransportRequest(method, path, body, extraHeaders)
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
        countMode?.let { params.add("count=$it") }
        return if (params.isEmpty()) "" else "?" + params.joinToString("&")
    }

    private fun formatValue(value: Any?): String = when (value) {
        null -> "null"
        is Boolean -> if (value) "true" else "false"
        is List<*> -> "(" + value.joinToString(",") { formatValue(it) } + ")"
        is Map<*, *> -> encode(jsonStringify(value))
        else -> encode(value.toString())
    }

    /** Minimal JSON stringifier for PostgREST filter values (GeoJSON objects). */
    private fun jsonStringify(value: Any?): String = when (value) {
        null -> "null"
        is String -> "\"$value\""
        is Number -> value.toString()
        is Boolean -> value.toString()
        is List<*> -> "[" + value.joinToString(",") { jsonStringify(it) } + "]"
        is Map<*, *> -> "{" + value.entries.joinToString(",") { "\"${it.key}\":${jsonStringify(it.value)}" } + "}"
        else -> "\"$value\""
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
        countMode = countMode,
    )
}

/**
 * Internal: expose the transport for mutation requests (PATCH/DELETE/POST-with-headers).
 * Merges extra headers (e.g. Prefer for upsert) on top of defaults.
 */
@PublishedApi
internal suspend fun FluxbaseHttpClient.doTransportRequest(
    method: HttpMethod,
    path: String,
    body: Any?,
    extraHeaders: Map<String, String> = emptyMap(),
): HttpResponse = transport.request(method, path, body, defaultHeaders.toMap() + extraHeaders)
