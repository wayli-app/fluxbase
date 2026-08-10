---
title: "RealtimeChannel"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.realtime](../index.md)/[RealtimeChannel](index.md)/[RealtimeChannel](-realtime-channel.md)

# RealtimeChannel

[jvm]\
constructor(baseUrl: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), channelName: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), token: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)?, transport: [WebSocketTransport](../-web-socket-transport/index.md), coroutineDispatcher: [CoroutineContext](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.coroutines/-coroutine-context/index.html) = Dispatchers.IO)

#### Parameters

jvm

| | |
|---|---|
| baseUrl | the Fluxbase base URL (http(s)→ws(s)). |
| channelName | the channel identifier. |
| token | the JWT (sent as `?token=` query param, not subprotocol). |
| transport | the [WebSocketTransport](../-web-socket-transport/index.md) SPI (tests inject a fake). |
