---
title: "KtorWebSocketTransport"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.realtime](../index.md)/[KtorWebSocketTransport](index.md)

# KtorWebSocketTransport

[jvm]\
class [KtorWebSocketTransport](index.md) : [WebSocketTransport](../-web-socket-transport/index.md)

Ktor-backed [WebSocketTransport](../-web-socket-transport/index.md) — the production WebSocket implementation for JVM/Android.

Connects to Fluxbase's `/realtime` endpoint via Ktor's WebSocket client (OkHttp engine). The connection is established eagerly on an internal Dispatchers.IO scope when [connect](connect.md) is called; inbound text frames are bridged onto the returned Flow, which [RealtimeChannel](../-realtime-channel/index.md) collects. Outbound messages are sent via [send](send.md) once the session handshake completes (observable via [isConnected](is-connected.md)).

The protocol itself (subscribe/heartbeat/postgres_changes/broadcast) lives in [RealtimeChannel](../-realtime-channel/index.md); this class is purely the transport seam — the same SPI tests fake with FakeWebSocketTransport.

## Constructors

| | |
|---|---|
| [KtorWebSocketTransport](-ktor-web-socket-transport.md) | [jvm]<br>constructor() |

## Properties

| Name | Summary |
|---|---|
| [isConnected](is-connected.md) | [jvm]<br>@[Volatile](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.jvm/-volatile/index.html)<br>open override var [isConnected](is-connected.md): [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html)<br>Whether the transport is currently connected. |

## Functions

| Name | Summary |
|---|---|
| [close](close.md) | [jvm]<br>open override fun [close](close.md)()<br>Close the connection. |
| [connect](connect.md) | [jvm]<br>open override fun [connect](connect.md)(url: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): Flow&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt;<br>Connect to the given URL and return a flow of incoming JSON text messages. |
| [send](send.md) | [jvm]<br>open suspend override fun [send](send.md)(text: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html))<br>Send a text message. |
