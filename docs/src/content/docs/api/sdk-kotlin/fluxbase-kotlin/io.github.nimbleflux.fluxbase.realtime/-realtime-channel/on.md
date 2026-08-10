---
title: "on"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.realtime](../index.md)/[RealtimeChannel](index.md)/[on](on.md)

# on

[jvm]\
fun [on](on.md)(event: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), config: [PostgresChangesConfig](../-postgres-changes-config/index.md)? = null, callback: [ChangeEventCallback](../-change-event-callback/index.md)): [RealtimeChannel](index.md)

Register a callback for a postgres_changes event type. Port of `channel.on("INSERT"|"UPDATE"|"DELETE"|"*", cb)` in `realtime.ts`.
