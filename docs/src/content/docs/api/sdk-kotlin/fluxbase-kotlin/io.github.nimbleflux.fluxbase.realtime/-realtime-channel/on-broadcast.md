---
title: "onBroadcast"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.realtime](../../)/[RealtimeChannel](../)/[onBroadcast](./)

# onBroadcast

[jvm]\
fun [onBroadcast](./)(event: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), callback: [BroadcastCallback](../../-broadcast-callback/)): [RealtimeChannel](../)

Register a callback for broadcast events. Port of `channel.on("broadcast", {event}, cb)` in `realtime.ts`.
