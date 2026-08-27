---
title: "KtorWebSocketTransport"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.realtime](../)/[KtorWebSocketTransport](./)

# KtorWebSocketTransport

[jvm]\
class [KtorWebSocketTransport](./) : [WebSocketTransport](../-web-socket-transport/)

Ktor-backed [WebSocketTransport](../-web-socket-transport/) — the production WebSocket implementation for JVM/Android.

Connects to Fluxbase's `/realtime` endpoint via Ktor's WebSocket client (OkHttp engine). The connection is established eagerly on an internal Dispatchers.IO scope when [connect](connect/) is called; inbound text frames are bridged onto the returned Flow, which [RealtimeChannel](../-realtime-channel/) collects. Outbound messages are sent via [send](send/) once the session handshake completes (observable via [isConnected](is-connected/)).

The protocol itself (subscribe/heartbeat/postgres_changes/broadcast) lives in [RealtimeChannel](../-realtime-channel/); this class is purely the transport seam — the same SPI tests fake with FakeWebSocketTransport.

## Constructors

| | |
|---|---|
| [KtorWebSocketTransport](-ktor-web-socket-transport/) | [jvm]<br>constructor() |

## Properties

| Name | Summary |
|---|---|
| [isConnected](is-connected/) | [jvm]<br>@[Volatile](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.jvm/-volatile/index.html)<br>open override var [isConnected](is-connected/): [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html)<br>Whether the transport is currently connected. |

## Functions

| Name | Summary |
|---|---|
| [close](close/) | [jvm]<br>open override fun [close](close/)()<br>Close the connection. |
| [connect](connect/) | [jvm]<br>open override fun [connect](connect/)(url: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): Flow&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt;<br>Connect to the given URL and return a flow of incoming JSON text messages. |
| [send](send/) | [jvm]<br>open suspend override fun [send](send/)(text: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html))<br>Send a text message. |
