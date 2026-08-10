//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.postgrest](../index.md)/[QueryBuilder](index.md)

# QueryBuilder

[jvm]\
class [QueryBuilder](index.md)&lt;[T](index.md)&gt;

A chainable PostgREST query builder. Port of `QueryBuilder<T>` from `sdk/src/query-builder.ts`.

Builds PostgREST-style query strings against Fluxbase's `/api/v1/tables/{schema}/{table}` endpoint. The builder is immutable — each filter method returns a new [QueryBuilder](index.md) (matching the TS `clone()` pattern) so a base query can be reused with different filters.

Generic type [T](index.md) is resolved at compile time via the reified from extension.

Usage:

```kotlin
val result = client.from<Trip>().select().eq("user_id", uid).execute()
```

## Types

| Name | Summary |
|---|---|
| [Filter](-filter/index.md) | [jvm]<br>data class [Filter](-filter/index.md)(val column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val operator: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val value: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?) |

## Functions

| Name | Summary |
|---|---|
| [delete](delete.md) | [jvm]<br>suspend fun [delete](delete.md)(): [PostgrestResponse](../-postgrest-response/index.md)&lt;[T](index.md)&gt;<br>DELETE (requires filters). |
| [eq](eq.md) | [jvm]<br>fun [eq](eq.md)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?): [QueryBuilder](index.md)&lt;[T](index.md)&gt; |
| [execute](execute.md) | [jvm]<br>suspend fun [execute](execute.md)(): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[T](index.md)&gt;&gt;<br>Execute a SELECT query and return the list of rows. |
| [gt](gt.md) | [jvm]<br>fun [gt](gt.md)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?): [QueryBuilder](index.md)&lt;[T](index.md)&gt; |
| [gte](gte.md) | [jvm]<br>fun [gte](gte.md)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?): [QueryBuilder](index.md)&lt;[T](index.md)&gt; |
| [ilike](ilike.md) | [jvm]<br>fun [ilike](ilike.md)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [QueryBuilder](index.md)&lt;[T](index.md)&gt; |
| [in](in.md) | [jvm]<br>fun [in](in.md)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), values: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?&gt;): [QueryBuilder](index.md)&lt;[T](index.md)&gt; |
| [insert](insert.md) | [jvm]<br>suspend fun [insert](insert.md)(values: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?&gt;): [PostgrestResponse](../-postgrest-response/index.md)&lt;[T](index.md)&gt;<br>INSERT. |
| [is_](is_.md) | [jvm]<br>fun [is_](is_.md)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?): [QueryBuilder](index.md)&lt;[T](index.md)&gt; |
| [like](like.md) | [jvm]<br>fun [like](like.md)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [QueryBuilder](index.md)&lt;[T](index.md)&gt; |
| [limit](limit.md) | [jvm]<br>fun [limit](limit.md)(limit: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)): [QueryBuilder](index.md)&lt;[T](index.md)&gt; |
| [lt](lt.md) | [jvm]<br>fun [lt](lt.md)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?): [QueryBuilder](index.md)&lt;[T](index.md)&gt; |
| [lte](lte.md) | [jvm]<br>fun [lte](lte.md)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?): [QueryBuilder](index.md)&lt;[T](index.md)&gt; |
| [maybeSingle](maybe-single.md) | [jvm]<br>suspend fun [maybeSingle](maybe-single.md)(): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[T](index.md)?&gt;<br>Limit to 1 row; null if 0 rows (no error). Port of `maybeSingle()`. |
| [neq](neq.md) | [jvm]<br>fun [neq](neq.md)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?): [QueryBuilder](index.md)&lt;[T](index.md)&gt; |
| [offset](offset.md) | [jvm]<br>fun [offset](offset.md)(offset: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)): [QueryBuilder](index.md)&lt;[T](index.md)&gt; |
| [order](order.md) | [jvm]<br>fun [order](order.md)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), ascending: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = true): [QueryBuilder](index.md)&lt;[T](index.md)&gt; |
| [range](range.md) | [jvm]<br>fun [range](range.md)(from: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html), to: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)): [QueryBuilder](index.md)&lt;[T](index.md)&gt;<br>Range-based pagination — returns a clone so the base builder can be reused. |
| [select](select.md) | [jvm]<br>fun [select](select.md)(columns: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;*&quot;): [QueryBuilder](index.md)&lt;[T](index.md)&gt; |
| [single](single.md) | [jvm]<br>suspend fun [single](single.md)(): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[T](index.md)&gt;<br>Limit to 1 row; error if 0 rows (PGRST116). Port of `single()`. |
| [update](update.md) | [jvm]<br>suspend fun [update](update.md)(values: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?&gt;): [PostgrestResponse](../-postgrest-response/index.md)&lt;[T](index.md)&gt;<br>UPDATE (requires filters). |
