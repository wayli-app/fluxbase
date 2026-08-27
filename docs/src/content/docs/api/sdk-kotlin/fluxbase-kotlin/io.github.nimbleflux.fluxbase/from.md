---
title: "from"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase](../)/[from](./)

# from

[jvm]\
inline fun &lt;[T](./)&gt; [FluxbaseClient](../-fluxbase-client/).[from](./)(table: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), schema: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null): [QueryBuilder](../../iogithubnimblefluxfluxbasepostgrest/-query-builder/)&lt;[T](./)&gt;

Start a PostgREST query against [table](./). Uses a reified type parameter so the kotlinx.serialization serializer is resolved at compile time.

Usage: `client.from<Trip>("trips").select().eq("user_id", uid).execute()`

Port of `client.from(table)` in `client.ts:447`.
