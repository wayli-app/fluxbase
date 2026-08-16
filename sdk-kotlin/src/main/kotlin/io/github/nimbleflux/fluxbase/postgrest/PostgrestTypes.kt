package io.github.nimbleflux.fluxbase.postgrest

/**
 * Vector similarity metric for pgvector queries.
 * Port of `VectorMetric` from `sdk/src/query-builder.ts:507`.
 */
enum class VectorMetric(val operator: String) {
    L2("vec_l2"),
    COSINE("vec_cos"),
    INNER_PRODUCT("vec_ip"),
}

/**
 * Count mode for PostgREST queries.
 * Port of `CountType` from `sdk/src/types.ts:271`.
 */
enum class CountType(val value: String) {
    EXACT("exact"),
    PLANNED("planned"),
    ESTIMATED("estimated"),
}
