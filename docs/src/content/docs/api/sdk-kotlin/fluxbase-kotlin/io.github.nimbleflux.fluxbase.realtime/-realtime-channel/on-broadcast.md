---
title: "onBroadcast"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.realtime](../index.md)/[RealtimeChannel](index.md)/[onBroadcast](on-broadcast.md)

# onBroadcast

[jvm]\
fun [onBroadcast](on-broadcast.md)(event: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), callback: [BroadcastCallback](../-broadcast-callback/index.md)): [RealtimeChannel](index.md)

Register a callback for broadcast events. Port of `channel.on("broadcast", {event}, cb)` in `realtime.ts`.
