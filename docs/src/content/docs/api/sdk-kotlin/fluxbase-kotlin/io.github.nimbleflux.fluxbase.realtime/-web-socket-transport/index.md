---
title: "WebSocketTransport"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.realtime](../)/[WebSocketTransport](./)

# WebSocketTransport

interface [WebSocketTransport](./)

SPI for WebSocket I/O — the seam where tests inject a fake transport (see FakeWebSocketTransport) instead of a real WS connection.

The real implementation uses Ktor's WebSocket client; the fake simulates server messages via FakeWebSocketTransport.simulateMessage.

#### Inheritors

| |
|---|
| [KtorWebSocketTransport](../-ktor-web-socket-transport/) |

## Properties

| Name | Summary |
|---|---|
| [isConnected](is-connected/) | [jvm]<br>abstract val [isConnected](is-connected/): [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html)<br>Whether the transport is currently connected. |

## Functions

| Name | Summary |
|---|---|
| [close](close/) | [jvm]<br>abstract fun [close](close/)()<br>Close the connection. |
| [connect](connect/) | [jvm]<br>abstract fun [connect](connect/)(url: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): Flow&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt;<br>Connect to the given URL and return a flow of incoming JSON text messages. |
| [send](send/) | [jvm]<br>abstract suspend fun [send](send/)(text: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html))<br>Send a text message. |
