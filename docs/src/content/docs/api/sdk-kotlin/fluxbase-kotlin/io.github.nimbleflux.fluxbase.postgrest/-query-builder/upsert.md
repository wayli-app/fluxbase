//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.postgrest](../index.md)/[QueryBuilder](index.md)/[upsert](upsert.md)

# upsert

[jvm]\
suspend fun [upsert](upsert.md)(values: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?&gt;): [PostgrestResponse](../-postgrest-response/index.md)&lt;[T](index.md)&gt;

UPSERT (INSERT with conflict resolution). Port of `upsert()` in `query-builder.ts:102`. Adds the `Prefer: resolution=merge-duplicates` header to a POST.
