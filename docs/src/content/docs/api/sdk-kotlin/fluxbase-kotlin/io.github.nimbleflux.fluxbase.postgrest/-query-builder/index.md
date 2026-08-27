---
title: "QueryBuilder"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.postgrest](../)/[QueryBuilder](./)

# QueryBuilder

[jvm]\
class [QueryBuilder](./)&lt;[T](./)&gt;

A chainable PostgREST query builder. Port of `QueryBuilder<T>` from `sdk/src/query-builder.ts`.

Builds PostgREST-style query strings against Fluxbase's `/api/v1/tables/{schema}/{table}` endpoint. The builder is immutable — each filter method returns a new [QueryBuilder](./) (matching the TS `clone()` pattern) so a base query can be reused with different filters.

Generic type [T](./) is resolved at compile time via the reified from extension.

Usage:

```kotlin
val result = client.from<Trip>().select().eq("user_id", uid).execute()
```

## Types

| Name | Summary |
|---|---|
| [Filter](-filter/) | [jvm]<br>data class [Filter](-filter/)(val column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val operator: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val value: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?) |

## Functions

| Name | Summary |
|---|---|
| [between](between/) | [jvm]<br>fun [between](between/)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), min: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?, max: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?): [QueryBuilder](./)&lt;[T](./)&gt;<br>Filter values between [min](between/) and [max](between/) (inclusive). Adds gte + lte filters. |
| [count](count/) | [jvm]<br>fun [count](count/)(countType: [CountType](../-count-type/) = CountType.EXACT): [QueryBuilder](./)&lt;[T](./)&gt;<br>Request a count of total matching rows. |
| [crosses](crosses/) | [jvm]<br>fun [crosses](crosses/)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), geojson: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?): [QueryBuilder](./)&lt;[T](./)&gt;<br>PostGIS ST_Crosses. |
| [delete](delete/) | [jvm]<br>suspend fun [delete](delete/)(): [PostgrestResponse](../-postgrest-response/)&lt;[T](./)&gt;<br>DELETE (requires filters). |
| [eq](eq/) | [jvm]<br>fun [eq](eq/)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?): [QueryBuilder](./)&lt;[T](./)&gt; |
| [execute](execute/) | [jvm]<br>suspend fun [execute](execute/)(): [PostgrestResponse](../-postgrest-response/)&lt;[List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[T](./)&gt;&gt;<br>Execute a SELECT query and return the list of rows (with count if requested). |
| [gt](gt/) | [jvm]<br>fun [gt](gt/)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?): [QueryBuilder](./)&lt;[T](./)&gt; |
| [gte](gte/) | [jvm]<br>fun [gte](gte/)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?): [QueryBuilder](./)&lt;[T](./)&gt; |
| [ilike](ilike/) | [jvm]<br>fun [ilike](ilike/)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [QueryBuilder](./)&lt;[T](./)&gt; |
| [in](in/) | [jvm]<br>fun [in](in/)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), values: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?&gt;): [QueryBuilder](./)&lt;[T](./)&gt; |
| [insert](insert/) | [jvm]<br>suspend fun [insert](insert/)(values: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?&gt;): [PostgrestResponse](../-postgrest-response/)&lt;[T](./)&gt;<br>INSERT. |
| [intersects](intersects/) | [jvm]<br>fun [intersects](intersects/)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), geojson: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?): [QueryBuilder](./)&lt;[T](./)&gt;<br>PostGIS ST_Intersects — filter geometries that intersect [geojson](intersects/). |
| [is_](is_/) | [jvm]<br>fun [is_](is_/)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?): [QueryBuilder](./)&lt;[T](./)&gt; |
| [like](like/) | [jvm]<br>fun [like](like/)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [QueryBuilder](./)&lt;[T](./)&gt; |
| [limit](limit/) | [jvm]<br>fun [limit](limit/)(limit: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)): [QueryBuilder](./)&lt;[T](./)&gt; |
| [lt](lt/) | [jvm]<br>fun [lt](lt/)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?): [QueryBuilder](./)&lt;[T](./)&gt; |
| [lte](lte/) | [jvm]<br>fun [lte](lte/)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?): [QueryBuilder](./)&lt;[T](./)&gt; |
| [maybeSingle](maybe-single/) | [jvm]<br>suspend fun [maybeSingle](maybe-single/)(): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[T](./)?&gt;<br>Limit to 1 row; null if 0 rows (no error). Port of `maybeSingle()`. |
| [neq](neq/) | [jvm]<br>fun [neq](neq/)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?): [QueryBuilder](./)&lt;[T](./)&gt; |
| [notBetween](not-between/) | [jvm]<br>fun [notBetween](not-between/)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), min: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?, max: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?): [QueryBuilder](./)&lt;[T](./)&gt;<br>Filter values NOT between [min](not-between/) and [max](not-between/). Adds lt + gt filters. |
| [offset](offset/) | [jvm]<br>fun [offset](offset/)(offset: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)): [QueryBuilder](./)&lt;[T](./)&gt; |
| [order](order/) | [jvm]<br>fun [order](order/)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), ascending: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = true): [QueryBuilder](./)&lt;[T](./)&gt; |
| [orderByVector](order-by-vector/) | [jvm]<br>fun [orderByVector](order-by-vector/)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), vector: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[Double](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-double/index.html)&gt;, metric: [VectorMetric](../-vector-metric/)): [QueryBuilder](./)&lt;[T](./)&gt;<br>Order by vector similarity. Adds a vector order clause. Port of `orderByVector()` in `query-builder.ts:500`. |
| [range](range/) | [jvm]<br>fun [range](range/)(from: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html), to: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)): [QueryBuilder](./)&lt;[T](./)&gt;<br>Range-based pagination — returns a clone so the base builder can be reused. |
| [select](select/) | [jvm]<br>fun [select](select/)(columns: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;*&quot;): [QueryBuilder](./)&lt;[T](./)&gt; |
| [single](single/) | [jvm]<br>suspend fun [single](single/)(): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[T](./)&gt;<br>Limit to 1 row; error if 0 rows (PGRST116). Port of `single()`. |
| [stContains](st-contains/) | [jvm]<br>fun [stContains](st-contains/)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), geojson: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?): [QueryBuilder](./)&lt;[T](./)&gt;<br>PostGIS ST_Contains — filter geometries that contain [geojson](st-contains/). |
| [stDistance](st-distance/) | [jvm]<br>fun [stDistance](st-distance/)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), geojson: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?): [QueryBuilder](./)&lt;[T](./)&gt;<br>PostGIS ST_Distance — returns distance (used for ordering). |
| [stDWithin](st-d-within/) | [jvm]<br>fun [stDWithin](st-d-within/)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), geojson: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?, distanceMeters: [Double](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-double/index.html)): &lt;Error class: unknown class&gt;<br>PostGIS ST_DWithin — within [distanceMeters](st-d-within/) of [geojson](st-d-within/). |
| [stOverlaps](st-overlaps/) | [jvm]<br>fun [stOverlaps](st-overlaps/)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), geojson: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?): [QueryBuilder](./)&lt;[T](./)&gt;<br>PostGIS ST_Overlaps. |
| [touches](touches/) | [jvm]<br>fun [touches](touches/)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), geojson: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?): [QueryBuilder](./)&lt;[T](./)&gt;<br>PostGIS ST_Touches. |
| [update](update/) | [jvm]<br>suspend fun [update](update/)(values: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?&gt;): [PostgrestResponse](../-postgrest-response/)&lt;[T](./)&gt;<br>UPDATE (requires filters). |
| [upsert](upsert/) | [jvm]<br>suspend fun [upsert](upsert/)(values: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?&gt;): [PostgrestResponse](../-postgrest-response/)&lt;[T](./)&gt;<br>UPSERT (INSERT with conflict resolution). Port of `upsert()` in `query-builder.ts:102`. Adds the `Prefer: resolution=merge-duplicates` header to a POST. |
| [vectorSearch](vector-search/) | [jvm]<br>fun [vectorSearch](vector-search/)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), vector: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[Double](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-double/index.html)&gt;, metric: [VectorMetric](../-vector-metric/) = VectorMetric.COSINE): [QueryBuilder](./)&lt;[T](./)&gt;<br>Vector similarity search — combines an order-by-distance + a distance filter. Port of `vectorSearch()` in `query-builder.ts:540`. |
| [within](within/) | [jvm]<br>fun [within](within/)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), geojson: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?): [QueryBuilder](./)&lt;[T](./)&gt;<br>PostGIS ST_Within — filter geometries within [geojson](within/). |
