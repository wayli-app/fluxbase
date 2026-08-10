//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.realtime](../index.md)/[WebSocketTransport](index.md)

# WebSocketTransport

interface [WebSocketTransport](index.md)

SPI for WebSocket I/O — the seam where tests inject a fake transport (see FakeWebSocketTransport) instead of a real WS connection.

The real implementation uses Ktor's WebSocket client; the fake simulates server messages via FakeWebSocketTransport.simulateMessage.

#### Inheritors

| |
|---|
| [KtorWebSocketTransport](../-ktor-web-socket-transport/index.md) |

## Properties

| Name | Summary |
|---|---|
| [isConnected](is-connected.md) | [jvm]<br>abstract val [isConnected](is-connected.md): [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html)<br>Whether the transport is currently connected. |

## Functions

| Name | Summary |
|---|---|
| [close](close.md) | [jvm]<br>abstract fun [close](close.md)()<br>Close the connection. |
| [connect](connect.md) | [jvm]<br>abstract fun [connect](connect.md)(url: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): Flow&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt;<br>Connect to the given URL and return a flow of incoming JSON text messages. |
| [send](send.md) | [jvm]<br>abstract suspend fun [send](send.md)(text: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html))<br>Send a text message. |
