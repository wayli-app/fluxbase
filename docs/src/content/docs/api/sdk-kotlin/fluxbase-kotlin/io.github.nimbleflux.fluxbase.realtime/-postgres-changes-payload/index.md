---
title: "PostgresChangesPayload"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.realtime](../)/[PostgresChangesPayload](./)

# PostgresChangesPayload

[jvm]\
@Serializable

data class [PostgresChangesPayload](./)(val eventType: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val schema: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val table: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val commitTimestamp: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val newRecord: JsonElement? = null, val oldRecord: JsonElement? = null, val errors: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null)

A normalized postgres_changes payload — the Supabase-compatible shape the TS SDK produces by converting the server's `new_record`/`old_record` to `new`/`old`. Port of `RealtimePostgresChangesPayload` from `sdk/src/types.ts:408`.

## Constructors

| | |
|---|---|
| [PostgresChangesPayload](-postgres-changes-payload/) | [jvm]<br>constructor(eventType: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), schema: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, table: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, commitTimestamp: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, newRecord: JsonElement? = null, oldRecord: JsonElement? = null, errors: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null) |

## Properties

| Name | Summary |
|---|---|
| [commitTimestamp](commit-timestamp/) | [jvm]<br>@SerialName(value = &quot;commit_timestamp&quot;)<br>val [commitTimestamp](commit-timestamp/): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null |
| [errors](errors/) | [jvm]<br>val [errors](errors/): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null |
| [eventType](event-type/) | [jvm]<br>val [eventType](event-type/): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [new](new/) | [jvm]<br>val [new](new/): JsonElement?<br>Convenience accessor — the TS SDK exposes `new` and `old`; in Kotlin those are keywords, so we use [newRecord](new-record/) / [oldRecord](old-record/). |
| [newRecord](new-record/) | [jvm]<br>val [newRecord](new-record/): JsonElement? = null |
| [old](old/) | [jvm]<br>val [old](old/): JsonElement? |
| [oldRecord](old-record/) | [jvm]<br>val [oldRecord](old-record/): JsonElement? = null |
| [schema](schema/) | [jvm]<br>val [schema](schema/): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null |
| [table](table/) | [jvm]<br>val [table](table/): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null |
