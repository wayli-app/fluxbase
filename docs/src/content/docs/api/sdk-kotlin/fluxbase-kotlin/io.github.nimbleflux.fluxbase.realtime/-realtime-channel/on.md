---
title: "on"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.realtime](../../)/[RealtimeChannel](../)/[on](./)

# on

[jvm]\
fun [on](./)(event: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), config: [PostgresChangesConfig](../../-postgres-changes-config/)? = null, callback: [ChangeEventCallback](../../-change-event-callback/)): [RealtimeChannel](../)

Register a callback for a postgres_changes event type. Port of `channel.on("INSERT"|"UPDATE"|"DELETE"|"*", cb)` in `realtime.ts`.
