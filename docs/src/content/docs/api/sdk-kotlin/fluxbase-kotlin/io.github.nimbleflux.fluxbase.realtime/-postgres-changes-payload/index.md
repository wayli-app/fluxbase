---
title: "PostgresChangesPayload"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.realtime](../index.md)/[PostgresChangesPayload](index.md)

# PostgresChangesPayload

[jvm]\
@Serializable

data class [PostgresChangesPayload](index.md)(val eventType: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val schema: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val table: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val commitTimestamp: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val newRecord: JsonElement? = null, val oldRecord: JsonElement? = null, val errors: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null)

A normalized postgres_changes payload — the Supabase-compatible shape the TS SDK produces by converting the server's `new_record`/`old_record` to `new`/`old`. Port of `RealtimePostgresChangesPayload` from `sdk/src/types.ts:408`.

## Constructors

| | |
|---|---|
| [PostgresChangesPayload](-postgres-changes-payload.md) | [jvm]<br>constructor(eventType: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), schema: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, table: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, commitTimestamp: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, newRecord: JsonElement? = null, oldRecord: JsonElement? = null, errors: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null) |

## Properties

| Name | Summary |
|---|---|
| [commitTimestamp](commit-timestamp.md) | [jvm]<br>@SerialName(value = &quot;commit_timestamp&quot;)<br>val [commitTimestamp](commit-timestamp.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null |
| [errors](errors.md) | [jvm]<br>val [errors](errors.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null |
| [eventType](event-type.md) | [jvm]<br>val [eventType](event-type.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [new](new.md) | [jvm]<br>val [new](new.md): JsonElement?<br>Convenience accessor — the TS SDK exposes `new` and `old`; in Kotlin those are keywords, so we use [newRecord](new-record.md) / [oldRecord](old-record.md). |
| [newRecord](new-record.md) | [jvm]<br>val [newRecord](new-record.md): JsonElement? = null |
| [old](old.md) | [jvm]<br>val [old](old.md): JsonElement? |
| [oldRecord](old-record.md) | [jvm]<br>val [oldRecord](old-record.md): JsonElement? = null |
| [schema](schema.md) | [jvm]<br>val [schema](schema.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null |
| [table](table.md) | [jvm]<br>val [table](table.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null |
