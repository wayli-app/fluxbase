---
title: "channel"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase](../../)/[FluxbaseClient](../)/[channel](./)

# channel

[jvm]\
fun [channel](./)(name: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), transport: [WebSocketTransport](../../../iogithubnimblefluxfluxbaserealtime/-web-socket-transport/)? = null): [RealtimeChannel](../../../iogithubnimblefluxfluxbaserealtime/-realtime-channel/)

Create a realtime channel for postgres_changes/broadcast/presence subscriptions. Port of `channel()` in `client.ts:654-674`.

Usage:

```kotlin
val channel = client.channel("public:trips")
channel.on("INSERT", PostgresChangesConfig(table = "trips")) { payload -> ... }
channel.subscribe()
```
