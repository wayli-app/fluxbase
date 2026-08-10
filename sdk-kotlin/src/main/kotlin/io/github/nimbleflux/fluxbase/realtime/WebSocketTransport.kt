package io.github.nimbleflux.fluxbase.realtime

import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.asSharedFlow

/**
 * SPI for WebSocket I/O — the seam where tests inject a fake transport
 * (see [FakeWebSocketTransport]) instead of a real WS connection.
 *
 * The real implementation uses Ktor's WebSocket client; the fake simulates
 * server messages via [FakeWebSocketTransport.simulateMessage].
 */
interface WebSocketTransport {
    /** Connect to the given URL and return a flow of incoming JSON text messages. */
    fun connect(url: String): Flow<String>

    /** Send a text message. */
    suspend fun send(text: String)

    /** Close the connection. */
    fun close()

    /** Whether the transport is currently connected. */
    val isConnected: Boolean
}
