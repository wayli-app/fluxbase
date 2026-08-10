---
title: "from"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../index.md)/[io.github.nimbleflux.fluxbase](index.md)/[from](from.md)

# from

[jvm]\
inline fun &lt;[T](from.md)&gt; [FluxbaseClient](-fluxbase-client/index.md).[from](from.md)(table: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), schema: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null): [QueryBuilder](../io.github.nimbleflux.fluxbase.postgrest/-query-builder/index.md)&lt;[T](from.md)&gt;

Start a PostgREST query against [table](from.md). Uses a reified type parameter so the kotlinx.serialization serializer is resolved at compile time.

Usage: `client.from<Trip>("trips").select().eq("user_id", uid).execute()`

Port of `client.from(table)` in `client.ts:447`.
