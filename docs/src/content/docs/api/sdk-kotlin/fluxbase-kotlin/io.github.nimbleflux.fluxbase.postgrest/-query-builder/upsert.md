---
title: "upsert"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.postgrest](../../)/[QueryBuilder](../)/[upsert](./)

# upsert

[jvm]\
suspend fun [upsert](./)(values: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?&gt;): [PostgrestResponse](../../-postgrest-response/)&lt;[T](../)&gt;

UPSERT (INSERT with conflict resolution). Port of `upsert()` in `query-builder.ts:102`. Adds the `Prefer: resolution=merge-duplicates` header to a POST.
