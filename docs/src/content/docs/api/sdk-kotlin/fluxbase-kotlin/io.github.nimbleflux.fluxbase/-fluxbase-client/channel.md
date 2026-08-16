---
title: "channel"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase](../index.md)/[FluxbaseClient](index.md)/[channel](channel.md)

# channel

[jvm]\
fun [channel](channel.md)(name: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), transport: [WebSocketTransport](../../io.github.nimbleflux.fluxbase.realtime/-web-socket-transport/index.md)? = null): [RealtimeChannel](../../io.github.nimbleflux.fluxbase.realtime/-realtime-channel/index.md)

Create a realtime channel for postgres_changes/broadcast/presence subscriptions. Port of `channel()` in `client.ts:654-674`.

Usage:

```kotlin
val channel = client.channel("public:trips")
channel.on("INSERT", PostgresChangesConfig(table = "trips")) { payload -> ... }
channel.subscribe()
```
