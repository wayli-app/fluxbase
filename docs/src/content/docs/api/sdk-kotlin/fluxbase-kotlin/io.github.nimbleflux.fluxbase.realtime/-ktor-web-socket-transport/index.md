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

Uses Ktor's WebSocket client to connect to Fluxbase's `/realtime` endpoint. The actual Ktor WS session is established on [connect](connect.md) and messages are received as a Flow of JSON text.

TODO: implement the full Ktor WebSocket session (connect, send, receive, close) once the integration test infrastructure is in place. For now this is a placeholder that throws on connect — tests use FakeWebSocketTransport.

## Constructors

| | |
|---|---|
| [KtorWebSocketTransport](-ktor-web-socket-transport.md) | [jvm]<br>constructor() |

## Properties

| Name | Summary |
|---|---|
| [isConnected](is-connected.md) | [jvm]<br>open override var [isConnected](is-connected.md): [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html)<br>Whether the transport is currently connected. |

## Functions

| Name | Summary |
|---|---|
| [close](close.md) | [jvm]<br>open override fun [close](close.md)()<br>Close the connection. |
| [connect](connect.md) | [jvm]<br>open override fun [connect](connect.md)(url: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): Flow&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt;<br>Connect to the given URL and return a flow of incoming JSON text messages. |
| [send](send.md) | [jvm]<br>open suspend override fun [send](send.md)(text: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html))<br>Send a text message. |
